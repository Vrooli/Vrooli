package handlers

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"path"
	"slices"
	"strings"

	"agent-manager/internal/domain"

	"github.com/google/uuid"
	requestidentity "github.com/vrooli/api-core/identity"
	"github.com/vrooli/cli-core/cliutil"
	pb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/domain"
)

// EffortEnrollmentReader reads one effort's enrollment. The quiesce guard uses
// it to learn which scenario an orchestrator's effort targets.
//
// seam: handlers.EffortEnrollmentReader
type EffortEnrollmentReader interface {
	EffortEnrollment(ctx context.Context, ref string) (*pb.EffortEnrollment, error)
}

// WithEffortEnrollments installs the effort enrollment reader used to admit an
// effort orchestrator's quiesce of the scenario its effort targets.
func WithEffortEnrollments(reader EffortEnrollmentReader) HandlerOption {
	return func(h *Handler) {
		h.effortEnrollments = reader
	}
}

const quiesceRunAuthorityHint = "forced quiesce cancels other agents' runs, so a run may call it only as the running root orchestrator " +
	"(profile scope " + OrchestrateScope + ") of an effort whose destination is that scenario; otherwise ask the operator or the coordinating session"

// denyRunInitiatedQuiesce lets verified callers use the default read/wait drain
// while retaining operator and effort-orchestrator checks for forced cancel.
// A presented but unverified run credential is refused, never treated as
// absence. A refusal names the failed condition.
func (h *Handler) denyRunInitiatedQuiesce(w http.ResponseWriter, r *http.Request, scenario string, force bool) bool {
	token := strings.TrimSpace(r.Header.Get(cliutil.HeaderAgentIdentityToken))
	if token == "" {
		principal, present := requestidentity.PrincipalFromContext(r.Context())
		if present && !principal.IsVerified() {
			writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "verified caller required"})
			return true
		}
		// Human operator requests retain the existing no-run-token path. An
		// agent/service principal cannot turn that absence into forced cancellation.
		if force && present && principal.Kind != requestidentity.ActorHuman {
			writeJSON(w, http.StatusForbidden, map[string]string{"error": "forced quiesce requires the running effort orchestrator"})
			return true
		}
		return false
	}
	verified, err := h.svc.VerifyIdentityToken(r.Context(), token)
	if err != nil {
		writeError(w, r, err)
		return true
	}
	if verified == nil || !verified.Valid || verified.Claims == nil {
		writeJSON(w, http.StatusUnauthorized, map[string]string{
			"error": "run identity is invalid or unavailable",
		})
		return true
	}
	// Non-forced quiesce is the reviewed read/wait path. Verifying a presented
	// run credential is still mandatory; only Force=true reaches the existing
	// effort-orchestrator authorization below.
	if !force {
		return false
	}
	runID := verified.Claims.RunID
	cause := h.effortQuiesceRefusal(r.Context(), runID, scenario)
	if cause == "" {
		log.Printf("agent-manager quiesce admitted effort orchestrator run_id=%s scenario=%s", runID, scenario)
		return false
	}
	recordLifecycleRefusal("quiesce")
	log.Printf("agent-manager lifecycle refusal caller=%s run_id=%s operation=quiesce cause=%q", r.RemoteAddr, runID, cause)
	writeJSON(w, http.StatusForbidden, map[string]any{
		"error":         fmt.Sprintf("run %s may not quiesce %q: %s", runID, scenario, cause),
		"operation":     "quiesce",
		"run_id":        runID.String(),
		"run_status":    string(verified.RunStatus),
		"recovery_hint": quiesceRunAuthorityHint,
	})
	return true
}

// effortQuiesceRefusal returns why a run may not quiesce scenario, or "" when it
// is a running root run whose profile declares OrchestrateScope and which holds
// an active orchestrator reference to an enrolled effort targeting scenario.
// Requiring a root run keeps a child from claiming an orchestrator reference it
// declared for itself.
func (h *Handler) effortQuiesceRefusal(ctx context.Context, callerID uuid.UUID, scenario string) string {
	if h.svc.RunService == nil || h.svc.ProfileService == nil {
		return "agent-manager cannot resolve run authority"
	}
	caller, err := h.svc.GetRun(ctx, callerID)
	if err != nil || caller == nil {
		return "its run record is unavailable"
	}
	if caller.Status != domain.RunStatusRunning {
		return fmt.Sprintf("it is %s, not running", caller.Status)
	}
	if caller.ParentRunID != nil {
		return "it is a child run; only an effort's root orchestrator may drain a scenario"
	}
	if caller.AgentProfileID == nil {
		return "it has no agent profile declaring " + OrchestrateScope
	}
	profile, err := h.svc.GetProfile(ctx, *caller.AgentProfileID)
	if err != nil || profile == nil || !slices.Contains(profile.DeclaredScopes, OrchestrateScope) {
		return "its profile does not declare " + OrchestrateScope
	}
	refs := orchestratorEffortRefs(caller)
	if len(refs) == 0 {
		return "it holds no active orchestrator effort reference"
	}
	if h.effortEnrollments == nil {
		return "effort enrollments are unavailable"
	}
	findings := make([]string, 0, len(refs))
	for _, ref := range refs {
		enrollment, err := h.effortEnrollments.EffortEnrollment(ctx, ref)
		switch {
		case err != nil || enrollment == nil:
			findings = append(findings, ref+" is not enrolled")
		case enrollment.GetWithdrawn():
			findings = append(findings, ref+" is withdrawn")
		default:
			target := effortDestinationScenario(enrollment.GetDestinationRef())
			if target == scenario {
				return ""
			}
			if target == "" {
				findings = append(findings, fmt.Sprintf("%s destination %q names no scenario", ref, enrollment.GetDestinationRef()))
			} else {
				findings = append(findings, ref+" targets "+target)
			}
		}
	}
	return "its effort does not target this scenario (" + strings.Join(findings, "; ") + ")"
}

// orchestratorEffortRefs lists the efforts a run holds as an active, verified
// orchestrator reference.
func orchestratorEffortRefs(run *domain.Run) []string {
	return effortRefsWithRelationship(run, "orchestrator")
}

// effortDestinationScenario derives the scenario an effort destination lies in.
// It accepts the destination forms enrollments use: "scenario:<name>" and a
// repository-relative path under scenarios/<name>, bare or with a path:, repo:
// or doc: prefix, optionally with a #fragment. Anything else names no scenario.
func effortDestinationScenario(destination string) string {
	destination = strings.TrimSpace(destination)
	destination, _, _ = strings.Cut(destination, "#")
	if name, ok := strings.CutPrefix(destination, "scenario:"); ok {
		name, _, _ = strings.Cut(name, "/")
		name, _, _ = strings.Cut(name, "@")
		return strings.TrimSpace(name)
	}
	for _, scheme := range []string{"path:", "repo:", "doc:"} {
		if rest, ok := strings.CutPrefix(destination, scheme); ok {
			destination = rest
			break
		}
	}
	if strings.HasPrefix(destination, "/") {
		return ""
	}
	// Clean against a root so "scenarios/x/../../y" cannot escape the prefix.
	rest, ok := strings.CutPrefix(path.Clean("/"+destination), "/scenarios/")
	if !ok {
		return ""
	}
	name, _, _ := strings.Cut(rest, "/")
	return name
}
