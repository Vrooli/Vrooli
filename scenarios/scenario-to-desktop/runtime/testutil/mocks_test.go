package testutil

import (
	"context"
	"errors"
	"io"
	"os"
	"testing"
	"time"

	"github.com/vrooli/vrooli/scenarios/scenario-to-desktop/runtime/manifest"
)

func TestMockClockAndCommandRunnerProvideDeterministicRuntimeControls(t *testing.T) {
	start := time.Date(2026, time.July, 27, 0, 0, 0, 0, time.UTC)
	clock := NewMockClock(start)
	clock.Sleep(time.Minute)
	if got := clock.Now(); !got.Equal(start.Add(time.Minute)) {
		t.Fatalf("Now() = %v", got)
	}
	if got := <-clock.After(time.Second); !got.Equal(start.Add(time.Minute + time.Second)) {
		t.Fatalf("After() = %v", got)
	}

	runner := NewMockCommandRunner()
	runner.SetOutput([]byte("ready"))
	if err := runner.Run(context.Background(), "desktop-runtime", []string{"status"}); err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if output, err := runner.Output(context.Background(), "desktop-runtime", "status"); err != nil || string(output) != "ready" {
		t.Fatalf("Output() = %q, %v", output, err)
	}
	if len(runner.Commands()) != 2 {
		t.Fatalf("Commands() = %v, want two entries", runner.Commands())
	}
}

func TestMockNetworkAndFileSystemModelSuccessAndFailure(t *testing.T) {
	dialer := NewMockDialer()
	dialer.SetPort(4312, true)
	connection, err := dialer.Dial("tcp", "127.0.0.1:4312")
	if err != nil {
		t.Fatalf("Dial() error: %v", err)
	}
	if bytes, writeErr := connection.Write([]byte("ping")); writeErr != nil || bytes != 4 {
		t.Fatalf("Write() = (%d, %v)", bytes, writeErr)
	}
	if _, err := dialer.DialTimeout("tcp", "127.0.0.1:4313", time.Second); err == nil {
		t.Fatal("closed port unexpectedly accepted")
	}
	dialer.SetShouldFail(true)
	if _, err := dialer.Dial("tcp", "127.0.0.1:4312"); err == nil {
		t.Fatal("configured dial failure was not returned")
	}
	if _, err := dialer.Listen("tcp", "127.0.0.1:0"); err == nil {
		t.Fatal("unimplemented listen unexpectedly succeeded")
	}

	filesystem := NewMockFileSystem()
	if err := filesystem.MkdirAll("state", 0o755); err != nil {
		t.Fatalf("MkdirAll() error: %v", err)
	}
	if err := filesystem.WriteFile("state/config.json", []byte("{}"), 0o644); err != nil {
		t.Fatalf("WriteFile() error: %v", err)
	}
	file, err := filesystem.OpenFile("state/log.txt", os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		t.Fatalf("OpenFile() error: %v", err)
	}
	if _, err := file.Write([]byte("first")); err != nil {
		t.Fatalf("first write: %v", err)
	}
	if _, err := file.Write([]byte(" second")); err != nil {
		t.Fatalf("second write: %v", err)
	}
	if err := file.Sync(); err != nil {
		t.Fatalf("Sync() error: %v", err)
	}
	if err := file.Close(); err != nil {
		t.Fatalf("Close() error: %v", err)
	}
	data, err := filesystem.ReadFile("state/log.txt")
	if err != nil || string(data) != "first second" {
		t.Fatalf("ReadFile() = %q, %v", data, err)
	}
	info, err := filesystem.Stat("state")
	if err != nil || !info.IsDir() {
		t.Fatalf("Stat(state) = %#v, %v", info, err)
	}
	if err := filesystem.Remove("state/log.txt"); err != nil {
		t.Fatalf("Remove() error: %v", err)
	}
	if _, err := filesystem.ReadFile("state/log.txt"); err == nil {
		t.Fatal("removed file can still be read")
	}
}

