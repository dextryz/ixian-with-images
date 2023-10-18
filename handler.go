package main

import (
	"html/template"
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
	http.Redirect(w, r, "/events", http.StatusMovedPermanently)
}

func (s *Handler) ListEvents(w http.ResponseWriter, r *http.Request) {

	search := r.URL.Query().Get("pubkey")

	data := Data{
		SearchQuery: search,
	}

	// TODO: Validate PubKey similar to email validation.

	if search != "" {
		data.Events = s.repository.FindByPubKey(search)
	} else {
		data.Events = s.repository.All()
	}

	tmpl, err := template.ParseFiles("template/home.html", "template/index.html")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	tmpl.ExecuteTemplate(w, "home.html", data)
}
