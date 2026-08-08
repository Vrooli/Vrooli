package manifestvalidation

import (
	"testing"

	validate "buf.build/gen/go/bufbuild/protovalidate/protocolbuffers/go/buf/validate"
	"github.com/vrooli/cli-core/cliapp"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protodesc"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/descriptorpb"
)

func semanticRequestDescriptor(t *testing.T) protoreflect.MessageDescriptor {
	t.Helper()
	requiredOptions := &descriptorpb.FieldOptions{}
	proto.SetExtension(requiredOptions, validate.E_Field, &validate.FieldRules{
		Type: &validate.FieldRules_Bytes{Bytes: &validate.BytesRules{MinLen: proto.Uint64(1)}},
	})
	file, err := protodesc.NewFile(&descriptorpb.FileDescriptorProto{
		Syntax:  proto.String("proto3"),
		Name:    proto.String("semantic-fixture.proto"),
		Package: proto.String("fixture.v1"),
		MessageType: []*descriptorpb.DescriptorProto{
			{
				Name: proto.String("Profile"),
				Field: []*descriptorpb.FieldDescriptorProto{
					{Name: proto.String("device_name"), JsonName: proto.String("deviceName"), Number: proto.Int32(1), Label: descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(), Type: descriptorpb.FieldDescriptorProto_TYPE_STRING.Enum()},
				},
			},
			{
				Name: proto.String("Request"),
				Field: []*descriptorpb.FieldDescriptorProto{
					{Name: proto.String("shared"), JsonName: proto.String("shared"), Number: proto.Int32(1), Label: descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(), Type: descriptorpb.FieldDescriptorProto_TYPE_STRING.Enum()},
					{Name: proto.String("required_payload"), JsonName: proto.String("requiredPayload"), Number: proto.Int32(2), Label: descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(), Type: descriptorpb.FieldDescriptorProto_TYPE_BYTES.Enum(), Options: requiredOptions},
					{Name: proto.String("profile"), JsonName: proto.String("profile"), Number: proto.Int32(3), Label: descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(), Type: descriptorpb.FieldDescriptorProto_TYPE_MESSAGE.Enum(), TypeName: proto.String(".fixture.v1.Profile")},
					{Name: proto.String("entries"), JsonName: proto.String("entries"), Number: proto.Int32(4), Label: descriptorpb.FieldDescriptorProto_LABEL_REPEATED.Enum(), Type: descriptorpb.FieldDescriptorProto_TYPE_MESSAGE.Enum(), TypeName: proto.String(".fixture.v1.Profile")},
				},
			},
		},
	}, nil)
	if err != nil {
		t.Fatalf("build semantic descriptor: %v", err)
	}
	return file.Messages().ByName("Request")
}

func semanticCommand(args []cliapp.ManifestFlag) cliapp.ManifestCommand {
	return cliapp.ManifestCommand{
		Name:       "check",
		Flags:      args,
		Binding:    cliapp.ManifestBinding{Kind: "connect-rpc", Service: "Svc", Method: "Do"},
		Governance: cliapp.ManifestGovernance{Effect: "read", RunEligible: true},
	}
}

func semanticFindingsFor(t *testing.T, command cliapp.ManifestCommand) []Finding {
	t.Helper()
	manifest := testBindingManifest(command)
	return semanticFindings(manifest, ProtoSurface{Requests: map[string]protoreflect.MessageDescriptor{
		"Svc.Do": semanticRequestDescriptor(t),
	}}, "cli/manifest.json")
}

func requireSemanticCode(t *testing.T, findings []Finding, code string, severity Severity) {
	t.Helper()
	for _, finding := range findings {
		if finding.Code == code {
			if finding.Severity != severity || finding.Suggestion == "" {
				t.Fatalf("finding %q has severity=%q suggestion=%q", code, finding.Severity, finding.Suggestion)
			}
			return
		}
	}
	t.Fatalf("missing semantic finding %q in %+v", code, findings)
}

