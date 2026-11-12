// Package skillscripts runs the research-coordination skill's scripts
// (mission_check.py, packet.py) and reads its templates/limits.json. The desk
// never re-implements what these files own; it runs them and reads their
// JSON. Each script also works without the desk.
package skillscripts

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"research-desk/internal/home"
)

// Paths locates the skills folder.
type Paths struct {
	// Skills is <repo>/scenarios/research-desk/skills.
	Skills string
}

// Coordination is the research-coordination skill folder.
func (p Paths) Coordination() string { return filepath.Join(p.Skills, "research-coordination") }

// MissionCheck is mission_check.py.
func (p Paths) MissionCheck() string {
	return filepath.Join(p.Coordination(), "scripts", "mission_check.py")
}

// Packet is packet.py.
func (p Paths) Packet() string { return filepath.Join(p.Coordination(), "scripts", "packet.py") }

// Limits is templates/limits.json.
func (p Paths) Limits() string { return filepath.Join(p.Coordination(), "templates", "limits.json") }

// Result is one finished script run.
type Result struct {
	Stdout   []byte
	Stderr   []byte
	ExitCode int
}

// Runner runs one Python script. The seam lets tests run scripts without a
// subprocess and lets the API report an absent interpreter as a lens.
type Runner interface {
	Run(ctx context.Context, script string, args []string, stdin []byte) (Result, error)
}

// Python runs scripts with python3 under a timeout.
type Python struct {
	Interpreter string
	Timeout     time.Duration
}

// NewPython returns a Runner using python3 and a 30 s timeout.
func NewPython() Python { return Python{Interpreter: "python3", Timeout: 30 * time.Second} }

// Run executes the script. A non-zero exit is a Result, not an error; an error
// means the script could not run or did not finish.
func (p Python) Run(ctx context.Context, script string, args []string, stdin []byte) (Result, error) {
	if _, err := os.Stat(script); err != nil {
		return Result{}, fmt.Errorf("%s is missing; the research-coordination skill must be in %s: %w", filepath.Base(script), filepath.Dir(filepath.Dir(script)), err)
	}
	ctx, cancel := context.WithTimeout(ctx, p.Timeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, p.Interpreter, append([]string{"-I", script}, args...)...)
	cmd.Stdin = bytes.NewReader(stdin)
	var stdout, stderr bytes.Buffer
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	err := cmd.Run()
	result := Result{Stdout: stdout.Bytes(), Stderr: stderr.Bytes()}
	var exitErr *exec.ExitError
	switch {
	case err == nil:
		return result, nil
	case ctx.Err() != nil:
		return result, fmt.Errorf("%s did not finish within %s", filepath.Base(script), p.Timeout)
	case errors.As(err, &exitErr):
		result.ExitCode = exitErr.ExitCode()
		return result, nil
	default:
		return result, fmt.Errorf("run %s with %s (install python3 3.10+): %w", filepath.Base(script), p.Interpreter, err)
	}
}

// MissionChecker runs mission_check.py --json over a research home.
type MissionChecker struct {
	Paths  Paths
	Runner Runner
}

// MissionCheck returns the script's JSON list. Exit 1 (some topic has errors)
// still carries the full list; the errors are per topic in the output.
func (m MissionChecker) MissionCheck(ctx context.Context, researchDir string) ([]byte, string, error) {
	script := m.Paths.MissionCheck()
	result, err := m.Runner.Run(ctx, script, []string{researchDir, "--json"}, nil)
	if err != nil {
		return nil, script, err
	}
	if result.ExitCode != 0 && result.ExitCode != 1 {
		return nil, script, fmt.Errorf("mission_check.py exited %d: %s", result.ExitCode, firstLine(result.Stderr))
	}
	return result.Stdout, script, nil
}

// Limits is templates/limits.json: packet gates, launch defaults and the
// read-set ceilings the no-growth test enforces.
type Limits struct {
	TemplateVersion string `json:"template_version"`
	Gates           struct {
		CharterBytes int    `json:"charter_bytes"`
		StanceChars  int    `json:"stance_chars"`
		KernelChars  int    `json:"kernel_chars"`
		StanceEnding string `json:"stance_ending"`
	} `json:"gates"`
	Defaults struct {
		SupervisorModel  string `json:"supervisor_model"`
		SupervisorEffort string `json:"supervisor_effort"`
		OwnerModelTier   string `json:"owner_model_tier"`
	} `json:"defaults"`
	// Models lists the values packet.py accepts for each model field.
	Models struct {
		SupervisorModel  []string `json:"supervisor_model"`
		SupervisorEffort []string `json:"supervisor_effort"`
		OwnerModelTier   []string `json:"owner_model_tier"`
	} `json:"models"`
	Ceilings struct {
		SchedulerStartBytes int            `json:"scheduler_start_bytes"`
		SchedulerUnitBytes  int            `json:"scheduler_unit_bytes"`
		OwnerStartBytes     int            `json:"owner_start_bytes"`
		SkillBytes          map[string]int `json:"skill_bytes"`
		LongEffortBytes     int            `json:"long_effort_s2_s3_bytes"`
	} `json:"ceilings"`
	// ReadSets name what each role is told to read: "charter", "STATE.md",
	// "long_effort_s2_s3", or a path under the skills folder.
	ReadSets struct {
		SchedulerStart []string `json:"scheduler_start"`
		SchedulerUnit  []string `json:"scheduler_unit"`
		OwnerStart     []string `json:"owner_start"`
	} `json:"read_sets"`
	// Path is where the limits were read from.
	Path string `json:"-"`
}

