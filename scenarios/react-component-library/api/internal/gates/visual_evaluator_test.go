package gates

import (
	"bytes"
	"image"
	"image/png"
	"os"
	"path/filepath"
	"testing"
)

func TestValidateVisualChecksOnlyMaterializedRenderablePreviews(t *testing.T) {
	validPNG := func() []byte {
		var buffer bytes.Buffer
		if err := png.Encode(&buffer, image.NewRGBA(image.Rect(0, 0, 1, 1))); err != nil {
			t.Fatalf("png.Encode() error = %v", err)
		}
		return buffer.Bytes()
	}
	for _, test := range []struct {
		name                 string
		kind                 string
		latest               string
		preview              []byte
		wantCode             string
		wantInspec           int
		addRenderableControl bool
	}{
		{name: "valid rendered preview", kind: "component", latest: "1.0.0", preview: validPNG(), wantInspec: 1},
		{name: "missing latest release", kind: "component", wantCode: "catalog.visual_preview_missing", wantInspec: 1},
		{name: "missing preview capture", kind: "component", latest: "1.0.0", wantCode: "catalog.visual_preview_missing", wantInspec: 1},
		{name: "invalid preview bytes", kind: "component", latest: "1.0.0", preview: []byte("not a png"), wantCode: "catalog.visual_preview_invalid", wantInspec: 1},
		{name: "non-renderable asset", kind: "hook", latest: "1.0.0", wantInspec: 1, addRenderableControl: true},
	} {
		t.Run(test.name, func(t *testing.T) {
			root := t.TempDir()
			writeVisualFixture(t, root, "scenarios/react-component-library/catalog/assets/components/panel.json", `{"asset":{"id":"controls.panel","kind":"`+test.kind+`","name":"Panel"}}`)
			writeVisualFixture(t, root, "scenarios/react-component-library/library/components/Panel/component.json", `{"catalogId":"controls.panel","latest":"`+test.latest+`"}`)
			if test.addRenderableControl {
				writeVisualFixture(t, root, "scenarios/react-component-library/catalog/assets/components/button.json", `{"asset":{"id":"controls.button","kind":"component","name":"Button"}}`)
				writeVisualFixture(t, root, "scenarios/react-component-library/library/components/Button/component.json", `{"catalogId":"controls.button","latest":"1.0.0"}`)
				writeVisualFixtureBytes(t, root, "scenarios/react-component-library/library/components/Button/versions/1.0.0/preview.png", validPNG())
			}
			if len(test.preview) > 0 {
				writeVisualFixtureBytes(t, root, "scenarios/react-component-library/library/components/Panel/versions/1.0.0/preview.png", test.preview)
			}

			result, err := ValidateVisual(Scope{Root: root})
			if err != nil {
				t.Fatalf("ValidateVisual() error = %v", err)
			}
			if result.Inspected != test.wantInspec {
				t.Fatalf("Inspected = %d, want %d; result: %+v", result.Inspected, test.wantInspec, result)
			}
			if test.wantCode == "" {
				if len(result.Findings) != 0 {
					t.Fatalf("Findings = %+v, want clean result", result.Findings)
				}
				return
			}
			if len(result.Findings) != 1 || result.Findings[0].Code != test.wantCode {
				t.Fatalf("Findings = %+v, want one %s finding", result.Findings, test.wantCode)
			}
		})
	}
}

func writeVisualFixture(t *testing.T, root, name, contents string) {
	t.Helper()
	writeVisualFixtureBytes(t, root, name, []byte(contents))
}

func writeVisualFixtureBytes(t *testing.T, root, name string, contents []byte) {
	t.Helper()
	path := filepath.Join(root, filepath.FromSlash(name))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("MkdirAll(%s): %v", filepath.Dir(path), err)
	}
	if err := os.WriteFile(path, contents, 0o644); err != nil {
		t.Fatalf("WriteFile(%s): %v", path, err)
	}
}
