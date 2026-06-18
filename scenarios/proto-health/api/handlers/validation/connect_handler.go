package validation

import (
	"context"
	"fmt"
	"log"

	"connectrpc.com/connect"

	"proto-health/internal/protosurface"
	internal "proto-health/internal/validation"

	"github.com/vrooli/maturity-go/assessment"
	commonv1 "github.com/vrooli/vrooli/packages/proto/gen/go/common/v1"
	sharedv1 "github.com/vrooli/vrooli/packages/proto/gen/go/proto-health/v1/shared"
	validationv1 "github.com/vrooli/vrooli/packages/proto/gen/go/proto-health/v1/validation"
	scenariovalidationv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-validation/v1"
)

type Validator interface {
	ValidateScenario(ctx context.Context, scenario string) (internal.Report, error)
	DescribeScenarioProtos(ctx context.Context, scenario string) (protosurface.Surface, error)
	DescribeScenariosProtos(ctx context.Context, scenarios []string, limit int32, stabilityFilter string) ([]internal.SurfaceResult, error)
}

type Deps struct {
	Logger       *log.Logger
	Validator    Validator
	MaturitySpec *assessment.Spec
}

type connectHandler struct {
	deps Deps
}

func NewConnectHandler(d Deps) *connectHandler {
	if d.Logger == nil {
		d.Logger = log.Default()
	}
	return &connectHandler{deps: d}
}

func (h *connectHandler) ValidateScenario(ctx context.Context, req *connect.Request[scenariovalidationv1.ValidateScenarioRequest]) (*connect.Response[scenariovalidationv1.ValidateScenarioResponse], error) {
	if h.deps.Validator == nil {
		return nil, connect.NewError(connect.CodeUnimplemented, fmt.Errorf("validation validator is not wired"))
	}
	report, err := h.deps.Validator.ValidateScenario(ctx, req.Msg.GetScenario())
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	maturityAssessment, err := buildMaturityAssessment(report, h.deps.MaturitySpec)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("build maturity assessment: %w", err))
	}
	resp, err := assessment.BuildValidationResponse(report.Scenario, maturityAssessment, nil)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("build shared validation response: %w", err))
	}
	return connect.NewResponse(resp), nil
}

func buildMaturityAssessment(rep internal.Report, spec *assessment.Spec) (*commonv1.MaturityAssessment, error) {
	if spec == nil {
		return nil, nil
	}
	findings := make([]assessment.Finding, 0, len(rep.Findings))
	for _, f := range rep.Findings {
		findings = append(findings, assessment.Finding{
			Code:        f.Code,
			Severity:    severityToken(f.Severity),
			Message:     f.Message,
			Location:    f.Location,
			Remediation: f.Suggestion,
			Phase:       spec.Phase,
		})
	}
	return assessment.BuildProtoAssessment(assessment.BuildInput{
		Scenario: rep.Scenario,
		Spec:     *spec,
		Findings: findings,
	})
}

func (h *connectHandler) DescribeScenarioProtos(ctx context.Context, req *connect.Request[validationv1.DescribeScenarioProtosRequest]) (*connect.Response[validationv1.DescribeScenarioProtosResponse], error) {
	if h.deps.Validator == nil {
		return nil, connect.NewError(connect.CodeUnimplemented, fmt.Errorf("validation validator is not wired"))
	}
	surface, err := h.deps.Validator.DescribeScenarioProtos(ctx, req.Msg.GetScenario())
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	return connect.NewResponse(&validationv1.DescribeScenarioProtosResponse{
		Surface: surfaceToProto(surface),
	}), nil
}

func (h *connectHandler) DescribeScenariosProtos(ctx context.Context, req *connect.Request[validationv1.DescribeScenariosProtosRequest]) (*connect.Response[validationv1.DescribeScenariosProtosResponse], error) {
	if h.deps.Validator == nil {
		return nil, connect.NewError(connect.CodeUnimplemented, fmt.Errorf("validation validator is not wired"))
	}
	results, err := h.deps.Validator.DescribeScenariosProtos(ctx, req.Msg.GetScenarios(), req.Msg.GetLimit(), req.Msg.GetStabilityFilter())
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	out := make([]*validationv1.ProtoSurfaceResult, 0, len(results))
	for _, result := range results {
		item := &validationv1.ProtoSurfaceResult{
			Scenario: result.Scenario,
			Error:    result.Error,
		}
		if result.Error == "" {
			item.Surface = surfaceToProto(result.Surface)
		}
		out = append(out, item)
	}
	return connect.NewResponse(&validationv1.DescribeScenariosProtosResponse{
		Results: out,
	}), nil
}

