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

type Deps struct{ Service internal.Service }

type handler struct{ deps Deps }

func NewHandler(deps Deps) *handler { return &handler{deps: deps} }

var upstreamHTTPStatusPattern = regexp.MustCompile(`(?i)\bHTTP\s+(\d{3})\b`)

type scenarioProxyFailure struct {
	code           connect.Code
	classification string
	message        string
	retry          bool
	upstreamStatus int
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
		writeScenarioProxyFailure(w, r, classifyScenarioProxyFailure(err, scenarioName, "/"+strings.Trim(vars["procedure"], "/")))
		return
	}
	w.Header().Set("Content-Type", "application/proto")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(response.Body)
}

func writeProxyError(w http.ResponseWriter, status int, err error) {
	http.Error(w, strings.TrimSpace(err.Error()), status)
}
