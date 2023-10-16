package main

import (
	"fmt"
	"html/template"
	"io"
	"log"
	"net/http"
	"net/mail"
	"net/url"
	"strconv"

	"github.com/ffiat/nostr"
	"github.com/gorilla/mux"
)

// Define template with the functions
var funcMap = template.FuncMap{
	"subtract": func(a, b int) int { return a - b },
	"add":      func(a, b int) int { return a + b },
}

type Data struct {
	Title       string
	Page        int
	Content     string
	Profiles    []*Profile
	SearchQuery string
}

type Handler struct {
	repository Repository
}

func (s *Handler) IndexHandler(w http.ResponseWriter, r *http.Request) {
	http.Redirect(w, r, "/contact", http.StatusMovedPermanently)
}

func (s *Handler) SearchProfile(w http.ResponseWriter, r *http.Request) {

	page, contacts := s.paging(r.URL.Query().Get("page"))

	search := r.URL.Query().Get("q")
	if search != "" {
		header := r.Header.Get("HX-Trigger")
		if header == "search" {
			log.Println("Header search FOUND")
			contacts = s.repository.Search(search)
		}
	}

	tmpl, err := template.New("home").Funcs(funcMap).ParseFiles("template/layout.html", "template/index.html", "template/row.html")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	data := Data{
		Page:        page,
		Profiles:    contacts,
		SearchQuery: search,
	}
	tmpl.ExecuteTemplate(w, "layout.html", data)
}

func (s *Handler) paging(pageStr string) (int, []*nostr.Event) {

	page, _ := strconv.Atoi(pageStr)
	if page < 1 {
		page = 1
	}

	events := s.repository.All()

	if len(events) == 0 {
		log.Fatalln("no events found")
	}

	startIndex := (page - 1) * 10
	endIndex := startIndex + 10
	if endIndex > len(events) {
		endIndex = len(events)
	}

	return page, events[startIndex:endIndex]
}
