package peerrecord

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/vrooli/cli-core/cliutil"
	"github.com/vrooli/vrooli/internal/fsx"
	"github.com/vrooli/vrooli/internal/process"
)

func TestWriteReadRoundTripUsesTypedJSONContract(t *testing.T) {
	home := t.TempDir()
	want := Record{
		Scenario:      "demo",
		Instance:      "instance-1",
		Tier:          1,
		OwnerPID:      os.Getpid(),
		StartedAt:     time.Unix(123, 456).UTC(),
		Ports:         map[string]int{"api": 17573},
		AuthTokenPath: "/tmp/demo-token",
	}
	if err := Write(home, want); err != nil {
		t.Fatal(err)
	}

	got, err := Read(home, want.Scenario+"@"+want.Instance)
	if err != nil {
		t.Fatal(err)
	}
	if got.SchemaVersion != SchemaVersion || got.Scenario != want.Scenario || got.Instance != want.Instance || got.OwnerPID != want.OwnerPID || !got.StartedAt.Equal(want.StartedAt) {
		t.Fatalf("round-trip record = %#v, want %#v", got, want)
	}
	if got.Incarnation == "" {
		t.Fatal("round-trip record has no host process incarnation")
	}
	if got.Ports["api"] != want.Ports["api"] || got.AuthTokenPath != want.AuthTokenPath {
		t.Fatalf("round-trip payload = %#v, want %#v", got, want)
	}
}

func TestReadRejectsLivePIDWithStaleProcessIncarnation(t *testing.T) {
	home := t.TempDir()
	record := Record{
		Scenario:  "stale-incarnation",
		OwnerPID:  os.Getpid(),
		StartedAt: time.Unix(1, 0).UTC(), // A plausible but stale supervisor timestamp.
		Ports:     map[string]int{"api": 17573},
	}
	if err := Write(home, record); err != nil {
		t.Fatal(err)
	}

	// The PID is genuinely live, but simulate a record left by its prior
	// incarnation. Read must compare the host identity, independently of the
	// supervisor-supplied StartedAt value.
	path := Path(home, record.Scenario)
	var stored Record
	if err := fsx.ReadJSON(path, &stored); err != nil {
		t.Fatal(err)
	}
	stored.Incarnation = "old-boot:old-start-ticks"
	data, err := fsx.MarshalJSON(stored)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatal(err)
	}

	if _, err := Read(home, record.Scenario); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("Read stale live-PID record error = %v, want not-exist", err)
	}
}

func TestPeerRecordRejectsForeignLiveOwnerAndCleansStaleOwner(t *testing.T) {
	cmd := exec.Command(os.Args[0], "-test.run=^TestPeerRecordOwnerHelper$")
	cmd.Env = append(os.Environ(), "VROOLI_PEERRECORD_OWNER_HELPER=1")
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = cmd.Process.Kill(); _ = cmd.Wait() })
	home := t.TempDir()
	if err := Write(home, Record{Scenario: "owned", OwnerPID: cmd.Process.Pid, Ports: map[string]int{"api": 12345}}); err != nil {
		t.Fatal(err)
	}
	if err := Write(home, Record{Scenario: "owned", OwnerPID: os.Getpid(), Ports: map[string]int{"api": 12346}}); err == nil {
		t.Fatal("Write replaced another live owner")
	}
	if err := Remove(home, "owned", os.Getpid()); err == nil {
		t.Fatal("Remove deleted another live owner's record")
	}
	if err := cmd.Process.Kill(); err != nil {
		t.Fatal(err)
	}
	_ = cmd.Wait()
	if err := Remove(home, "owned", os.Getpid()); err != nil {
		t.Fatalf("Remove stale record: %v", err)
	}
}

func TestPeerRecordOwnerHelper(t *testing.T) {
	if os.Getenv("VROOLI_PEERRECORD_OWNER_HELPER") != "1" {
		return
	}
	time.Sleep(time.Hour)
}

func TestPeerRecordConcurrentPublishersCannotReplaceLiveOwner(t *testing.T) {
	home := t.TempDir()
	syncDir := t.TempDir()
	commands := []*exec.Cmd{
		peerRecordRaceHelper(t, home, syncDir, "write", 0, "concurrent-publisher"),
		peerRecordRaceHelper(t, home, syncDir, "write", 0, "concurrent-publisher"),
	}
	startPeerRecordRace(t, syncDir, commands)
	results := finishPeerRecordRace(t, syncDir, commands)
	successes := 0
	var winningPID int
	for i, result := range results {
		if result == "ok" {
			successes++
			winningPID = commands[i].Process.Pid
		} else if !strings.Contains(result, "owned by live process") {
			t.Fatalf("publisher %d failed unexpectedly: %s", i, result)
		}
	}
	if successes != 1 {
		t.Fatalf("successful publishers = %d, want exactly one; results=%v", successes, results)
	}
	got, err := readRecord(home, "concurrent-publisher")
	if err != nil {
		t.Fatalf("read winning record: %v", err)
	}
	if got.OwnerPID != winningPID {
		t.Fatalf("record owner PID = %d, want successful publisher %d", got.OwnerPID, winningPID)
	}
}

