package main

import (
	"context"
	"time"

	"command-center/upstream"
)

type observationEndpoint struct {
	source UpstreamSource
	path   string
}

type observationResult struct {
	envelope Envelope
	err      error
	commit   bool
}

// fetchObservations reads each source endpoint at most once for a board
// projection. Metrics that share a producer response therefore see the same
// observation time and do not multiply source load by the number of tiles.
func (s *Server) fetchObservations(ctx context.Context, entries []MetricEntry) map[string]observationResult {
	results := make(map[string]observationResult, len(entries))
	groups := make(map[observationEndpoint][]MetricEntry)
	for _, entry := range entries {
		coverage := effectiveCoverage(entry)
		if coverage != CoverageNow && coverage != CoverageInReach {
			continue
		}
		source := entry.UpstreamSource
		if source == "" {
			source = sourceFromBinding(entry.Source.Binding)
		}
		path := entry.Source.Read
		if path == "" {
			path = defaultPathFor(source)
		}
		key := observationEndpoint{source: source, path: path}
		groups[key] = append(groups[key], entry)
	}

	for endpoint, metrics := range groups {
		cached := make(map[string]Envelope, len(metrics))
		fresh := make(map[string]bool, len(metrics))
		needsFetch := false
		for _, metric := range metrics {
			key := observationCacheKey(endpoint.source, endpoint.path, metric.ID)
			value, isFresh, ok := s.cache.Get(key)
			if ok {
				cached[metric.ID] = value
				fresh[metric.ID] = isFresh
			}
			if !ok || !isFresh {
				needsFetch = true
			}
		}
		if !needsFetch {
			for id, value := range cached {
				results[id] = observationResult{envelope: value}
			}
			continue
		}

		client := s.clientFor(endpoint.source)
		var raw []byte
		var err error
		if client == nil {
			err = upstream.ErrNotAvailable
		} else {
			raw, err = client.Fetch(ctx, endpoint.path)
		}
		var observed Envelope
		if err == nil {
			observed = Envelope{Data: raw, CachedAt: time.Now().UTC(), Source: string(endpoint.source), ObservationAt: observationTime(raw)}
			if observed.ObservationAt == nil {
				err = errMissingObservationTime
			} else {
				observed = s.cache.CommitEndpoint(observationEndpointCacheKey(endpoint.source, endpoint.path), observed)
			}
		}

		for _, metric := range metrics {
			key := observationCacheKey(endpoint.source, endpoint.path, metric.ID)
			if err == nil {
				results[metric.ID] = observationResult{envelope: observed, commit: true}
				continue
			}
			if fresh[metric.ID] {
				results[metric.ID] = observationResult{envelope: cached[metric.ID]}
				continue
			}
			s.cache.MarkStale(key, time.Now().UTC())
			if previous, _, ok := s.cache.Get(key); ok {
				results[metric.ID] = observationResult{envelope: previous, err: err}
			} else if observed.Data != nil {
				results[metric.ID] = observationResult{envelope: observed, err: err}
			} else {
				results[metric.ID] = observationResult{err: err}
			}
		}
	}
	return results
}

func observationEndpointCacheKey(source UpstreamSource, path string) string {
	return string(source) + ":" + path
}
