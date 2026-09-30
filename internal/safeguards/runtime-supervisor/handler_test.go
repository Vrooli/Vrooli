package runtimesupervisorsafeguard

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	platformgo "github.com/vrooli/platform-go"
	"github.com/vrooli/vrooli/internal/hostreqkit"
	"github.com/vrooli/vrooli/internal/hostreqspec"
)

const manifestName = "runtime_supervisor"

type fakeHost struct {
	commands []string
	runs     []string
	show     string
	installs map[string]string
}

// seams points every host interaction at the fake and returns the restore
// function. The executable is /bin/sh so the real systemd validator, when it
// runs, sees an ExecStart that exists.
func seams(t *testing.T, home string, fake *fakeHost) {
	t.Helper()
	originalHome, originalRoot, originalOutput, originalRun, originalInstall, originalExecutable, originalValidate := userHomeFn, resolveRootFn, commandOutputFn, runCommandFn, installFileFn, executableFn, validateFn
	userHomeFn = func() (string, error) { return home, nil }
	resolveRootFn = func() (string, error) { return filepath.Join(home, "Vrooli"), nil }
	commandOutputFn = func(name string, args ...string) ([]byte, error) {
		fake.commands = append(fake.commands, name+" "+strings.Join(args, " "))
		return []byte(fake.show), nil
	}
	runCommandFn = func(name string, args []string, _ hostreqkit.EnsureOptions) error {
		fake.runs = append(fake.runs, name+" "+strings.Join(args, " "))
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
	})
}

func activeShow() string {
	return "ActiveState=active\nUnitFileState=enabled\nNRestarts=0\nResult=success\n"
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
