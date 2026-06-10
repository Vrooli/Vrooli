package signals

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	architecturev1 "github.com/vrooli/vrooli/packages/proto/gen/go/architecture/v1"
)

// phasesCollector reads cached test-genie phase results and keeps the
// newest result per phase across coverage/runs/<id>/phase-results/*.json
// and the legacy top-level coverage/phase-results/*.json.
type phasesCollector struct{}

func (phasesCollector) Name() string { return "phases" }

func (phasesCollector) Collect(snap *Snapshot) error {
	best := map[string]phaseCandidate{}
	collected := false

	runsDir := filepath.Join(snap.Root, "coverage", "runs")
	runEntries, err := os.ReadDir(runsDir)
	switch {
	case err == nil:
		collected = true
		for _, entry := range runEntries {
			if !entry.IsDir() {
				continue
			}
			dir := filepath.Join(runsDir, entry.Name(), "phase-results")
			mergePhaseDir(dir, entry.Name(), best)
		}
	case !os.IsNotExist(err):
		return fmt.Errorf("read runs dir: %w", err)
	}

	legacyDir := filepath.Join(snap.Root, "coverage", "phase-results")
	if _, statErr := os.Stat(legacyDir); statErr == nil {
		collected = true
		// Legacy top-level files carry an empty run-dir key so they lose
		// ties to run-dir results.
		mergePhaseDir(legacyDir, "", best)
	} else if !os.IsNotExist(statErr) {
		return fmt.Errorf("stat legacy phase-results dir: %w", statErr)
	}

	if !collected {
		// Never-tested scenario: missing coverage trees are normal.
		return nil
	}

	phases := make(map[string]PhaseResult, len(best))
	for name, cand := range best {
		phases[name] = cand.result
	}
	snap.Phases = PhaseSignals{Collected: true, Phases: phases}
	return nil
}

// phaseCandidate pairs a decoded result with its freshness ordering keys.
type phaseCandidate struct {
	result  PhaseResult
	hasTime bool
	// runDir is the YYYYMMDD-HHMMSS-xxxx run directory name ("" for legacy
	// top-level files); names sort chronologically.
	runDir string
}

// newerThan: updated_at decides when both candidates carry one; otherwise
// run-dir name ordering, with a parseable timestamp breaking exact ties.
func (a phaseCandidate) newerThan(b phaseCandidate) bool {
	if a.hasTime && b.hasTime && !a.result.UpdatedAt.Equal(b.result.UpdatedAt) {
		return a.result.UpdatedAt.After(b.result.UpdatedAt)
	}
	if a.runDir != b.runDir {
		return a.runDir > b.runDir
	}
	if a.hasTime != b.hasTime {
		return a.hasTime
	}
	return false
}

// mergePhaseDir folds every decodable result file in dir into best.
// Unreadable directories and malformed individual files are skipped.
func mergePhaseDir(dir, runDir string, best map[string]phaseCandidate) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return
	}
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}
		name, cand, ok := decodePhaseFile(filepath.Join(dir, entry.Name()), runDir)
		if !ok {
			continue
		}
		if existing, seen := best[name]; !seen || cand.newerThan(existing) {
			best[name] = cand
		}
	}
}

// phaseFile is the on-disk result shape. Findings stays raw so a decode
// failure degrades to "status only" instead of dropping the file.
type phaseFile struct {
	Phase     string          `json:"phase"`
	Status    string          `json:"status"`
	UpdatedAt string          `json:"updated_at"`
	Findings  json.RawMessage `json:"findings"`
}

func decodePhaseFile(path, runDir string) (string, phaseCandidate, bool) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", phaseCandidate{}, false
	}
	var pf phaseFile
	if err := json.Unmarshal(data, &pf); err != nil || pf.Status == "" {
		return "", phaseCandidate{}, false
	}

	name := pf.Phase
	if name == "" {
		name = strings.TrimSuffix(filepath.Base(path), ".json")
	}

	cand := phaseCandidate{runDir: runDir}
	cand.result.Status = pf.Status
	if t, err := time.Parse(time.RFC3339, pf.UpdatedAt); err == nil {
		cand.result.UpdatedAt = t
		cand.hasTime = true
	}

	// `"findings": null` counts as absent; a key that fails to decode
	// keeps the status but contributes no findings.
	if len(pf.Findings) > 0 && !bytes.Equal(bytes.TrimSpace(pf.Findings), []byte("null")) {
		var findings []*architecturev1.ArchitectureFinding
		if err := json.Unmarshal(pf.Findings, &findings); err == nil {
			cand.result.Findings = findings
			cand.result.HasFindings = true
		}
	}
	return name, cand, true
}
