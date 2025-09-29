package nodes

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"

	"consistent-hashing/loadbalancer/internal/hash"
	"consistent-hashing/loadbalancer/internal/sse"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
	"github.com/go-chi/chi/v5"
)

func HandleAddNode(ring *hash.HashRing) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := context.Background()
		cli, _ := client.NewClientWithOpts(client.FromEnv)

		imageName := os.Getenv("NODE_IMAGE")

		resp, err := cli.ContainerCreate(ctx, &container.Config{
			Image:  imageName,
			Labels: map[string]string{"service": "node"},
		}, &container.HostConfig{
			NetworkMode: "consistent-hashing_ringnet",
		}, nil, nil, "")

		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}

		if err := cli.ContainerStart(ctx, resp.ID, container.StartOptions{}); err != nil {
			http.Error(w, err.Error(), 500)
			return
		}

		inspect, _ := cli.ContainerInspect(ctx, resp.ID)
		ip := inspect.NetworkSettings.Networks["consistent-hashing_ringnet"].IPAddress // Get name from env and think on using dns names
		addr := fmt.Sprintf("http://%s:8080", ip)

		key := ring.Add(addr, resp.ID)

		go sse.Broadcast(ring)

		json.NewEncoder(w).Encode(map[string]interface{}{
			"id":   resp.ID,
			"addr": addr,
			"key":  key,
		})
	}
}

func HandleRemoveNode(ring *hash.HashRing) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		fmt.Println("Incoming DELETE Request:", r.Method, r.URL.Path)
		id := chi.URLParam(r, "id")

		ctx := context.Background()
		cli, _ := client.NewClientWithOpts(client.FromEnv)

		_, err := cli.ContainerInspect(ctx, id)
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}

		ring.Remove(id)
		go sse.BroadcastStats(ring)

		cli.ContainerStop(ctx, id, container.StopOptions{})
		cli.ContainerRemove(ctx, id, container.RemoveOptions{})

		w.WriteHeader(http.StatusOK)
	}
}