func TestPeerRecordStaleRemovalCannotDeleteConcurrentPublisher(t *testing.T) {
	home := t.TempDir()
	// Seed an expired but structurally valid record directly. Write correctly
	// refuses to publish an incarnation for a PID that does not exist.
	stale := Record{SchemaVersion: SchemaVersion, Scenario: "remove-publish-race", OwnerPID: 2147483647, Incarnation: "linux:stale:1", StartedAt: time.Now(), Ports: map[string]int{}}
	data, err := fsx.MarshalJSON(stale)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Dir(Path(home, stale.Scenario)), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(Path(home, stale.Scenario), data, 0o600); err != nil {
		t.Fatal(err)
	}
	syncDir := t.TempDir()
	commands := []*exec.Cmd{
		peerRecordRaceHelper(t, home, syncDir, "remove", 2147483647, "remove-publish-race"),
		peerRecordRaceHelper(t, home, syncDir, "write", 0, "remove-publish-race"),
	}
	startPeerRecordRace(t, syncDir, commands)
	results := finishPeerRecordRace(t, syncDir, commands)
	if results[1] != "ok" {
		t.Fatalf("publisher result = %q, want ok; all results=%v", results[1], results)
	}
	got, err := readRecord(home, "remove-publish-race")
	if err != nil {
		t.Fatalf("stale cleanup removed or lost fresh owner record: %v (results=%v)", err, results)
	}
	if got.OwnerPID != commands[1].Process.Pid {
		t.Fatalf("record owner PID = %d, want publisher %d", got.OwnerPID, commands[1].Process.Pid)
	}
}

func peerRecordRaceHelper(t *testing.T, home, syncDir, mode string, stalePID int, scenario string) *exec.Cmd {
	t.Helper()
	cmd := exec.Command(os.Args[0], "-test.run=^TestPeerRecordConcurrentMutationHelper$")
	env := append(os.Environ(),
		"VROOLI_PEERRECORD_RACE_HELPER=1",
		"VROOLI_PEERRECORD_RACE_MODE="+mode,
		"VROOLI_PEERRECORD_RACE_HOME="+home,
		"VROOLI_PEERRECORD_RACE_DIR="+syncDir,
		"VROOLI_PEERRECORD_RACE_SCENARIO="+scenario,
		fmt.Sprintf("VROOLI_PEERRECORD_RACE_STALE_PID=%d", stalePID),
	)
	cmd.Env = env
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if cmd.Process != nil {
			_ = cmd.Process.Kill()
			_ = cmd.Wait()
		}
	})
	return cmd
}

func startPeerRecordRace(t *testing.T, syncDir string, commands []*exec.Cmd) {
	t.Helper()
	for _, cmd := range commands {
		waitForPeerRecordRaceFile(t, filepath.Join(syncDir, fmt.Sprintf("ready-%d", cmd.Process.Pid)))
	}
	if err := os.WriteFile(filepath.Join(syncDir, "start"), []byte("go"), 0o600); err != nil {
		t.Fatal(err)
	}
}

func finishPeerRecordRace(t *testing.T, syncDir string, commands []*exec.Cmd) []string {
	t.Helper()
	results := make([]string, len(commands))
	for i, cmd := range commands {
		resultPath := filepath.Join(syncDir, fmt.Sprintf("result-%d", cmd.Process.Pid))
		waitForPeerRecordRaceFile(t, resultPath)
		contents, err := os.ReadFile(resultPath)
		if err != nil {
			t.Fatal(err)
		}
		results[i] = string(contents)
	}
	if err := os.WriteFile(filepath.Join(syncDir, "finish"), []byte("done"), 0o600); err != nil {
		t.Fatal(err)
	}
	for _, cmd := range commands {
		if err := cmd.Wait(); err != nil {
			t.Fatalf("race helper %d: %v", cmd.Process.Pid, err)
		}
	}
	return results
}

