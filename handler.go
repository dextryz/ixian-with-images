package main

import (
	"log"
	"net/http"
	"regexp"
	"sort"
	"text/template"
	"time"

	"github.com/ffiat/nostr"
)

type Article struct {
	Id        string
	Image     string
	Title     string
	Tags      []string
	Content   string
	CreatedAt string
	PubKey    string // TODO: change to author with NIP-05
}

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

func BoldHashtags(content string) string {
	re := regexp.MustCompile(`#(\w+)`)
	return re.ReplaceAllString(content, "<b>#$1</b>")
}

func (s *Handler) Home(w http.ResponseWriter, r *http.Request) {

	data := Data{}

	tmpl, err := template.ParseFiles("static/home.html", "static/index.html", "static/article.html", "static/notify.html", "static/profile.html")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	tmpl.ExecuteTemplate(w, "home.html", data)
}

func (s *Handler) Validate(w http.ResponseWriter, r *http.Request) {

	log.Println("PubKey")

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

	events := []*nostr.Event{}
	if npub != "" {
		pk, err := nostr.DecodeBech32(npub)
		if err != nil {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("invalid npub"))
			return
		}
		events = s.repository.FindByPubKey(pk.(string))
	}

	// Newest to latest
	sort.Slice(events, func(i, j int) bool {
		return events[i].CreatedAt > events[j].CreatedAt
	})

	// Apply CSS styling to event content.
	articles := []Article{}
	for _, e := range events {

		// Sample Unix timestamp: 1635619200 (represents 2021-10-30)
		unixTimestamp := int64(e.CreatedAt)

		// Convert Unix timestamp to time.Time
		t := time.Unix(unixTimestamp, 0)

		// Format time.Time to "yyyy-mm-dd"
		createdAt := t.Format("2006-01-02")

		a := Article{
			Id:        e.Id,
			Content:   e.Content,
			CreatedAt: createdAt,
			PubKey:    npub[:15],
		}

		for _, t := range e.Tags {
			if t.Key() == "image" {
				a.Image = t.Value()
			}
			if t.Key() == "title" {
				a.Title = t.Value()
			}
			if t.Key() == "t" {
				a.Tags = append(a.Tags, t.Value())
			}
		}

		articles = append(articles, a)
	}

	tmpl, err := template.ParseFiles("static/article.html")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	tmpl.Execute(w, articles)
}
