package guidance

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	repocontract "github.com/vrooli/repo-contract-go"
)

type Runner interface {
	NextGate(ctx context.Context, scenario string) (NextGateResult, error)
}

type Service struct {
	runner Runner
}

func NewService(runner Runner) *Service {
	if runner == nil {
		runner = SubprocessRunner{}
	}
	return &Service{runner: runner}
}

func (s *Service) NextGate(ctx context.Context, scenario string) (NextGateResult, error) {
	scenario = strings.TrimSpace(scenario)
	if scenario == "" {
		return NextGateResult{}, fmt.Errorf("scenario is required")
	}
	return s.runner.NextGate(ctx, scenario)
}

type NextGateResult struct {
	Scenario         string
	Finalized        bool
	Complete         bool
	FinalizeRequired bool
	Completed        int32
	Required         int32
	Gate             *Gate
	Message          string
}

type Gate struct {
	ID          string
	Title       string
	Description string
	Required    bool
	Complete    bool
	Docs        []string
	Checks      []Check
	Remediation []string
}

type Check struct {
	Kind     string
	Label    string
	Passed   bool
	Skipped  bool
	Optional bool
	Message  string
}

type SubprocessRunner struct {
	Timeout time.Duration
}

func (r SubprocessRunner) NextGate(ctx context.Context, scenario string) (NextGateResult, error) {
	timeout := r.Timeout
	if timeout <= 0 {
		timeout = 20 * time.Minute
	}
	runCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	cmd := exec.CommandContext(runCtx, "vrooli", "scenario", "orient", scenario, "--json") // #nosec G204 -- executable is fixed; scenario is a CLI argument, not shell-expanded.
	if root, err := repocontract.FindRepoRootFromEnvOrCWD(); err == nil {
		cmd.Dir = root
	}
	cmd.Env = os.Environ()
	out, err := cmd.CombinedOutput()
	if runCtx.Err() != nil {
		return NextGateResult{}, fmt.Errorf("vrooli scenario orient %s timed out after %s", scenario, timeout)
	}
	jsonOut := trimToJSON(out)
	if err != nil && !json.Valid(jsonOut) {
		return NextGateResult{}, fmt.Errorf("vrooli scenario orient %s failed: %w: %s", scenario, err, strings.TrimSpace(string(out)))
	}
	return ParseOrientationOutput(jsonOut)
}

func trimToJSON(out []byte) []byte {
	out = bytes.TrimSpace(out)
	if len(out) == 0 || out[0] == '{' {
		return out
	}
	start := bytes.IndexByte(out, '{')
	if start < 0 {
		return out
	}
	return out[start:]
}

type orientationJSON struct {
	Success     bool `json:"success"`
	Orientation struct {
		Scenario         string `json:"scenario"`
		Finalized        bool   `json:"finalized"`
		Completed        int32  `json:"completed"`
		Required         int32  `json:"required"`
		FinalizeRequired bool   `json:"finalize_required"`
		Message          string `json:"message"`
		NextStep         *struct {
			ID          string   `json:"id"`
			Title       string   `json:"title"`
			Description string   `json:"description"`
			Docs        []string `json:"docs"`
			Required    bool     `json:"required"`
			Complete    bool     `json:"complete"`
			Checks      []struct {
				Kind     string `json:"kind"`
				Label    string `json:"label"`
				Passed   bool   `json:"passed"`
				Skipped  bool   `json:"skipped"`
				Message  string `json:"message"`
				Optional bool   `json:"optional"`
			} `json:"checks"`
		} `json:"next_step"`
	} `json:"orientation"`
}

func ParseOrientationOutput(out []byte) (NextGateResult, error) {
	var raw orientationJSON
	if err := json.Unmarshal(out, &raw); err != nil {
		return NextGateResult{}, fmt.Errorf("parse orientation JSON: %w", err)
	}
	result := NextGateResult{
		Scenario:         raw.Orientation.Scenario,
		Finalized:        raw.Orientation.Finalized,
		Complete:         raw.Success && raw.Orientation.NextStep == nil,
		FinalizeRequired: raw.Orientation.FinalizeRequired,
		Completed:        raw.Orientation.Completed,
		Required:         raw.Orientation.Required,
		Message:          raw.Orientation.Message,
	}
	if raw.Orientation.NextStep == nil {
		return result, nil
	}
	gate := &Gate{
		ID:          raw.Orientation.NextStep.ID,
		Title:       raw.Orientation.NextStep.Title,
		Description: raw.Orientation.NextStep.Description,
		Required:    raw.Orientation.NextStep.Required,
		Complete:    raw.Orientation.NextStep.Complete,
		Docs:        raw.Orientation.NextStep.Docs,
		Remediation: remediationFor(raw.Orientation.NextStep.Docs),
	}
	for _, check := range raw.Orientation.NextStep.Checks {
		gate.Checks = append(gate.Checks, Check{
			Kind:     check.Kind,
			Label:    check.Label,
			Passed:   check.Passed,
			Skipped:  check.Skipped,
			Optional: check.Optional,
			Message:  check.Message,
		})
	}
	result.Gate = gate
	return result, nil
}

func remediationFor(docs []string) []string {
	out := []string{"Run `template-manager guidance next <scenario> --json` after each change to refresh this work order."}
	for _, doc := range docs {
		if strings.TrimSpace(doc) == "" {
			continue
		}
		out = append(out, "Read "+doc+" for the gate contract and local remediation guidance.")
	}
	return out
}
