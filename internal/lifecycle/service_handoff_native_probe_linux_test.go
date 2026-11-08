//go:build linux

package lifecycle

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	platform "github.com/vrooli/platform-go"
	testkit "github.com/vrooli/repo-contract-go/repocontracttest"
	"github.com/vrooli/vrooli/internal/process"
	"github.com/vrooli/vrooli/internal/scenario"
)

// These operator-opt-in probes never run through the common unit factory.
// They exercise real placement, with private fixture state and exact owners.
const handoffPlanEnv = "VROOLI_SERVICE_HANDOFF_NATIVE_PLAN"
const handoffCaseEnv = "VROOLI_SERVICE_HANDOFF_NATIVE_CASE"

type handoffNativePlan struct {
	Worker        string              `json:"worker"`
	WorkerSHA     string              `json:"worker_sha256"`
	SourceRoot    string              `json:"source_root"`
	Executable    string              `json:"executable"`
	ExecutableSHA string              `json:"executable_sha256"`
	Script        string              `json:"script"`
	ScriptSHA     string              `json:"script_sha256"`
	Proposal      string              `json:"proposal"`
	ProposalSHA   string              `json:"proposal_sha256"`
	Sources       map[string]string   `json:"sources"`
	Cases         []handoffNativeCase `json:"cases"`
}
type handoffNativeCase struct {
	Mode     string `json:"mode"`
	Attempt  string `json:"attempt_id"`
	Owner    string `json:"operation_owner"`
	Slug     string `json:"scenario_slug"`
	Step     string `json:"step"`
	Unit     string `json:"scope_unit"`
	Dir      string `json:"private_dir"`
	PlanPath string `json:"plan_path"`
}
type handoffPID struct {
	PID   int    `json:"pid"`
	Start string `json:"start_tick"`
	Scope string `json:"scope"`
	Exe   string `json:"exe"`
}
type handoffLaunch struct {
	Caller       handoffPID  `json:"caller"`
	Service      handoffPID  `json:"service"`
	StandIn      *handoffPID `json:"stand_in,omitempty"`
	LaunchMillis int64       `json:"launch_ms"`
}

func handoffWrite(t *testing.T, path string, v any) {
	t.Helper()
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		t.Fatalf("receipt already exists or cannot be created %s: %v", path, err)
	}
	if _, err = f.Write(append(b, '\n')); err == nil {
		err = f.Sync()
	}
	closeErr := f.Close()
	if err != nil {
		t.Fatal(err)
	}
	if closeErr != nil {
		t.Fatal(closeErr)
	}
}
func handoffRead(t *testing.T, path string, v any) {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if err = json.Unmarshal(b, v); err != nil {
		t.Fatal(err)
	}
}
func handoffHash(t *testing.T, path, want string) {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	v := sha256.Sum256(b)
	if hex.EncodeToString(v[:]) != want {
		t.Fatalf("frozen input changed: %s", path)
	}
}
func handoffPlan(t *testing.T, path string) handoffNativePlan {
	t.Helper()
	var p handoffNativePlan
	handoffRead(t, path, &p)
	handoffHash(t, p.Executable, p.ExecutableSHA)
	handoffHash(t, p.Worker, p.WorkerSHA)
	handoffHash(t, p.Script, p.ScriptSHA)
	handoffHash(t, p.Proposal, p.ProposalSHA)
	if p.ProposalSHA != "c3dd6cd277427f76b0f30f4c55d365aa49f68640125a61ef69ee9aa296f5f9d2" || len(p.Sources) != 7 || len(p.Cases) != 2 {
		t.Fatal("unreviewed native plan")
	}
	for name, hash := range p.Sources {
		handoffHash(t, filepath.Join(p.SourceRoot, name), hash)
	}
	actual, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	handoffHash(t, actual, p.ExecutableSHA)
	return p
}
func handoffAwaitFile(t *testing.T, path string, limit time.Duration) {
	t.Helper()
	deadline := time.Now().Add(limit)
	for {
		if _, err := os.Stat(path); err == nil {
			return
		}
		if time.Now().After(deadline) {
			t.Fatalf("receipt not received within %s: %s", limit, path)
		}
		time.Sleep(20 * time.Millisecond)
	}
}
func handoffCaseEnvList(t *testing.T, path string) []string {
	t.Helper()
	home, err := os.UserHomeDir()
	if err != nil {
		t.Fatal(err)
	}
	return []string{"HOME=" + home, "PATH=" + platform.DefaultPath("linux", home), handoffCaseEnv + "=" + path}
}

