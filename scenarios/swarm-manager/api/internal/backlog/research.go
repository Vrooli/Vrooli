// Research operations for backlog items: the research/finalize HTTP entrypoint
// that starts a refine or finalize operation through the declarative operation
// runner (see ops_reroute.go). Prompt construction now lives in the operating
// mode the runner binds, not here.
package backlog

import (
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"time"

	"swarm-manager/internal/agentops"
	"swarm-manager/internal/apierr"
	"swarm-manager/internal/httputil"

	apipb "github.com/vrooli/vrooli/packages/proto/gen/go/swarm-manager/v1/api"
)

// parseResearchMode normalizes a raw mode string into a ResearchMode constant.
// Returns an error for unrecognized values.
func parseResearchMode(raw string) (ResearchMode, error) {
	candidate := strings.ToLower(strings.TrimSpace(raw))
	switch candidate {
	case "workshop", "":
		return ResearchModeWorkshop, nil
	case "finalize":
		return ResearchModeFinalize, nil
	case "initialize":
		return ResearchModeInitialize, nil
	// Capture-related modes (clarify, suggest, enhance) are treated as workshop.
	case "clarify", "suggest", "enhance":
		return ResearchModeWorkshop, nil
	default:
		return "", fmt.Errorf("unsupported research mode %q: must be workshop, finalize, or initialize", candidate)
	}
}

// validateResearchModeForKind checks whether a mode is valid for a given kind.
func validateResearchModeForKind(_ BacklogKind, _ ResearchMode) error {
	// All modes are valid for all kinds.
	return nil
}

// normalizeResearchRequest trims and lowercases optional fields on a research
// request, clearing empty strings to nil.
func normalizeResearchRequest(req *apipb.BacklogResearchRequest) {
	if req == nil {
		return
	}
	if req.Mode != nil {
		trimmed := strings.ToLower(strings.TrimSpace(*req.Mode))
		if trimmed == "" {
			req.Mode = nil
		} else {
			req.Mode = &trimmed
		}
	}
}

