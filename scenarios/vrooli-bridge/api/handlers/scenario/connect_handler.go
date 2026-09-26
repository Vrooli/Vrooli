package scenario

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strconv"
	"strings"

	"connectrpc.com/connect"
	"vrooli-bridge/internal/auth"
	internal "vrooli-bridge/internal/scenario"

	"github.com/gorilla/mux"
)

type Deps struct {
	Service internal.Service
	// Facts, when set, explains a contract failure in version terms. It is only
	// consulted on the failure path.
	Facts TargetFacts
}

// TargetFacts supplies what turns "the target is missing a procedure" into an
// actionable answer: which revision the target runs, which the control plane
// runs, and the one update path that can close the gap.
type TargetFacts interface {
	VersionFacts(ctx context.Context, nodeID, scenario string) VersionFacts
}

// VersionFacts is the version context attached to a contract failure. Empty
// fields are unknown and are omitted from the wire.
type VersionFacts struct {
	TargetRevision       string
	ControlPlaneRevision string
	// UpdatePath is "provision" when `provision sync` can update the node,
	// "reonboard" when only re-running onboarding can, or "restart" when the
	// node already has the control plane's revision and only the scenario's
	// running process predates it.
	UpdatePath    string
	UpdateCommand string
}

type handler struct{ deps Deps }

func NewHandler(deps Deps) *handler { return &handler{deps: deps} }

const (
	deviceControlScenario = "device-control"
	desktopSessionService = "vrooli.device_control.v1.desktop.DesktopSessionService"
)

var upstreamHTTPStatusPattern = regexp.MustCompile(`(?i)\bHTTP\s+(\d{3})\b`)

type scenarioProxyFailure struct {
	code           connect.Code
	classification string
	message        string
	retry          bool
	upstreamStatus int
	version        VersionFacts
}

// withVersionFacts explains a contract failure as the version skew it almost
// always is. A missing procedure on a reachable target means the target runs an
// older build than its caller, so the remedy is to update the target, not to
// retry or re-apply.
func (f scenarioProxyFailure) withVersionFacts(facts VersionFacts) scenarioProxyFailure {
	if f.classification != "target_incompatible" && f.classification != "contract_mismatch" {
		return f
	}
	f.version = facts
	switch {
	case facts.UpdatePath == "restart":
		// Same revision on both sides: the source is not behind, the running
		// process is. "Update the target" would send the operator to re-ship a
		// tree that is already there.
		f.message += fmt.Sprintf("; the target already has revision %s but its running scenario has not restarted onto it", facts.TargetRevision)
	case facts.TargetRevision != "" && facts.ControlPlaneRevision != "" && facts.TargetRevision != facts.ControlPlaneRevision:
		f.message += fmt.Sprintf("; the target runs revision %s and the control plane runs %s", facts.TargetRevision, facts.ControlPlaneRevision)
	}
	if facts.UpdateCommand != "" {
		f.message += "; fix it with `" + facts.UpdateCommand + "`"
	}
	return f
}

// classifyScenarioProxyFailure turns the node agent's bounded response into a
// stable Connect error. The agent intentionally returns an opaque error string
// because the scenario response body is otherwise raw protobuf; the Bridge is
// the first owner that has enough context to classify the failure for callers.
func classifyScenarioProxyFailure(err error, scenario, procedure string) scenarioProxyFailure {
	message := strings.TrimSpace(err.Error())
	failure := scenarioProxyFailure{
		code:           connect.CodeUnavailable,
		classification: "target_unreachable",
		message:        fmt.Sprintf("target scenario %q could not answer %s: %s", scenario, procedure, message),
		retry:          true,
	}
	if errors.Is(err, context.DeadlineExceeded) || strings.Contains(strings.ToLower(message), "deadline exceeded") || strings.Contains(strings.ToLower(message), "timed out") {
		failure.code = connect.CodeDeadlineExceeded
		failure.classification = "target_timeout"
		failure.message = fmt.Sprintf("target scenario %q did not answer %s before the deadline; retry after checking the target agent", scenario, procedure)
		return failure
	}

	statusMatch := upstreamHTTPStatusPattern.FindStringSubmatch(message)
	if len(statusMatch) != 2 {
		return failure
	}
	status, parseErr := strconv.Atoi(statusMatch[1])
	if parseErr != nil {
		return failure
	}
	failure.upstreamStatus = status
	switch {
	case status == http.StatusNotFound:
		failure.code = connect.CodeFailedPrecondition
		failure.classification = "target_incompatible"
		failure.retry = false
		failure.message = fmt.Sprintf("target scenario %q is missing the required API procedure %s; refresh or redeploy that scenario on the target", scenario, procedure)
	case status == http.StatusUnauthorized || status == http.StatusForbidden:
		failure.code = connect.CodePermissionDenied
		failure.classification = "target_unauthorized"
		failure.retry = false
		failure.message = fmt.Sprintf("target scenario %q refused API procedure %s; refresh the target trust or permission grant", scenario, procedure)
	case status == http.StatusBadRequest || status == http.StatusUnprocessableEntity:
		failure.code = connect.CodeFailedPrecondition
		failure.classification = "contract_mismatch"
		failure.retry = false
		failure.message = fmt.Sprintf("target scenario %q rejected API procedure %s because its contract is incompatible; refresh or redeploy the target scenario", scenario, procedure)
	case status >= http.StatusInternalServerError:
		failure.classification = "target_service_failure"
		failure.message = fmt.Sprintf("target scenario %q failed while answering %s; retry after checking the target service", scenario, procedure)
	}
	return failure
}