func TestServiceHandoffNativeFixtureProbe(t *testing.T) {
	planPath := os.Getenv(handoffPlanEnv)
	if planPath == "" {
		t.Skip("requires exact reviewed service-handoff native plan")
	}
	p := handoffPlan(t, planPath)
	for _, c := range p.Cases {
		if !t.Run(c.Mode, func(t *testing.T) { handoffRunNativeCase(t, p, c, planPath) }) {
			return
		}
	}
}
func handoffRunNativeCase(t *testing.T, p handoffNativePlan, c handoffNativeCase, planPath string) {
	// A pre-existing case is a lost-ack/reconciliation condition, never permission
	// to replay native dispatch under its old attempt.
	if err := os.Mkdir(c.Dir, 0700); err != nil {
		t.Fatalf("private case already exists or cannot be created; reconcile without replay: %v", err)
	}
	c.PlanPath = planPath
	casePath := filepath.Join(c.Dir, "case.json")
	handoffWrite(t, casePath, c)
	root, home := filepath.Join(c.Dir, "root"), filepath.Join(c.Dir, "home")
	testkit.WriteRepoContract(t, root, "scenarios")
	source := filepath.Join(root, "scenarios", c.Slug)
	if err := os.MkdirAll(filepath.Join(source, ".vrooli"), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(home, 0700); err != nil {
		t.Fatal(err)
	}
	manifest := scenario.ServiceManifest{Service: scenario.ServiceMetadata{Name: c.Slug}, Lifecycle: scenario.Lifecycle{Develop: scenario.Phase{Steps: []scenario.PhaseStep{{Name: c.Step, Exec: []string{"/usr/bin/python3", p.Script, c.Dir}, Background: true}}}}}
	b, err := json.Marshal(manifest)
	if err != nil {
		t.Fatal(err)
	}
	if err = os.WriteFile(filepath.Join(source, ".vrooli", "service.json"), b, 0600); err != nil {
		t.Fatal(err)
	}
	spec := platform.OperationSpec{Path: p.Worker, Args: []string{}, Env: handoffCaseEnvList(t, casePath), Dir: root, Limits: platform.OperationLimits{MemoryMaxBytes: 512 << 20, TasksMax: 64, CPUWeight: 50, WallTime: 15 * time.Second}}
	intent, err := platform.NewOperationIntent(c.Owner, c.Attempt, spec)
	if err != nil {
		t.Fatal(err)
	}
	handoffWrite(t, filepath.Join(c.Dir, "intent.json"), intent)
	ctx, cancel := context.WithTimeout(t.Context(), 70*time.Second)
	defer cancel()
	start := time.Now()
	owned, err := platform.StartOwnedOperation(ctx, intent, spec)
	if err != nil {
		t.Fatalf("dispatch uncertain; retain intent, do not replay: %v", err)
	}
	handoffWrite(t, filepath.Join(c.Dir, "ref.json"), owned.Ref)
	handoffAwaitFile(t, filepath.Join(c.Dir, "caller-root.json"), 2*time.Second)
	var rootPID handoffPID
	handoffRead(t, filepath.Join(c.Dir, "caller-root.json"), &rootPID)
	if rootPID.Scope != owned.Ref.Scope.Path {
		t.Fatal("native caller scope mismatch")
	}
	limits := map[string]string{}
	for _, file := range []string{"memory.max", "pids.max", "cpu.weight"} {
		b, e := os.ReadFile(filepath.Join("/sys/fs/cgroup", owned.Ref.Scope.Path, file))
		if e != nil {
			t.Fatal(e)
		}
		limits[file] = strings.TrimSpace(string(b))
	}
	if limits["memory.max"] != "536870912" || limits["pids.max"] != "64" || limits["cpu.weight"] != "50" {
		t.Fatalf("wrong caller limits: %v", limits)
	}
	show := exec.CommandContext(ctx, "systemctl", "--user", "show", "--property=RuntimeMaxUSec,TimeoutStopUSec,InvocationID,MainPID,ControlGroup", "--", owned.Ref.Unit)
	native, outErr := show.Output()
	if outErr != nil {
		t.Fatal(outErr)
	}
	handoffWrite(t, filepath.Join(c.Dir, "native-live.json"), map[string]any{"ref": owned.Ref, "root": rootPID, "kernel_limits": limits, "manager": string(native)})
	handoffWrite(t, filepath.Join(c.Dir, "observer-ack.json"), map[string]any{"at": time.Now().UTC()})
	handoffAwaitFile(t, filepath.Join(c.Dir, "launch.json"), 5*time.Second)
	var launch handoffLaunch
	handoffRead(t, filepath.Join(c.Dir, "launch.json"), &launch)
	if filepath.Base(launch.Service.Scope) != c.Unit || launch.Caller.Scope != owned.Ref.Scope.Path {
		t.Fatal("actual launch membership differs")
	}
	if c.Mode == "native-expiry" && (launch.StandIn == nil || launch.StandIn.Scope != owned.Ref.Scope.Path) {
		t.Fatal("missing bounded provider stand-in")
	}
	serviceRef, err := platform.CaptureServiceScope(ctx, platform.ScopeRef{Name: strings.TrimSuffix(c.Unit, ".scope"), Kind: platform.ScopeKindCgroup, Path: launch.Service.Scope}, launch.Service.PID, launch.Service.Start)
	if err != nil {
		t.Fatalf("service ownership uncertain; retain references without unsafe cleanup: %v", err)
	}
	handoffWrite(t, filepath.Join(c.Dir, "service-ref.json"), serviceRef)
	handoffAwaitFile(t, filepath.Join(c.Dir, "service-ready.json"), 2*time.Second)
	var ready struct {
		Port   int              `json:"port"`
		Limits map[string]int64 `json:"limits"`
	}
	handoffRead(t, filepath.Join(c.Dir, "service-ready.json"), &ready)
	health := func() int64 {
		client := http.Client{Timeout: time.Second}
		resp, e := client.Get("http://127.0.0.1:" + strconv.Itoa(ready.Port) + "/health")
		if e != nil {
			t.Fatal(e)
		}
		defer resp.Body.Close()
		var v struct {
			Tick int64 `json:"tick"`
		}
		if e = json.NewDecoder(io.LimitReader(resp.Body, 4096)).Decode(&v); e != nil || resp.StatusCode != 200 {
			t.Fatalf("fake service unavailable: status%d %v", resp.StatusCode, e)
		}
		return v.Tick
	}
	first := health()
	if err = owned.Wait(ctx); err != nil {
		t.Fatalf("exact caller wait incomplete; retain references: %v", err)
	}
	elapsed := time.Since(start)
	observation, err := platform.InspectOwnedOperation(ctx, owned.Ref)
	if err != nil {
		t.Fatal(err)
	}
	if observation.Populated || observation.RootAlive || !observation.ExitKnown {
		t.Fatalf("caller not excluded: %+v", observation)
	}
	if c.Mode == "early-return-stdio" {
		var pipe struct {
			EOF     bool  `json:"eof"`
			Elapsed int64 `json:"elapsed_ms"`
		}
		handoffRead(t, filepath.Join(c.Dir, "stdio-return.json"), &pipe)
		if !pipe.EOF || pipe.Elapsed > 5000 || elapsed > 5*time.Second || observation.ExecMainStatus != 0 {
			t.Fatalf("caller return/EOF not proved: pipe=%+v elapsed=%s native=%+v", pipe, elapsed, observation)
		}
	} else if elapsed < 15*time.Second || observation.Result != "timeout" {
		t.Fatalf("natural native expiry not proved: elapsed=%s native=%+v", elapsed, observation)
	}
	afterExit := health()
	time.Sleep(3 * time.Second)
	later := health()
	if later <= afterExit || afterExit < first {
		t.Fatal("service heartbeat did not survive caller departure")
	}
	handoffWrite(t, filepath.Join(c.Dir, "survival.json"), map[string]any{"elapsed_ms": elapsed.Milliseconds(), "caller_terminal": observation, "launch": launch, "service_limits": ready.Limits, "tick_before_wait": first, "tick_after_exit": afterExit, "tick_three_seconds_later": later, "at": time.Now().UTC()})
	stop, err := platform.StopServiceScope(ctx, serviceRef)
	if err != nil || !stop.RootGone || !stop.TreeQuiescent {
		t.Fatalf("exact service cleanup incomplete; retain ref: %v %+v", err, stop)
	}
	serviceObservation, err := platform.InspectServiceScope(ctx, serviceRef)
	if err != nil || serviceObservation.RootAlive || serviceObservation.TreePopulated {
		t.Fatalf("service cleanup observation incomplete: %v %+v", err, serviceObservation)
	}
	if err = platform.ReleaseOwnedOperation(ctx, owned.Ref); err != nil {
		t.Fatalf("exact caller release incomplete: %v", err)
	}
	if err = process.RemoveScenarioRecord(home, c.Slug, c.Step); err != nil {
		t.Fatal(err)
	}
	handoffWrite(t, filepath.Join(c.Dir, "cleanup.json"), map[string]any{"service_stop": stop, "service_observation": serviceObservation, "caller_terminal": observation, "caller_released": true, "private_process_record_removed": true})
	t.Logf("PASS case=%s unit=%s service=%s caller_elapsed=%s", c.Mode, owned.Ref.Unit, c.Unit, elapsed)
}
