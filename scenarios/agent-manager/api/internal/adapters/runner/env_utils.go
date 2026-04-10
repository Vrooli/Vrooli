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
	// buildEnv sets this explicitly per-request.
	"CLAUDE_CODE_AGENT_TAG": {},
}

func sanitizedBaseEnv() []string {
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

func appendEnvMap(env []string, extras map[string]string) []string {
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
