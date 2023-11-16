package main

import (
	"context"
	"log"
	"regexp"

	"github.com/gomarkdown/markdown"
	"github.com/gomarkdown/markdown/html"
	"github.com/gomarkdown/markdown/parser"

	"github.com/dextryz/nostr"
)

const limit = 100

// Abstracts the connection between the local databases and relays.
type Repository struct {
	db *Db
	ws []*Connection
}

func markdownToHtml(md []byte) string {

	text, err := swapLinks(string(md))
	if err != nil {
		log.Fatalln(err)
	}

	// create markdown parser with extensions
	extensions := parser.CommonExtensions
	p := parser.NewWithExtensions(extensions)
	doc := p.Parse([]byte(text))

	// create HTML renderer with extensions
	htmlFlags := html.CommonFlags | html.HrefTargetBlank
	opts := html.RendererOptions{Flags: htmlFlags}
	renderer := html.NewRenderer(opts)

	c := markdown.Render(doc, renderer)

	return string(c)
}

// text := "Click [me](nostr:nevent17915d512457e4bc461b54ba95351719c150946ed4aa00b1d83a263deca69dae) to"
// replacement := `<a href="#" hx-get="article/$2" hx-push-url="true" hx-target="body" hx-swap="outerHTML">$1</a>`
func swapLinks(text string) (string, error) {

	// Define the regular expression pattern to match the markdown-like link
	//pattern := `\[(.*?)\]\((.*?)\)`
	pattern := `\[(.*?)\]\(nostr:(.*?)\)`

	// Compile the regular expression
	re := regexp.MustCompile(pattern)

	// Define the replacement pattern
	replacement := `<a href="#" class="inline"
        hx-get="$2"
        hx-push-url="true"
        hx-target="body"
        hx-swap="outerHTML">$1
    </a>`

	// Replace the matched patterns with the HTML tag
	result := re.ReplaceAllString(text, replacement)

	return result, nil
}

func (s *Repository) Close() error {

	// 1. Close all WS connections.

	// 2. Close database connection.

	return nil
}

// Retrieve article from local cache.
func (s *Repository) Profile(pubkey string) (*Profile, error) {

	// TODO: Convert to filter to send to database

	profile, err := s.db.queryProfileByPubkey(pubkey)
	if err != nil {
		return nil, err
	}

	return profile, nil
}

// Retrieve article from local cache.
func (s *Repository) Article(id string) (*Article, error) {

	article, err := s.db.queryArticleById(id)
	if err != nil {
		return nil, err
	}

	return article, nil
}

func (s *Repository) ArticleByTag(tag string) ([]*Article, error) {

	articles, err := s.db.queryArticleByTag(tag)
	if err != nil {
		return nil, err
	}

	return articles, nil
}

func (s *Repository) FindArticles(pk string) (*Profile, []*Article, error) {

	ctx := context.Background()

	// Retrieve all NIP-23 articles from nostr relays
	events, err := s.pull(pk, nostr.KindArticle)
	if err != nil {
		return nil, nil, err
	}

	// Retrieve user profile from nostr relays
	metadata, err := s.pull(pk, nostr.KindSetMetadata)
	if err != nil {
		return nil, nil, err
	}

	// Only one profile can be pulled per pubkey.

	p, err := nostr.ParseMetadata(*metadata[0])
	if err != nil {
		return nil, nil, err
	}

	profile, err := s.db.StoreProfile(ctx, p, pk)
	if err != nil {
		return nil, nil, err
	}

	// Create article from event and profile, cache and return to handler.
	articles := []*Article{}
	for _, e := range events {
		a, err := s.db.StoreArticle(ctx, e)
		if err != nil {
			return nil, nil, err
		}
		articles = append(articles, a)
	}

	return profile, articles, nil
}

func (s *Repository) CategorizedPeople(id string) (*nostr.Event, error) {

	f := nostr.Filter{
		Ids:   []string{id},
		Kinds: []uint32{3000},
		Limit: limit,
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

	// Make sure the event is a NIP-51 list
	e := events[0]
	if e.Kind != 3000 {
		log.Fatalln("not a NIP-51 categorized people list")
	}

	return e, nil
}

// Pull events from nostr relays.
func (s *Repository) pullArticle(id string) (*nostr.Event, error) {

	f := nostr.Filter{
		Ids:   []string{id},
		Kinds: []uint32{30023},
		Limit: 1,
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

	return events[0], nil
}

// Pull events from nostr relays.
func (s *Repository) pull(pk string, kind uint32) ([]*nostr.Event, error) {

	f := nostr.Filter{
		Authors: []string{pk},
		Kinds:   []uint32{kind},
		Limit:   limit,
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
