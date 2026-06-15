package pagediscovery

import (
	"path/filepath"
	"testing"
)

func lighthousePath(dir string) string {
	return filepath.Join(dir, ".vrooli", "lighthouse.json")
}

func TestDiscoverLighthouse(t *testing.T) {
	dir := "/scenarios/web-console"
	cfg := `{"enabled": true, "pages": [
		{"id": "home", "path": "/", "label": "Home"},
		{"path": "/backlog", "label": "Backlog", "waitForSelector": "[data-testid='ready']"}
	]}`
	d := New(FakeFileReader{Files: map[string][]byte{lighthousePath(dir): []byte(cfg)}})

	pages, method := d.Discover(dir, nil)
	if method != MethodLighthouse {
		t.Fatalf("method = %q, want %q", method, MethodLighthouse)
	}
	if len(pages) != 2 {
		t.Fatalf("got %d pages, want 2", len(pages))
	}
	if pages[0].Path != "/" || pages[1].Path != "/backlog" {
		t.Fatalf("unexpected page paths: %+v", pages)
	}
	if pages[1].WaitForSelector != "[data-testid='ready']" {
		t.Fatalf("waitForSelector not carried through: %+v", pages[1])
	}
}

func TestDiscoverFallbackWhenNoConfig(t *testing.T) {
	d := New(FakeFileReader{}) // no files
	pages, method := d.Discover("/scenarios/no-config", nil)
	if method != MethodFallback {
		t.Fatalf("method = %q, want %q", method, MethodFallback)
	}
	if len(pages) != 1 || pages[0].Path != "/" {
		t.Fatalf("fallback should be a single home page, got %+v", pages)
	}
}

func TestDiscoverFallbackWhenDisabledOrEmpty(t *testing.T) {
	dir := "/scenarios/disabled"
	cases := map[string]string{
		"disabled": `{"enabled": false, "pages": [{"path": "/x"}]}`,
		"no-pages": `{"enabled": true, "pages": []}`,
		"bad-json": `{not json`,
	}
	for name, cfg := range cases {
		t.Run(name, func(t *testing.T) {
			d := New(FakeFileReader{Files: map[string][]byte{lighthousePath(dir): []byte(cfg)}})
			pages, method := d.Discover(dir, nil)
			if method != MethodFallback {
				t.Fatalf("method = %q, want %q", method, MethodFallback)
			}
			if len(pages) != 1 || pages[0].Path != "/" {
				t.Fatalf("expected single home fallback, got %+v", pages)
			}
		})
	}
}

func TestDiscoverExplicitWins(t *testing.T) {
	dir := "/scenarios/web-console"
	cfg := `{"enabled": true, "pages": [{"path": "/from-config"}]}`
	d := New(FakeFileReader{Files: map[string][]byte{lighthousePath(dir): []byte(cfg)}})

	pages, method := d.Discover(dir, []string{"/a", "/b"})
	if method != MethodExplicit {
		t.Fatalf("method = %q, want %q", method, MethodExplicit)
	}
	if len(pages) != 2 || pages[0].Path != "/a" || pages[1].Path != "/b" {
		t.Fatalf("explicit pages not honored: %+v", pages)
	}
}

func TestDiscoverNilFSUsesRealFilesystem(t *testing.T) {
	// New(nil) must not panic and must fall back when the path is absent.
	d := New(nil)
	pages, method := d.Discover(t.TempDir(), nil)
	if method != MethodFallback || len(pages) != 1 {
		t.Fatalf("expected fallback with real fs, got %v / %+v", method, pages)
	}
}
