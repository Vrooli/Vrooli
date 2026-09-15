package mocks

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"scenario-to-desktop-api/signing/types"
)

func TestMockFileSystemModelsFileLifecycleAndOverrides(t *testing.T) {
	fs := NewMockFileSystem().AddFile("cert.p12", []byte("certificate")).AddDirectory("keys")
	if !fs.Exists("cert.p12") || !fs.Exists("keys") || fs.Exists("missing") {
		t.Fatal("Exists did not reflect configured entries")
	}
	contents, err := fs.ReadFile("cert.p12")
	if err != nil || string(contents) != "certificate" {
		t.Fatalf("ReadFile = %q, %v", contents, err)
	}
	if _, err := fs.ReadFile("missing"); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("missing ReadFile error = %v", err)
	}
	if err := fs.WriteFile("generated.p12", []byte("generated"), 0o600); err != nil || !fs.Exists("generated.p12") {
		t.Fatalf("WriteFile = %v", err)
	}
	file, err := fs.Stat("generated.p12")
	if err != nil || file.Size() != int64(len("generated")) || file.IsDir() {
		t.Fatalf("file Stat = %#v, %v", file, err)
	}
	dir, err := fs.Stat("keys")
	if err != nil || !dir.IsDir() {
		t.Fatalf("directory Stat = %#v, %v", dir, err)
	}
	if _, err := fs.Stat("missing"); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("missing Stat error = %v", err)
	}
	if err := fs.MkdirAll("nested", 0o755); err != nil || !fs.Exists("nested") {
		t.Fatalf("MkdirAll = %v", err)
	}
	if err := fs.Remove("generated.p12"); err != nil || fs.Exists("generated.p12") {
		t.Fatalf("Remove = %v", err)
	}

	fs.ExistsFunc = func(path string) bool { return path == "virtual" }
	if !fs.Exists("virtual") || fs.Exists("cert.p12") {
		t.Fatal("ExistsFunc was not used")
	}
	readErr := errors.New("read override")
	fs.ReadFileFunc = func(string) ([]byte, error) { return nil, readErr }
	if _, err := fs.ReadFile("cert.p12"); !errors.Is(err, readErr) {
		t.Fatalf("ReadFileFunc error = %v", err)
	}
	writeErr := errors.New("write override")
	fs.WriteFileFunc = func(string, []byte, os.FileMode) error { return writeErr }
	if err := fs.WriteFile("x", nil, 0); !errors.Is(err, writeErr) {
		t.Fatalf("WriteFileFunc error = %v", err)
	}
	customInfo := &mockFileInfo{name: "custom", size: 9, mode: 0o600, modTime: time.Now()}
	fs.StatFunc = func(string) (os.FileInfo, error) { return customInfo, nil }
	if got, err := fs.Stat("anything"); err != nil || got != customInfo {
		t.Fatalf("StatFunc = %#v, %v", got, err)
	}
}

func TestMockRepositoryAndScenarioLocatorSupportPlatformScopedSigning(t *testing.T) {
	ctx := context.Background()
	repository := NewMockRepository()
	config := &types.SigningConfig{}
	if err := repository.Save(ctx, "demo", config); err != nil {
		t.Fatal(err)
	}
	if got, err := repository.Get(ctx, "demo"); err != nil || got != config {
		t.Fatalf("Get = %#v, %v", got, err)
	}
	if exists, err := repository.Exists(ctx, "demo"); err != nil || !exists {
		t.Fatalf("Exists = %v, %v", exists, err)
	}
	if repository.GetPath("demo") != "scenarios/demo/signing.json" {
		t.Fatal("GetPath did not preserve scenario layout")
	}
	windows := &types.WindowsSigningConfig{}
	macOS := &types.MacOSSigningConfig{}
	linux := &types.LinuxSigningConfig{}
	for _, entry := range []struct {
		platform string
		config   interface{}
	}{
		{types.PlatformWindows, windows},
		{types.PlatformMacOS, macOS},
		{types.PlatformLinux, linux},
	} {
		if err := repository.SaveForPlatform(ctx, "demo", entry.platform, entry.config); err != nil {
			t.Fatalf("SaveForPlatform(%s): %v", entry.platform, err)
		}
		got, err := repository.GetForPlatform(ctx, "demo", entry.platform)
		if err != nil || got != entry.config {
			t.Fatalf("GetForPlatform(%s) = %#v, %v", entry.platform, got, err)
		}
		if err := repository.DeleteForPlatform(ctx, "demo", entry.platform); err != nil {
			t.Fatalf("DeleteForPlatform(%s): %v", entry.platform, err)
		}
		got, err = repository.GetForPlatform(ctx, "demo", entry.platform)
		if err != nil || !isNilPlatformConfig(got) {
			t.Fatalf("deleted GetForPlatform(%s) = %#v, %v", entry.platform, got, err)
		}
	}
	if got, err := repository.GetForPlatform(ctx, "missing", "unknown"); err != nil || got != nil {
		t.Fatalf("unknown platform = %#v, %v", got, err)
	}
	getErr := errors.New("get")
	repository.GetError = getErr
	if _, err := repository.Get(ctx, "demo"); !errors.Is(err, getErr) {
		t.Fatalf("GetError = %v", err)
	}
	if _, err := repository.GetForPlatform(ctx, "demo", types.PlatformLinux); !errors.Is(err, getErr) {
		t.Fatalf("GetForPlatform GetError = %v", err)
	}
	repository.GetError = nil
	saveErr := errors.New("save")
	repository.SaveError = saveErr
	if err := repository.Save(ctx, "demo", config); !errors.Is(err, saveErr) {
		t.Fatalf("SaveError = %v", err)
	}
	if err := repository.SaveForPlatform(ctx, "demo", types.PlatformLinux, linux); !errors.Is(err, saveErr) {
		t.Fatalf("SaveForPlatform error = %v", err)
	}
	repository.SaveError = nil
	deleteErr := errors.New("delete")
	repository.DeleteError = deleteErr
	if err := repository.Delete(ctx, "demo"); !errors.Is(err, deleteErr) {
		t.Fatalf("DeleteError = %v", err)
	}
	if err := repository.DeleteForPlatform(ctx, "demo", types.PlatformLinux); !errors.Is(err, deleteErr) {
		t.Fatalf("DeleteForPlatform error = %v", err)
	}

	locator := NewMockScenarioLocator().AddScenario("demo", "/scenarios/demo")
	if path, err := locator.GetScenarioPath("demo"); err != nil || path != "/scenarios/demo" {
		t.Fatalf("GetScenarioPath = %q, %v", path, err)
	}
	if _, err := locator.GetScenarioPath("missing"); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("missing scenario error = %v", err)
	}
	locator.GetPathError = getErr
	if _, err := locator.GetScenarioPath("demo"); !errors.Is(err, getErr) {
		t.Fatalf("GetPathError = %v", err)
	}
	if names, err := locator.ListScenarios(); err != nil || len(names) != 1 || names[0] != "demo" {
		t.Fatalf("ListScenarios = %#v, %v", names, err)
	}
}

