package codecs

import (
	"context"
	"os/exec"
	"regexp"
	"strings"
	"testing"
	"time"

	"agent-manager/internal/adapters/runner"
	"agent-manager/internal/domain"
)

// TestInstalledCLIHelpAgreesWithDeclaredCapabilities is a cheap drift guard:
// no runner is launched and a missing optional binary is explicitly skipped.
func TestInstalledCLIHelpAgreesWithDeclaredCapabilities(t *testing.T) {
	cases := []struct {
		name, binary string
		help         []string
		codec        Codec
		flags        []string
		valueDomain  bool
	}{
		{"claude", ClaudeCLICommand, []string{"--help"}, NewClaudeForTest(), []string{"--allowedTools", "--disallowedTools", "--effort"}, true},
		{"codex", CodexCLICommand, []string{"exec", "--help"}, NewCodexForTest(), []string{}, false},
		{"grok", GrokCLICommand, []string{"--help"}, NewGrokForTest(), []string{"--allow", "--deny", "--effort"}, true},
		{"opencode", OpenCodeCLICommand, []string{"run", "--help"}, NewOpenCodeForTest(), []string{"--variant"}, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			path, err := exec.LookPath(tc.binary)
			if err != nil {
				t.Skipf("%s absent: %v", tc.binary, err)
			}
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			output, err := exec.CommandContext(ctx, path, tc.help...).CombinedOutput()
			if err != nil {
				t.Fatalf("%s help: %v: %s", tc.binary, err, output)
			}
			help := string(output)
			for _, flag := range tc.flags {
				if !strings.Contains(help, flag) {
					t.Fatalf("%s declares/emits %s but installed help does not accept it", tc.binary, flag)
				}
			}
			if tc.valueDomain {
				values := publishedEffortValues(help)
				if len(values) == 0 {
					t.Fatalf("%s publishes an effort flag but no parseable value domain", tc.name)
				}
				for declared := range tc.codec.Capabilities().EffortMappings {
					if !values[declared] {
						t.Fatalf("%s declares effort %q but installed help omits it", tc.name, declared)
					}
				}
			}
			cfg := &domain.RunConfig{RunnerType: tc.codec.Type(), Effort: domain.EffortHigh, AllowedTools: []string{"read"}, DeniedTools: []string{"shell"}}
			if tc.codec.Type() == domain.RunnerTypeOpenCode {
				cfg.Model = "openai/gpt-5"
			}
			args := tc.codec.BuildArgs(tc.codec.NewState(), runner.ExecuteRequest{ResolvedConfig: cfg, WorkingDir: "/work", Prompt: "test"})
			caps := tc.codec.Capabilities()
			if caps.SupportsToolRestriction && tc.codec.Type() == domain.RunnerTypeGrok && (!containsArg(args, "--allow") || !containsArg(args, "--deny")) {
				t.Fatal("grok declares tool restrictions but does not emit both flags")
			}
			if caps.SupportsEffort && tc.codec.Type() == domain.RunnerTypeOpenCode && !containsArg(args, "--variant") {
				t.Fatal("opencode declares effort but does not emit --variant")
			}
		})
	}
}

var effortValuesPattern = regexp.MustCompile(`(?is)--effort.{0,240}?(?:possible values:\s*|\()([a-z]+(?:\s*,\s*[a-z]+)*)`)

func publishedEffortValues(help string) map[string]bool {
	match := effortValuesPattern.FindStringSubmatch(help)
	values := map[string]bool{}
	if len(match) != 2 {
		return values
	}
	for _, value := range strings.Split(match[1], ",") {
		values[strings.TrimSpace(strings.ToLower(value))] = true
	}
	return values
}

func containsArg(args []string, want string) bool {
	for _, arg := range args {
		if arg == want {
			return true
		}
	}
	return false
}
