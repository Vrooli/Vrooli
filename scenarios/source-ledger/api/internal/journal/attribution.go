package journal

import (
	"strings"

	"github.com/vrooli/api-core/provenance"
)

// AttributionFrom projects api-core provenance onto an entry's durable
// attribution and correlation. Both write seams — the Connect handler and
// direct Service.Append callers — go through it, so an entry's provenance
// cannot depend on which door it came through.
//
// ActorID, RunID, and WorkflowExecutionID are taken from provenance
// unconditionally: a caller must not be able to forge an agent run, or
// silently omit one while working inside it. ActorKind and SourceRuntime are
// caller-first so internal writers (harness import) keep their own descriptive
// labels. Harness observation fields fill in only when the caller left them
// empty.
func AttributionFrom(source provenance.Provenance, existing Attribution) (Attribution, Correlation) {
	actorID, actorKind, sourceRuntime, status, runID, workflowExecutionID := source.WriteFields()
	harnessSessionID, harnessKind := source.ObservationFields()

	out := existing
	out.ActorID = actorID
	out.VerificationStatus = status
	if strings.TrimSpace(out.ActorKind) == "" {
		out.ActorKind = actorKind
	}
	if strings.TrimSpace(out.SourceRuntime) == "" {
		out.SourceRuntime = sourceRuntime
	}
	if strings.TrimSpace(out.HarnessSessionID) == "" {
		out.HarnessSessionID = harnessSessionID
		out.HarnessKind = harnessKind
	}
	return out, Correlation{
		RunID:               runID,
		WorkflowExecutionID: workflowExecutionID,
		ActorKind:           out.ActorKind,
	}
}
