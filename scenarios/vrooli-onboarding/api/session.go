package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"sort"
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
		CatalogRevision: value.CatalogRevision, ConsequenceDigest: value.ConsequenceDigest,
		BaseRevision:    value.BaseRevision,
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
	// The persisted session is authoritative for the previous consequence
	// digest. A client may send it for convenience, but it cannot choose the
	// digest used to decide whether an old apply receipt is still valid.
	if current != nil {
		value.ConsequenceDigest = current.ConsequenceDigest
	} else {
		value.ConsequenceDigest = ""
	}
	reconciled := s.reconcileProfileSession(ctx, &value)
	if current != nil {
		referenceChanges := diffProfileReferences(current, &value)
		reconciled.ReconciliationChanges = append(reconciled.ReconciliationChanges, referenceChanges...)
		for _, change := range referenceChanges {
			if change.RequiresReview {
				reconciled.ReconciliationState = "review_required"
				reconciled.ReconciliationReasons = append(reconciled.ReconciliationReasons, change.Impact)
				if reconciled.NextAction == "" {
					reconciled.NextAction = "review-profile"
				}
			}
		}
		contextChanges := diffTargetContext(current.TargetContext, value.TargetContext)
		reconciled.ReconciliationChanges = append(reconciled.ReconciliationChanges, contextChanges...)
		for _, change := range contextChanges {
			if change.RequiresReview {
				reconciled.ReconciliationState = "review_required"
				reconciled.ReconciliationReasons = append(reconciled.ReconciliationReasons, change.Impact)
				if reconciled.NextAction == "" {
					reconciled.NextAction = "review-profile"
				}
			}
		}
		sort.Strings(reconciled.ReconciliationReasons)
		sortReconciliationChanges(reconciled.ReconciliationChanges)
	}
	document, err := operatorStateService().SaveProfileSession(ctx, operatorstate.ProfileSession{
		Target: value.Target, Actor: value.Actor, Mode: value.Mode,
		ProfileID: value.ProfileID, ProfileVersion: value.ProfileVersion,
		CatalogRevision: value.CatalogRevision, ConsequenceDigest: reconciled.ConsequenceDigest,
		BaseRevision: value.BaseRevision,
		Answers:      cloneRawMessages(value.Answers), ManualDecisions: cloneBools(value.ManualDecisions),
		TargetContext: cloneStrings(value.TargetContext),
	}, expectedRevision)
	if err != nil {
		return nil, err
	}
	reconciled.Revision = operatorstate.Revision(document)
	if document.Session != nil && document.Session.Profile != nil {
		// Return the durable timestamp assigned by operatorstate. The UI uses
		// the response as its new resume point, so an empty timestamp here would
		// make a successful save look like an uncommitted client draft.
		reconciled.UpdatedAt = document.Session.Profile.UpdatedAt
	}
	return reconciled, nil
}

// reconcileProfileSession compares the persisted profile reference with the
// current owner-authored catalog. A changed or revoked profile is surfaced as
// review-required; it never silently changes the selected profile or answers.
func (s *Server) reconcileProfileSession(ctx context.Context, value *sessiondomain.ProfileSession) *sessiondomain.ProfileSession {
	if value == nil {
		return nil
	}
	previousDigest := value.ConsequenceDigest
	value.ReconciliationState = "manual"
	value.CurrentProfileVersion = ""
	value.ReconciliationReasons = nil
	value.ReconciliationChanges = nil
	value.NextQuestionID = ""
	value.NextAction = ""
	if strings.TrimSpace(value.ProfileID) == "" {
		value.NextAction = "choose-profile"
		return value
	}
	profiles, err := onboardingProfilesService().List(ctx)
	if err != nil {
		value.ReconciliationState = "unavailable"
		value.ReconciliationReasons = []string{"The current profile catalog could not be checked."}
		value.NextAction = "retry-profile-check"
		return value
	}
	var currentProfileVersion string
	for _, profile := range profiles {
		if profile.ID != value.ProfileID {
			continue
		}
		currentProfileVersion = profile.Version
		break
	}
	if currentProfileVersion == "" {
		value.ReconciliationState = "profile_revoked"
		value.ReconciliationReasons = []string{fmt.Sprintf("Profile %q is no longer available; existing answers were retained.", value.ProfileID)}
		value.ReconciliationChanges = []sessiondomain.ReconciliationChange{{
			Kind: "profile-revoked", Field: "profileId", Before: value.ProfileID, After: "unavailable",
			Impact: "Existing answers remain retained, but a profile must be selected before applying changes.", RequiresReview: true,
		}}
		value.NextAction = "choose-profile"
		return value
	}
	value.CurrentProfileVersion = currentProfileVersion
	if strings.TrimSpace(value.ProfileVersion) != "" && currentProfileVersion != value.ProfileVersion {
		value.ReconciliationChanges = append(value.ReconciliationChanges, sessiondomain.ReconciliationChange{
			Kind: "profile-version", Field: "profileVersion", Before: value.ProfileVersion, After: currentProfileVersion,
			Impact: "Review profile changes before accepting new recommendations or privileges.", RequiresReview: true,
		})
	}

	answers := rawAnswerValues(value.Answers)
	targetContext := make(map[string]any, len(value.TargetContext)+1)
	for key, item := range value.TargetContext {
		targetContext[key] = item
	}
	if value.CatalogRevision != "" {
		targetContext["catalogRevision"] = value.CatalogRevision
	}
	evaluation, err := onboardingProfilesService().EvaluateWithManualDecisions(ctx, value.ProfileID, answers, targetContext, value.ManualDecisions)
	if err != nil {
		value.ReconciliationState = "unavailable"
		value.ReconciliationReasons = []string{fmt.Sprintf("The selected profile could not be evaluated: %v", err)}
		value.NextAction = "retry-profile-check"
		return value
	}
	value.ConsequenceDigest = evaluation.Digest
	if previousDigest != "" && previousDigest != evaluation.Digest {
		value.ReconciliationChanges = append(value.ReconciliationChanges, sessiondomain.ReconciliationChange{
			Kind: "consequence-digest", Field: "consequenceDigest", Before: previousDigest, After: evaluation.Digest,
			Impact: "Any previous apply review for this profile must be reviewed again.", RequiresReview: true,
		})
	}
	knownQuestions := make(map[string]bool, len(evaluation.KnownQuestionIDs))
	for _, questionID := range evaluation.KnownQuestionIDs {
		knownQuestions[questionID] = true
	}
	for answerID := range value.Answers {
		if knownQuestions[answerID] {
			continue
		}
		value.ReconciliationChanges = append(value.ReconciliationChanges, sessiondomain.ReconciliationChange{
			Kind: "question-removed", Field: answerID, Before: compactRaw(value.Answers[answerID]), After: "removed",
			Impact: "The answer is retained for audit and cannot affect the current profile.", RequiresReview: true,
		})
	}
	for _, outstanding := range evaluation.Outstanding {
		if value.NextQuestionID == "" && knownQuestions[outstanding.Field] {
			value.NextQuestionID = outstanding.Field
		}
		if outstanding.Code == "unsupported" {
			value.ReconciliationChanges = append(value.ReconciliationChanges, sessiondomain.ReconciliationChange{
				Kind: "unsupported-recommendation", Field: outstanding.Field, After: outstanding.CapabilityRef,
				Impact: outstanding.Message, RequiresReview: true,
			})
		}
	}
	for _, issue := range evaluation.Issues {
		if value.NextQuestionID == "" && knownQuestions[issue.Field] {
			value.NextQuestionID = issue.Field
		}
	}
	if value.NextQuestionID != "" {
		value.NextAction = "answer-question"
	} else if len(evaluation.Outstanding) > 0 || len(evaluation.Issues) > 0 {
		value.NextAction = "review-recommendations"
	}
	for _, change := range value.ReconciliationChanges {
		if change.RequiresReview {
			value.ReconciliationReasons = append(value.ReconciliationReasons, change.Impact)
		}
	}
	if len(value.ReconciliationReasons) > 0 {
		value.ReconciliationState = "review_required"
	} else {
		value.ReconciliationState = "current"
	}
	if value.ReconciliationState == "review_required" && value.NextAction == "" {
		value.NextAction = "review-profile"
	}
	sort.Strings(value.ReconciliationReasons)
	sortReconciliationChanges(value.ReconciliationChanges)
	return value
}

