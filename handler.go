package main

import (
	"fmt"
	"log"
	"net/http"
	"strings"
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

func (s *Handler) Tag(w http.ResponseWriter, r *http.Request) {

	vars := mux.Vars(r)

	cards := []*Article{}

	// THis id is noteID from NIP-21
	for _, nid := range s.repository.hashtags[vars["tag"]] {

		a, err := s.repository.Article(nid)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		cards = append(cards, a)
	}

	tmpl, err := template.ParseFiles("static/taglist.html")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	tmpl.Execute(w, cards)
}

func (s *Handler) Profile(w http.ResponseWriter, r *http.Request) {

	vars := mux.Vars(r)

	// TODO: For now prefix should only be 'note'
	_, event, err := nostr.DecodeBech32(vars["id"])
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	a, err := s.repository.Article(event)
	if err != nil {
		log.Println(err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// 1. Pull lists
	// 1. Pull bookmarks
	// 1. Pull hightlight
	// 1. Pull graphs
	// 1. Pull orphans

	tmpl, err := template.ParseFiles("static/profile.html")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	tmpl.Execute(w, a.Profile)
}

func (s *Handler) Article(w http.ResponseWriter, r *http.Request) {

	vars := mux.Vars(r)

	// TODO: For now prefix should only be 'note'
	_, event, err := nostr.DecodeBech32(vars["id"])
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	a, err := s.repository.Article(event)
	if err != nil {
		log.Println(err)
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

	tmpl, err := template.ParseFiles("static/home.html", "static/card.html")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	articles := []*Article{}
	err = tmpl.ExecuteTemplate(w, "home.html", articles)
	if err != nil {
		fmt.Println("Error executing template:", err)
	}
}

func (s *Handler) Validate(w http.ResponseWriter, r *http.Request) {
	pk := r.URL.Query().Get("search")
	if pk != "" {
		_, _, err := nostr.DecodeBech32(pk)
		log.Println(pk)
		log.Println(err)
		if err != nil {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`<span class="message error">Invalid entity</span>`))
			return
		}
	}
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`<span class="message success">Valid entity</span>`))
}

func (s *Handler) ListEvents(w http.ResponseWriter, r *http.Request) {

	search := r.URL.Query().Get("search")
	if search == "" {
		log.Fatalln("no npub provided")
	}

	articles := []*Article{}

	if strings.HasPrefix(search, nostr.UriEvent) {

		id := strings.TrimPrefix(search, nostr.UriEvent)

		// Pull the NIP-51 list event using event ID.
		event, err := s.repository.CategorizedPeople(id)
		if err != nil {
			panic(err)
		}

		// Loop all authors (pubkeys) in NIP-51 event tags (list).
		for _, value := range event.Tags {

			t, v := value[0], value[1]

			if t == "p" {
				notes, err := s.repository.FindArticles(v)
				if err != nil {
					http.Error(w, err.Error(), http.StatusInternalServerError)
					return
				}
				for _, n := range notes {
					articles = append(articles, n)
				}
			}
		}
	} else if strings.HasPrefix(search, nostr.UriPub) {

		log.Println("pull profile NIP-01")

		npub := strings.TrimPrefix(search, nostr.Prefix)

		_, pk, err := nostr.DecodeBech32(npub)
		if err != nil {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("invalid npub"))
			return
		}

		articles, err = s.repository.FindArticles(pk)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

	}

	tmpl, err := template.ParseFiles("static/card.html")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	tmpl.Execute(w, articles)
}
