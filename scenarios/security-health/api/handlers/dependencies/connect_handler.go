// Package dependencies hosts the Connect-RPC handler for security-health's
// DependencyService — the fleet Dependency & Vulnerability Intelligence query
// surface. It maps the internal dependencies.Service domain types onto the
// proto wire shape.
package dependencies

import (
	"context"
	"log"

	"connectrpc.com/connect"

	depdomain "security-health/internal/dependencies"

	dependenciesv1 "github.com/vrooli/vrooli/packages/proto/gen/go/security-health/v1/dependencies"
)

// Searcher is the slice of dependencies.Service the handler exercises.
type Searcher interface {
	Search(ctx context.Context, req depdomain.SearchRequest) (depdomain.SearchResponse, error)
	Status(ctx context.Context) (depdomain.Status, error)
}

// Deps wires the handler's seams.
type Deps struct {
	Logger  *log.Logger
	Service Searcher
}

type connectHandler struct {
	deps Deps
}

// NewConnectHandler returns a handler satisfying DependencyServiceHandler.
func NewConnectHandler(d Deps) *connectHandler {
	if d.Logger == nil {
		d.Logger = log.Default()
	}
	return &connectHandler{deps: d}
}

func (h *connectHandler) Search(ctx context.Context, req *connect.Request[dependenciesv1.SearchRequest]) (*connect.Response[dependenciesv1.SearchResponse], error) {
	m := req.Msg
	resp, err := h.deps.Service.Search(ctx, depdomain.SearchRequest{
		Query:          m.GetQuery(),
		Limit:          int(m.GetLimit()),
		Mode:           modeFromProto(m.GetMode()),
		Ecosystem:      ecosystemFromProto(m.GetEcosystem()),
		VulnerableOnly: m.GetVulnerableOnly(),
		NameGlob:       m.GetNameGlob(),
	})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	out := &dependenciesv1.SearchResponse{ModeUsed: modeToProto(resp.ModeUsed)}
	for _, r := range resp.Results {
		out.Results = append(out.Results, &dependenciesv1.SearchResult{
			Record: recordToProto(r.Record),
			Score:  r.Score,
		})
	}
	return connect.NewResponse(out), nil
}

func (h *connectHandler) Status(ctx context.Context, _ *connect.Request[dependenciesv1.StatusRequest]) (*connect.Response[dependenciesv1.StatusResponse], error) {
	st, err := h.deps.Service.Status(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&dependenciesv1.StatusResponse{
		Available:            st.Available,
		Ollama:               st.Ollama,
		Qdrant:               st.Qdrant,
		IndexedCount:         int32(st.IndexedCount),
		VulnerableCount:      int32(st.VulnerableCount),
		LastReconcileAt:      st.LastReconcileAt,
		LastReconcileOutcome: st.LastReconcileOutcome,
		IndexedVectors:       int32(st.IndexedVectors),
		ExpectedVectors:      int32(st.ExpectedVectors),
		IndexReady:           st.IndexReady,
	}), nil
}

func recordToProto(r depdomain.DependencyRecord) *dependenciesv1.DependencyRecord {
	return &dependenciesv1.DependencyRecord{
		Scenario:    r.Scenario,
		Ecosystem:   ecosystemToProto(r.Ecosystem),
		Name:        r.Name,
		Version:     r.Version,
		SourceFile:  r.SourceFile,
		VulnIds:     r.VulnIDs,
		MaxSeverity: r.MaxSeverity,
		LastSeen:    r.LastSeen,
	}
}

func modeFromProto(m dependenciesv1.Mode) depdomain.Mode {
	switch m {
	case dependenciesv1.Mode_MODE_AI:
		return depdomain.ModeAI
	case dependenciesv1.Mode_MODE_TEXT:
		return depdomain.ModeText
	default:
		return depdomain.ModeUnspecified
	}
}

func modeToProto(m depdomain.Mode) dependenciesv1.Mode {
	switch m {
	case depdomain.ModeAI:
		return dependenciesv1.Mode_MODE_AI
	case depdomain.ModeText:
		return dependenciesv1.Mode_MODE_TEXT
	default:
		return dependenciesv1.Mode_MODE_UNSPECIFIED
	}
}

func ecosystemFromProto(e dependenciesv1.Ecosystem) depdomain.Ecosystem {
	switch e {
	case dependenciesv1.Ecosystem_ECOSYSTEM_GO:
		return depdomain.EcosystemGo
	case dependenciesv1.Ecosystem_ECOSYSTEM_NPM:
		return depdomain.EcosystemNPM
	default:
		return depdomain.EcosystemUnspecified
	}
}

func ecosystemToProto(e depdomain.Ecosystem) dependenciesv1.Ecosystem {
	switch e {
	case depdomain.EcosystemGo:
		return dependenciesv1.Ecosystem_ECOSYSTEM_GO
	case depdomain.EcosystemNPM:
		return dependenciesv1.Ecosystem_ECOSYSTEM_NPM
	default:
		return dependenciesv1.Ecosystem_ECOSYSTEM_UNSPECIFIED
	}
}
