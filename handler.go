package main

import (
	"fmt"
	"log"
	"net/http"
	"strings"
	"text/template"

	"github.com/dextryz/nostr"
	"github.com/gorilla/mux"
)

type Note struct {
	Article *Article
	Profile *Profile
}

type Handler struct {
	repository Repository
}

func (s *Handler) Home(w http.ResponseWriter, r *http.Request) {

	tmpl, err := template.ParseFiles("static/index.html", "static/card.html")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	notes := []*Note{}
	err = tmpl.ExecuteTemplate(w, "index.html", notes)
	if err != nil {
		fmt.Println("Error executing template:", err)
	}
}

func (s *Handler) Tag(w http.ResponseWriter, r *http.Request) {

	log.Println("Requesting hashtag articles")

	vars := mux.Vars(r)
	hashtag := vars["ht"]

	articles, err := s.repository.ArticleByTag(hashtag)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	cards := []*Note{}

	for _, a := range articles {

		p, err := s.repository.ProfileByArticle(a.Id)
		if err != nil {
			log.Println(err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		n := &Note{
			Article: a,
			Profile: p,
		}

		cards = append(cards, n)
	}

	tmpl, err := template.ParseFiles("static/taglist.html")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	tmpl.Execute(w, cards)
}

// 1. Pull lists
// 1. Pull bookmarks
// 1. Pull hightlight
// 1. Pull graphs
// 1. Pull orphans

func (s *Handler) Profile(w http.ResponseWriter, r *http.Request) {

	vars := mux.Vars(r)
	pubkey := vars["npub"]

	log.Printf("Pulling profile with npub: %s", pubkey)

	profile, err := s.repository.Profile(pubkey)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	tmpl, err := template.ParseFiles("static/profile.html")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	tmpl.Execute(w, profile)
}

func (s *Handler) Article(w http.ResponseWriter, r *http.Request) {

	vars := mux.Vars(r)
	id := vars["nid"]

	article, err := s.repository.Article(id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	tmpl, err := template.ParseFiles("static/article.html")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	tmpl.Execute(w, article)
}

func (s *Handler) Validate(w http.ResponseWriter, r *http.Request) {

	pk := r.URL.Query().Get("search")

	if pk != "" {

		prefix, _, err := nostr.DecodeBech32(pk)

		if err != nil {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`<span class="message error">Invalid entity</span>`))
			return
		}

		if prefix[0] != 'n' {
			log.Println("start with npub")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`<span class="message error">Start with npub</span>`))
			return
		}

        // Add text to show valid if you want to.
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`<span class="message success"> </span>`))
	}
}

func (s *Handler) ListEvents(w http.ResponseWriter, r *http.Request) {

	search := r.URL.Query().Get("search")

	notes := []*Note{}

	if strings.HasPrefix(search, nostr.UriEvent) {

		// Pull the NIP-51 list event using event ID.
		event, err := s.repository.CategorizedPeople(search)
		if err != nil {
			panic(err)
		}

		// Loop all authors (pubkeys) in NIP-51 event tags (list).
		for _, value := range event.Tags {

			t, v := value[0], value[1]

			if t == "p" {
				profile, articles, err := s.repository.FindArticles(v)
				if err != nil {
					http.Error(w, err.Error(), http.StatusInternalServerError)
					return
				}
				for _, a := range articles {
					n := &Note{
						Article: a,
						Profile: profile,
					}
					notes = append(notes, n)
				}
			}
		}
	} else if strings.HasPrefix(search, nostr.UriPub) {

		log.Println("pull profile NIP-01")

		profile, articles, err := s.repository.FindArticles(search)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		for _, a := range articles {
			n := &Note{
				Article: a,
				Profile: profile,
			}
			notes = append(notes, n)
		}
	}

	tmpl, err := template.ParseFiles("static/card.html")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	tmpl.Execute(w, notes)
}
