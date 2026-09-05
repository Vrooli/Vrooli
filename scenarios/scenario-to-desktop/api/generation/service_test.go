package generation

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"scenario-to-desktop-api/signing"
)

// mockBuildStore implements BuildStore for testing QueueBuild.
type mockBuildStore struct {
	createCalls int
	updateCalls int
	statuses    map[string]*BuildStatus
}

func newMockBuildStore() *mockBuildStore {
	return &mockBuildStore{
		statuses: make(map[string]*BuildStatus),
	}
}

func (m *mockBuildStore) Create(buildID string) *BuildStatus {
	m.createCalls++
	status := &BuildStatus{
		BuildID:   buildID,
		Status:    "building",
		StartedAt: time.Now(),
		BuildLog:  []string{},
		ErrorLog:  []string{},
		Artifacts: map[string]string{},
		Metadata:  map[string]interface{}{},
	}
	m.statuses[buildID] = status
	return status
}

func (m *mockBuildStore) Get(buildID string) (*BuildStatus, bool) {
	status, ok := m.statuses[buildID]
	return status, ok
}

func (m *mockBuildStore) Update(buildID string, fn func(status *BuildStatus)) {
	m.updateCalls++
	if status, ok := m.statuses[buildID]; ok {
		fn(status)
	}
}

// mockRecordStore implements RecordStore for testing.
type mockRecordStore struct {
	upsertCalls int
	records     map[string]*DesktopAppRecord
}

func newMockRecordStore() *mockRecordStore {
	return &mockRecordStore{
		records: make(map[string]*DesktopAppRecord),
	}
}

func (m *mockRecordStore) Upsert(record *DesktopAppRecord) error {
	m.upsertCalls++
	m.records[record.ID] = record
	return nil
}

func TestQueueBuild_SavesBuildToStore(t *testing.T) {
	buildStore := newMockBuildStore()
	recordStore := newMockRecordStore()

	service := NewService(
		WithVrooliRoot("/tmp/vrooli"),
 WithStagingRoot(func() (string, error) { return t.TempDir(), nil }),
		WithBuildStore(buildStore),
		WithRecordStore(recordStore),
	)

	config := &DesktopConfig{
		AppName:      "test-app",
		LocationMode: "proper",
	}

	// Call QueueBuild
	result := service.QueueBuild(config, nil, false)

	// Verify build ID was returned
	if result == nil {
		t.Fatal("expected non-nil build status")
	}
	if result.BuildID == "" {
		t.Fatal("expected build ID to be set")
	}

	// Verify Create was called (this is the critical fix)
	if buildStore.createCalls != 1 {
		t.Errorf("expected Create to be called once, got %d calls", buildStore.createCalls)
	}

	// Verify Update was called after Create to set additional fields
	if buildStore.updateCalls != 1 {
		t.Errorf("expected Update to be called once, got %d calls", buildStore.updateCalls)
	}

	// Verify the build can be retrieved from the store
	stored, ok := buildStore.Get(result.BuildID)
	if !ok {
		t.Fatal("expected build to be retrievable from store after QueueBuild")
	}
	if stored.BuildID != result.BuildID {
		t.Errorf("stored build ID %q doesn't match result %q", stored.BuildID, result.BuildID)
	}
}

func TestQueueBuild_SetsOutputPath(t *testing.T) {
	buildStore := newMockBuildStore()

	service := NewService(
		WithVrooliRoot("/tmp/vrooli"),
 WithStagingRoot(func() (string, error) { return t.TempDir(), nil }),
		WithBuildStore(buildStore),
	)

	config := &DesktopConfig{
		AppName:      "test-app",
		LocationMode: "proper",
	}

	result := service.QueueBuild(config, nil, false)

	// Verify output path was set
	if result.OutputPath == "" {
		t.Error("expected OutputPath to be set")
	}

	// Verify it's stored in the store
	stored, ok := buildStore.Get(result.BuildID)
	if !ok {
		t.Fatal("expected build to be in store")
	}
	if stored.OutputPath != result.OutputPath {
		t.Errorf("stored OutputPath %q doesn't match result %q", stored.OutputPath, result.OutputPath)
	}
}

