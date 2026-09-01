package main

import (
	"encoding/json"
	"fmt"
	"os"
	"time"
)

type PredictionRecord struct {
	MetricID  string    `json:"metricId"`
	Direction string    `json:"direction"`
	Target    any       `json:"target"`
	Horizon   time.Time `json:"horizon"`
	Verdict   Empirical `json:"verdict"`
	Reason    string    `json:"reason,omitempty"`
}

func loadPredictions() ([]PredictionRecord, error) {
	path := os.Getenv("COMMAND_CENTER_PREDICTIONS_PATH")
	if path == "" {
		return nil, nil
	}
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read prediction ledger: %w", err)
	}
	var rows []PredictionRecord
	if err := json.Unmarshal(b, &rows); err != nil {
		return nil, fmt.Errorf("parse prediction ledger: %w", err)
	}
	return rows, nil
}
func (s *Server) joinPredictions(readings []MetricEntry) {
	rows, err := loadPredictions()
	if err != nil {
		return
	}
	byID := map[string]PredictionRecord{}
	for _, p := range rows {
		byID[p.MetricID] = p
	}
	for i := range readings {
		p, ok := byID[readings[i].ID]
		if !ok {
			continue
		}
		if p.Horizon.After(time.Now()) {
			readings[i].Empirical = EmpiricalPending
			readings[i].Prediction = &Prediction{Target: p.Target, Direction: p.Direction, RemainingHorizonSeconds: int(time.Until(p.Horizon).Seconds())}
		} else if validEmpirical(p.Verdict) && p.Verdict != EmpiricalNone && p.Verdict != EmpiricalPending {
			readings[i].Empirical = p.Verdict
			readings[i].Prediction = nil
		}
	}
}
