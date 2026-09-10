package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"

	"github.com/vrooli/vrooli/internal/operatorstate"
	sessiondomain "github.com/vrooli/vrooli/scenarios/vrooli-onboarding/internal/session"
)

type onboardingStep struct {
	ID        string
	Ordinal   int
	Title     string
	Route     string
	Deferred  bool
	Satisfied func(OperatorState) bool
}

var onboardingSteps = []onboardingStep{
	{ID: "welcome", Ordinal: 0, Title: "Welcome", Route: "/setup/welcome", Satisfied: func(s OperatorState) bool { return s.Session != nil }},
	{ID: "scenarios", Ordinal: 1, Title: "Scenarios", Route: "/setup/scenarios", Satisfied: func(s OperatorState) bool {
		for _, choice := range s.Scenarios {
			if choice.Enabled != nil && *choice.Enabled {
				return true
			}
		}
		return false
	}},
	{ID: "core-set", Ordinal: 2, Title: "Core Set", Route: "/setup/core-set", Satisfied: func(s OperatorState) bool { return s.Core != nil && len(s.Core.Seed) > 0 }},
	{ID: "resources", Ordinal: 3, Title: "Resources", Route: "/setup/resources", Satisfied: func(s OperatorState) bool { return s.Resources != nil }},
	{ID: "credentials", Ordinal: 4, Title: "Credentials", Route: "/setup/credentials", Satisfied: func(s OperatorState) bool { return s.Scenarios != nil }},
	{ID: "integrations", Ordinal: 5, Title: "Integrations", Route: "/setup/integrations", Deferred: true, Satisfied: func(s OperatorState) bool { return s.Version != "" }},
	{ID: "host", Ordinal: 6, Title: "Host", Route: "/setup/host", Satisfied: func(s OperatorState) bool { return s.HostTools != nil || s.HostSafeguards != nil }},
	{ID: "operating-mode", Ordinal: 7, Title: "Operating Mode", Route: "/setup/operating-mode", Satisfied: func(s OperatorState) bool { return s.Scenarios != nil }},
	{ID: "apply", Ordinal: 8, Title: "Apply", Route: "/setup/apply", Satisfied: func(s OperatorState) bool { return s.Completion != nil }},
	{ID: "validation", Ordinal: 9, Title: "Validation", Route: "/setup/validation", Satisfied: func(s OperatorState) bool { return s.Completion != nil }},
}

type sessionResponse struct {
	Step                 int    `json:"step"`
	StepID               string `json:"step_id,omitempty"`
	FirstUnsatisfiedStep int    `json:"first_unsatisfied_step"`
	Completion           bool   `json:"completion"`
}

func stepIDAt(ordinal int) string {
	for _, step := range onboardingSteps {
		if step.Ordinal == ordinal {
			return step.ID
		}
	}
	return ""
}

func stepOrdinalForID(id string) (int, bool) {
	for _, step := range onboardingSteps {
		if step.ID == id {
			return step.Ordinal, true
		}
	}
	return 0, false
}

func sessionPosition(state OperatorState) (int, string) {
	if state.Session == nil {
		return 0, stepIDAt(0)
	}
	if id := strings.TrimSpace(state.Session.StepID); id != "" {
		if ordinal, ok := stepOrdinalForID(id); ok {
			return ordinal, id
		}
	}
	return state.Session.Step, stepIDAt(state.Session.Step)
}

func (s *Server) getSession(ctx context.Context) (sessiondomain.Response, error) {
	state, err := loadOperatorStateFor(ctx)
	if err != nil {
		return sessiondomain.Response{}, err
	}
	step, stepID := sessionPosition(state)
	if state.Session != nil && state.Session.StepID == "" && stepID != "" {
		patch, _ := json.Marshal(map[string]any{"session": operatorstate.Session{Step: step, StepID: stepID}})
		if migrated, migrationErr := operatorStateService().Apply(ctx, patch); migrationErr == nil {
			state = migrated
			slog.Info("onboarding session migrated", "step", step, "step_id", stepID)
		}
	}
	return sessiondomain.Response{Step: int32(step), StepID: stepID, FirstUnsatisfiedStep: int32(firstUnsatisfiedStep(state)), Completion: state.Completion != nil}, nil
}

