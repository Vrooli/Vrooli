package generation

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestWailsGeneratorCreatesInProcessProjectAndCopiesAssets(t *testing.T) {
	root := t.TempDir()
	scenarioRoot := filepath.Join(root, "scenarios", "hello-desktop")
	apiRoot := filepath.Join(scenarioRoot, "api")
	if err := os.MkdirAll(filepath.Join(apiRoot, "desktop"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(apiRoot, "go.mod"), []byte("module example.test/hello/api\n\ngo 1.25.0\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(apiRoot, "desktop", "embed.go"), []byte("package desktop\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	uiDist := filepath.Join(scenarioRoot, "ui", "dist")
	if err := os.MkdirAll(uiDist, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(uiDist, "index.html"), []byte("<main>fixture</main>"), 0o644); err != nil {
		t.Fatal(err)
	}
	output := filepath.Join(scenarioRoot, "platforms", WailsStrategyID)
	store := newMockBuildStore()
	store.Create("wails-test")
	service := &DefaultService{vrooliRoot: root, templateDir: filepath.Join("..", "..", "templates"), builds: store}
	service.runWailsProjectGenerator("wails-test", &DesktopConfig{AppName: "hello-desktop", AppDisplayName: "Hello Desktop", AppDescription: "Embedded reference app", ScenarioPath: uiDist, OutputPath: output})
	status, ok := store.Get("wails-test")
	if !ok || status.Status != "ready" {
		t.Fatalf("generation status = %#v, want ready", status)
	}
	mainSource, err := os.ReadFile(filepath.Join(output, "main.go"))
	if err != nil {
		t.Fatal(err)
	}
	for _, expected := range []string{"example.test/hello/api/desktop", "lifecycle.Start(ctx)", "application.AssetOptions{Handler: assetHandler}", "app.Window.NewWithOptions", "events.Common.WindowShow", "packaged-launch.v1", "/__s2d/launch-mark", "validRendererMark", "isAPIRequest(r)", "AppInteractiveSource: r.appInteractiveSource"} {
		if !strings.Contains(string(mainSource), expected) {
			t.Errorf("generated main.go missing %q", expected)
		}
	}
	if strings.Contains(string(mainSource), "net.Listen") || strings.Contains(string(mainSource), "http.ListenAndServe") {
		t.Fatal("Wails project must not create a TCP API listener")
	}
	goMod, err := os.ReadFile(filepath.Join(output, "go.mod"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(goMod), "replace example.test/hello/api => ../../api") {
		t.Fatalf("generated go.mod does not point at the scenario API: %s", goMod)
	}
	if _, err := os.Stat(filepath.Join(output, "assets", "index.html")); err != nil {
		t.Fatalf("expected copied UI asset: %v", err)
	}
	index, err := os.ReadFile(filepath.Join(output, "assets", "index.html"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(index), `<script src="/__s2d/launch-observer.js"></script>`) {
		t.Fatalf("Wails UI entry point does not load the launch observer: %s", index)
	}
	observer, err := os.ReadFile(filepath.Join(output, "assets", "__s2d", "launch-observer.js"))
	if err != nil {
		t.Fatal(err)
	}
	for _, expected := range []string{"first-contentful-paint", "app-interactive", "500", "longtask", "__s2d/launch-mark"} {
		if !strings.Contains(string(observer), expected) {
			t.Errorf("Wails renderer observer missing %q", expected)
		}
	}
	taskfile, err := os.ReadFile(filepath.Join(output, "Taskfile.yml"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(taskfile), "production,{{.EXTRA_TAGS | default \"gtk3\"}}") || !strings.Contains(string(taskfile), "go build -tags") {
		t.Fatalf("Wails build task does not preserve the GTK3 build tag: %s", taskfile)
	}
	if _, err := os.Stat(filepath.Join(output, "build", "config.yml")); err != nil {
		t.Fatalf("expected Wails app config: %v", err)
	}
}

func TestInstallWailsLaunchObserverPreservesPageAndIsIdempotent(t *testing.T) {
	assetsDir := t.TempDir()
	indexPath := filepath.Join(assetsDir, "index.html")
	index := `<!doctype html><html><head><title>Fixture</title></head><body><main>hello</main></body></html>`
	if err := os.WriteFile(indexPath, []byte(index), 0o644); err != nil {
		t.Fatal(err)
	}
	templateRoot := filepath.Join("..", "..", "templates")
	for range 2 {
		if err := installWailsLaunchObserver(templateRoot, assetsDir); err != nil {
			t.Fatal(err)
		}
	}
	updated, err := os.ReadFile(indexPath)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Count(string(updated), `<script src="/__s2d/launch-observer.js"></script>`) != 1 {
		t.Fatalf("launch observer injection is not idempotent: %s", updated)
	}
	for _, preserved := range []string{"<title>Fixture</title>", "<main>hello</main>"} {
		if !strings.Contains(string(updated), preserved) {
			t.Errorf("launch observer injection removed %q: %s", preserved, updated)
		}
	}
}

func TestWailsGeneratorRequiresEmbeddingContract(t *testing.T) {
	root := t.TempDir()
	apiRoot := filepath.Join(root, "scenarios", "fixture", "api")
	if err := os.MkdirAll(apiRoot, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(apiRoot, "go.mod"), []byte("module example.test/fixture\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	store := newMockBuildStore()
	store.Create("missing-embed")
	service := &DefaultService{vrooliRoot: root, builds: store}
	service.runWailsProjectGenerator("missing-embed", &DesktopConfig{AppName: "fixture", OutputPath: filepath.Join(root, "out")})
	status, _ := store.Get("missing-embed")
	if status.Status != "failed" || !strings.Contains(strings.Join(status.ErrorLog, "\n"), "api/desktop/embed.go") {
		t.Fatalf("missing embed contract status = %#v, want explicit failure", status)
	}
}
