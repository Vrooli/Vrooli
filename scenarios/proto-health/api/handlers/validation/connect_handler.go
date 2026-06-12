package validation

import (
	"context"
	"fmt"
	"log"

	"connectrpc.com/connect"

	"proto-health/internal/protosurface"
	internal "proto-health/internal/validation"

	sharedv1 "github.com/vrooli/vrooli/packages/proto/gen/go/proto-health/v1/shared"
	validationv1 "github.com/vrooli/vrooli/packages/proto/gen/go/proto-health/v1/validation"
)

type Validator interface {
	ValidateScenario(ctx context.Context, scenario string) (internal.Report, error)
	DescribeScenarioProtos(ctx context.Context, scenario string) (protosurface.Surface, error)
}

type Deps struct {
	Logger    *log.Logger
	Validator Validator
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

func (h *connectHandler) ValidateScenario(ctx context.Context, req *connect.Request[validationv1.ValidateScenarioRequest]) (*connect.Response[validationv1.ValidateScenarioResponse], error) {
	if h.deps.Validator == nil {
		return nil, connect.NewError(connect.CodeUnimplemented, fmt.Errorf("validation validator is not wired"))
	}
	report, err := h.deps.Validator.ValidateScenario(ctx, req.Msg.GetScenario())
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	return connect.NewResponse(&validationv1.ValidateScenarioResponse{
		Scenario: report.Scenario,
		Passed:   report.Passed,
		Findings: findingsToProto(report.Findings),
		Summary: &validationv1.Summary{
			Errors:   int32(report.Summary.Errors),
			Warnings: int32(report.Summary.Warnings),
			Infos:    int32(report.Summary.Infos),
		},
	}), nil
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

func findingsToProto(in []internal.Finding) []*validationv1.Finding {
	out := make([]*validationv1.Finding, 0, len(in))
	for _, f := range in {
		out = append(out, &validationv1.Finding{
			Severity:   severityToProto(f.Severity),
			Code:       f.Code,
			Location:   f.Location,
			Message:    f.Message,
			Suggestion: f.Suggestion,
		})
	}
	return out
}

func severityToProto(s internal.Severity) validationv1.Severity {
	switch s {
	case internal.SeverityError:
		return validationv1.Severity_SEVERITY_ERROR
	case internal.SeverityWarning:
		return validationv1.Severity_SEVERITY_WARNING
	case internal.SeverityInfo:
		return validationv1.Severity_SEVERITY_INFO
	default:
		return validationv1.Severity_SEVERITY_UNSPECIFIED
	}
}

func surfaceToProto(in protosurface.Surface) *sharedv1.ProtoSurface {
	out := &sharedv1.ProtoSurface{
		Scenario:             in.Scenario,
		TransportWorld:       transportWorldToProto(in.TransportWorld),
		Files:                make([]*sharedv1.ProtoFile, 0, len(in.Files)),
		Services:             make([]*sharedv1.ProtoService, 0, len(in.Services)),
		Messages:             make([]*sharedv1.ProtoMessage, 0, len(in.Messages)),
		IntraScenarioImports: make([]*sharedv1.ProtoImport, 0, len(in.IntraScenarioImports)),
		CrossScenarioImports: make([]*sharedv1.ProtoImport, 0, len(in.CrossScenarioImports)),
		AdoptionSignals:      make([]*sharedv1.AdoptionSignal, 0, len(in.AdoptionSignals)),
		RestExceptionRefs:    make([]*sharedv1.RestExceptionRef, 0, len(in.RESTExceptionRefs)),
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
			FilePath: m.FilePath,
			Package:  m.Package,
			Name:     m.Name,
			FullName: m.FullName,
			Domain:   m.Domain,
			Fields:   make([]*sharedv1.ProtoField, 0, len(m.Fields)),
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
	for _, sig := range in.AdoptionSignals {
		out.AdoptionSignals = append(out.AdoptionSignals, &sharedv1.AdoptionSignal{Name: sig.Name, Present: sig.Present, Detail: sig.Detail})
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
	return out
}

func importToProto(imp protosurface.Import) *sharedv1.ProtoImport {
	return &sharedv1.ProtoImport{
		FromFile:   imp.FromFile,
		ToFile:     imp.ToFile,
		FromDomain: imp.FromDomain,
		ToDomain:   imp.ToDomain,
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
