package runtimesupervisorsafeguard

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	repocontract "github.com/vrooli/repo-contract-go"
	testkit "github.com/vrooli/repo-contract-go/repocontracttest"
	"github.com/vrooli/vrooli/internal/buildinfo"
	"github.com/vrooli/vrooli/internal/coreactivation"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"

	platformgo "github.com/vrooli/platform-go"
	"github.com/vrooli/vrooli/internal/hostreqkit"
	"github.com/vrooli/vrooli/internal/hostreqspec"
)

func TestConvergeRejectsActiveUnitWithoutIntendedProcessImage(t *testing.T) {
	for _, mode := range []string{"stale", "unreadable", "missing pid", "adopted"} {
		t.Run(mode, func(t *testing.T) {
			home := t.TempDir()
			fake := &fakeHost{show: activeShow(), keepStale: true}
			switch mode {
			case "stale":
				fake.imageStale = true
			case "unreadable":
				fake.imageErr = errors.New("fixture image unavailable")
			case "missing pid":
				fake.show = strings.ReplaceAll(fake.show, "MainPID=1234\n", "")
			}
			seams(t, home, fake)
			_, err := Converge(context.Background(), ConvergeOptions{OS: "linux", Home: home, Root: filepath.Join(home, "Vrooli"), Restart: true})
			if mode == "adopted" {
				if err != nil {
					t.Fatal(err)
				}
			} else if err == nil {
				t.Fatal("native active state was accepted without proving the intended process image")
			}
		})
	}
}

func TestConvergenceRefusesUnresolvedActivationBeforeAnyHostMutation(t *testing.T) {
	home := t.TempDir()
	fake := &fakeHost{show: activeShow()}
	seams(t, home, fake)
	guardRootActivationFn = func(context.Context, string) error { return errors.New("root activation remains unresolved") }
	if _, err := Converge(context.Background(), ConvergeOptions{OS: "linux", Home: home, Root: filepath.Join(home, "source"), Restart: true}); err == nil {
		t.Fatal("unresolved root operation was crossed")
	}
	if len(fake.runs) != 0 || len(fake.installs) != 0 || len(fake.commands) != 0 {
		t.Fatalf("host effects after denied activation: runs=%v installs=%v probes=%v", fake.runs, fake.installs, fake.commands)
	}
}

func TestConvergencePreservesUnitWhenInstalledBootstrapIsUnqualified(t *testing.T) {
	home := t.TempDir()
	fake := &fakeHost{show: activeShow()}
	seams(t, home, fake)
	probeRootBootstrapFn = func(context.Context, string, string) error {
		return errors.New("compiled authority missing or mismatched")
	}
	if _, err := Converge(context.Background(), ConvergeOptions{OS: "linux", Home: home, Root: filepath.Join(home, "source"), Restart: true}); err == nil {
		t.Fatal("unqualified installed CLI changed native unit")
	}
	if len(fake.runs) != 0 || len(fake.installs) != 0 || len(fake.commands) != 0 {
		t.Fatal("native effects occurred before bootstrap qualification")
	}
}

const manifestName = "runtime_supervisor"

type fakeHost struct {
	commands   []string
	runs       []string
	show       string
	installs   map[string]string
	imageStale bool
	keepStale  bool
	imageErr   error
}

