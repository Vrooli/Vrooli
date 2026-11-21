// Package evidence composes the read-only evidence artifacts behind the
// traceability matrix and the evidence_traceability findings.
//
// Single-writer discipline (plan decision D1): test-genie is the ONLY
// writer of run evidence (coverage/requirements-sync/, coverage/runs/).
// This package READS those artifacts and owns exactly one ledger of its
// own — coverage/manual-validations/log.jsonl — which AppendAttestation
// writes. readonly_test.go asserts no other write escapes this package.
package evidence

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/vrooli/vrooli/packages/artifactpaths"
)

// SyncSnapshot is the typed shape of coverage/requirements-sync/latest.json
// (test-genie's sync artifact, version 1.0.0).
type SyncSnapshot struct {
	Version            string             `json:"version"`
	GeneratedAt        time.Time          `json:"generated_at"`
	Summary            SyncSummary        `json:"summary"`
	OperationalTargets []SyncTarget       `json:"operational_targets"`
	Modules            []SyncModuleRollup `json:"modules"`
}

type SyncSummary struct {
	TotalRequirements int     `json:"total_requirements"`
	TotalValidations  int     `json:"total_validations"`
	CompletionRate    float64 `json:"completion_rate"`
	PassRate          float64 `json:"pass_rate"`
	CriticalGap       int     `json:"critical_gap"`
}

type SyncTarget struct {
	ID             string   `json:"id"`
	Title          string   `json:"title"`
	Priority       string   `json:"priority"`
	Status         string   `json:"status"`
	RequirementIDs []string `json:"requirement_ids"`
	CompletionRate float64  `json:"completion_rate"`
}

type SyncModuleRollup struct {
	Name           string  `json:"name"`
	FilePath       string  `json:"file_path"`
	Total          int     `json:"total"`
	Complete       int     `json:"complete"`
	InProgress     int     `json:"in_progress"`
	Pending        int     `json:"pending"`
	CompletionRate float64 `json:"completion_rate"`
}

// Attestation is one manual-validation ledger entry.
type Attestation struct {
	Scenario      string    `json:"scenario"`
	RequirementID string    `json:"requirement_id"`
	AttestedBy    string    `json:"attested_by"`
	AttestedAt    time.Time `json:"attested_at"`
	ExpiresAt     time.Time `json:"expires_at"`
	Notes         string    `json:"notes,omitempty"`
}

// Expired reports whether the attestation has aged out at the given time.
func (a Attestation) Expired(now time.Time) bool {
	return !a.ExpiresAt.IsZero() && now.After(a.ExpiresAt)
}

// DefaultManualValidityWindow is the expiry policy business-health owns
// (D1): an attended check older than this no longer counts as evidence.
const DefaultManualValidityWindow = 90 * 24 * time.Hour

const manualLedgerRelPath = "coverage/manual-validations/log.jsonl"

// Store reads a scenario's evidence artifacts.
type Store struct {
	scenarioDir     string
	runEvidenceRoot string
	now             func() time.Time
}

// NewStore builds a Store over one scenario tree. now is substitutable for
// staleness/expiry tests; nil means time.Now.
func NewStore(scenarioDir string, now func() time.Time) Store {
	return newStore(scenarioDir, scenarioDir, now)
}

// NewTargetStore builds the production store for a source scenario. Source-owned
// manual attestations remain next to the business contract, while Test Genie run
// evidence is read through its named artifact-authority contract.
func NewTargetStore(scenarioDir string, now func() time.Time) (Store, error) {
	runEvidenceRoot, err := artifactpaths.ScenarioRootForDir(scenarioDir)
	if err != nil {
		return Store{}, fmt.Errorf("resolve test-genie evidence root: %w", err)
	}
	return newStore(scenarioDir, runEvidenceRoot, now), nil
}

func newStore(scenarioDir, runEvidenceRoot string, now func() time.Time) Store {
	if now == nil {
		now = time.Now
	}
	return Store{scenarioDir: scenarioDir, runEvidenceRoot: runEvidenceRoot, now: now}
}

// LedgerPath is the scenario-relative manual-validations ledger path.
func (Store) LedgerPath() string { return manualLedgerRelPath }

// ReadSnapshot loads the sync snapshot. A missing file yields (zero, false,
// nil) — absence is a finding, not an error.
func (s Store) ReadSnapshot() (SyncSnapshot, bool, error) {
	path := artifactpaths.RequirementsSnapshotPath(s.runEvidenceRoot)
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return SyncSnapshot{}, false, nil
		}
		return SyncSnapshot{}, false, err
	}
	var snap SyncSnapshot
	if err := json.Unmarshal(data, &snap); err != nil {
		return SyncSnapshot{}, false, fmt.Errorf("parse requirements-sync snapshot: %w", err)
	}
	return snap, true, nil
}

