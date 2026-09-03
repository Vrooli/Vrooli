package agentsessioncontainment

import (
	"os"
	"slices"
	"strconv"
	"strings"
	"testing"

	platformgo "github.com/vrooli/platform-go"
	"github.com/vrooli/vrooli/internal/hostreqkit"
	"github.com/vrooli/vrooli/internal/hostreqkit/hostreqkittest"
	"github.com/vrooli/vrooli/internal/hostreqspec"
)

const physical = int64(64) << 30

// fixture is a fake user manager and kernel: what systemctl answers and what
// the cgroup files say, plus what was installed and run.
type fixture struct {
	files     map[string]string
	active    string
	tasksMax  int64
	cpuWeight int64
	memoryMax int64
	cgroup    string
	commands  []string
	installed map[string]string
	showFails bool
}

func newFixture(t *testing.T) *fixture {
	t.Helper()
	restore := hostreqkittest.StubLookups(t)
	f := &fixture{files: map[string]string{}, active: "active", tasksMax: DefaultTasksMax, cpuWeight: DefaultCPUWeight, memoryMax: physical * DefaultMemoryMaxPercent / percent, cgroup: "/user.slice/user-1000.slice/user@1000.service/vrooli-agents.slice", installed: map[string]string{}}
	f.files["/proc/meminfo"] = "MemTotal:       67108864 kB\n"
	f.syncCgroup()
	origHome, origValidate, origInstall, origRoot := homeDir, validateFn, installFileFn, hostreqkit.RunningAsRootFn
	homeDir = func() (string, error) { return "/home/op", nil }
	validateFn = func(platformgo.RenderedArtifact, platformgo.Scope) platformgo.Verdict {
		return platformgo.Verdict{State: platformgo.VerdictAccepted, Validator: "test"}
	}
	installFileFn = func(path, content string, _ hostreqkit.EnsureOptions) error {
		f.installed[path] = content
		f.files[path] = content
		return nil
	}
	hostreqkit.RunningAsRootFn = func() bool { return false }
	hostreqkit.ReadFileFn = func(path string) ([]byte, error) {
		if c, ok := f.files[path]; ok {
			return []byte(c), nil
		}
		return nil, os.ErrNotExist
	}
	hostreqkit.CombinedOutputFn = func(name string, args ...string) ([]byte, error) {
		f.commands = append(f.commands, name+" "+strings.Join(args, " "))
		if slices.Contains(args, "show") {
			if f.showFails {
				return []byte("Failed to connect to bus"), os.ErrNotExist
			}
			return []byte("ActiveState=" + f.active + "\nControlGroup=" + f.cgroup + "\nMemoryMax=" + itoa(f.memoryMax) + "\nTasksMax=" + itoa(f.tasksMax) + "\nCPUWeight=" + itoa(f.cpuWeight) + "\n"), nil
		}
		if slices.Contains(args, "start") {
			f.active = "active"
			f.syncCgroup()
		}
		return nil, nil
	}
	t.Cleanup(func() {
		restore()
		homeDir, validateFn, installFileFn, hostreqkit.RunningAsRootFn = origHome, origValidate, origInstall, origRoot
	})
	return f
}

func (f *fixture) syncCgroup() {
	dir := "/sys/fs/cgroup" + f.cgroup
	f.files[dir+"/memory.max"] = itoa(f.memoryMax) + "\n"
	f.files[dir+"/pids.max"] = itoa(f.tasksMax) + "\n"
	f.files[dir+"/cpu.weight"] = itoa(f.cpuWeight) + "\n"
}

func itoa(n int64) string { return strconv.FormatInt(n, 10) }

func newTestHandler() hostreqkit.Handler {
	return NewHandler(hostreqkit.SafeguardManifest{Name: "agent_session_containment", Handler: "agent_session_containment"})
}

func linuxReq() hostreqspec.ResolvedRequirement {
	return hostreqspec.ResolvedRequirement{Name: "agent_session_containment", Kind: hostreqspec.KindSafeguard, Required: true}
}

func (f *fixture) installRendered(t *testing.T) {
	t.Helper()
	artifact, err := Render(DefaultSettings())
	if err != nil {
		t.Fatal(err)
	}
	f.files["/home/op/.config/systemd/user/vrooli-agents.slice"] = artifact.Primary().Content
}

func TestRenderMatchesPlatformFixture(t *testing.T) {
	artifact, err := Render(DefaultSettings())
	if err != nil {
		t.Fatal(err)
	}
	content := artifact.Primary().Content
	for _, want := range []string{"[Slice]", "CPUWeight=50", "MemoryHigh=50%", "MemoryMax=60%", "TasksMax=4096", "ManagedOOMMemoryPressure=kill"} {
		if !strings.Contains(content, want) {
			t.Errorf("rendered slice lacks %q:\n%s", want, content)
		}
	}
	if artifact.Primary().Name != "vrooli-agents.slice" {
		t.Errorf("unit file name = %q", artifact.Primary().Name)
	}
}

