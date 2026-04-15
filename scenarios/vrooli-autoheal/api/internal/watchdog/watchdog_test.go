// Package watchdog tests
// [REQ:WATCH-DETECT-001]
package watchdog

import (
	"errors"
	"io/fs"
	"os"
	"os/user"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"vrooli-autoheal/internal/platform"
)

type fakeCommandResult struct {
	output []byte
	err    error
}

type fakeProbe struct {
	goosValue        string
	commandOutputs   map[string]fakeCommandResult
	commandInputs    map[string]fakeCommandResult
	commandRuns      map[string]error
	dirEntries       map[string][]os.DirEntry
	files            map[string][]byte
	stats            map[string]error
	mkdirErrs        map[string]error
	writeErrs        map[string]error
	removeErrs       map[string]error
	writtenFiles     map[string][]byte
	removedFiles     map[string]bool
	tempPath         string
	tempWriteErr     error
	currentUserValue *user.User
	currentUserErr   error
	userHomeDirPath  string
	env              map[string]string
}

func newFakeProbe() *fakeProbe {
	return &fakeProbe{
		goosValue:      "linux",
		commandOutputs: map[string]fakeCommandResult{},
		commandInputs:  map[string]fakeCommandResult{},
		commandRuns:    map[string]error{},
		dirEntries:     map[string][]os.DirEntry{},
		files:          map[string][]byte{},
		stats:          map[string]error{},
		mkdirErrs:      map[string]error{},
		writeErrs:      map[string]error{},
		removeErrs:     map[string]error{},
		writtenFiles:   map[string][]byte{},
		removedFiles:   map[string]bool{},
		tempPath:       "/tmp/vrooli-autoheal-test.xml",
		env:            map[string]string{},
	}
}

func (f *fakeProbe) goos() string { return f.goosValue }

func (f *fakeProbe) commandOutput(name string, args ...string) ([]byte, error) {
	result, ok := f.commandOutputs[commandKey(name, args...)]
	if !ok {
		return nil, errors.New("command output not configured")
	}
	return result.output, result.err
}

func (f *fakeProbe) commandOutputInput(name, input string, args ...string) ([]byte, error) {
	result, ok := f.commandInputs[commandInputKey(name, input, args...)]
	if !ok {
		return nil, errors.New("command output input not configured")
	}
	return result.output, result.err
}

func (f *fakeProbe) commandRun(name string, args ...string) error {
	if err, ok := f.commandRuns[commandKey(name, args...)]; ok {
		return err
	}
	return errors.New("command run not configured")
}

func (f *fakeProbe) readDir(path string) ([]os.DirEntry, error) {
	entries, ok := f.dirEntries[path]
	if !ok {
		return nil, errors.New("dir not configured")
	}
	return entries, nil
}

func (f *fakeProbe) readFile(path string) ([]byte, error) {
	data, ok := f.files[path]
	if !ok {
		return nil, errors.New("file not configured")
	}
	return data, nil
}

func (f *fakeProbe) stat(path string) error {
	if err, ok := f.stats[path]; ok {
		return err
	}
	return os.ErrNotExist
}

func (f *fakeProbe) mkdirAll(path string, perm os.FileMode) error {
	if err, ok := f.mkdirErrs[path]; ok {
		return err
	}
	return nil
}

func (f *fakeProbe) writeFile(path string, data []byte, perm os.FileMode) error {
	if err, ok := f.writeErrs[path]; ok {
		return err
	}
	f.writtenFiles[path] = append([]byte(nil), data...)
	f.stats[path] = nil
	return nil
}

func (f *fakeProbe) remove(path string) error {
	if err, ok := f.removeErrs[path]; ok {
		return err
	}
	f.removedFiles[path] = true
	delete(f.stats, path)
	return nil
}

func (f *fakeProbe) writeTempFile(pattern string, data []byte) (string, error) {
	if f.tempWriteErr != nil {
		return "", f.tempWriteErr
	}
	f.writtenFiles[f.tempPath] = append([]byte(nil), data...)
	return f.tempPath, nil
}

func (f *fakeProbe) currentUser() (*user.User, error) {
	if f.currentUserErr != nil {
		return nil, f.currentUserErr
	}
	if f.currentUserValue == nil {
		return nil, errors.New("user not configured")
	}
	return f.currentUserValue, nil
}

func (f *fakeProbe) userHomeDir() (string, error) {
	if f.userHomeDirPath == "" {
		return "", errors.New("home dir not configured")
	}
	return f.userHomeDirPath, nil
}

func (f *fakeProbe) getenv(key string) string {
	return f.env[key]
}

type fakeDirEntry struct {
	name  string
	isDir bool
}

