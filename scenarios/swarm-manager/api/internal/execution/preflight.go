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

// ProcessPreflight evaluates whether a backlog item is ready for processing.
func (s *Service) ProcessPreflight(ctx context.Context, backlogKind, backlogName string) (ProcessPreflight, error) {
	item, err := s.loadBacklogItem(backlogKind, backlogName)
	if err != nil {
		return ProcessPreflight{}, err
	}
	return s.processPreflightForItem(ctx, item, true), nil
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
		preflight.BlockingReasons = append(preflight.BlockingReasons, fmt.Sprintf("backlog item cannot be queued from current status: %s", item.Status))
	}

	// Execution requires an explicit acceptance of the current canonical plan.
	// Workshop artifacts remain historical context only; score thresholds are
	// deliberately not a release gate.
	hasDeliverable := hasExecutionPlanRef(item)
	if !hasDeliverable {
		preflight.BlockingReasons = append(preflight.BlockingReasons, missingDeliverableReason(item.Kind, ""))
	}
	if hasDeliverable {
		acceptanceReason := s.planAcceptanceBlockingReason(ctx, item)
		if acceptanceReason != "" {
			preflight.BlockingReasons = append(preflight.BlockingReasons, acceptanceReason)
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

func (s *Service) planAcceptanceBlockingReason(ctx context.Context, item backlogItem) string {
	if item.PlanAcceptance == nil {
		return "canonical plan has not been explicitly accepted — accept the current plan revision before queueing"
	}
	// Production wiring always supplies the Plan Manager renderer. The nil
	// seam is retained for narrow domain tests and embedded callers that can
	// verify stored acceptance but do not own a Plan Manager connection.
	if s.planRenderer == nil {
		return ""
	}
	rendered, err := resolveRenderedPlanContent(ctx, item, s.planRenderer)
	if err != nil {
		return fmt.Sprintf("canonical plan validation unavailable: %s", err)
	}
	if strings.TrimSpace(rendered.QualityStatus) != "pass" {
		return fmt.Sprintf("canonical plan is not valid: quality status is %q", rendered.QualityStatus)
	}
	if rendered.Status == "PLAN_STATUS_DRAFT" || rendered.Status == "PLAN_STATUS_ARCHIVED" {
		return fmt.Sprintf("canonical plan is not executable in status %q", rendered.Status)
	}
	if strings.TrimSpace(rendered.ContentHash) == "" {
		return "canonical plan validation unavailable: plan-manager returned no content hash"
	}
	if strings.TrimSpace(item.PlanAcceptance.PlanContentHash) != strings.TrimSpace(rendered.ContentHash) {
		return "canonical plan changed after acceptance — accept the current revision before queueing"
	}
	if strings.TrimSpace(item.PlanAcceptance.SubjectVersion) != executionPlanAcceptanceSubjectVersion(item) {
		return "work contract changed after plan acceptance — accept the current revision before queueing"
	}
	return ""
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