func TestQueueBuild_WithMetadata(t *testing.T) {
	buildStore := newMockBuildStore()

	service := NewService(
		WithVrooliRoot("/tmp/vrooli"),
 WithStagingRoot(func() (string, error) { return t.TempDir(), nil }),
		WithBuildStore(buildStore),
	)

	config := &DesktopConfig{
		AppName:      "test-app",
		ServerType:   "external",
		LocationMode: "proper",
	}

	metadata := &ScenarioMetadata{
		Name:       "test-app",
		UIDistPath: "/path/to/dist",
		Category:   "utility",
		Version:    "1.0.0",
	}

	result := service.QueueBuild(config, metadata, true)

	stored, ok := buildStore.Get(result.BuildID)
	if !ok {
		t.Fatal("expected build to be in store")
	}

	// Verify metadata was stored
	if stored.Metadata["auto_detected"] != true {
		t.Error("expected auto_detected metadata to be true")
	}
	if stored.Metadata["ui_dist_path"] != "/path/to/dist" {
		t.Errorf("expected ui_dist_path metadata, got %v", stored.Metadata["ui_dist_path"])
	}
	if stored.Metadata["category"] != "utility" {
		t.Errorf("expected category metadata, got %v", stored.Metadata["category"])
	}
}

func TestQueueBuild_PersistsRecord(t *testing.T) {
	buildStore := newMockBuildStore()
	recordStore := newMockRecordStore()

	service := NewService(
		WithVrooliRoot("/tmp/vrooli"),
 WithStagingRoot(func() (string, error) { return t.TempDir(), nil }),
		WithBuildStore(buildStore),
		WithRecordStore(recordStore),
	)

	config := &DesktopConfig{
		AppName:        "test-app",
		AppDisplayName: "Test Application",
		TemplateType:   "universal",
		Framework:      "electron",
		LocationMode:   "proper",
		DeploymentMode: "proxy",
	}

	result := service.QueueBuild(config, nil, false)

	// Verify record was persisted
	if recordStore.upsertCalls != 1 {
		t.Errorf("expected Upsert to be called once, got %d calls", recordStore.upsertCalls)
	}

	record, ok := recordStore.records[result.BuildID]
	if !ok {
		t.Fatal("expected record to be in store")
	}
	if record.ScenarioName != "test-app" {
		t.Errorf("expected scenario name 'test-app', got %q", record.ScenarioName)
	}
	if record.AppDisplayName != "Test Application" {
		t.Errorf("expected display name 'Test Application', got %q", record.AppDisplayName)
	}
}

func TestQueueBuild_GeneratesUniqueBuildIDs(t *testing.T) {
	buildStore := newMockBuildStore()

	service := NewService(
		WithVrooliRoot("/tmp/vrooli"),
 WithStagingRoot(func() (string, error) { return t.TempDir(), nil }),
		WithBuildStore(buildStore),
	)

	config := &DesktopConfig{
		AppName:      "test-app",
		LocationMode: "proper",
	}

	// Generate multiple builds
	ids := make(map[string]bool)
	for i := 0; i < 100; i++ {
		result := service.QueueBuild(config, nil, false)
		if ids[result.BuildID] {
			t.Errorf("duplicate build ID generated: %s", result.BuildID)
		}
		ids[result.BuildID] = true
	}

	// Verify we got 100 unique IDs
	if len(ids) != 100 {
		t.Errorf("expected 100 unique IDs, got %d", len(ids))
	}
}

func TestQueueBuild_LocationModes(t *testing.T) {
	tests := []struct {
		name          string
		locationMode  string
		expectStaging bool
	}{
		{"proper mode", "proper", false},
		{"temp mode", "temp", true},
		{"staging mode", "staging", true},
		{"custom mode", "custom", false},
		{"empty defaults to proper", "", false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			buildStore := newMockBuildStore()
			recordStore := newMockRecordStore()

			service := NewService(
				WithVrooliRoot("/tmp/vrooli"),
 WithStagingRoot(func() (string, error) { return t.TempDir(), nil }),
				WithBuildStore(buildStore),
				WithRecordStore(recordStore),
			)

			config := &DesktopConfig{
				AppName:      "test-app",
				LocationMode: tc.locationMode,
			}

			result := service.QueueBuild(config, nil, false)

			record := recordStore.records[result.BuildID]
			if record == nil {
				t.Fatal("expected record to be created")
			}

			if tc.expectStaging {
				if record.StagingPath == "" {
					t.Error("expected StagingPath to be set for temp/staging mode")
				}
			}
		})
	}
}

