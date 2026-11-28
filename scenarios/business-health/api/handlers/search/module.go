// Package search mounts the scenario-local intent SearchService over
// internal/aisearch — the query surface behind the `business-health.intent`
// search-hub leaf.
package search

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"sync"

	"connectrpc.com/connect"
	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/connectx"

	"business-health/internal/aisearch"
	"business-health/internal/changeintent"
	"business-health/internal/module"

	searchv1 "github.com/vrooli/vrooli/packages/proto/gen/go/business-health/v1/search"
	searchconnect "github.com/vrooli/vrooli/packages/proto/gen/go/business-health/v1/search/search_v1connect"
)

// ProtoFile exposes the search domain's proto FileDescriptor for the
// global parity test.
var ProtoFile = searchv1.File_business_health_v1_search_search_proto

// TokenHolder is the in-memory home for the control token search-hub mints
// at self-registration. Get returns "" until Set runs; the control gate
// treats an empty token as deny, so control stays closed until
// registration completes. Memory-only by design — search-hub persists the
// authoritative copy and re-registration re-acquires it.
type TokenHolder struct {
	mu    sync.RWMutex
	token string
}

func NewTokenHolder() *TokenHolder { return &TokenHolder{} }

func (h *TokenHolder) Set(token string) {
	h.mu.Lock()
	h.token = token
	h.mu.Unlock()
}

func (h *TokenHolder) Get() string {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.token
}

// Searcher is the aisearch seam.
type Searcher interface {
	Search(ctx context.Context, query string, limit int, mode aisearch.SearchMode, opts ...func()) (*aisearch.SearchResponse, error)
	Status(ctx context.Context) aisearch.StatusReport
	Collection() string
}

type handler struct {
	logger   *log.Logger
	searcher *aisearch.Service
}

// Module mounts the SearchService. searcher may be nil (no search backend
// configured — the handler answers Unimplemented).
func Module(logger *log.Logger, searcher *aisearch.Service, repoRoot string) module.Module {
	path, svc := searchconnect.NewSearchServiceHandler(&handler{logger: logger, searcher: searcher})
	return module.Module{
		Name: "search",
		Mount: func(r *mux.Router) {
			connectx.RegisterServices(r, connectx.ServiceMount{Path: path, Handler: svc})
			r.HandleFunc("/api/v1/change-intent/assess", assessIntent(repoRoot)).Methods(http.MethodPost)
		},
		Endpoints: Endpoints,
	}
}

func assessIntent(repoRoot string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		r.Body = http.MaxBytesReader(w, r.Body, 8<<10)
		decoder := json.NewDecoder(r.Body)
		decoder.DisallowUnknownFields()
		var request changeintent.Request
		if err := decoder.Decode(&request); err != nil {
			http.Error(w, "invalid request: "+err.Error(), http.StatusBadRequest)
			return
		}
		if err := decoder.Decode(&struct{}{}); err != io.EOF {
			http.Error(w, "request must contain one JSON object", http.StatusBadRequest)
			return
		}
		if strings.TrimSpace(request.Scenario) != "business-health" {
			http.Error(w, "scenario must be business-health", http.StatusBadRequest)
			return
		}
		response, err := changeintent.Assess(repoRoot, request)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Cache-Control", "no-store")
		if err := json.NewEncoder(w).Encode(response); err != nil {
			return
		}
	}
}

// Schema returns "" — the index lives in Qdrant, not the scenario database.
func Schema() string { return "" }

func (h *handler) Search(ctx context.Context, req *connect.Request[searchv1.SearchRequest]) (*connect.Response[searchv1.SearchResponse], error) {
	if h.searcher == nil {
		return nil, connect.NewError(connect.CodeUnimplemented, fmt.Errorf("intent search backend is not configured"))
	}
	limit := int(req.Msg.GetLimit())
	if limit <= 0 {
		limit = 10
	}
	mode := aisearch.ModeAuto
	switch req.Msg.GetMode() {
	case searchv1.Mode_MODE_AI:
		mode = aisearch.ModeAI
	case searchv1.Mode_MODE_TEXT:
		mode = aisearch.ModeText
	}
	resp, err := h.searcher.Search(ctx, req.Msg.GetQuery(), limit, mode)
	if err != nil {
		return nil, connect.NewError(connect.CodeUnavailable, err)
	}
	out := &searchv1.SearchResponse{Mode: searchv1.Mode_MODE_TEXT}
	if resp.Method == "ai" {
		out.Mode = searchv1.Mode_MODE_AI
	}
	for _, hit := range resp.Results {
		out.Results = append(out.Results, &searchv1.IntentHit{
			Id:       hit.ID,
			Scenario: hit.Scenario,
			Type:     hit.Type,
			Title:    hit.Title,
			Snippet:  hit.Snippet,
			Anchor:   hit.Anchor,
			PrdRef:   hit.PRDRef,
			Score:    float32(hit.Score),
			Weak:     hit.Weak,
		})
	}
	return connect.NewResponse(out), nil
}

