package runner

import (
	"fmt"
	"os"
	"strings"
)

var inheritedEnvDenylist = map[string]struct{}{
	"API_PORT":          {},
	"VROOLI_SCENARIO":   {},
	"VROOLI_PROCESS_ID": {},
	"VROOLI_STEP":       {},
	// Prevent nested Claude CLI sessions when agent-manager itself runs inside Claude Code.
	"CLAUDECODE": {},
	// Prevent parent agent tag from leaking into child agent processes.
	// BuildEnv sets this explicitly per-request.
	"CLAUDE_CODE_AGENT_TAG": {},
	// Same for the grok per-run tag.
	"GROK_AGENT_TAG": {},
	// Prevent a parent web-console/codex session's CODEX_HOME from leaking
	// into nested sandboxed agent runs. When agent-manager runs inside a
	// web-console session, CODEX_HOME points at ~/.vrooli/state/.../codex,
	// which the vrooli-aware sandbox profile mounts READ-ONLY — so codex
	// dies at startup with "Read-only file system (os error 30)" trying to
	// write its session/logs there. Stripping it lets codex fall back to
	// its default $HOME/.codex, which carries auth and is writable via the
	// home overlay.
	"CODEX_HOME": {},
}

// SanitizedBaseEnv returns os.Environ() filtered through inheritedEnvDenylist.
// Used by every codec's BuildEnv implementation as the base for the agent
// process's environment so parent-process tags and per-scenario state never
// leak into nested agent runs.
func SanitizedBaseEnv() []string {
	base := os.Environ()
	out := make([]string, 0, len(base))
	for _, entry := range base {
		if entry == "" {
			continue
		}
		key, _, found := strings.Cut(entry, "=")
		if !found || key == "" {
			continue
		}
		if _, blocked := inheritedEnvDenylist[key]; blocked {
			continue
		}
		out = append(out, entry)
	}
	return out
}

// AppendEnvMap appends a map of extra KEY=VALUE entries onto an env slice,
// skipping entries with empty keys. Used by codec BuildEnv implementations
// to merge per-request environment overrides on top of SanitizedBaseEnv.
func AppendEnvMap(env []string, extras map[string]string) []string {
	if len(extras) == 0 {
		return env
	}
	for key, value := range extras {
		if strings.TrimSpace(key) == "" {
			continue
		}
		env = append(env, fmt.Sprintf("%s=%s", key, value))
	}
	return env
}
