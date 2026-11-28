package control

import (
	"context"
	"errors"
	"fmt"

	internalflows "device-control/internal/flows"
)

type FlowExport struct {
	RunID    string            `json:"run_id"`
	Flow     Flow              `json:"flow"`
	Rungs    map[string]string `json:"rungs"`
	Excluded []string          `json:"excluded_steps,omitempty"`
}

func (s *Service) ExportFlow(id string) (FlowExport, error) {
	s.mu.Lock()
	result, ok := s.runs[id]
	flow, flowOK := s.flowRuns[id]
	s.mu.Unlock()
	if (!ok || !flowOK) && s.retainedRuns != nil {
		retained, err := s.retainedRuns.Get(context.Background(), id)
		if err == nil {
			result, flow, ok, flowOK = retained.Result, retained.Flow, true, true
		} else if !errors.Is(err, internalflows.ErrInvalidRequest) {
			return FlowExport{}, fmt.Errorf("read retained run: %w", err)
		}
	}
	if !ok || !flowOK {
		return FlowExport{}, fmt.Errorf("run %q not found: %w", id, internalflows.ErrInvalidRequest)
	}
	export := FlowExport{RunID: id, Flow: flow, Rungs: map[string]string{}, Excluded: []string{}}
	failed := map[string]bool{}
	for _, chapter := range result.Chapters {
		if chapter.Disposition == "failed" || chapter.Disposition == "cancelled" || chapter.Disposition == "device_disconnected" {
			failed[chapter.ID] = true
			export.Excluded = append(export.Excluded, chapter.ID+": "+chapter.Message)
		}
	}
	steps := make([]Step, 0, len(flow.Steps))
	for _, step := range flow.Steps {
		if !failed[step.ID] {
			steps = append(steps, step)
		}
	}
	export.Flow.Steps = steps
	for _, resolution := range result.Resolutions {
		export.Rungs[resolution.Target] = resolution.Rung
	}
	return export, nil
}
