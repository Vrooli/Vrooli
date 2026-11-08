// Package analysisoperations owns durable, bounded fleet analysis and graph
// rebuild operations. HTTP observers never own the execution context.
package analysisoperations

import (
	"context"
	"errors"
	"time"
)

type Kind string

const (
	FleetAnalysis Kind = "fleet-analysis"
	GraphRebuild  Kind = "graph-rebuild"
)

type State string

const (
	Queued      State = "queued"
	Running     State = "running"
	Succeeded   State = "succeeded"
	Partial     State = "partial"
	Failed      State = "failed"
	Interrupted State = "interrupted"
)

func (s State) Terminal() bool {
	switch s {
	case Succeeded, Partial, Failed, Interrupted:
		return true
	default:
		return false
	}
}

var (
	ErrNotFound = errors.New("analysis operation not found")
	ErrConflict = errors.New("caller operation ID is bound to different input")
	ErrCapacity = errors.New("analysis operation receipt capacity exhausted")
	ErrState    = errors.New("analysis operation state changed")
	ErrStopped  = errors.New("analysis operation manager is not accepting work")
	ErrInvalid  = errors.New("invalid analysis operation request")
)

const (
	MaxScenarios       = 1024
	MaxCallerIDs       = 1024
	MaxReceiptKeys     = 100000
	RetainedDetails    = 200
	MaxDiagnosticBytes = 512
)

type Spec struct {
	Kind       Kind
	ApplyGraph bool
}

func (s Spec) signature() string {
	if s.ApplyGraph {
		return string(s.Kind) + ":apply"
	}
	return string(s.Kind) + ":preview"
}

type ScenarioOutcome struct {
	Scenario               string
	State                  State
	DetectedResources      int
	ScenarioDependencies   int
	DeploymentDependencies int
	Duration               time.Duration
	Error                  string
}

type GraphOutcome struct {
	Applied         bool
	Edges           int
	ScenarioEdges   int
	ResourceEdges   int
	DegradedSources []string
}

type Operation struct {
	ID            string
	Spec          Spec
	State         State
	CreatedAt     time.Time
	StartedAt     *time.Time
	CompletedAt   *time.Time
	Total         int
	Completed     int
	Failures      int
	Error         string
	Scenarios     []ScenarioOutcome
	Graph         *GraphOutcome
	DetailsPruned bool
}

// Repository hides persistence from lifecycle and wire handling. Item and
// terminal writes are fenced against interrupted/completed operations.
type Repository interface {
	Submit(context.Context, string, Spec, []string, time.Time) (Operation, bool, error)
	Get(context.Context, string) (Operation, error)
	ClaimNext(context.Context, time.Time) (Operation, error)
	StartScenario(context.Context, string, string) error
	CompleteScenario(context.Context, string, ScenarioOutcome) error
	Finish(context.Context, string, State, string, *GraphOutcome, time.Time) error
	Recover(context.Context, time.Time) ([]string, error)
	Prune(context.Context) error
}

// Runner interprets source files and builds graphs through the existing owner.
// Analyze returns only a summary; detailed reports stay in their existing home.
type Runner interface {
	Scenarios(context.Context) ([]string, error)
	Analyze(context.Context, string) (ScenarioOutcome, error)
	RebuildGraph(context.Context, bool) (GraphOutcome, error)
}
