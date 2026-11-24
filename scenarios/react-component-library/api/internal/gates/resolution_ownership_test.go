package gates

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func putResolutionFixture(t *testing.T, root, name, body string) string {
	t.Helper()
	path := filepath.Join(root, filepath.FromSlash(name))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestValidateDistResolution(t *testing.T) {
	manifest := `{"dependencies":{"react":"^18"},"exports":{".":"./dist/index.js","./button":"./dist/Button.js"}}`
	t.Run("accepts declared externals and existing relative imports", func(t *testing.T) {
		root := t.TempDir()
		putResolutionFixture(t, root, "packages/react-component-library/package.json", manifest)
		putResolutionFixture(t, root, "packages/react-component-library/dist/Button.js", `import React from "react"; import {x} from "./chunk.js"; import "node:fs";`)
		putResolutionFixture(t, root, "packages/react-component-library/dist/chunk.js", `export const x = 1`)
		result, err := ValidateDistResolution(Scope{Root: root})
		if err != nil || result.Inspected != 2 || len(result.Findings) != 0 {
			t.Fatalf("result=%+v err=%v", result, err)
		}
	})
	t.Run("reports undeclared and unexported imports", func(t *testing.T) {
		root := t.TempDir()
		putResolutionFixture(t, root, "packages/react-component-library/package.json", manifest)
		putResolutionFixture(t, root, "packages/react-component-library/dist/Button.js", `import "left-pad"; import "@vrooli/react-component-library/private";`)
		result, err := ValidateDistResolution(Scope{Root: root})
		if err != nil || result.Inspected != 1 || len(result.Findings) != 2 {
			t.Fatalf("result=%+v err=%v", result, err)
		}
	})
	t.Run("asset scope filters inspected bundles", func(t *testing.T) {
		root := t.TempDir()
		putResolutionFixture(t, root, "packages/react-component-library/package.json", manifest)
		putResolutionFixture(t, root, "packages/react-component-library/dist/Button.js", `import "unknown"`)
		putResolutionFixture(t, root, "packages/react-component-library/dist/Card.js", `import "unknown"`)
		result, err := ValidateDistResolution(Scope{Root: root, Assets: []string{"catalog.button"}})
		if err != nil || result.Inspected != 1 || len(result.Findings) != 1 {
			t.Fatalf("result=%+v err=%v", result, err)
		}
	})
	t.Run("errors for missing or malformed manifest", func(t *testing.T) {
		if _, err := ValidateDistResolution(Scope{Root: t.TempDir()}); err == nil {
			t.Fatal("expected missing manifest error")
		}
		root := t.TempDir()
		putResolutionFixture(t, root, "packages/react-component-library/package.json", `{`)
		if _, err := ValidateDistResolution(Scope{Root: root}); err == nil {
			t.Fatal("expected malformed manifest error")
		}
	})
}

func TestDistResolutionHelpers(t *testing.T) {
	if !distPathMatchesAsset("/x/dist/ControlButton.js", []string{"catalog.button"}) {
		t.Fatal("expected suffix-based asset match")
	}
	if distPathMatchesAsset("/x/dist/Card.js", []string{"catalog.button"}) {
		t.Fatal("unexpected asset match")
	}
	root := t.TempDir()
	importer := putResolutionFixture(t, root, "packages/react-component-library/dist/a.js", "")
	putResolutionFixture(t, root, "packages/react-component-library/dist/chunk.js", "")
	putResolutionFixture(t, root, "packages/react-component-library/package.json", `{}`)
	putResolutionFixture(t, root, "packages/design-tokens/package.json", `{}`)
	packageRoot := filepath.Join(root, "packages/react-component-library")
	declared := map[string]bool{"react": true}
	exports := map[string]json.RawMessage{".": json.RawMessage(`"./dist/index.js"`)}
	cases := []struct {
		name, spec string
		allowed    bool
	}{
		{"existing relative", "./chunk.js", true},
		{"missing relative", "./missing.js", false},
		{"relative path escaping dist", "../package.json", false},
		{"builtin", "node:fs", true},
		{"package subpath", "react/jsx-runtime", true},
		{"workspace package", "@vrooli/design-tokens/colors", true},
		{"missing workspace package", "@vrooli/unknown-package/colors", false},
		{"undeclared package", "left-pad", false},
		{"root export", "@vrooli/react-component-library", true},
		{"unknown export", "@vrooli/react-component-library/private", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := distImportAllowed(packageRoot, importer, tc.spec, declared, exports); got != tc.allowed {
				t.Fatalf("allowed=%v want %v", got, tc.allowed)
			}
		})
	}
}

func TestValidateFieldOwnership(t *testing.T) {
	root := t.TempDir()
	manifest := func(body string) {
		putResolutionFixture(t, root, "scenarios/react-component-library/library/components/button/component.json", body)
	}
	t.Run("accepts user-owned fields and source headers", func(t *testing.T) {
		manifest(`{"catalogId":"controls.button","title":"Button"}`)
		putResolutionFixture(t, root, "scenarios/react-component-library/library/components/button/versions/v1/Button.tsx", "/** @param label button label */\nexport function Button() {}")
		result, err := ValidateFieldOwnership(Scope{Root: root})
		if err != nil || result.Inspected != 1 || len(result.Findings) != 0 {
			t.Fatalf("result=%+v err=%v", result, err)
		}
	})
	t.Run("reports owned manifest fields and description headers", func(t *testing.T) {
		manifest(`{"catalogId":"controls.button","description":"x","displayName":"Button","tags":[]}`)
		putResolutionFixture(t, root, "scenarios/react-component-library/library/components/button/versions/v1/Button.tsx", "/**\n * @description Button\n */")
		result, err := ValidateFieldOwnership(Scope{Root: root})
		if err != nil || result.Inspected != 1 || len(result.Findings) != 4 {
			t.Fatalf("result=%+v err=%v", result, err)
		}
	})
	t.Run("skips supplemental and unselected assets", func(t *testing.T) {
		manifest(`{"catalogId":"controls.button","supplemental":true,"description":"ignored"}`)
		result, err := ValidateFieldOwnership(Scope{Root: root})
		if err != nil || result.Inspected != 0 || len(result.Findings) != 1 || result.Findings[0].Code != "catalog.gate_inspected_nothing" {
			t.Fatalf("result=%+v err=%v", result, err)
		}
	})
	t.Run("returns malformed manifest error", func(t *testing.T) {
		manifest(`{`)
		if _, err := ValidateFieldOwnership(Scope{Root: root}); err == nil {
			t.Fatal("expected malformed manifest error")
		}
	})
}
