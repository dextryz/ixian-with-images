package main

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/gomarkdown/markdown"
	"github.com/gomarkdown/markdown/ast"
	"github.com/gomarkdown/markdown/html"
	"github.com/gomarkdown/markdown/parser"

	"github.com/ffiat/nostr"
)

type Article struct {
	Id        string
	Image     string
	Title     string
	Tags      []string
	Content   string
	CreatedAt string
	Profile   *nostr.Profile
}

type Repository struct {
	db map[string]*Article
	ws []*Connection
}

var printAst = false

func markdownToHtml(md []byte) string {
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

	c := markdown.Render(doc, renderer)

	return string(c)
}

func (s *Repository) Close() error {

	// 1. Close all WS connections.

	// 2. Close database connection.

	return nil
}

// Retrieve article from local cache.
func (s *Repository) Article(id string) (*Article, error) {
	a, ok := s.db[id]
	if !ok {
		return nil, fmt.Errorf("article not found (id: %s)", id)
	}
	return a, nil
}

func (s *Repository) FindArticles(pk string) ([]*Article, error) {

	// Retrieve all NIP-23 articles from nostr relays
	eventsArticle, err := s.pull(pk, 30023)
	if err != nil {
		return nil, err
	}

	// Retrieve user profile from nostr relays
	eventsMetadata, err := s.pull(pk, nostr.KindSetMetadata)
	if err != nil {
		return nil, err
	}

	profile := eventsMetadata[0]
	p, err := nostr.ParseMetadata(*profile)
	if err != nil {
		return nil, err
	}

	// Create article from event and profile, cache and return to handler.
	articles := []*Article{}
	for _, e := range eventsArticle {
		a, err := s.cache(p, e)
		if err != nil {
			return nil, err
		}
		articles = append(articles, a)
	}

	return articles, nil
}

// Create and store article in local cache.
func (s *Repository) cache(p *nostr.Profile, e *nostr.Event) (*Article, error) {

	// Sample Unix timestamp: 1635619200 (represents 2021-10-30)
	unixTimestamp := int64(e.CreatedAt)

	// Convert Unix timestamp to time.Time
	t := time.Unix(unixTimestamp, 0)

	// Format time.Time to "yyyy-mm-dd"
	createdAt := t.Format("2006-01-02")

	// Create article with Markdown content converted to HTML.
	a := &Article{
		Id:        e.Id,
		Content:   markdownToHtml([]byte(e.Content)),
		CreatedAt: createdAt,
		Profile:   p,
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

	s.db[e.Id] = a

	return a, nil
}

// Pull events from nostr relays.
func (s *Repository) pull(pk string, kind uint32) ([]*nostr.Event, error) {

	f := nostr.Filter{
		Authors: []string{pk},
		Kinds:   []uint32{kind},
		Limit:   10,
	}

	events := []*nostr.Event{}

	// Subscribe the PubKey to every open connection to a relay.
	for _, ws := range s.ws {

		sub, err := ws.Subscribe(nostr.Filters{f})
		if err != nil {
			log.Fatalf("\nunable to subscribe: %#v", err)
		}

		orDone := func(done <-chan struct{}, stream <-chan *nostr.Event) <-chan *nostr.Event {
			valStream := make(chan *nostr.Event)
			go func() {
				defer close(valStream)
				for {
					select {
					case <-done:
						return
					case v, ok := <-stream:
						if ok == false {
							return
						}
						valStream <- v
					}
				}
			}()
			return valStream
		}

		for e := range orDone(sub.Done, sub.EventStream) {
			events = append(events, e)
		}

		//cc.Close()
	}

	return events, nil
}
