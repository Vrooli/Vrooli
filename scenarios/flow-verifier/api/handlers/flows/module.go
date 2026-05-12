// Package flows wires the HTTP surface for flow discovery and inspection.
//
// Two list modes:
//
//   - With no query string: aggregate every flow across every scenario
//     under the Vrooli root, each row stamped with its scenarioId. This
//     is what the inventory UI's flat "all flows" view hits.
//   - With ?scenario=<id>: list flows for one scenario by reading its
//     embedded flow list from scenarios.Service.Detail.
//   - With ?root=<path>: legacy / CLI testing override that lists flows
//     under an arbitrary directory. Kept because the CLI's `flows list
//     --root <path>` shape still routes through here.
package flows

import (
	"encoding/json"
	"errors"
	"net/http"

	"flow-verifier/internal/flows"
	"flow-verifier/internal/httpx"
	"flow-verifier/internal/module"
	"flow-verifier/internal/scenarios"

	"github.com/gorilla/mux"
)

// ScenariosService is the subset of scenarios.Service this module
// depends on. Declared inline (vs. importing the handler's Service
// interface) so the dependency arrow stays pointed inward: handlers
// flows → internal scenarios, not handlers flows → handlers scenarios.
type ScenariosService interface {
	List() ([]scenarios.Summary, error)
	Detail(id string) (scenarios.Detail, error)
}

// Module returns the flows domain's HTTP contribution. svc may be nil
// for tests that exercise the legacy ?root= path only; the
// scenario-aware branches return 500 in that case rather than panic.
func Module(svc ScenariosService) module.Module {
	return module.Module{
		Name: "flows",
		Mount: func(r *mux.Router) {
			r.HandleFunc("/api/v1/flows", listHandler(svc)).Methods(http.MethodGet)
			r.HandleFunc("/api/v1/flows/{id}", getHandler(svc)).Methods(http.MethodGet)
		},
		Endpoints: Endpoints,
	}
}

// Schema returns "" — flow inventory is filesystem-truth, no tables.
func Schema() string { return "" }

func listHandler(svc ScenariosService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		switch {
		case q.Get("scenario") != "":
			listByScenario(w, svc, q.Get("scenario"))
		case q.Get("root") != "":
			listByRoot(w, q.Get("root"))
		default:
			listAggregate(w, svc)
		}
	}
}

func listByScenario(w http.ResponseWriter, svc ScenariosService, id string) {
	if svc == nil {
		httpx.WriteError(w, http.StatusInternalServerError, httpx.CodeInternal, "scenarios service not configured")
		return
	}
	detail, err := svc.Detail(id)
	if err != nil {
		if errors.Is(err, scenarios.ErrScenarioNotFound) {
			httpx.WriteError(w, http.StatusNotFound, httpx.CodeNotFound, err.Error())
			return
		}
		httpx.WriteError(w, http.StatusInternalServerError, httpx.CodeInternal, err.Error())
		return
	}
	rows := stampScenarioID(detail.Flows, id)
	writeJSON(w, http.StatusOK, map[string]any{"flows": rows})
}

func listByRoot(w http.ResponseWriter, root string) {
	rows, err := flows.List(root, "")
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, httpx.CodeInvalidRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"flows": rows})
}

func listAggregate(w http.ResponseWriter, svc ScenariosService) {
	if svc == nil {
		httpx.WriteError(w, http.StatusInternalServerError, httpx.CodeInternal, "scenarios service not configured")
		return
	}
	all, err := svc.List()
	if err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, httpx.CodeInternal, err.Error())
		return
	}
	out := make([]flows.Summary, 0)
	for _, scenario := range all {
		// Skip scenarios that errored during discovery — the scenarios
		// list endpoint surfaces the error on the per-row card; we
		// don't fail the global flow list because one scenario broke.
		if scenario.DiscoveryErr != "" || scenario.FlowCount == 0 {
			continue
		}
		detail, err := svc.Detail(scenario.ID)
		if err != nil {
			continue
		}
		out = append(out, stampScenarioID(detail.Flows, scenario.ID)...)
	}
	writeJSON(w, http.StatusOK, map[string]any{"flows": out})
}

func stampScenarioID(rows []flows.Summary, id string) []flows.Summary {
	out := make([]flows.Summary, len(rows))
	for i, r := range rows {
		r.ScenarioID = id
		out[i] = r
	}
	return out
}

func getHandler(svc ScenariosService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := mux.Vars(r)["id"]
		q := r.URL.Query()
		// Resolution rules for the flow's root, in priority order:
		//   1. Explicit ?root= (legacy CLI override).
		//   2. ?scenario=<id> → resolve through the scenarios service.
		//   3. Scan every scenario and find the one that owns the flow.
		root, err := resolveFlowRoot(svc, q.Get("root"), q.Get("scenario"), id)
		if err != nil {
			httpx.WriteError(w, http.StatusNotFound, httpx.CodeNotFound, err.Error())
			return
		}
		detail, err := flows.Detail(root, id)
		if err != nil {
			httpx.WriteError(w, http.StatusNotFound, httpx.CodeNotFound, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, detail)
	}
}

func resolveFlowRoot(svc ScenariosService, root, scenarioID, flowID string) (string, error) {
	if root != "" {
		return root, nil
	}
	if svc == nil {
		return "", errors.New("scenarios service not configured")
	}
	if scenarioID != "" {
		detail, err := svc.Detail(scenarioID)
		if err != nil {
			return "", err
		}
		return detail.Path, nil
	}
	// No scenario hint → find which scenario owns this flow id. O(N)
	// over scenarios; acceptable because the UI passes ?scenario= when
	// it knows.
	all, err := svc.List()
	if err != nil {
		return "", err
	}
	for _, scenario := range all {
		if scenario.DiscoveryErr != "" || scenario.FlowCount == 0 {
			continue
		}
		detail, err := svc.Detail(scenario.ID)
		if err != nil {
			continue
		}
		for _, row := range detail.Flows {
			if row.FlowID == flowID {
				return scenario.Path, nil
			}
		}
	}
	return "", errors.New("flow not found in any scenario")
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}
