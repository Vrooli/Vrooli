package evidence

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"strings"
	"time"

	"test-genie/internal/requirements/types"
)

// manualValidationEntry represents a single entry in log.jsonl.
type manualValidationEntry struct {
	RequirementID string `json:"requirement_id"`
	Status        string `json:"status"`
	ValidatedAt   string `json:"validated_at,omitempty"`
	ExpiresAt     string `json:"expires_at,omitempty"`
	ValidatedBy   string `json:"validated_by,omitempty"`
	ArtifactPath  string `json:"artifact_path,omitempty"`
	Notes         string `json:"notes,omitempty"`
	Action        string `json:"action,omitempty"` // "validate" or "invalidate"
}

// loadManualFromFile loads manual validations from a JSONL file.
func loadManualFromFile(ctx context.Context, reader Reader, filePath string) (*types.ManualManifest, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	data, err := reader.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	manifest := types.NewManualManifest(filePath)

	// Parse JSONL (one JSON object per line)
	scanner := bufio.NewScanner(bytes.NewReader(data))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue // Skip empty lines and comments
		}

		var entry manualValidationEntry
		if err := json.Unmarshal([]byte(line), &entry); err != nil {
			continue // Skip invalid entries
		}

		validation := convertManualEntry(entry)

		// Handle invalidation action
		if strings.ToLower(entry.Action) == "invalidate" {
			// Remove from manifest if it exists
			delete(manifest.ByRequirement, validation.RequirementID)
			continue
		}

		manifest.Add(validation)
	}

	return manifest, nil
}

// convertManualEntry converts a raw entry to ManualValidation.
func convertManualEntry(entry manualValidationEntry) types.ManualValidation {
	validation := types.ManualValidation{
		RequirementID: strings.TrimSpace(entry.RequirementID),
		Status:        entry.Status,
		ValidatedBy:   entry.ValidatedBy,
		ArtifactPath:  entry.ArtifactPath,
		Notes:         entry.Notes,
	}

	// Parse timestamps
	if entry.ValidatedAt != "" {
		if t, err := parseTimestamp(entry.ValidatedAt); err == nil {
			validation.ValidatedAt = t
		}
	}
	if entry.ExpiresAt != "" {
		if t, err := parseTimestamp(entry.ExpiresAt); err == nil {
			validation.ExpiresAt = t
		}
	}

	// Default validated_at to now if not specified
	if validation.ValidatedAt.IsZero() {
		validation.ValidatedAt = time.Now()
	}

	return validation
}

// parseTimestamp parses various timestamp formats.
func parseTimestamp(s string) (time.Time, error) {
	formats := []string{
		time.RFC3339,
		time.RFC3339Nano,
		"2006-01-02T15:04:05Z",
		"2006-01-02T15:04:05",
		"2006-01-02 15:04:05",
		"2006-01-02",
	}

	s = strings.TrimSpace(s)
	for _, format := range formats {
		if t, err := time.Parse(format, s); err == nil {
			return t, nil
		}
	}

	return time.Time{}, &time.ParseError{Value: s}
}

// CreateManualValidation creates a new ManualValidation entry.
func CreateManualValidation(
	requirementID string,
	status types.LiveStatus,
	validatedBy string,
	notes string,
	expiresIn time.Duration,
) types.ManualValidation {
	validation := types.ManualValidation{
		RequirementID: requirementID,
		Status:        string(status),
		ValidatedAt:   time.Now(),
		ValidatedBy:   validatedBy,
		Notes:         notes,
	}

	if expiresIn > 0 {
		validation.ExpiresAt = time.Now().Add(expiresIn)
	}

	return validation
}

// SerializeManualValidation serializes a ManualValidation to JSON.
func SerializeManualValidation(v types.ManualValidation) ([]byte, error) {
	entry := manualValidationEntry{
		RequirementID: v.RequirementID,
		Status:        v.Status,
		ValidatedBy:   v.ValidatedBy,
		ArtifactPath:  v.ArtifactPath,
		Notes:         v.Notes,
		Action:        "validate",
	}

	if !v.ValidatedAt.IsZero() {
		entry.ValidatedAt = v.ValidatedAt.Format(time.RFC3339)
	}
	if !v.ExpiresAt.IsZero() {
		entry.ExpiresAt = v.ExpiresAt.Format(time.RFC3339)
	}

	return json.Marshal(entry)
}

// FilterValidManualValidations returns only valid (non-expired) validations.
func FilterValidManualValidations(validations []types.ManualValidation) []types.ManualValidation {
	var valid []types.ManualValidation
	for _, v := range validations {
		if v.IsValid() {
			valid = append(valid, v)
		}
	}
	return valid
}

// GetLatestManualValidation returns the most recent validation for a requirement.
func GetLatestManualValidation(validations []types.ManualValidation, requirementID string) *types.ManualValidation {
	var latest *types.ManualValidation

	for i := range validations {
		v := &validations[i]
		if v.RequirementID != requirementID {
			continue
		}
		if latest == nil || v.ValidatedAt.After(latest.ValidatedAt) {
			latest = v
		}
	}

	return latest
}

// ManualValidationToEvidence converts a manual validation to an evidence record.
func ManualValidationToEvidence(v types.ManualValidation) types.EvidenceRecord {
	return types.EvidenceRecord{
		RequirementID: v.RequirementID,
		Status:        v.ToLiveStatus(),
		Phase:         "manual",
		Evidence:      v.Notes,
		UpdatedAt:     v.ValidatedAt,
		SourcePath:    v.ArtifactPath,
		Metadata: map[string]any{
			"validated_by": v.ValidatedBy,
			"expires_at":   v.ExpiresAt,
		},
	}
}
