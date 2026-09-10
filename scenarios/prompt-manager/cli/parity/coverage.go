package parity

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	repocontract "github.com/vrooli/repo-contract-go"
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
	root, err := repocontract.FindRepoRootFromEnvOrCWD()
	if err != nil {
		return nil, fmt.Errorf("resolve parity repository: %w", err)
	}
	path := filepath.Join(root, "scenarios", "prompt-manager", "cli", "parity", "coverage.json")
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

// APIMainPath uses the repository contract, not compiler debug paths (which
// become module-relative under Test Genie's -trimpath builds).
func APIMainPath() (string, error) {
	root, err := repocontract.FindRepoRootFromEnvOrCWD()
	if err != nil {
		return "", fmt.Errorf("resolve parity repository: %w", err)
	}
	return filepath.Join(root, "scenarios", "prompt-manager", "api", "main.go"), nil
}
