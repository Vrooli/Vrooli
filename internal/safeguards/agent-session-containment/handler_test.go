package agentsessioncontainment

import (
	"os"
	"slices"
	"strconv"
	"strings"
	"testing"

	platformgo "github.com/vrooli/platform-go"
	"github.com/vrooli/vrooli/internal/agentscope"
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
	services, err := RenderServices(DefaultSettings())
	if err != nil {
		t.Fatal(err)
	}
	f.files["/home/op/.config/systemd/user/vrooli-services.slice"] = services.Primary().Content
}

func TestRenderMatchesPlatformFixture(t *testing.T) {
	artifact, err := Render(DefaultSettings())
	if err != nil {
		t.Fatal(err)
	}
	content := artifact.Primary().Content
	for _, want := range []string{"[Slice]", "CPUWeight=50", "MemoryHigh=50%", "MemoryMax=60%", "TasksMax=16384", "ManagedOOMMemoryPressure=kill"} {
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
	if !strings.Contains(strings.Join(status.Notes, "\n"), "live TasksMax 164514, want 16384") {
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

// A ceiling is a safety property only while there is room under it. On
// 2026-09-04 the slice reached its task ceiling with every host-level bar
// green, and the first symptom was Codex reporting its database corrupt.
func TestOccupancyNotesReportASaturatedTaskCeiling(t *testing.T) {
	s := DefaultSettings()
	l := live{Occupancy: platformgo.Occupancy{Tasks: 4059, TasksMax: 4096, MemoryBytes: 1, MemoryMaxBytes: 100}}
	notes := strings.Join(occupancyNotes(l, s), "\n")
	if !strings.Contains(notes, "4059 of 4096 tasks") {
		t.Fatalf("a saturated task ceiling must be reported: %q", notes)
	}
	if !strings.Contains(notes, "refuses the fork") {
		t.Fatalf("the note must say what happens at the ceiling: %q", notes)
	}
}

// A full slice is a correctly configured slice: occupancy must never be read
// as configuration drift, or applying the safeguard would look like the fix.
func TestFullSliceIsNotDrift(t *testing.T) {
	s := DefaultSettings()
	l := live{
		ActiveState: "active", ControlGroup: "/user.slice/vrooli-agents.slice",
		TasksMax: int64(s.TasksMax), CPUWeight: int64(s.CPUWeight),
		CgroupPids: strconv.Itoa(s.TasksMax), CgroupWeight: strconv.Itoa(s.CPUWeight),
		Occupancy: platformgo.Occupancy{Tasks: 16000, TasksMax: 16384},
	}
	for _, mismatch := range liveMismatches(l, s) {
		if strings.Contains(mismatch, "tasks") && strings.Contains(mismatch, "16000") {
			t.Fatalf("occupancy must not be reported as drift: %q", mismatch)
		}
	}
	if len(occupancyNotes(l, s)) == 0 {
		t.Fatal("a slice at 98% of its task ceiling must still be reported")
	}
}

// A ceiling with no reading is undetermined, never room to spare.
func TestOccupancyNotesSayUndeterminedRatherThanHealthy(t *testing.T) {
	l := live{Occupancy: platformgo.Occupancy{Tasks: platformgo.Unknown, TasksMax: 16384}}
	notes := strings.Join(occupancyNotes(l, DefaultSettings()), "\n")
	if !strings.Contains(notes, "undetermined") {
		t.Fatalf("an unreadable ceiling is undetermined: %q", notes)
	}
}

// The tasks a session left behind are the ratchet: nothing releases them, so
// the usable ceiling falls with every session that started a scenario.
func TestOccupancyNotesNameTheAgentlessScopes(t *testing.T) {
	l := live{
		Occupancy: platformgo.Occupancy{Tasks: 10, TasksMax: 16384, MemoryBytes: 1, MemoryMaxBytes: 100},
		Sessions: []agentscope.Entry{
			{Name: "vrooli-agent-codex-live.scope", Tasks: 38},
			{Name: "vrooli-agent-codex-eb0c.scope", Tasks: 787, Agentless: true},
		},
	}
	notes := strings.Join(occupancyNotes(l, DefaultSettings()), "\n")
	if !strings.Contains(notes, "1 of 2 session scopes hold 787 tasks") {
		t.Fatalf("agentless scopes must be named with what they hold: %q", notes)
	}
}

// The services slice is the other half of the containment layout. Without it
// a long-running service has nowhere to go but the cgroup of whatever started
// it, which is how the agent slice came to hold the fleet.
func TestServicesSliceCarriesNoKillingCeiling(t *testing.T) {
	c := DefaultSettings().ServicesContainment()
	if c.MemoryMax != "" {
		t.Fatalf("a hard memory ceiling on services makes the kernel kill llama-server or qdrant: MemoryMax=%q", c.MemoryMax)
	}
	if c.MemoryHigh == "" {
		t.Fatal("services must still be throttled under pressure")
	}
	if c.CPUWeight <= DefaultCPUWeight {
		t.Fatalf("services must outrank agent sessions for CPU: services %d, agents %d", c.CPUWeight, DefaultCPUWeight)
	}
	if c.TasksMax < minimumTasksMax {
		t.Fatalf("services need a runaway ceiling: TasksMax=%d", c.TasksMax)
	}
}

func TestRenderServicesProducesItsOwnUnit(t *testing.T) {
	artifact, err := RenderServices(DefaultSettings())
	if err != nil {
		t.Fatal(err)
	}
	if artifact.Primary().Name != ServicesSliceUnit {
		t.Fatalf("unit name: got %q, want %q", artifact.Primary().Name, ServicesSliceUnit)
	}
	content := artifact.Primary().Content
	for _, want := range []string{"[Slice]", "CPUWeight=200", "MemoryHigh=70%", "TasksMax=16384"} {
		if !strings.Contains(content, want) {
			t.Fatalf("rendered services slice is missing %q:\n%s", want, content)
		}
	}
	for _, unwanted := range []string{"MemoryMax=", "ManagedOOMMemoryPressure=kill"} {
		if strings.Contains(content, unwanted) {
			t.Fatalf("rendered services slice must not carry %q:\n%s", unwanted, content)
		}
	}
}

// Both slices are one safeguard's business: a host with the agent ceiling and
// nowhere for services to live is the state that produced the outage.
func TestInspectIsPendingWhenTheServicesSliceIsMissing(t *testing.T) {
	f := newFixture(t)
	f.installRendered(t)
	delete(f.files, "/home/op/.config/systemd/user/vrooli-services.slice")
	status := newTestHandler().Inspect(hostreqkittest.LinuxHost(), linuxReq())
	if status.Applied {
		t.Fatal("a host with no services slice is not fully applied")
	}
	if !strings.Contains(strings.Join(status.Notes, "\n"), ServicesSliceUnit) {
		t.Fatalf("the missing services slice must be named: %v", status.Notes)
	}
}
