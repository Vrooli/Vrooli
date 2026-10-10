package nativeisolation

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"github.com/vrooli/api-core/effortauthority"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

// Runtime is configured only from an owner-installed manifest. No install,
// daemon reload, UID creation, key resolution or fallback happens here.
type Runtime struct {
	manifest     Manifest
	witness      *Witness
	unitOwner    *FixedUnitOwner
	mu           sync.Mutex
	busy         bool
	occupiedUnit string
	driver       driver
	qualify      func(context.Context) error
}
type child interface {
	PID() int
	Stdout() io.Reader
	Stderr() io.Reader
	Wait() error
}
type driver interface {
	Start(context.Context, Plan, io.Reader) (child, error)
	Show(context.Context, string) (map[string]string, error)
	Signal(context.Context, string, string) error
}

func NewRuntime(m Manifest) (*Runtime, error) {
	// Deep copy every map/slice/pointer: later caller mutations cannot amend the
	// accepted manifest or enable a previously disabled installation.
	raw, err := jsonMarshal(m)
	if err != nil {
		return nil, ErrRefused
	}
	var copy Manifest
	if jsonUnmarshal(raw, &copy) != nil {
		return nil, ErrRefused
	}
	if copy.Enabled && copy.Validate() != nil {
		return nil, ErrRefused
	}
	rt := &Runtime{manifest: copy, driver: &systemdDriver{manifest: copy}}
	rt.qualify = rt.qualifyHost
	return rt, nil
}

// NewRuntimeWithWitness requires an explicitly captured continuing witness.
// Startup never launches a probe or recaptures a dead/restarted witness.
func NewRuntimeWithWitness(m Manifest, w *Witness) (*Runtime, error) {
	if w == nil || w.ValidateManifest(m) != nil {
		return nil, ErrRefused
	}
	rt, e := NewRuntime(m)
	if e != nil {
		return nil, e
	}
	rt.witness = w
	return rt, nil
}
func (rt *Runtime) Enabled() bool { return rt != nil && rt.manifest.Enabled }

// Ready distinguishes a planning manifest from captured installation authority.
func (rt *Runtime) Ready() bool {
	return rt.Enabled() && rt.witness != nil && rt.unitOwner != nil && rt.unitOwner.CheckBinding(rt.manifest, rt.witness) == nil && rt.witness.ValidateManifest(rt.manifest) == nil
}

// CheckReady is a bounded caller-side read gate before reservation. It creates
// no manager/unit state and always consumes the exact continuing witness.
func (rt *Runtime) CheckReady(ctx context.Context) error {
	if !rt.Ready() {
		return ErrRefused
	}
	if rt.unitOwner.current(ctx) != nil {
		return ErrRefused
	}
	return rt.qualify(ctx)
}

// CheckBinding performs no host reads or manager effects. Admission uses it
// before reserving a slot; a configured different commission cannot consume it.
func (rt *Runtime) CheckBinding(id string, binding Binding) error {
	if !rt.Enabled() || rt.manifest.Validate() != nil {
		return ErrRefused
	}
	expected, ok := rt.manifest.bindingForProfile(id, binding.ProfileDigest)
	if !ok || expected.PolicyDigest != binding.PolicyDigest || expected.ProfileDigest != binding.ProfileDigest || expected.Repository != binding.Repository || !expected.Deadline.Equal(binding.Deadline) {
		return ErrRefused
	}
	return nil
}

