package nodes

import (
	"context"
	"fmt"

	"consistent-hashing/loadbalancer/internal/hash"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
)

func PopulateRingFromDocker(ring *hash.HashRing) {
	ctx := context.Background()

	cli, err := client.NewClientWithOpts(client.FromEnv)
	if err != nil {
		fmt.Println("Docker client error:", err)
		return
	}

	containers, err := cli.ContainerList(ctx, container.ListOptions{})
	if err != nil {
		fmt.Println("List container error:", err)
	}

	for _, c := range containers {
		fmt.Println("Container ", c.ID)
		if c.Labels["service"] == "node" {

			inspect, err := cli.ContainerInspect(ctx, c.ID)
			if err != nil {
				fmt.Println("Inspect error for", c.ID, ":", err)
				return
			}

			netInfo, ok := inspect.NetworkSettings.Networks["consistent-hashing_ringnet"]
			if !ok || netInfo == nil {
				fmt.Println("No ringnet Network for", c.ID)
				continue
			}

			ip := netInfo.IPAddress
			if ip == "" {
				fmt.Println("No IP yet for", c.ID)
				continue
			}

			addr := fmt.Sprintf("http://%s:8080", ip)
			ring.Add(addr, c.ID)
			fmt.Println("Added node to ring:", addr)
		}
	}
}
