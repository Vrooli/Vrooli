// Package provision prepares a complete local model runtime through the same
// installation and lifecycle owners used by ordinary inference.
package provision

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"image-tools/internal/jobs"
	"image-tools/internal/models"
)

const Operation = "model_provision"
const EvidenceKey = jobs.MetaProvisioningEvidence

type Request struct {
	ModelIDs []string `json:"model_ids"`
	Plan     Plan     `json:"plan"`
}

type ModelResult struct {
	Lifecycle models.ModelLifecycle `json:"lifecycle"`
	Seconds   float64               `json:"seconds"`
	Error     string                `json:"error,omitempty"`
}

type Report struct {
	SchemaVersion       int                  `json:"schema_version"`
	StartedAt           time.Time            `json:"started_at"`
	FinishedAt          time.Time            `json:"finished_at"`
	Seconds             float64              `json:"seconds"`
	RuntimeSeconds      float64              `json:"runtime_seconds"`
	RuntimeError        string               `json:"runtime_error,omitempty"`
	Ready               bool                 `json:"ready"`
	Mode                string               `json:"mode"`
	CoverageComplete    bool                 `json:"coverage_complete"`
	SelectedModelIDs    []string             `json:"selected_model_ids"`
	CoveredOperations   []string             `json:"covered_operations,omitempty"`
	UncoveredOperations []UncoveredOperation `json:"uncovered_operations,omitempty"`
	ExcludedModels      []ExcludedModel      `json:"excluded_models,omitempty"`
	CandidateGaps       []CandidateGap       `json:"candidate_gaps,omitempty"`
	Models              []ModelResult        `json:"models"`
}

type Service struct {
	Installer     *models.Installer
	EnsureRuntime func(context.Context, func(int, string)) error
	EnsureBackend func(context.Context, models.Model) error
	RuntimeNeeded func(models.Model) bool
}

// Run retains timings and partial outcomes even when a model fails. A canceled
// client cannot cancel this server-owned job; explicit job cancellation stops
// further work and each installer releases its claim before returning.
func (s *Service) Run(ctx context.Context, job jobs.Job, emit func(int, string)) (jobs.Result, error) {
	var request Request
	if err := json.Unmarshal(job.Payload, &request); err != nil {
		return jobs.Result{}, err
	}
	if len(request.ModelIDs) == 0 && request.Plan.Mode != PlanHostFit {
		return jobs.Result{}, errors.New("provision: at least one model is required")
	}
	if emit == nil {
		emit = func(int, string) {}
	}
	mode := request.Plan.Mode
	if mode == "" {
		mode = PlanExplicit
	}
	coverageComplete := request.Plan.CoverageComplete
	if mode == PlanExplicit {
		coverageComplete = true
	}
	report := Report{
		SchemaVersion:       1,
		StartedAt:           time.Now().UTC(),
		Ready:               true,
		Mode:                mode,
		CoverageComplete:    coverageComplete,
		SelectedModelIDs:    append([]string(nil), request.ModelIDs...),
		CoveredOperations:   append([]string(nil), request.Plan.CoveredOperations...),
		UncoveredOperations: append([]UncoveredOperation(nil), request.Plan.UncoveredOperations...),
		ExcludedModels:      append([]ExcludedModel(nil), request.Plan.ExcludedModels...),
		CandidateGaps:       append([]CandidateGap(nil), request.Plan.CandidateGaps...),
		Models:              make([]ModelResult, 0, len(request.ModelIDs)),
	}
	var failures []error
	var runtimeErr error
	runtimeDone := false
	for i, id := range request.ModelIDs {
		if err := ctx.Err(); err != nil {
			failures = append(failures, err)
			break
		}
		start := time.Now()
		m, err := s.Installer.Resolve(ctx, id)
		if err == nil && m.CapabilityLabels.CommercialUse != models.CommercialUseYes {
			err = fmt.Errorf("model %s has no verified commercial-use rights", id)
		}
		if err == nil && m.Backend == models.BackendOpenRouter {
			err = fmt.Errorf("model %s is remote; local provisioning cannot supply credentials", id)
		}
		if err == nil && s.RuntimeNeeded != nil && s.RuntimeNeeded(m) {
			if !runtimeDone {
				runtimeDone = true
				began := time.Now()
				if s.EnsureRuntime != nil {
					runtimeErr = s.EnsureRuntime(ctx, emit)
				}
				report.RuntimeSeconds = time.Since(began).Seconds()
				if runtimeErr != nil {
					report.RuntimeError = runtimeErr.Error()
				}
			}
			err = runtimeErr
		}
		if err == nil && s.EnsureBackend != nil {
			err = s.EnsureBackend(ctx, m)
		}
		if err == nil {
			_, err = s.Installer.InstallForJob(ctx, id, job.ID+":"+id, func(p int, message string) {
				emit((i*100+p)/len(request.ModelIDs), id+": "+message)
			})
		}
		readCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
		state, stateErr := s.Installer.Lifecycle(readCtx, id)
		cancel()
		err = errors.Join(err, stateErr)
		if err == nil && !state.Ready() {
			err = fmt.Errorf("model %s remains %s: %s", id, state.State, state.Reason)
		}
		result := ModelResult{Lifecycle: state, Seconds: time.Since(start).Seconds()}
		if err != nil {
			result.Error = err.Error()
			failures = append(failures, err)
		}
		report.Models = append(report.Models, result)
	}
	if mode == PlanHostFit && !coverageComplete {
		var operations []string
		for _, gap := range report.UncoveredOperations {
			operations = append(operations, gap.Operation)
		}
		failures = append(failures, fmt.Errorf("provision: enabled capability coverage is incomplete (%s)", strings.Join(operations, ", ")))
	}
	report.FinishedAt = time.Now().UTC()
	report.Seconds = report.FinishedAt.Sub(report.StartedAt).Seconds()
	report.Ready = len(request.ModelIDs) > 0 && len(failures) == 0 && len(report.Models) == len(request.ModelIDs) && (mode != PlanHostFit || coverageComplete)
	raw, err := json.Marshal(report)
	if err != nil {
		return jobs.Result{}, err
	}
	return jobs.Result{Meta: map[string]string{EvidenceKey: string(raw)}}, errors.Join(failures...)
}
