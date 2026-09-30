package validationprovider

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	scenariovalidationv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-validation/v1"
	"google.golang.org/protobuf/types/known/anypb"
	"google.golang.org/protobuf/types/known/structpb"
)

func TestValidateIsolationEvidence(t *testing.T) {
	valid := nativeDetailForTest(t, map[string]any{
		"installed":                            true,
		"lease_id":                             "lease-1",
		"install_error":                        "",
		"heartbeat_error":                      "",
		"clear_error":                          "",
		"test_pool_requests":                   2,
		"primary_during_test_mode_requests":    0,
		"test_root_writes":                     1,
		"primary_root_writes_during_test_mode": 0,
	})
	if err := validateIsolationEvidence(valid); err != nil {
		t.Fatalf("valid isolation evidence rejected: %v", err)
	}

	for _, test := range []struct {
		name   string
		mutate func(map[string]any)
	}{
		{name: "missing lease", mutate: func(fields map[string]any) { fields["lease_id"] = "" }},
		{name: "install failure", mutate: func(fields map[string]any) { fields["install_error"] = "storage unavailable" }},
		{name: "primary request", mutate: func(fields map[string]any) { fields["primary_during_test_mode_requests"] = 1 }},
		{name: "primary write", mutate: func(fields map[string]any) { fields["primary_root_writes_during_test_mode"] = 1 }},
	} {
		t.Run(test.name, func(t *testing.T) {
			fields := map[string]any{
				"installed":                            true,
				"lease_id":                             "lease-1",
				"install_error":                        "",
				"heartbeat_error":                      "",
				"clear_error":                          "",
				"primary_during_test_mode_requests":    0,
				"primary_root_writes_during_test_mode": 0,
			}
			test.mutate(fields)
			if err := validateIsolationEvidence(nativeDetailForTest(t, fields)); err == nil {
				t.Fatal("invalid isolation evidence was accepted")
			}
		})
	}
}

func nativeDetailForTest(t *testing.T, isolation map[string]any) *anypb.Any {
	t.Helper()
	detail, err := structpb.NewStruct(map[string]any{
		"execution": map[string]any{"isolation": isolation},
	})
	if err != nil {
		t.Fatal(err)
	}
	native, err := anypb.New(detail)
	if err != nil {
		t.Fatal(err)
	}
	return native
}

func TestArtifactEvidenceUsesNormalizedWorkflowURI(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "workflow.json")
	if err := os.WriteFile(path, []byte(`{"ok":true}`), 0o600); err != nil {
		t.Fatal(err)
	}
	reference, err := anypb.New(mustStruct(t, map[string]any{"reference": path}))
	if err != nil {
		t.Fatal(err)
	}
	evidence, err := artifactEvidence(&scenariovalidationv1.ValidationRun{RunId: "run-1", ArtifactReferences: []*anypb.Any{reference}}, root, Request{})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(evidence[0].GetUri(), "run-1//") {
		t.Fatalf("workflow URI retained an absolute-path double slash: %q", evidence[0].GetUri())
	}
}

func mustStruct(t *testing.T, fields map[string]any) *structpb.Struct {
	t.Helper()
	value, err := structpb.NewStruct(fields)
	if err != nil {
		t.Fatal(err)
	}
	return value
}
