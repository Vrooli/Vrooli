package codecs

import (
	"context"
	"os/exec"
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
	}{
		{"claude", ClaudeCLICommand, []string{"--help"}, NewClaudeForTest(), []string{"--allowedTools", "--disallowedTools", "--effort"}},
		{"codex", CodexCLICommand, []string{"exec", "--help"}, NewCodexForTest(), []string{}},
		{"grok", GrokCLICommand, []string{"--help"}, NewGrokForTest(), []string{"--allow", "--deny", "--effort"}},
		{"opencode", OpenCodeCLICommand, []string{"run", "--help"}, NewOpenCodeForTest(), []string{"--variant"}},
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
			cfg := &domain.RunConfig{RunnerType: tc.codec.Type(), Effort: domain.EffortHigh, AllowedTools: []string{"read"}, DeniedTools: []string{"shell"}}
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

func containsArg(args []string, want string) bool {
	for _, arg := range args {
		if arg == want {
			return true
		}
	}
	return false
}
