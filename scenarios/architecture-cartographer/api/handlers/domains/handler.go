// Package domains is the Connect-RPC surface for the domains domain. It
// translates between the proto wire types and the domain types owned by
// internal/domains, applies the error-mapping policy, and is the only
// place outside internal/domains that touches the generated domains_v1 /
// domains_v1connect packages.
package domains

import (
	"context"
	"errors"
	"strings"

	"architecture-cartographer/internal/domains"

	"connectrpc.com/connect"
	domainsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/architecture-cartographer/v1/domains"
	"github.com/vrooli/vrooli/packages/proto/gen/go/architecture-cartographer/v1/domains/domains_v1connect"
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
		Scenario:  m.Scenario,
		Authority: sourceToProto(m.Authority),
	}
	for _, f := range domains.Convergence(m) {
		out.Findings = append(out.Findings, convergenceToProto(f))
	}
	return connect.NewResponse(out), nil
}

func convergenceToProto(f domains.ConvergenceFinding) *domainsv1.ConvergenceFinding {
	out := &domainsv1.ConvergenceFinding{
		Kind:     f.Kind,
		Domain:   f.Domain,
		Severity: convergenceSeverityToProto(f.Severity),
		Message:  f.Message,
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
		Scenario:        m.Scenario,
		SharedSubstrate: append([]string(nil), m.SharedSubstrate...),
		NonDomains:      append([]string(nil), m.NonDomains...),
		Authority:       sourceToProto(m.Authority),
	}
	if !m.DerivedAt.IsZero() {
		out.DerivedAt = timestamppb.New(m.DerivedAt)
	}
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
		Name:      d.Name,
		Paths:     append([]string(nil), d.Paths...),
		Glossary:  append([]string(nil), d.Glossary...),
		Archetype: d.Archetype,
	}
	for _, s := range d.Provenance {
		out.Provenance = append(out.Provenance, sourceToProto(s))
	}
	return out
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
