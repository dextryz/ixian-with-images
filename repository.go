package main

import (
	"context"
	"fmt"
	"log"

	"github.com/dextryz/nostr"
)

// Abstracts the connection between the local databases and relays.
type Repository struct {
	db *Db
	ws []*Connection
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

func (s *Repository) ProfileByArticle(id string) (*Profile, error) {

	profile, err := s.db.queryProfileByArticle(id)
	if err != nil {
		return nil, err
	}

	return profile, nil
}

// Retrieve article from local cache.
func (s *Repository) Article(nid string) (*Article, error) {

	article, err := s.db.queryArticleById(nid)
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

func (s *Repository) FindArticles(npub string) (*Profile, []*Article, error) {

	ctx := context.Background()

	// Retrieve all NIP-23 articles from nostr relays
	events, err := s.reqRelays(npub, nostr.KindArticle)
	if err != nil {
		return nil, nil, err
	}

	// Retrieve user profile from nostr relays
	metadata, err := s.reqRelays(npub, nostr.KindSetMetadata)
	if err != nil {
		return nil, nil, err
	}

	// Only one profile can be pulled per pubkey.
	p, err := nostr.ParseMetadata(*metadata[0])
	if err != nil {
		return nil, nil, err
	}

	profile, err := s.db.StoreProfile(ctx, p, npub)
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
		Limit: s.db.QueryLimit,
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

func (s *Repository) reqRelays(npub string, kind uint32) ([]*nostr.Event, error) {

	prefix, pk, err := nostr.DecodeBech32(npub)
	if err != nil {
		return nil, err
	}

	if prefix != "npub" {
		return nil, fmt.Errorf("public key is not of NIP-19 standard")
	}

	f := nostr.Filter{
		Authors: []string{pk},
		Kinds:   []uint32{kind},
		Limit:   s.db.QueryLimit,
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