func isNilPlatformConfig(value interface{}) bool {
	switch config := value.(type) {
	case *types.WindowsSigningConfig:
		return config == nil
	case *types.MacOSSigningConfig:
		return config == nil
	case *types.LinuxSigningConfig:
		return config == nil
	default:
		return value == nil
	}
}

func TestMockCommandRunnerAndEnvironmentPreserveConfiguredContracts(t *testing.T) {
	runner := NewMockCommandRunner().
		AddCommand("codesign --verify app", []byte("verified"), []byte("warning"), nil).
		AddCommand("notarytool", []byte("accepted"), nil, nil).
		AddLookPath("codesign", "/usr/bin/codesign")
	stdout, stderr, err := runner.Run(context.Background(), "codesign", "--verify", "app")
	if err != nil || string(stdout) != "verified" || string(stderr) != "warning" || len(runner.CallLog) != 1 {
		t.Fatalf("exact Run = %q %q %v %#v", stdout, stderr, err, runner.CallLog)
	}
	stdout, _, err = runner.Run(context.Background(), "notarytool", "submit")
	if err != nil || string(stdout) != "accepted" {
		t.Fatalf("name fallback Run = %q %v", stdout, err)
	}
	if stdout, stderr, err = runner.Run(context.Background(), "missing"); err != nil || stdout != nil || stderr != nil {
		t.Fatalf("missing Run = %q %q %v", stdout, stderr, err)
	}
	if path, err := runner.LookPath("codesign"); err != nil || path != "/usr/bin/codesign" {
		t.Fatalf("LookPath = %q %v", path, err)
	}
	missing := errors.New("not installed")
	runner.AddLookPathError("security", missing)
	if _, err := runner.LookPath("security"); !errors.Is(err, missing) {
		t.Fatalf("configured lookup error = %v", err)
	}
	if _, err := runner.LookPath("missing"); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("missing lookup error = %v", err)
	}
	runErr := errors.New("run override")
	runner.RunFunc = func(context.Context, string, ...string) ([]byte, []byte, error) {
		return []byte("override"), nil, runErr
	}
	if stdout, _, err := runner.Run(context.Background(), "anything"); !errors.Is(err, runErr) || string(stdout) != "override" {
		t.Fatalf("RunFunc = %q %v", stdout, err)
	}
	runner.LookPathFunc = func(string) (string, error) { return "/custom/tool", nil }
	if path, err := runner.LookPath("anything"); err != nil || path != "/custom/tool" {
		t.Fatalf("LookPathFunc = %q %v", path, err)
	}

	now := time.Date(2026, 7, 27, 12, 0, 0, 0, time.UTC)
	if got := NewMockTimeProvider(now).Now(); !got.Equal(now) {
		t.Fatalf("Now = %v", got)
	}
	env := NewMockEnvironmentReader().SetEnv("SIGNING_PASSWORD", "secret")
	if got := env.GetEnv("SIGNING_PASSWORD"); got != "secret" {
		t.Fatalf("GetEnv = %q", got)
	}
	if got, ok := env.LookupEnv("SIGNING_PASSWORD"); !ok || got != "secret" {
		t.Fatalf("LookupEnv = %q, %v", got, ok)
	}
	if _, ok := env.LookupEnv("MISSING"); ok {
		t.Fatal("missing environment value reported present")
	}
}