func (rt *Runtime) Plan(ctx context.Context, r Request) (Plan, error) {
	return rt.manifest.Plan(ctx, r, time.Now())
}
func checkExecutable(x Executable, native uint32) error {
	info, e := os.Lstat(x.Path)
	if e != nil || !info.Mode().IsRegular() || info.Mode().Perm()&0022 != 0 {
		return ErrRefused
	}
	b, e := os.ReadFile(x.Path)
	if e != nil {
		return ErrRefused
	}
	h := sha256.Sum256(b)
	if hex.EncodeToString(h[:]) != x.SHA256 {
		return ErrRefused
	}
	return trustedPath(x.Path, native)
}
func (rt *Runtime) qualifyHost(ctx context.Context) error {
	m := rt.manifest
	if !m.Enabled || m.Validate() != nil {
		return ErrRefused
	}
	if rt.witness != nil {
		if rt.witness.ValidateManifest(m) != nil || rt.witness.Qualify(ctx) != nil {
			return ErrRefused
		}
	} else {
		return ErrRefused
	}
	for _, x := range []Executable{m.SystemdRun, m.Systemctl} {
		if checkExecutable(x, m.NativeUID) != nil {
			return ErrRefused
		}
	}
	if trustedPath(filepath.Dir(m.WorkspaceSource), m.NativeUID) != nil || trustedTree(m.RootFS, m.NativeUID) != nil {
		return ErrRefused
	}
	image, e := treeDigest(m.RootFS)
	if e != nil || image != m.RootFSDigest {
		return ErrRefused
	}
	for path, hash := range m.NativeBinaries {
		if checkExecutable(Executable{filepath.Join(m.RootFS, path), hash}, m.NativeUID) != nil {
			return ErrRefused
		}
	}
	return nil
}
func (rt *Runtime) Start(ctx context.Context, r Request, input io.Reader) (*Process, error) {
	if !rt.Ready() {
		return nil, ErrRefused
	}
	plan, e := rt.Plan(ctx, r)
	if e != nil {
		return nil, e
	}
	if rt.qualify(ctx) != nil {
		return nil, ErrRefused
	}
	rt.mu.Lock()
	if rt.busy {
		rt.mu.Unlock()
		return nil, ErrRefused
	}
	rt.busy = true
	rt.occupiedUnit = plan.Unit
	rt.mu.Unlock()
	release := func() {
		rt.mu.Lock()
		if rt.occupiedUnit == plan.Unit {
			rt.busy = false
			rt.occupiedUnit = ""
		}
		rt.mu.Unlock()
	}
	// Resolve the actual accepted workspace, refusing symlinks/socket aliases.
	// A regular native file cannot bypass the distinct UID's protected-state DAC.
	if checkWorkspace(rt.manifest.WorkspaceSource, rt.manifest.Isolation.ProtectedPaths) != nil {
		release()
		return nil, ErrRefused
	}
	owned, ok := rt.driver.(*fixedUnitDriver)
	if !ok || owned.owner != rt.unitOwner || owned.prepare(ctx, plan) != nil {
		release()
		return nil, ErrRefused
	}
	old, err := rt.driver.Show(ctx, plan.Unit)
	if err != nil || old["LoadState"] != "not-found" {
		release()
		return nil, ErrRefused
	}
	c, err := rt.driver.Start(ctx, plan, input)
	if err != nil {
		// A manager transport error does not prove zero execution. Keep occupancy
		// and return the owned unit handle; Wait obtains terminal evidence.
		if c == nil {
			return &Process{rt: rt, unit: plan.Unit, done: make(chan struct{}), startErr: ErrUnknown}, nil
		}
	}
	p := &Process{rt: rt, unit: plan.Unit, child: c, done: make(chan struct{})}
	// Never identify systemd-run's wrapper as the native process. Until the
	// manager + kernel confirm native leader identity, PID remains zero and
	// durable process recording refuses it. This is a held unknown, not a PID.
	deadline := time.NewTimer(5 * time.Second)
	defer deadline.Stop()
	tick := time.NewTicker(20 * time.Millisecond)
	defer tick.Stop()
ready:
	for {
		show, e := rt.driver.Show(ctx, plan.Unit)
		if e == nil {
			pid, _ := strconv.Atoi(show["MainPID"])
			if pid > 0 && nativeLeader(pid, rt.manifest.NativeUID) {
				p.pid = pid
				break
			}
			if terminalState(show) {
				break
			}
		}
		select {
		case <-ctx.Done():
			p.Kill()
			break ready
		case <-deadline.C:
			p.Kill()
			break ready
		case <-tick.C:
		}
	}
	go p.await(ctx, release)
	return p, nil
}

