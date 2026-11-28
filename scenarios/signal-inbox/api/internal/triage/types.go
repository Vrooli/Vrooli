package triage

import (
	"context"
	"fmt"
	"time"
)

type State string

const (
	New     State = "new"
	Triaged State = "triaged"
	Routed  State = "routed"
	Done    State = "done"
	Dropped State = "dropped"
)

type Author string

const (
	Operator Author = "operator"
	Agent    Author = "agent"
	System   Author = "system"
)

type OutcomeKind string

const (
	OutcomeScenario       OutcomeKind = "scenario"
	OutcomeBacklog        OutcomeKind = "backlog"
	OutcomeIdeaPipeline   OutcomeKind = "idea_pipeline"
	OutcomeKnowledgeTopic OutcomeKind = "knowledge_topic"
)

type (
	Disposition struct {
		SignalID  string
		State     State
		RevisitAt *time.Time
		UpdatedAt time.Time
	}
	Outcome struct {
		Kind     OutcomeKind
		TargetID string
	}
	Annotation struct {
		ID, SignalID string
		Author       Author
		Body         string
		Outcome      *Outcome
		CreatedAt    time.Time
	}
	Repository interface {
		GetDisposition(context.Context, string) (Disposition, bool, error)
		UpsertDisposition(context.Context, Disposition) (Disposition, error)
		AppendAnnotation(context.Context, Annotation) (Annotation, error)
		ListAnnotations(context.Context, string) ([]Annotation, error)
	}
	ErrInvalidTransition struct{ From, To State }
)

func (e ErrInvalidTransition) Error() string {
	return fmt.Sprintf("invalid disposition transition: %s -> %s", e.From, e.To)
}

type ErrInvalidTriage struct{ Reason string }

func (e ErrInvalidTriage) Error() string { return e.Reason }
