package proxy

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
)

func ProxyRequest(nodeAddr string, w http.ResponseWriter, r *http.Request) {
	target, err := url.Parse(nodeAddr)
	if err != nil {
		http.Error(w, "invalid node address", http.StatusInternalServerError)
		return
	}

	var bodyBytes []byte
	if r.Body != nil {
		bodyBytes, err = io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "cannot read request body", http.StatusBadRequest)
			return
		}
		r.Body.Close()
	}

	// Restore the Body so it can be read by the proxy
	r.Body = io.NopCloser(bytes.NewReader(bodyBytes))
	r.ContentLength = int64(len(bodyBytes))
	r.Header.Set("Content-Length", fmt.Sprint(len(bodyBytes)))

	proxy := httputil.NewSingleHostReverseProxy(target)

	log.Printf("Proxy request to %s", nodeAddr)

	proxy.ServeHTTP(w, r)
}