func (h *handler) Status(ctx context.Context, req *connect.Request[searchv1.StatusRequest]) (*connect.Response[searchv1.StatusResponse], error) {
	if h.searcher == nil {
		return connect.NewResponse(&searchv1.StatusResponse{Available: false, Detail: "intent search backend is not configured"}), nil
	}
	r := h.searcher.Status(ctx)
	return connect.NewResponse(&searchv1.StatusResponse{
		Available:       r.Available,
		OllamaUp:        r.Ollama,
		QdrantUp:        r.Qdrant,
		RerankerUp:      r.Reranker != "" && r.Reranker != "none" && r.Reranker != "degraded",
		Indexed:         int64(r.IndexedCount),
		Collection:      h.searcher.Collection(),
		Detail:          r.LastReconcileOutcome,
		LastReconcileAt: r.LastReconcileAt,
	}), nil
}

var Endpoints = []module.EndpointDescriptor{
	{
		ID: "change_intent_assess", Path: "/api/v1/change-intent/assess", Method: "POST",
		Summary:     "Relate one explicit change statement to bounded requirement and source anchors",
		Description: "One-scenario, read-only traceability candidates from checked-in requirements and exact [REQ:ID] source tags. Every relation carries scenario identity and declaration path/line/hash; support means an exact ID mention only, lexical candidates remain unknown, and source pointers are not verified change rationale. Partial source scans are reported as degradations. Does not read generated PRD or runtime/test status.",
		Category:    "search",
		Request:     &module.Schema{Type: "object", Properties: map[string]string{"scenario": "string (must be business-health)", "intent": "string (max 4096 bytes)", "limit": "integer (max 20)"}},
		Response:    &module.Schema{Type: "object", Properties: map[string]string{"scenario": "string", "results": "array<RequirementRelation(scenario, requirement_id, relation, relation_basis, requirement_source.path/line/sha256, implementation_anchors[])>", "limitations": "array<string>", "degradations": "array<string> (partial scan/cap state)"}},
		Errors:      []module.ErrorDesc{{Status: 400, Code: "invalid_argument", Description: "Invalid statement, unsupported scenario or malformed JSON"}},
	},
	{
		ID:          "search_query",
		Path:        searchconnect.SearchServiceSearchProcedure,
		Method:      "POST",
		Summary:     "Search the fleet-wide intent corpus",
		Description: "Semantic (AI) and text-fallback search over every scenario's PRD purpose, operational targets, and requirements — 'which scenario provides capability X' and 'what is Y supposed to deliver'. Pointer-only results (anchors into PRD/requirements).",
		Category:    "search",
		Request:     &module.Schema{Type: "object", Properties: map[string]string{"query": "string", "limit": "int32", "mode": "Mode"}},
		Response:    &module.Schema{Type: "object", Properties: map[string]string{"results": "array<IntentHit>", "mode": "Mode", "degraded_reason": "string"}},
		Errors: []module.ErrorDesc{
			{Status: 501, Code: "unimplemented", Description: "No search backend configured"},
			{Status: 503, Code: "unavailable", Description: "Both the AI leg and the text fallback failed"},
		},
		Examples: []module.Example{
			{Name: "Capability query", Curl: "curl http://localhost:${API_PORT}/vrooli.business_health.v1.search.SearchService/Search -H 'Content-Type: application/json' -d '{\"query\":\"which scenario resizes images\",\"limit\":5}'"},
		},
	},
	{
		ID:          "search_status",
		Path:        searchconnect.SearchServiceStatusProcedure,
		Method:      "POST",
		Summary:     "Report intent-search backend availability",
		Description: "Reports ollama/qdrant/reranker reachability, indexed doc count, and the live collection.",
		Category:    "search",
		Response:    &module.Schema{Type: "object", Properties: map[string]string{"available": "bool", "indexed": "int64", "collection": "string"}},
	},
}
