package signals

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"connectrpc.com/connect"
	"github.com/vrooli/api-core/discovery"
	architecturev1 "github.com/vrooli/vrooli/packages/proto/gen/go/architecture/v1"
	runspb "github.com/vrooli/vrooli/packages/proto/gen/go/test-genie/v1/runs"
	runsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/test-genie/v1/runs/runs_v1connect"
)

// phasesCollector reads cached test-genie phase results and keeps the
// newest result per phase across coverage/runs/<id>/phase-results/*.json
// and the legacy top-level coverage/phase-results/*.json.
type phasesCollector struct{ source phaseArtifactSource }

type phaseArtifact struct {
	runID      string
	phase      string
	status     string
	observedAt time.Time
	content    []byte
}

type phaseArtifactSource interface {
	Load(context.Context, string) ([]phaseArtifact, bool, error)
}

type testGeniePhaseSource struct{}

func (testGeniePhaseSource) Load(ctx context.Context, scenario string) ([]phaseArtifact, bool, error) {
	baseURL, err := discovery.ResolveScenarioURLDefault(ctx, "test-genie")
	if err != nil {
		return nil, false, fmt.Errorf("resolve test-genie: %w", err)
	}
	client := runsconnect.NewRunsServiceClient(&http.Client{Timeout: 5 * time.Second}, strings.TrimRight(baseURL, "/"))
	response, err := client.ListRuns(ctx, connect.NewRequest(&runspb.ListRunsRequest{Target: scenario}))
	if err != nil {
		return nil, false, fmt.Errorf("list test-genie runs: %w", err)
	}
	artifacts := make([]phaseArtifact, 0)
	for _, run := range response.Msg.GetRuns() {
		observedAt, _ := time.Parse(time.RFC3339, run.GetCompletedAt())
		if observedAt.IsZero() {
			observedAt, _ = time.Parse(time.RFC3339, run.GetStartedAt())
		}
		for _, phase := range run.GetPhases() {
			item := phaseArtifact{runID: run.GetRunId(), phase: phase.GetName(), status: phase.GetStatus(), observedAt: observedAt}
			artifact, artifactErr := client.GetPhaseArtifact(ctx, connect.NewRequest(&runspb.GetPhaseArtifactRequest{
				Target: scenario, RunId: run.GetRunId(), Phase: phase.GetName(),
			}))
			if artifactErr == nil {
				item.content = []byte(artifact.Msg.GetContent())
			}
			artifacts = append(artifacts, item)
		}
	}
	return artifacts, true, nil
}

func (phasesCollector) Name() string { return "phases" }

func (c phasesCollector) Collect(snap *Snapshot) error {
	source := c.source
	if source == nil {
		source = testGeniePhaseSource{}
	}
	artifacts, collected, err := source.Load(context.Background(), snap.Scenario)
	if err != nil {
		return err
	}
	if !collected {
		return nil
	}
	best := map[string]phaseCandidate{}
	for _, artifact := range artifacts {
		name, candidate, ok := decodePhaseData(artifact.content, artifact.phase, artifact.runID)
		if !ok {
			if strings.TrimSpace(artifact.status) == "" {
				continue
			}
			name = artifact.phase
			candidate = phaseCandidate{result: PhaseResult{Status: artifact.status, UpdatedAt: artifact.observedAt}, hasTime: !artifact.observedAt.IsZero(), runDir: artifact.runID}
		}
		if existing, seen := best[name]; !seen || candidate.newerThan(existing) {
			best[name] = candidate
		}
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

// phaseFile is the on-disk result shape. Findings stays raw so a decode
// failure degrades to "status only" instead of dropping the file.
type phaseFile struct {
	Phase     string          `json:"phase"`
	Status    string          `json:"status"`
	UpdatedAt string          `json:"updated_at"`
	Findings  json.RawMessage `json:"findings"`
}

func decodePhaseData(data []byte, fallbackName, runDir string) (string, phaseCandidate, bool) {
	var pf phaseFile
	if err := json.Unmarshal(data, &pf); err != nil || pf.Status == "" {
		return "", phaseCandidate{}, false
	}

	name := pf.Phase
	if name == "" {
		name = fallbackName
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
