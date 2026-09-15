package evidence

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"test-genie/internal/requirements/types"
)

const externalAutomationEvidenceSchemaVersion = 1

type externalAutomationManifest struct {
	SchemaVersion int                             `json:"schema_version"`
	Producer      string                          `json:"producer"`
	Scenario      string                          `json:"scenario"`
	GeneratedAt   string                          `json:"generated_at"`
	Records       []externalAutomationEvidenceRow `json:"records"`
}

type externalAutomationEvidenceRow struct {
	RequirementID   string         `json:"requirement_id"`
	ValidationRef   string         `json:"validation_ref"`
	Status          string         `json:"status"`
	Phase           string         `json:"phase"`
	Evidence        string         `json:"evidence"`
	SourcePath      string         `json:"source_path"`
	RunID           string         `json:"run_id"`
	ExecutedAt      string         `json:"executed_at,omitempty"`
	DurationSeconds float64        `json:"duration_seconds,omitempty"`
	Metadata        map[string]any `json:"metadata,omitempty"`
}

func loadExternalAutomationFile(ctx context.Context, reader Reader, path string) (types.EvidenceMap, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	raw, err := reader.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var manifest externalAutomationManifest
	if err := json.Unmarshal(raw, &manifest); err != nil {
		return nil, fmt.Errorf("decode external automation evidence %s: %w", path, err)
	}
	if manifest.SchemaVersion != externalAutomationEvidenceSchemaVersion ||
		strings.TrimSpace(manifest.Producer) == "" ||
		strings.TrimSpace(manifest.Scenario) == "" ||
		len(manifest.Records) == 0 {
		return nil, fmt.Errorf("invalid external automation evidence envelope %s", path)
	}

	generatedAt := parseExternalTimestamp(manifest.GeneratedAt)
	if generatedAt.IsZero() {
		return nil, fmt.Errorf("external automation evidence %s has no generated_at", path)
	}

	result := make(types.EvidenceMap)
	for _, row := range manifest.Records {
		if strings.TrimSpace(row.RequirementID) == "" ||
			strings.TrimSpace(row.ValidationRef) == "" ||
			strings.TrimSpace(row.Phase) == "" ||
			strings.TrimSpace(row.RunID) == "" ||
			strings.TrimSpace(row.Evidence) == "" {
			continue
		}

		status := types.NormalizeLiveStatus(row.Status)
		if status == types.LiveUnknown {
			continue
		}

		updatedAt := parseExternalTimestamp(row.ExecutedAt)
		if updatedAt.IsZero() {
			updatedAt = generatedAt
		}
		metadata := make(map[string]any, len(row.Metadata)+3)
		for key, value := range row.Metadata {
			metadata[key] = value
		}
		metadata["producer"] = manifest.Producer
		metadata["run_id"] = row.RunID
		metadata["manifest_path"] = path

		sourcePath := strings.TrimSpace(row.SourcePath)
		if sourcePath == "" {
			sourcePath = path
		}
		result.Add(types.EvidenceRecord{
			RequirementID:   strings.TrimSpace(row.RequirementID),
			ValidationRef:   strings.TrimSpace(row.ValidationRef),
			Status:          status,
			Phase:           strings.TrimSpace(row.Phase),
			Evidence:        strings.TrimSpace(row.Evidence),
			UpdatedAt:       updatedAt,
			DurationSeconds: row.DurationSeconds,
			SourcePath:      sourcePath,
			Metadata:        metadata,
		})
	}

	if result.Count() == 0 {
		return nil, fmt.Errorf("external automation evidence %s has no valid records", path)
	}
	return result, nil
}

func parseExternalTimestamp(value string) time.Time {
	value = strings.TrimSpace(value)
	if value == "" {
		return time.Time{}
	}
	parsed, err := time.Parse(time.RFC3339Nano, value)
	if err != nil {
		return time.Time{}
	}
	return parsed
}