func waitForPeerRecordRaceFile(t *testing.T, path string) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if _, err := os.Stat(path); err == nil {
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatalf("timed out waiting for %s", path)
}

func TestPeerRecordConcurrentMutationHelper(t *testing.T) {
	if os.Getenv("VROOLI_PEERRECORD_RACE_HELPER") != "1" {
		return
	}
	home := os.Getenv("VROOLI_PEERRECORD_RACE_HOME")
	syncDir := os.Getenv("VROOLI_PEERRECORD_RACE_DIR")
	pid := os.Getpid()
	if err := os.WriteFile(filepath.Join(syncDir, fmt.Sprintf("ready-%d", pid)), []byte("ready"), 0o600); err != nil {
		t.Fatal(err)
	}
	waitForPeerRecordRaceFile(t, filepath.Join(syncDir, "start"))
	var err error
	switch os.Getenv("VROOLI_PEERRECORD_RACE_MODE") {
	case "write":
		err = Write(home, Record{Scenario: os.Getenv("VROOLI_PEERRECORD_RACE_SCENARIO"), OwnerPID: pid, Ports: map[string]int{"api": 18080}})
	case "remove":
		stalePID := 0
		if _, scanErr := fmt.Sscanf(os.Getenv("VROOLI_PEERRECORD_RACE_STALE_PID"), "%d", &stalePID); scanErr != nil {
			t.Fatal(scanErr)
		}
		err = Remove(home, os.Getenv("VROOLI_PEERRECORD_RACE_SCENARIO"), stalePID)
	default:
		t.Fatalf("unsupported helper mode %q", os.Getenv("VROOLI_PEERRECORD_RACE_MODE"))
	}
	result := "ok"
	if err != nil {
		result = err.Error()
	}
	if err := os.WriteFile(filepath.Join(syncDir, fmt.Sprintf("result-%d", pid)), []byte(result), 0o600); err != nil {
		t.Fatal(err)
	}
	waitForPeerRecordRaceFile(t, filepath.Join(syncDir, "finish"))
}

func TestReadRejectsTrailingJSONValue(t *testing.T) {
	home := t.TempDir()
	if err := Write(home, Record{Scenario: "demo", OwnerPID: os.Getpid()}); err != nil {
		t.Fatal(err)
	}
	path := Path(home, "demo")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	data = append(data, []byte("{}")...)
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := Read(home, "demo"); err == nil {
		t.Fatal("expected trailing JSON to be rejected")
	}
}

func TestVariantsHaveSeparatePeerRecordsAndRemovalIsScoped(t *testing.T) {
	home := t.TempDir()
	base := Record{Scenario: "browser-automation-studio", OwnerPID: os.Getpid(), Ports: map[string]int{"api": 17116}}
	if err := Write(home, base); err != nil {
		t.Fatal(err)
	}
	shadow := base
	shadow.Instance = "shadow"
	shadow.Ports = map[string]int{"api": 15372}
	if err := Write(home, shadow); err != nil {
		t.Fatal(err)
	}

	for _, tc := range []struct {
		target string
		port   int
	}{{"browser-automation-studio", 17116}, {"browser-automation-studio@live", 17116}, {"browser-automation-studio@shadow", 15372}} {
		got, err := Read(home, tc.target)
		if err != nil {
			t.Fatalf("Read(%q): %v", tc.target, err)
		}
		if got.Ports["api"] != tc.port {
			t.Errorf("Read(%q) api port = %d, want %d", tc.target, got.Ports["api"], tc.port)
		}
	}

	if err := Remove(home, "browser-automation-studio@shadow", os.Getpid()); err != nil {
		t.Fatal(err)
	}
	if _, err := Read(home, "browser-automation-studio@shadow"); !os.IsNotExist(err) {
		t.Fatalf("shadow Read after remove error = %v, want not-exist", err)
	}
	got, err := Read(home, "browser-automation-studio")
	if err != nil {
		t.Fatalf("live record missing after shadow removal: %v", err)
	}
	if got.Ports["api"] != 17116 {
		t.Fatalf("live api port after shadow removal = %d, want 17116", got.Ports["api"])
	}
}