func TestResolveSettingsKeepsDefaultsOnBadValues(t *testing.T) {
	s := ResolveSettings(map[string]any{"cpu_weight": 0, "memory_high_percent": 90, "memory_max_percent": 70, "tasks_max": 2})
	if s.CPUWeight != DefaultCPUWeight || s.MemoryHighPercent != 70 || s.MemoryMaxPercent != 70 || s.TasksMax != DefaultTasksMax {
		t.Fatalf("settings = %+v", s)
	}
	if c := s.Containment(); c.Slice != "vrooli-agents.slice" || c.MemoryMax != "70%" || c.TasksMax != DefaultTasksMax {
		t.Fatalf("containment = %+v", c)
	}
}

func TestInspectReportsAppliedWhenFileAndLiveMatch(t *testing.T) {
	f := newFixture(t)
	f.installRendered(t)
	status := newTestHandler().Inspect(hostreqkittest.LinuxHost(), linuxReq())
	if !status.Applied || status.ExecutionState != hostreqkit.ExecutionAlreadyPresent {
		t.Fatalf("status = %+v", status)
	}
	if _, ok := status.Evidence["validator_verdict"]; !ok {
		t.Fatalf("evidence lacks the validator verdict: %+v", status.Evidence)
	}
}

// [REQ:STORM-002] A slice that is written but not loaded, or loaded with
// other values, is not applied.
func TestInspectReportsNotAppliedWhenLiveValuesDiffer(t *testing.T) {
	f := newFixture(t)
	f.installRendered(t)
	f.tasksMax = 164514
	f.syncCgroup()
	status := newTestHandler().Inspect(hostreqkittest.LinuxHost(), linuxReq())
	if status.Applied || status.ExecutionState != hostreqkit.ExecutionPending {
		t.Fatalf("status = %+v", status)
	}
	if !strings.Contains(strings.Join(status.Notes, "\n"), "live TasksMax 164514, want 4096") {
		t.Fatalf("notes = %v", status.Notes)
	}
	f.tasksMax = DefaultTasksMax
	f.active = "inactive"
	status = newTestHandler().Inspect(hostreqkittest.LinuxHost(), linuxReq())
	if status.Applied || !strings.Contains(strings.Join(status.Notes, "\n"), "not active") {
		t.Fatalf("inactive slice read as applied: %+v", status)
	}
}

func TestInspectIsUndeterminedWhenTheProbeCannotRun(t *testing.T) {
	f := newFixture(t)
	f.installRendered(t)
	f.showFails = true
	status := newTestHandler().Inspect(hostreqkittest.LinuxHost(), linuxReq())
	if status.Applied || status.Evidence["probe"] != "undetermined" || status.ExecutionState != hostreqkit.ExecutionFailed {
		t.Fatalf("status = %+v", status)
	}
}

func TestApplyReverifiesLiveCgroup(t *testing.T) {
	f := newFixture(t)
	f.active = "inactive"
	h := newTestHandler()
	status := h.Inspect(hostreqkittest.LinuxHost(), linuxReq())
	if status.Applied {
		t.Fatal("nothing installed yet must not read as applied")
	}
	applied, err := h.Apply(hostreqkittest.LinuxHost(), status, hostreqkit.EnsureOptions{})
	if err != nil || !applied.Applied || applied.ExecutionState != hostreqkit.ExecutionApplied {
		t.Fatalf("apply = %+v, %v", applied, err)
	}
	if _, ok := f.installed["/home/op/.config/systemd/user/vrooli-agents.slice"]; !ok {
		t.Fatalf("slice not installed: %v", f.installed)
	}
	joined := strings.Join(f.commands, "\n")
	if !strings.Contains(joined, "daemon-reload") || !strings.Contains(joined, "start vrooli-agents.slice") {
		t.Fatalf("commands = %v", f.commands)
	}
	// The same apply against a manager that never activates the slice is a
	// failure, not a claim.
	g := newFixture(t)
	g.active = "inactive"
	hostreqkit.CombinedOutputFn = func(name string, args ...string) ([]byte, error) {
		if slices.Contains(args, "show") {
			return []byte("ActiveState=inactive\nControlGroup=\nMemoryMax=infinity\nTasksMax=infinity\nCPUWeight=100\n"), nil
		}
		return nil, nil
	}
	status = h.Inspect(hostreqkittest.LinuxHost(), linuxReq())
	applied, err = h.Apply(hostreqkittest.LinuxHost(), status, hostreqkit.EnsureOptions{})
	if err != nil || applied.Applied || applied.ExecutionState != hostreqkit.ExecutionFailed {
		t.Fatalf("unproven apply = %+v, %v", applied, err)
	}
}

func TestInspectOffLinuxReportsLauncherDefaults(t *testing.T) {
	status := newTestHandler().Inspect(hostreqkit.Host{OS: "darwin"}, linuxReq())
	if status.SupportClass != hostreqkit.SupportUnsupported || !strings.Contains(strings.Join(status.Notes, "\n"), "rlimit shim") {
		t.Fatalf("status = %+v", status)
	}
}
