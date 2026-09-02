package scenario

import (
	"errors"
	"io"
	"net/http"
	"strings"

	"vrooli-bridge/internal/auth"
	internal "vrooli-bridge/internal/scenario"

	"github.com/gorilla/mux"
)

type Deps struct{ Service internal.Service }

type handler struct{ deps Deps }

func NewHandler(deps Deps) *handler { return &handler{deps: deps} }

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
	admissionMethod := methodName
	if strings.HasPrefix(methodName, "v2/apply/") {
		admissionMethod = "v2/apply/{run_id}"
	}
	response, err := h.deps.Service.Call(r.Context(), internal.Request{
		Actor: "owner", NodeID: nodeID, Scenario: scenarioName, Service: serviceName, Method: admissionMethod,
		HTTPPath:   strings.Trim(vars["procedure"], "/"),
		HTTPMethod: r.Header.Get("X-Vrooli-HTTP-Method"),
		Body:       body, TimeoutSeconds: 30, MaxResponseBytes: internal.MaxResponseBytes,
	})
	if err != nil {
		writeProxyError(w, http.StatusBadGateway, err)
		return
	}
	w.Header().Set("Content-Type", "application/proto")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(response.Body)
}

func writeProxyError(w http.ResponseWriter, status int, err error) {
	http.Error(w, strings.TrimSpace(err.Error()), status)
}