func severityToken(s internal.Severity) string {
	switch s {
	case internal.SeverityError:
		return "SEVERITY_ERROR"
	case internal.SeverityWarning:
		return "SEVERITY_WARNING"
	case internal.SeverityInfo:
		return "SEVERITY_INFO"
	default:
		return "SEVERITY_UNSPECIFIED"
	}
}

func surfaceToProto(in protosurface.Surface) *sharedv1.ProtoSurface {
	out := &sharedv1.ProtoSurface{
		Scenario:              in.Scenario,
		TransportWorld:        transportWorldToProto(in.TransportWorld),
		Files:                 make([]*sharedv1.ProtoFile, 0, len(in.Files)),
		Services:              make([]*sharedv1.ProtoService, 0, len(in.Services)),
		Messages:              make([]*sharedv1.ProtoMessage, 0, len(in.Messages)),
		IntraScenarioImports:  make([]*sharedv1.ProtoImport, 0, len(in.IntraScenarioImports)),
		CrossScenarioImports:  make([]*sharedv1.ProtoImport, 0, len(in.CrossScenarioImports)),
		RestExceptionRefs:     make([]*sharedv1.RestExceptionRef, 0, len(in.RESTExceptionRefs)),
		RestExceptions:        make([]*sharedv1.RestExceptionEndpoint, 0, len(in.RESTExceptions)),
		RestExceptionPayloads: make([]*sharedv1.RestExceptionPayloadRef, 0, len(in.RESTExceptionPayloads)),
	}
	for _, f := range in.Files {
		pf := &sharedv1.ProtoFile{
			Path:        f.Path,
			Package:     f.Package,
			Version:     f.Version,
			Domain:      f.Domain,
			Stability:   f.Stability,
			Annotations: make([]*sharedv1.Annotation, 0, len(f.Annotations)),
		}
		for _, a := range f.Annotations {
			pf.Annotations = append(pf.Annotations, &sharedv1.Annotation{Name: a.Name, Value: a.Value})
		}
		out.Files = append(out.Files, pf)
	}
	for _, s := range in.Services {
		ps := &sharedv1.ProtoService{
			FilePath: s.FilePath,
			Package:  s.Package,
			Name:     s.Name,
			FullName: s.FullName,
			Domain:   s.Domain,
			Rpcs:     make([]*sharedv1.ProtoRpc, 0, len(s.RPCs)),
		}
		for _, r := range s.RPCs {
			ps.Rpcs = append(ps.Rpcs, &sharedv1.ProtoRpc{
				Name:      r.Name,
				Input:     r.Input,
				Output:    r.Output,
				Transport: transportKindToProto(r.Transport),
			})
		}
		out.Services = append(out.Services, ps)
	}
	for _, m := range in.Messages {
		pm := &sharedv1.ProtoMessage{
			FilePath:   m.FilePath,
			Package:    m.Package,
			Name:       m.Name,
			FullName:   m.FullName,
			Domain:     m.Domain,
			IsMapEntry: m.IsMapEntry,
			Fields:     make([]*sharedv1.ProtoField, 0, len(m.Fields)),
		}
		for _, f := range m.Fields {
			pm.Fields = append(pm.Fields, &sharedv1.ProtoField{
				Name:        f.Name,
				Type:        f.Type,
				MessageType: f.MessageType,
				EnumType:    f.EnumType,
				Repeated:    f.Repeated,
				Optional:    f.Optional,
				Number:      f.Number,
			})
		}
		out.Messages = append(out.Messages, pm)
	}
	for _, imp := range in.IntraScenarioImports {
		out.IntraScenarioImports = append(out.IntraScenarioImports, importToProto(imp))
	}
	for _, imp := range in.CrossScenarioImports {
		out.CrossScenarioImports = append(out.CrossScenarioImports, importToProto(imp))
	}
	for _, ref := range in.RESTExceptionRefs {
		out.RestExceptionRefs = append(out.RestExceptionRefs, &sharedv1.RestExceptionRef{
			EndpointId: ref.EndpointID,
			Path:       ref.Path,
			Method:     ref.Method,
			Domain:     ref.Domain,
			Message:    ref.Message,
			FullName:   ref.FullName,
		})
	}
	for _, endpoint := range in.RESTExceptions {
		out.RestExceptions = append(out.RestExceptions, &sharedv1.RestExceptionEndpoint{
			EndpointId:             endpoint.EndpointID,
			Path:                   endpoint.Path,
			Method:                 endpoint.Method,
			Domain:                 endpoint.Domain,
			Reason:                 endpoint.Reason,
			HasPayloadDeclarations: endpoint.HasPayloadDeclarations,
		})
	}
	for _, ref := range in.RESTExceptionPayloads {
		out.RestExceptionPayloads = append(out.RestExceptionPayloads, &sharedv1.RestExceptionPayloadRef{
			EndpointId:    ref.EndpointID,
			Path:          ref.Path,
			Method:        ref.Method,
			Domain:        ref.Domain,
			Reason:        ref.Reason,
			Role:          restPayloadRoleToProto(ref.Role),
			ProtoFullName: ref.ProtoFullName,
			Transport:     ref.Transport,
			Conformance:   ref.Conformance,
			ProofStatus:   restPayloadProofStatusToProto(ref.ProofStatus),
		})
	}
	return out
}

