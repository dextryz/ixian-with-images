package main

import (
	"net/http"
	"regexp"
	"sort"
	"text/template"

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

func BoldHashtags(content string) string {
	re := regexp.MustCompile(`#(\w+)`)
	return re.ReplaceAllString(content, "<b>#$1</b>")
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

    // Apply CSS styling to event content.
    for _, e := range data.Events {
        e.Content = BoldHashtags(e.Content)
    }

    // Newest to latest
    sort.Slice(data.Events, func(i, j int) bool {
		return data.Events[i].CreatedAt > data.Events[j].CreatedAt
	})

	tmpl, err := template.ParseFiles("template/home.html", "template/index.html")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	tmpl.ExecuteTemplate(w, "home.html", data)
}