// LoadLimits reads limits.json.
func LoadLimits(path string) (*Limits, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", path, err)
	}
	var limits Limits
	if err := json.Unmarshal(data, &limits); err != nil {
		return nil, fmt.Errorf("%s is not valid limits JSON: %w", path, err)
	}
	if limits.Gates.CharterBytes <= 0 || limits.Ceilings.SchedulerUnitBytes <= 0 {
		return nil, fmt.Errorf("%s has no charter_bytes gate or scheduler_unit_bytes ceiling", path)
	}
	limits.Path = path
	return &limits, nil
}

func firstLine(b []byte) string {
	line, _, _ := bytes.Cut(bytes.TrimSpace(b), []byte("\n"))
	if len(line) == 0 {
		return "no stderr"
	}
	return string(line)
}

// Ceilings reads limits.json's read sets and ceilings and measures each
// skill-text item they name (rulings E2: limits.json is the only list).
func (p Paths) Ceilings() (*home.Ceilings, error) {
	limits, err := LoadLimits(p.Limits())
	if err != nil {
		return nil, err
	}
	c := &home.Ceilings{
		CharterBytes: limits.Gates.CharterBytes,
		Roles: []home.RoleReads{
			{Key: "scheduler_start", Items: limits.ReadSets.SchedulerStart, CeilingBytes: int64(limits.Ceilings.SchedulerStartBytes)},
			{Key: "scheduler_unit", Items: limits.ReadSets.SchedulerUnit, CeilingBytes: int64(limits.Ceilings.SchedulerUnitBytes)},
			{Key: home.OwnerStartKey, Items: limits.ReadSets.OwnerStart, CeilingBytes: int64(limits.Ceilings.OwnerStartBytes)},
		},
		Fixed:      map[string]home.FileFact{},
		Unmeasured: map[string]string{},
		Source:     limits.Path,
	}
	for _, role := range c.Roles {
		for _, item := range role.Items {
			if home.CommissionItem(item) != "" {
				continue
			}
			if fact, err := p.Measure(item); err != nil {
				c.Unmeasured[item] = err.Error()
			} else {
				c.Fixed[item] = fact
			}
		}
	}
	return c, nil
}

// LongEffortItem is the read-set item for §2–§3 of
// docs/agent-system/LONG_EFFORT_DESIGN.md, bounded as limits.json's note says.
const (
	LongEffortItem = "long_effort_s2_s3"
	longEffortFrom = "## 2. Design rules"
	longEffortTo   = "## 4. Codex"
)

// LongEffortDesign is docs/agent-system/LONG_EFFORT_DESIGN.md in the repo
// that holds the skills folder.
func (p Paths) LongEffortDesign() string {
	return filepath.Join(p.Skills, "..", "..", "..", "docs", "agent-system", "LONG_EFFORT_DESIGN.md")
}

// Measure sizes one skill-text item: a path under the skills folder or the
// long-effort excerpt.
func (p Paths) Measure(item string) (home.FileFact, error) {
	if item == LongEffortItem {
		path := filepath.Clean(p.LongEffortDesign())
		data, err := os.ReadFile(path)
		if err != nil {
			return home.FileFact{}, fmt.Errorf("read %s: %w", path, err)
		}
		text := string(data)
		from, to := strings.Index(text, longEffortFrom), strings.Index(text, longEffortTo)
		if from < 0 || to < from {
			return home.FileFact{}, fmt.Errorf("%s no longer has %q before %q", path, longEffortFrom, longEffortTo)
		}
		info, err := os.Stat(path)
		if err != nil {
			return home.FileFact{}, err
		}
		return home.FileFact{Name: "LONG_EFFORT_DESIGN.md §2–§3", Path: path, Bytes: int64(to - from), Modified: info.ModTime()}, nil
	}
	path := filepath.Join(p.Skills, filepath.FromSlash(item))
	info, err := os.Stat(path)
	if err != nil {
		return home.FileFact{}, fmt.Errorf("%s is not in the skills folder: %w", item, err)
	}
	if !info.Mode().IsRegular() {
		return home.FileFact{}, fmt.Errorf("%s is not a file", path)
	}
	return home.FileFact{Name: item, Path: path, Bytes: info.Size(), Modified: info.ModTime()}, nil
}
