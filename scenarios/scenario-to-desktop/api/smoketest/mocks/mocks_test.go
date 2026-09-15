package mocks

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"scenario-to-desktop-api/screenrecording"
	"scenario-to-desktop-api/smoketest"
)

func TestMockStoreAndCancellationPreserveOrchestrationSemantics(t *testing.T) {
	store := NewMockStore().AddStatus(&smoketest.Status{SmokeTestID: "one", Status: "running"})
	store.Save(&smoketest.Status{SmokeTestID: "two", Status: "queued"})
	if got, ok := store.Get("two"); !ok || got.Status != "queued" {
		t.Fatalf("Get() = %#v, %v", got, ok)
	}
	if !store.Update("one", func(status *smoketest.Status) { status.Status = "passed" }) || store.Statuses["one"].Status != "passed" || store.Update("missing", func(*smoketest.Status) {}) {
		t.Fatalf("Update state=%#v calls=%#v", store.Statuses, store.UpdateCalls)
	}
	if len(store.UpdateCalls) != 2 {
		t.Fatalf("update calls = %#v", store.UpdateCalls)
	}
	called := false
	cancels := NewMockCancelManager()
	cancels.SetCancel("one", func() { called = true })
	if cancel := cancels.TakeCancel("one"); cancel == nil {
		t.Fatal("missing cancel")
	} else {
		cancel()
	}
	if !called || cancels.TakeCancel("one") != nil {
		t.Fatal("take cancel lifecycle incorrect")
	}
	cancels.SetCancel("two", func() {})
	cancels.Clear("two")
	if len(cancels.ClearCalls) != 1 || cancels.TakeCancel("two") != nil {
		t.Fatal("clear lifecycle incorrect")
	}
}

func TestMockExecutorsResolversAndParsersExposeConfiguredBehavior(t *testing.T) {
	exec := NewMockProcessExecutor().AddLookPath("xvfb-run", "/usr/bin/xvfb-run")
	exec.ExecuteResult.Output = "combined"
	output, err := exec.Execute(context.Background(), "/work", "cmd", []string{"arg"}, []string{"A=B"}, time.Second)
	if err != nil || output != "combined" || len(exec.ExecuteCalls) != 1 {
		t.Fatalf("Execute = %q %v calls=%#v", output, err, exec.ExecuteCalls)
	}
	result, err := exec.ExecuteWithResult(context.Background(), "", "cmd", nil, nil, time.Second)
	if err != nil || result.Combined != "combined" || len(exec.ExecuteCalls) != 2 {
		t.Fatalf("ExecuteWithResult = %#v %v", result, err)
	}
	if path, err := exec.LookPath("xvfb-run"); err != nil || path == "" {
		t.Fatalf("LookPath = %q, %v", path, err)
	}
	expected := errors.New("missing")
	exec.AddLookPathError("missing", expected)
	if _, err := exec.LookPath("missing"); !errors.Is(err, expected) {
		t.Fatalf("LookPath error = %v", err)
	}

	resolver := NewMockPlatformResolver()
	resolver.ResolveResult.Cmd, resolver.ResolveResult.Args, resolver.ResolveResult.Display = "run", []string{"--headless"}, ":99"
	cmd, args, display, err := resolver.ResolveCommand("linux", "/artifact")
	if err != nil || cmd != "run" || len(args) != 1 || display != ":99" {
		t.Fatalf("ResolveCommand = %q %#v %q %v", cmd, args, display, err)
	}
	resolver.HeadlessResult.Needed, resolver.HeadlessResult.WrapperCmd = true, "xvfb-run"
	if needed, wrapper, _, _ := resolver.RequiresHeadlessWrapper(); !needed || wrapper != "xvfb-run" {
		t.Fatal("headless result lost")
	}

	parser := NewMockOutputParser()
	parser.Result.Passed, parser.LifecycleStateResult, parser.SessionIDResult = true, "ready", "session-1"
	if !parser.ParseResult("output").Passed || parser.ExtractLastLifecycleState("output") != "ready" || parser.ExtractSessionID("output") != "session-1" {
		t.Fatal("parser configured values lost")
	}
	telemetry := NewMockTelemetryPathResolver()
	telemetry.ExtractResult, telemetry.ResolveResult = "/tmp/events.json", "/artifact/events.json"
	telemetry.ReadEventsResult.Events = []map[string]interface{}{{"event": "ready"}}
	if telemetry.ExtractFromOutput("output") == "" || telemetry.ResolveFromArtifact("linux", "/artifact", "demo") == "" || len(mustReadTelemetry(t, telemetry)) != 1 {
		t.Fatal("telemetry resolver values lost")
	}
}

func mustReadTelemetry(t *testing.T, resolver *MockTelemetryPathResolver) []map[string]interface{} {
	t.Helper()
	events, err := resolver.ReadTelemetryEvents("/tmp/events.json", 10)
	if err != nil {
		t.Fatal(err)
	}
	return events
}

