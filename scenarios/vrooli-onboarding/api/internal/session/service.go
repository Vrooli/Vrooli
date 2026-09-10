// Package session owns the transport free onboarding wizard pointer and step
// model. It deliberately has no HTTP, Connect, or router dependency.
package session

import (
	"context"
	"encoding/json"

	"github.com/vrooli/vrooli/scenarios/vrooli-onboarding/internal/servicecall"
)

type Step struct {
	ID       string
	Ordinal  int32
	Title    string
	Route    string
	Deferred bool
}

type Response struct {
	Step                 int32
	StepID               string
	FirstUnsatisfiedStep int32
	Completion           bool
}

type Model struct {
	Steps []Step
}

type Draft struct {
	Target       string
	Actor        string
	BaseRevision string
	Revision     string
	StepID       string
	Choices      map[string]string
	UpdatedAt    string
}

type ProfileSession struct {
	Target                string
	Actor                 string
	Mode                  string
	ProfileID             string
	ProfileVersion        string
	CatalogRevision       string
	BaseRevision          string
	Answers               map[string]json.RawMessage
	ManualDecisions       map[string]bool
	TargetContext         map[string]string
	UpdatedAt             string
	Revision              string
	ReconciliationState   string
	CurrentProfileVersion string
	ReconciliationReasons []string
}

type Service struct {
	Get                  func(context.Context) (Response, error)
	Advance              func(context.Context, string) (Response, error)
	Model                func(context.Context) (Model, error)
	GetDraftFn           func(context.Context, string, string) (Draft, error)
	SaveDraftFn          func(context.Context, string, string, string, string, string, map[string]string) (Draft, error)
	DiscardDraftFn       func(context.Context, string, string) (Draft, error)
	GetProfileSessionFn  func(context.Context, string, string) (*ProfileSession, error)
	SaveProfileSessionFn func(context.Context, ProfileSession, string) (*ProfileSession, error)
}

func (s Service) GetSession(ctx context.Context) (Response, error) {
	return servicecall.Invoke(s.Get != nil, func() (Response, error) { return s.Get(ctx) }, Response{})
}

func (s Service) AdvanceSessionStep(ctx context.Context, stepID string) (Response, error) {
	return servicecall.Invoke(s.Advance != nil, func() (Response, error) { return s.Advance(ctx, stepID) }, Response{})
}

func (s Service) GetStepModel(ctx context.Context) (Model, error) {
	return servicecall.Invoke(s.Model != nil, func() (Model, error) { return s.Model(ctx) }, Model{})
}

func (s Service) GetDraft(ctx context.Context, target, actor string) (Draft, error) {
	return servicecall.Invoke(s.GetDraftFn != nil, func() (Draft, error) { return s.GetDraftFn(ctx, target, actor) }, Draft{})
}

func (s Service) SaveDraft(ctx context.Context, target, actor, expectedRevision, baseRevision, stepID string, choices map[string]string) (Draft, error) {
	return servicecall.Invoke(s.SaveDraftFn != nil, func() (Draft, error) {
		return s.SaveDraftFn(ctx, target, actor, expectedRevision, baseRevision, stepID, choices)
	}, Draft{})
}

func (s Service) DiscardDraft(ctx context.Context, target, actor string) (Draft, error) {
	return servicecall.Invoke(s.DiscardDraftFn != nil, func() (Draft, error) { return s.DiscardDraftFn(ctx, target, actor) }, Draft{})
}

func (s Service) GetProfileSession(ctx context.Context, target, actor string) (*ProfileSession, error) {
	return servicecall.Invoke(s.GetProfileSessionFn != nil, func() (*ProfileSession, error) {
		return s.GetProfileSessionFn(ctx, target, actor)
	}, nil)
}

func (s Service) SaveProfileSession(ctx context.Context, value ProfileSession, expectedRevision string) (*ProfileSession, error) {
	return servicecall.Invoke(s.SaveProfileSessionFn != nil, func() (*ProfileSession, error) {
		return s.SaveProfileSessionFn(ctx, value, expectedRevision)
	}, nil)
}