// readOptionalString dereferences a string pointer, returning "" for nil.
func readOptionalString(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

// parseResearchRequestBody decodes and validates the research request body,
// parses the research mode, and validates mode/kind compatibility. Returns
// the parsed request, resolved mode, and ok=true on success. On failure it
// writes the appropriate error response and returns ok=false. Extracting this
// collapses five branches (body decode, normalize, validate, mode parse,
// mode/kind check) into one branch in Research.
func (h *Handler) parseResearchRequestBody(w http.ResponseWriter, r *http.Request, kind BacklogKind) (*apipb.BacklogResearchRequest, ResearchMode, bool) {
	req := &apipb.BacklogResearchRequest{}
	if r.Body != nil && r.ContentLength != 0 {
		if err := httputil.DecodeProtoJSON(r, req); err != nil {
			apierr.MapError(w, "[backlog] research", apierr.BadRequest("invalid request body"))
			return nil, "", false
		}
		normalizeResearchRequest(req)
		if !httputil.ValidateProtoRequest(w, "[backlog] research", "invalid request body", req) {
			return nil, "", false
		}
	}
	mode, modeErr := parseResearchMode(readOptionalString(req.Mode))
	if modeErr != nil {
		apierr.MapError(w, "[backlog] research", apierr.BadRequest("%s", modeErr.Error()))
		return nil, "", false
	}
	if err := validateResearchModeForKind(kind, mode); err != nil {
		apierr.MapError(w, "[backlog] research", apierr.BadRequest("%s", err.Error()))
		return nil, "", false
	}
	return req, mode, true
}

// cancelPendingAdvanceIfNeeded cancels any deferred auto-advance registered
// for this item when the user explicitly triggers a workshop or finalize run.
// Extracted from Research to remove two branches (mode check + ticker nil
// check) from the handler body.
func (h *Handler) cancelPendingAdvanceIfNeeded(kind BacklogKind, name, itemName string, mode ResearchMode) {
	if mode != ResearchModeWorkshop && mode != ResearchModeFinalize {
		return
	}
	if err := h.cancelDeferredAdvanceIntent(kind, itemName); err != nil {
		slog.Warn("research: cancel pending auto-advance failed", "kind", kind, "name", name, "mode", mode, "err", err)
	}
}

// researchHandleDependencyBlocking evaluates dependency blocking for a research
// request and, when the request must not proceed, writes the appropriate
// (blocked or dry-run) response and returns true. It returns false when the
// caller may continue spawning the research agent.
func (h *Handler) researchHandleDependencyBlocking(w http.ResponseWriter, item BacklogItem, kind BacklogKind, name string, confirm, force bool) bool {
	depReasons, depErr := EvaluateDependencyBlocking(item, h.store)
	if depErr != nil {
		slog.Error("research dependency check failed", "kind", kind, "name", name, "err", depErr)
		apierr.MapError(w, "[backlog] research", apierr.Internal("failed to check dependencies"))
		return true
	}
	if len(depReasons) == 0 {
		return false
	}

	protoReasons := make([]*apipb.BlockingReason, len(depReasons))
	for i, r := range depReasons {
		protoReasons[i] = &apipb.BlockingReason{Message: r.Message, Forceable: r.Forceable}
	}

	writeBlocked := func(message string) bool {
		resp := &apipb.BacklogResearchResponse{
			DryRun:          true,
			Started:         false,
			Message:         message,
			BlockingReasons: protoReasons,
		}
		if err := httputil.ProtoJSONWithStatus(w, http.StatusOK, resp); err != nil {
			apierr.MapError(w, "[backlog] research", apierr.Internal("failed to encode blocked response"))
		}
		return true
	}

	if !confirm {
		return writeBlocked("Research blocked by dependencies. Use confirm=true and force=true (CLI: --execute --force) to override.")
	}
	if !force || HasNonForceableReasons(depReasons) {
		return writeBlocked("Research blocked by dependencies.")
	}
	// force=true and all reasons are forceable — proceed.
	return false
}

// researchFinalizePrecheck enforces the finalize-mode preconditions. The bool
// is false when a precondition failed and an error response has already been
// written.
func (h *Handler) researchFinalizePrecheck(w http.ResponseWriter, item BacklogItem, kind BacklogKind, name string) (string, bool) {
	itemDir := h.store.ItemDir(kind, item.Name)
	latestRound, roundCount, loadErr := LoadLatestRound(itemDir)
	if loadErr != nil {
		apierr.MapError(w, "[backlog] research", apierr.Internal("failed to load workshop rounds for finalize"))
		return "", false
	}
	if latestRound == nil {
		apierr.MapError(w, "[backlog] research", apierr.Conflict("finalize requires at least one workshop round"))
		return "", false
	}
	if CountPendingDecisions(latestRound) > 0 {
		apierr.MapError(w, "[backlog] research", apierr.Conflict("finalize is only available after answering all workshop decisions"))
		return "", false
	}
	effective := ComputeEffectiveScores(latestRound.Readiness, roundCount, kind)
	if !IsReady(effective) {
		apierr.MapError(w, "[backlog] research", apierr.Conflict("finalize is only available when the latest workshop round is ready"))
		return "", false
	}
	if !NeedsSynthesis(latestRound) {
		apierr.MapError(w, "[backlog] research", apierr.Conflict("finalize is only available when the latest workshop answers have not been synthesized yet"))
		return "", false
	}

	return "", true
}

// writeResearchDryRun writes the dry-run research response (no agent spawned).
func writeResearchDryRun(w http.ResponseWriter) {
	resp := &apipb.BacklogResearchResponse{
		TaskId:  "dry-run-task",
		RunId:   "dry-run-run",
		BaseUrl: "",
		Created: time.Now().UTC().Format(time.RFC3339),
		DryRun:  true,
		Started: false,
		Message: "Dry run. No agent spawned.",
	}
	if err := httputil.ProtoJSONWithStatus(w, http.StatusOK, resp); err != nil {
		apierr.MapError(w, "[backlog] research", apierr.Internal("failed to encode dry-run response"))
	}
}

// Research starts a research/finalize operation for the specified backlog item.
func (h *Handler) Research(w http.ResponseWriter, r *http.Request) {
	kind, name, ok := h.parseKindAndName(w, r, "research")
	if !ok {
		return
	}

	item, err := h.store.LoadItem(kind, name)
	if err != nil {
		if os.IsNotExist(err) {
			apierr.MapError(w, "[backlog] research", apierr.NotFound("backlog item not found"))
			return
		}
		apierr.MapError(w, "[backlog] research", apierr.Internal("failed to load backlog item"))
		return
	}

	req, mode, ok2 := h.parseResearchRequestBody(w, r, kind)
	if !ok2 {
		return
	}

	confirm := req.GetConfirm()
	force := req.GetForce()

	// Dependency blocking applies to initialize and workshop modes.
	// Finalize, clarify, suggest, and enhance skip dep checks — once a
	// workshop has started or is being refined, it should complete
	// regardless of dependency state.
	if mode == ResearchModeInitialize || mode == ResearchModeWorkshop {
		if handled := h.researchHandleDependencyBlocking(w, item, kind, name, confirm, force); handled {
			return
		}
	}

	if mode == ResearchModeInitialize && item.Status != StatusBacklog {
		apierr.MapError(w, "[backlog] research", apierr.Conflict("initialize is only available for items in 'backlog' status"))
		return
	}
	// Pre-finalization precheck enforces the finalize preconditions (>=1 round,
	// all decisions answered, readiness reached, not yet synthesized). Its error
	// responses are written inside; the advisory gap report it used to feed the
	// prompt is now the operating mode's concern.
	if mode == ResearchModeFinalize {
		if _, ok := h.researchFinalizePrecheck(w, item, kind, name); !ok {
			return
		}
	}

	// Cancel any pending auto-advance for workshop/finalize modes so a deferred
	// advance does not race the operator's explicit run.
	h.cancelPendingAdvanceIfNeeded(kind, name, item.Name, mode)

	if httputil.IsDryRun(r) {
		writeResearchDryRun(w)
		return
	}
	if mode == ResearchModeWorkshop {
		handle, err := h.startWorkshopRoundWorkflow(r.Context(), item, workshopOperatorNote(req))
		if err != nil {
			mapResearchInvokeError(w, err)
			return
		}
		resp := &apipb.BacklogResearchResponse{
			TaskId: handle.ExecutionID, RunId: handle.RunID, Created: time.Now().UTC().Format(time.RFC3339),
			DryRun: false, Started: true, Message: "Backlog workshop workflow started: " + handle.ExecutionID,
		}
		if err := httputil.ProtoJSONWithStatus(w, http.StatusCreated, resp); err != nil {
			apierr.MapError(w, "[backlog] research", apierr.Internal("failed to encode response"))
		}
		return
	}

	// The bound operating mode owns prompt construction from the item folder; the
	// entrypoint forwards the operator's typed research context (prompt + attached
	// context) so the research-refine mode's caller-context providers can steer the
	// round. Finalize takes no research context (its contract declares only an
	// operator note), so caller inputs are gated to the refine operation.
	op := operationForResearchMode(mode)
	var callerInputs map[string]any
	if op == agentops.OpResearchRefine {
		callerInputs = researchRefineCallerInputs(req)
	}
	handle, err := h.invokeItemOperation(r.Context(), kind, item.Name, op, "", callerInputs)
	if err != nil {
		mapResearchInvokeError(w, err)
		return
	}

	resp := &apipb.BacklogResearchResponse{
		TaskId:  handle.TaskID,
		RunId:   handle.RunID,
		BaseUrl: handle.BaseURL,
		Created: handle.CreatedAt,
		DryRun:  false,
		Started: true,
		Message: "Research operation started.",
	}
	if err := httputil.ProtoJSONWithStatus(w, http.StatusCreated, resp); err != nil {
		apierr.MapError(w, "[backlog] research", apierr.Internal("failed to encode response"))
	}
}

// researchRefineCallerInputs maps the research request's operator-supplied context
// onto the research-refine operation's typed caller inputs. Only non-empty fields
// are included, so the pinned snapshot carries exactly what the operator supplied;
// a request with no context yields a nil map (an empty caller-input set, which the
// mode still accepts). The repeated context fields are joined one-per-line to match
// the CONTEXT_* providers' rendered shape.
//
// GAP_REPORT is intentionally NOT forwarded here: the readiness/gap report the
// legacy finalize prompt embedded is now the operating mode's concern, derived from
// the item folder rather than passed by the caller (see the Research handler), and
// there is no request field carrying one.
func researchRefineCallerInputs(req *apipb.BacklogResearchRequest) map[string]any {
	if req == nil {
		return nil
	}
	inputs := map[string]any{}
	putCallerString(inputs, "USER_PROMPT", readOptionalString(req.Prompt))
	putCallerString(inputs, "CONTEXT_PATHS", strings.Join(req.GetContextPaths(), "\n"))
	putCallerString(inputs, "CONTEXT_TARGETS", strings.Join(req.GetContextTargetIds(), "\n"))
	putCallerString(inputs, "CONTEXT_REQUIREMENTS", strings.Join(req.GetContextRequirementIds(), "\n"))
	if len(inputs) == 0 {
		return nil
	}
	return inputs
}

// mapResearchInvokeError classifies a runner Invoke error into the API error the
// legacy spawn path returned (unavailable / busy / internal).
func mapResearchInvokeError(w http.ResponseWriter, err error) {
	switch mapInvokeError(err).kind {
	case invokeUnavailable:
		apierr.MapError(w, "[backlog] research", apierr.Unavailable("agent-manager is not available"))
	case invokeBusy:
		apierr.MapError(w, "[backlog] research", apierr.Conflict("an agent is already active for this backlog item"))
	default:
		apierr.MapError(w, "[backlog] research", apierr.Internal("failed to start research operation"))
	}
}
