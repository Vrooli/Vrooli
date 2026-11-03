package aimoneyfixtures

import (
	"encoding/json"
	"net"
	"net/http"
	"sync"
)

// FakeRouter serves a minimal OpenAI-compatible response on an OS-assigned
// loopback port. Close it with Close; it never opens a public listener.
type FakeRouter struct {
	Server   *http.Server
	Listener net.Listener
	mu       sync.Mutex
	Requests []map[string]any
	Response map[string]any
}

func NewFakeRouter(response map[string]any) (*FakeRouter, error) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return nil, err
	}
	f := &FakeRouter{Listener: listener, Response: response}
	mux := http.NewServeMux()
	mux.HandleFunc("/v1/chat/completions", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		var body map[string]any
		if json.NewDecoder(r.Body).Decode(&body) != nil {
			http.Error(w, "invalid json", http.StatusBadRequest)
			return
		}
		f.mu.Lock()
		f.Requests = append(f.Requests, body)
		resp := f.Response
		f.mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	})
	f.Server = &http.Server{Handler: mux}
	go func() { _ = f.Server.Serve(listener) }()
	return f, nil
}
func (f *FakeRouter) URL() string {
	if f == nil || f.Listener == nil {
		return ""
	}
	return "http://" + f.Listener.Addr().String()
}
func (f *FakeRouter) Close() error {
	if f == nil || f.Server == nil {
		return nil
	}
	return f.Server.Close()
}
