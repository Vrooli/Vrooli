package main

import (
	"context"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"sync"
	"time"

	"web-console/backends/claude"
	"web-console/internal/backend"
)

// DOC: docs/internal/MESSAGES-VIEW-PROJECTION-UX.md#maturity-levels-per-harness

// promptAnsweringEnv is the per-host switch for answering prompts from
// Messages (level 3): a comma-separated list of harnesses, e.g.
// "claude,opencode". Unset — the default — keeps every prompt read-only.
const promptAnsweringEnv = "WEB_CONSOLE_PROMPT_ANSWERING"

// promptAnsweringFile is the same switch as a file in the scenario's data
// directory, for hosts whose scenario environment cannot be set; the
// environment variable wins when both are present.
const promptAnsweringFile = "prompt-answering"

// promptAnswering is the host's policy for answering prompts from Messages.
type promptAnswering struct {
	harnesses     map[string]bool
	claudeVersion func() string
}

func newPromptAnswering(setting string, claudeVersion func() string) promptAnswering {
	harnesses := map[string]bool{}
	for _, name := range strings.Split(setting, ",") {
		if name = strings.ToLower(strings.TrimSpace(name)); name != "" {
			harnesses[name] = true
		}
	}
	return promptAnswering{harnesses: harnesses, claudeVersion: claudeVersion}
}

// decide reports the harness version it judged by and whether the prompt may
// be answered: Claude Code with keystrokes on a verified version, OpenCode
// permissions through its reply API. Everything else stays read-only.
func (p promptAnswering) decide(harness string, prompt backend.PendingPrompt) (string, bool) {
	readable := prompt.Kind != "unknown" && len(prompt.Options) > 0
	switch harness {
	case "claude":
		if !p.harnesses["claude"] || p.claudeVersion == nil {
			return "", false
		}
		version := p.claudeVersion()
		return version, readable && claude.AnswerVerified(version)
	case "opencode":
		return "", p.harnesses["opencode"] && readable && prompt.Kind == "permission"
	}
	return "", false
}

// promptAnsweringSetting reads the switch: the environment value, else the
// host file, else off.
func promptAnsweringSetting(env, hostFile string) string {
	if env = strings.TrimSpace(env); env != "" {
		return env
	}
	raw, err := os.ReadFile(hostFile)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(raw))
}

var claudeVersionPattern = regexp.MustCompile(`\d+\.\d+\.\d+`)

// localClaudeVersion reads the installed Claude Code version once. Local
// sessions run this host's claude, so its version decides whether answers are
// verified; an unreadable version keeps prompts read-only.
func localClaudeVersion() func() string {
	var once sync.Once
	var version string
	return func() string {
		once.Do(func() {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			if out, err := exec.CommandContext(ctx, "claude", "--version").Output(); err == nil {
				version = claudeVersionPattern.FindString(string(out))
			}
		})
		return version
	}
}
