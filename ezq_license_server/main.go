package main

import (
	"encoding/json"
	"log"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"strings"
	"time"
)

const defaultUpstream = "https://ezq-license-server.ezq-license-03dd9a47d121811a.workers.dev"

func jsonOut(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}

func probeUpstream(target *url.URL, transport *http.Transport) {
	ctx, cancel := context.WithTimeout(context.Background(), 25*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, strings.TrimRight(target.String(), "/")+"/health", nil)
	if err != nil {
		log.Printf("upstream startup probe build failed: %v", err)
		return
	}
	resp, err := (&http.Client{Transport: transport}).Do(req)
	if err != nil {
		log.Printf("upstream startup probe failed: %v", err)
		return
	}
	defer resp.Body.Close()
	log.Printf("upstream startup probe success: status=%d host=%s", resp.StatusCode, target.Host)
}

func main() {
	raw := strings.TrimSpace(os.Getenv("UPSTREAM_URL"))
	if raw == "" {
		raw = defaultUpstream
	}
	target, err := url.Parse(strings.TrimRight(raw, "/"))
	if err != nil || target.Scheme != "https" || target.Host == "" {
		log.Fatalf("invalid UPSTREAM_URL: %q", raw)
	}

	transport := &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		DialContext: (&net.Dialer{
			Timeout:   12 * time.Second,
			KeepAlive: 30 * time.Second,
		}).DialContext,
		ForceAttemptHTTP2:     true,
		MaxIdleConns:          100,
		MaxIdleConnsPerHost:   32,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   15 * time.Second,
		ResponseHeaderTimeout: 30 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
	}

	go probeUpstream(target, transport)\n\n\tproxy := httputil.NewSingleHostReverseProxy(target)
	proxy.Transport = transport
	origDirector := proxy.Director
	proxy.Director = func(r *http.Request) {
		origDirector(r)
		r.Host = target.Host
		r.Header.Set("X-EZQ-Gateway", "render-d1-v1")
	}
	proxy.ErrorHandler = func(w http.ResponseWriter, r *http.Request, e error) {
		log.Printf("proxy error %s %s: %v", r.Method, r.URL.Path, e)
		jsonOut(w, http.StatusBadGateway, map[string]any{
			"ok":      false,
			"status":  "upstream_unreachable",
			"error":   "Cloudflare D1 backend temporarily unreachable",
			"gateway": "render-d1-v1",
		})
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/gateway-health", func(w http.ResponseWriter, r *http.Request) {
		jsonOut(w, 200, map[string]any{
			"ok":       true,
			"gateway":  "render-d1-v1",
			"upstream": target.Host,
		})
	})
	mux.Handle("/", proxy)

	port := strings.TrimSpace(os.Getenv("PORT"))
	if port == "" {
		port = "10000"
	}
	log.Printf("EZQ Render D1 gateway listening on :%s -> %s", port, target.String())
	log.Fatal(http.ListenAndServe(":"+port, mux))
}
