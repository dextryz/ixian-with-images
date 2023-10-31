package main

import (
	"log"

	"github.com/ffiat/nostr"
)

type Repository struct {
	db map[string]*nostr.Event
	ws []*Connection
}

func (s *Repository) Store(e *nostr.Event) error {
	s.db[e.Id] = e
	return nil
}

func (s *Repository) Close() error {

	// 1. Close all WS connections.

	// 2. Close database connection.

	return nil
}

func (s *Repository) All() []*nostr.Event {

	var events []*nostr.Event
	for _, v := range s.db {
		events = append(events, v)
	}

	return events
}

// TODO: Cache the pulled events.
func (s *Repository) FindProfile(pk string) *nostr.Event {

	var events []*nostr.Event

	f := nostr.Filter{
		Authors: []string{pk},
		Kinds: []uint32{nostr.KindSetMetadata}, // KindProfile
		Limit: 10,
	}

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

	return events[0]
}

// TODO: Cache the pulled events.
func (s *Repository) FindByPubKey(pk string) []*nostr.Event {

	var events []*nostr.Event

	f := nostr.Filter{
		Authors: []string{pk},
		//Kinds:   []uint32{1}, // KindArticle
		Kinds: []uint32{30023}, // KindArticle
		Limit: 10,
	}

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

	return events
}
