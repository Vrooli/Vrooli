//nolint:goconst // test data deliberately reuses stable template fixtures.
package resources

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestListBlueprintsLoadsSeededRecordsInSortedOrder(t *testing.T) {
	controller := NewController(projectRoot(t), t.TempDir())

	items, err := controller.ListBlueprints()
	if err != nil {
		t.Fatalf("ListBlueprints: %v", err)
	}
	if len(items) < 8 {
		t.Fatalf("len(items) = %d, want at least 8", len(items))
	}
	for i := 1; i < len(items); i++ {
		if items[i-1].Name > items[i].Name {
			t.Fatalf("items not sorted: %q before %q", items[i-1].Name, items[i].Name)
		}
	}
}

func TestBlueprintReturnsKnownRecord(t *testing.T) {
	controller := NewController(projectRoot(t), t.TempDir())

	item, err := controller.Blueprint("terraform")
	if err != nil {
		t.Fatalf("Blueprint(terraform): %v", err)
	}
	if item.SuggestedTemplate != "external-cli" {
		t.Fatalf("SuggestedTemplate = %q, want external-cli", item.SuggestedTemplate)
	}
}

func TestSearchBlueprintsMatchesSummaryAndCategory(t *testing.T) {
	controller := NewController(projectRoot(t), t.TempDir())

	items, err := controller.SearchBlueprints("network")
	if err != nil {
		t.Fatalf("SearchBlueprints: %v", err)
	}
	if len(items) == 0 {
		t.Fatal("expected network search to return at least one blueprint")
	}
}

func TestValidateBlueprintsRejectsFilenameMismatch(t *testing.T) {
	root := t.TempDir()
	home := t.TempDir()
	path := filepath.Join(root, filepath.FromSlash(blueprintDirPath), "wrong-name.json")
	writeBlueprintFixture(t, path, `{
  "name": "right-name",
  "display_name": "Right Name",
  "category": "testing",
  "summary": "Fixture summary",
  "why_it_matters": "Fixture importance",
  "when_to_use": ["Fixture use"],
  "integration_kind": "external-cli",
  "platform_support": {
    "notes": "Fixture notes",
    "linux": "supported",
    "macos": "supported",
    "windows": "supported"
  },
  "suggested_template": "external-cli",
  "implementation_notes": ["Implement me"],
  "operational_notes": ["Operate me"],
  "risks": ["Risk"],
  "status": "candidate",
  "references": [{"kind": "note", "value": "fixture"}],
  "last_reviewed": "2026-04-11"
}`)

	_, err := NewController(root, home).ListBlueprints()
	if err == nil || !strings.Contains(err.Error(), "filename") {
		t.Fatalf("expected filename validation error, got %v", err)
	}
}

func TestValidateBlueprintsRejectsMissingRequiredFields(t *testing.T) {
	root := t.TempDir()
	home := t.TempDir()
	path := filepath.Join(root, filepath.FromSlash(blueprintDirPath), "broken.json")
	writeBlueprintFixture(t, path, `{
  "name": "broken",
  "display_name": "Broken",
  "category": "testing",
  "summary": "Fixture summary",
  "why_it_matters": "Fixture importance",
  "when_to_use": [],
  "integration_kind": "external-cli",
  "platform_support": {
    "notes": "Fixture notes",
    "linux": "supported",
    "macos": "supported",
    "windows": "supported"
  },
  "suggested_template": "external-cli",
  "implementation_notes": ["Implement me"],
  "operational_notes": ["Operate me"],
  "risks": ["Risk"],
  "status": "candidate",
  "references": [{"kind": "note", "value": "fixture"}],
  "last_reviewed": "2026-04-11"
}`)

	_, err := NewController(root, home).ValidateBlueprints()
	if err == nil || !strings.Contains(err.Error(), "when_to_use") {
		t.Fatalf("expected when_to_use validation error, got %v", err)
	}
}

func TestValidateBlueprintsRejectsTemplateRuleMismatch(t *testing.T) {
	root := t.TempDir()
	home := t.TempDir()
	path := filepath.Join(root, filepath.FromSlash(blueprintDirPath), "broken-template-rule.json")
	writeBlueprintFixture(t, path, `{
  "name": "broken-template-rule",
  "display_name": "Broken Template Rule",
  "category": "testing",
  "summary": "Fixture summary",
  "why_it_matters": "Fixture importance",
  "when_to_use": ["Fixture use"],
  "integration_kind": "cloud-api",
  "platform_support": {
    "notes": "Fixture notes",
    "linux": "supported",
    "macos": "supported",
    "windows": "supported"
  },
  "suggested_template": "external-cli",
  "implementation_notes": ["Implement me"],
  "operational_notes": ["Operate me"],
  "risks": ["Risk"],
  "status": "candidate",
  "references": [{"kind": "note", "value": "fixture"}],
  "last_reviewed": "2026-04-11"
}`)

	_, err := NewController(root, home).ValidateBlueprints()
	if err == nil || !strings.Contains(err.Error(), "suggested_template") {
		t.Fatalf("expected suggested_template validation error, got %v", err)
	}
}

func writeBlueprintFixture(t *testing.T, path, contents string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", filepath.Dir(path), err)
	}
	if err := os.WriteFile(path, []byte(contents+"\n"), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

func projectRoot(t *testing.T) string {
	t.Helper()
	root, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatalf("resolve root: %v", err)
	}
	return root
}
