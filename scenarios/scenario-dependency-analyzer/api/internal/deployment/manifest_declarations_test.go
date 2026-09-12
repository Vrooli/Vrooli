package deployment

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	types "github.com/vrooli/vrooli/scenarios/scenario-dependency-analyzer/api/internal/types"
)

func TestResolveResourceTierSupportRequiresDeclaredRequirements(t *testing.T) {
	repoRoot := t.TempDir()
	resourceDir := filepath.Join(repoRoot, "resources", "temporary-undeclared")
	if err := os.MkdirAll(resourceDir, 0o755); err != nil {
		t.Fatalf("mkdir resource: %v", err)
	}
	manifest := map[string]any{
		"name":     "temporary-undeclared",
		"bundling": "vendorable",
		"platforms": map[string]string{
			"linux":   "supported",
			"macos":   "supported",
			"windows": "supported",
		},
	}
	raw, err := json.Marshal(manifest)
	if err != nil {
		t.Fatalf("marshal manifest: %v", err)
	}
	if err := os.WriteFile(filepath.Join(resourceDir, "resource.json"), raw, 0o644); err != nil {
		t.Fatalf("write manifest: %v", err)
	}

	result := resolveResourceTierSupport(repoRoot, "temporary-undeclared", true)
	for tier, summary := range result {
		if summary.Supported != nil {
			t.Fatalf("tier %s unexpectedly received a support verdict: %+v", tier, summary)
		}
		if summary.Reason == "" {
			t.Fatalf("tier %s did not preserve the unknown reason", tier)
		}
	}
}

func TestBuildResourceDependencyNodeUsesLiveManifestOverAuthoredMetadata(t *testing.T) {
	repoRoot := t.TempDir()
	resourceDir := filepath.Join(repoRoot, "resources", "live-resource")
	if err := os.MkdirAll(resourceDir, 0o755); err != nil {
		t.Fatalf("mkdir resource: %v", err)
	}
	manifest := map[string]any{
		"name":     "live-resource",
		"bundling": "host-required",
		"platforms": map[string]string{
			"linux":   "supported",
			"macos":   "unsupported",
			"windows": "supported",
		},
		"requirements": map[string]any{
			"class": "live-class", "weight": 1, "ram_mb": 256,
			"disk_mb": 512, "cpu_cores": 1, "source": "measured", "confidence": "high",
		},
	}
	raw, err := json.Marshal(manifest)
	if err != nil {
		t.Fatalf("marshal manifest: %v", err)
	}
	if err := os.WriteFile(filepath.Join(resourceDir, "resource.json"), raw, 0o644); err != nil {
		t.Fatalf("write manifest: %v", err)
	}

	supported := true
	node := buildResourceDependencyNode(repoRoot, "live-resource", &types.DeploymentDependency{
		ResourceType: "stale-authored-type",
		Footprint:    &types.DeploymentRequirements{RAMMB: ptr(9999.0)},
		PlatformSupport: map[string]types.DependencyTierSupport{
			"tier-1-local": {Supported: &supported},
		},
	}, true)

	if node.ResourceType != "live-class" {
		t.Fatalf("resource type = %q, want live manifest class", node.ResourceType)
	}
	if node.Requirements == nil || node.Requirements.RAMMB == nil || *node.Requirements.RAMMB != 256 {
		t.Fatalf("requirements = %+v, want live manifest requirements", node.Requirements)
	}
	local := node.TierSupport["tier-1-local"]
	if local.Supported == nil || *local.Supported {
		t.Fatalf("local support = %+v, want unsupported aggregate from macOS declaration", local)
	}
}