func TestSemanticFindingsRejectEachSeededViolation(t *testing.T) {
	tests := []struct {
		name     string
		command  cliapp.ManifestCommand
		code     string
		severity Severity
	}{
		{
			name: "field_collision",
			command: semanticCommand([]cliapp.ManifestFlag{
				{Name: "first", Bind: &cliapp.ManifestFlagBind{Field: "shared"}},
				{Name: "second", Bind: &cliapp.ManifestFlagBind{Field: "shared"}},
			}),
			code: CodeBindingFieldCollision, severity: SeverityError,
		},
		{
			name: "control_flag_bound",
			command: semanticCommand([]cliapp.ManifestFlag{
				{Name: "json", Bind: &cliapp.ManifestFlagBind{Field: "shared"}},
			}),
			code: CodeBindingControlFlagBound, severity: SeverityError,
		},
		{
			name: "required_field_unpopulated",
			command: semanticCommand([]cliapp.ManifestFlag{
				{Name: "name", Bind: &cliapp.ManifestFlagBind{Field: "shared"}},
			}),
			code: CodeBindingRequiredFieldUnpopulated, severity: SeverityError,
		},
		{
			name: "bind_where_rename_suffices",
			command: semanticCommand([]cliapp.ManifestFlag{
				{Name: "shared", Bind: &cliapp.ManifestFlagBind{Field: "shared"}},
			}),
			code: CodeBindingBindWhereRenameSuffices, severity: SeverityWarning,
		},
		{
			// The device-sync-hub shape: a name string bound to the whole
			// profile message. It resolves, so callability gates pass.
			name: "scalar_bound_to_singular_message",
			command: semanticCommand([]cliapp.ManifestFlag{
				{Name: "name", Bind: &cliapp.ManifestFlagBind{Field: "profile"}},
			}),
			code: CodeBindingScalarBoundToMessage, severity: SeverityError,
		},
		{
			// One CLI value can never construct a list of messages.
			name: "scalar_bound_to_repeated_message",
			command: semanticCommand([]cliapp.ManifestFlag{
				{Name: "from-audit", Bind: &cliapp.ManifestFlagBind{Field: "entries"}},
			}),
			code: CodeBindingScalarBoundToMessage, severity: SeverityError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			requireSemanticCode(t, semanticFindingsFor(t, tt.command), tt.code, tt.severity)
		})
	}
}

func TestSemanticFindingsAcceptCorrectManifest(t *testing.T) {
	findings := semanticFindingsFor(t, semanticCommand([]cliapp.ManifestFlag{
		{Name: "payload", Bind: &cliapp.ManifestFlagBind{Field: "required_payload"}},
	}))
	for _, finding := range findings {
		if finding.Code == CodeBindingFieldCollision || finding.Code == CodeBindingControlFlagBound || finding.Code == CodeBindingRequiredFieldUnpopulated || finding.Code == CodeBindingBindWhereRenameSuffices {
			t.Fatalf("correct manifest emitted semantic finding: %+v", finding)
		}
	}
}

// A structured decoder is the sanctioned way to populate a message field, so
// json_file and json_inline must not trip the new rule. Auto-descent onto a
// scalar inside the envelope must not trip it either.
func TestSemanticFindingsAcceptStructuredDecoderAndEnvelopeDescent(t *testing.T) {
	findings := semanticFindingsFor(t, semanticCommand([]cliapp.ManifestFlag{
		{Name: "profile-file", Bind: &cliapp.ManifestFlagBind{Field: "profile", Kind: "json_file"}},
		{Name: "entries-json", Bind: &cliapp.ManifestFlagBind{Field: "entries", Kind: "json_inline"}},
		{Name: "device-name"},
	}))
	for _, finding := range findings {
		if finding.Code == CodeBindingScalarBoundToMessage {
			t.Fatalf("structured decoder or envelope descent emitted the rule: %+v", finding)
		}
	}
}

func TestSemanticFindingsScalarBoundToMessageWaiverSuppresses(t *testing.T) {
	findings := semanticFindingsFor(t, semanticCommand([]cliapp.ManifestFlag{
		{Name: "name", Bind: &cliapp.ManifestFlagBind{Field: "profile"}, BindWaiver: "the server accepts a bare name for this legacy RPC"},
	}))
	for _, finding := range findings {
		if finding.Code == CodeBindingScalarBoundToMessage {
			t.Fatalf("waived structured bind still emitted: %+v", finding)
		}
	}
}

func TestSemanticFindingsWaiverSuppressesIntentionalSemanticRule(t *testing.T) {
	findings := semanticFindingsFor(t, semanticCommand([]cliapp.ManifestFlag{
		{Name: "json", Bind: &cliapp.ManifestFlagBind{Field: "shared"}, BindWaiver: "json is request data for this RPC"},
		{Name: "shared", Bind: &cliapp.ManifestFlagBind{Field: "shared"}, BindWaiver: "shared is decoded with the explicit file-aware binding"},
	}))
	for _, finding := range findings {
		if finding.Code == CodeBindingControlFlagBound || finding.Code == CodeBindingBindWhereRenameSuffices {
			t.Fatalf("waived semantic bind still emitted: %+v", finding)
		}
	}
}
