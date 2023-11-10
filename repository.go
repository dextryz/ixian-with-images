package main

import (
	"log"
	"regexp"
	"time"

	"github.com/gomarkdown/markdown"
	"github.com/gomarkdown/markdown/html"
	"github.com/gomarkdown/markdown/parser"

	"github.com/ffiat/nostr"
)

const limit = 100

type Profile struct {
	Name       string
	About      string
	Website    string
	Banner     string
	Picture    string
	Identifier string
	Articles   int
	Notes      int
	Lists      int
	Bookmarks  int
	Highlights int
	Graphs     int
	Orphans    int
}

type Article struct {
	// NIP-19 note id (note1fntxtkcy9pjwucqwa9mddn7v03wwwsu9j330jj350nvhpky2tuaspk6nqc)
	Id        string
	Image     string
	Title     string
	Tags      []string
	Content   string
	CreatedAt string
	// TODO
	//Profile   *nostr.Profile
	Profile *Profile
}

type Repository struct {

	// Basic ID index cache
	db map[string]*Article

	// Cache using a hashtag as the index key [tag: []article.ID] stores a list of article ids
	hashtags map[string][]string

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
	replacement := `<a href="#" class="inline" hx-get="$2" hx-push-url="true" hx-target="body" hx-swap="outerHTML">$1</a>`

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
func (s *Repository) Article(id string) (*Article, error) {

	a, ok := s.db[id]

	// If not cached, try and pull and cache indivdual aricle
	if !ok {

		// TODO: this is DRY With FindARticles
		event, err := s.pullArticle(id)
		if err != nil {
			return nil, err
		}

		// Retrieve user profile from nostr relays
		eventsMetadata, err := s.pull(event.PubKey, nostr.KindSetMetadata)
		if err != nil {
			return nil, err
		}

		profile := eventsMetadata[0]
		p, err := nostr.ParseMetadata(*profile)
		if err != nil {
			return nil, err
		}

		// Create article from event and profile, cache and return to handler.
		articleCached, err := s.createArticle(p, event)
		if err != nil {
			return nil, err
		}

		return articleCached, nil
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
		a, err := s.createArticle(p, e)
		if err != nil {
			return nil, err
		}
		articles = append(articles, a)
	}

	return articles, nil
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

// Create and store article in local cache.
func (s *Repository) createArticle(p *nostr.Profile, e *nostr.Event) (*Article, error) {

	// Sample Unix timestamp: 1635619200 (represents 2021-10-30)
	unixTimestamp := int64(e.CreatedAt)

	// Convert Unix timestamp to time.Time
	t := time.Unix(unixTimestamp, 0)

	// Format time.Time to "yyyy-mm-dd"
	createdAt := t.Format("2006-01-02")

	// Encode NIP-01 event id to NIP-19 note id
	id, err := nostr.EncodeNote(e.Id)
	if err != nil {
		return nil, err
	}

	profile := &Profile{
		Name:       p.Name,
		About:      p.About,
		Website:    p.Website,
		Banner:     p.Banner,
		Picture:    p.Picture,
		Identifier: p.Nip05,
		Articles:   100,
		Notes:      100,
		Lists:      100,
		Bookmarks:  100,
		Highlights: 100,
		Graphs:     100,
		Orphans:    100,
	}

	// Create article with Markdown content converted to HTML.
	a := &Article{
		Id:        id,
		Content:   markdownToHtml([]byte(e.Content)),
		CreatedAt: createdAt,
		Profile:   profile,
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

			// Add to hashtag index.
			s.hashtags[t.Value()] = append(s.hashtags[t.Value()], a.Id)
		}
	}

    s.db[a.Id] = a

	return a, nil
}

// Pull events from nostr relays.
func (s *Repository) pullArticle(id string) (*nostr.Event, error) {

	f := nostr.Filter{
		Ids:   []string{id},
		Kinds: []uint32{30026},
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
