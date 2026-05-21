// Package manifest is the Connect-RPC surface for the manifest domain.
// It translates between the proto wire types and the domain types
// owned by internal/manifest, applies the error-mapping policy, and
// is the only place outside internal/manifest that touches the
// generated manifest_v1 / manifest_v1connect packages.
package manifest

import (
	"context"
	"errors"
	"strings"

	"architecture-cartographer/internal/manifest"

	"connectrpc.com/connect"
	manifestv1 "github.com/vrooli/vrooli/packages/proto/gen/go/architecture-cartographer/v1/manifest"
	"github.com/vrooli/vrooli/packages/proto/gen/go/architecture-cartographer/v1/manifest/manifest_v1connect"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// Handler implements manifest_v1connect.ManifestServiceHandler over
// the internal/manifest application Service.
type Handler struct {
	manifest_v1connect.UnimplementedManifestServiceHandler
	svc manifest.Service
}

// NewHandler constructs the Connect handler for the manifest service.
func NewHandler(svc manifest.Service) *Handler {
	return &Handler{svc: svc}
}

var _ manifest_v1connect.ManifestServiceHandler = (*Handler)(nil)

// ValidateManifest parses + validates the supplied source bytes and
// persists the result on success.
func (h *Handler) ValidateManifest(ctx context.Context, req *connect.Request[manifestv1.ValidateManifestRequest]) (*connect.Response[manifestv1.ValidateManifestResponse], error) {
	in := req.Msg
	scenario := strings.TrimSpace(in.GetScenario())
	source := in.GetSource()
	if scenario == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("scenario is required"))
	}
	if len(source) == 0 {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("source is required"))
	}
	m, diags, err := h.svc.ValidateSource(ctx, scenario, source, manifest.ContentType(in.GetContentType()))
	var invalid manifest.ErrInvalidManifest
	switch {
	case err == nil:
		resp := &manifestv1.ValidateManifestResponse{
			Manifest:    manifestToProto(m),
			Diagnostics: diagnosticsToProto(diags),
			Valid:       !hasErrorDiag(diags),
		}
		return connect.NewResponse(resp), nil
	case errors.As(err, &invalid):
		// Invalid manifest is the success path of the wire shape: we
		// return diagnostics so the caller can render them. We surface
		// it as InvalidArgument so non-Connect callers still see a
		// failure status without losing the diagnostic payload.
		resp := &manifestv1.ValidateManifestResponse{
			Manifest:    manifestToProto(m),
			Diagnostics: diagnosticsToProto(diags),
			Valid:       false,
		}
		out := connect.NewResponse(resp)
		// Echo the diagnostics on the wire envelope AND return the
		// typed error so Connect clients can branch cleanly.
		return out, connect.NewError(connect.CodeInvalidArgument, err)
	default:
		return nil, connect.NewError(manifest.ErrorToConnectCode(err), err)
	}
}

func (h *Handler) GetManifest(ctx context.Context, req *connect.Request[manifestv1.GetManifestRequest]) (*connect.Response[manifestv1.GetManifestResponse], error) {
	scenario := strings.TrimSpace(req.Msg.GetScenario())
	if scenario == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("scenario is required"))
	}
	m, err := h.svc.GetManifest(ctx, scenario)
	if err != nil {
		return nil, connect.NewError(manifest.ErrorToConnectCode(err), err)
	}
	return connect.NewResponse(&manifestv1.GetManifestResponse{Manifest: manifestToProto(m)}), nil
}

func (h *Handler) ListDomains(ctx context.Context, req *connect.Request[manifestv1.ListDomainsRequest]) (*connect.Response[manifestv1.ListDomainsResponse], error) {
	scenario := strings.TrimSpace(req.Msg.GetScenario())
	if scenario == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("scenario is required"))
	}
	doms, err := h.svc.ListDomains(ctx, scenario)
	if err != nil {
		return nil, connect.NewError(manifest.ErrorToConnectCode(err), err)
	}
	out := &manifestv1.ListDomainsResponse{}
	for _, d := range doms {
		out.Domains = append(out.Domains, domainSpecToProto(d))
	}
	return connect.NewResponse(out), nil
}

