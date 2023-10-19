package main

import (
	"log"
	"net/http"
	"regexp"
	"sort"
	"text/template"

	"github.com/ffiat/nostr"
)

type Data struct {
	Events      []*nostr.Event
	SearchQuery string
    Error string
    Invalid bool
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

	tmpl, err := template.ParseFiles("template/home.html", "template/index.html", "template/events.html")
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

	pk := r.URL.Query().Get("pubkey")

	data := Data{}

	if pk != "" {

        pk, err := nostr.DecodeBech32(pk)
        if err != nil {
            w.WriteHeader(http.StatusOK)
            w.Write([]byte("invalid npub"))
            return
        }

		data.Events = s.repository.FindByPubKey(pk.(string))
	}

    // Apply CSS styling to event content.
    for _, e := range data.Events {
        e.Content = BoldHashtags(e.Content)
    }

    // Newest to latest
    sort.Slice(data.Events, func(i, j int) bool {
		return data.Events[i].CreatedAt > data.Events[j].CreatedAt
	})

    log.Println(len(data.Events))

	tmpl, err := template.ParseFiles("template/events.html")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	tmpl.Execute(w, data)
}