// seams points every host interaction at the fake and returns the restore
// function. The executable is /bin/sh so the real systemd validator, when it
// runs, sees an ExecStart that exists.
func seams(t *testing.T, home string, fake *fakeHost) {
	t.Helper()
	originalHome, originalRoot, originalOutput, originalRun, originalInstall, originalExecutable, originalValidate := userHomeFn, resolveRootFn, commandOutputFn, runCommandFn, installFileFn, executableFn, validateFn
	originalImage := processImageMatchesFn
	originalLock, originalGuard := acquireInstallLockFn, guardRootActivationFn
	originalBootstrap := probeRootBootstrapFn
	acquireInstallLockFn = func(context.Context, string) (func(), error) { return func() {}, nil }
	probeRootBootstrapFn = func(context.Context, string, string) error { return nil }
	processImageMatchesFn = func(int, string) (bool, error) { return !fake.imageStale, fake.imageErr }
	userHomeFn = func() (string, error) { return home, nil }
	resolveRootFn = func() (string, error) { return filepath.Join(home, "Vrooli"), nil }
	commandOutputFn = func(name string, args ...string) ([]byte, error) {
		fake.commands = append(fake.commands, name+" "+strings.Join(args, " "))
		return []byte(fake.show), nil
	}
	runCommandFn = func(name string, args []string, _ hostreqkit.EnsureOptions) error {
		fake.runs = append(fake.runs, name+" "+strings.Join(args, " "))
		if strings.Contains(strings.Join(args, " "), "restart vrooli-runtime-supervisor.service") && !fake.keepStale {
			fake.imageStale = false
		}
		return nil
	}
	installFileFn = func(path, content string, _ hostreqkit.EnsureOptions) error {
		if fake.installs == nil {
			fake.installs = map[string]string{}
		}
		fake.installs[path] = content
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			return err
		}
		return os.WriteFile(path, []byte(content), 0o644)
	}
	executableFn = func(string, string) (string, bool, error) { return "/bin/sh", true, nil }
	validateFn = func(platformgo.RenderedArtifact, platformgo.Scope) platformgo.Verdict {
		return platformgo.Verdict{State: platformgo.VerdictAccepted, Validator: "fake"}
	}
	t.Cleanup(func() {
		userHomeFn, resolveRootFn, commandOutputFn, runCommandFn, installFileFn, executableFn, validateFn = originalHome, originalRoot, originalOutput, originalRun, originalInstall, originalExecutable, originalValidate
		processImageMatchesFn = originalImage
		acquireInstallLockFn, guardRootActivationFn = originalLock, originalGuard
		probeRootBootstrapFn = originalBootstrap
	})
}

func activeShow() string {
	return "ActiveState=active\nUnitFileState=enabled\nNRestarts=0\nResult=success\nMainPID=1234\n"
}

func TestInstalledBinaryRequiresRunningProcessAdoption(t *testing.T) {
	for _, mode := range []string{"stale", "unreadable", "missing pid"} {
		t.Run(mode, func(t *testing.T) {
			home := t.TempDir()
			fake := &fakeHost{show: activeShow()}
			switch mode {
			case "stale":
				fake.imageStale = true
			case "unreadable":
				fake.imageErr = errors.New("process image unreadable")
			case "missing pid":
				fake.show = strings.ReplaceAll(fake.show, "MainPID=1234\n", "")
			}
			seams(t, home, fake)
			rendered, err := Render("linux", home, filepath.Join(home, "Vrooli"), "")
			if err != nil {
				t.Fatal(err)
			}
			writeUnit(t, home, rendered.Artifact.Primary().Content)
			status := NewHandler(hostreqkit.SafeguardManifest{Name: manifestName}).Inspect(linuxHost(), requirement())
			if status.Applied || status.Evidence["process_image_current"] != false {
				t.Fatalf("active unit masked image gap: %+v", status)
			}
		})
	}
}

func TestApplyAdoptsNewBinaryWhenUnitContentIsCurrent(t *testing.T) {
	for _, remainsStale := range []bool{false, true} {
		t.Run(strconv.FormatBool(remainsStale), func(t *testing.T) {
			home := t.TempDir()
			fake := &fakeHost{show: activeShow(), imageStale: true, keepStale: remainsStale}
			seams(t, home, fake)
			rendered, err := Render("linux", home, filepath.Join(home, "Vrooli"), "")
			if err != nil {
				t.Fatal(err)
			}
			writeUnit(t, home, rendered.Artifact.Primary().Content)
			h := NewHandler(hostreqkit.SafeguardManifest{Name: manifestName})
			status := h.Inspect(linuxHost(), requirement())
			applied, err := h.Apply(linuxHost(), status, hostreqkit.EnsureOptions{})
			if err != nil {
				t.Fatal(err)
			}
			if !strings.Contains(strings.Join(fake.runs, "\n"), "restart vrooli-runtime-supervisor.service") {
				t.Fatal("current unit content suppressed binary adoption")
			}
			if remainsStale {
				if applied.Applied || applied.ExecutionState != hostreqkit.ExecutionFailed {
					t.Fatalf("unadopted binary reported applied: %+v", applied)
				}
			} else if !applied.Applied || applied.Evidence["process_image_current"] != true {
				t.Fatalf("adoption unverified: %+v", applied)
			}
		})
	}
}

