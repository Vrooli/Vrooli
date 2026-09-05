package parity

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
)

// CoverageEntry describes how a single API route is exposed (or
// intentionally not exposed) on the CLI.
type CoverageEntry struct {
	// CLI is the canonical CLI invocation that wraps this route. Empty if Status
	// is "intentionally-absent".
	CLI string `json:"cli,omitempty"`

	// Status is one of: "covered", "intentionally-absent", "audit-pending".
	// "audit-pending" exists so the parity guard can be turned on without
	// forcing every route to be classified in a single PR; it MUST be
	// drained before the guard is considered fully enforced.
	Status string `json:"status"`

	// Reason is required when Status is "intentionally-absent" or
	// "audit-pending"; it explains the deferral or omission.
	Reason string `json:"reason,omitempty"`
}

// Valid status values for a CoverageEntry.
const (
	StatusCovered             = "covered"
	StatusIntentionallyAbsent = "intentionally-absent"
	StatusAuditPending        = "audit-pending"
)

// LoadCoverage reads coverage.json from the package directory.
func LoadCoverage() (map[string]CoverageEntry, error) {
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		return nil, fmt.Errorf("cannot resolve parity package path")
	}
	path := filepath.Join(filepath.Dir(thisFile), "coverage.json")
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read coverage.json: %w", err)
	}
	var out map[string]CoverageEntry
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, fmt.Errorf("parse coverage.json: %w", err)
	}
	for key, entry := range out {
		switch entry.Status {
		case StatusCovered, StatusIntentionallyAbsent, StatusAuditPending:
		default:
			return nil, fmt.Errorf("coverage[%q]: invalid status %q", key, entry.Status)
		}
		if entry.Status == StatusIntentionallyAbsent && entry.Reason == "" {
			return nil, fmt.Errorf("coverage[%q]: intentionally-absent requires a reason", key)
		}
		if entry.Status == StatusAuditPending && entry.Reason == "" {
			return nil, fmt.Errorf("coverage[%q]: audit-pending requires a reason", key)
		}
		if entry.Status == StatusCovered && entry.CLI == "" {
			return nil, fmt.Errorf("coverage[%q]: covered requires a non-empty cli field", key)
		}
	}
	return out, nil
}

// APIMainPath returns the absolute path to the API's main.go from this
// package's location, allowing tests to find the source without depending on
// the caller's working directory.
func APIMainPath() (string, error) {
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		return "", fmt.Errorf("cannot resolve parity package path")
	}
	// scenarios/prompt-manager/cli/parity → scenarios/prompt-manager/api
	return filepath.Join(filepath.Dir(thisFile), "..", "..", "api", "main.go"), nil
}
