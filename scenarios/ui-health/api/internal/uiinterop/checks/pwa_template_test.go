package checks

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"ui-health/internal/uiinterop"
)

func TestReactViteTemplatePassesPWANativeReadinessRules(t *testing.T) {
	root := filepath.Clean(filepath.Join("..", "..", "..", "..", "..", "..", "templates", "scenarios", "react-vite"))
	ctx := uiinterop.CheckContext{ScenarioRoot: root, TechStack: []string{"React"}, ScenarioName: "react-vite-template"}
	for _, tc := range []struct {
		name  string
		check uiinterop.CheckFunc
	}{
		{"manifest install fields", checkPWAManifestInstallFields},
		{"launch scope", checkPWALaunchScope},
		{"service worker offline shell", checkPWAServiceWorkerOffline},
		{"optional platform fields", checkPWAOptionalPlatformFields},
	} {
		t.Run(tc.name, func(t *testing.T) {
			result := tc.check(ctx)
			if result.Skipped {
				t.Fatalf("rule skipped: %s", result.SkipReason)
			}
			if !result.Passed {
				t.Fatalf("rule failed: %s (%v)", result.Message, result.Violations)
			}
		})
	}
}

func TestPWAOptionalPlatformFieldsValidateDeclaredCapabilities(t *testing.T) {
	root := t.TempDir()
	writeTestFile(t, root, "ui/public/shortcut.png", "png")
	writeTestFile(t, root, "ui/public/site.webmanifest", `{
		"scope": ".",
		"shortcuts": [{
			"name": "Open",
			"url": ".",
			"icons": [{ "src": "shortcut.png", "sizes": "96x96", "type": "image/png" }]
		}],
		"share_target": {
			"action": ".",
			"method": "POST",
			"enctype": "multipart/form-data",
			"params": { "title": "title", "text": "text" }
		},
		"protocol_handlers": [{ "protocol": "web+demo", "url": "./open?value=%s" }],
		"file_handlers": [{ "action": ".", "accept": { "text/plain": [".txt"] } }],
		"related_applications": [{ "platform": "webapp", "url": "." }],
		"display_override": ["standalone"],
		"launch_handler": { "client_mode": "navigate-existing" }
	}`)

	result := checkPWAOptionalPlatformFields(uiinterop.CheckContext{ScenarioRoot: root, TechStack: []string{"React"}})
	if result.Skipped {
		t.Fatalf("rule skipped: %s", result.SkipReason)
	}
	if !result.Passed {
		t.Fatalf("rule failed: %s (%v)", result.Message, result.Violations)
	}
}

func TestPWAOptionalPlatformFieldsRejectMalformedDeclaredCapabilities(t *testing.T) {
	root := t.TempDir()
	writeTestFile(t, root, "ui/public/site.webmanifest", `{
		"scope": "/app/",
		"shortcuts": [{ "name": "", "url": "http://localhost:5173/open" }],
		"share_target": { "action": "http://localhost:5173/share", "method": "PATCH", "enctype": "application/json", "params": {} },
		"protocol_handlers": [{ "protocol": "mailto", "url": "./open" }],
		"file_handlers": [{ "action": "http://localhost:5173/file", "accept": {} }],
		"related_applications": [{ "url": "http://localhost:5173/app" }],
		"display_override": ["teleport"],
		"launch_handler": { "client_mode": "clone-existing" }
	}`)

	result := checkPWAOptionalPlatformFields(uiinterop.CheckContext{ScenarioRoot: root, TechStack: []string{"React"}})
	if result.Skipped {
		t.Fatalf("rule skipped: %s", result.SkipReason)
	}
	if result.Passed {
		t.Fatalf("expected malformed optional platform fields to fail")
	}
	for _, want := range []string{"shortcut", "share_target", "protocol handler", "file handler", "related application", "display_override", "launch_handler"} {
		if !pwaViolationsContain(result.Violations, want) {
			t.Fatalf("expected violation containing %q, got %#v", want, result.Violations)
		}
	}
}

func writeTestFile(t *testing.T, root, rel, content string) {
	t.Helper()
	abs := filepath.Join(root, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(abs), 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", filepath.Dir(abs), err)
	}
	if err := os.WriteFile(abs, []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", rel, err)
	}
}

func pwaViolationsContain(violations []uiinterop.Violation, needle string) bool {
	for _, violation := range violations {
		haystack := strings.ToLower(violation.Title + " " + violation.Description + " " + violation.Recommendation)
		if strings.Contains(haystack, strings.ToLower(needle)) {
			return true
		}
	}
	return false
}

func TestReactViteTemplateUsesProxySafePWAURLs(t *testing.T) {
	root := filepath.Clean(filepath.Join("..", "..", "..", "..", "..", "..", "templates", "scenarios", "react-vite"))
	for _, tc := range []struct {
		rel       string
		forbidden []string
		required  []string
	}{
		{
			rel:       "ui/index.html",
			forbidden: []string{`href="/site.webmanifest"`, `href="/favicon-196.png"`, `href="/apple-icon-180.png"`},
			required:  []string{`href="site.webmanifest"`, `href="favicon-196.png"`, `href="apple-icon-180.png"`, "viewport-fit=cover"},
		},
		{
			rel:       "ui/src/main.tsx",
			forbidden: []string{`register("/sw.js")`},
			required:  []string{`register("sw.js")`},
		},
		{
			rel:       "ui/public/sw.js",
			forbidden: []string{`"/"`, `"/site.webmanifest"`},
			required:  []string{`"./"`, `"./site.webmanifest"`},
		},
	} {
		t.Run(tc.rel, func(t *testing.T) {
			data, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(tc.rel)))
			if err != nil {
				t.Fatalf("read %s: %v", tc.rel, err)
			}
			content := string(data)
			for _, forbidden := range tc.forbidden {
				if strings.Contains(content, forbidden) {
					t.Fatalf("%s contains proxy-unsafe %q", tc.rel, forbidden)
				}
			}
			for _, required := range tc.required {
				if !strings.Contains(content, required) {
					t.Fatalf("%s missing %q", tc.rel, required)
				}
			}
		})
	}
}
