package cliapp

import (
	"errors"
	"path/filepath"
	"strings"
	"testing"

	"github.com/vrooli/vrooli/packages/proto/descriptorimage"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protodesc"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"
	"google.golang.org/protobuf/types/descriptorpb"
	_ "google.golang.org/protobuf/types/known/structpb"
	_ "google.golang.org/protobuf/types/known/timestamppb"
)

func testMessageDescriptor(t *testing.T) protoreflect.MessageDescriptor {
	t.Helper()
	set := &descriptorpb.FileDescriptorSet{File: []*descriptorpb.FileDescriptorProto{{
		Name: strptr("test.proto"), Package: strptr("test"), Syntax: strptr("proto3"),
		MessageType: []*descriptorpb.DescriptorProto{
			{Name: strptr("Envelope"), Field: []*descriptorpb.FieldDescriptorProto{{Name: strptr("role"), JsonName: strptr("role"), Number: int32ptr(1), Label: labelptr(descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL), Type: typeptr(descriptorpb.FieldDescriptorProto_TYPE_STRING)}}},
			{Name: strptr("Request"), Field: []*descriptorpb.FieldDescriptorProto{{Name: strptr("request"), JsonName: strptr("request"), Number: int32ptr(1), Label: labelptr(descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL), Type: typeptr(descriptorpb.FieldDescriptorProto_TYPE_MESSAGE), TypeName: strptr(".test.Envelope")}, {Name: strptr("query"), JsonName: strptr("query"), Number: int32ptr(2), Label: labelptr(descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL), Type: typeptr(descriptorpb.FieldDescriptorProto_TYPE_STRING)}}},
		},
	}}}
	files, err := protodesc.NewFiles(set)
	if err != nil {
		t.Fatal(err)
	}
	desc, err := files.FindDescriptorByName("test.Request")
	if err != nil {
		t.Fatal(err)
	}
	return desc.(protoreflect.MessageDescriptor)
}

func strptr(s string) *string { return &s }
func int32ptr(n int32) *int32 { return &n }
func labelptr(v descriptorpb.FieldDescriptorProto_Label) *descriptorpb.FieldDescriptorProto_Label {
	return &v
}

func typeptr(v descriptorpb.FieldDescriptorProto_Type) *descriptorpb.FieldDescriptorProto_Type {
	return &v
}

func TestResolveArgFieldLadder(t *testing.T) {
	md := testMessageDescriptor(t)
	cases := []struct {
		name   string
		arg    string
		schema ArgSchema
		want   string
		kind   string
	}{
		{name: "query", want: "query", kind: "raw_string"},
		{name: "role", want: "request.role", kind: "raw_string"},
		{name: "payload", schema: ArgSchema{Flags: []Flag{{Name: "payload", Bind: FlagBind{Field: "query", Kind: "json_inline"}}}}, want: "query", kind: "json_inline"},
		{name: "positional-bind", arg: "payload", schema: ArgSchema{Positionals: []Positional{{Name: "payload", Bind: FlagBind{Field: "query", Kind: "json_inline"}}}}, want: "query", kind: "json_inline"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			arg := tc.arg
			if arg == "" {
				arg = tc.name
			}
			resolved, err := ResolveArgField(md, arg, tc.schema)
			if err != nil {
				t.Fatal(err)
			}
			if got := strings.Join(fieldNames(resolved.Path), "."); got != tc.want {
				t.Fatalf("path=%q want %q", got, tc.want)
			}
			if resolved.Kind != tc.kind {
				t.Fatalf("kind=%q want %q", resolved.Kind, tc.kind)
			}
		})
	}
}

func TestProtoBindingOptionsExposeRendererOverride(t *testing.T) {
	options := ProtoBindingOptions{Render: map[string]Renderer{}}
	options.Render["NotesService.ListNotes"] = func(RunContext, proto.Message) error { return nil }
	if options.Render["NotesService.ListNotes"] == nil {
		t.Fatal("renderer override was not retained")
	}
}

