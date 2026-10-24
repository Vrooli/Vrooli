// Package capabilityprobe probes host capabilities without importing the
// control plane. It is deliberately standalone so the Bridge node agent can
// cross-compile it for every supported OS and architecture.
package capabilityprobe

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

type State string

const (
	Ready         State = "ready"
	Missing       State = "missing"
	NotApplicable State = "not_applicable"
	Unknown       State = "unknown"
)

type Definition struct {
	Capability string
	ID         string
	Label      string
	Command    string
	VersionArg []string
}

type Observation struct {
	Capability string    `json:"capability"`
	ID         string    `json:"id"`
	Label      string    `json:"label"`
	State      State     `json:"state"`
	Path       string    `json:"path,omitempty"`
	Version    string    `json:"version,omitempty"`
	ProbedAt   time.Time `json:"probed_at"`
	Detail     string    `json:"detail,omitempty"`
}

// AITools is generated from internal/tools/*/tool.json where capability is
// ai-cli. Keep this table sorted by ID; the drift test protects the contract.
var AITools = []Definition{
	{Capability: "ai-cli", ID: "agy", Label: "Antigravity", Command: "agy", VersionArg: []string{"--version"}},
	{Capability: "ai-cli", ID: "claude", Label: "Claude Code", Command: "claude", VersionArg: []string{"--version"}},
	{Capability: "ai-cli", ID: "codex", Label: "Codex", Command: "codex", VersionArg: []string{"--version"}},
	{Capability: "ai-cli", ID: "grok", Label: "Grok", Command: "grok", VersionArg: []string{"--version"}},
	{Capability: "ai-cli", ID: "opencode", Label: "OpenCode", Command: "opencode", VersionArg: []string{"--version"}},
}

type (
	LookPath   func(string) (string, error)
	RunVersion func(context.Context, string, []string) (string, error)
)

const (
	DefaultCommandTimeout = 3 * time.Second
	DefaultProbeTimeout   = 15 * time.Second
)

func Probe(ctx context.Context, definitions []Definition) []Observation {
	return ProbeWith(ctx, definitions, ManagedLookPath, runVersion, time.Now)
}

// ManagedPathEntries are the tool directories a long-lived node service must
// search, most specific first. It mirrors platform-go's DefaultPathEntries;
// this package deliberately carries no dependencies (see the package doc) so
// the Bridge agent can cross-compile it, which is why the list is duplicated
// rather than imported. Keep the two in step.
//
// ManagedLookPath resolves the TOOL against these. The child environment below
// needs them too, for a different reason: an interpreted tool such as a
// `#!/usr/bin/env node` trampoline resolves its INTERPRETER against the PATH
// the process inherits. A launchd daemon inherits `/usr/bin:/bin:/usr/sbin:/sbin`,
// which holds no Homebrew or nvm node, so probing `codex --version` there died
// with exit 127 while the identical command worked in a login shell.
func ManagedPathEntries(home string) []string {
	entries := []string{"/opt/homebrew/bin", "/usr/local/go/bin", "/usr/local/bin"}
	if home = strings.TrimSpace(home); home != "" {
		entries = append(entries,
			filepath.Join(home, ".cargo", "bin"),
			filepath.Join(home, "go", "bin"),
			filepath.Join(home, ".local", "bin"),
			filepath.Join(home, "bin"),
			filepath.Join(home, ".vrooli", "bin"),
		)
	}
	return entries
}

// managedEnviron returns the parent environment with the managed tool
// directories appended to PATH. Existing entries keep priority: this widens
// what a probe can resolve and never shadows an operator's own choice.
func managedEnviron() []string {
	env := os.Environ()
	home, err := os.UserHomeDir()
	if err != nil {
		home = ""
	}
	existing := os.Getenv("PATH")
	seen := make(map[string]struct{})
	merged := make([]string, 0, 12)
	add := func(dir string) {
		if dir == "" {
			return
		}
		if _, dup := seen[dir]; dup {
			return
		}
		seen[dir] = struct{}{}
		merged = append(merged, dir)
	}
	for _, dir := range strings.Split(existing, string(os.PathListSeparator)) {
		add(dir)
	}
	for _, dir := range ManagedPathEntries(home) {
		add(dir)
	}
	value := "PATH=" + strings.Join(merged, string(os.PathListSeparator))
	for i, item := range env {
		if strings.HasPrefix(item, "PATH=") {
			env[i] = value
			return env
		}
	}
	return append(env, value)
}

