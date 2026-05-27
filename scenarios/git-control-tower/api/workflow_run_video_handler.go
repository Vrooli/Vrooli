package main

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/gorilla/mux"

	"github.com/vrooli/api-core/discovery"
)

// handleWorkflowRunVideo proxies a test-genie playbooks-run video to the UI so
// the browser stays single-origin (Plan B Decision 3). It forwards the Range
// header so the <video> element can seek, and copies the streaming-relevant
// response headers back. Structured listing is WorkflowReplayService; this is
// the binary side.
func (s *Server) handleWorkflowRunVideo(w http.ResponseWriter, r *http.Request) {
	runID := strings.TrimSpace(mux.Vars(r)["runId"])
	scenario := strings.TrimSpace(r.URL.Query().Get("scenario"))
	path := strings.TrimSpace(r.URL.Query().Get("path"))
	if scenario == "" || path == "" {
		http.Error(w, "scenario and path are required", http.StatusBadRequest)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 60*time.Second)
	defer cancel()

	base, err := discovery.ResolveScenarioURLDefault(ctx, "test-genie")
	if err != nil {
		http.Error(w, fmt.Sprintf("resolve test-genie: %v", err), http.StatusBadGateway)
		return
	}

	target := fmt.Sprintf("%s/api/v1/scenarios/%s/runs/%s/artifact?path=%s",
		strings.TrimRight(base, "/"),
		url.PathEscape(scenario),
		url.PathEscape(runID),
		url.QueryEscape(path),
	)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, target, nil)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if rng := r.Header.Get("Range"); rng != "" {
		req.Header.Set("Range", rng)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		http.Error(w, fmt.Sprintf("fetch video: %v", err), http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	for _, h := range []string{"Content-Type", "Content-Length", "Content-Range", "Accept-Ranges", "Cache-Control", "Last-Modified", "ETag"} {
		if v := resp.Header.Get(h); v != "" {
			w.Header().Set(h, v)
		}
	}
	w.WriteHeader(resp.StatusCode)
	_, _ = io.Copy(w, resp.Body)
}
