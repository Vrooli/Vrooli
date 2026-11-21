package attempt

import (
	"context"
	"fmt"
	"strings"
)

// DecisionRequest is the transport-neutral operator decision envelope. It is
// deliberately independent of any domain model so one Connect RPC can route
// every attempt without importing domain packages into each other.
type DecisionRequest struct {
	SubjectKind         string
	SubjectRef          string
	RoundNum            int
	Decision            string
	Actor               string
	Rationale           string
	AcceptedProposalIDs []string
}

type DecisionResult struct {
	Decision  string
	Status    string
	Rationale string
	DecidedAt string
}

type Decider interface {
	DecideAttempt(context.Context, DecisionRequest) (DecisionResult, error)
}

type DeciderFunc func(context.Context, DecisionRequest) (DecisionResult, error)

func (f DeciderFunc) DecideAttempt(ctx context.Context, request DecisionRequest) (DecisionResult, error) {
	return f(ctx, request)
}

// Router is the one domain-neutral dispatch point for operator decisions.
// Registration happens at composition; domain adapters retain mutation
// authority and only receive their own typed subject references.
type Router struct{ deciders map[string]Decider }

func NewRouter() *Router { return &Router{deciders: make(map[string]Decider)} }

func (r *Router) Register(subjectKind string, decider Decider) error {
	key := strings.TrimSpace(subjectKind)
	if key == "" || decider == nil {
		return fmt.Errorf("attempt decision registration requires subject kind and decider")
	}
	if _, exists := r.deciders[key]; exists {
		return fmt.Errorf("attempt decision decider already registered for %q", key)
	}
	r.deciders[key] = decider
	return nil
}

func (r *Router) DecideAttempt(ctx context.Context, request DecisionRequest) (DecisionResult, error) {
	if strings.TrimSpace(request.Actor) == "" {
		return DecisionResult{}, fmt.Errorf("attempt decision actor is required")
	}
	if request.RoundNum < 1 {
		return DecisionResult{}, fmt.Errorf("attempt decision round number is required")
	}
	decider, ok := r.deciders[strings.TrimSpace(request.SubjectKind)]
	if !ok {
		return DecisionResult{}, fmt.Errorf("unsupported attempt subject_kind %q", request.SubjectKind)
	}
	return decider.DecideAttempt(ctx, request)
}
