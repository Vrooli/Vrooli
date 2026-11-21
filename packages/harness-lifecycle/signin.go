package agentinstall

import (
	"bytes"
	"context"
	"errors"
	"os/exec"
	"strings"
	"time"

	harnesscatalog "github.com/vrooli/cli-core/agentcatalog/harness"
)

// statusOutput bounds combined stdout/stderr and never becomes report text.
type statusOutput struct{ bytes.Buffer }

func (b *statusOutput) Write(p []byte) (int, error) {
	if len(p) > 4096-b.Len() {
		return 0, errors.New("native status output exceeded limit")
	}
	return b.Buffer.Write(p)
}

func inspectSignIn(ctx context.Context, executable string, probe *harnesscatalog.SignInProbe) string {
	if probe == nil || executable == "" {
		return "unknown"
	}
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, executable, probe.Args...)
	var output statusOutput
	cmd.Stdout = &output
	cmd.Stderr = &output
	err := cmd.Run()
	if ctx.Err() != nil {
		return "unknown"
	}
	// A signed-out command commonly exits nonzero. Only its exact declared
	// status text can establish that state; an exit code alone cannot.
	return classifySignIn(output.String(), err, probe)
}

func classifySignIn(output string, commandErr error, probe *harnesscatalog.SignInProbe) string {
	if probe == nil {
		return "unknown"
	}
	signedIn, signedOut := false, false
	for _, line := range strings.Split(output, "\n") {
		line = strings.TrimSpace(line)
		for _, prefix := range probe.SignedInPrefixes {
			if strings.HasPrefix(line, prefix) {
				signedIn = true
			}
		}
		for _, text := range probe.SignedOutLines {
			if line == text {
				signedOut = true
			}
		}
	}
	if signedOut && !signedIn {
		return "signed_out"
	}
	if signedIn && !signedOut && commandErr == nil {
		return "signed_in"
	}
	return "unknown"
}
