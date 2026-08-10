package execution

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// PreflightSpec is the caller-supplied form of a backlog item's spec. It
// exists for callers that already hold the item loaded — a list projection
// reads every spec once, and re-reading each one to answer readiness is pure
// duplicate IO. Every field readiness consults is present, so a spec built
// from an already-loaded item produces the same verdict as one read from disk.
type PreflightSpec struct {
	Kind               string
	Name               string
	Title              string
	Description        string
	Status             string
	SourceScenarioName string
	AcceptanceAllow    []string
	AcceptanceDeny     []string
	Creates            []string
	ArchivedAt         *string
	PlanRef            *PlanRefSpec
	PlanAcceptance     *PlanAcceptanceSpec
}

// PreflightPlanRef mirrors the plan_ref block of a backlog spec.
type PlanRefSpec struct {
	Provider string
	PlanID   string
	Slug     string
	Role     string
}

// PlanAcceptanceSpec mirrors the plan_acceptance block of a backlog spec.
type PlanAcceptanceSpec struct {
	Actor           string
	AcceptedAt      string
	PlanContentHash string
	SubjectVersion  string
}

// ProcessPreflight evaluates whether a backlog item is ready for processing,
// reading the item's spec from disk.
func (s *Service) ProcessPreflight(ctx context.Context, backlogKind, backlogName string) (ProcessPreflight, error) {
	item, err := s.loadBacklogItem(backlogKind, backlogName)
	if err != nil {
		return ProcessPreflight{}, err
	}
	return s.processPreflightForItem(ctx, item, true), nil
}

// ProcessPreflightForSpec evaluates readiness from an already-loaded spec.
func (s *Service) ProcessPreflightForSpec(ctx context.Context, spec PreflightSpec) ProcessPreflight {
	return s.processPreflightForItem(ctx, spec.toBacklogItem(), true)
}

func (spec PreflightSpec) toBacklogItem() backlogItem {
	item := backlogItem{
		Name:               strings.TrimSpace(spec.Name),
		Title:              spec.Title,
		Description:        spec.Description,
		Status:             spec.Status,
		Kind:               strings.ToLower(strings.TrimSpace(spec.Kind)),
		SourceScenarioName: spec.SourceScenarioName,
		AcceptanceAllow:    spec.AcceptanceAllow,
		AcceptanceDeny:     spec.AcceptanceDeny,
		Creates:            spec.Creates,
		ArchivedAt:         spec.ArchivedAt,
		Tags:               []string{},
	}
	if spec.PlanRef != nil {
		item.PlanRef = &planRef{Provider: spec.PlanRef.Provider, PlanID: spec.PlanRef.PlanID, Slug: spec.PlanRef.Slug, Role: spec.PlanRef.Role}
	}
	if spec.PlanAcceptance != nil {
		item.PlanAcceptance = &planAcceptance{
			Actor:           spec.PlanAcceptance.Actor,
			AcceptedAt:      spec.PlanAcceptance.AcceptedAt,
			PlanContentHash: spec.PlanAcceptance.PlanContentHash,
			SubjectVersion:  spec.PlanAcceptance.SubjectVersion,
		}
	}
	return item
}

func (s *Service) processPreflightForItem(ctx context.Context, item backlogItem, checkQueueable bool) ProcessPreflight {
	targetScenarioID, archivedRevival := resolveTargetScenario(item)
	targetScenarioExists := false
	if strings.TrimSpace(targetScenarioID) != "" {
		targetScenarioExists = scenarioExists(filepath.Join(s.scenariosRootDir(), targetScenarioID))
	}

	preflight := ProcessPreflight{
		BacklogKind:              strings.TrimSpace(item.Kind),
		BacklogName:              strings.TrimSpace(item.Name),
		Ready:                    true,
		ArchivedRevival:          archivedRevival,
		ResolvedTargetScenarioID: targetScenarioID,
		TargetScenarioExists:     targetScenarioExists,
		SuggestedOperation:       "generator",
		SuggestedSteerProfileID:  "rapid-mvp",
	}
	if targetScenarioExists {
		preflight.SuggestedOperation = "improver"
		preflight.SuggestedSteerProfileID = "production-ready"
	}

	isArchived := item.ArchivedAt != nil
	if checkQueueable && !isQueueableStatus(item.Kind, item.Status) && !(isArchived && strings.ToLower(strings.TrimSpace(item.Kind)) == "idea") {
		appendPreflightBlocker(&preflight, "circuit_open", fmt.Sprintf("backlog item cannot be queued from current status: %s", item.Status), false)
	}

	// Execution requires an explicit acceptance of the current canonical plan.
	// Workshop artifacts remain historical context only; score thresholds are
	// deliberately not a release gate.
	hasDeliverable := hasExecutionPlanRef(item)
	if !hasDeliverable {
		appendPreflightBlocker(&preflight, "plan_invalid", missingDeliverableReason(), false)
	}
	if hasDeliverable {
		acceptanceBlocker := s.planAcceptanceBlockingReason(ctx, item)
		if acceptanceBlocker.Message != "" {
			appendPreflightBlocker(&preflight, acceptanceBlocker.Code, acceptanceBlocker.Message, false)
		}
	}

	// Fix-before-feature gate applies only at queue time (checkQueueable),
	// never on start/retry/followup of an already-running item.
	if checkQueueable {
		s.applyFixBeforeFeatureGate(item, &preflight)
	}

	preflight.Ready = len(preflight.BlockingReasons) == 0 && len(preflight.ForceableBlockingReasons) == 0
	return preflight
}

