package planscli

import (
	"strings"
	"testing"
)

func TestParseImportRequestWorkspace(t *testing.T) {
	req, err := ParseImportRequest([]string{"docs/plans/legacy.md", "--workspace", "/workspace"})
	if err != nil {
		t.Fatalf("ParseImportRequest: %v", err)
	}
	if req.Path != "docs/plans/legacy.md" || req.Workspace != "/workspace" {
		t.Fatalf("request = %#v", req)
	}
}

func TestParseRejectsRepoWorkspaceConflict(t *testing.T) {
	_, err := ParseAddRequest([]string{"--stdin", "--repo", "/repo", "--workspace", "/workspace"})
	if err == nil || !strings.Contains(err.Error(), "use --workspace, not both") {
		t.Fatalf("ParseAddRequest err = %v, want repo/workspace conflict", err)
	}
}

func TestParseListRejectsDeprecatedAll(t *testing.T) {
	_, err := ParseListRequest([]string{"--all"})
	if err == nil || !strings.Contains(err.Error(), "--all is deprecated") {
		t.Fatalf("ParseListRequest err = %v, want --all deprecation", err)
	}
}