func TestMockFilesystemLoggerAndTelemetryCaptureCalls(t *testing.T) {
	fs := NewMockFileSystem().AddFile("/file", []byte("data")).AddDirectory("/dir")
	if info, err := fs.Stat("/file"); err != nil || info.Size() != 4 {
		t.Fatalf("Stat file = %#v %v", info, err)
	}
	if info, err := fs.Stat("/dir"); err != nil || !info.IsDir() {
		t.Fatalf("Stat dir = %#v %v", info, err)
	}
	if data, err := fs.ReadFile("/file"); err != nil || string(data) != "data" {
		t.Fatalf("ReadFile = %q %v", data, err)
	}
	if err := fs.Chmod("/file", 0o755); err != nil || len(fs.ChmodCalls) != 1 || fs.ChmodCalls[0].Mode != 0o755 {
		t.Fatalf("Chmod = %v calls=%#v", err, fs.ChmodCalls)
	}
	if _, err := fs.Stat("missing"); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("missing stat = %v", err)
	}
	logger := NewMockLogger()
	logger.Info("info", "id", 1)
	logger.Warn("warn")
	logger.Error("error")
	if len(logger.InfoCalls) != 1 || len(logger.WarnCalls) != 1 || len(logger.ErrorCalls) != 1 {
		t.Fatalf("log calls = %#v", logger)
	}
	ingestor := NewMockTelemetryIngestor()
	ingestor.IngestResult.ID, ingestor.IngestResult.Count = "batch", 1
	if id, count, err := ingestor.IngestEvents("demo", "instance", "smoke", []map[string]interface{}{{"event": "ready"}}); err != nil || id != "batch" || count != 1 || len(ingestor.IngestCalls) != 1 {
		t.Fatalf("IngestEvents = %q %d %v calls=%#v", id, count, err, ingestor.IngestCalls)
	}
}

func TestMockTimePrerequisiteTelemetryAndRecordingContracts(t *testing.T) {
	now := time.Date(2026, 7, 27, 12, 0, 0, 0, time.UTC)
	clock := NewMockClock(now)
	if got := clock.Now(); !got.Equal(now) {
		t.Fatalf("Now = %v", got)
	}
	clock.Advance(time.Minute)
	if <-clock.After(time.Second) != clock.CurrentTime || len(clock.AfterCalls) != 1 {
		t.Fatalf("clock state = %#v", clock)
	}
	clock.Set(now)
	if !clock.CurrentTime.Equal(now) {
		t.Fatal("clock set failed")
	}

	prereqs := NewMockPrerequisiteChecker()
	prereqs.CheckAllResult = []smoketest.PrerequisiteResult{{Kind: smoketest.PrereqDisplay, Passed: true}}
	if results := prereqs.CheckAll("/artifact", "linux", 19000); len(results) != 1 || len(prereqs.CheckAllCalls) != 1 || prereqs.HasFatalFailure(results) {
		t.Fatalf("prerequisites=%#v calls=%#v", results, prereqs.CheckAllCalls)
	}
	prereqs.HasFatalFailureResult = true
	if !prereqs.HasFatalFailure(nil) {
		t.Fatal("configured fatal failure lost")
	}

	extractor := NewMockTelemetryErrorExtractor()
	errorValue := &smoketest.TelemetryError{Message: "app failed"}
	extractor.WithLatestError(errorValue).WithLatestErrorForSession(errorValue)
	if latest, err := extractor.ExtractLatestError("/events"); err != nil || latest != errorValue {
		t.Fatalf("latest = %#v %v", latest, err)
	}
	if latest, err := extractor.ExtractLatestErrorForSession("/events", "session"); err != nil || latest != errorValue {
		t.Fatalf("session latest = %#v %v", latest, err)
	}
	extractor.ExtractErrorsResult.Errors = []smoketest.TelemetryError{{Message: "older"}}
	if values, err := extractor.ExtractErrors("/events", 5); err != nil || len(values) != 1 || len(extractor.ExtractErrorsCalls) != 1 {
		t.Fatalf("errors = %#v %v", values, err)
	}

	recorder := NewMockRecorder()
	recorder.StartResult.CaptureID = "capture-1"
	recorder.StopResult.VideoPath, recorder.StopResult.DurationMs, recorder.StopResult.FileSizeBytes = "/video.mp4", 1200, 42
	if id, err := recorder.StartCapture(context.Background(), screenrecording.CaptureConfig{Display: ":99", Width: 1280, Height: 720, FPS: 15}); err != nil || id != "capture-1" {
		t.Fatalf("StartCapture = %q %v", id, err)
	}
	if result, err := recorder.StopCapture(context.Background(), "capture-1"); err != nil || result.VideoPath != "/video.mp4" || result.DurationMs != 1200 || len(recorder.StopCalls) != 1 {
		t.Fatalf("StopCapture = %#v %v", result, err)
	}
}

