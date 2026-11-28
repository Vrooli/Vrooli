// Package synthesisruntime owns behavior shared by Web Search's L1 and L2
// synthesis adapters. Prompt and citation contracts remain with each domain.
package synthesisruntime

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

const DefaultRole = "summarize.default"
const DefaultTimeout = 60 * time.Second

type Runner func(ctx context.Context, args []string, stdin string) ([]byte, error)

func NormalizeRole(role string) string {
	if role = strings.TrimSpace(role); role == "" {
		return DefaultRole
	}
	return role
}

// Run preserves the injectable runner seam and the fixed, shell-free Ollama
// command used by both synthesis domains.
func Run(ctx context.Context, runner Runner, args []string, stdin string) ([]byte, error) {
	if runner != nil {
		return runner(ctx, args, stdin)
	}
	// #nosec G204 -- fixed executable and argument vector; query/source text travels only on stdin, never through a shell.
	cmd := exec.CommandContext(ctx, "resource-ollama", args...)
	cmd.Stdin = strings.NewReader(stdin)
	out, err := cmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok && len(exitErr.Stderr) > 0 {
			return nil, fmt.Errorf("%w: %s", err, strings.TrimSpace(string(exitErr.Stderr)))
		}
		return nil, err
	}
	return out, nil
}

// FirstJSONObject returns the first balanced JSON object, ignoring braces in
// string values. A malformed first object does not fall through to later text.
func FirstJSONObject(s string) string {
	start := strings.Index(s, "{")
	if start == -1 {
		return ""
	}
	depth, inStr, escaped := 0, false, false
	for i := start; i < len(s); i++ {
		c := s[i]
		switch {
		case escaped:
			escaped = false
		case c == '\\' && inStr:
			escaped = true
		case c == '"':
			inStr = !inStr
		case inStr:
			// Structural characters inside strings do not affect the balance.
		case c == '{':
			depth++
		case c == '}':
			depth--
			if depth == 0 {
				return s[start : i+1]
			}
		}
	}
	return ""
}