// ManagedLookPath mirrors the runtime PATH contract for long-lived node
// services. Native service managers commonly omit the interactive user's
// PATH, while Vrooli-owned CLIs are deliberately installed in user-owned
// directories. Prefer the real PATH and then resolve the managed locations;
// never scan arbitrary directories or execute a shell.
//
// It searches exactly ManagedPathEntries, the same list the probe puts on the
// child's PATH, so a tool this function can find is a tool whose interpreter
// the probe can also find. Two lists drifted apart is how a tool became
// resolvable while its runtime stayed invisible.
func ManagedLookPath(binary string) (string, error) {
	if path, err := exec.LookPath(binary); err == nil {
		return path, nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	for _, dir := range ManagedPathEntries(home) {
		candidate := filepath.Join(dir, binary)
		if info, statErr := os.Stat(candidate); statErr == nil && info.Mode().IsRegular() && info.Mode().Perm()&0o111 != 0 {
			return candidate, nil
		}
	}
	return "", exec.ErrNotFound
}

func ProbeWith(ctx context.Context, definitions []Definition, lookPath LookPath, version RunVersion, now func() time.Time) []Observation {
	if now == nil {
		now = time.Now
	}
	probeCtx, cancel := context.WithTimeout(ctx, DefaultProbeTimeout)
	defer cancel()
	observed := now().UTC()
	result := make([]Observation, 0, len(definitions))
	for _, definition := range definitions {
		item := Observation{Capability: definition.Capability, ID: definition.ID, Label: definition.Label, State: Unknown, ProbedAt: observed}
		path, err := lookPath(definition.Command)
		if err != nil || path == "" {
			item.State = Missing
			item.Detail = "command is not on PATH"
			result = append(result, item)
			continue
		}
		item.Path = path
		if version != nil {
			value, err := version(probeCtx, path, definition.VersionArg)
			if err != nil {
				item.State = Unknown
				item.Detail = versionFailureDetail(definition.Command, value, err)
			} else {
				item.State = Ready
				item.Version = strings.TrimSpace(value)
			}
		} else {
			item.State = Ready
		}
		result = append(result, item)
	}
	return result
}

// versionFailureDetail says WHY a version could not be read. The previous
// single string ("command was found but its version could not be read")
// collapsed three different operator situations into one unactionable line: a
// tool whose interpreter is missing, a tool that hung, and a tool that simply
// exited non-zero. Only the first is a host-environment problem, and nothing
// downstream could tell them apart.
//
// output is the command's combined output, which carries the shebang loader's
// message ("env: node: No such file or directory") that names the real cause.
func versionFailureDetail(command, output string, err error) string {
	output = strings.TrimSpace(output)
	if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) {
		return command + " did not answer --version within the probe budget"
	}
	if interpreter, ok := missingInterpreter(output); ok {
		return command + " is installed, but its " + interpreter +
			" runtime is not on this service's PATH (" + firstLine(output) + ")"
	}
	detail := command + " exited non-zero for --version"
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		detail = command + " exited " + strconv.Itoa(exitErr.ExitCode()) + " for --version"
	}
	if output != "" {
		detail += " (" + firstLine(output) + ")"
	}
	return detail
}

// missingInterpreter recognises the shebang loader's failure. A script such as
// `#!/usr/bin/env node` reports the INTERPRETER as not found, which reads as
// the tool being broken unless the distinction is made explicit.
func missingInterpreter(output string) (string, bool) {
	line := firstLine(output)
	rest, ok := strings.CutPrefix(line, "env: ")
	if !ok {
		return "", false
	}
	name, found := strings.CutSuffix(rest, ": No such file or directory")
	if !found {
		return "", false
	}
	if name = strings.TrimSpace(name); name == "" {
		return "", false
	}
	return name, true
}

func firstLine(value string) string {
	if index := strings.IndexAny(value, "\r\n"); index >= 0 {
		value = value[:index]
	}
	return strings.TrimSpace(value)
}

// runVersion bounds EVERY version command at DefaultCommandTimeout, not just
// the ones whose caller supplied a deadline-free context. ProbeWith always
// passes a context carrying the whole-probe deadline, so the previous
// `if !hasDeadline` guard meant the per-command bound never applied on the only
// path that uses it: one hanging tool could spend the entire probe budget and
// leave every tool after it unreadable.
func runVersion(ctx context.Context, path string, args []string) (string, error) {
	commandCtx, cancel := context.WithTimeout(ctx, DefaultCommandTimeout)
	defer cancel()
	command := exec.CommandContext(commandCtx, path, args...)
	command.Env = managedEnviron()
	return stringOutput(command.CombinedOutput())
}

// stringOutput exists to keep the version runner's return shape explicit.
func stringOutput(output []byte, err error) (string, error) { return string(output), err }
