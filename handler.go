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

func (s *Handler) Tag(w http.ResponseWriter, r *http.Request) {

	vars := mux.Vars(r)
    hashtag := vars["ht"]

    articles, err := s.repository.ArticleByTag(hashtag)
    if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }

    cards := []*Note{}
    for _, a := range articles {
        n := &Note{
            Article: a,
            Profile: &Profile{},
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

func (s *Handler) Profile(w http.ResponseWriter, r *http.Request) {

	vars := mux.Vars(r)
    pubkey := vars["pk"]

// 	_, pubkey, err := nostr.DecodeBech32(vars["pk"])
// 	if err != nil {
// 		http.Error(w, err.Error(), http.StatusInternalServerError)
// 		return
// 	}

    log.Printf("Pulling profile with pubkey: %s", pubkey)

	profile, err := s.repository.Profile(pubkey)
	if err != nil {
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

	tmpl.Execute(w, profile)
}

func (s *Handler) Article(w http.ResponseWriter, r *http.Request) {

	vars := mux.Vars(r)
    id := vars["id"]

// 	// TODO: For now prefix should only be 'note'
// 	_, event, err := nostr.DecodeBech32(vars["id"])
// 	if err != nil {
// 		http.Error(w, err.Error(), http.StatusInternalServerError)
// 		return
// 	}

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

func (s *Handler) Home(w http.ResponseWriter, r *http.Request) {

	tmpl, err := template.ParseFiles("static/home.html", "static/card.html")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	notes := []*Note{}
	err = tmpl.ExecuteTemplate(w, "home.html", notes)
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

	notes := []*Note{}

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

		npub := strings.TrimPrefix(search, nostr.Prefix)

		_, pk, err := nostr.DecodeBech32(npub)
		if err != nil {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("invalid npub"))
			return
		}

		profile, articles, err := s.repository.FindArticles(pk)
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
