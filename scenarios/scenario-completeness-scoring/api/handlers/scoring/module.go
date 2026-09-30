// Package scoring exposes the ScoreService Connect-RPC surface: the fast,
// cached scenario status read path.
package scoring

import (
	"log"

	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/connectx"

	scoringconnect "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-completeness-scoring/v1/scoring/scoring_v1connect"

	"scenario-completeness-scoring/internal/module"
	internalscoring "scenario-completeness-scoring/internal/scoring"
)

// Module returns the scoring domain's contribution to the API. The scorer
// is injected (constructed in main.go via internal/scoring.New) so the
// scenarios-root resolution failure surfaces at boot, not per request.
func Module(scorer Scorer, snapshots SnapshotRepository, logger *log.Logger) module.Module {
	connectPath, connectHandler := scoringconnect.NewScoreServiceHandler(NewConnectHandler(Deps{
		Scorer:    scorer,
		Snapshots: snapshots,
		Logger:    logger,
	}))
	return module.Module{
		Name: "scoring",
		Mount: func(r *mux.Router) {
			connectx.RegisterServices(r, connectx.ServiceMount{Path: connectPath, Handler: connectHandler})
		},
		Endpoints: Endpoints,
	}
}

// Schema returns the scoring domain's durable snapshot schema.
func Schema() string { return internalscoring.Schema() }

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
	},
	{
		ID:          "scoring_trend",
		Path:        scoringconnect.ScoreServiceGetScoreTrendProcedure,
		Method:      "POST",
		Summary:     "Get a scenario's score trend",
		Description: "Reads digest-deduplicated score snapshots for one scenario from the persisted score history store, newest first.",
		Category:    "scoring",
		Request: &module.Schema{
			Type: "object",
			Properties: map[string]string{
				"scenario": "string (required, scenario directory name)",
				"limit":    "int32 (optional)",
				"since":    "Timestamp (optional)",
			},
		},
		Response: &module.Schema{
			Type: "object",
			Properties: map[string]string{
				"scenario":  "string",
				"snapshots": "array<ScoreSnapshot>",
			},
		},
		Errors: []module.ErrorDesc{
			{Status: 500, Code: "internal", Description: "Trend read failure"},
			{Status: 503, Code: "unavailable", Description: "Snapshot repository unavailable"},
		},
		Examples: []module.Example{
			{Name: "Trend for a scenario", Curl: "curl http://localhost:${API_PORT}/vrooli.scenario_completeness_scoring.v1.scoring.ScoreService/GetScoreTrend -H 'Content-Type: application/json' -d '{\"scenario\":\"cli-health\",\"limit\":10}'"},
		},
	},
	{
		ID:          "scoring_list",
		Path:        scoringconnect.ScoreServiceListScoresProcedure,
		Method:      "POST",
		Summary:     "List latest scenario scores",
		Description: "Reads the latest persisted score snapshot per scenario with server-side sort, filter, and pagination. This path is O(query) over score_snapshots and never recomputes the fleet.",
		Category:    "scoring",
		Request: &module.Schema{
			Type: "object",
			Properties: map[string]string{
				"sort_by":    "ScoreSortBy",
				"order":      "SortOrder",
				"page_size":  "int32",
				"page_token": "string",
				"min_score":  "int32 (optional)",
				"max_score":  "int32 (optional)",
				"rung":       "string (optional)",
				"category":   "string (optional)",
				"recompute":  "bool (optional, page-bounded later phase)",
			},
		},
		Response: &module.Schema{
			Type: "object",
			Properties: map[string]string{
				"scores":          "array<ScoreRow>",
				"next_page_token": "string",
			},
		},
		Errors: []module.ErrorDesc{
			{Status: 400, Code: "invalid_argument", Description: "Bad page_token"},
			{Status: 500, Code: "internal", Description: "List read failure"},
			{Status: 503, Code: "unavailable", Description: "Snapshot repository unavailable"},
		},
		Examples: []module.Example{
			{Name: "List lowest scores", Curl: "curl http://localhost:${API_PORT}/vrooli.scenario_completeness_scoring.v1.scoring.ScoreService/ListScores -H 'Content-Type: application/json' -d '{\"sort_by\":\"SCORE_SORT_BY_COMPOSITE\",\"order\":\"SORT_ORDER_ASC\",\"page_size\":10}'"},
		},
	},
}