func (s *Server) advanceSession(ctx context.Context, stepID string) (sessiondomain.Response, error) {
	stepID = strings.TrimSpace(stepID)
	ordinal, ok := stepOrdinalForID(stepID)
	if !ok {
		return sessiondomain.Response{}, fmt.Errorf("step_id %q is outside the onboarding step model", stepID)
	}
	patch, _ := json.Marshal(map[string]any{"session": operatorstate.Session{Step: ordinal, StepID: stepID}})
	state, err := operatorStateService().Apply(ctx, patch)
	if err != nil {
		return sessiondomain.Response{}, err
	}
	step, currentID := sessionPosition(state)
	return sessiondomain.Response{Step: int32(step), StepID: currentID, FirstUnsatisfiedStep: int32(firstUnsatisfiedStep(state)), Completion: state.Completion != nil}, nil
}

func (s *Server) getStepModel(_ context.Context) (sessiondomain.Model, error) {
	steps := make([]sessiondomain.Step, 0, len(onboardingSteps))
	for _, step := range onboardingSteps {
		steps = append(steps, sessiondomain.Step{ID: step.ID, Ordinal: int32(step.Ordinal), Title: step.Title, Route: step.Route, Deferred: step.Deferred})
	}
	return sessiondomain.Model{Steps: steps}, nil
}

func operatorDraftToDomain(draft operatorstate.Draft) sessiondomain.Draft {
	return sessiondomain.Draft{Target: draft.Target, Actor: draft.Actor, BaseRevision: draft.BaseRevision, Revision: draft.Revision, StepID: draft.StepID, Choices: draft.Choices, UpdatedAt: draft.UpdatedAt}
}

func (s *Server) getDraft(ctx context.Context, target, actor string) (sessiondomain.Draft, error) {
	state, err := loadOperatorStateFor(ctx)
	if err != nil {
		return sessiondomain.Draft{}, err
	}
	draft, ok := state.Drafts[operatorstate.DraftKey(target, actor)]
	if !ok {
		return sessiondomain.Draft{}, nil
	}
	return operatorDraftToDomain(draft), nil
}

func (s *Server) saveDraft(ctx context.Context, target, actor, expectedRevision, baseRevision, stepID string, choices map[string]string) (sessiondomain.Draft, error) {
	document, err := operatorStateService().SaveDraft(ctx, target, actor, expectedRevision, baseRevision, stepID, choices)
	if err != nil {
		return sessiondomain.Draft{}, err
	}
	return operatorDraftToDomain(document.Drafts[operatorstate.DraftKey(target, actor)]), nil
}

func (s *Server) discardDraft(ctx context.Context, target, actor string) (sessiondomain.Draft, error) {
	document, err := operatorStateService().DiscardDraft(ctx, target, actor)
	if err != nil {
		return sessiondomain.Draft{}, err
	}
	return operatorDraftToDomain(document.Drafts[operatorstate.DraftKey(target, actor)]), nil
}

func operatorProfileSessionToDomain(value *operatorstate.ProfileSession, revision string) *sessiondomain.ProfileSession {
	if value == nil {
		return nil
	}
	return &sessiondomain.ProfileSession{
		Target: value.Target, Actor: value.Actor, Mode: value.Mode,
		ProfileID: value.ProfileID, ProfileVersion: value.ProfileVersion,
		CatalogRevision: value.CatalogRevision, BaseRevision: value.BaseRevision,
		Answers:         cloneRawMessages(value.Answers),
		ManualDecisions: cloneBools(value.ManualDecisions),
		TargetContext:   cloneStrings(value.TargetContext), UpdatedAt: value.UpdatedAt, Revision: revision,
	}
}

func (s *Server) getProfileSession(ctx context.Context, target, actor string) (*sessiondomain.ProfileSession, error) {
	document, err := operatorStateService().Load(ctx)
	if err != nil {
		return nil, err
	}
	if document.Session == nil || document.Session.Profile == nil {
		return nil, nil
	}
	value := document.Session.Profile
	if value.Target != strings.TrimSpace(target) || value.Actor != strings.TrimSpace(actor) {
		return nil, fmt.Errorf("profile session is outside the requested target or actor scope")
	}
	return s.reconcileProfileSession(ctx, operatorProfileSessionToDomain(value, operatorstate.Revision(document))), nil
}