func importToProto(imp protosurface.Import) *sharedv1.ProtoImport {
	return &sharedv1.ProtoImport{
		FromFile:     imp.FromFile,
		ToFile:       imp.ToFile,
		FromScenario: imp.FromScenario,
		ToScenario:   imp.ToScenario,
		FromPackage:  imp.FromPackage,
		ToPackage:    imp.ToPackage,
		FromVersion:  imp.FromVersion,
		ToVersion:    imp.ToVersion,
		FromDomain:   imp.FromDomain,
		ToDomain:     imp.ToDomain,
		Kind:         importKindToProto(imp.Kind),
	}
}

func importKindToProto(kind protosurface.ImportKind) sharedv1.ImportKind {
	switch kind {
	case protosurface.ImportKindScenarioLocal:
		return sharedv1.ImportKind_IMPORT_KIND_SCENARIO_LOCAL
	case protosurface.ImportKindCrossScenario:
		return sharedv1.ImportKind_IMPORT_KIND_CROSS_SCENARIO
	case protosurface.ImportKindExternal:
		return sharedv1.ImportKind_IMPORT_KIND_EXTERNAL
	default:
		return sharedv1.ImportKind_IMPORT_KIND_UNSPECIFIED
	}
}

func transportWorldToProto(world protosurface.TransportWorld) sharedv1.TransportWorld {
	switch world {
	case protosurface.TransportWorldConnect:
		return sharedv1.TransportWorld_TRANSPORT_WORLD_CONNECT
	case protosurface.TransportWorldHandRolled:
		return sharedv1.TransportWorld_TRANSPORT_WORLD_HAND_ROLLED
	case protosurface.TransportWorldMixed:
		return sharedv1.TransportWorld_TRANSPORT_WORLD_MIXED
	case protosurface.TransportWorldNone:
		return sharedv1.TransportWorld_TRANSPORT_WORLD_NONE
	default:
		return sharedv1.TransportWorld_TRANSPORT_WORLD_UNSPECIFIED
	}
}

func transportKindToProto(kind protosurface.TransportKind) sharedv1.TransportKind {
	switch kind {
	case protosurface.TransportKindConnect:
		return sharedv1.TransportKind_TRANSPORT_KIND_CONNECT
	case protosurface.TransportKindREST:
		return sharedv1.TransportKind_TRANSPORT_KIND_REST
	case protosurface.TransportKindHandRolled:
		return sharedv1.TransportKind_TRANSPORT_KIND_HAND_ROLLED
	case protosurface.TransportKindNotServed:
		return sharedv1.TransportKind_TRANSPORT_KIND_NOT_SERVED
	default:
		return sharedv1.TransportKind_TRANSPORT_KIND_UNSPECIFIED
	}
}

func restPayloadRoleToProto(role protosurface.RESTPayloadRole) sharedv1.RestPayloadRole {
	switch role {
	case protosurface.RESTPayloadRoleRequest:
		return sharedv1.RestPayloadRole_REST_PAYLOAD_ROLE_REQUEST
	case protosurface.RESTPayloadRoleResponse:
		return sharedv1.RestPayloadRole_REST_PAYLOAD_ROLE_RESPONSE
	case protosurface.RESTPayloadRoleError:
		return sharedv1.RestPayloadRole_REST_PAYLOAD_ROLE_ERROR
	default:
		return sharedv1.RestPayloadRole_REST_PAYLOAD_ROLE_UNSPECIFIED
	}
}

func restPayloadProofStatusToProto(status protosurface.RESTPayloadProofStatus) sharedv1.RestPayloadProofStatus {
	switch status {
	case protosurface.RESTPayloadProofNotEvaluated:
		return sharedv1.RestPayloadProofStatus_REST_PAYLOAD_PROOF_STATUS_NOT_EVALUATED
	default:
		return sharedv1.RestPayloadProofStatus_REST_PAYLOAD_PROOF_STATUS_UNSPECIFIED
	}
}
