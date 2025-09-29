package hash

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
)

func moveKeys(fromAddr, toAddr string, ring *HashRing) {
	resp, err := http.Get(fromAddr + "/items")
	if err != nil {
		log.Println("Failed to fetch keys from", fromAddr, err)
		return
	}
	defer resp.Body.Close()

	var allKeys map[string]string
	if err := json.NewDecoder(resp.Body).Decode(&allKeys); err != nil {
		log.Println("Failed to decode keys from", fromAddr, err)
		return
	}

	// Collect keys that belong to the target node
	var toMove []map[string]string
	for k, v := range allKeys {
		if ring.Get(k) == toAddr {
			toMove = append(toMove, map[string]string{"key": k, "value": v})
		}
	}

	if len(toMove) == 0 {
		return
	}

	// Send all keys in a single bulk request
	payload, _ := json.Marshal(toMove)
	_, err = http.Post(toAddr+"/items/bulk", "application/json", bytes.NewReader(payload))
	if err != nil {
		log.Println("Failed to POST bulk keys to", toAddr, err)
		return
	}

	// Delete the moved keys from the source node
	for _, kv := range toMove {
		deleteURL := fmt.Sprintf("%s/items/%s", fromAddr, kv["key"])
		req, err := http.NewRequest("DELETE", deleteURL, nil)
		if err != nil {
			log.Println("Failed to create DELETE request for key", kv["key"], ":", err)
			continue
		}

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			log.Println("Failed to delete key", kv["key"], "from", fromAddr, ":", err)
			continue
		}
		resp.Body.Close()
	}
}
