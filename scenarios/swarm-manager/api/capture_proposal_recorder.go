package main

import (
	"context"
	"fmt"
	"strings"

	"swarm-manager/internal/agentsessions"
)

type captureProposalRecorder struct{ sessions *agentsessions.Service }

func (r captureProposalRecorder) RecordCaptureProposals(ctx context.Context, captureID, title, summary, baseVersion, executionID, runID string, payload []byte) error {
	if r.sessions == nil {
		return fmt.Errorf("agent session service is not configured")
	}
	title = strings.TrimSpace(title)
	if title == "" {
		title = captureID
	}
	summary = strings.TrimSpace(summary)
	if summary == "" {
		summary = fmt.Sprintf("Capture classification produced reviewable proposals for %s.", title)
	}
	runID = strings.TrimSpace(runID)
	if runID == "" {
		runID = strings.TrimSpace(executionID)
	}
	_, _, err := r.sessions.RecordWorkflowMutationProposals(ctx, "Capture: "+title, summary, baseVersion, agentsessions.ProposalTarget{
		Type: agentsessions.ContextCapture,
		Ref:  captureID,
		Name: title,
	}, []string{string(payload)}, agentsessions.Attribution{
		Type:   agentsessions.AttributionAgent,
		RunID:  runID,
		Source: "capture/" + captureID,
	})
	return err
}
