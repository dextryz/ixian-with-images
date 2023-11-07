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
		_, err := nostr.DecodeBech32(pk)
		if err != nil {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("invalid npub"))
			return
		}
	}
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

        pk, err := nostr.DecodeBech32(npub)
        if err != nil {
            w.WriteHeader(http.StatusOK)
            w.Write([]byte("invalid npub"))
            return
        }

        articles, err = s.repository.FindArticles(pk.(string))
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
