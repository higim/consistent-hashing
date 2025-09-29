package hash

import (
	"encoding/json"
	"net/http"
	"sort"

	"github.com/spaolacci/murmur3"
)

type Node struct {
	NodeID string `json:"nodeID"`
	Addr   string `json:"addr"`
	Key    int    `json:"key"`
}

type HashRing struct {
	size    int
	hashMap map[int]Node
	keys    []int
}

func NewHashRing(size int) *HashRing {
	return &HashRing{
		hashMap: make(map[int]Node),
		size:    size,
	}
}

func (r *HashRing) hashKey(key string) int {
	return int(murmur3.Sum32([]byte(key)) % uint32(r.size))
}

func (r *HashRing) Add(addr, nodeID string) int {
	key := r.hashKey(addr)
	node := Node{
		NodeID: nodeID,
		Addr:   addr,
		Key:    key,
	}
	r.hashMap[key] = node

	r.keys = append(r.keys, key)
	sort.Ints(r.keys)

	// Find successor
	idx := sort.Search(len(r.keys), func(i int) bool { return r.keys[i] > key })
	if idx == len(r.keys) {
		idx = 0
	}
	succKey := r.keys[idx]
	succNode := r.hashMap[succKey]

	// Migrate keys from successor to new node
	go moveKeys(succNode.Addr, addr, r)

	return key
}

func (r *HashRing) Remove(nodeID string) {
	var removeKey int = -1
	for k, n := range r.hashMap {
		if n.NodeID == nodeID {
			removeKey = k
			break
		}
	}
	if removeKey == -1 {
		return
	}

	// Find successor node
	idx := sort.Search(len(r.keys), func(i int) bool { return r.keys[i] > removeKey })
	if idx == len(r.keys) {
		idx = 0
	}
	succKey := r.keys[idx]
	succNode := r.hashMap[succKey]
	fromNode := r.hashMap[removeKey]

	// Migrate keys from the node being removed to successor
	go moveKeys(fromNode.Addr, succNode.Addr, r)

	// Remove node from ring
	delete(r.hashMap, removeKey)
	for i, k := range r.keys {
		if k == removeKey {
			r.keys = append(r.keys[:i], r.keys[i+1:]...)
			break
		}
	}
}

func (r *HashRing) Get(key string) string {
	if len(r.hashMap) == 0 {
		return ""
	}
	h := r.hashKey(key)

	idx := sort.Search(len(r.keys), func(i int) bool { return r.keys[i] >= h })
	if idx == len(r.keys) {
		idx = 0 // wrap around the ring
	}

	return r.hashMap[r.keys[idx]].Addr
}

func (r *HashRing) Info() map[string]interface{} {
	rInfo := make(map[string]interface{})
	rInfo["size"] = r.size

	var nodes []map[string]interface{}
	for _, n := range r.hashMap {
		nodeData := map[string]interface{}{
			"nodeID": n.NodeID,
			"addr":   n.Addr,
			"key":    n.Key,
		}

		resp, err := http.Get(n.Addr + "/items")
		if err == nil {
			defer resp.Body.Close()
			var kv map[string]string
			if json.NewDecoder(resp.Body).Decode(&kv) == nil {
				nodeData["keys"] = kv
			}
		}

		if resp, err := http.Get(n.Addr + "/stats"); err == nil {
			defer resp.Body.Close()
			var stats map[string]int
			if json.NewDecoder(resp.Body).Decode(&stats) == nil {
				nodeData["slots"] = stats["slots"]
				nodeData["filled"] = stats["filled"]
			}
		}

		nodes = append(nodes, nodeData)
	}
	rInfo["nodes"] = nodes
	return rInfo
}
