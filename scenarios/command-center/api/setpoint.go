package main

import (
	"encoding/json"
	"fmt"
	"os"
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
	if s.SchemaVersion == "" {
		return nil, fmt.Errorf("setpoint is missing schemaVersion")
	}
	bars := map[string]SetpointBar{}
	for _, b := range s.Bars {
		if b.MetricID == "" {
			return nil, fmt.Errorf("setpoint bar missing metricId")
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

func equalJSON(a, b any) bool {
	aa, _ := json.Marshal(a)
	bb, _ := json.Marshal(b)
	return string(aa) == string(bb)
}