// LatestRunTime derives the newest suite-run timestamp from coverage/runs
// directory names (`20260702-061719-...`). Zero time when no runs exist.
func (s Store) LatestRunTime() time.Time {
	entries, err := os.ReadDir(artifactpaths.RunsRootPath(s.runEvidenceRoot))
	if err != nil {
		return time.Time{}
	}
	var names []string
	for _, e := range entries {
		if e.IsDir() {
			names = append(names, e.Name())
		}
	}
	sort.Strings(names)
	for i := len(names) - 1; i >= 0; i-- {
		if ts, ok := parseRunStamp(names[i]); ok {
			return ts
		}
	}
	return time.Time{}
}

func parseRunStamp(name string) (time.Time, bool) {
	parts := strings.SplitN(name, "-", 3)
	if len(parts) < 2 {
		return time.Time{}, false
	}
	ts, err := time.ParseInLocation("20060102-150405", parts[0]+"-"+parts[1], time.UTC)
	if err != nil {
		return time.Time{}, false
	}
	return ts, true
}

// ReadAttestations loads the manual-validations ledger, newest last. A
// missing ledger is an empty list.
func (s Store) ReadAttestations() ([]Attestation, error) {
	path := filepath.Join(s.scenarioDir, filepath.FromSlash(manualLedgerRelPath))
	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	defer f.Close()
	var out []Attestation
	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		var a Attestation
		if err := json.Unmarshal([]byte(line), &a); err != nil {
			return out, fmt.Errorf("parse %s: %w", manualLedgerRelPath, err)
		}
		out = append(out, a)
	}
	return out, scanner.Err()
}

// LatestAttestations reduces the ledger to the newest attestation per
// requirement ID.
func (s Store) LatestAttestations() (map[string]Attestation, error) {
	all, err := s.ReadAttestations()
	if err != nil {
		return nil, err
	}
	out := make(map[string]Attestation, len(all))
	for _, a := range all {
		prev, ok := out[a.RequirementID]
		// Ties go to the later ledger line (append order is chronological).
		if !ok || !a.AttestedAt.Before(prev.AttestedAt) {
			out[a.RequirementID] = a
		}
	}
	return out, nil
}

// AppendAttestation records a manual validation — the ONE evidence write
// business-health owns. The expiry stamp applies the package's validity
// window; attested_by is required (attestation identity).
func (s Store) AppendAttestation(scenario, requirementID, attestedBy, notes string) (Attestation, error) {
	if strings.TrimSpace(requirementID) == "" {
		return Attestation{}, fmt.Errorf("requirement id is required")
	}
	if strings.TrimSpace(attestedBy) == "" {
		return Attestation{}, fmt.Errorf("attested_by is required (attestation identity)")
	}
	now := s.now().UTC()
	a := Attestation{
		Scenario:      scenario,
		RequirementID: strings.TrimSpace(requirementID),
		AttestedBy:    strings.TrimSpace(attestedBy),
		AttestedAt:    now,
		ExpiresAt:     now.Add(DefaultManualValidityWindow),
		Notes:         strings.TrimSpace(notes),
	}
	path := filepath.Join(s.scenarioDir, filepath.FromSlash(manualLedgerRelPath))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return Attestation{}, err
	}
	line, err := json.Marshal(a)
	if err != nil {
		return Attestation{}, err
	}
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return Attestation{}, err
	}
	defer f.Close()
	if _, err := f.Write(append(line, '\n')); err != nil {
		return Attestation{}, err
	}
	return a, nil
}

// Staleness is the snapshot-freshness verdict.
type Staleness struct {
	// SnapshotPresent is false when no sync snapshot exists.
	SnapshotPresent bool
	// Stale is true when the snapshot predates the newest suite run (or is
	// missing while runs exist).
	Stale bool
	// Detail is a human explanation ("" when fresh).
	Detail string
}

// SnapshotStaleness compares the snapshot stamp against the newest run.
func (s Store) SnapshotStaleness(snap SyncSnapshot, present bool) Staleness {
	latest := s.LatestRunTime()
	switch {
	case !present && latest.IsZero():
		return Staleness{SnapshotPresent: false, Stale: false, Detail: "no evidence artifacts yet (no suite runs, no snapshot)"}
	case !present:
		return Staleness{SnapshotPresent: false, Stale: true, Detail: "suite runs exist but no requirements-sync snapshot was written"}
	case !latest.IsZero() && snap.GeneratedAt.Before(latest.Add(-time.Minute)):
		return Staleness{
			SnapshotPresent: true,
			Stale:           true,
			Detail:          fmt.Sprintf("snapshot generated %s predates the latest suite run %s", snap.GeneratedAt.UTC().Format(time.RFC3339), latest.Format(time.RFC3339)),
		}
	default:
		return Staleness{SnapshotPresent: true}
	}
}

// Now exposes the store clock (matrix/checks share one notion of now).
func (s Store) Now() time.Time { return s.now() }
