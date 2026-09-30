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
	ListVulnerabilities(ctx context.Context, req depdomain.VulnerabilityQuery) (depdomain.VulnerabilityList, error)
	ExplainVulnerability(ctx context.Context, req depdomain.VulnerabilityQuery) (depdomain.VulnerabilityRecord, bool, error)
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

func (h *connectHandler) ListVulnerabilities(ctx context.Context, req *connect.Request[dependenciesv1.ListVulnerabilitiesRequest]) (*connect.Response[dependenciesv1.ListVulnerabilitiesResponse], error) {
	resp, err := h.deps.Service.ListVulnerabilities(ctx, depdomain.VulnerabilityQuery{
		Ecosystem:         ecosystemFromProto(req.Msg.GetEcosystem()),
		PackageName:       req.Msg.GetPackageName(),
		Scenario:          req.Msg.GetScenario(),
		VulnerabilityID:   req.Msg.GetVulnerabilityId(),
		MinimumConfidence: confidenceFromProto(req.Msg.GetMinimumConfidence()),
		Limit:             int(req.Msg.GetLimit()),
	})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	out := &dependenciesv1.ListVulnerabilitiesResponse{Total: int32(resp.Total)}
	for _, vuln := range resp.Vulnerabilities {
		out.Vulnerabilities = append(out.Vulnerabilities, vulnerabilityToProto(vuln))
	}
	return connect.NewResponse(out), nil
}

