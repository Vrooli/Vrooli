package requirements

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"test-genie/internal/requirements/evidence"
	"test-genie/internal/requirements/types"
)

func runManualLog(args []string) error {
	fs := flag.NewFlagSet("requirements manual-log", flag.ContinueOnError)
	reqID := fs.String("requirement", "", "Requirement ID to log (e.g., REQ-001)")
	status := fs.String("status", "passed", "Validation status (passed|failed|skipped)")
	notes := fs.String("notes", "", "Notes or summary")
	artifact := fs.String("artifact", "", "Artifact path (relative to scenario)")
	validatedBy := fs.String("validated-by", defaultValidator(), "Validator identity")
	validatedAt := fs.String("validated-at", "", "Timestamp (RFC3339). Defaults to now.")
	expiresIn := fs.Int("expires-in", 30, "Expiration window in days")
	expiresAt := fs.String("expires-at", "", "Expiration timestamp (RFC3339). Overrides expires-in.")
	manifestPath := fs.String("manifest", "", "Override manifest path (default: coverage/manual-validations/log.jsonl)")
	dryRun := fs.Bool("dry-run", false, "Print entry instead of writing")
	dirFlag, scenarioFlag := parseCommonFlags(fs)

	if err := fs.Parse(args); err != nil {
		return err
	}
	if *reqID == "" {
		return fmt.Errorf("--requirement is required")
	}

	dir, err := resolveDir(*dirFlag)
	if err != nil {
		return err
	}
	if err := ensureDir(dir); err != nil {
		return err
	}

	entry, err := buildManualEntry(*reqID, *status, *validatedBy, *notes, *artifact, *validatedAt, *expiresIn, *expiresAt)
	if err != nil {
		return err
	}

	manifest := *manifestPath
	if manifest == "" {
		manifest = filepath.Join(dir, "coverage", "manual-validations", "log.jsonl")
	} else if !filepath.IsAbs(manifest) {
		manifest = filepath.Join(dir, manifest)
	}

	if *dryRun {
		data, _ := evidence.SerializeManualValidation(entry)
		fmt.Println(string(data))
		fmt.Printf("Manifest: %s\n", manifest)
		return nil
	}

	if err := os.MkdirAll(filepath.Dir(manifest), 0o755); err != nil {
		return fmt.Errorf("failed to create manifest directory: %w", err)
	}

	data, err := evidence.SerializeManualValidation(entry)
	if err != nil {
		return fmt.Errorf("failed to serialize manual entry: %w", err)
	}
	f, err := os.OpenFile(manifest, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return fmt.Errorf("failed to open manifest: %w", err)
	}
	defer f.Close()
	if _, err := f.Write(append(data, '\n')); err != nil {
		return fmt.Errorf("failed to write manifest entry: %w", err)
	}

	fmt.Printf("Logged manual validation for %s in %s\n", *reqID, scenarioNameFromDir(dir, *scenarioFlag))
	return nil
}

func buildManualEntry(reqID, status, validator, notes, artifact, validatedAt string, expiresIn int, expiresAt string) (types.ManualValidation, error) {
	entry := types.ManualValidation{
		RequirementID: strings.TrimSpace(reqID),
		Status:        string(types.NormalizeLiveStatus(status)),
		ValidatedBy:   validator,
		ArtifactPath:  strings.TrimSpace(artifact),
		Notes:         strings.TrimSpace(notes),
	}

	// Parse timestamps
	if validatedAt != "" {
		t, err := time.Parse(time.RFC3339, validatedAt)
		if err != nil {
			return types.ManualValidation{}, fmt.Errorf("invalid --validated-at: %w", err)
		}
		entry.ValidatedAt = t
	} else {
		entry.ValidatedAt = time.Now()
	}
	if expiresAt != "" {
		t, err := time.Parse(time.RFC3339, expiresAt)
		if err != nil {
			return types.ManualValidation{}, fmt.Errorf("invalid --expires-at: %w", err)
		}
		entry.ExpiresAt = t
	} else if expiresIn > 0 {
		entry.ExpiresAt = entry.ValidatedAt.Add(time.Duration(expiresIn) * 24 * time.Hour)
	}

	return entry, nil
}

func defaultValidator() string {
	if v := os.Getenv("VROOLI_AGENT_ID"); v != "" {
		return v
	}
	if v := os.Getenv("USER"); v != "" {
		return v
	}
	return "manual-validator"
}

// toJSON helper used in tests.
func toJSON(v any) string {
	data, _ := json.MarshalIndent(v, "", "  ")
	return string(data)
}