func appendPreflightBlocker(preflight *ProcessPreflight, code, message string, forceable bool) {
	if forceable {
		preflight.ForceableBlockingReasons = append(preflight.ForceableBlockingReasons, message)
		preflight.ForceableBlockingDetails = append(preflight.ForceableBlockingDetails, ProcessBlockingReason{Code: code, Message: message})
		return
	}
	preflight.BlockingReasons = append(preflight.BlockingReasons, message)
	preflight.BlockingDetails = append(preflight.BlockingDetails, ProcessBlockingReason{Code: code, Message: message})
}

func (s *Service) planAcceptanceBlockingReason(ctx context.Context, item backlogItem) ProcessBlockingReason {
	if item.PlanAcceptance == nil {
		return ProcessBlockingReason{Code: "plan_not_accepted", Message: "canonical plan has not been explicitly accepted — accept the current plan revision before queueing"}
	}
	// Production wiring always supplies the Plan Manager renderer. The nil
	// seam is retained for narrow domain tests and embedded callers that can
	// verify stored acceptance but do not own a Plan Manager connection.
	if s.planRenderer == nil {
		return ProcessBlockingReason{}
	}
	rendered, err := resolveRenderedPlanContent(ctx, item, s.planRenderer)
	if err != nil {
		return ProcessBlockingReason{Code: "plan_invalid", Message: fmt.Sprintf("canonical plan validation unavailable: %s", err)}
	}
	if strings.TrimSpace(rendered.QualityStatus) != "pass" {
		return ProcessBlockingReason{Code: "plan_invalid", Message: fmt.Sprintf("canonical plan is not valid: quality status is %q", rendered.QualityStatus)}
	}
	if rendered.Status == "PLAN_STATUS_DRAFT" || rendered.Status == "PLAN_STATUS_ARCHIVED" {
		return ProcessBlockingReason{Code: "plan_invalid", Message: fmt.Sprintf("canonical plan is not executable in status %q", rendered.Status)}
	}
	if strings.TrimSpace(rendered.ContentHash) == "" {
		return ProcessBlockingReason{Code: "plan_invalid", Message: "canonical plan validation unavailable: plan-manager returned no content hash"}
	}
	if strings.TrimSpace(item.PlanAcceptance.PlanContentHash) != strings.TrimSpace(rendered.ContentHash) {
		return ProcessBlockingReason{Code: "plan_changed", Message: "canonical plan changed after acceptance — accept the current revision before queueing"}
	}
	if strings.TrimSpace(item.PlanAcceptance.SubjectVersion) != executionPlanAcceptanceSubjectVersion(item) {
		return ProcessBlockingReason{Code: "plan_changed", Message: "work contract changed after plan acceptance — accept the current revision before queueing"}
	}
	return ProcessBlockingReason{}
}

func executionPlanAcceptanceSubjectVersion(item backlogItem) string {
	payload := struct {
		Kind            string   `json:"kind"`
		Name            string   `json:"name"`
		Title           string   `json:"title"`
		Description     string   `json:"description"`
		AcceptanceAllow []string `json:"acceptance_allow,omitempty"`
		AcceptanceDeny  []string `json:"acceptance_deny,omitempty"`
		Creates         []string `json:"creates,omitempty"`
		PlanRef         *planRef `json:"plan_ref,omitempty"`
	}{
		Kind: item.Kind, Name: item.Name, Title: item.Title, Description: item.Description,
		AcceptanceAllow: item.AcceptanceAllow, AcceptanceDeny: item.AcceptanceDeny,
		Creates: item.Creates, PlanRef: item.PlanRef,
	}
	raw, _ := json.Marshal(payload)
	sum := sha256.Sum256(raw)
	return "sha256:" + hex.EncodeToString(sum[:])
}

func hasExecutionPlanRef(item backlogItem) bool {
	if item.PlanRef == nil {
		return false
	}
	return strings.TrimSpace(item.PlanRef.Provider) == planRefProviderPlanManager &&
		strings.TrimSpace(item.PlanRef.Role) == planRefRoleExecutionSpec &&
		(strings.TrimSpace(item.PlanRef.PlanID) != "" || strings.TrimSpace(item.PlanRef.Slug) != "")
}

func resolveTargetScenario(item backlogItem) (string, bool) {
	source := strings.TrimSpace(item.SourceScenarioName)
	if source != "" {
		return source, true
	}
	return strings.TrimSpace(item.Name), item.ArchivedAt != nil
}

// allBlockingReasons returns every reason that makes a preflight not-ready —
// both structural (non-forceable) and forceable — for display in error
// messages. The Ready flag already accounts for both slices.
func allBlockingReasons(preflight ProcessPreflight) []string {
	if len(preflight.ForceableBlockingReasons) == 0 {
		return preflight.BlockingReasons
	}
	combined := make([]string, 0, len(preflight.BlockingReasons)+len(preflight.ForceableBlockingReasons))
	combined = append(combined, preflight.BlockingReasons...)
	combined = append(combined, preflight.ForceableBlockingReasons...)
	return combined
}

func hasNonForceableExecutionReasons(reasons []string) bool {
	return len(reasons) > 0
}

func scenarioExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}
