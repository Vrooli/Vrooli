package experiment

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// VariantStats aggregates outcome metrics for a single experiment variant.
type VariantStats struct {
	VariantID       string  `json:"variantId"`
	TotalRuns       int     `json:"totalRuns"`
	ReadyCount      int     `json:"readyCount"`
	NeedsWorkCount  int     `json:"needsWorkCount"`
	FixupRate       float64 `json:"fixupRate"`
	AvgDurationSecs float64 `json:"avgDurationSecs,omitempty"`
}

// ExperimentResults contains analyzed metrics for an experiment.
type ExperimentResults struct {
	ExperimentID  string         `json:"experimentId"`
	SkillID       string         `json:"skillId,omitempty"`
	Status        string         `json:"status,omitempty"`
	Variants      []VariantStats `json:"variants"`
	TotalOutcomes int            `json:"totalOutcomes"`
	AnalyzedAt    string         `json:"analyzedAt"`
}

// outcomeEnvelope is the wire format stored by prompt-manager.
type outcomeEnvelope struct {
	VariantID     string          `json:"variantId"`
	SchemaVersion int             `json:"schemaVersion"`
	Data          json.RawMessage `json:"data"`
}

// Analyze parses raw outcome JSON blobs and computes per-variant statistics.
func Analyze(outcomes []json.RawMessage) (*ExperimentResults, error) {
	type accumulator struct {
		total     int
		ready     int
		needsWork int
		fixups    int
		totalDur  float64
		durCount  int
	}
	byVariant := map[string]*accumulator{}

	for _, raw := range outcomes {
		var env outcomeEnvelope
		if err := json.Unmarshal(raw, &env); err != nil {
			return nil, fmt.Errorf("unmarshal outcome envelope: %w", err)
		}

		acc, ok := byVariant[env.VariantID]
		if !ok {
			acc = &accumulator{}
			byVariant[env.VariantID] = acc
		}
		acc.total++

		switch env.SchemaVersion {
		case 1:
			var data OutcomeDataV1
			if err := json.Unmarshal(env.Data, &data); err != nil {
				return nil, fmt.Errorf("unmarshal outcome data v1: %w", err)
			}
			cls := strings.ToLower(strings.TrimSpace(data.Classification))
			switch cls {
			case "ready", "ready_with_notes":
				acc.ready++
			case "needs_work":
				acc.needsWork++
			}
			if data.HadFixup {
				acc.fixups++
			}
			if data.DurationSecs > 0 {
				acc.totalDur += data.DurationSecs
				acc.durCount++
			}
		default:
			// Unknown schema version; count but skip deep analysis.
		}
	}

	variants := make([]VariantStats, 0, len(byVariant))
	for vid, acc := range byVariant {
		stat := VariantStats{
			VariantID:      vid,
			TotalRuns:      acc.total,
			ReadyCount:     acc.ready,
			NeedsWorkCount: acc.needsWork,
		}
		if acc.total > 0 {
			stat.FixupRate = float64(acc.fixups) / float64(acc.total)
		}
		if acc.durCount > 0 {
			stat.AvgDurationSecs = acc.totalDur / float64(acc.durCount)
		}
		variants = append(variants, stat)
	}

	return &ExperimentResults{
		Variants:      variants,
		TotalOutcomes: len(outcomes),
		AnalyzedAt:    time.Now().UTC().Format(time.RFC3339),
	}, nil
}
