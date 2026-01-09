package main

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"

	"knowledge-observatory/internal/services/graph"
)

type GraphRequest struct {
	Center     string   `json:"center_concept"`
	Collection string   `json:"collection,omitempty"`
	Namespaces []string `json:"namespaces,omitempty"`
	Visibility []string `json:"visibility,omitempty"`
	Tags       []string `json:"tags,omitempty"`
	Depth      int      `json:"depth,omitempty"`
	Limit      int      `json:"limit,omitempty"`
	Threshold  float64  `json:"threshold,omitempty"`
}

func (s *Server) handleGraph(w http.ResponseWriter, r *http.Request) {
	start := time.Now()

	var req GraphRequest
	if r.Method == http.MethodGet {
		req.Center = strings.TrimSpace(r.URL.Query().Get("center_concept"))
		req.Collection = strings.TrimSpace(r.URL.Query().Get("collection"))
		req.Namespaces = splitCommaList(r.URL.Query().Get("namespaces"))
		req.Tags = splitCommaList(r.URL.Query().Get("tags"))
		req.Visibility = splitCommaList(r.URL.Query().Get("visibility"))
		req.Depth = parseIntQuery(r, "depth")
		req.Limit = parseIntQuery(r, "limit")
		req.Threshold = parseFloatQuery(r, "threshold")
	} else {
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			s.respondError(w, http.StatusBadRequest, "Invalid request body")
			return
		}
	}

	req.Namespaces = normalizeStringList(req.Namespaces)
	req.Tags = normalizeStringList(req.Tags)
	visibility, err := normalizeVisibilityListStrict(req.Visibility)
	if err != nil {
		s.respondError(w, http.StatusBadRequest, err.Error())
		return
	}
	req.Visibility = visibility

	if s == nil || s.graphService == nil {
		s.respondError(w, http.StatusInternalServerError, "Graph service unavailable")
		return
	}

	out, err := s.graphService.Graph(r.Context(), graph.Request{
		Center:     req.Center,
		Collection: req.Collection,
		Namespaces: req.Namespaces,
		Visibility: req.Visibility,
		Tags:       req.Tags,
		Depth:      req.Depth,
		Limit:      req.Limit,
		Threshold:  req.Threshold,
	})
	if err != nil {
		s.respondError(w, http.StatusBadRequest, err.Error())
		return
	}
	out.TookMS = maxInt64(out.TookMS, time.Since(start).Milliseconds())

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(out)
}

func splitCommaList(value string) []string {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}
	parts := strings.Split(value, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		out = append(out, strings.TrimSpace(p))
	}
	return out
}

func parseIntQuery(r *http.Request, key string) int {
	raw := strings.TrimSpace(r.URL.Query().Get(key))
	if raw == "" {
		return 0
	}
	v, err := strconv.Atoi(raw)
	if err != nil {
		return 0
	}
	return v
}

func parseFloatQuery(r *http.Request, key string) float64 {
	raw := strings.TrimSpace(r.URL.Query().Get(key))
	if raw == "" {
		return 0
	}
	v, err := strconv.ParseFloat(raw, 64)
	if err != nil {
		return 0
	}
	return v
}

func normalizeVisibilityListStrict(values []string) ([]string, error) {
	if len(values) == 0 {
		return nil, nil
	}
	out := make([]string, 0, len(values))
	seen := map[string]struct{}{}
	for _, v := range values {
		normalized, err := normalizeVisibility(v)
		if err != nil {
			return nil, err
		}
		if _, ok := seen[normalized]; ok {
			continue
		}
		seen[normalized] = struct{}{}
		out = append(out, normalized)
	}
	return out, nil
}
