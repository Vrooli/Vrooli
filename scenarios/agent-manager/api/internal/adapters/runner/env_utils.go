package runner

import (
	"fmt"
	"os"
	"strings"
)

var inheritedEnvAllowlist = map[string]struct{}{
	"PATH": {}, "HOME": {}, "USER": {}, "LOGNAME": {}, "SHELL": {},
	"LANG": {}, "LC_ALL": {}, "LC_CTYPE": {}, "TERM": {}, "TMPDIR": {},
	"HTTP_PROXY": {}, "HTTPS_PROXY": {}, "NO_PROXY": {}, "http_proxy": {}, "https_proxy": {}, "no_proxy": {},
	"ANTHROPIC_API_KEY": {}, "OPENAI_API_KEY": {}, "XAI_API_KEY": {},
}

// SanitizedBaseEnv returns only explicitly allowlisted inherited variables.
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
		if _, allowed := inheritedEnvAllowlist[key]; !allowed {
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
