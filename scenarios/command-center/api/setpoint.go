package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

type SetpointBar struct {
	MetricID        string `json:"metricId"`
	Direction       string `json:"direction"`
	Bar             any    `json:"bar"`
	Unit            string `json:"unit"`
	AuthoredAgainst any    `json:"authoredAgainst"`
	DecisionRef     string `json:"decisionRef"`
}
type NotGradeable struct {
	MetricID string `json:"metricId"`
	Reason   string `json:"reason"`
}
type Setpoint struct {
	SchemaRef     string         `json:"$schema,omitempty"`
	SchemaVersion string         `json:"schemaVersion"`
	Bars          []SetpointBar  `json:"bars"`
	NotGradeable  []NotGradeable `json:"notGradeable"`
}

// LoadSetpoint parses and validates the checked-in target contract. It is
// intentionally read-only: no API route or CLI command writes this file.
func LoadSetpoint(path string, registry *Registry) (*Setpoint, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open setpoint: %w", err)
	}
	defer f.Close()
	dec := json.NewDecoder(f)
	dec.DisallowUnknownFields()
	var s Setpoint
	if err := dec.Decode(&s); err != nil {
		return nil, fmt.Errorf("parse setpoint: %w", err)
	}
	var trailing json.RawMessage
	if err := dec.Decode(&trailing); err != io.EOF {
		if err == nil {
			return nil, fmt.Errorf("parse setpoint: multiple JSON values")
		}
		return nil, fmt.Errorf("parse setpoint: trailing data: %w", err)
	}
	if s.SchemaVersion != "1.0.0" {
		return nil, fmt.Errorf("unsupported setpoint schemaVersion %q", s.SchemaVersion)
	}
	bars := map[string]SetpointBar{}
	for _, b := range s.Bars {
		if b.MetricID == "" {
			return nil, fmt.Errorf("setpoint bar missing metricId")
		}
		if b.Direction != "up" && b.Direction != "down" {
			return nil, fmt.Errorf("setpoint bar %s has invalid direction %q", b.MetricID, b.Direction)
		}
		if strings.TrimSpace(b.Unit) == "" {
			return nil, fmt.Errorf("setpoint bar %s is missing unit", b.MetricID)
		}
		if _, ok := numericReading(b.Bar); !ok {
			return nil, fmt.Errorf("setpoint bar %s must be numeric", b.MetricID)
		}
		if _, ok := numericReading(b.AuthoredAgainst); !ok {
			return nil, fmt.Errorf("setpoint bar %s is missing a numeric authoredAgainst value", b.MetricID)
		}
		if _, ok := bars[b.MetricID]; ok {
			return nil, fmt.Errorf("duplicate setpoint bar %s", b.MetricID)
		}
		if equalJSON(b.Bar, b.AuthoredAgainst) {
			return nil, fmt.Errorf("setpoint bar %s equals authoredAgainst", b.MetricID)
		}
		if !equalJSON(b.Bar, b.AuthoredAgainst) && b.DecisionRef == "" {
			return nil, fmt.Errorf("setpoint bar %s changed without decisionRef", b.MetricID)
		}
		bars[b.MetricID] = b
	}
	gradeable := map[string]bool{}
	for _, n := range s.NotGradeable {
		if n.MetricID == "" || n.Reason == "" {
			return nil, fmt.Errorf("not-gradeable entry requires metricId and reason")
		}
		gradeable[n.MetricID] = true
	}
	if registry != nil {
		metrics := make(map[string]MetricEntry, len(registry.Metrics))
		for _, metric := range registry.Metrics {
			metrics[metric.ID] = metric
		}
		for id, bar := range bars {
			metric, ok := metrics[id]
			if !ok {
				return nil, fmt.Errorf("setpoint bar %s does not match a registered metric", id)
			}
			if strings.TrimSpace(bar.Unit) == "" || bar.Unit != metric.Unit {
				return nil, fmt.Errorf("setpoint bar %s unit %q does not match metric unit %q", id, bar.Unit, metric.Unit)
			}
		}
		for _, m := range registry.Metrics {
			if effectiveCoverage(m) != CoverageNow {
				continue
			}
			if _, ok := bars[m.ID]; !ok && !gradeable[m.ID] && !gradeable["all"] {
				return nil, fmt.Errorf("NOW metric %s has no bar or not-gradeable reason", m.ID)
			}
		}
	}
	return &s, nil
}

func (s *Server) readSetpoint() (*Setpoint, error) {
	if err := ensureConfigRoot(); err != nil {
		return nil, err
	}
	return LoadSetpoint(filepath.Join(catalogDir(), "setpoint.json"), s.registry)
}

func applySetpoint(readings []MetricEntry, setpoint *Setpoint) {
	bars := make(map[string]SetpointBar, len(setpoint.Bars))
	for _, bar := range setpoint.Bars {
		bars[bar.MetricID] = bar
	}
	for i := range readings {
		readings[i].Target = nil
		if bar, ok := bars[readings[i].ID]; ok {
			target := &Target{Direction: bar.Direction, Bar: bar.Bar, BarRef: "setpoint:" + bar.MetricID}
			if value, valueOK := numericReading(readings[i].Value); valueOK {
				if threshold, thresholdOK := numericReading(bar.Bar); thresholdOK {
					distance := threshold - value
					if bar.Direction == "down" {
						distance = value - threshold
					}
					if distance < 0 {
						distance = 0
					}
					target.DistanceToTarget = &distance
				}
			}
			readings[i].Target = target
		}
	}
}

func writeSetpointReadError(w http.ResponseWriter, err error) {
	if isConfigRootUnavailable(err) {
		writeConfigReadError(w, err)
		return
	}
	writeError(w, http.StatusInternalServerError, "setpoint_invalid", "configured setpoint could not be read or validated", nil)
}

func equalJSON(a, b any) bool {
	aa, _ := json.Marshal(a)
	bb, _ := json.Marshal(b)
	return string(aa) == string(bb)
}
