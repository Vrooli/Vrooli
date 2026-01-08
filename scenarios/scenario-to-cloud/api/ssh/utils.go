package ssh

import (
	"context"
	"fmt"
	"log"
	"path"
	"strconv"
	"strings"
)

// QuoteSingle quotes a string for safe use in single-quoted shell contexts.
// This is the standard way to quote strings for SSH command arguments.
func QuoteSingle(s string) string {
	if s == "" {
		return "''"
	}
	return "'" + strings.ReplaceAll(s, "'", `'"'"'`) + "'"
}

// LocalSSHCommand builds a local SSH command string for display/logging.
func LocalSSHCommand(cfg Config, cmd string) string {
	args := []string{
		"ssh",
		"-o", "BatchMode=yes",
		"-o", "ConnectTimeout=5",
		"-p", strconv.Itoa(cfg.Port),
	}
	if strings.TrimSpace(cfg.KeyPath) != "" {
		args = append(args, "-i", cfg.KeyPath)
	}
	args = append(args, fmt.Sprintf("%s@%s", cfg.User, cfg.Host), "--", "bash", "-lc", QuoteSingle(cmd))
	return strings.Join(args, " ")
}

// LocalSCPCommand builds a local SCP command string for display/logging.
func LocalSCPCommand(cfg Config, localPath, remotePath string) string {
	args := []string{
		"scp",
		"-o", "BatchMode=yes",
		"-o", "ConnectTimeout=5",
		"-P", strconv.Itoa(cfg.Port),
	}
	if strings.TrimSpace(cfg.KeyPath) != "" {
		args = append(args, "-i", cfg.KeyPath)
	}
	args = append(args, localPath, fmt.Sprintf("%s@%s:%s", cfg.User, cfg.Host, remotePath))
	return strings.Join(args, " ")
}

// SafeRemoteJoin joins path elements for a remote (POSIX) path.
func SafeRemoteJoin(elem ...string) string {
	cleaned := make([]string, 0, len(elem))
	for _, e := range elem {
		e = strings.TrimSpace(e)
		if e == "" {
			continue
		}
		cleaned = append(cleaned, e)
	}
	if len(cleaned) == 0 {
		return ""
	}
	return path.Clean(path.Join(cleaned...))
}

// VrooliCommand wraps a vrooli command with PATH setup for SSH non-interactive sessions.
// SSH non-interactive commands don't source .bashrc, so we need to set up PATH explicitly.
func VrooliCommand(workdir, cmd string) string {
	pathSetup := `export PATH="$HOME/.local/bin:$HOME/bin:/usr/local/bin:$PATH"`
	return fmt.Sprintf("%s && cd %s && %s", pathSetup, QuoteSingle(workdir), cmd)
}

// FormatCommandForLog formats an SSH command for logging, redacting sensitive info.
func FormatCommandForLog(cfg Config, cmd string) string {
	parts := []string{
		"ssh",
		"-o", "BatchMode=yes",
		"-o", "ConnectTimeout=5",
		"-o", "ServerAliveInterval=5",
		"-o", "ServerAliveCountMax=1",
		"-o", "StrictHostKeyChecking=accept-new",
		"-p", strconv.Itoa(cfg.Port),
	}
	if cfg.KeyPath != "" {
		parts = append(parts, "-i", "<redacted>")
	}
	target := fmt.Sprintf("%s@%s", cfg.User, cfg.Host)
	parts = append(parts, target, "--", cmd)
	return strings.Join(parts, " ")
}

// RunWithOutput executes an SSH command and returns an error with output context if it fails.
// The returned error includes the last 50 lines of stdout for troubleshooting.
func RunWithOutput(ctx context.Context, runner Runner, cfg Config, cmd string, validateCmd func(string) error) error {
	if validateCmd != nil {
		if err := validateCmd(cmd); err != nil {
			return err
		}
	}
	log.Printf("ssh command: %s", FormatCommandForLog(cfg, cmd))
	res, err := runner.Run(ctx, cfg, cmd)
	if err != nil {
		return BuildError(err, res)
	}
	return nil
}

// ValidateTildeExpansion checks if a command contains a tilde inside single quotes,
// which would prevent home directory expansion. Returns an error if found.
func ValidateTildeExpansion(cmd string) error {
	if containsTildeInSingleQuotes(cmd) {
		return fmt.Errorf("invalid command: tilde inside single quotes prevents home expansion; use $HOME or an absolute path. command: %s", cmd)
	}
	return nil
}

// containsTildeInSingleQuotes checks if a tilde appears inside single quotes.
func containsTildeInSingleQuotes(command string) bool {
	inSingleQuote := false
	for _, r := range command {
		if r == '\'' {
			inSingleQuote = !inSingleQuote
			continue
		}
		if inSingleQuote && r == '~' {
			return true
		}
	}
	return false
}

// BuildError constructs an error with output context from an SSH result.
func BuildError(err error, res Result) error {
	var outputParts []string
	if res.Stderr != "" {
		outputParts = append(outputParts, "stderr: "+res.Stderr)
	}
	if res.Stdout != "" {
		lines := strings.Split(res.Stdout, "\n")
		if len(lines) > 50 {
			lines = lines[len(lines)-50:]
			outputParts = append(outputParts, "stdout (last 50 lines): "+strings.Join(lines, "\n"))
		} else {
			outputParts = append(outputParts, "stdout: "+res.Stdout)
		}
	}
	if len(outputParts) > 0 {
		return fmt.Errorf("%w\n%s", err, strings.Join(outputParts, "\n"))
	}
	return err
}