func TestProcessImageComparisonUsesLiveFileIdentity(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("procfs identity is Linux-only")
	}
	path, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	if match, err := processImageMatches(os.Getpid(), path); err != nil || !match {
		t.Fatalf("current image match=%t error=%v", match, err)
	}
	other := filepath.Join(t.TempDir(), "replacement")
	if err := os.WriteFile(other, []byte("new image"), 0o700); err != nil {
		t.Fatal(err)
	}
	if match, err := processImageMatches(os.Getpid(), other); err != nil || match {
		t.Fatalf("replacement match=%t error=%v", match, err)
	}
	if match, err := processImageMatches(0, path); err == nil || match {
		t.Fatal("missing PID proved an image match")
	}
}

func linuxHost() hostreqkit.Host {
	return hostreqkit.Host{OS: "linux", SupportsSystemd: true}
}

func requirement() hostreqspec.ResolvedRequirement {
	return hostreqspec.ResolvedRequirement{Name: manifestName, Kind: hostreqspec.KindSafeguard, Required: true}
}

func writeUnit(t *testing.T, home, content string) string {
	t.Helper()
	path := platformgo.RuntimeSupervisorUnitPath("linux", home)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

// [REQ:BOOT-RECOVERY-001] A unit whose ExecStart carries a retired global is
// reported not-applied with a note that names the drift. This is the exact
// state of this host on 2026-09-02 07:36, when the unit rendered on 08-18
// still passed --no-stale-check and crash-looped 495 times.
func TestRuntimeSupervisorSafeguardDetectsStaleExecStart(t *testing.T) {
	home := t.TempDir()
	fake := &fakeHost{show: activeShow()}
	seams(t, home, fake)
	writeUnit(t, home, "[Unit]\nDescription=old\n\n[Service]\nExecStart=/bin/sh --no-stale-check runtime supervisor run\n")

	status := NewHandler(hostreqkit.SafeguardManifest{Name: manifestName}).Inspect(linuxHost(), requirement())
	if status.Applied {
		t.Fatalf("status = %+v, want not applied for a stale unit", status)
	}
	notes := strings.Join(status.Notes, "\n")
	if !strings.Contains(notes, "missing or stale") || !strings.Contains(notes, "--no-stale-check") || !strings.Contains(notes, "runtime supervisor run") {
		t.Fatalf("notes = %q, want the drift named with both ExecStart lines", notes)
	}
	if _, ok := status.Evidence["validator_verdict"]; !ok {
		t.Fatalf("evidence = %+v, want validator_verdict", status.Evidence)
	}
}

// [REQ:BOOT-RECOVERY-001] A matching, validated, enabled and active unit is
// applied, with the unit state in evidence.
func TestInspectReportsAppliedWhenUnitMatchesAndIsActive(t *testing.T) {
	home := t.TempDir()
	fake := &fakeHost{show: activeShow()}
	seams(t, home, fake)
	rendered, err := Render("linux", home, filepath.Join(home, "Vrooli"), "")
	if err != nil {
		t.Fatal(err)
	}
	writeUnit(t, home, rendered.Artifact.Primary().Content)

	status := NewHandler(hostreqkit.SafeguardManifest{Name: manifestName}).Inspect(linuxHost(), requirement())
	if !status.Applied || status.ExecutionState != hostreqkit.ExecutionAlreadyPresent {
		t.Fatalf("status = %+v, want applied", status)
	}
	state, ok := status.Evidence["unit_state"].(map[string]string)
	if !ok || state["NRestarts"] != "0" || state["Result"] != "success" {
		t.Fatalf("unit_state evidence = %+v, want NRestarts and Result", status.Evidence["unit_state"])
	}
}

// [REQ:BOOT-RECOVERY-001] An active unit whose content drifted is restarted
// onto the new definition after reset-failed, and Apply re-verifies.
func TestApplyRestartsWhenContentChangedAndReverifies(t *testing.T) {
	home := t.TempDir()
	fake := &fakeHost{show: activeShow()}
	seams(t, home, fake)
	writeUnit(t, home, "[Unit]\nDescription=old\n\n[Service]\nExecStart=/bin/sh --no-stale-check runtime supervisor run\n")

	h := NewHandler(hostreqkit.SafeguardManifest{Name: manifestName})
	status := h.Inspect(linuxHost(), requirement())
	applied, err := h.Apply(linuxHost(), status, hostreqkit.EnsureOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if !applied.Applied || applied.ExecutionState != hostreqkit.ExecutionApplied {
		t.Fatalf("applied = %+v, want applied", applied)
	}
	runs := strings.Join(fake.runs, "\n")
	for _, want := range []string{"daemon-reload", "enable --now vrooli-runtime-supervisor.service", "reset-failed vrooli-runtime-supervisor.service", "restart vrooli-runtime-supervisor.service"} {
		if !strings.Contains(runs, want) {
			t.Fatalf("runs = %q, want %q", runs, want)
		}
	}
	installed := fake.installs[platformgo.RuntimeSupervisorUnitPath("linux", home)]
	if strings.Contains(installed, "--no-stale-check") || !strings.Contains(installed, "runtime supervisor run") {
		t.Fatalf("installed unit = %q, want the rendered definition", installed)
	}
}

// [REQ:BOOT-RECOVERY-001] A rejected render installs nothing.
func TestApplyRefusesRejectedRender(t *testing.T) {
	home := t.TempDir()
	fake := &fakeHost{show: activeShow()}
	seams(t, home, fake)
	validateFn = func(platformgo.RenderedArtifact, platformgo.Scope) platformgo.Verdict {
		return platformgo.Verdict{State: platformgo.VerdictRejected, Validator: "fake", Output: "bad directive"}
	}
	h := NewHandler(hostreqkit.SafeguardManifest{Name: manifestName})
	status := h.Inspect(linuxHost(), requirement())
	applied, err := h.Apply(linuxHost(), status, hostreqkit.EnsureOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if applied.Applied || applied.ExecutionState != hostreqkit.ExecutionFailed || len(fake.installs) != 0 {
		t.Fatalf("applied = %+v installs = %v, want failed with nothing installed", applied, fake.installs)
	}
}

// [REQ:BOOT-RECOVERY-001] An enabled but inactive unit is pending, and the
// non-success Result is surfaced with the log path.
func TestInspectReportsCrashLoopingUnit(t *testing.T) {
	home := t.TempDir()
	fake := &fakeHost{show: "ActiveState=activating\nUnitFileState=enabled\nNRestarts=495\nResult=exit-code\n"}
	seams(t, home, fake)
	rendered, err := Render("linux", home, filepath.Join(home, "Vrooli"), "")
	if err != nil {
		t.Fatal(err)
	}
	writeUnit(t, home, rendered.Artifact.Primary().Content)
	status := NewHandler(hostreqkit.SafeguardManifest{Name: manifestName}).Inspect(linuxHost(), requirement())
	notes := strings.Join(status.Notes, "\n")
	if status.Applied || !strings.Contains(notes, "exit-code") || !strings.Contains(notes, "495") {
		t.Fatalf("status = %+v, want pending with the crash-loop result named", status)
	}
}

func TestNativeDefinitionAuthoritySurvivesSourceAbsence(t *testing.T) {
	root := t.TempDir()
	testkit.WriteRepoContract(t, root, "scenarios")
	c, err := repocontract.LoadDefault(root)
	if err != nil {
		t.Fatal(err)
	}
	data, err := c.RuntimeHomeSnapshot()
	if err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(data)
	want := hex.EncodeToString(sum[:])
	old := buildinfo.RuntimeHomeSnapshotBase64
	t.Cleanup(func() { buildinfo.RuntimeHomeSnapshotBase64 = old })
	buildinfo.RuntimeHomeSnapshotBase64 = base64.StdEncoding.EncodeToString(data)
	if err := os.RemoveAll(root); err != nil {
		t.Fatal(err)
	}
	got, err := nativeDefinitionHomeDigest(root)
	if err != nil || got != want {
		t.Fatalf("compiled authority required absent source: %s %v", got, err)
	}
	buildinfo.RuntimeHomeSnapshotBase64 = "invalid"
	if _, err := nativeDefinitionHomeDigest(root); err == nil {
		t.Fatal("invalid compiled authority fell back")
	}
	buildinfo.RuntimeHomeSnapshotBase64 = ""
	if _, err := nativeDefinitionHomeDigest(root); err == nil {
		t.Fatal("unstamped source-backed helper silently invented source-free authority")
	}
}

func TestRetainedSupervisorDefinitionGuardUnderInstallLock(t *testing.T) {
	for _, foreign := range []bool{false, true} {
		t.Run(strconv.FormatBool(foreign), func(t *testing.T) {
			home := t.TempDir()
			fake := &fakeHost{show: activeShow()}
			seams(t, home, fake)
			executable := filepath.Join(home, "prior-executable")
			data, err := os.ReadFile("/bin/sh")
			if err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(executable, data, 0700); err != nil {
				t.Fatal(err)
			}
			executableFn = func(string, string) (string, bool, error) { return executable, true, nil }

			prior, err := Render("linux", home, filepath.Join(home, "absent-source"), "")
			if err != nil {
				t.Fatal(err)
			}
			if err := os.MkdirAll(filepath.Dir(prior.Path), 0700); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(prior.Path, []byte("own candidate definition"), 0644); err != nil {
				t.Fatal(err)
			}
			current, _ := boundedDefinitionDigest(prior.Path)
			exe, err := coreactivation.ImageDigest(context.Background(), prior.Executable)
			if err != nil {
				t.Fatal(err)
			}
			sum := sha256.Sum256([]byte(prior.Artifact.Primary().Content))
			target := hex.EncodeToString(sum[:])
			// Simulate the previous canonical publisher completing while our caller
			// waits for the install lock. The expected-current check must occur later.
			acquireInstallLockFn = func(context.Context, string) (func(), error) {
				if foreign {
					if err := os.WriteFile(prior.Path, []byte("foreign definition"), 0644); err != nil {
						t.Fatal(err)
					}
				}
				return func() {}, nil
			}
			oldRestore := restoreDefinitionFn
			t.Cleanup(func() { restoreDefinitionFn = oldRestore })
			restoreDefinitionFn = func(path, content string) error {
				fake.installs = map[string]string{path: content}
				return os.WriteFile(path, []byte(content), 0644)
			}
			_, err = Converge(context.Background(), ConvergeOptions{OS: "linux", Home: home, Root: filepath.Join(home, "absent-source"), Restart: true, ExpectedExecutableSHA: exe, ExpectedCurrentDefinitionSHA: current, RetainedDefinitionSHA: target})
			if foreign {
				if err == nil {
					t.Fatal("foreign current definition overwritten")
				}
				if len(fake.installs) != 0 || len(fake.runs) != 0 {
					t.Fatal("native effects before foreign guard")
				}
				data, _ := os.ReadFile(prior.Path)
				if string(data) != "foreign definition" {
					t.Fatal("foreign bytes changed")
				}
			} else if err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestGuardedRetainedConvergenceDryRunHasNoEffects(t *testing.T) {
	home := t.TempDir()
	fake := &fakeHost{show: activeShow()}
	seams(t, home, fake)
	old := restoreDefinitionFn
	t.Cleanup(func() { restoreDefinitionFn = old })
	restoreDefinitionFn = func(string, string) error { t.Fatal("guarded dry-run wrote a definition"); return nil }
	if _, err := Converge(t.Context(), ConvergeOptions{OS: "linux", Home: home, Root: filepath.Join(home, "source"), ExpectedExecutableSHA: strings.Repeat("a", 64), ExpectedCurrentDefinitionSHA: strings.Repeat("b", 64), RetainedDefinitionSHA: strings.Repeat("c", 64), Ensure: hostreqkit.EnsureOptions{DryRun: true}}); err == nil {
		t.Fatal("guarded dry-run silently executed")
	}
	if len(fake.installs) != 0 || len(fake.runs) != 0 || len(fake.commands) != 0 {
		t.Fatal("guarded dry-run reached manager effects")
	}
}