// Terminal proves the exact retained unit has no live descendants. A vanished
// unit, manager fault or wrapper exit cannot settle a finite reservation.
func (rt *Runtime) Terminal(ctx context.Context, run string) error {
	if !rt.Ready() || !runID.MatchString(run) || rt.qualify(ctx) != nil {
		return ErrUnknown
	}
	state, e := rt.driver.Show(ctx, "vrooli-finite-"+run+".service")
	if e != nil || !terminalState(state) {
		return ErrUnknown
	}
	rt.mu.Lock()
	if rt.occupiedUnit == "vrooli-finite-"+run+".service" {
		rt.busy = false
		rt.occupiedUnit = ""
	}
	rt.mu.Unlock()
	return nil
}
func terminalState(s map[string]string) bool {
	return s["LoadState"] == "loaded" && (s["SubState"] == "exited" || s["SubState"] == "failed" || s["SubState"] == "dead") && s["MainPID"] == "0" && s["TasksCurrent"] == "0" && (s["ExecMainCode"] == "1" || s["ExecMainCode"] == "2" || s["ExecMainCode"] == "3") && (s["Result"] == "success" || s["Result"] == "exit-code" || s["Result"] == "signal" || s["Result"] == "timeout")
}

// Exit status is deliberately conservative: non-success or unavailable unit
// evidence remains unknown until an owner reconciles it. No automatic slot
// release, replacement launch or reattach follows from wrapper failure.
type Process struct {
	rt           *Runtime
	unit         string
	child        child
	pid          int
	done         chan struct{}
	err          error
	startErr     error
	once         sync.Once
	timed        atomic.Bool
	idleMu       sync.Mutex
	idle         *time.Timer
	idleDuration time.Duration
}

func (p *Process) Stdout() io.Reader {
	if p.child == nil {
		return strings.NewReader("")
	}
	return p.child.Stdout()
}
func (p *Process) Stderr() io.Reader {
	if p.child == nil {
		return strings.NewReader("")
	}
	return p.child.Stderr()
}
func (p *Process) PID() int { return p.pid }
func (p *Process) SetIdleTimeout(d time.Duration) {
	if d <= 0 {
		return
	}
	p.idleMu.Lock()
	defer p.idleMu.Unlock()
	p.idleDuration = d
	select {
	case <-p.done:
		return
	default:
	}
	p.idle = time.AfterFunc(d, func() { p.timed.Store(true); p.Kill() })
}
func (p *Process) ResetIdleTimer() {
	p.idleMu.Lock()
	defer p.idleMu.Unlock()
	if p.idle != nil {
		p.idle.Reset(p.idleDuration)
	}
}
func (p *Process) TimedOut() bool { return p.timed.Load() }
func (p *Process) signal(sig string) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = p.rt.driver.Signal(ctx, p.unit, sig)
}
func (p *Process) Kill() { p.signal("SIGKILL") }
func (p *Process) Signal(grace time.Duration) {
	p.signal("SIGTERM")
	go func() {
		timer := time.NewTimer(grace)
		defer timer.Stop()
		select {
		case <-p.done:
			return
		case <-timer.C:
			p.Kill()
		}
	}()
}
func (p *Process) Wait() error {
	if p.startErr != nil {
		return p.startErr
	}
	<-p.done
	return p.err
}
func (p *Process) await(ctx context.Context, release func()) {
	// Both cancellation and idle/stop address the entire owned service cgroup,
	// not the wrapper's process group. A failed kill never proves termination.
	cancelDone := make(chan struct{})
	go func() {
		select {
		case <-ctx.Done():
			p.Kill()
		case <-cancelDone:
		}
	}()
	var waitErr error
	if p.child != nil {
		waitErr = p.child.Wait()
	}
	close(cancelDone)
	check, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	s, e := p.rt.driver.Show(check, p.unit)
	if e == nil && terminalState(s) {
		p.err = waitErr
		release()
	} else {
		p.err = ErrUnknown
	}
	p.idleMu.Lock()
	if p.idle != nil {
		p.idle.Stop()
	}
	p.idleMu.Unlock()
	close(p.done)
}

type systemdDriver struct{ manifest Manifest }
type commandChild struct {
	cmd      *exec.Cmd
	out, err io.ReadCloser
}

