package agentscope

import (
	"os"
	"strings"

	"github.com/vrooli/cli-core/cliutil"
)

// SessionEvidence names why a caller is a coding-agent session, or returns ""
// for the operator. It combines the canonical caller detector
// (cliutil.DetectCallerKind: launcher identity, sandbox and per-harness
// markers) with the kernel's own record of an agent session scope, which a
// process cannot leave by clearing its environment. An operator's web-console
// pane runs in a tmux-spawn scope under the same slice and carries harness
// variables of the console itself; neither makes it an agent session.
//
// This is the in-process backstop for operator-only commands. It is not an
// authority boundary: a same-user process outside any agent scope that scrubs
// its environment passes.
func SessionEvidence(kind cliutil.CallerKind, cgroup string) string {
	switch kind {
	case cliutil.CallerKindVrooliAgent, cliutil.CallerKindExternalAgent, cliutil.CallerKindOverride:
		return "the caller detector classifies it as " + kind.String()
	}
	for _, line := range strings.Split(cgroup, "\n") {
		for _, segment := range strings.Split(line, "/") {
			if IsSessionScope(strings.TrimSpace(segment)) {
				return "its process runs in the agent session scope " + strings.TrimSpace(segment)
			}
		}
	}
	return ""
}

// SelfCgroup is this process's cgroup membership, or "" where the kernel does
// not expose one.
func SelfCgroup() string {
	data, err := os.ReadFile("/proc/self/cgroup")
	if err != nil {
		return ""
	}
	return string(data)
}

// CallerSessionEvidence is SessionEvidence for this process.
func CallerSessionEvidence() string {
	return SessionEvidence(cliutil.DetectCallerKind(), SelfCgroup())
}
