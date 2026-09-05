package platform

import (
	"errors"
	"fmt"
	"io"
	"math"
	"os/exec"
	"strconv"
	"strings"
)

// Session containment: every coding-agent session is born inside a ceiling.
//
// ContainedCommand builds a command whose process tree lives under
// Containment on the running platform, ContainSelf moves the calling process
// into such a ceiling before it replaces its image, and FreezeScope /
// ThawScope pause and resume a tree by its ScopeRef. Each platform body is
// selected by build tag; the vocabulary and the ScopeRef grammar are shared
// so a scope minted on one host is a string every other component can read.
//
// Platform evidence: Linux is host-verified (systemd scopes, cgroup v2). The
// macOS body (rlimit shim + process group) and the Windows body (Job Object
// quotas) are compile- and fixture-verified only.

// Scope kinds name the native tree primitive a ScopeRef points at.
const (
	ScopeKindCgroup       = "cgroup" // Linux: a cgroup v2 path under /sys/fs/cgroup
	ScopeKindProcessGroup = "pgid"   // macOS: a process group id
	ScopeKindJob          = "job"    // Windows: a Job Object, identified by its root pid
	ScopeKindNone         = "none"   // no containment primitive was applied
)

// Containment methods report how a tree was contained, for lease rows and
// readiness evidence.
const (
	MethodSystemdRun    = "systemd-run"    // Linux: systemd-run --user --scope
	MethodTransientUnit = "transient-unit" // Linux: StartTransientUnit adopting an existing pid
	MethodCgroupWrite   = "cgroup-write"   // Linux without systemd-run: a hand-made cgroup
	MethodRlimitShim    = "rlimit-shim"    // macOS: setrlimit through the rlimit-exec shim
	MethodJob           = "job"            // Windows: a Job Object with quotas
	MethodNone          = "none"
)

// ScopeRef identifies a contained process tree.
type ScopeRef struct {
	Name string
	Kind string
	Path string
	PID  int
}

// String renders the ref as "<kind>:<identity>"; ParseScopeRef reads it back.
func (r ScopeRef) String() string {
	switch r.Kind {
	case ScopeKindCgroup:
		return ScopeKindCgroup + ":" + r.Path
	case ScopeKindProcessGroup, ScopeKindJob:
		return r.Kind + ":" + strconv.Itoa(r.PID)
	default:
		return ScopeKindNone
	}
}

// ParseScopeRef reads a ScopeRef rendered by String.
func ParseScopeRef(value string) (ScopeRef, error) {
	value = strings.TrimSpace(value)
	if value == "" || value == ScopeKindNone {
		return ScopeRef{Kind: ScopeKindNone}, nil
	}
	kind, rest, ok := strings.Cut(value, ":")
	if !ok {
		return ScopeRef{}, fmt.Errorf("platform: scope ref %q has no kind", value)
	}
	switch kind {
	case ScopeKindCgroup:
		if !strings.HasPrefix(rest, "/") {
			return ScopeRef{}, fmt.Errorf("platform: cgroup scope %q is not an absolute path", rest)
		}
		return ScopeRef{Kind: kind, Path: rest}, nil
	case ScopeKindProcessGroup, ScopeKindJob:
		pid, err := strconv.Atoi(rest)
		if err != nil || pid <= 0 {
			return ScopeRef{}, fmt.Errorf("platform: %s scope %q is not a pid", kind, rest)
		}
		return ScopeRef{Kind: kind, PID: pid}, nil
	default:
		return ScopeRef{}, fmt.Errorf("platform: unknown scope kind %q", kind)
	}
}

// ContainedSpec describes a process to start inside a ceiling.
type ContainedSpec struct {
	Path   string
	Args   []string
	Env    []string
	Dir    string
	Stdin  io.Reader
	Stdout io.Writer
	Stderr io.Writer
	// Scope is the unit name of the tree ("vrooli-agent-<id>").
	Scope       string
	Containment Containment
}

// Contained is a command wrapped in its platform's containment primitive.
type Contained struct {
	Cmd     *exec.Cmd
	Method  string
	Scope   ScopeRef
	cleanup func()
	after   func(*Contained) error
}

// ContainedCommand builds the command for spec on this platform. Start it
// through Contained.Start so the post-start step (a cgroup write, a Job
// assignment) runs and Scope is filled.
func ContainedCommand(spec ContainedSpec) (*Contained, error) {
	if strings.TrimSpace(spec.Path) == "" {
		return nil, errors.New("platform: contained command needs a path")
	}
	if strings.TrimSpace(spec.Scope) == "" {
		return nil, errors.New("platform: contained command needs a scope name")
	}
	if err := spec.Containment.validate(spec.Scope); err != nil {
		return nil, err
	}
	return containedCommand(spec)
}

// Start starts the command and applies the containment step that must
// follow a start on this platform.
func (c *Contained) Start() error {
	if c == nil || c.Cmd == nil {
		return errors.New("platform: nil contained command")
	}
	if err := c.Cmd.Start(); err != nil {
		return err
	}
	if c.after != nil {
		return c.after(c)
	}
	return nil
}

// Release frees what containment allocated: an empty fallback cgroup, a Job
// handle. It never signals the tree.
func (c *Contained) Release() {
	if c != nil && c.cleanup != nil {
		c.cleanup()
		c.cleanup = nil
	}
}

