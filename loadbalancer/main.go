package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"consistent-hashing/loadbalancer/internal/hash"
	"consistent-hashing/loadbalancer/internal/nodes"
	"consistent-hashing/loadbalancer/internal/proxy"
	"consistent-hashing/loadbalancer/internal/sse"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
)

func main() {
	ring := hash.NewHashRing(1024)

	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"*"}, // Allow all origins, or specify your domains
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token"},
		ExposedHeaders:   []string{"Link"},
		AllowCredentials: true,
		MaxAge:           300, // Maximum value not ignored by any of major browsers
	}))

	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok"}`))
		fmt.Println("Health check called")
	})

	r.Get("/ring", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(ring.Info())
	})

	r.Route("/items", func(r chi.Router) {

		r.Get("/{key}", func(w http.ResponseWriter, r *http.Request) {
			key := chi.URLParam(r, "key")
			nodeAddr := ring.Get(key)
			if nodeAddr == "" {
				http.Error(w, "No nodes available", http.StatusInternalServerError)
				return
			}

			proxy.ProxyRequest(nodeAddr, w, r)

		})

		r.Post("/", func(w http.ResponseWriter, r *http.Request) {
			// Read body bytes
			bodyBytes, err := io.ReadAll(r.Body)
			if err != nil {
				http.Error(w, "cannot read body", http.StatusBadRequest)
				return
			}
			r.Body.Close()

			// Decode the body to extract the key
			var payload struct {
				Key   string `json:"key"`
				Value string `json:"value"`
			}
			if err := json.Unmarshal(bodyBytes, &payload); err != nil {
				http.Error(w, "invalid JSON", http.StatusBadRequest)
				return
			}
			if payload.Key == "" {
				http.Error(w, "key required", http.StatusBadRequest)
				return
			}

			// Restore the body for proxying
			r.Body = io.NopCloser(bytes.NewReader(bodyBytes))
			r.ContentLength = int64(len(bodyBytes))
			r.Header.Set("Content-Length", fmt.Sprint(len(bodyBytes)))

			nodeAddr := ring.Get(payload.Key)
			if nodeAddr == "" {
				http.Error(w, "no nodes available", http.StatusInternalServerError)
				return
			}

			proxy.ProxyRequest(nodeAddr, w, r)

			go func() {
				sse.BroadcastKey(payload.Key, nodeAddr)
				sse.BroadcastStats(ring)
			}()
		})
	})

	r.Post("/nodes", nodes.HandleAddNode(ring))
	r.Delete("/nodes/{id}", nodes.HandleRemoveNode(ring))
	r.Get("/events", sse.SSEHandler)

	nodes.PopulateRingFromDocker(ring)

	fmt.Println("Load balancer listening on :8080")
	err := http.ListenAndServe(":8080", r)
	if err != nil {
		fmt.Println("Failed to start server:", err)
	}
}
