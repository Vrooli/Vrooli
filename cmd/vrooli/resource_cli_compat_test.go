package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestMigratedResourceCLIsDelegateStandardCommandsToNativeControlPlane(t *testing.T) {
	root := projectRootForCLI(t)
	fakeBin := t.TempDir()
	logFile := filepath.Join(fakeBin, "vrooli.log")
	fakeVrooli := filepath.Join(fakeBin, "vrooli")
	script := "#!/usr/bin/env bash\nset -euo pipefail\nprintf '%s\\n' \"$*\" >\"" + logFile + "\"\nprintf 'delegated\\n'\n"
	if err := os.WriteFile(fakeVrooli, []byte(script), 0o755); err != nil {
		t.Fatalf("write fake vrooli: %v", err)
	}

	cases := []struct {
		resource string
		args     []string
		want     string
	}{
		{resource: "postgres", args: []string{"manage", "start"}, want: "resource start postgres"},
		{resource: "redis", args: []string{"logs"}, want: "resource logs redis"},
		{resource: "qdrant", args: []string{"status", "--fast"}, want: "resource status qdrant --fast"},
		{resource: "browserless", args: []string{"status"}, want: "resource status browserless"},
		{resource: "vault", args: []string{"manage", "restart"}, want: "resource restart vault"},
		{resource: "claude-code", args: []string{"logs"}, want: "resource logs claude-code"},
		{resource: "codex", args: []string{"manage", "stop"}, want: "resource stop codex"},
		{resource: "k6", args: []string{"status"}, want: "resource status k6"},
		{resource: "opencode", args: []string{"manage", "start"}, want: "resource start opencode"},
		{resource: "sqlite", args: []string{"manage", "install"}, want: "resource install sqlite"},
		{resource: "gemini", args: []string{"manage", "start"}, want: "resource start gemini"},
		{resource: "openrouter", args: []string{"status", "--fast"}, want: "resource status openrouter --fast"},
		{resource: "twilio", args: []string{"logs"}, want: "resource logs twilio"},
		{resource: "cloudflare-ai-gateway", args: []string{"manage", "restart"}, want: "resource restart cloudflare-ai-gateway"},
		{resource: "minio", args: []string{"logs"}, want: "resource logs minio"},
		{resource: "litellm", args: []string{"manage", "stop"}, want: "resource stop litellm"},
		{resource: "neo4j", args: []string{"status"}, want: "resource status neo4j"},
		{resource: "questdb", args: []string{"manage", "start"}, want: "resource start questdb"},
		{resource: "searxng", args: []string{"manage", "restart"}, want: "resource restart searxng"},
	}

	for _, tc := range cases {
		t.Run(tc.resource, func(t *testing.T) {
			_ = os.Remove(logFile)
			cliPath := filepath.Join(root, "resources", tc.resource, "cli.sh")
			cmd := exec.Command("bash", append([]string{cliPath}, tc.args...)...)
			cmd.Dir = root
			cmd.Env = append(os.Environ(),
				"APP_ROOT="+root,
				"VROOLI_CLI_BIN="+fakeVrooli,
			)
			output, err := cmd.CombinedOutput()
			if err != nil {
				t.Fatalf("%s %v failed: %v\n%s", tc.resource, tc.args, err, string(output))
			}
			data, err := os.ReadFile(logFile)
			if err != nil {
				t.Fatalf("read delegated command: %v", err)
			}
			got := strings.TrimSpace(string(data))
			if got != tc.want {
				t.Fatalf("delegated command = %q, want %q", got, tc.want)
			}
		})
	}
}