// ContainSelf moves the calling process into a ceiling named scope. It is
// the exec-replace branch of a launcher: the limits survive the exec that
// follows, so the agent that replaces the launcher is born contained.
func ContainSelf(scope string, containment Containment) (ScopeRef, string, error) {
	if strings.TrimSpace(scope) == "" {
		return ScopeRef{Kind: ScopeKindNone}, MethodNone, errors.New("platform: contain self needs a scope name")
	}
	if err := containment.validate(scope); err != nil {
		return ScopeRef{Kind: ScopeKindNone}, MethodNone, err
	}
	return containSelf(scope, containment)
}

// FreezeScope pauses every task in the tree: cgroup.freeze on Linux,
// SIGSTOP to the process group on macOS, Job termination on Windows (where
// no pause primitive spans a tree; documented, fixture-verified). It refuses
// a Linux cgroup that is not under an agent or test slice so a supervisor or
// a resource can never be frozen through it.
func FreezeScope(ref ScopeRef) error { return freezeScope(ref) }

// ThawScope reverses FreezeScope.
func ThawScope(ref ScopeRef) error { return thawScope(ref) }

// ScopeFrozen reports whether the tree is currently paused.
func ScopeFrozen(ref ScopeRef) (bool, error) { return scopeFrozen(ref) }

// sizeSuffixes are systemd's binary size suffixes.
var sizeSuffixes = map[string]int64{"K": kibibyte, "M": kibibyte * kibibyte, "G": kibibyte * kibibyte * kibibyte, "T": kibibyte * kibibyte * kibibyte * kibibyte}

const (
	kibibyte      = 1 << 10
	percentWhole  = 100
	neutralWeight = 100
	niceMin       = -20
	niceMax       = 19
	// Nice scale below and above the neutral share (see the weight table).
	niceScaleBelowNeutral = 10.0
	niceScaleAboveNeutral = 2.5
)

// Task Scheduler priorities (0 most urgent, 7 default, 10 least).
const (
	taskPriorityUrgent  = 2
	taskPriorityHigh    = 4
	taskPriorityRaised  = 5
	taskPriorityDefault = 7
	taskPriorityLowered = 8
	taskPriorityLow     = 9
	// niceLoweredCeiling is the nice value above which a session is merely
	// lowered rather than put at the bottom of the queue.
	niceLoweredCeiling = 5
)

// parseMemoryCeiling reads systemd's memory syntax: a percentage ("50%") or
// an absolute size with an optional K/M/G/T suffix. It returns the value
// (percent or bytes) and whether it is a percentage.
func parseMemoryCeiling(value string) (int64, bool, error) {
	value = strings.TrimSpace(value)
	if value == "" || value == "infinity" {
		return 0, false, nil
	}
	if percent, ok := strings.CutSuffix(value, "%"); ok {
		n, err := strconv.ParseInt(strings.TrimSpace(percent), 10, 64)
		if err != nil || n <= 0 || n > percentWhole {
			return 0, true, fmt.Errorf("%q is not a percentage between 1 and 100", value)
		}
		return n, true, nil
	}
	multiplier := int64(1)
	number := value
	if last := value[len(value)-1]; last < '0' || last > '9' {
		factor, ok := sizeSuffixes[strings.ToUpper(string(last))]
		if !ok {
			return 0, false, fmt.Errorf("%q has an unknown size suffix", value)
		}
		multiplier = factor
		number = value[:len(value)-1]
	}
	n, err := strconv.ParseInt(strings.TrimSpace(number), 10, 64)
	if err != nil || n <= 0 {
		return 0, false, fmt.Errorf("%q is not a size", value)
	}
	return n * multiplier, false, nil
}

// memoryCeilingBytes resolves a ceiling to bytes against the host's physical
// memory when it is a percentage.
func memoryCeilingBytes(value string, physical int64) (int64, error) {
	n, percent, err := parseMemoryCeiling(value)
	if err != nil || n == 0 {
		return 0, err
	}
	if !percent {
		return n, nil
	}
	if physical <= 0 {
		return 0, fmt.Errorf("platform: cannot resolve %s without the host's memory size", value)
	}
	return physical * n / percentWhole, nil
}

// Weight mapping. systemd's CPUWeight is a share (100 neutral, 400 four
// times the share, 50 half). Off systemd the closest lever is a priority
// band, so the same weight renders through one monotone table:
//
//	CPUWeight   nice (launchd)   Task Scheduler priority
//	   50            10                  9
//	  100             0                  7
//	  200            -3                  5
//	  400            -5                  4
//	 1000            -8                  4
//	 4000           -13                  2
//
// Below the neutral weight nice = -10*log2(weight/100) (half the share is
// nice 10); above it nice = -2.5*log2(weight/100) (four times the share is
// nice -5), both rounded half away from zero and clamped to -20..19.
func niceForWeight(weight int) int {
	if weight <= 0 {
		return 0
	}
	ratio := math.Log2(float64(weight) / neutralWeight)
	scale := niceScaleBelowNeutral
	if ratio > 0 {
		scale = niceScaleAboveNeutral
	}
	return min(max(int(math.Round(-scale*ratio)), niceMin), niceMax)
}

// windowsPriorityForWeight maps the same weight onto Task Scheduler's 0..10
// priority (lower is more urgent; 7 is the scheduler's default).
func windowsPriorityForWeight(weight int) int {
	switch nice := niceForWeight(weight); {
	case nice <= -10:
		return taskPriorityUrgent
	case nice <= -5:
		return taskPriorityHigh
	case nice <= -3:
		return taskPriorityRaised
	case nice <= 0:
		return taskPriorityDefault
	case nice <= niceLoweredCeiling:
		return taskPriorityLowered
	default:
		return taskPriorityLow
	}
}