func TestQueueBuild_NilBuildStore(t *testing.T) {
	// Verify QueueBuild works even without a build store
	service := NewService(
		WithVrooliRoot("/tmp/vrooli"),
 WithStagingRoot(func() (string, error) { return t.TempDir(), nil }),
	)

	config := &DesktopConfig{
		AppName:      "test-app",
		LocationMode: "proper",
	}

	// Should not panic
	result := service.QueueBuild(config, nil, false)

	if result == nil {
		t.Fatal("expected non-nil result")
	}
	if result.BuildID == "" {
		t.Error("expected build ID even without store")
	}
}

func TestNewService_Defaults(t *testing.T) {
	service := NewService()

	if service == nil {
		t.Fatal("expected service to be created")
	}
	if service.logger == nil {
		t.Error("expected default logger to be set")
	}
}

func TestNewService_WithOptions(t *testing.T) {
	buildStore := newMockBuildStore()
	recordStore := newMockRecordStore()

	service := NewService(
		WithVrooliRoot("/custom/root"),
		WithTemplateDir("/custom/templates"),
		WithBuildStore(buildStore),
		WithRecordStore(recordStore),
	)

	if service.vrooliRoot != "/custom/root" {
		t.Errorf("expected vrooliRoot '/custom/root', got %q", service.vrooliRoot)
	}
	if service.templateDir != "/custom/templates" {
		t.Errorf("expected templateDir '/custom/templates', got %q", service.templateDir)
	}
}

func TestStandardOutputPath(t *testing.T) {
	service := NewService(
		WithVrooliRoot("/home/user/vrooli"),
	)

	path := service.StandardOutputPath("my-scenario")
	expected := "/home/user/vrooli/scenarios/my-scenario/platforms/electron"

	if path != expected {
		t.Errorf("expected %q, got %q", expected, path)
	}
}

func TestScenarioRoot(t *testing.T) {
	service := NewService(
		WithVrooliRoot("/home/user/vrooli"),
	)

	path := service.ScenarioRoot("my-scenario")
	expected := "/home/user/vrooli/scenarios/my-scenario"

	if path != expected {
		t.Errorf("expected %q, got %q", expected, path)
	}
}

type recordingIconSyncer struct {
	err   error
	calls int
}

func (s *recordingIconSyncer) SyncIcons(_ string, _ string, _ func(string, map[string]interface{})) error {
	s.calls++
	return s.err
}

type recordingBundlePackager struct {
	result *BundlePackageResult
	err    error
	calls  int
}

func (p *recordingBundlePackager) Package(_ string, _ string, _ []string) (*BundlePackageResult, error) {
	p.calls++
	return p.result, p.err
}

func TestPostGenerateStepsCapturesIconWarningsAndBundleMetadata(t *testing.T) {
	iconSyncer := &recordingIconSyncer{err: errors.New("icon unavailable")}
	packager := &recordingBundlePackager{result: &BundlePackageResult{
		BundleDir: "bundle", ManifestPath: "bundle.json", RuntimeBinaries: []string{"runtime"}, TotalSizeBytes: 42, TotalSizeHuman: "42 B",
		SizeWarning: &SizeWarning{Level: "warn", Message: "large bundle"},
	}}
	service := NewService(WithIconSyncer(iconSyncer), WithBundlePackager(packager))
	status := &BuildStatus{Status: "ready", Metadata: map[string]interface{}{}, BuildLog: []string{}}
	service.postGenerateSteps(&DesktopConfig{ScenarioName: "demo", OutputPath: "out", DeploymentMode: "bundled", BundleManifestPath: "bundle.json", Platforms: []string{"linux"}}, status)
	if iconSyncer.calls != 1 || packager.calls != 1 || status.Status != "ready" || status.Metadata["bundle_dir"] != "bundle" || !strings.Contains(strings.Join(status.BuildLog, "\n"), "Icon sync warning") {
		t.Fatalf("post generation status = %#v; icon=%d package=%d", status, iconSyncer.calls, packager.calls)
	}

	packager.err = errors.New("package failed")
	failed := &BuildStatus{Status: "ready", Metadata: map[string]interface{}{}}
	service.postGenerateSteps(&DesktopConfig{DeploymentMode: "bundled", BundleManifestPath: "bundle.json"}, failed)
	if failed.Status != "failed" || len(failed.ErrorLog) != 1 {
		t.Fatalf("bundle failure status = %#v", failed)
	}
}

