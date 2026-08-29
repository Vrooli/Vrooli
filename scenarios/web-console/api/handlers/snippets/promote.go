package snippets

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"regexp"
	"strings"
)

var (
	ErrPromptManagerUnavailable = errors.New("prompt-manager is not available on this host")
	skillIdentifierPattern      = regexp.MustCompile(`\[([^\]]+)\]\s+in\s+[^/]+/`)
)

// CommandRunner is the narrow process seam used by snippet promotion.
type CommandRunner interface {
	Run(context.Context, string, []string, string) (string, error)
}

type execCommandRunner struct{}

func (execCommandRunner) Run(ctx context.Context, name string, args []string, stdin string) (string, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Stdin = strings.NewReader(stdin)
	output, err := cmd.CombinedOutput()
	return string(output), err
}

// CommandFailure preserves the governed CLI's own diagnostic for the client.
type CommandFailure struct {
	Output string
	Err    error
}

func (e *CommandFailure) Error() string {
	if message := strings.TrimSpace(e.Output); message != "" {
		return message
	}
	return e.Err.Error()
}

func (e *CommandFailure) Unwrap() error { return e.Err }

func promoteSnippet(ctx context.Context, runner CommandRunner, name, body string) (string, error) {
	output, err := runner.Run(ctx, "prompt-manager", []string{
		"skill", "create", name,
		"--folder=local",
		"--description=Promoted from a Web Console snippet",
	}, strings.TrimRight(body, "\r\n")+"\n")
	if err != nil {
		var execError *exec.Error
		if errors.Is(err, exec.ErrNotFound) || errors.As(err, &execError) {
			return "", ErrPromptManagerUnavailable
		}
		return "", &CommandFailure{Output: output, Err: err}
	}

	match := skillIdentifierPattern.FindStringSubmatch(output)
	if len(match) != 2 || strings.TrimSpace(match[1]) == "" {
		return "", fmt.Errorf("prompt-manager returned no skill identifier: %s", strings.TrimSpace(output))
	}
	return strings.TrimSpace(match[1]), nil
}
