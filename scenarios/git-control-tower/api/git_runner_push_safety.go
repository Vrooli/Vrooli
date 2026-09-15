package main

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"git-control-tower/internal/pushsafety"
)

// safetyCommands is the only process boundary for outgoing-history inspection
// and isolated preparation. No shell, inherited index override or Git hooks.
type safetyCommands struct {
	path string
	cred *StoredCredential
}
type boundedGitOutput struct {
	bytes.Buffer
	limit int
}

func (b *boundedGitOutput) Write(p []byte) (int, error) {
	if b.Len()+len(p) > b.limit {
		return 0, errors.New("Git inspection output budget exceeded")
	}
	return b.Buffer.Write(p)
}

func (c safetyCommands) Run(ctx context.Context, dir string, input []byte, args ...string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, c.path, append([]string{"--no-optional-locks", "-C", dir, "-c", "core.hooksPath=/dev/null", "-c", "core.fsmonitor=false", "-c", "gc.auto=0", "-c", "maintenance.auto=false", "-c", "core.replaceRefs=false"}, args...)...)
	env, cleanup, e := gitCredentialEnv(c.cred)
	if e != nil {
		return nil, e
	}
	defer cleanup()
	for _, v := range env {
		k, _, _ := strings.Cut(v, "=")
		if strings.HasPrefix(k, "GIT_") && k != "GIT_SSH_COMMAND" && k != "GIT_ASKPASS" && k != "GIT_TERMINAL_PROMPT" && k != "GIT_USERNAME" && k != "GIT_PASSWORD" {
			continue
		}
		cmd.Env = append(cmd.Env, v)
	}
	cmd.Env = append(cmd.Env, "GIT_NO_REPLACE_OBJECTS=1", "GIT_LITERAL_PATHSPECS=1", "GIT_TERMINAL_PROMPT=0")
	cmd.Stdin = bytes.NewReader(input)
	out := &boundedGitOutput{limit: 64 * 1024 * 1024}
	stderr := &boundedGitOutput{limit: 64 * 1024}
	cmd.Stdout = out
	cmd.Stderr = stderr
	if e = cmd.Run(); e != nil {
		return nil, fmt.Errorf("Git safety operation %s failed: %w", args[0], e)
	} // do not expose credential-bearing URLs/stderr
	return out.Bytes(), nil
}

func (r *ExecGitRunner) InspectPushSafety(ctx context.Context, repo, remote, branch string, cred *StoredCredential) pushsafety.Report {
	ctx, cancel := context.WithTimeout(ctx, 60*time.Second)
	defer cancel()
	report := pushsafety.Inspect(ctx, safetyCommands{r.gitPath(), cred}, repo, remote, branch)
	pushsafety.InspectStaged(ctx, safetyCommands{r.gitPath(), cred}, repo, &report)
	return report
}

func (r *ExecGitRunner) PreparePushRecovery(ctx context.Context, repo, root string, report pushsafety.Report, cred *StoredCredential) (pushsafety.Artifact, error) {
	return pushsafety.Prepare(ctx, safetyCommands{r.gitPath(), cred}, pushsafety.DiskStore{Root: root}, repo, report)
}

func (r *ExecGitRunner) GetPushRecovery(ctx context.Context, repo, root, key string) (pushsafety.Artifact, error) {
	a, e := (pushsafety.DiskStore{Root: root}).LoadContext(ctx, repo, key)
	if e == nil && a.State == "prepared" {
		current, readErr := r.RevParse(ctx, repo, "HEAD")
		if readErr != nil || strings.TrimSpace(string(current)) != a.Head {
			a.State = "stale"
			a.Message = "The source commit changed after preparation. The retained bundles remain available; prepare a new preview before applying."
		}
	}
	return a, e
}
