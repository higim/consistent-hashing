package sse

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sync"

	"consistent-hashing/loadbalancer/internal/hash"
)

type Event struct {
	Type string      `json:"type"`
	Data interface{} `json:"data"`
}

var SSEClients = make(map[chan string]bool)
var SSEMu sync.Mutex

func SSEHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	ch := make(chan string, 10)
	SSEMu.Lock()
	SSEClients[ch] = true
	SSEMu.Unlock()

	flusher := w.(http.Flusher)
	ctx := r.Context()

	// writer loop
	for {
		select {
		case <-ctx.Done():
			SSEMu.Lock()
			delete(SSEClients, ch)
			SSEMu.Unlock()
			return
		case msg := <-ch:
			fmt.Fprintf(w, "data: %s\n\n", msg)
			flusher.Flush()
		}
	}
}

func Broadcast(v interface{}) {
	b, _ := json.Marshal(v)
	SSEMu.Lock()
	for ch := range SSEClients {
		select {
		case ch <- string(b):
		default:
		}
	}
	SSEMu.Unlock()
}

func BroadcastKey(key, node string) {
	Broadcast(Event{Type: "key", Data: map[string]string{
		"key":  key,
		"node": node,
	}})
}

func BroadcastStats(r *hash.HashRing) {
	stats := r.Info()
	Broadcast(Event{Type: "stats", Data: stats})
}
