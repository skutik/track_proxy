package web_app

import (
	"html/template"
	"log"
	"net/http"
	"track_proxy/requests_storage"
)

var lastFetchedId = ""

func HandleIndex(w http.ResponseWriter, r *http.Request) {
	tmpl, err := template.ParseFiles(
		"templates/index.html",
		"templates/requests_table.html",
		"templates/toolbar.html",
	)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	err = tmpl.ExecuteTemplate(w, "index.html", nil)
	if err != nil {
		log.Println("error rendering template:", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func HandleRequestsTable(w http.ResponseWriter, r *http.Request) {
	tmpl, err := template.ParseFiles("templates/requests_table.html")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	requests, lastId := requests_storage.Storage.GetRequestSinceId(lastFetchedId)
	lastFetchedId = lastId
	if len(requests) == 0 {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	err = tmpl.ExecuteTemplate(w, "tableContent", requests)
	if err != nil {
		log.Println("error rendering template:", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}
