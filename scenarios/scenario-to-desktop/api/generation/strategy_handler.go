package generation

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"path/filepath"
	"strings"
	"time"

	"github.com/gorilla/mux"
)

type strategyListResponse struct {
	PolicyVersion string `json:"policy_version"`
	Default       string `json:"default_strategy"`
	Strategies    any    `json:"strategies"`
}

type packagedEvidenceGroup struct {
	Scenario     string         `json:"scenario"`
	Ramp         string         `json:"ramp"`
	Strategy     string         `json:"strategy"`
	Platform     string         `json:"platform"`
	Condition    string         `json:"condition"`
	ExperimentID string         `json:"experiment_id"`
	Comparable   bool           `json:"comparable"`
	Samples      int            `json:"samples"`
	Metrics      map[string]any `json:"metrics"`
}

type packagedEvidenceReport struct {
	Groups []packagedEvidenceGroup `json:"groups"`
}

type strategyExplainResponse struct {
	*PackagingSelectionRecord
	PackagedEvidence *packagedEvidenceReport `json:"packaged_evidence,omitempty"`
	EvidenceStatus   string                  `json:"packaged_evidence_status"`
}

type packagedEvidenceProvider func(context.Context, string, string) (*packagedEvidenceReport, error)

func RegisterStrategyRoutes(router *mux.Router, scenariosRoot string, providers ...packagedEvidenceProvider) {
	router.HandleFunc("/api/v1/strategies", func(w http.ResponseWriter, _ *http.Request) {
		writeStrategyJSON(w, http.StatusOK, strategyListResponse{PolicyVersion: SelectionPolicyV1, Default: DefaultStrategyID, Strategies: PackagingStrategies().List()})
	}).Methods(http.MethodGet)
	router.HandleFunc("/api/v1/strategies/{scenario}/explain", func(w http.ResponseWriter, r *http.Request) {
		scenario := strings.TrimSpace(mux.Vars(r)["scenario"])
		if scenario == "" || strings.ContainsAny(scenario, `/\\`) || scenario == "." || scenario == ".." {
			http.Error(w, "invalid scenario name", http.StatusBadRequest)
			return
		}
		selection, err := ResolvePackagingSelection(filepath.Join(scenariosRoot, scenario), r.URL.Query().Get("strategy"), r.URL.Query().Get("platform"))
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		response := strategyExplainResponse{PackagingSelectionRecord: selection, EvidenceStatus: "unconfigured"}
		if len(providers) != 0 && providers[0] != nil {
			response.EvidenceStatus = "available"
			report, reportErr := providers[0](r.Context(), scenario, DefaultStrategyID)
			if reportErr != nil {
				response.EvidenceStatus = "unavailable"
			} else {
				response.PackagedEvidence = report
			}
		}
		writeStrategyJSON(w, http.StatusOK, response)
	}).Methods(http.MethodGet)
}

func NewPackagedEvidenceProvider(baseURL string) packagedEvidenceProvider {
	baseURL = strings.TrimRight(strings.TrimSpace(baseURL), "/")
	return func(ctx context.Context, scenario, strategy string) (*packagedEvidenceReport, error) {
		if baseURL == "" {
			return nil, http.ErrNotSupported
		}
		parsed, err := url.Parse(baseURL + "/api/v1/packaged/report")
		if err != nil {
			return nil, err
		}
		query := parsed.Query()
		query.Set("scenario", scenario)
		query.Set("ramp", "desktop")
		query.Set("strategy", strategy)
		parsed.RawQuery = query.Encode()
		request, err := http.NewRequestWithContext(ctx, http.MethodGet, parsed.String(), nil)
		if err != nil {
			return nil, err
		}
		client := &http.Client{Timeout: 2 * time.Second}
		response, err := client.Do(request)
		if err != nil {
			return nil, err
		}
		defer response.Body.Close()
		if response.StatusCode < 200 || response.StatusCode >= 300 {
			return nil, http.ErrNotSupported
		}
		var report packagedEvidenceReport
		if err := json.NewDecoder(io.LimitReader(response.Body, 1<<20)).Decode(&report); err != nil {
			return nil, err
		}
		return &report, nil
	}
}

func writeStrategyJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}
