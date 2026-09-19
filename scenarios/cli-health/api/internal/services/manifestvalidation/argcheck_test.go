package manifestvalidation

import (
	"testing"

	"github.com/vrooli/cli-core/cliapp"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protodesc"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/descriptorpb"
)

func testRequestDescriptor(t *testing.T) protoreflect.MessageDescriptor {
	t.Helper()
	file, err := protodesc.NewFile(&descriptorpb.FileDescriptorProto{
		Syntax:  proto.String("proto3"),
		Name:    proto.String("fixture.proto"),
		Package: proto.String("fixture.v1"),
		MessageType: []*descriptorpb.DescriptorProto{{
			Name: proto.String("Request"),
			Field: []*descriptorpb.FieldDescriptorProto{{
				Name:     proto.String("known"),
				JsonName: proto.String("known"),
				Number:   proto.Int32(1),
				Label:    descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(),
				Type:     descriptorpb.FieldDescriptorProto_TYPE_STRING.Enum(),
			}},
		}},
	}, nil)
	if err != nil {
		t.Fatalf("build fixture descriptor: %v", err)
	}
	return file.Messages().ByName("Request")
}

func testBindingManifest(command cliapp.ManifestCommand) *cliapp.Manifest {
	return &cliapp.Manifest{
		Name: "fixture",
		Groups: []cliapp.ManifestGroup{{
			Name:     "g1",
			Commands: []cliapp.ManifestCommand{command},
		}},
	}
}

func testConnectCommand() cliapp.ManifestCommand {
	return cliapp.ManifestCommand{
		Name:       "do",
		Binding:    cliapp.ManifestBinding{Kind: "connect-rpc", Service: "Svc", Method: "Do"},
		Governance: cliapp.ManifestGovernance{Effect: "read", RunEligible: true},
	}
}

func TestArgumentFindingsReportsSeededUnmappedArgument(t *testing.T) {
	command := testConnectCommand()
	command.Flags = []cliapp.ManifestFlag{{Name: "unknown", Required: true}}
	manifest := testBindingManifest(command)
	surface := ProtoSurface{
		Requests: map[string]protoreflect.MessageDescriptor{"Svc.Do": testRequestDescriptor(t)},
	}

	findings := argumentFindings(manifest, surface, "cli/manifest.json")
	if len(findings) != 1 {
		t.Fatalf("expected one finding, got %+v", findings)
	}
	if findings[0].Code != CodeBindingArgUnmapped || findings[0].Severity != SeverityError {
		t.Fatalf("expected binding.arg_unmapped error, got %+v", findings[0])
	}
}

func TestArgumentFindingsReportsSeededAmbiguousService(t *testing.T) {
	manifest := testBindingManifest(testConnectCommand())
	surface := ProtoSurface{
		RequestCandidates: map[string][]ProtoRequestCandidate{
			"Svc.Do": {
				{Source: "fixture/one.proto"},
				{Source: "fixture/two.proto"},
			},
		},
	}

	findings := argumentFindings(manifest, surface, "cli/manifest.json")
	if len(findings) != 1 {
		t.Fatalf("expected one finding, got %+v", findings)
	}
	if findings[0].Code != CodeBindingAmbiguousSvc || findings[0].Severity != SeverityError {
		t.Fatalf("expected binding.ambiguous_service error, got %+v", findings[0])
	}
}