func TestMockProcessPortHealthAndSecretsSupportLifecycleTesting(t *testing.T) {
	process := NewMockProcess(41)
	if process.Pid() != 41 {
		t.Fatalf("Pid() = %d", process.Pid())
	}
	if err := process.Signal(os.Interrupt); err != nil || !process.Signaled() {
		t.Fatalf("Signal() = %v, signaled=%v", err, process.Signaled())
	}
	process.Exit(errors.New("exit failure"))
	if err := process.Wait(); err == nil {
		t.Fatal("Wait() lost configured error")
	}
	if err := process.Kill(); err != nil || !process.Killed() {
		t.Fatalf("Kill() = %v, killed=%v", err, process.Killed())
	}

	runner := NewMockProcessRunner()
	runner.SetProcesses([]*MockProcess{NewMockProcess(42)})
	started, err := runner.Start(context.Background(), "desktop", []string{"--headless"}, []string{"A=B"}, "work", io.Discard, io.Discard)
	if err != nil || started.Pid() != 42 {
		t.Fatalf("Start() = (%v, %v)", started, err)
	}
	if got := runner.StartedCmds(); len(got) != 1 || got[0] != "desktop" {
		t.Fatalf("StartedCmds() = %#v", got)
	}
	if got := runner.StartedArgs(); len(got) != 1 || got[0][0] != "--headless" {
		t.Fatalf("StartedArgs() = %#v", got)
	}
	if got := runner.StartedEnvs(); len(got) != 1 || got[0][0] != "A=B" {
		t.Fatalf("StartedEnvs() = %#v", got)
	}
	if got := runner.StartedDirs(); len(got) != 1 || got[0] != "work" {
		t.Fatalf("StartedDirs() = %#v", got)
	}
	runner.SetShouldFail(true)
	if _, err := runner.Start(context.Background(), "desktop", nil, nil, "", io.Discard, io.Discard); err == nil {
		t.Fatal("configured start failure was not returned")
	}

	ports := NewMockPortAllocator()
	ports.SetPort("desktop-api", "api", 19925)
	if err := ports.Allocate(); err != nil {
		t.Fatalf("Allocate() error: %v", err)
	}
	if port, err := ports.Resolve("desktop-api", "api"); err != nil || port != 19925 {
		t.Fatalf("Resolve() = (%d, %v)", port, err)
	}
	if _, err := ports.Resolve("missing", "api"); err == nil {
		t.Fatal("missing port unexpectedly resolved")
	}
	copy := ports.Map()
	copy["desktop-api"]["api"] = 1
	if port, _ := ports.Resolve("desktop-api", "api"); port != 19925 {
		t.Fatal("Map() exposed mutable allocator state")
	}

	health := NewMockHealthChecker()
	if err := health.WaitForReadiness(context.Background(), "api"); err != nil {
		t.Fatalf("WaitForReadiness() error: %v", err)
	}
	health.SetCheckOnceResult(false)
	if health.CheckOnce(context.Background(), "api") {
		t.Fatal("configured health check result was ignored")
	}
	health.SetWaitReadinessErr(errors.New("not ready"))
	if err := health.WaitForReadiness(context.Background(), "api"); err == nil {
		t.Fatal("configured readiness error was not returned")
	}
	health.SetWaitDepsErr(errors.New("dependency unavailable"))
	if err := health.WaitForDependencies(context.Background(), &manifest.Service{ID: "api"}); err == nil {
		t.Fatal("configured dependency error was not returned")
	}

	store := NewMockSecretStore(&manifest.Manifest{Secrets: []manifest.Secret{{ID: "token"}}})
	store.SetSecrets(map[string]string{"token": "old"})
	if err := store.Persist(map[string]string{"token": "new"}); err != nil {
		t.Fatalf("Persist() error: %v", err)
	}
	loaded, err := store.Load()
	if err != nil || loaded["token"] != "new" {
		t.Fatalf("Load() = %#v, %v", loaded, err)
	}
	if store.FindSecret("token") == nil || store.FindSecret("missing") != nil {
		t.Fatal("FindSecret() did not honor manifest")
	}
	if merged := store.Merge(map[string]string{"extra": "value"}); merged["token"] != "new" || merged["extra"] != "value" {
		t.Fatalf("Merge() = %#v", merged)
	}
	store.SetMissingRequired([]string{"token"})
	if got := store.MissingRequiredFrom(nil); len(got) != 1 || got[0] != "token" {
		t.Fatalf("MissingRequiredFrom() = %#v", got)
	}
	store.SetGeneratedResult(map[string]string{"token": "generated"})
	if generated, err := store.GenerateMissing(nil); err != nil || generated["token"] != "generated" {
		t.Fatalf("GenerateMissing() = %#v, %v", generated, err)
	}
	store.SetValidateErr(errors.New("invalid"))
	if err := store.Validate(nil); err == nil {
		t.Fatal("configured validation error was not returned")
	}
	store.SetLoadErr(errors.New("load failed"))
	if _, err := store.Load(); err == nil {
		t.Fatal("configured load error was not returned")
	}
	store.SetPersistErr(errors.New("persist failed"))
	if err := store.Persist(nil); err == nil {
		t.Fatal("configured persist error was not returned")
	}
}

