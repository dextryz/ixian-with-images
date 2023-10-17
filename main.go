package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/ffiat/nostr"
	"github.com/gorilla/mux"
)

func StringEnv(key string) string {
	value, ok := os.LookupEnv(key)
	if !ok {
		log.Fatalf("address env variable \"%s\" not set, usual", key)
	}
	return value
}

var (
	CONFIG_NOSTR = StringEnv("CONFIG_NOSTR")
)

func main() {

	cfg, err := DecodeConfig(CONFIG_NOSTR)
	if err != nil {
		log.Fatalf("unable to decode local cfg: %v", err)
	}

	repository := Repository{
		db:  make(map[string]*nostr.Event),
		cfg: cfg,
	}

	handler := Handler{
		repository: repository,
	}

	r := mux.NewRouter()
	r.HandleFunc("/", handler.IndexHandler)
	r.HandleFunc("/events", handler.ListEvents).Methods("GET")

	server := &http.Server{
		Addr:    ":8081",
		Handler: r,
	}

	// Create a channel to listen for OS signals
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	go func() {
		if err = server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("listen: %s\n", err)
		}
	}()

	<-stop

	// Create a context with a timeout for the server's shutdown process
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err = server.Shutdown(ctx); err != nil {
		log.Fatalf("Server Shutdown Failed:%+v", err)
	}
	log.Println("Server gracefully stopped")
}