func hasErrorDiag(diags []manifest.Diagnostic) bool {
	for _, d := range diags {
		if d.Severity == manifest.DiagnosticSeverityError {
			return true
		}
	}
	return false
}

func manifestToProto(m manifest.ManifestDefinition) *manifestv1.ManifestDefinition {
	out := &manifestv1.ManifestDefinition{
		ManifestVersion: versionToProto(m.Version),
		Scenario:        m.Scenario,
		SharedSubstrate: append([]string(nil), m.SharedSubstrate...),
		SignalWeights:   weightsToProto(m.SignalWeights),
		ContentHash:     m.ContentHash,
	}
	if !m.ParsedAt.IsZero() {
		out.ParsedAt = timestamppb.New(m.ParsedAt)
	}
	for _, d := range m.Domains {
		out.Domains = append(out.Domains, domainSpecToProto(d))
	}
	for _, t := range m.Thresholds {
		out.Thresholds = append(out.Thresholds, &manifestv1.Threshold{Tier: t.Tier, MinValue: t.MinValue})
	}
	for _, t := range m.Transitional {
		out.Transitional = append(out.Transitional, &manifestv1.TransitionalDeclaration{
			Id:          t.ID,
			Kind:        t.Kind,
			Locator:     t.Locator,
			Rationale:   t.Rationale,
			ExpiresWhen: t.ExpiresWhen,
		})
	}
	return out
}

func domainSpecToProto(d manifest.DomainSpec) *manifestv1.DomainSpec {
	return &manifestv1.DomainSpec{
		Name:                  d.Name,
		Paths:                 append([]string(nil), d.Paths...),
		AllowedDependencies:   append([]string(nil), d.AllowedDependencies...),
		Glossary:              append([]string(nil), d.Glossary...),
		SignalWeightOverrides: weightsToProto(d.SignalWeightOverrides),
	}
}

func weightsToProto(w manifest.SignalWeights) *manifestv1.SignalWeights {
	if len(w.Weights) == 0 {
		return nil
	}
	out := &manifestv1.SignalWeights{Weights: make(map[string]float64, len(w.Weights))}
	for k, v := range w.Weights {
		out.Weights[k] = v
	}
	return out
}

func versionToProto(v manifest.ManifestVersion) manifestv1.ManifestVersion {
	switch v {
	case manifest.ManifestVersionV1:
		return manifestv1.ManifestVersion_MANIFEST_VERSION_V1
	default:
		return manifestv1.ManifestVersion_MANIFEST_VERSION_UNSPECIFIED
	}
}

func diagnosticsToProto(in []manifest.Diagnostic) []*manifestv1.Diagnostic {
	out := make([]*manifestv1.Diagnostic, 0, len(in))
	for _, d := range in {
		out = append(out, &manifestv1.Diagnostic{
			Severity: severityToProto(d.Severity),
			Path:     d.Path,
			Line:     int32(d.Line),
			Column:   int32(d.Column),
			Message:  d.Message,
			Code:     d.Code,
		})
	}
	return out
}

func severityToProto(s manifest.DiagnosticSeverity) manifestv1.DiagnosticSeverity {
	switch s {
	case manifest.DiagnosticSeverityInfo:
		return manifestv1.DiagnosticSeverity_DIAGNOSTIC_SEVERITY_INFO
	case manifest.DiagnosticSeverityWarn:
		return manifestv1.DiagnosticSeverity_DIAGNOSTIC_SEVERITY_WARN
	case manifest.DiagnosticSeverityError:
		return manifestv1.DiagnosticSeverity_DIAGNOSTIC_SEVERITY_ERROR
	default:
		return manifestv1.DiagnosticSeverity_DIAGNOSTIC_SEVERITY_UNSPECIFIED
	}
}