func (s *Server) saveProfileSession(ctx context.Context, value sessiondomain.ProfileSession, expectedRevision string) (*sessiondomain.ProfileSession, error) {
	current, err := operatorStateService().ProfileSession(ctx)
	if err != nil {
		return nil, err
	}
	if current != nil && (current.Target != strings.TrimSpace(value.Target) || current.Actor != strings.TrimSpace(value.Actor)) {
		return nil, fmt.Errorf("profile session is already owned by another target or actor")
	}
	document, err := operatorStateService().SaveProfileSession(ctx, operatorstate.ProfileSession{
		Target: value.Target, Actor: value.Actor, Mode: value.Mode,
		ProfileID: value.ProfileID, ProfileVersion: value.ProfileVersion,
		CatalogRevision: value.CatalogRevision, BaseRevision: value.BaseRevision,
		Answers: cloneRawMessages(value.Answers), ManualDecisions: cloneBools(value.ManualDecisions),
		TargetContext: cloneStrings(value.TargetContext),
	}, expectedRevision)
	if err != nil {
		return nil, err
	}
	return s.reconcileProfileSession(ctx, operatorProfileSessionToDomain(document.Session.Profile, operatorstate.Revision(document))), nil
}

// reconcileProfileSession compares the persisted profile reference with the
// current owner-authored catalog. A changed or revoked profile is surfaced as
// review-required; it never silently changes the selected profile or answers.
func (s *Server) reconcileProfileSession(ctx context.Context, value *sessiondomain.ProfileSession) *sessiondomain.ProfileSession {
	if value == nil {
		return nil
	}
	value.ReconciliationState = "manual"
	if strings.TrimSpace(value.ProfileID) == "" {
		return value
	}
	profiles, err := onboardingProfilesService().List(ctx)
	if err != nil {
		value.ReconciliationState = "unavailable"
		value.ReconciliationReasons = []string{"The current profile catalog could not be checked."}
		return value
	}
	for _, profile := range profiles {
		if profile.ID != value.ProfileID {
			continue
		}
		value.CurrentProfileVersion = profile.Version
		if strings.TrimSpace(value.ProfileVersion) != "" && profile.Version != value.ProfileVersion {
			value.ReconciliationState = "review_required"
			value.ReconciliationReasons = []string{fmt.Sprintf("Profile %q changed from version %s to %s.", value.ProfileID, value.ProfileVersion, profile.Version)}
			return value
		}
		value.ReconciliationState = "current"
		return value
	}
	value.ReconciliationState = "profile_revoked"
	value.ReconciliationReasons = []string{fmt.Sprintf("Profile %q is no longer available; existing answers were retained.", value.ProfileID)}
	return value
}

func cloneRawMessages(values map[string]json.RawMessage) map[string]json.RawMessage {
	if len(values) == 0 {
		return nil
	}
	result := make(map[string]json.RawMessage, len(values))
	for key, value := range values {
		result[key] = append(json.RawMessage(nil), value...)
	}
	return result
}

func cloneBools(values map[string]bool) map[string]bool {
	if len(values) == 0 {
		return nil
	}
	result := make(map[string]bool, len(values))
	for key, value := range values {
		result[key] = value
	}
	return result
}

func cloneStrings(values map[string]string) map[string]string {
	if len(values) == 0 {
		return nil
	}
	result := make(map[string]string, len(values))
	for key, value := range values {
		result[key] = value
	}
	return result
}

type stepModelResponse struct {
	ID       string `json:"id"`
	Ordinal  int    `json:"ordinal"`
	Title    string `json:"title"`
	Route    string `json:"route"`
	Deferred bool   `json:"deferred"`
}

func publicStepModel() []stepModelResponse {
	model := make([]stepModelResponse, 0, len(onboardingSteps))
	for _, step := range onboardingSteps {
		model = append(model, stepModelResponse{ID: step.ID, Ordinal: step.Ordinal, Title: step.Title, Route: step.Route, Deferred: step.Deferred})
	}
	return model
}

func firstUnsatisfiedStep(state OperatorState) int {
	for _, step := range onboardingSteps {
		if !step.Satisfied(state) {
			return step.Ordinal
		}
	}
	return len(onboardingSteps) - 1
}