func TestResolveArgFieldAgainstRealAIGatewayDescriptor(t *testing.T) {
	source, err := descriptorimage.New(descriptorimage.Config{DescriptorPath: filepath.Join("..", "..", "proto", "gen", "descriptor", "image.binpb")})
	if err != nil {
		t.Fatal(err)
	}
	snapshot, err := source.Snapshot()
	if err != nil {
		t.Fatal(err)
	}
	desc, err := snapshot.Files.FindDescriptorByName("vrooli.ai_gateway.v1.routing.PreviewRouteRequest")
	if err != nil {
		t.Fatal(err)
	}
	resolved, err := ResolveArgField(desc.(protoreflect.MessageDescriptor), "role", ArgSchema{})
	if err != nil {
		t.Fatal(err)
	}
	if got := strings.Join(fieldNames(resolved.Path), "."); got != "request.role" {
		t.Fatalf("path=%q want request.role", got)
	}
}

func fieldNames(path []protoreflect.FieldDescriptor) []string {
	out := make([]string, len(path))
	for i, field := range path {
		out[i] = string(field.Name())
	}
	return out
}

func TestResolveArgFieldTopLevelWins(t *testing.T) {
	md := testMessageDescriptor(t)
	resolved, err := ResolveArgField(md, "query", ArgSchema{})
	if err != nil {
		t.Fatal(err)
	}
	if got := strings.Join(fieldNames(resolved.Path), "."); got != "query" {
		t.Fatalf("path=%q want query", got)
	}
}

func TestResolveArgFieldUnmappedNamesArgument(t *testing.T) {
	_, err := ResolveArgField(testMessageDescriptor(t), "missing-value", ArgSchema{})
	var resolutionErr *ArgFieldResolutionError
	if !errors.As(err, &resolutionErr) {
		t.Fatalf("error=%T %v is not an ArgFieldResolutionError", err, err)
	}
	if resolutionErr.ArgumentName != "missing-value" || resolutionErr.RequestType != "test.Request" {
		t.Fatalf("resolution metadata = %+v", resolutionErr)
	}
	if !containsString(resolutionErr.Candidates, "query") || !containsString(resolutionErr.Candidates, "request.role") {
		t.Fatalf("candidate fields = %v", resolutionErr.Candidates)
	}
	if got := strings.Count(err.Error(), `argument "missing-value"`); got != 1 {
		t.Fatalf("argument appears %d times in %q", got, err)
	}
	if !strings.Contains(err.Error(), "candidate fields: request, request.role, query") {
		t.Fatalf("error=%v does not render candidates", err)
	}
}

func containsString(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}

