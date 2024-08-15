package web_app

import (
	"html/template"
	"log"
	"net/http"
	"track_proxy/requests_storage"
)

var lastFetchedId = ""

var activeFilter = requests_storage.SearchFilter{}

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
	requests, lastId := requests_storage.Storage.GetRequestSinceId(lastFetchedId, activeFilter)
	lastFetchedId = lastId
	if len(requests) == 0 {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	tmpl, err := template.ParseFiles("templates/requests_table.html")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	err = tmpl.ExecuteTemplate(w, "tableContent", requests)
	if err != nil {
		log.Println("error rendering template:", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func HandleFilterRequests(w http.ResponseWriter, r *http.Request) {
	err := r.ParseForm()
	if err != nil {
		log.Println("error parsing filters form:", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
	activeFilter.UpdateFilters(&r.Form)
	requests, lastId := requests_storage.Storage.GetRequestSinceId("", activeFilter)
	lastFetchedId = lastId
	if len(requests) == 0 {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	tmpl, err := template.ParseFiles("templates/requests_table.html")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	err = tmpl.ExecuteTemplate(w, "tableContent", requests)
	if err != nil {
		log.Println("error rendering template:", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}