func (h *connectHandler) ExplainVulnerability(ctx context.Context, req *connect.Request[dependenciesv1.ExplainVulnerabilityRequest]) (*connect.Response[dependenciesv1.ExplainVulnerabilityResponse], error) {
	vuln, found, err := h.deps.Service.ExplainVulnerability(ctx, depdomain.VulnerabilityQuery{
		VulnerabilityID: req.Msg.GetVulnerabilityId(),
		Ecosystem:       ecosystemFromProto(req.Msg.GetEcosystem()),
		PackageName:     req.Msg.GetPackageName(),
		Limit:           1,
	})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&dependenciesv1.ExplainVulnerabilityResponse{
		Vulnerability: vulnerabilityToProto(vuln),
		Found:         found,
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

func vulnerabilityToProto(v depdomain.VulnerabilityRecord) *dependenciesv1.VulnerabilityRecord {
	out := &dependenciesv1.VulnerabilityRecord{
		VulnerabilityId:    v.VulnerabilityID,
		Aliases:            v.Aliases,
		Ecosystem:          ecosystemToProto(v.Ecosystem),
		Name:               v.Name,
		Version:            v.Version,
		Severity:           v.Severity,
		NormalizedSeverity: v.NormalizedSeverity,
		AdvisoryUrl:        v.AdvisoryURL,
		Summary:            v.Summary,
		Details:            v.Details,
		Source:             sourceToProto(v.Source),
		Reachability:       reachabilityToProto(v.Reachability),
		Confidence:         confidenceToProto(v.Confidence),
		Production:         v.Production,
		DevOnly:            v.DevOnly,
		FirstSeen:          v.FirstSeen,
		LastSeen:           v.LastSeen,
		Scenarios:          v.Scenarios,
		SourceFiles:        v.SourceFiles,
		Remediation:        v.Remediation,
	}
	for _, r := range v.AffectedRanges {
		out.AffectedRanges = append(out.AffectedRanges, &dependenciesv1.AffectedVersionRange{
			Range:        r.Range,
			Introduced:   r.Introduced,
			Fixed:        r.Fixed,
			LastAffected: r.LastAffected,
		})
	}
	for _, r := range v.FixedRanges {
		out.FixedRanges = append(out.FixedRanges, &dependenciesv1.FixedVersionRange{
			Range:   r.Range,
			Version: r.Version,
		})
	}
	return out
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
	case dependenciesv1.Ecosystem_ECOSYSTEM_YARN:
		return depdomain.EcosystemYarn
	case dependenciesv1.Ecosystem_ECOSYSTEM_BUN:
		return depdomain.EcosystemBun
	case dependenciesv1.Ecosystem_ECOSYSTEM_PYTHON:
		return depdomain.EcosystemPython
	case dependenciesv1.Ecosystem_ECOSYSTEM_RUST:
		return depdomain.EcosystemRust
	case dependenciesv1.Ecosystem_ECOSYSTEM_C:
		return depdomain.EcosystemC
	case dependenciesv1.Ecosystem_ECOSYSTEM_CPP:
		return depdomain.EcosystemCPP
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
	case depdomain.EcosystemYarn:
		return dependenciesv1.Ecosystem_ECOSYSTEM_YARN
	case depdomain.EcosystemBun:
		return dependenciesv1.Ecosystem_ECOSYSTEM_BUN
	case depdomain.EcosystemPython:
		return dependenciesv1.Ecosystem_ECOSYSTEM_PYTHON
	case depdomain.EcosystemRust:
		return dependenciesv1.Ecosystem_ECOSYSTEM_RUST
	case depdomain.EcosystemC:
		return dependenciesv1.Ecosystem_ECOSYSTEM_C
	case depdomain.EcosystemCPP:
		return dependenciesv1.Ecosystem_ECOSYSTEM_CPP
	default:
		return dependenciesv1.Ecosystem_ECOSYSTEM_UNSPECIFIED
	}
}

func sourceToProto(s depdomain.VulnerabilitySource) dependenciesv1.VulnerabilitySource {
	switch s {
	case depdomain.VulnerabilitySourceOSV:
		return dependenciesv1.VulnerabilitySource_VULNERABILITY_SOURCE_OSV
	case depdomain.VulnerabilitySourceGovulncheck:
		return dependenciesv1.VulnerabilitySource_VULNERABILITY_SOURCE_GOVULNCHECK
	case depdomain.VulnerabilitySourcePnpmAudit:
		return dependenciesv1.VulnerabilitySource_VULNERABILITY_SOURCE_PNPM_AUDIT
	default:
		return dependenciesv1.VulnerabilitySource_VULNERABILITY_SOURCE_UNSPECIFIED
	}
}

func reachabilityToProto(r depdomain.Reachability) dependenciesv1.Reachability {
	switch r {
	case depdomain.ReachabilityUnknown:
		return dependenciesv1.Reachability_REACHABILITY_UNKNOWN
	case depdomain.ReachabilityLockfileAffected:
		return dependenciesv1.Reachability_REACHABILITY_LOCKFILE_AFFECTED
	case depdomain.ReachabilityReachable:
		return dependenciesv1.Reachability_REACHABILITY_REACHABLE
	default:
		return dependenciesv1.Reachability_REACHABILITY_UNSPECIFIED
	}
}

func confidenceFromProto(c dependenciesv1.EvidenceConfidence) depdomain.EvidenceConfidence {
	switch c {
	case dependenciesv1.EvidenceConfidence_EVIDENCE_CONFIDENCE_DEGRADED:
		return depdomain.EvidenceConfidenceDegraded
	case dependenciesv1.EvidenceConfidence_EVIDENCE_CONFIDENCE_ADVISORY:
		return depdomain.EvidenceConfidenceAdvisory
	case dependenciesv1.EvidenceConfidence_EVIDENCE_CONFIDENCE_GATING:
		return depdomain.EvidenceConfidenceGating
	default:
		return depdomain.EvidenceConfidenceUnspecified
	}
}

func confidenceToProto(c depdomain.EvidenceConfidence) dependenciesv1.EvidenceConfidence {
	switch c {
	case depdomain.EvidenceConfidenceDegraded:
		return dependenciesv1.EvidenceConfidence_EVIDENCE_CONFIDENCE_DEGRADED
	case depdomain.EvidenceConfidenceAdvisory:
		return dependenciesv1.EvidenceConfidence_EVIDENCE_CONFIDENCE_ADVISORY
	case depdomain.EvidenceConfidenceGating:
		return dependenciesv1.EvidenceConfidence_EVIDENCE_CONFIDENCE_GATING
	default:
		return dependenciesv1.EvidenceConfidence_EVIDENCE_CONFIDENCE_UNSPECIFIED
	}
}