func TestMockGPUAndEnvironmentExposeConfiguredValues(t *testing.T) {
	gpu := NewMockGPUDetector(false)
	if gpu.Detect().Available {
		t.Fatal("new detector availability = true")
	}
	gpu.SetStatus(GPUStatus{Available: true, Method: "pci", Reason: "detected"})
	if got := gpu.Detect(); !got.Available || got.Method != "pci" {
		t.Fatalf("Detect() = %#v", got)
	}
	env := NewMockEnvReader()
	env.SetEnv("DESKTOP_MODE", "bundle")
	if got := env.Getenv("DESKTOP_MODE"); got != "bundle" {
		t.Fatalf("Getenv() = %q", got)
	}
	if value, ok := env.LookupEnv("DESKTOP_MODE"); !ok || value != "bundle" {
		t.Fatalf("LookupEnv() = (%q, %v)", value, ok)
	}
	if _, ok := env.LookupEnv("MISSING"); ok {
		t.Fatal("missing environment variable exists")
	}
	if values := env.Environ(); len(values) != 1 || values[0] != "DESKTOP_MODE=bundle" {
		t.Fatalf("Environ() = %#v", values)
	}
}

func TestMockSupportFailureBranchesAndCopiesAreObservable(t *testing.T) {
	clock := NewMockClock(time.Now())
	ticker := clock.NewTicker(time.Second)
	clock.Advance(time.Second)
	<-ticker.C()
	ticker.Stop()

	runner := NewMockCommandRunner()
	runner.SetShouldErr(true)
	if err := runner.Run(context.Background(), "desktop", nil); err == nil {
		t.Fatal("Run() did not return configured error")
	}
	if _, err := runner.Output(context.Background(), "desktop"); err == nil {
		t.Fatal("Output() did not return configured error")
	}
	if path, err := runner.LookPath("desktop"); err != nil || path != "desktop" {
		t.Fatalf("LookPath() = (%q, %v)", path, err)
	}

	store := NewMockSecretStore(nil)
	store.SetGenerateErr(errors.New("generation failed"))
	if _, err := store.GenerateMissing(nil); err == nil {
		t.Fatal("GenerateMissing() did not return configured error")
	}
	store.SetSecrets(map[string]string{"copy": "one"})
	copy := store.Get()
	copy["copy"] = "changed"
	if store.Get()["copy"] != "one" {
		t.Fatal("Get() exposed mutable secret state")
	}
	missing := []string{"required"}
	store.SetMissingRequired(missing)
	missing[0] = "mutated"
	if store.MissingRequired()[0] != "required" {
		t.Fatal("missing requirement configuration exposed mutable state")
	}
}
