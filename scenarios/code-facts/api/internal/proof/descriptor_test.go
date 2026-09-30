package proof

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/vrooli/vrooli/packages/proto/descriptorimage"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/descriptorpb"
)

func TestDescriptorSourceExtractsResolvedContractsWithProtoProvenance(t *testing.T) {
	repoRoot := t.TempDir()
	protoPath := filepath.Join(repoRoot, "packages", "proto", "schemas", "demo", "v1", "demo.proto")
	descriptorPath := filepath.Join(repoRoot, "packages", "proto", "gen", "descriptor", "image.binpb")
	mustWriteFile(t, protoPath, []byte("syntax = \"proto3\";\npackage demo.v1;\n// Greeter docs.\nservice Greeter {\n  rpc SayHello(Request) returns (Response);\n}\nmessage Request { string name = 1; }\nmessage Response { string greeting = 1; }\nenum State { STATE_UNSPECIFIED = 0; }\n"))
	set := descriptorFixture()
	raw, err := proto.Marshal(set)
	if err != nil {
		t.Fatal(err)
	}
	mustWriteFile(t, descriptorPath, raw)
	source, err := descriptorimage.New(descriptorimage.Config{DescriptorPath: descriptorPath})
	if err != nil {
		t.Fatal(err)
	}
	got, err := (DescriptorSource{Source: source, RepoRoot: repoRoot, Reader: OSProtoReader{}}).Snapshot(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if got.Digest == "" || got.DescriptorGeneration != 1 || len(got.ProvenanceFailures) != 0 {
		t.Fatalf("unexpected descriptor snapshot metadata: %+v", got)
	}
	method := contractByID(t, got.Contracts, "contract:method:demo.v1.Greeter.SayHello")
	if method.Path != filepath.ToSlash(protoPath) || method.StartLine != 5 || method.EndLine != 5 || method.SourceHash == "" {
		t.Fatalf("method provenance mismatch: %+v", method)
	}
	if method.Attributes["input_type"] != ".demo.v1.Request" || method.Attributes["output_type"] != ".demo.v1.Response" {
		t.Fatalf("method resolved types missing: %+v", method.Attributes)
	}
	if method.Attributes["deprecated"] != "true" || !strings.Contains(method.Attributes["options"], "deprecated:true") {
		t.Fatalf("method descriptor options missing: %+v", method.Attributes)
	}
	service := contractByID(t, got.Contracts, "contract:service:demo.v1.Greeter")
	if service.Comment != "Greeter docs." {
		t.Fatalf("service comment mismatch: %q", service.Comment)
	}
	alias := contractByID(t, got.Contracts, "contract:generated_alias:go:demo.v1.Greeter.SayHello")
	if alias.ParentID != method.ID || alias.Attributes["authority"] != "resolved_contract" {
		t.Fatalf("compact generated alias does not point to authority: %+v", alias)
	}
	field := contractByID(t, got.Contracts, "contract:field:demo.v1.Request.name")
	if field.Attributes["number"] != "1" || field.Attributes["type"] != "TYPE_STRING" {
		t.Fatalf("field structure missing: %+v", field.Attributes)
	}
}

func TestDescriptorSourceServesLastKnownGoodAndReportsReloadFailure(t *testing.T) {
	repoRoot := t.TempDir()
	protoPath := filepath.Join(repoRoot, "packages", "proto", "schemas", "demo", "v1", "demo.proto")
	descriptorPath := filepath.Join(repoRoot, "packages", "proto", "gen", "descriptor", "image.binpb")
	mustWriteFile(t, protoPath, []byte("syntax = \"proto3\"; package demo.v1; message Request {} message Response {} service Greeter { rpc SayHello(Request) returns (Response); }"))
	raw, err := proto.Marshal(descriptorFixture())
	if err != nil {
		t.Fatal(err)
	}
	mustWriteFile(t, descriptorPath, raw)
	source, err := descriptorimage.New(descriptorimage.Config{DescriptorPath: descriptorPath})
	if err != nil {
		t.Fatal(err)
	}
	adapter := DescriptorSource{Source: source, RepoRoot: repoRoot, Reader: OSProtoReader{}}
	first, err := adapter.Snapshot(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	mustWriteFile(t, descriptorPath, []byte("not a descriptor image"))
	second, err := adapter.Snapshot(context.Background())
	if err != nil {
		t.Fatalf("known-good descriptor should remain serviceable: %v", err)
	}
	if second.Digest != first.Digest || second.LastReloadFailure == "" || second.DescriptorGeneration != first.DescriptorGeneration {
		t.Fatalf("reload failure was not surfaced against known-good snapshot: first=%+v second=%+v", first, second)
	}
}

func TestCommittedDescriptorImageResolvesCodeFactsSearchToAuthoritativeProto(t *testing.T) {
	repoRoot, err := filepath.Abs(filepath.Join("..", "..", "..", "..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	source, err := descriptorimage.New(descriptorimage.Config{
		DescriptorPath: filepath.Join(repoRoot, "packages", "proto", "gen", "descriptor", "image.binpb"),
	})
	if err != nil {
		t.Fatal(err)
	}
	snapshot, err := (DescriptorSource{Source: source, RepoRoot: repoRoot, Reader: OSProtoReader{}}).Snapshot(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	method := contractByID(t, snapshot.Contracts, "contract:method:vrooli.code_facts.v1.facts.CodeFactsService.Search")
	wantSuffix := "packages/proto/schemas/code-facts/v1/facts/facts.proto"
	if !strings.HasSuffix(method.Path, wantSuffix) || method.SourceHash == "" || method.Digest != snapshot.Digest {
		t.Fatalf("committed descriptor provenance mismatch: %+v", method)
	}
	if method.StartLine <= 0 || method.Attributes["provenance"] != "source_info" {
		t.Fatalf("committed descriptor omitted source range: %+v", method)
	}
}

func descriptorFixture() *descriptorpb.FileDescriptorSet {
	return &descriptorpb.FileDescriptorSet{File: []*descriptorpb.FileDescriptorProto{{
		Name:    proto.String("demo/v1/demo.proto"),
		Package: proto.String("demo.v1"),
		Syntax:  proto.String("proto3"),
		MessageType: []*descriptorpb.DescriptorProto{
			{Name: proto.String("Request"), Field: []*descriptorpb.FieldDescriptorProto{{Name: proto.String("name"), Number: proto.Int32(1), Label: descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(), Type: descriptorpb.FieldDescriptorProto_TYPE_STRING.Enum()}}},
			{Name: proto.String("Response"), Field: []*descriptorpb.FieldDescriptorProto{{Name: proto.String("greeting"), Number: proto.Int32(1), Label: descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(), Type: descriptorpb.FieldDescriptorProto_TYPE_STRING.Enum()}}},
		},
		EnumType: []*descriptorpb.EnumDescriptorProto{{Name: proto.String("State"), Value: []*descriptorpb.EnumValueDescriptorProto{{Name: proto.String("STATE_UNSPECIFIED"), Number: proto.Int32(0)}}}},
		Service: []*descriptorpb.ServiceDescriptorProto{{
			Name:   proto.String("Greeter"),
			Method: []*descriptorpb.MethodDescriptorProto{{Name: proto.String("SayHello"), InputType: proto.String(".demo.v1.Request"), OutputType: proto.String(".demo.v1.Response"), Options: &descriptorpb.MethodOptions{Deprecated: proto.Bool(true)}}},
		}},
		SourceCodeInfo: &descriptorpb.SourceCodeInfo{Location: []*descriptorpb.SourceCodeInfo_Location{
			{Path: []int32{6, 0}, Span: []int32{3, 0, 5, 1}, LeadingComments: proto.String(" Greeter docs.\n")},
			{Path: []int32{6, 0, 2, 0}, Span: []int32{4, 2, 4, 50}},
			{Path: []int32{4, 0}, Span: []int32{6, 0, 6, 36}},
			{Path: []int32{4, 0, 2, 0}, Span: []int32{6, 18, 6, 34}},
		}},
	}}}
}

func contractByID(t *testing.T, contracts []Contract, id string) Contract {
	t.Helper()
	for _, contract := range contracts {
		if contract.ID == id {
			return contract
		}
	}
	var ids []string
	for _, contract := range contracts {
		ids = append(ids, contract.ID)
	}
	t.Fatalf("contract %q not found in %s", id, strings.Join(ids, ", "))
	return Contract{}
}

func mustWriteFile(t *testing.T, path string, payload []byte) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, payload, 0o644); err != nil {
		t.Fatal(err)
	}
}
