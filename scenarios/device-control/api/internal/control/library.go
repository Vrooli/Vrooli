package control

import (
	"context"
	"fmt"
	"reflect"
	"strings"

	internalflows "device-control/internal/flows"

	"github.com/google/uuid"
)

func (s *Service) ListSavedFlows(ctx context.Context, device, cohort string) ([]internalflows.SavedFlow, error) {
	if s.library == nil {
		return nil, fmt.Errorf("durable flow library unavailable")
	}
	if strings.TrimSpace(device) == "" || strings.TrimSpace(cohort) == "" {
		return nil, fmt.Errorf("device and context required")
	}
	return s.library.List(ctx, device, cohort)
}

func (s *Service) GetSavedFlow(ctx context.Context, id string, version int32) (internalflows.SavedFlow, error) {
	if s.library == nil {
		return internalflows.SavedFlow{}, fmt.Errorf("durable flow library unavailable")
	}
	if id == "" || version < 0 {
		return internalflows.SavedFlow{}, fmt.Errorf("invalid flow selector")
	}
	return s.library.Get(ctx, id, version)
}

func (s *Service) SaveValidatedFlow(ctx context.Context, runID, device, cohort, id string, expected int32) (internalflows.SavedFlow, error) {
	if s.library == nil {
		return internalflows.SavedFlow{}, fmt.Errorf("durable flow library unavailable")
	}
	if prior, found, err := s.library.FindSource(ctx, runID, device, cohort); err != nil {
		return internalflows.SavedFlow{}, err
	} else if found {
		if (id == "" && expected == 0 && prior.Version == 1) || (id == prior.ID && expected+1 == prior.Version) {
			return prior, nil
		}
		return internalflows.SavedFlow{}, fmt.Errorf("source run already promoted with different revision intent")
	}
	s.mu.Lock()
	result, ok := s.runs[runID]
	flow, found := s.flowRuns[runID]
	actualDevice := s.runDevices[runID]
	s.mu.Unlock()
	if !ok || !found || actualDevice != device || result.Disposition != "passed" || result.Incomplete {
		return internalflows.SavedFlow{}, fmt.Errorf("promotion requires a passing complete run on the exact device")
	}
	if err := ValidatePromotionCandidate(flow); err != nil {
		return internalflows.SavedFlow{}, err
	}
	if len(result.Chapters) != len(flow.Steps) {
		return internalflows.SavedFlow{}, fmt.Errorf("promotion requires evidence for every step")
	}
	receipt := internalflows.ValidationReceipt{Disposition: "passed"}
	for i, ch := range result.Chapters {
		if ch.ID != flow.Steps[i].ID {
			return internalflows.SavedFlow{}, fmt.Errorf("chapter identity differs from candidate")
		}
		receipt.StepIDs = append(receipt.StepIDs, ch.ID)
		if ch.Disposition != "passed" {
			return internalflows.SavedFlow{}, fmt.Errorf("promotion requires all chapters to pass")
		}
	}
	for _, resolution := range result.Resolutions {
		if resolution.Rung == "vision" {
			return internalflows.SavedFlow{}, fmt.Errorf("vision-resolved run requires deterministic targeting before promotion")
		}
	}
	if id == "" {
		if expected != 0 {
			return internalflows.SavedFlow{}, fmt.Errorf("new flow requires expected version zero")
		}
		id = uuid.NewString()
	} else {
		if expected < 1 {
			return internalflows.SavedFlow{}, fmt.Errorf("repair requires expected version")
		}
		prior, err := s.library.Get(ctx, id, expected)
		if err != nil {
			return internalflows.SavedFlow{}, err
		}
		if prior.DeviceID != device || prior.ContextKey != cohort {
			return internalflows.SavedFlow{}, fmt.Errorf("repair context differs")
		}
		if err := preserveFlowChecks(prior.Flow, flow); err != nil {
			return internalflows.SavedFlow{}, err
		}
	}
	return s.library.Save(ctx, internalflows.SavedFlow{ID: id, DeviceID: device, ContextKey: cohort, SourceRunID: runID, Flow: flow, Receipt: receipt}, expected)
}

func preserveFlowChecks(before, after Flow) error {
	if before.RequireUnlocked && !after.RequireUnlocked || before.AuthProfileID != after.AuthProfileID || before.AllowUnredactedCapture != after.AllowUnredactedCapture || before.Transport != after.Transport {
		return fmt.Errorf("repair cannot change authentication, redaction, or transport policy")
	}
	for _, check := range before.Steps {
		if strings.Contains(check.Kind, "assert") {
			found := false
			for _, step := range after.Steps {
				if step.ID == check.ID && reflect.DeepEqual(step, check) {
					found = true
				}
			}
			if !found {
				return fmt.Errorf("repair must preserve assertion %s", check.ID)
			}
		}
	}
	return nil
}

func (s *Service) RunSavedFlow(ctx context.Context, id string, version int32, device, cohort, actor string) (RunResult, error) {
	if version < 1 {
		return RunResult{}, fmt.Errorf("exact saved version required")
	}
	f, err := s.GetSavedFlow(ctx, id, version)
	if err != nil {
		return RunResult{}, err
	}
	if f.DeviceID != device || f.ContextKey != cohort {
		return RunResult{}, fmt.Errorf("saved flow device/context mismatch")
	}
	return s.Run(ctx, f.Flow, device, actor)
}

func (s *Service) ValidateRepair(ctx context.Context, id string, version int32, candidate Flow) error {
	if version < 1 {
		return fmt.Errorf("repair requires exact expected version")
	}
	baseline, err := s.GetSavedFlow(ctx, id, version)
	if err != nil {
		return err
	}
	current, err := s.GetSavedFlow(ctx, id, 0)
	if err != nil {
		return err
	}
	if current.Version != version {
		return fmt.Errorf("flow version conflict")
	}
	return preserveFlowChecks(baseline.Flow, candidate)
}

// ValidatePromotionCandidate defines the acceptance contract shared by preflight and persistence.
func ValidatePromotionCandidate(flow Flow) error {
	if flow.SuppressActuation {
		return fmt.Errorf("dry-run cannot be promoted")
	}
	if len(flow.Steps) == 0 {
		return fmt.Errorf("empty flow cannot be promoted")
	}
	switch flow.Steps[len(flow.Steps)-1].Kind {
	case "semantic-assert", "assert-frame-different", "property-assert":
		return nil
	default:
		return fmt.Errorf("promotion requires a terminal outcome assertion")
	}
}
