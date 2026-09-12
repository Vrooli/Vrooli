// Package domains is the Connect-RPC surface for the domains domain. It
// translates between the proto wire types and the domain types owned by
// internal/domains, applies the error-mapping policy, and is the only
// place outside internal/domains that touches the generated domains_v1 /
// domains_v1connect packages.
package domains

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"architecture-cartographer/internal/archetype"
	"architecture-cartographer/internal/attest"
	"architecture-cartographer/internal/domains"

	"connectrpc.com/connect"
	domainsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/architecture-cartographer/v1/domains"
	"github.com/vrooli/vrooli/packages/proto/gen/go/architecture-cartographer/v1/domains/domains_v1connect"
	commonv1 "github.com/vrooli/vrooli/packages/proto/gen/go/common/v1"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// Handler implements domains_v1connect.DomainsServiceHandler over the
// internal/domains application Service.
type Handler struct {
	domains_v1connect.UnimplementedDomainsServiceHandler
	svc domains.Service
}

// NewHandler constructs the Connect handler for the domains service.
func NewHandler(svc domains.Service) *Handler {
	return &Handler{svc: svc}
}

var _ domains_v1connect.DomainsServiceHandler = (*Handler)(nil)

// ExtractDomains derives the domain map fresh from the scenario's on-disk
// sources.
func (h *Handler) ExtractDomains(ctx context.Context, req *connect.Request[domainsv1.ExtractDomainsRequest]) (*connect.Response[domainsv1.ExtractDomainsResponse], error) {
	scenario := strings.TrimSpace(req.Msg.GetScenario())
	if scenario == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("scenario is required"))
	}
	m, err := h.svc.ExtractDomains(ctx, scenario)
	if err != nil {
		return nil, connect.NewError(domains.ErrorToConnectCode(err), err)
	}
	return connect.NewResponse(&domainsv1.ExtractDomainsResponse{DomainMap: mapToProto(m)}), nil
}

// GetDomainMap returns the derived domain map for a scenario.
func (h *Handler) GetDomainMap(ctx context.Context, req *connect.Request[domainsv1.GetDomainMapRequest]) (*connect.Response[domainsv1.GetDomainMapResponse], error) {
	scenario := strings.TrimSpace(req.Msg.GetScenario())
	if scenario == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("scenario is required"))
	}
	m, err := h.svc.GetDomainMap(ctx, scenario)
	if err != nil {
		return nil, connect.NewError(domains.ErrorToConnectCode(err), err)
	}
	return connect.NewResponse(&domainsv1.GetDomainMapResponse{DomainMap: mapToProto(m)}), nil
}

// ConvergenceReport derives the map and reports cross-surface disagreements.
func (h *Handler) ConvergenceReport(ctx context.Context, req *connect.Request[domainsv1.ConvergenceReportRequest]) (*connect.Response[domainsv1.ConvergenceReportResponse], error) {
	scenario := strings.TrimSpace(req.Msg.GetScenario())
	if scenario == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("scenario is required"))
	}
	m, err := h.svc.GetDomainMap(ctx, scenario)
	if err != nil {
		return nil, connect.NewError(domains.ErrorToConnectCode(err), err)
	}
	out := &domainsv1.ConvergenceReportResponse{
		Scenario:            m.Scenario,
		Authority:           sourceToProto(m.Authority),
		AuthorityConfidence: confidenceToProto(m.AuthorityConfidence),
	}
	for _, f := range domains.Convergence(m) {
		out.Findings = append(out.Findings, convergenceToProto(f))
	}
	return connect.NewResponse(out), nil
}

// DraftDomains proposes a DOMAINS.md inventory without writing it.
func (h *Handler) DraftDomains(ctx context.Context, req *connect.Request[domainsv1.DraftDomainsRequest]) (*connect.Response[domainsv1.DraftDomainsResponse], error) {
	scenario := strings.TrimSpace(req.Msg.GetScenario())
	if scenario == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("scenario is required"))
	}
	draft, err := h.svc.DraftDomains(ctx, scenario)
	if err != nil {
		return nil, connect.NewError(domains.ErrorToConnectCode(err), err)
	}
	out := &domainsv1.DraftDomainsResponse{
		Scenario: draft.Scenario,
		Markdown: draft.Markdown,
	}
	for _, d := range draft.Domains {
		out.Domains = append(out.Domains, proposedDomainToProto(d))
	}
	return connect.NewResponse(out), nil
}

func proposedDomainToProto(d domains.ProposedDomain) *domainsv1.ProposedDomain {
	return &domainsv1.ProposedDomain{
		Name:       d.Name,
		Paths:      append([]string(nil), d.Paths...),
		Archetype:  primaryArchetypeName(d.Archetypes),
		Glossary:   append([]string(nil), d.Glossary...),
		Confidence: d.Confidence,
		Evidence:   append([]string(nil), d.Evidence...),
	}
}