func (c *commandChild) PID() int          { return c.cmd.Process.Pid }
func (c *commandChild) Stdout() io.Reader { return c.out }
func (c *commandChild) Stderr() io.Reader { return c.err }
func (c *commandChild) Wait() error       { return c.cmd.Wait() }
func (d *systemdDriver) Start(_ context.Context, p Plan, input io.Reader) (child, error) {
	// No CommandContext: killing a wrapper is not service termination. Runtime
	// handles cancellation through the exact system manager unit instead.
	cmd := exec.Command(p.Command, p.Args...)
	cmd.Env = p.Env
	cmd.Stdin = input
	out, e := cmd.StdoutPipe()
	if e != nil {
		return nil, e
	}
	err, e := cmd.StderrPipe()
	if e != nil {
		out.Close()
		return nil, e
	}
	if e = cmd.Start(); e != nil {
		out.Close()
		err.Close()
		return nil, e
	}
	return &commandChild{cmd, out, err}, nil
}
func (d *systemdDriver) command(ctx context.Context, args ...string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, d.manifest.Systemctl.Path, args...)
	cmd.Env = []string{"PATH=/usr/bin:/bin", "LANG=C"}
	return cmd.Output()
}
func (d *systemdDriver) Show(ctx context.Context, unit string) (map[string]string, error) {
	raw, e := d.command(ctx, "--system", "show", unit, "--property=LoadState,MainPID,SubState,TasksCurrent,ExecMainCode,Result", "--no-pager")
	if e != nil {
		return nil, e
	}
	return parseShow(raw)
}

func parseShow(raw []byte) (map[string]string, error) {
	allowed := map[string]bool{"LoadState": true, "MainPID": true, "SubState": true, "TasksCurrent": true, "ExecMainCode": true, "Result": true}
	seen := map[string]bool{}
	out := map[string]string{}
	for _, line := range strings.Split(strings.TrimSpace(string(raw)), "\n") {
		k, v, ok := strings.Cut(line, "=")
		if !ok || !allowed[k] || seen[k] {
			return nil, ErrUnknown
		}
		seen[k] = true
		out[k] = v
	}
	if len(out) != 6 {
		return nil, ErrUnknown
	}
	return out, nil
}
func (d *systemdDriver) Signal(ctx context.Context, unit, sig string) error {
	if sig != "SIGTERM" && sig != "SIGKILL" {
		return ErrRefused
	}
	_, e := d.command(ctx, "--system", "kill", "--kill-whom=all", "--signal="+sig, unit)
	return e
}
func jsonMarshal(m Manifest) ([]byte, error)    { return json.Marshal(m) }
func jsonUnmarshal(b []byte, m *Manifest) error { return json.Unmarshal(b, m) }

// NewRuntimeWithWitnessAndUnitOwner publishes only an exact immutable local
// owner capability. Merely supplying a witness never grants unit operations.
func NewRuntimeWithWitnessAndUnitOwner(m Manifest, w *Witness, o *FixedUnitOwner) (*Runtime, error) {
	if o == nil || o.CheckBinding(m, w) != nil {
		return nil, ErrRefused
	}
	rt, e := NewRuntimeWithWitness(m, w)
	if e != nil {
		return nil, e
	}
	rt.unitOwner = o
	rt.qualify = o.qualify
	rt.driver = &fixedUnitDriver{owner: o}
	return rt, nil
}

// CheckSerialEpisode consumes only current protected owner reads. An editable
// ParentRunID is never a substitute for the exact retained terminal handoff.
// Start/PrepareNative revalidate the same facts immediately before unit effects.
func (rt *Runtime) CheckSerialEpisode(ctx context.Context, b effortauthority.Binding, intent effortauthority.Intent, newRunID string) error {
	if !rt.Ready() || rt.unitOwner.current(ctx) != nil || b != rt.unitOwner.binding() || intent.Effect != "run.child" {
		return ErrRefused
	}
	slot, err := rt.unitOwner.authority.ReadReservation(ctx, b, intent.IdempotencyKey)
	if err != nil || !slot.NativeBound || slot.Terminal || slot.RunID != newRunID || effortauthority.Digest(slot.Intent) != effortauthority.Digest(intent) || rt.unitOwner.serialEpisode(ctx, slot) != nil {
		return ErrRefused
	}
	return rt.CheckBinding(b.PolicyID, Binding{PolicyDigest: b.PolicyDigest, ProfileDigest: intent.ProfileDigest, Repository: intent.Repository, Deadline: b.Deadline})
}
