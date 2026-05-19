package schema

import (
	"context"
	"encoding/json"

	"connectrpc.com/connect"
	"google.golang.org/protobuf/types/known/structpb"

	"github.com/vrooli/browser-automation-studio/workflow/validator"
	schemav1 "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/schema"
)

// service implements schemaconnect.SchemaServiceHandler.
type service struct {
	deps Deps
}

func (s *service) GetWorkflowSchema(
	_ context.Context,
	req *connect.Request[schemav1.GetWorkflowSchemaRequest],
) (*connect.Response[schemav1.GetWorkflowSchemaResponse], error) {
	nodeTypes := req.Msg.GetNodeTypes()

	var (
		raw json.RawMessage
		err error
	)
	if len(nodeTypes) == 0 {
		raw, err = s.deps.Provider.GetFullSchema()
	} else {
		raw, err = s.deps.Provider.GetFilteredSchema(nodeTypes)
	}
	if err != nil {
		s.deps.Logger.WithError(err).Error("schema.GetWorkflowSchema failed")
		return nil, connect.NewError(connect.CodeInternal, errSchemaUnavailable)
	}

	var generic map[string]any
	if err := json.Unmarshal(raw, &generic); err != nil {
		s.deps.Logger.WithError(err).Error("schema.GetWorkflowSchema: schema is not a JSON object")
		return nil, connect.NewError(connect.CodeInternal, errSchemaUnavailable)
	}
	st, err := structpb.NewStruct(generic)
	if err != nil {
		s.deps.Logger.WithError(err).Error("schema.GetWorkflowSchema: structpb conversion failed")
		return nil, connect.NewError(connect.CodeInternal, errSchemaUnavailable)
	}
	return connect.NewResponse(&schemav1.GetWorkflowSchemaResponse{Schema: st}), nil
}

func (s *service) GetNodeTypes(
	_ context.Context,
	_ *connect.Request[schemav1.GetNodeTypesRequest],
) (*connect.Response[schemav1.GetNodeTypesResponse], error) {
	types := s.deps.Provider.AvailableNodeTypes()
	return connect.NewResponse(&schemav1.GetNodeTypesResponse{NodeTypes: types}), nil
}

func (s *service) GetStepDefinitions(
	_ context.Context,
	req *connect.Request[schemav1.GetStepDefinitionsRequest],
) (*connect.Response[schemav1.GetStepDefinitionsResponse], error) {
	defs := s.deps.Provider.StepDefinitions(req.Msg.GetCliOnly())
	filter := req.Msg.GetTypes()
	if len(filter) > 0 {
		set := make(map[string]bool, len(filter))
		for _, t := range filter {
			set[t] = true
		}
		filtered := make([]validator.StepDefinition, 0, len(defs))
		for _, d := range defs {
			if set[d.Type] {
				filtered = append(filtered, d)
			}
		}
		defs = filtered
	}
	out := make([]*schemav1.StepDefinition, 0, len(defs))
	for _, d := range defs {
		out = append(out, toProtoStep(d))
	}
	return connect.NewResponse(&schemav1.GetStepDefinitionsResponse{Steps: out}), nil
}

func toProtoStep(d validator.StepDefinition) *schemav1.StepDefinition {
	out := &schemav1.StepDefinition{
		Type:         d.Type,
		Description:  d.Description,
		CliSupported: d.CLISupported,
	}
	if d.Positional != nil {
		out.Positional = &schemav1.StepPositional{
			Name:        d.Positional.Name,
			MapsTo:      d.Positional.MapsTo,
			Description: d.Positional.Description,
		}
	}
	for _, kv := range d.RequiredKVs {
		out.RequiredKvs = append(out.RequiredKvs, &schemav1.StepKV{
			Key:         kv.Key,
			Type:        kv.Type,
			Description: kv.Description,
		})
	}
	for _, kv := range d.OptionalKVs {
		out.OptionalKvs = append(out.OptionalKvs, &schemav1.StepKV{
			Key:         kv.Key,
			Type:        kv.Type,
			Description: kv.Description,
		})
	}
	for _, group := range d.RequireOneOf {
		out.RequireOneOf = append(out.RequireOneOf, &schemav1.StepRequireOneOf{Keys: group})
	}
	for _, ex := range d.Examples {
		out.Examples = append(out.Examples, &schemav1.StepExample{
			Description: ex.Description,
			Cli:         ex.CLI,
		})
	}
	return out
}
