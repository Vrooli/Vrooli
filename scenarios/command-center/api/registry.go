package main

import (
	"encoding/json"
	"fmt"
	"os"
)

// DataSourceStatus enumerates the tri-state availability of a metric.
type DataSourceStatus string

const (
	StatusLive    DataSourceStatus = "live"
	StatusPartial DataSourceStatus = "partial"
	StatusGap     DataSourceStatus = "gap"
)

// UpstreamSource identifies which upstream client supplies the metric's data.
type UpstreamSource string

const (
	SourceSwarm  UpstreamSource = "swarm"
	SourceVrooli UpstreamSource = "vrooli"
	SourceLPBS   UpstreamSource = "lpbs"
	SourceNone   UpstreamSource = "none"
)

// MetricEntry describes one metric's identity and data-availability stance.
type MetricEntry struct {
	ID             string           `json:"id"`
	Label          string           `json:"label"`
	DataSource     DataSourceStatus `json:"dataSource"`
	UpstreamSource UpstreamSource   `json:"upstreamSource"`
	Description    string           `json:"description,omitempty"`
	WhatIsNeeded   *string          `json:"whatIsNeeded,omitempty"`
}

// Registry is the top-level shape of config/gap-registry.json.
type Registry struct {
	Version    string                   `json:"version"`
	Dashboards map[string][]MetricEntry `json:"dashboards"`
}

// LoadRegistry reads the gap registry JSON file from disk and returns a
// parsed Registry. It uses DisallowUnknownFields so typos in the JSON are
// caught by the decoder rather than silently ignored.
func LoadRegistry(path string) (*Registry, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open registry: %w", err)
	}
	defer f.Close()

	dec := json.NewDecoder(f)
	dec.DisallowUnknownFields()

	var reg Registry
	if err := dec.Decode(&reg); err != nil {
		return nil, fmt.Errorf("decode registry: %w", err)
	}
	if reg.Version == "" {
		return nil, fmt.Errorf("registry is missing 'version'")
	}
	if len(reg.Dashboards) == 0 {
		return nil, fmt.Errorf("registry is missing 'dashboards'")
	}
	return &reg, nil
}

// Dashboard returns the entries for the given dashboard id, or nil if absent.
func (r *Registry) Dashboard(id string) []MetricEntry {
	if r == nil {
		return nil
	}
	return r.Dashboards[id]
}

// GapsByDashboard returns every entry whose dataSource is gap or partial,
// grouped by dashboard id. Empty dashboards are omitted.
func (r *Registry) GapsByDashboard() map[string][]MetricEntry {
	out := make(map[string][]MetricEntry, len(r.Dashboards))
	for id, entries := range r.Dashboards {
		var filtered []MetricEntry
		for _, e := range entries {
			if e.DataSource == StatusGap || e.DataSource == StatusPartial {
				filtered = append(filtered, e)
			}
		}
		if len(filtered) > 0 {
			out[id] = filtered
		}
	}
	return out
}