func TestReadRejectsShadowRecordMisfiledAtLegacyBarePath(t *testing.T) {
	home := t.TempDir()
	if err := os.MkdirAll(filepath.Dir(Path(home, "browser-automation-studio")), 0o700); err != nil {
		t.Fatal(err)
	}
	legacyWrong := `{"schema_version":1,"scenario":"browser-automation-studio","instance":"shadow","owner_pid":` + fmt.Sprint(os.Getpid()) + `,"ports":{"api":15372}}`
	if err := os.WriteFile(Path(home, "browser-automation-studio"), []byte(legacyWrong), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := Read(home, "browser-automation-studio"); err == nil {
		t.Fatal("bare live lookup accepted a shadow record stored at the legacy path")
	}
	if _, err := Read(home, "browser-automation-studio@shadow"); !os.IsNotExist(err) {
		t.Fatalf("shadow lookup error = %v, want missing canonical shadow path", err)
	}
}

func TestReadAcceptsLegacyRecordWithoutInstanceWhenIncarnationMatches(t *testing.T) {
	home := t.TempDir()
	if err := os.MkdirAll(filepath.Dir(Path(home, "demo")), 0o700); err != nil {
		t.Fatal(err)
	}
	incarnation, err := process.Incarnation(os.Getpid())
	if err != nil {
		t.Skipf("process incarnation unavailable on this platform: %v", err)
	}
	legacy := `{"schema_version":1,"scenario":"demo","owner_pid":` + fmt.Sprint(os.Getpid()) + `,"incarnation":"` + incarnation + `","ports":{"api":18080}}`
	if err := os.WriteFile(Path(home, "demo"), []byte(legacy), 0o600); err != nil {
		t.Fatal(err)
	}
	if got, err := Read(home, "demo"); err != nil || got.Ports["api"] != 18080 {
		t.Fatalf("live legacy read = %#v, %v", got, err)
	}
	if _, err := Read(home, "demo@shadow"); !os.IsNotExist(err) {
		t.Fatalf("legacy record satisfied shadow lookup, error=%v", err)
	}
}

func TestPeerRecordIdentityIgnoresAmbientVariantSelection(t *testing.T) {
	t.Setenv(cliutil.EnvShadowScenarios, "ambient-peer")
	home := t.TempDir()
	live := Record{Scenario: "ambient-peer", OwnerPID: os.Getpid(), Ports: map[string]int{"api": 17116}}
	shadow := live
	shadow.Instance = "shadow"
	shadow.Ports = map[string]int{"api": 15372}
	if err := Write(home, live); err != nil {
		t.Fatal(err)
	}
	if err := Write(home, shadow); err != nil {
		t.Fatal(err)
	}
	for _, target := range []string{"ambient-peer", "ambient-peer@live"} {
		got, err := Read(home, target)
		if err != nil || got.Ports["api"] != 17116 {
			t.Fatalf("Read(%q) under ambient shadow selection = %#v, %v; want live port 17116", target, got, err)
		}
	}
	if err := Remove(home, "ambient-peer@live", os.Getpid()); err != nil {
		t.Fatal(err)
	}
	if _, err := Read(home, "ambient-peer"); !os.IsNotExist(err) {
		t.Fatalf("live record after scoped removal: error = %v, want not-exist", err)
	}
	if got, err := Read(home, "ambient-peer@shadow"); err != nil || got.Ports["api"] != 15372 {
		t.Fatalf("shadow record after live removal = %#v, %v; want shadow port 15372", got, err)
	}

	cliutil.SetInstanceOverride("override-peer", "shadow")
	defer cliutil.SetInstanceOverride("override-peer", "")
	overrideHome := t.TempDir()
	overrideLive := Record{Scenario: "override-peer", OwnerPID: os.Getpid(), Ports: map[string]int{"api": 18180}}
	if err := Write(overrideHome, overrideLive); err != nil {
		t.Fatal(err)
	}
	if got, err := Read(overrideHome, "override-peer@live"); err != nil || got.Ports["api"] != 18180 {
		t.Fatalf("explicit live record under CLI shadow override = %#v, %v; want port 18180", got, err)
	}
}

func TestPeerRecordRejectsCrossPlatformPathSeparators(t *testing.T) {
	home := t.TempDir()
	for _, tc := range []struct {
		name   string
		record Record
	}{
		{name: "scenario separator", record: Record{Scenario: `demo\..\outside`, OwnerPID: os.Getpid()}},
		{name: "variant separator", record: Record{Scenario: "demo", Instance: `..\..\outside`, OwnerPID: os.Getpid()}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if err := Write(home, tc.record); err == nil {
				t.Fatal("Write accepted a path separator in the record identity")
			}
		})
	}
	for _, target := range []string{`demo\..\outside`, `demo@..\..\outside`} {
		if _, err := Read(home, target); err == nil {
			t.Errorf("Read(%q) accepted a path separator", target)
		}
		if err := Remove(home, target); err == nil {
			t.Errorf("Remove(%q) accepted a path separator", target)
		}
	}
}