func (e fakeDirEntry) Name() string               { return e.name }
func (e fakeDirEntry) IsDir() bool                { return e.isDir }
func (e fakeDirEntry) Type() fs.FileMode          { return 0 }
func (e fakeDirEntry) Info() (fs.FileInfo, error) { return nil, nil }

func commandKey(name string, args ...string) string {
	return name + "|" + strings.Join(args, "|")
}

func commandInputKey(name, input string, args ...string) string {
	return name + "|" + input + "|" + strings.Join(args, "|")
}

func detectorWithProbe(plat *platform.Capabilities, probe detectorProbe) *Detector {
	d := NewDetector(plat)
	d.probe = probe
	return d
}

func newWatchdogContractFixtureRepo(t *testing.T) string {
	t.Helper()

	root := t.TempDir()
	repoRoot := watchdogRepoRoot(t)
	contractData, err := os.ReadFile(filepath.Join(repoRoot, ".vrooli", "repo-contract.json"))
	if err != nil {
		t.Fatalf("read repo contract: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(root, ".vrooli"), 0o755); err != nil {
		t.Fatalf("mkdir .vrooli: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, ".vrooli", "repo-contract.json"), contractData, 0o644); err != nil {
		t.Fatalf("write repo contract: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module example.com/vrooli-autoheal-watchdog-test\n\ngo 1.24.0\n"), 0o644); err != nil {
		t.Fatalf("write go.mod: %v", err)
	}
	for _, dir := range []string{"templates", "scenarios", "resources", "packages", "cmd", "internal"} {
		if err := os.MkdirAll(filepath.Join(root, dir), 0o755); err != nil {
			t.Fatalf("mkdir %s: %v", dir, err)
		}
	}
	return root
}

func watchdogRepoRoot(t *testing.T) string {
	t.Helper()
	return filepath.Clean(filepath.Join(filepath.Dir(testFilePath(t)), "..", "..", "..", "..", ".."))
}

func testFilePath(t *testing.T) string {
	t.Helper()
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	return filename
}

func TestNewDetector(t *testing.T) {
	plat := &platform.Capabilities{
		Platform:        "linux",
		SupportsSystemd: true,
	}

	d := NewDetector(plat)
	if d == nil {
		t.Fatal("expected non-nil detector")
	}
	if d.platform != plat {
		t.Error("expected platform to be set")
	}
	if d.probe == nil {
		t.Error("expected probe to be initialized")
	}
}

func TestDetect_Linux(t *testing.T) {
	plat := &platform.Capabilities{
		Platform:        "linux",
		SupportsSystemd: true,
	}

	probe := newFakeProbe()
	probe.currentUserValue = &user.User{Username: "tester"}
	probe.env["HOME"] = "/home/tester"
	probe.stats["/etc/systemd/system/vrooli-autoheal.service"] = nil
	probe.commandRuns[commandKey("pgrep", "-f", "vrooli-autoheal.*loop")] = nil
	probe.commandOutputs[commandKey("systemctl", "is-enabled", "vrooli-autoheal")] = fakeCommandResult{
		output: []byte("enabled\n"),
	}
	probe.commandOutputs[commandKey("systemctl", "is-active", "vrooli-autoheal")] = fakeCommandResult{
		output: []byte("active\n"),
	}

	d := detectorWithProbe(plat, probe)
	status := d.Detect()

	if status == nil {
		t.Fatal("expected non-nil status")
	}

	if !status.LoopRunning {
		t.Error("expected LoopRunning to be true")
	}

	// Should be able to install on Linux with systemd
	if !status.CanInstall {
		t.Error("expected CanInstall to be true for Linux with systemd")
	}

	// WatchdogType should be systemd
	if status.WatchdogType != WatchdogTypeSystemd {
		t.Errorf("expected WatchdogType=%s, got=%s", WatchdogTypeSystemd, status.WatchdogType)
	}
	if !status.WatchdogInstalled || !status.WatchdogEnabled || !status.WatchdogRunning {
		t.Error("expected installed, enabled, and running system service")
	}
}

func TestDetect_LinuxLoopFallbackToProc(t *testing.T) {
	plat := &platform.Capabilities{
		Platform:        "linux",
		SupportsSystemd: false,
	}

	probe := newFakeProbe()
	probe.commandRuns[commandKey("pgrep", "-f", "vrooli-autoheal.*loop")] = errors.New("no match")
	probe.dirEntries["/proc"] = []os.DirEntry{
		fakeDirEntry{name: "not-pid", isDir: true},
		fakeDirEntry{name: "1234", isDir: true},
	}
	probe.files["/proc/1234/cmdline"] = []byte("/usr/bin/vrooli-autoheal\x00loop")

	d := detectorWithProbe(plat, probe)
	if !d.isLoopRunning() {
		t.Error("expected proc fallback to detect running loop")
	}
}

func TestDetect_LinuxUserServiceWithLingering(t *testing.T) {
	plat := &platform.Capabilities{
		Platform:        "linux",
		SupportsSystemd: true,
	}

	probe := newFakeProbe()
	probe.currentUserValue = &user.User{Username: "tester"}
	probe.env["HOME"] = "/home/tester"
	probe.stats["/home/tester/.config/systemd/user/vrooli-autoheal.service"] = nil
	probe.stats["/var/lib/systemd/linger/tester"] = os.ErrNotExist
	probe.commandRuns[commandKey("pgrep", "-f", "vrooli-autoheal.*loop")] = errors.New("no match")
	probe.commandOutputs[commandKey("systemctl", "--user", "is-enabled", "vrooli-autoheal")] = fakeCommandResult{
		output: []byte("enabled\n"),
	}
	probe.commandOutputs[commandKey("systemctl", "--user", "is-active", "vrooli-autoheal")] = fakeCommandResult{
		output: []byte("active\n"),
	}
	probe.commandOutputs[commandKey("loginctl", "show-user", "tester", "--property=Linger")] = fakeCommandResult{
		output: []byte("Linger=yes\n"),
	}

	d := detectorWithProbe(plat, probe)
	status := d.Detect()
	if !status.IsUserService {
		t.Error("expected user service detection")
	}
	if !status.LingeringEnabled {
		t.Error("expected lingering to be enabled from loginctl")
	}
	if status.Username != "tester" {
		t.Errorf("expected username tester, got %q", status.Username)
	}
}

func TestDetect_LinuxNoSystemd(t *testing.T) {
	plat := &platform.Capabilities{
		Platform:        "linux",
		SupportsSystemd: false,
	}

	d := NewDetector(plat)
	status := d.Detect()

	if status.CanInstall {
		t.Error("expected CanInstall to be false when systemd not available")
	}

	if status.LastError == "" {
		t.Error("expected LastError to be set when systemd not available")
	}
}

func TestDetect_MacOS(t *testing.T) {
	plat := &platform.Capabilities{
		Platform:        "macos",
		SupportsLaunchd: true,
	}

	d := NewDetector(plat)
	status := d.Detect()

	if status.WatchdogType != WatchdogTypeLaunchd {
		t.Errorf("expected WatchdogType=%s, got=%s", WatchdogTypeLaunchd, status.WatchdogType)
	}

	if !status.CanInstall {
		t.Error("expected CanInstall to be true for macOS with launchd")
	}
}

func TestDetect_Windows(t *testing.T) {
	plat := &platform.Capabilities{
		Platform:           "windows",
		SupportsWindowsSvc: true,
	}

	probe := newFakeProbe()
	probe.goosValue = "windows"
	probe.commandOutputs[commandKey("tasklist", "/FI", "IMAGENAME eq vrooli-autoheal*")] = fakeCommandResult{
		output: []byte("vrooli-autoheal.exe"),
	}
	probe.commandOutputs[commandKey("schtasks", "/Query", "/TN", "VrooliAutoheal")] = fakeCommandResult{
		output: []byte("Status: Running"),
	}

	d := detectorWithProbe(plat, probe)
	status := d.Detect()

	if status.WatchdogType != WatchdogTypeWindows {
		t.Errorf("expected WatchdogType=%s, got=%s", WatchdogTypeWindows, status.WatchdogType)
	}
	if !status.WatchdogInstalled || !status.WatchdogEnabled || !status.WatchdogRunning {
		t.Error("expected scheduled task to be installed, enabled, and running")
	}
	if status.LastError != "" {
		t.Errorf("expected no LastError, got %q", status.LastError)
	}
}

func TestDetect_UnsupportedPlatform(t *testing.T) {
	plat := &platform.Capabilities{
		Platform: "other",
	}

	d := NewDetector(plat)
	status := d.Detect()

	if status.CanInstall {
		t.Error("expected CanInstall to be false for unsupported platform")
	}

	if status.LastError == "" {
		t.Error("expected LastError to be set for unsupported platform")
	}
}

func TestGetCached(t *testing.T) {
	plat := &platform.Capabilities{
		Platform:        "linux",
		SupportsSystemd: true,
	}

	d := NewDetector(plat)

	// First call should trigger detection
	status1 := d.GetCached()
	if status1 == nil {
		t.Fatal("expected non-nil status from GetCached")
	}

	// Second call should return cached value
	status2 := d.GetCached()
	if status2 == nil {
		t.Fatal("expected non-nil status from second GetCached")
	}
}

func TestCalculateProtectionLevel(t *testing.T) {
	plat := &platform.Capabilities{Platform: "linux", SupportsSystemd: true}
	d := NewDetector(plat)

	tests := []struct {
		name     string
		status   Status
		expected ProtectionLevel
	}{
		{
			name: "full protection",
			status: Status{
				LoopRunning:       true,
				WatchdogInstalled: true,
				WatchdogEnabled:   true,
				WatchdogRunning:   true,
			},
			expected: ProtectionFull,
		},
		{
			name: "partial protection - loop only",
			status: Status{
				LoopRunning:       true,
				WatchdogInstalled: false,
			},
			expected: ProtectionPartial,
		},
		{
			name: "partial protection - installed but not running",
			status: Status{
				LoopRunning:       true,
				WatchdogInstalled: true,
				WatchdogEnabled:   true,
				WatchdogRunning:   false,
			},
			expected: ProtectionPartial,
		},
		{
			name: "no protection",
			status: Status{
				LoopRunning:       false,
				WatchdogInstalled: false,
			},
			expected: ProtectionNone,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := d.calculateProtectionLevel(&tt.status)
			if result != tt.expected {
				t.Errorf("expected %s, got %s", tt.expected, result)
			}
		})
	}
}

func TestGetServiceTemplate_Linux(t *testing.T) {
	plat := &platform.Capabilities{Platform: "linux", SupportsSystemd: true}
	d := NewDetector(plat)

	template, err := d.GetServiceTemplate()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if template == "" {
		t.Error("expected non-empty template")
	}

	// Check for key systemd directives
	if !strings.Contains(template, "[Unit]") {
		t.Error("expected [Unit] section in systemd template")
	}
	if !strings.Contains(template, "[Service]") {
		t.Error("expected [Service] section in systemd template")
	}
	if !strings.Contains(template, "Restart=always") {
		t.Error("expected Restart=always in systemd template")
	}
}

func TestGetSystemdTemplateForService_UserServiceOmitsUserDirective(t *testing.T) {
	plat := &platform.Capabilities{Platform: "linux", SupportsSystemd: true}
	probe := newFakeProbe()
	probe.userHomeDirPath = "/home/tester"
	probe.env["VROOLI_ROOT"] = "/workspace/Vrooli"
	d := detectorWithProbe(plat, probe)

	template := d.getSystemdTemplateForService(false)
	if strings.Contains(template, "\nUser=") {
		t.Fatalf("user service template should not include User= directive:\n%s", template)
	}
	if !strings.Contains(template, "WantedBy=default.target") {
		t.Fatalf("expected user service WantedBy=default.target:\n%s", template)
	}
}

func TestGetSystemdTemplateForService_SystemServiceUsesRootAndMultiUserTarget(t *testing.T) {
	plat := &platform.Capabilities{Platform: "linux", SupportsSystemd: true}
	probe := newFakeProbe()
	probe.userHomeDirPath = "/home/tester"
	probe.env["VROOLI_ROOT"] = "/workspace/Vrooli"
	d := detectorWithProbe(plat, probe)

	template := d.getSystemdTemplateForService(true)
	if !strings.Contains(template, "\nUser=root\n") {
		t.Fatalf("system service template should include User=root:\n%s", template)
	}
	if !strings.Contains(template, "WantedBy=multi-user.target") {
		t.Fatalf("expected system service WantedBy=multi-user.target:\n%s", template)
	}
}

func TestGetServiceTemplate_MacOS(t *testing.T) {
	plat := &platform.Capabilities{Platform: "macos", SupportsLaunchd: true}
	d := NewDetector(plat)

	template, err := d.GetServiceTemplate()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(template, "com.vrooli.autoheal") {
		t.Error("expected label in launchd template")
	}
	if !strings.Contains(template, "<key>KeepAlive</key>") {
		t.Error("expected KeepAlive in launchd template")
	}
}

func TestGetServiceTemplate_Windows(t *testing.T) {
	plat := &platform.Capabilities{Platform: "windows", SupportsWindowsSvc: true}
	d := NewDetector(plat)

	template, err := d.GetServiceTemplate()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(template, "BootTrigger") {
		t.Error("expected BootTrigger in Windows task template")
	}
	if !strings.Contains(template, "RestartOnFailure") {
		t.Error("expected RestartOnFailure in Windows task template")
	}
}

func TestGetServiceTemplate_Unsupported(t *testing.T) {
	plat := &platform.Capabilities{Platform: "other"}
	d := NewDetector(plat)

	_, err := d.GetServiceTemplate()
	if err == nil {
		t.Error("expected error for unsupported platform")
	}
}
