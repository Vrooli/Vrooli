package facts

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseProtoServiceMethodsSkipsCommentsAndNestedOptions(t *testing.T) {
	data := []byte(`syntax = "proto3";
// service SearchService { rpc Fake(Request) returns (Response); }
service SearchService {
  option (example.config) = { rpc Fake {} };
  rpc Search(Request) returns (Response);
  rpc Stream(Request) returns (stream Response);
}
`)
	got, complete := parseProtoServiceMethods(data, "SearchService")
	if !complete || got == nil || len(got.methods) != 2 {
		t.Fatalf("expected two declared RPC methods, got %#v complete=%v", got, complete)
	}
	if got.methods[0].name != "Search" || got.methods[1].name != "Stream" {
		t.Fatalf("unexpected method set: %#v", got.methods)
	}
}

func TestParseProtoServiceMethodsMarksDuplicateServiceAmbiguous(t *testing.T) {
	data := []byte(`service SearchService { rpc Search(Request) returns (Response); }
service SearchService { rpc Other(Request) returns (Response); }`)
	got, complete := parseProtoServiceMethods(data, "SearchService")
	if !complete || got == nil || !got.ambiguous {
		t.Fatalf("duplicate service declaration must be ambiguous: %#v complete=%v", got, complete)
	}
}

func TestParseProtoServiceMethodsFailsClosedOnRpcLookalikeOption(t *testing.T) {
	data := []byte(`service SearchService { option (example.marker) = rpc; }`)
	if got, complete := parseProtoServiceMethods(data, "SearchService"); complete || got != nil {
		t.Fatalf("non-declaration rpc token should make lexical parse incomplete: %#v complete=%v", got, complete)
	}
}

func TestProtoSchemaDirRequiresGeneratedProtoTreePath(t *testing.T) {
	if _, ok := protoSchemaDir("/repo", "packages/proto/gen/go/../../other"); ok {
		t.Fatal("path traversal must not resolve to schema source")
	}
	if _, ok := protoSchemaDir("/repo", "elsewhere/generated"); ok {
		t.Fatal("non-generated path must not resolve to schema source")
	}
}

func TestProtoSchemaDirMapsConnectPackageToSchemaPackage(t *testing.T) {
	repo := t.TempDir()
	want := filepath.Join(repo, "packages", "proto", "schemas", "code-facts", "v1", "facts")
	if err := os.MkdirAll(want, 0o700); err != nil {
		t.Fatal(err)
	}
	got, ok := protoSchemaDir(repo, "packages/proto/gen/go/code-facts/v1/facts/facts_v1connect")
	if !ok || got != want {
		t.Fatalf("schema directory = %q, %v; want %q", got, ok, want)
	}
}

func TestProtoSchemaDirRejectsSymlinkedSchemaComponents(t *testing.T) {
	repo := t.TempDir()
	schemas := filepath.Join(repo, "packages", "proto", "schemas")
	if err := os.MkdirAll(filepath.Join(repo, "packages", "proto"), 0o700); err != nil {
		t.Fatal(err)
	}
	external := t.TempDir()
	if err := os.Symlink(external, schemas); err != nil {
		t.Skipf("symlink unavailable: %v", err)
	}
	if _, ok := protoSchemaDir(repo, "packages/proto/gen/go/demo/v1/api/api_v1connect"); ok {
		t.Fatal("symlinked schema ancestry must not be followed")
	}
}

func TestLoadProtoSchemaServicesReturnsExactSourceMethodProvenance(t *testing.T) {
	repo := t.TempDir()
	dir := filepath.Join(repo, "packages", "proto", "schemas", "sample", "v1", "api")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, "api.proto")
	data := []byte("syntax = \"proto3\";\nservice SampleService {\n  rpc Read(Request) returns (Response);\n}\n")
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatal(err)
	}
	got, incomplete := loadProtoSchemaServices(repo, dir, "SampleService")
	if incomplete || len(got) != 1 || len(got[0].methods) != 1 {
		t.Fatalf("schema source did not resolve: %#v incomplete=%v", got, incomplete)
	}
	if got[0].source.Path != "packages/proto/schemas/sample/v1/api/api.proto" || got[0].source.Line != 2 || got[0].source.SHA256 == "" || got[0].methods[0].line != 3 {
		t.Fatalf("unexpected proto source provenance: %#v", got[0])
	}
}

func TestApplyProtoSchemaParityReportsMissingAndExtraGeneratedMethods(t *testing.T) {
	repo := t.TempDir()
	dir := filepath.Join(repo, "packages", "proto", "schemas", "demo", "v1", "api")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	data := []byte("service ParityService { rpc Existing(Request) returns (Response); rpc ProtoOnly(Request) returns (Response); }")
	if err := os.WriteFile(filepath.Join(dir, "api.proto"), data, 0o600); err != nil {
		t.Fatal(err)
	}
	contract := protoHandlerContract{
		service:       "ParityService",
		generatedPath: "packages/proto/gen/go/demo/v1/api/api_v1connect",
		methods: []protoHandlerMethodContract{
			{name: "Existing"},
			{name: "GeneratedOnly"},
		},
	}
	applyProtoSchemaParity(repo, &contract)
	if contract.protoStatus != "MISSING_GENERATED_METHOD" {
		t.Fatalf("proto mismatch status = %q", contract.protoStatus)
	}
	service := compareProtoHandlerContract(contract, nil, true)
	statuses := map[string]string{}
	for _, method := range service.ProtoMethods {
		statuses[method.Name] = method.Status
	}
	if statuses["Existing"] != "MATCH" || statuses["ProtoOnly"] != "MISSING_GENERATED_METHOD" || statuses["GeneratedOnly"] != "GENERATED_METHOD_UNDECLARED" {
		t.Fatalf("method parity did not expose both directions: %#v", service.ProtoMethods)
	}
}