func rawAnswerValues(values map[string]json.RawMessage) map[string]any {
	result := make(map[string]any, len(values))
	for key, raw := range values {
		var value any
		if json.Unmarshal(raw, &value) == nil {
			result[key] = value
		}
	}
	return result
}

func compactRaw(value json.RawMessage) string {
	var decoded any
	if json.Unmarshal(value, &decoded) != nil {
		return string(value)
	}
	encoded, err := json.Marshal(decoded)
	if err != nil {
		return string(value)
	}
	return string(encoded)
}

func diffTargetContext(before, after map[string]string) []sessiondomain.ReconciliationChange {
	keys := make(map[string]bool, len(before)+len(after))
	for key := range before {
		keys[key] = true
	}
	for key := range after {
		keys[key] = true
	}
	result := make([]sessiondomain.ReconciliationChange, 0)
	for key := range keys {
		if before[key] == after[key] {
			continue
		}
		result = append(result, sessiondomain.ReconciliationChange{
			Kind: "target-context", Field: key, Before: before[key], After: after[key],
			Impact: "The target context changed; review the resulting recommendations before applying.", RequiresReview: true,
		})
	}
	sortReconciliationChanges(result)
	return result
}

func diffProfileReferences(before *operatorstate.ProfileSession, after *sessiondomain.ProfileSession) []sessiondomain.ReconciliationChange {
	if before == nil || after == nil {
		return nil
	}
	result := make([]sessiondomain.ReconciliationChange, 0, 3)
	if before.ProfileID != after.ProfileID {
		result = append(result, sessiondomain.ReconciliationChange{
			Kind: "profile-selection", Field: "profileId", Before: before.ProfileID, After: after.ProfileID,
			Impact: "The selected profile changed; review its recommendations before applying.", RequiresReview: true,
		})
	}
	if before.CatalogRevision != after.CatalogRevision {
		result = append(result, sessiondomain.ReconciliationChange{
			Kind: "catalog-revision", Field: "catalogRevision", Before: before.CatalogRevision, After: after.CatalogRevision,
			Impact: "The capability catalog changed; review the resulting closure before applying.", RequiresReview: true,
		})
	}
	if before.Mode != after.Mode {
		result = append(result, sessiondomain.ReconciliationChange{
			Kind: "mode", Field: "mode", Before: before.Mode, After: after.Mode,
			Impact: "The session changed between guided and manual selection; explicit manual decisions remain preserved.", RequiresReview: false,
		})
	}
	sortReconciliationChanges(result)
	return result
}

func sortReconciliationChanges(values []sessiondomain.ReconciliationChange) {
	sort.SliceStable(values, func(i, j int) bool {
		if values[i].Kind != values[j].Kind {
			return values[i].Kind < values[j].Kind
		}
		if values[i].Field != values[j].Field {
			return values[i].Field < values[j].Field
		}
		return values[i].After < values[j].After
	})
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