func convergenceToProto(f domains.ConvergenceFinding) *domainsv1.ConvergenceFinding {
	out := &domainsv1.ConvergenceFinding{
		Kind:            f.Kind,
		Domain:          f.Domain,
		Severity:        convergenceSeverityToProto(f.Severity),
		Message:         f.Message,
		RolledUpDomains: append([]string(nil), f.RolledUpDomains...),
	}
	for _, s := range f.Sources {
		out.Sources = append(out.Sources, sourceToProto(s))
	}
	return out
}

func convergenceSeverityToProto(s domains.ConvergenceSeverity) domainsv1.ConvergenceSeverity {
	switch s {
	case domains.ConvergenceWarn:
		return domainsv1.ConvergenceSeverity_CONVERGENCE_SEVERITY_WARN
	case domains.ConvergenceInfo:
		return domainsv1.ConvergenceSeverity_CONVERGENCE_SEVERITY_INFO
	default:
		return domainsv1.ConvergenceSeverity_CONVERGENCE_SEVERITY_UNSPECIFIED
	}
}

func mapToProto(m domains.DerivedDomainMap) *domainsv1.DerivedDomainMap {
	out := &domainsv1.DerivedDomainMap{
		Scenario:            m.Scenario,
		SharedSubstrate:     append([]string(nil), m.SharedSubstrate...),
		NonDomains:          append([]string(nil), m.NonDomains...),
		Authority:           sourceToProto(m.Authority),
		AuthorityConfidence: confidenceToProto(m.AuthorityConfidence),
	}
	if !m.DerivedAt.IsZero() {
		out.DerivedAt = timestamppb.New(m.DerivedAt)
	}
	out.Attestation = mapAttestation(m)
	for _, d := range m.Domains {
		out.Domains = append(out.Domains, domainToProto(d))
	}
	for _, decl := range m.Declarations {
		out.Declarations = append(out.Declarations, &domainsv1.DomainDeclaration{
			Source:        sourceToProto(decl.Source),
			DomainNames:   append([]string(nil), decl.DomainNames...),
			Authoritative: decl.Authoritative,
		})
	}
	return out
}

func domainToProto(d domains.DerivedDomain) *domainsv1.DerivedDomain {
	out := &domainsv1.DerivedDomain{
		Name:            d.Name,
		Paths:           append([]string(nil), d.Paths...),
		Glossary:        append([]string(nil), d.Glossary...),
		Responsibility:  d.Responsibility,
		Purpose:         d.Purpose,
		OwnsData:        d.OwnsData,
		SecondaryTraits: append([]string(nil), d.SecondaryTraits...),
		Surfaces:        append([]string(nil), d.Surfaces...),
	}
	for _, archetype := range d.Archetypes {
		out.Archetypes = append(out.Archetypes, archetypeToProto(archetype))
	}
	for _, s := range d.Provenance {
		out.Provenance = append(out.Provenance, sourceToProto(s))
	}
	out.Attestation = domainAttestation(d)
	return out
}

// mapAttestation is the map-level Q2 honesty contract: the resolved domain map
// is DERIVED from on-disk sources; sufficiency tracks how curated the authority
// is (HIGH authority -> FULL, otherwise the ground truth is itself inferred ->
// PARTIAL).
func mapAttestation(m domains.DerivedDomainMap) *commonv1.AttestedAnswer {
	suff := commonv1.Sufficiency_SUFFICIENCY_PARTIAL
	if m.AuthorityConfidence == domains.ConfidenceHigh {
		suff = commonv1.Sufficiency_SUFFICIENCY_FULL
	}
	b := attest.New(fmt.Sprintf("domain map for %q: %d domain(s) resolved from %s", m.Scenario, len(m.Domains), m.Authority)).
		Basis(commonv1.Basis_BASIS_DERIVED).
		Sufficiency(suff)
	if m.Authority == domains.SourceDomainsDoc {
		b.CiteDoc(domains.DomainsDocPath, "authority rung")
	} else {
		b.CiteCode(string(m.Authority), "authority rung (derived source)")
	}
	if m.AuthorityConfidence != domains.ConfidenceHigh {
		b.Gap("authority fell back to a derived source; the resolved set is itself inferred")
		b.FollowUp("author docs/concepts/DOMAINS.md to raise the map to a curated authority")
	}
	a := b.Build()
	if attest.Validate(a) != nil {
		a.Basis = commonv1.Basis_BASIS_ABSENT
	}
	return a
}