func TestMockUtilityOverridesAndDisplayLifecycle(t *testing.T) {
	env := NewMockEnvironmentReader().SetEnv("SMOKE_TEST", "1")
	env.HomeDir = "/home/tester"
	if env.GetEnv("SMOKE_TEST") != "1" {
		t.Fatal("environment value lost")
	}
	if home, err := env.UserHomeDir(); err != nil || home != "/home/tester" {
		t.Fatalf("UserHomeDir = %q, %v", home, err)
	}
	homeErr := errors.New("home")
	env.HomeDirErr = homeErr
	if _, err := env.UserHomeDir(); !errors.Is(err, homeErr) {
		t.Fatalf("HomeDirErr = %v", err)
	}

	fs := NewMockFileSystem().AddFile("/file", []byte("content")).AddDirectory("/directory")
	entry := &MockDirEntry{EntryName: "file", EntryInfo: &MockFileInfo{NameVal: "file"}}
	fs.AddDirEntries("/directory", []os.DirEntry{entry})
	if entries, err := fs.ReadDir("/directory"); err != nil || len(entries) != 1 || entries[0].Name() != "file" {
		t.Fatalf("ReadDir = %#v, %v", entries, err)
	}
	if _, err := fs.ReadDir("missing"); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("missing ReadDir = %v", err)
	}
	info := &MockFileInfo{NameVal: "custom", SizeVal: 8, ModeVal: 0o600, ModTimeVal: time.Now()}
	fs.AddFileInfo("/custom", info)
	if got, err := fs.Stat("/custom"); err != nil || got != info || got.Name() != "custom" || got.Mode() != 0o600 {
		t.Fatalf("custom Stat = %#v, %v", got, err)
	}
	if got, err := entry.Info(); err != nil || got != entry.EntryInfo || entry.Type() != 0 || entry.IsDir() {
		t.Fatalf("DirEntry = %#v, %v", got, err)
	}
	statErr := errors.New("stat")
	fs.StatFunc = func(string) (os.FileInfo, error) { return nil, statErr }
	if _, err := fs.Stat("/file"); !errors.Is(err, statErr) {
		t.Fatalf("StatFunc = %v", err)
	}
	readErr := errors.New("read-dir")
	fs.ReadDirFunc = func(string) ([]os.DirEntry, error) { return nil, readErr }
	if _, err := fs.ReadDir("/directory"); !errors.Is(err, readErr) {
		t.Fatalf("ReadDirFunc = %v", err)
	}
	openErr := errors.New("open")
	fs.OpenFunc = func(string) (*os.File, error) { return nil, openErr }
	if _, err := fs.Open("/file"); !errors.Is(err, openErr) {
		t.Fatalf("OpenFunc = %v", err)
	}
	chmodErr := errors.New("chmod")
	fs.ChmodFunc = func(string, os.FileMode) error { return chmodErr }
	if err := fs.Chmod("/file", 0o700); !errors.Is(err, chmodErr) || len(fs.ChmodCalls) != 1 {
		t.Fatalf("Chmod = %v, %#v", err, fs.ChmodCalls)
	}

	chain := NewMockTelemetryChainExecutor()
	params := smoketest.TelemetryChainParams{ScenarioName: "demo"}
	chain.ExecuteResult = smoketest.TelemetryResult{Path: "/events"}
	if result := chain.Execute(context.Background(), params); result.Path != "/events" || len(chain.ExecuteCalls) != 1 {
		t.Fatalf("Execute = %#v, %#v", result, chain.ExecuteCalls)
	}
	chain.ExecuteFunc = func(context.Context, smoketest.TelemetryChainParams) smoketest.TelemetryResult {
		return smoketest.TelemetryResult{EventsIngested: 1}
	}
	if result := chain.Execute(context.Background(), params); result.EventsIngested != 1 {
		t.Fatalf("ExecuteFunc = %#v", result)
	}

	displays := NewMockDisplayManager()
	displays.CreateResult.DisplayID = ":88"
	id, cleanup, err := displays.CreateDisplay(800, 600)
	if err != nil || id != ":88" || len(displays.CreateCalls) != 1 {
		t.Fatalf("CreateDisplay = %q, %v, %#v", id, err, displays.CreateCalls)
	}
	cleanup()
	if !displays.CleanupCalled {
		t.Fatal("display cleanup was not recorded")
	}
	managed, err := displays.CreateManagedDisplay(1024, 768)
	if err != nil || managed.DisplayID != ":88" || managed.Width != 1024 {
		t.Fatalf("CreateManagedDisplay = %#v, %v", managed, err)
	}
	displayErr := errors.New("display")
	displays.CreateResult.Err = displayErr
	if _, _, err := displays.CreateDisplay(1, 1); !errors.Is(err, displayErr) {
		t.Fatalf("CreateDisplay error = %v", err)
	}
	if _, err := displays.CreateManagedDisplay(1, 1); !errors.Is(err, displayErr) {
		t.Fatalf("CreateManagedDisplay error = %v", err)
	}
}
