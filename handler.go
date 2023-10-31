package main

import (
	"log"
	"net/http"
	"text/template"

	"github.com/ffiat/nostr"
	"github.com/gorilla/mux"
)

type Data struct {
	Events      []*nostr.Event
	SearchQuery string
	Error       string
	Invalid     bool
}

type Handler struct {
	repository Repository
}

func (s *Handler) IndexHandler(w http.ResponseWriter, r *http.Request) {
	http.Redirect(w, r, "/home", http.StatusMovedPermanently)
}

func (s *Handler) Article(w http.ResponseWriter, r *http.Request) {

	vars := mux.Vars(r)
	a, err := s.repository.Article(vars["id"])
    if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
    }

	tmpl, err := template.ParseFiles("static/article.html")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	tmpl.Execute(w, a)
}

func (s *Handler) Home(w http.ResponseWriter, r *http.Request) {

	data := Data{}

	tmpl, err := template.ParseFiles("static/home.html", "static/card.html")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	tmpl.ExecuteTemplate(w, "home.html", data)
}

func (s *Handler) Validate(w http.ResponseWriter, r *http.Request) {
	pk := r.URL.Query().Get("pubkey")
	if pk != "" {
		_, err := nostr.DecodeBech32(pk)
		if err != nil {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("invalid npub"))
			return
		}
	}
}

func (s *Handler) ListEvents(w http.ResponseWriter, r *http.Request) {

	log.Println("Listing events")

	npub := r.URL.Query().Get("pubkey")
	if npub == "" {
        log.Fatalln("no npub provided")
    }

    pk, err := nostr.DecodeBech32(npub)
    if err != nil {
        w.WriteHeader(http.StatusOK)
        w.Write([]byte("invalid npub"))
        return
    }

    articles, err := s.repository.FindArticles(pk.(string))
    if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
    }

	tmpl, err := template.ParseFiles("static/card.html")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	tmpl.Execute(w, articles)
}
