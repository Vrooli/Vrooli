package interfacegraph

import "testing"

func TestAttributorAttributeFleetConventions(t *testing.T) {
	attr := NewAttributor([]string{"code-facts", "proto-health", "scenario-dependency-analyzer"})

	tests := []struct {
		name       string
		importPath string
		want       string
		wantOK     bool
	}{
		{
			name:       "generated go proto package",
			importPath: "github.com/vrooli/vrooli/packages/proto/gen/go/code-facts/v1/facts",
			want:       "code-facts",
			wantOK:     true,
		},
		{
			name:       "schema path",
			importPath: "packages/proto/schemas/proto-health/v1/validation/validation.proto",
			want:       "proto-health",
			wantOK:     true,
		},
		{
			name:       "scenario source path",
			importPath: "scenarios/scenario-dependency-analyzer/api/internal/interfacegraph",
			want:       "scenario-dependency-analyzer",
			wantOK:     true,
		},
		{
			name:       "external import",
			importPath: "connectrpc.com/connect",
			wantOK:     false,
		},
		{
			name:       "quoted import",
			importPath: `"github.com/vrooli/vrooli/packages/proto/gen/go/code-facts/v1/facts"`,
			want:       "code-facts",
			wantOK:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := attr.Attribute(tt.importPath)
			if ok != tt.wantOK {
				t.Fatalf("ok = %v, want %v", ok, tt.wantOK)
			}
			if got != tt.want {
				t.Fatalf("scenario = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestAttributorIgnoresSharedProtoPackages(t *testing.T) {
	attr := NewAttributor([]string{"scenario-dependency-analyzer", "common"})
	attr.AddScenario("common")

	if _, ok := attr.Attribute("github.com/vrooli/vrooli/packages/proto/gen/go/common/v1"); ok {
		t.Fatal("shared common proto package must not be attributed as a scenario dependency")
	}
}
