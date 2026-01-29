package scenarios

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

func executeCommand(ctx context.Context, timeout time.Duration, name string, args ...string) ([]byte, error) {
	ctxWithTimeout, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	cmd := exec.CommandContext(ctxWithTimeout, name, args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("%s %s failed: %w (output: %s)",
			name, strings.Join(args, " "), err, strings.TrimSpace(string(output)))
	}
	return output, nil
}

func executeVrooliCommand(ctx context.Context, timeout time.Duration, args ...string) ([]byte, error) {
	return executeCommand(ctx, timeout, "vrooli", args...)
}
