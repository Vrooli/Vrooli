package guidance

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/vrooli/vrooli/scenarios/template-manager/api/internal/templatecontracts"
	"github.com/vrooli/vrooli/scenarios/template-manager/api/internal/templateengine"
)

type Runner interface {
	NextGate(ctx context.Context, scenario string) (NextGateResult, error)
}

type Service struct {
	runner Runner
}

func NewService(runner Runner) *Service {
	if runner == nil {
		engine, err := templateengine.New("")
		if err != nil {
			runner = errorRunner{err: err}
		} else {
			runner = EngineRunner{Engine: engine}
		}
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

type EngineRunner struct {
	Engine  *templateengine.Engine
	Timeout time.Duration
}

func (r EngineRunner) NextGate(ctx context.Context, scenario string) (NextGateResult, error) {
	if r.Engine == nil {
		return NextGateResult{}, fmt.Errorf("template engine unavailable")
	}
	timeout := r.Timeout
	if timeout <= 0 {
		timeout = 20 * time.Minute
	}
	runCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	report, err := r.Engine.OrientScenario(runCtx, templatecontracts.OrientationRequest{Name: scenario, JSON: true})
	if runCtx.Err() != nil {
		return NextGateResult{}, fmt.Errorf("orient %s timed out after %s", scenario, timeout)
	}
	if err != nil {
		return NextGateResult{}, fmt.Errorf("orient %s: %w", scenario, err)
	}
	return ResultFromOrientationReport(report), nil
}

type errorRunner struct {
	err error
}

func (r errorRunner) NextGate(context.Context, string) (NextGateResult, error) {
	return NextGateResult{}, r.err
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

func ResultFromOrientationReport(report templatecontracts.OrientationReport) NextGateResult {
	result := NextGateResult{
		Scenario:         report.Scenario,
		Finalized:        report.Finalized,
		Complete:         report.NextStep == nil,
		FinalizeRequired: report.FinalizeRequired,
		Completed:        int32(report.Completed),
		Required:         int32(report.Required),
		Message:          report.Message,
	}
	if report.NextStep == nil {
		return result
	}
	gate := &Gate{
		ID:          report.NextStep.ID,
		Title:       report.NextStep.Title,
		Description: report.NextStep.Description,
		Required:    report.NextStep.Required,
		Complete:    report.NextStep.Complete,
		Docs:        report.NextStep.Docs,
		Remediation: remediationFor(report.NextStep.Docs),
	}
	for _, check := range report.NextStep.Checks {
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
	return result
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
