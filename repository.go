package main

import (
	"github.com/ffiat/nostr"
)

type Repository struct {
	db map[string]*nostr.Event
}

func (s *Repository) Store(e *nostr.Event) error {
	s.db[e.Id] = e
	return nil
}

func (s *Repository) All() []*nostr.Event {

	var events []*nostr.Event
	for _, v := range s.db {
		events = append(events, v)
	}

	return events
}

func (s *Repository) Find(id string) (*nostr.Event, error) {
	return s.db[id], nil
}
