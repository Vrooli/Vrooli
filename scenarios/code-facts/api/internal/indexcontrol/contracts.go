// Package indexcontrol owns durable reconciliation and generation controls.
package indexcontrol

import (
	"context"
	"time"
)

type Job struct {
	ID                    string
	Kind                  string
	State                 string
	Generation            string
	Progress              int64
	Total                 int64
	Error                 string
	Cursor                string
	CancellationRequested bool
	CreatedAt             time.Time
	UpdatedAt             time.Time
}

type JobStore interface {
	Create(context.Context, Job) error
	Update(context.Context, Job) error
	Get(context.Context, string) (Job, error)
	ListActive(context.Context) ([]Job, error)
	RequestCancel(context.Context, string, time.Time) error
	RecoverInterrupted(context.Context, time.Time) ([]Job, error)
}

type Status struct {
	ActiveGeneration     string
	PreviousGeneration   string
	State                string
	SourceFiles          int64
	SearchDocuments      int64
	SemanticCards        int64
	GraphFacts           int64
	StorageBytes         int64
	LastReconcileAt      time.Time
	LastReconcileOutcome string
	DescriptorDigest     string
	SourceDigest         string
	Degraded             []string
	ActiveJobs           []Job
}

type StatusReader interface {
	Status(context.Context) (Status, error)
}

type Change struct {
	Path      string
	Operation string
	Hash      string
}

type ChangeBatch struct {
	Cursor     string
	NextCursor string
	Changes    []Change
	Done       bool
}

type ChangeSource interface {
	Changes(context.Context, string, int) (ChangeBatch, error)
}

type Processor interface {
	Apply(context.Context, string, []Change) (int64, error)
}

type GenerationLifecycle interface {
	Active(context.Context) (string, error)
	BeginShadow(context.Context, string) error
	CompleteShadow(context.Context, string) error
	Activate(context.Context, string) error
	Rollback(context.Context, string) error
}

type AliasController interface {
	Promote(context.Context, string) error
	Rollback(context.Context, string) error
}

type GenerationValidator interface {
	Validate(context.Context, string) error
}

type PromotionStore interface {
	Prepare(context.Context, string, string, string, time.Time) error
	Transition(context.Context, string, string, string, time.Time) error
}

type Reconciler interface {
	Reconcile(context.Context, string) (Job, error)
	Cancel(context.Context, string) error
	Promote(context.Context, string) error
	Rollback(context.Context, string) error
	Cleanup(context.Context) error
}

type ProcessRunner interface {
	Run(context.Context, string, ...string) ([]byte, error)
}

type Clock interface{ Now() time.Time }
