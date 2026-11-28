package system

import (
	"log/slog"
	"testing"
)

// mockBuildStore implements BuildStore for system service tests.
type mockBuildStore struct {
	statuses map[string]*BuildStatus
}

func (m *mockBuildStore) Snapshot() map[string]*BuildStatus {
	if m.statuses == nil {
		return map[string]*BuildStatus{}
	}
	return m.statuses
}

func TestNewHandler(t *testing.T) {
	wineService := NewWineService(slog.Default())
	buildStore := &mockBuildStore{}
	h := NewHandler(wineService, buildStore, "/tmp/templates")

	if h == nil || h.wineService != wineService || h.builds != buildStore || h.templateDir != "/tmp/templates" {
		t.Fatalf("NewHandler() did not retain its domain dependencies: %#v", h)
	}
}

func TestTemplateInfosDefinesSupportedTemplates(t *testing.T) {
	templates := templateInfos()
	if len(templates) != 4 {
		t.Fatalf("templateInfos() returned %d templates, want 4", len(templates))
	}
	want := map[string]bool{"universal": true, "advanced": true, "multi_window": true, "kiosk": true}
	for _, template := range templates {
		if !want[template.Type] {
			t.Errorf("unexpected template type %q", template.Type)
		}
		delete(want, template.Type)
	}
	if len(want) != 0 {
		t.Errorf("missing template types: %v", want)
	}
}
