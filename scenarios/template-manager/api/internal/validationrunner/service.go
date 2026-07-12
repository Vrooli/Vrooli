package validationrunner

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/vrooli/vrooli/scenarios/template-manager/api/internal/catalog"

	"github.com/google/uuid"
)

type Runner interface {
	ValidateTemplate(ctx context.Context, req ValidateRequest) (ValidateResult, error)
	RecordFleetDrift(ctx context.Context) (DriftResult, error)
}

type ValidateRequest struct {
	TemplateID    string
	Mode          catalog.ValidationMode
	Trigger       string
	TestPreset    string
	WarningPolicy string
	RetainTemp    bool
}

type ValidateResult struct {
	Success      bool
	Mode         catalog.ValidationMode
	TemplateID   string
	Target       string
	PhaseResults []catalog.PhaseResult
	Findings     []catalog.ValidationFinding
}

type DriftResult struct {
	Success   bool
	Scenarios []DriftScenario
}

type DriftScenario struct {
	Scenario        string
	TemplateID      string
	Status          string
	ManifestDrifted bool
	ContentDrifted  bool
	Message         string
}

type Service struct {
	repo   catalog.Repository
	runner Runner
	now    func() time.Time
}

func NewService(repo catalog.Repository, runner Runner) *Service {
	return &Service{
		repo:   repo,
		runner: runner,
		now:    func() time.Time { return time.Now().UTC() },
	}
}

func (s *Service) RunValidation(ctx context.Context, req ValidateRequest) (catalog.ValidationRun, error) {
	if strings.TrimSpace(req.TemplateID) == "" {
		req.TemplateID = "react-vite"
	}
	if req.Mode == "" {
		req.Mode = catalog.ModeShallow
	}
	// A deep validation owns a Test Genie run whose completion is evidence even
	// when the HTTP client disconnects. Detach the complete engine-and-persist
	// path from the request here, rather than only detaching the final writes.
	executionCtx := context.WithoutCancel(ctx)
	started := s.now()
	result, err := s.runner.ValidateTemplate(executionCtx, req)
	finished := s.now()
	if err != nil {
		return catalog.ValidationRun{}, err
	}
	if result.TemplateID == "" {
		result.TemplateID = req.TemplateID
	}
	if result.Mode == "" {
		result.Mode = req.Mode
	}
	if result.Target == "" {
		result.Target = fmt.Sprintf("templates/scenarios/%s", result.TemplateID)
	}
	// The real Test Genie execution is server-owned and can finish after the
	// caller's HTTP attachment has gone away. Its completed result is evidence,
	// so persist the terminal run and debt projection independently of request
	// cancellation rather than losing it at the final write.
	persistenceCtx := executionCtx
	status := "failed"
	if result.Success {
		status = "passed"
	}
	run := catalog.ValidationRun{
		ID:           "validation-" + uuid.NewString(),
		TemplateID:   result.TemplateID,
		Mode:         result.Mode,
		Target:       result.Target,
		Status:       status,
		Trigger:      nonEmpty(req.Trigger, "manual"),
		StartedAt:    started,
		FinishedAt:   finished,
		PhaseResults: result.PhaseResults,
		Findings:     result.Findings,
	}
	if err := s.repo.SaveValidationRun(persistenceCtx, run); err != nil {
		return catalog.ValidationRun{}, err
	}
	// A clean deep validation is the release gate for source-owned debt. Fleet
	// drift is intentionally excluded: it describes downstream divergence, not
	// a defect in the current template release.
	if run.Mode == catalog.ModeDeep && run.Status == "passed" && len(run.Findings) == 0 {
		if err := s.repo.ResolveSourceDebt(persistenceCtx, run.TemplateID, finished); err != nil {
			return catalog.ValidationRun{}, err
		}
	}
	if run.Mode == catalog.ModeDeep && run.Status != "passed" {
		if err := s.repo.ResolveSupersededDeepValidationDebt(persistenceCtx, run.TemplateID, finished); err != nil {
			return catalog.ValidationRun{}, err
		}
	}
	for _, finding := range result.Findings {
		if strings.TrimSpace(finding.Key) == "" {
			continue
		}
		if err := s.repo.UpsertDebt(persistenceCtx, catalog.DebtEntry{
			Key:         finding.Key,
			TemplateID:  run.TemplateID,
			Source:      nonEmpty(finding.Source, "template validation"),
			Severity:    nonEmpty(finding.Severity, "warning"),
			Status:      "open",
			Title:       nonEmpty(finding.Summary, finding.Key),
			Detail:      nonEmpty(finding.Summary, finding.Key),
			FirstSeenAt: started,
			LastSeenAt:  finished,
		}); err != nil {
			return catalog.ValidationRun{}, err
		}
	}
	return run, nil
}

func (s *Service) RecordFleetDrift(ctx context.Context) (catalog.DriftSnapshot, error) {
	started := s.now()
	result, err := s.runner.RecordFleetDrift(ctx)
	if err != nil {
		return catalog.DriftSnapshot{}, err
	}
	driftCount := int32(0)
	for _, scenario := range result.Scenarios {
		if scenario.ManifestDrifted || scenario.ContentDrifted || scenario.Status == "drifted" {
			driftCount++
			key := "drift." + scenario.Scenario
			if scenario.TemplateID != "" {
				key = "drift." + scenario.TemplateID + "." + scenario.Scenario
			}
			if err := s.repo.UpsertDebt(ctx, catalog.DebtEntry{
				Key:         key,
				TemplateID:  nonEmpty(scenario.TemplateID, "react-vite"),
				Source:      "template drift",
				Severity:    "warning",
				Status:      "open",
				Title:       fmt.Sprintf("%s has template drift", scenario.Scenario),
				Detail:      nonEmpty(scenario.Message, fmt.Sprintf("%s reported %s during fleet drift", scenario.Scenario, scenario.Status)),
				FirstSeenAt: started,
				LastSeenAt:  started,
			}); err != nil {
				return catalog.DriftSnapshot{}, err
			}
		}
	}
	status := "failed"
	if result.Success {
		status = "recorded"
	}
	snapshot := catalog.DriftSnapshot{
		ID:         "drift-" + uuid.NewString(),
		TemplateID: "react-vite",
		Target:     "fleet",
		Status:     status,
		DriftCount: driftCount,
		CapturedAt: started,
	}
	if err := s.repo.SaveDriftSnapshot(ctx, snapshot); err != nil {
		return catalog.DriftSnapshot{}, err
	}
	return snapshot, nil
}

func nonEmpty(value, fallback string) string {
	if strings.TrimSpace(value) != "" {
		return value
	}
	return fallback
}
