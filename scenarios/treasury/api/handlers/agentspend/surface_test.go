package agentspend_test

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
	authorizationv1 "github.com/vrooli/vrooli/packages/proto/gen/go/treasury/v1/authorization"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protodesc"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/descriptorpb"
)

var permittedAgentSpendMethods = map[protoreflect.Name]struct{}{
	"ProposeCharge":     {},
	"GetAuthorization":  {},
	"GetBudgetHeadroom": {},
	"ListMandates":      {},
	"ReportOutcome":     {},
}

// validateAgentSpendSurface encodes Decision D-004 as an exact allowlist. A
// new convenience RPC is unsafe until this guard is deliberately reviewed;
// name heuristics would let a mutating method hide behind vague wording.
func validateAgentSpendSurface(service protoreflect.ServiceDescriptor) error {
	if service == nil || service.Name() != "AgentSpend" {
		return fmt.Errorf("AgentSpend descriptor is required")
	}
	methods := service.Methods()
	for i := 0; i < methods.Len(); i++ {
		method := methods.Get(i)
		if _, ok := permittedAgentSpendMethods[method.Name()]; !ok {
			return fmt.Errorf("AgentSpend method %s is outside the reviewed non-policy surface", method.Name())
		}
	}
	if methods.Len() != len(permittedAgentSpendMethods) {
		return fmt.Errorf("AgentSpend declares %d methods; reviewed surface requires %d", methods.Len(), len(permittedAgentSpendMethods))
	}
	return nil
}

// [REQ:TRS-P0-004] The generated descriptor, rather than a hand-maintained
// method list in production code, proves that policy mutation is absent.
func TestGeneratedAgentSpendDescriptorHasOnlyReviewedMethods(t *testing.T) {
	service := authorizationv1.File_treasury_v1_authorization_authorization_proto.Services().ByName("AgentSpend")
	require.NoError(t, validateAgentSpendSurface(service))
}

// This mutation test proves the guard is live: adding SetGating to an otherwise
// valid AgentSpend descriptor fails the same validator used above.
func TestAgentSpendDescriptorGuardRejectsPolicyMutation(t *testing.T) {
	methodNames := []string{"ProposeCharge", "GetAuthorization", "ListMandates", "ReportOutcome", "SetGating"}
	methods := make([]*descriptorpb.MethodDescriptorProto, 0, len(methodNames))
	for _, name := range methodNames {
		methods = append(methods, &descriptorpb.MethodDescriptorProto{Name: proto.String(name), InputType: proto.String(".test.Request"), OutputType: proto.String(".test.Response")})
	}
	file, err := protodesc.NewFile(&descriptorpb.FileDescriptorProto{
		Name:    proto.String("test/agent_spend.proto"),
		Package: proto.String("test"),
		Syntax:  proto.String("proto3"),
		MessageType: []*descriptorpb.DescriptorProto{
			{Name: proto.String("Request")},
			{Name: proto.String("Response")},
		},
		Service: []*descriptorpb.ServiceDescriptorProto{{Name: proto.String("AgentSpend"), Method: methods}},
	}, nil)
	require.NoError(t, err)
	require.ErrorContains(t, validateAgentSpendSurface(file.Services().ByName("AgentSpend")), "SetGating")
}