// scalarDecodableFixture carries one field of every shape the predicate must
// classify, including the repeated-scalar case that must stay legal because a
// repeatable flag supplies it element by element.
func scalarDecodableFixture(t *testing.T) protoreflect.MessageDescriptor {
	t.Helper()
	set := &descriptorpb.FileDescriptorSet{File: []*descriptorpb.FileDescriptorProto{{
		Name: strptr("shapes.proto"), Package: strptr("shapes"), Syntax: strptr("proto3"),
		Dependency: []string{"google/protobuf/timestamp.proto", "google/protobuf/struct.proto"},
		EnumType: []*descriptorpb.EnumDescriptorProto{{
			Name:  strptr("Kind"),
			Value: []*descriptorpb.EnumValueDescriptorProto{{Name: strptr("KIND_UNSPECIFIED"), Number: int32ptr(0)}},
		}},
		MessageType: []*descriptorpb.DescriptorProto{
			{Name: strptr("Profile"), Field: []*descriptorpb.FieldDescriptorProto{
				{Name: strptr("device_name"), JsonName: strptr("deviceName"), Number: int32ptr(1), Label: labelptr(descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL), Type: typeptr(descriptorpb.FieldDescriptorProto_TYPE_STRING)},
			}},
			{Name: strptr("Shapes"), Field: []*descriptorpb.FieldDescriptorProto{
				{Name: strptr("text"), JsonName: strptr("text"), Number: int32ptr(1), Label: labelptr(descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL), Type: typeptr(descriptorpb.FieldDescriptorProto_TYPE_STRING)},
				{Name: strptr("kind"), JsonName: strptr("kind"), Number: int32ptr(2), Label: labelptr(descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL), Type: typeptr(descriptorpb.FieldDescriptorProto_TYPE_ENUM), TypeName: strptr(".shapes.Kind")},
				{Name: strptr("tags"), JsonName: strptr("tags"), Number: int32ptr(3), Label: labelptr(descriptorpb.FieldDescriptorProto_LABEL_REPEATED), Type: typeptr(descriptorpb.FieldDescriptorProto_TYPE_STRING)},
				{Name: strptr("kinds"), JsonName: strptr("kinds"), Number: int32ptr(4), Label: labelptr(descriptorpb.FieldDescriptorProto_LABEL_REPEATED), Type: typeptr(descriptorpb.FieldDescriptorProto_TYPE_ENUM), TypeName: strptr(".shapes.Kind")},
				{Name: strptr("profile"), JsonName: strptr("profile"), Number: int32ptr(5), Label: labelptr(descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL), Type: typeptr(descriptorpb.FieldDescriptorProto_TYPE_MESSAGE), TypeName: strptr(".shapes.Profile")},
				{Name: strptr("profiles"), JsonName: strptr("profiles"), Number: int32ptr(6), Label: labelptr(descriptorpb.FieldDescriptorProto_LABEL_REPEATED), Type: typeptr(descriptorpb.FieldDescriptorProto_TYPE_MESSAGE), TypeName: strptr(".shapes.Profile")},
				{Name: strptr("created_at"), JsonName: strptr("createdAt"), Number: int32ptr(7), Label: labelptr(descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL), Type: typeptr(descriptorpb.FieldDescriptorProto_TYPE_MESSAGE), TypeName: strptr(".google.protobuf.Timestamp")},
				{Name: strptr("payload"), JsonName: strptr("payload"), Number: int32ptr(8), Label: labelptr(descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL), Type: typeptr(descriptorpb.FieldDescriptorProto_TYPE_MESSAGE), TypeName: strptr(".google.protobuf.Struct")},
			}},
		},
	}}}
	files, err := protodesc.NewFiles(mergeWellKnown(t, set))
	if err != nil {
		t.Fatal(err)
	}
	desc, err := files.FindDescriptorByName("shapes.Shapes")
	if err != nil {
		t.Fatal(err)
	}
	return desc.(protoreflect.MessageDescriptor)
}

func mergeWellKnown(t *testing.T, set *descriptorpb.FileDescriptorSet) *descriptorpb.FileDescriptorSet {
	t.Helper()
	for _, name := range []string{"google/protobuf/timestamp.proto", "google/protobuf/struct.proto"} {
		file, err := protoregistry.GlobalFiles.FindFileByPath(name)
		if err != nil {
			t.Fatal(err)
		}
		set.File = append([]*descriptorpb.FileDescriptorProto{protodesc.ToFileDescriptorProto(file)}, set.File...)
	}
	return set
}

func TestScalarDecodableFieldClassifiesEveryShape(t *testing.T) {
	desc := scalarDecodableFixture(t)
	tests := []struct {
		field string
		want  bool
		why   string
	}{
		{"text", true, "plain scalar"},
		{"kind", true, "enum decodes from its name"},
		{"tags", true, "repeated scalar is supplied by a repeatable flag"},
		{"kinds", true, "repeated enum is supplied by a repeatable flag"},
		{"profile", false, "a bare string cannot carry message structure"},
		{"profiles", false, "no sequence of bare strings builds a message list"},
		{"created_at", true, "Timestamp is protojson-encoded as a string"},
		{"payload", false, "Struct is a JSON object and needs a structured decoder"},
	}
	for _, tt := range tests {
		t.Run(tt.field, func(t *testing.T) {
			field := desc.Fields().ByName(protoreflect.Name(tt.field))
			if field == nil {
				t.Fatalf("fixture has no field %q", tt.field)
			}
			if got := ScalarDecodableField(field); got != tt.want {
				t.Fatalf("ScalarDecodableField(%s) = %t, want %t (%s)", tt.field, got, tt.want, tt.why)
			}
		})
	}
}

func TestStructuredDecodeKindExemptsOnlyStructuredDecoders(t *testing.T) {
	for kind, want := range map[string]bool{
		"json_file": true, "json_inline": true, "raw_string": false, "": false,
	} {
		if got := StructuredDecodeKind(kind); got != want {
			t.Fatalf("StructuredDecodeKind(%q) = %t, want %t", kind, got, want)
		}
	}
}
