package web_app

import (
	"html/template"
	"log"
	"net/http"
	"track_proxy/requests_storage"
)

func HandleIndex(w http.ResponseWriter, r *http.Request) {
	tmpl, err := template.ParseFiles("templates/index.html")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	requests_storage.Lock.Lock()
	defer requests_storage.Lock.Unlock()
	err = tmpl.ExecuteTemplate(w, "index", requests_storage.Storage)
	if err != nil {
		log.Println("error rendering template:", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}