func writeScenarioProxyFailure(w http.ResponseWriter, r *http.Request, failure scenarioProxyFailure) {
	connectErr := connect.NewError(failure.code, errors.New(failure.message))
	connectErr.Meta().Set("X-Vrooli-Error-Code", failure.classification)
	connectErr.Meta().Set("X-Vrooli-Retryable", strconv.FormatBool(failure.retry))
	if failure.upstreamStatus != 0 {
		connectErr.Meta().Set("X-Vrooli-Upstream-Status", strconv.Itoa(failure.upstreamStatus))
	}
	for key, value := range map[string]string{
		"X-Vrooli-Target-Revision":        failure.version.TargetRevision,
		"X-Vrooli-Control-Plane-Revision": failure.version.ControlPlaneRevision,
		"X-Vrooli-Update-Path":            failure.version.UpdatePath,
		"X-Vrooli-Update-Command":         failure.version.UpdateCommand,
	} {
		if value != "" {
			connectErr.Meta().Set(key, value)
		}
	}
	if err := connect.NewErrorWriter().Write(w, r, connectErr); err != nil {
		http.Error(w, failure.message, http.StatusBadGateway)
	}
}

// Call is the raw-protobuf HTTP edge used by typed Connect clients after
// discovery resolves a registered target. The request path contributes only
// the governed scenario/service/method identity; payload bytes remain opaque.
func (h *handler) Call(w http.ResponseWriter, r *http.Request) {
	if _, err := auth.RequireOwner(r.Context()); err != nil {
		writeProxyError(w, http.StatusUnauthorized, err)
		return
	}
	vars := mux.Vars(r)
	nodeID := strings.TrimSpace(vars["node"])
	scenarioName := strings.TrimSpace(vars["scenario"])
	serviceName, methodName, err := splitProcedure(vars["procedure"])
	if nodeID == "" || scenarioName == "" || err != nil {
		if err != nil {
			writeProxyError(w, http.StatusBadRequest, err)
			return
		}
		writeProxyError(w, http.StatusBadRequest, errors.New("target, scenario, service, and method are required"))
		return
	}
	body, err := io.ReadAll(io.LimitReader(r.Body, int64(internal.MaxResponseBytes)+1))
	if err != nil {
		writeProxyError(w, http.StatusBadRequest, err)
		return
	}
	if uint64(len(body)) > internal.MaxResponseBytes {
		writeProxyError(w, http.StatusRequestEntityTooLarge, errors.New("scenario request exceeds byte limit"))
		return
	}
	response, err := h.deps.Service.Call(r.Context(), internal.Request{
		Actor: "owner", NodeID: nodeID, Scenario: scenarioName, Service: serviceName, Method: methodName,
		HTTPPath:   strings.Trim(vars["procedure"], "/"),
		HTTPMethod: r.Header.Get("X-Vrooli-HTTP-Method"),
		Body:       body, TimeoutSeconds: 30, MaxResponseBytes: internal.MaxResponseBytes,
	})
	if err != nil {
		failure := classifyScenarioProxyFailure(err, scenarioName, "/"+strings.Trim(vars["procedure"], "/"))
		if h.deps.Facts != nil && (failure.classification == "target_incompatible" || failure.classification == "contract_mismatch") {
			failure = failure.withVersionFacts(h.deps.Facts.VersionFacts(r.Context(), nodeID, scenarioName))
		}
		writeScenarioProxyFailure(w, r, failure)
		return
	}
	if scenarioName == deviceControlScenario && serviceName == desktopSessionService {
		// Web Console uses Connect-JSON for the browser-facing desktop pane.
		// The node agent makes the matching JSON hop to the managed companion;
		// ordinary scenario proxy procedures remain raw protobuf.
		w.Header().Set("Content-Type", "application/json")
	} else {
		w.Header().Set("Content-Type", "application/proto")
	}
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(response.Body)
}

func writeProxyError(w http.ResponseWriter, status int, err error) {
	http.Error(w, strings.TrimSpace(err.Error()), status)
}
