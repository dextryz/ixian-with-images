package main

import (
	"html/template"
	"log"
	"net/http"

	"github.com/ffiat/nostr"
)

type Data struct {
	Events      []*nostr.Event
	SearchQuery string
}

type Handler struct {
	repository Repository
}

func (s *Handler) IndexHandler(w http.ResponseWriter, r *http.Request) {
	http.Redirect(w, r, "/contact", http.StatusMovedPermanently)
}

func (s *Handler) ListEvents(w http.ResponseWriter, r *http.Request) {

	log.Println("Searching")

	search := r.URL.Query().Get("q")

	data := Data{
		SearchQuery: search,
	}

	if search != "" {
		header := r.Header.Get("HX-Trigger")
		if header == "search" {
			log.Println("Header search FOUND")
			//events = s.repository.Find(search)
		}
	} else {
		data.Events = s.repository.All()
	}

	tmpl, err := template.ParseFiles("template/home.html", "template/index.html")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	log.Println(data)

	tmpl.ExecuteTemplate(w, "home.html", data)
}
