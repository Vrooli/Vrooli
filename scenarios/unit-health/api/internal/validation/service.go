// Package validation is Unit Health's core engine: it discovers test surfaces,
// plans and (in later phases) runs the canonical test commands, analyzes
// coverage/architecture/quality, and produces provider-local test maturity.
//
// Phase 2 ships the skeleton: a schema-valid Response that honestly reports a
// not-yet-implemented engine. Phase 3 wires Code Facts intake and the test plan
// builder, Phase 4 the bounded executor, and Phase 5 the analyzers and the
// maturity assessor. The Response shape mirrors the validation.proto contract
// one-to-one so the handler conversion stays a flat field copy.
package validation

import (
	"context"
	"fmt"
	"time"
)

// Service runs Unit Health validation. Seams (Code Facts discoverer, executor,
// analyzers) land as fields in later phases.
type Service struct {
	Now func() time.Time
}

// Request identifies the validation target and execution options.
type Request struct {
	Scenario         string
	Path             string
	Workspaces       []string
	IncludeExecution bool
	UseCache         bool
}

// Response is the engine's normalized result. It maps one-to-one onto
// validationv1.ValidateScenarioResponse.
type Response struct {
	RunID          string
	Status         string
	Summary        string
	Scenario       string
	TargetKind     string
	TargetPath     string
	DegradedReason string
	Surfaces       []Surface
	Workspaces     []Workspace
	Plan           ExecutionPlan
	CommandResults []CommandResult
	Coverage       []CoverageTarget
	Findings       []Finding
	Diagnostics    []Diagnostic
	Maturity       Maturity
	NextSteps      []string
}

// Surface is a discovered scenario surface from Code Facts.
type Surface struct {
	ID             string
	Kind           string
	Language       string
	Framework      string
	RootPath       string
	PackageManager string
	Status         string
	Confidence     float64
}

// Workspace is a testable unit with its canonical framework and commands.
type Workspace struct {
	ID                 string
	Language           string
	RootPath           string
	Framework          string
	CanonicalFramework string
	TestCommand        string
	CoverageCommand    string
	PackageManager     string
	Status             string
	DegradedReason     string
}

// ExecutionPlan is the bounded set of commands Unit Health would run.
type ExecutionPlan struct {
	Commands []PlannedCommand
	Notes    string
}

// PlannedCommand is a single command in the execution plan.
type PlannedCommand struct {
	WorkspaceID      string
	Name             string
	Command          string
	WorkingDirectory string
	TimeoutSeconds   int
}

// CommandResult is the outcome of one executed command.
type CommandResult struct {
	Name             string
	Command          string
	WorkingDirectory string
	Status           string
	ExitCode         int
	StdoutExcerpt    string
	StderrExcerpt    string
	TimeoutSeconds   int
	FailureReason    string
	FailureClass     string
	DurationMS       int64
}

// CoverageTarget is per-file/per-surface coverage.
type CoverageTarget struct {
	ID              string
	Language        string
	SurfaceID       string
	FilePath        string
	CoveredLines    int64
	TotalLines      int64
	CoveragePercent float64
	Threshold       float64
	Status          string
}

// Finding is a normalized Unit Health finding. Code maps to a
// `.vrooli/maturity.json` entry.
type Finding struct {
	ID            string
	Scenario      string
	SurfaceID     string
	WorkspaceID   string
	Language      string
	Framework     string
	Code          string
	Category      string
	Severity      string
	FilePath      string
	Symbol        string
	Message       string
	Evidence      string
	Expected      string
	Observed      string
	WhyItMatters  string
	Remediation   string
	SourceCommand string
	CreatedAt     string
}

// Diagnostic is a flake/runtime/hang diagnostic.
type Diagnostic struct {
	Kind        string
	WorkspaceID string
	Message     string
	Evidence    string
	Severity    string
}

// Maturity is the provider-local maturity summary.
type Maturity struct {
	Rung      int
	Label     string
	Rationale string
}

// New returns a Service with default (real-clock) wiring.
func New() *Service { return &Service{} }

func (s *Service) now() time.Time {
	if s.Now != nil {
		return s.Now()
	}
	return time.Now()
}

// Validate runs the Unit Health validation for the requested target.
//
// Phase 2 skeleton: returns a schema-valid, honestly-degraded Response with no
// findings. Later phases replace the body with real discovery/execution/
// analysis while keeping this signature stable.
func (s *Service) Validate(_ context.Context, req Request) (Response, error) {
	now := s.now()
	target := req.Scenario
	if target == "" {
		target = req.Path
	}
	return Response{
		RunID:          "uh-" + now.UTC().Format("20060102-150405"),
		Status:         "degraded",
		Scenario:       req.Scenario,
		TargetKind:     "scenario",
		TargetPath:     req.Path,
		DegradedReason: "Unit Health validation engine is not yet implemented; the Phase 2 skeleton returns an empty assessment.",
		Maturity:       Maturity{Rung: 0, Label: "L0", Rationale: "Validation engine not yet implemented."},
		Summary:        fmt.Sprintf("%s: Unit Health validation skeleton (no test surfaces analyzed yet).", target),
		NextSteps:      []string{"Phase 3 wires Code Facts intake and the test plan builder."},
	}, nil
}
