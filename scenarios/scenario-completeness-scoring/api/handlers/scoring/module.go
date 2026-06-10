// Package scoring exposes the ScoreService Connect-RPC surface: the fast,
// cached scenario status read path.
package scoring

import (
	"log"

	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/connectx"

	scoringconnect "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-completeness-scoring/v1/scoring/scoring_v1connect"

	"scenario-completeness-scoring/internal/module"
)

// Module returns the scoring domain's contribution to the API. The scorer
// is injected (constructed in main.go via internal/scoring.New) so the
// scenarios-root resolution failure surfaces at boot, not per request.
func Module(scorer Scorer, logger *log.Logger) module.Module {
	connectPath, connectHandler := scoringconnect.NewScoreServiceHandler(NewConnectHandler(Deps{
		Scorer: scorer,
		Logger: logger,
	}))
	return module.Module{
		Name: "scoring",
		Mount: func(r *mux.Router) {
			connectx.RegisterServices(r, connectx.ServiceMount{Path: connectPath, Handler: connectHandler})
		},
		Endpoints: Endpoints,
	}
}

// Schema returns "" — scoring persists nothing (no score history in v1).
func Schema() string { return "" }

// Endpoints describes the scoring module's public surface for codegen and
// the global proto/Connect parity test.
var Endpoints = []module.EndpointDescriptor{
	{
		ID:          "scoring_get",
		Path:        scoringconnect.ScoreServiceGetScoreProcedure,
		Method:      "POST",
		Summary:     "Get a scenario's completeness score",
		Description: "Computes the cached-artifact status payload for one scenario: maturity rung (as of tree digest), 0-100 composite with classification and breakdown, prioritized recommendations with point impact, action plan, and per-phase freshness verdicts with a refresh command. Filesystem-only; <1s warm.",
		Category:    "scoring",
		Request: &module.Schema{
			Type: "object",
			Properties: map[string]string{
				"scenario": "string (required, scenario directory name)",
			},
		},
		Response: &module.Schema{
			Type: "object",
			Properties: map[string]string{
				"scenario":        "string",
				"category":        "string",
				"maturity":        "MaturityHeadline",
				"composite":       "CompositeScore",
				"freshness":       "FreshnessBlock",
				"recommendations": "array<Recommendation>",
				"action_plan":     "array<ActionPhase>",
				"degradations":    "array<CollectorDegradation>",
				"calculated_at":   "Timestamp",
			},
		},
		Errors: []module.ErrorDesc{
			{Status: 404, Code: "not_found", Description: "Unknown scenario name"},
			{Status: 500, Code: "internal", Description: "Score assembly failure"},
		},
		Examples: []module.Example{
			{Name: "Score a scenario", Curl: "curl http://localhost:${API_PORT}/vrooli.scenario_completeness_scoring.v1.scoring.ScoreService/GetScore -H 'Content-Type: application/json' -d '{\"scenario\":\"cli-health\"}'"},
		},
		CLIMapping: &module.CLIMapping{
			Command: "scenario-completeness-scoring score get",
			Args:    []string{"<scenario>"},
		},
	},
}
