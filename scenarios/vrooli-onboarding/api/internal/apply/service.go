// Package apply owns the transport-free apply contract used by the Connect
// handler and the onboarding composition root.
package apply

import (
	"context"

	"github.com/vrooli/vrooli/scenarios/vrooli-onboarding/internal/servicecall"
)

type Item struct {
	ID            string
	Kind          string
	Name          string
	Dependencies  []string
	Required      bool
	Privileged    bool
	ObservedState string
}

type Step struct {
	Item
	State         string
	LegacyOutcome string
	Disposition   string
	Error         string
	Remediation   string
	BlockedBy     string
	ErrorCode     string
	StartedAt     string
	CompletedAt   string
}

type Run struct {
	ID              string
	Status          string
	SelectionDigest string
	StartedAt       string
	CompletedAt     string
	Error           string
	Steps           []Step
	Blockers        []Blocker
	Degraded        []Blocker
	DegradedDigest  string
	RunnerPID       int
	Heartbeat       string
}

type Blocker struct {
	Kind        string
	Name        string
	Reason      string
	Remediation string
}

type Plan struct {
	Items     []Item
	Target    string
	PlanID    string
	Digest    string
	Revision  string
	ExpiresAt string
}

type ReviewRequest struct {
	Target           string
	PlanID           string
	PlanDigest       string
	ExpectedRevision string
}

type Review struct {
	Target           string
	PlanID           string
	PlanDigest       string
	Revision         string
	ConsentReceiptID string
	ExpiresAt        string
}

type StartRequest struct {
	Target           string
	PlanID           string
	PlanDigest       string
	ExpectedRevision string
	ConsentReceiptID string
	IdempotencyKey   string
}

type Service struct {
	Start  func(context.Context, StartRequest) (Run, error)
	Cancel func(context.Context, string) (Run, error)
	Review func(context.Context, ReviewRequest) (Review, error)
	Get    func(context.Context, string) (Run, error)
	Plan   func(context.Context, string) (Plan, error)
}

func (s Service) CancelApply(ctx context.Context, id string) (Run, error) {
	return servicecall.Invoke(s.Cancel != nil, func() (Run, error) { return s.Cancel(ctx, id) }, Run{})
}

func (s Service) StartApply(ctx context.Context, request StartRequest) (Run, error) {
	return servicecall.Invoke(s.Start != nil, func() (Run, error) { return s.Start(ctx, request) }, Run{})
}

func (s Service) ReviewApply(ctx context.Context, request ReviewRequest) (Review, error) {
	return servicecall.Invoke(s.Review != nil, func() (Review, error) { return s.Review(ctx, request) }, Review{})
}

func (s Service) GetApplyRun(ctx context.Context, id string) (Run, error) {
	return servicecall.Invoke(s.Get != nil, func() (Run, error) { return s.Get(ctx, id) }, Run{})
}

func (s Service) GetApplyPlan(ctx context.Context, target string) (Plan, error) {
	return servicecall.Invoke(s.Plan != nil, func() (Plan, error) { return s.Plan(ctx, target) }, Plan{})
}