func TestWriteConfigFileProducesInspectableJSON(t *testing.T) {
	builds := newMockBuildStore()
	builds.Create("build")
	service := NewService(WithBuildStore(builds))
	path, err := service.writeConfigFile("build", &DesktopConfig{AppName: "demo", Features: map[string]interface{}{"offline": true}})
	if err != nil {
		t.Fatalf("write config: %v", err)
	}
	t.Cleanup(func() { _ = os.Remove(path) })
	data, err := os.ReadFile(path)
	if err != nil || !strings.Contains(string(data), `"app_name": "demo"`) {
		t.Fatalf("config content = %s, %v", data, err)
	}
	if filepath.Base(path) != "desktop-config-build.json" {
		t.Fatalf("config path = %q", path)
	}
}

func TestGenerateRecordsTemplateGeneratorSuccessAndFailure(t *testing.T) {
	builds := newMockBuildStore()
	iconSyncer := &recordingIconSyncer{}
	templateDir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(templateDir, "build-tools", "dist"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(templateDir, "build-tools", "dist", "template-generator.js"), []byte("// fixture"), 0o644); err != nil {
		t.Fatal(err)
	}
	binDir := t.TempDir()
	writeNodeFixture(t, binDir, "printf 'generated %s\\n' \"$1\"")
	t.Setenv("PATH", binDir+string(os.PathListSeparator)+os.Getenv("PATH"))
	service := NewService(WithTemplateDir(templateDir), WithBuildStore(builds), WithIconSyncer(iconSyncer))
	builds.Create("success")
	output := t.TempDir()
	service.Generate("success", &DesktopConfig{ScenarioName: "demo", AppName: "demo", OutputPath: output})
	status, ok := builds.Get("success")
	if !ok || status.Status != "ready" || status.Artifacts["output_path"] != output || status.CompletedAt == nil || iconSyncer.calls != 1 {
		t.Fatalf("successful generation status = %#v; icon calls=%d", status, iconSyncer.calls)
	}
	if !strings.Contains(strings.Join(status.BuildLog, "\n"), "generated") {
		t.Fatalf("generator output was not retained: %#v", status.BuildLog)
	}

	failureBinDir := t.TempDir()
	writeNodeFixture(t, failureBinDir, "echo generator-failed >&2; exit 7")
	t.Setenv("PATH", failureBinDir+string(os.PathListSeparator)+os.Getenv("PATH"))
	builds.Create("failed")
	service.Generate("failed", &DesktopConfig{ScenarioName: "demo", AppName: "demo", OutputPath: output})
	status, ok = builds.Get("failed")
	if !ok || status.Status != "failed" || status.CompletedAt == nil || len(status.ErrorLog) < 2 || !strings.Contains(strings.Join(status.ErrorLog, "\n"), "generator-failed") {
		t.Fatalf("failed generation status = %#v", status)
	}
}

func TestGenerateSigningArtifactsIsNoOpWhenDisabledAndWritesMacOSFiles(t *testing.T) {
	service := NewService()
	if err := service.generateSigningArtifacts(&DesktopConfig{}); err != nil {
		t.Fatalf("disabled signing generated an error: %v", err)
	}
	if err := service.generateSigningArtifacts(&DesktopConfig{CodeSigning: &signing.SigningConfig{Enabled: true}}); err == nil || !strings.Contains(err.Error(), "output path not set") {
		t.Fatalf("missing output path error = %v", err)
	}
	output := t.TempDir()
	config := &DesktopConfig{OutputPath: output, CodeSigning: &signing.SigningConfig{
		Enabled: true,
		MacOS:   &signing.MacOSSigningConfig{},
	}}
	if err := service.generateSigningArtifacts(config); err != nil {
		t.Fatalf("generate macOS signing artifacts: %v", err)
	}
	if config.CodeSigning.MacOS.EntitlementsFile != "entitlements.mac.plist" {
		t.Fatalf("default entitlements path = %q", config.CodeSigning.MacOS.EntitlementsFile)
	}
	if _, err := os.Stat(filepath.Join(output, "entitlements.mac.plist")); err != nil {
		t.Fatalf("entitlements artifact missing: %v", err)
	}
}

func writeNodeFixture(t *testing.T, directory, body string) {
	t.Helper()
	path := filepath.Join(directory, "node")
	if err := os.WriteFile(path, []byte("#!/bin/sh\n"+body+"\n"), 0o755); err != nil {
		t.Fatal(err)
	}
}
