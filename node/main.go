package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"sync"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/spaolacci/murmur3"
)

type Slot struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

type NodeStore struct {
	sync.RWMutex
	slots []Slot
}

func NewNodeStore(numSlots int) *NodeStore {
	return &NodeStore{
		slots: make([]Slot, numSlots),
	}
}

func (ns *NodeStore) hashKeyToSlot(key string) int {
	return int(murmur3.Sum32([]byte(key)) % uint32(len(ns.slots)))
}

func (ns *NodeStore) Put(key, value string) {
	ns.Lock()
	defer ns.Unlock()
	idx := ns.hashKeyToSlot(key)
	ns.slots[idx] = Slot{Key: key, Value: value}
}

func (ns *NodeStore) Get(key string) (string, bool) {
	ns.RLock()
	defer ns.RUnlock()
	idx := ns.hashKeyToSlot(key)
	slot := ns.slots[idx]
	if slot.Key == key {
		return slot.Value, true
	}
	return "", false
}

func (ns *NodeStore) AllSlots() []Slot {
	ns.RLock()
	defer ns.RUnlock()
	copied := make([]Slot, len(ns.slots))
	copy(copied, ns.slots)
	return copied
}

func countFilled(slots []Slot) int {
	c := 0
	for _, s := range slots {
		if s.Key != "" {
			c++
		}
	}
	return c
}

func main() {
	nodeID, _ := os.Hostname()
	fmt.Println("Starting node:", nodeID)

	numSlots := 100 // fixed but do it configurable in docker compose
	store := NewNodeStore(numSlots)

	r := chi.NewRouter()
	r.Use(middleware.Logger)

	r.Route("/items", func(r chi.Router) {
		r.Get("/{key}", func(w http.ResponseWriter, r *http.Request) {
			key := chi.URLParam(r, "key")

			value, ok := store.Get(key)
			if !ok {
				http.Error(w, "key not found", http.StatusNotFound)
				return
			}
			json.NewEncoder(w).Encode(map[string]string{"value": value})
		})

		r.Post("/", func(w http.ResponseWriter, r *http.Request) {
			var payload struct {
				Key   string `json:"key"`
				Value string `json:"value"`
			}

			if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
				http.Error(w, "invalid json", http.StatusBadRequest)
				return
			}

			if payload.Key == "" {
				http.Error(w, "key required", http.StatusBadRequest)
				return
			}

			store.Put(payload.Key, payload.Value)
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(fmt.Sprintf("Stored key '%s'", payload.Key)))
		})

		r.Delete("/{key}", func(w http.ResponseWriter, r *http.Request) {
			key := chi.URLParam(r, "key")
			store.Lock()
			defer store.Unlock()

			idx := store.hashKeyToSlot(key)
			slot := store.slots[idx]
			if slot.Key == key {
				store.slots[idx] = Slot{}
			}
			w.WriteHeader(http.StatusOK)
		})

		r.Get("/", func(w http.ResponseWriter, r *http.Request) {
			store.RLock()
			defer store.RUnlock()

			keys := make(map[string]string)
			for _, s := range store.slots {
				if s.Key != "" {
					keys[s.Key] = s.Value
				}
			}
			json.NewEncoder(w).Encode(keys)
		})

		r.Post("/bulk", func(w http.ResponseWriter, r *http.Request) {
			var payload []Slot
			if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
				http.Error(w, "invalid json", http.StatusBadRequest)
				return
			}

			for _, s := range payload {
				if s.Key != "" {
					store.Put(s.Key, s.Value)
				}
			}
			w.WriteHeader(http.StatusOK)
		})
	})

	r.Get("/stats", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]int{
			"slots":  len(store.AllSlots()),
			"filled": countFilled(store.AllSlots()),
		})
	})

	http.ListenAndServe(":8080", r)
}