// domainAttestation is the per-domain Q2 honesty contract: VALIDATED when the
// domain is both declared in DOMAINS.md and present in code, DERIVED when only
// code implies it, DECLARED_UNVERIFIED when only the doc declares it.
func domainAttestation(d domains.DerivedDomain) *commonv1.AttestedAnswer {
	hasDoc, hasCode := false, false
	for _, s := range d.Provenance {
		switch s {
		case domains.SourceDomainsDoc, domains.SourceAPIManifest:
			hasDoc = true
		case domains.SourceAPIFolders, domains.SourceCLIGroups, domains.SourceUIFeatures:
			hasCode = true
		}
	}
	basis := attest.ConvergenceBasis(hasCode, hasDoc, hasCode && hasDoc)
	suff := commonv1.Sufficiency_SUFFICIENCY_PARTIAL
	if hasDoc && strings.TrimSpace(d.Responsibility) != "" {
		suff = commonv1.Sufficiency_SUFFICIENCY_FULL
	}
	b := attest.New(fmt.Sprintf("domain %q owns %d path(s)", d.Name, len(d.Paths))).
		Basis(basis).
		Sufficiency(suff)
	if hasDoc {
		b.CiteDoc(domains.DomainsDocPath, "declared in the Domain Inventory")
	}
	for _, p := range d.Paths {
		b.CiteCode(p, "")
	}
	a := b.Build()
	if attest.Validate(a) != nil {
		a.Basis = commonv1.Basis_BASIS_ABSENT
	}
	return a
}

func archetypeToProto(a domains.DomainArchetype) *domainsv1.DomainArchetype {
	return &domainsv1.DomainArchetype{
		Archetype:     archetypeNameToProto(a.Name),
		Source:        archetypeSourceToProto(a.Source),
		Confidence:    a.Confidence,
		Evidence:      append([]string(nil), a.Evidence...),
		DeclaredLabel: a.DeclaredLabel,
	}
}

func archetypeNameToProto(name string) domainsv1.Archetype {
	switch archetype.Name(name) {
	case archetype.Reporting:
		return domainsv1.Archetype_ARCHETYPE_REPORTING
	case archetype.Service:
		return domainsv1.Archetype_ARCHETYPE_SERVICE
	case archetype.Mutation:
		return domainsv1.Archetype_ARCHETYPE_MUTATION
	case archetype.Classification:
		return domainsv1.Archetype_ARCHETYPE_CLASSIFICATION
	case archetype.Orchestration:
		return domainsv1.Archetype_ARCHETYPE_ORCHESTRATION
	case archetype.Scoring:
		return domainsv1.Archetype_ARCHETYPE_SCORING
	case archetype.Query:
		return domainsv1.Archetype_ARCHETYPE_QUERY
	default:
		return domainsv1.Archetype_ARCHETYPE_UNSPECIFIED
	}
}

func archetypeSourceToProto(s domains.ArchetypeSource) domainsv1.ArchetypeSource {
	switch s {
	case domains.ArchetypeSourceDeclared:
		return domainsv1.ArchetypeSource_ARCHETYPE_SOURCE_DECLARED
	case domains.ArchetypeSourceInferred:
		return domainsv1.ArchetypeSource_ARCHETYPE_SOURCE_INFERRED
	default:
		return domainsv1.ArchetypeSource_ARCHETYPE_SOURCE_UNSPECIFIED
	}
}

func primaryArchetypeName(archetypes []domains.DomainArchetype) string {
	if len(archetypes) == 0 {
		return ""
	}
	for _, archetype := range archetypes {
		if archetype.Source == domains.ArchetypeSourceDeclared && archetype.Name != "" {
			return archetype.Name
		}
	}
	return archetypes[0].Name
}

func confidenceToProto(c domains.AuthorityConfidence) domainsv1.AuthorityConfidence {
	switch c {
	case domains.ConfidenceHigh:
		return domainsv1.AuthorityConfidence_AUTHORITY_CONFIDENCE_HIGH
	case domains.ConfidenceLow:
		return domainsv1.AuthorityConfidence_AUTHORITY_CONFIDENCE_LOW
	default:
		return domainsv1.AuthorityConfidence_AUTHORITY_CONFIDENCE_UNSPECIFIED
	}
}

func sourceToProto(s domains.Source) domainsv1.DomainSource {
	switch s {
	case domains.SourceAPIManifest:
		return domainsv1.DomainSource_DOMAIN_SOURCE_API_MANIFEST
	case domains.SourceDomainsDoc:
		return domainsv1.DomainSource_DOMAIN_SOURCE_DOMAINS_DOC
	case domains.SourceAPIFolders:
		return domainsv1.DomainSource_DOMAIN_SOURCE_API_FOLDERS
	case domains.SourceCLIGroups:
		return domainsv1.DomainSource_DOMAIN_SOURCE_CLI_GROUPS
	case domains.SourceUIFeatures:
		return domainsv1.DomainSource_DOMAIN_SOURCE_UI_FEATURES
	default:
		return domainsv1.DomainSource_DOMAIN_SOURCE_UNSPECIFIED
	}
}
