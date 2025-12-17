package main

import (
	"fmt"
	"net/http"
	"strings"
	"time"
)

func (s *Server) qdrantDo(req *http.Request) (*http.Response, error) {
	if key := strings.TrimSpace(s.qdrantAPIKey()); key != "" {
		req.Header.Set("api-key", key)
	}

	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("qdrant request failed: %w", err)
	}
	return resp, nil
}

