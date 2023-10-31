package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"regexp"
	"sort"
	"text/template"
	"time"

	"github.com/gomarkdown/markdown"
	"github.com/gomarkdown/markdown/ast"
	"github.com/gomarkdown/markdown/html"
	"github.com/gomarkdown/markdown/parser"

	"github.com/ffiat/nostr"
	"github.com/gorilla/mux"
)

type Article struct {
	Id        string
	Image     string
	Title     string
	Tags      []string
	Content   string
	CreatedAt string
	PubKey    string // TODO: change to author with NIP-05
	Profile   *nostr.Profile
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

var printAst = false

func mdToHTML(md []byte) []byte {
	// create markdown parser with extensions
	extensions := parser.CommonExtensions
	p := parser.NewWithExtensions(extensions)
	doc := p.Parse(md)

	if printAst {
		fmt.Print("--- AST tree:\n")
		ast.Print(os.Stdout, doc)
		fmt.Print("\n")
	}

	// create HTML renderer with extensions
	htmlFlags := html.CommonFlags | html.HrefTargetBlank
	opts := html.RendererOptions{Flags: htmlFlags}
	renderer := html.NewRenderer(opts)

	return markdown.Render(doc, renderer)
}

func (s *Handler) IndexHandler(w http.ResponseWriter, r *http.Request) {
	http.Redirect(w, r, "/home", http.StatusMovedPermanently)
}

func BoldHashtags(content string) string {
	re := regexp.MustCompile(`#(\w+)`)
	return re.ReplaceAllString(content, "<b>#$1</b>")
}

func (s *Handler) Article(w http.ResponseWriter, r *http.Request) {

	vars := mux.Vars(r)
	id := vars["id"]

    log.Printf("Fetching article with ID: %s", id)

	article := s.repository.FindArticle(id)

	md := []byte(article.Content)
	article.Content = string(mdToHTML(md))

	log.Println(article.Content)

	tmpl, err := template.ParseFiles("static/article.html")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	tmpl.Execute(w, article)
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

	profile := nostr.Event{}
	events := []*nostr.Event{}
	if npub != "" {
		pk, err := nostr.DecodeBech32(npub)
		if err != nil {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("invalid npub"))
			return
		}
		profile = *s.repository.FindProfile(pk.(string))
		events = s.repository.FindByPubKey(pk.(string))
	}

	// Newest to latest
	sort.Slice(events, func(i, j int) bool {
		return events[i].CreatedAt > events[j].CreatedAt
	})

	p, err := nostr.ParseMetadata(profile)
	if err != nil {
		log.Fatalln(err)
	}

	// Apply CSS styling to event content.
	articles := []*Article{}
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
			Profile:   p,
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

		articles = append(articles, &a)
	}

	tmpl, err := template.ParseFiles("static/card.html")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	tmpl.Execute(w, articles)
}
