package cliutil

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
)

// codingAgentAliases maps every name a shim may be installed under to the
// canonical runner name understood by [LaunchCodingAgent]. The keys are the
// binary names the agents themselves ship, because that is what a shim on PATH
// has to impersonate.
var codingAgentAliases = map[string]string{
	"claude":   "claude-code",
	"codex":    "codex",
	"grok":     "grok",
	"opencode": "opencode",
	"agy":      "antigravity",
}

// CodingAgentAliases returns the binary names a shim may be installed under,
// sorted so installers and tests see a stable order.
func CodingAgentAliases() []string {
	names := make([]string, 0, len(codingAgentAliases))
	for name := range codingAgentAliases {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// CodingAgentForAlias maps an installed shim name (with any executable
// extension already stripped) to its canonical runner name.
func CodingAgentForAlias(alias string) (string, bool) {
	runner, ok := codingAgentAliases[strings.ToLower(strings.TrimSpace(alias))]
	return runner, ok
}

// ShimAliasFromArgv0 reports the agent a shim was invoked as, given argv[0].
// It exists so one binary can serve as every agent's shim: the installer
// creates a link per alias and the alias is read back here.
func ShimAliasFromArgv0(argv0 string) (string, bool) {
	base := filepath.Base(strings.TrimSpace(argv0))
	if ext := filepath.Ext(base); ext != "" && isExecutableExtension(ext) {
		base = strings.TrimSuffix(base, ext)
	}
	return CodingAgentForAlias(base)
}

// ResolveAgentBinaryExcluding finds the real agent executable on PATH while
// refusing to resolve to self.
//
// This is the whole safety story for PATH shims. A shim installed as `codex`
// ahead of the real `codex` would otherwise resolve to itself and fork-bomb the
// machine, so candidates are compared by resolved path against the shim's own
// executable and skipped on a match. Comparing resolved paths (rather than
// directories) also covers the case where the shim is reachable from more than
// one PATH entry — a copy, a symlink farm, or a bind mount.
func ResolveAgentBinaryExcluding(binary, selfPath string) (string, error) {
	binary = strings.TrimSpace(binary)
	if binary == "" {
		return "", fmt.Errorf("no agent binary requested")
	}
	// An explicit path bypasses PATH entirely; honour it as-is.
	if strings.ContainsRune(binary, filepath.Separator) {
		if isExecutableFile(binary) {
			return binary, nil
		}
		return "", fmt.Errorf("resolve %q: not an executable file", binary)
	}

	self := resolvedPath(selfPath)
	var skipped bool
	for _, dir := range filepath.SplitList(os.Getenv("PATH")) {
		if strings.TrimSpace(dir) == "" {
			dir = "."
		}
		for _, candidate := range executableCandidates(filepath.Join(dir, binary)) {
			if !isExecutableFile(candidate) {
				continue
			}
			if self != "" && resolvedPath(candidate) == self {
				skipped = true
				continue
			}
			return candidate, nil
		}
	}
	if skipped {
		return "", fmt.Errorf("resolve %q: only the Vrooli shim was found on PATH; the real agent is not installed", binary)
	}
	return "", fmt.Errorf("resolve %q: not found on PATH", binary)
}

// CodingAgentBinary returns the executable name a runner ships as, so callers
// outside this package can resolve it themselves without duplicating the
// runner-to-binary table.
func CodingAgentBinary(agent string) (string, error) {
	_, binary, err := codingAgentSpec(agent)
	return binary, err
}

// ExecAgent hands this process over to the agent. On Unix it replaces the
// process image and never returns on success; on Windows, which has no such
// primitive, it spawns the agent and propagates its exit status.
//
// argv0 is the name the agent should see as its own, which is the agent's name
// rather than the shim's path.
func ExecAgent(path, argv0 string, args, environment []string) error {
	if execReplaceSupported {
		// Only returns on failure, in which case spawning is still correct.
		_ = execReplace(path, append([]string{argv0}, args...), environment)
	}
	command := exec.Command(path, args...)
	if len(command.Args) > 0 {
		command.Args[0] = argv0
	}
	command.Env = environment
	command.Stdin = os.Stdin
	command.Stdout = os.Stdout
	command.Stderr = os.Stderr
	return command.Run()
}

// resolvedPath returns path with symlinks resolved, or "" when it cannot be
// resolved. Callers treat "" as "cannot be self", which is the safe direction:
// a failed comparison must never make the resolver skip a real agent.
func resolvedPath(path string) string {
	if strings.TrimSpace(path) == "" {
		return ""
	}
	resolved, err := filepath.EvalSymlinks(path)
	if err != nil {
		return ""
	}
	absolute, err := filepath.Abs(resolved)
	if err != nil {
		return resolved
	}
	return absolute
}
