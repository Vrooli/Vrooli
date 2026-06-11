// Package search hosts the Connect-RPC handler for cli-health's
// SearchService, wired to the aisearch service (AI-first with text fallback).
package search

import (
	"context"
	"crypto/subtle"
	"errors"
	"log"
	"net/http"
	"strings"

	"connectrpc.com/connect"

	"cli-health/internal/aisearch"
	pkg "github.com/vrooli/ai-go/search"
	searchv1 "github.com/vrooli/vrooli/packages/proto/gen/go/cli-health/v1/search"
)

// Searcher is the seam between the Connect handler and the aisearch service.
// Unit tests inject fakes; production wires a real *aisearch.Service. The
// variadic pkg.SearchOptions carry per-request query-time overrides — the
// handler supplies them only after the override gate (token + flag) passes.
type Searcher interface {
	Search(ctx context.Context, query string, limit int, mode aisearch.SearchMode, opts ...pkg.SearchOption) (*aisearch.SearchResponse, error)
	Status(ctx context.Context) aisearch.StatusReport
}

// OverrideGate is the OUTER security layer of the query-time override channel
// (the inner layer is the engine's clamping OverridePolicy). A request's
// overrides are honored only when its control-token header matches the token
// search-hub minted for this provider at registration (Token, cached in memory
// from self-registration). Since search-hub is the only holder of that token,
// only search-hub's sweep can vary the query-time factors; an ordinary public
// request carries no token and gets the normal search. A nil gate, an empty
// cached token (not yet registered), or a mismatched token all mean "ignore the
// override, serve the ordinary search" — never an error. There is no env flag:
// the contract is governed by the SSOT (a provider opts out of being tuned by
// omitting its control endpoints in search.json) + this token.
type OverrideGate struct {
	// Token returns the provider's current cached control token ("" until the
	// scenario has self-registered with search-hub). Must be safe for concurrent
	// use with the registration goroutine that sets it.
	Token func() string
}

// Deps wires the seams the Connect search handler needs.
type Deps struct {
	Logger   *log.Logger
	Searcher Searcher
	// Overrides, when non-nil, enables the token-gated query-time override
	// channel. nil (the default) leaves the channel closed: override headers are
	// silently ignored and only the public search path runs.
	Overrides *OverrideGate
}

type connectHandler struct {
	deps Deps
}

// NewConnectHandler builds the Connect handler. Searcher may be nil when no
// search backend is configured — Search/Status then return Unimplemented.
func NewConnectHandler(d Deps) *connectHandler {
	if d.Logger == nil {
		d.Logger = log.Default()
	}
	return &connectHandler{deps: d}
}

func (h *connectHandler) Search(ctx context.Context, req *connect.Request[searchv1.SearchRequest]) (*connect.Response[searchv1.SearchResponse], error) {
	if h.deps.Searcher == nil {
		return nil, connect.NewError(connect.CodeUnimplemented, errors.New("search service not configured"))
	}
	r := req.Msg
	mode := protoModeToService(r.GetMode())
	opts := h.overrideOptions(req.Header())
	resp, err := h.deps.Searcher.Search(ctx, r.GetQuery(), int(r.GetLimit()), mode, opts...)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	wire := &searchv1.SearchResponse{
		ModeUsed: serviceMethodToProto(resp.Method),
		Reranker: resp.Reranker,
	}
	for _, hit := range resp.Results {
		wire.Results = append(wire.Results, &searchv1.SearchResult{
			Origin:      hit.Origin,
			Group:       hit.Group,
			Name:        hit.Name,
			Description: hit.Description,
			Score:       hit.Score,
			Source:      hit.Source,
			FullPath:    hit.FullPath,
			Tags:        hit.Tags,
			Binding:     hit.Binding,
			Weak:        hit.Weak,
		})
	}
	return connect.NewResponse(wire), nil
}

// overrideOptions decides whether to honor per-request query-time overrides and
// returns the SearchOptions to thread into the engine. An ordinary request (no
// override header) returns nil silently. A request that DOES carry overrides is
// honored only when the control token matches; every other outcome is logged once
// as telemetry and returns nil, so a rejected override degrades to the ordinary
// public search rather than failing.
func (h *connectHandler) overrideOptions(hdr http.Header) []pkg.SearchOption {
	raw := strings.TrimSpace(hdr.Get(pkg.OverridesHeader))
	if raw == "" {
		return nil // ordinary request — no override path, no telemetry.
	}
	gate := h.deps.Overrides
	if gate == nil {
		h.deps.Logger.Printf("[cli-health] search override ignored: channel not configured")
		return nil
	}
	if !gate.tokenMatches(hdr.Get(pkg.ControlTokenHeader)) {
		h.deps.Logger.Printf("[cli-health] search override ignored: control token mismatch")
		return nil
	}
	ov, err := pkg.ParseOverridesHeader(raw)
	if err != nil {
		h.deps.Logger.Printf("[cli-health] search override ignored: %v", err)
		return nil
	}
	if ov.IsZero() {
		return nil
	}
	return []pkg.SearchOption{pkg.WithOverrides(ov)}
}

// tokenMatches reports whether presented equals the gate's cached control token,
// using a constant-time compare. An empty cached token (not yet registered) or
// an empty presented token never matches.
func (g *OverrideGate) tokenMatches(presented string) bool {
	if g.Token == nil {
		return false
	}
	want := g.Token()
	if want == "" || presented == "" {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(presented), []byte(want)) == 1
}

func (h *connectHandler) Status(ctx context.Context, _ *connect.Request[searchv1.StatusRequest]) (*connect.Response[searchv1.StatusResponse], error) {
	if h.deps.Searcher == nil {
		return nil, connect.NewError(connect.CodeUnimplemented, errors.New("search service not configured"))
	}
	rep := h.deps.Searcher.Status(ctx)
	return connect.NewResponse(&searchv1.StatusResponse{
		Available:            rep.Available,
		Ollama:               rep.Ollama,
		Qdrant:               rep.Qdrant,
		IndexedCount:         int32(rep.IndexedCount),
		LastReconcileAt:      rep.LastReconcileAt,
		LastReconcileOutcome: rep.LastReconcileOutcome,
		Reranker:             rep.Reranker,
	}), nil
}

func protoModeToService(m searchv1.Mode) aisearch.SearchMode {
	switch m {
	case searchv1.Mode_MODE_AI:
		return aisearch.ModeAI
	case searchv1.Mode_MODE_TEXT:
		return aisearch.ModeText
	default:
		return aisearch.ModeAuto
	}
}

func serviceMethodToProto(method string) searchv1.Mode {
	switch method {
	case "ai":
		return searchv1.Mode_MODE_AI
	case "text":
		return searchv1.Mode_MODE_TEXT
	default:
		return searchv1.Mode_MODE_UNSPECIFIED
	}
}
