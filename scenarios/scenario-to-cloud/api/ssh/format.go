package ssh

import (
	"fmt"
	"strings"
	"time"

	"scenario-to-cloud/internal/shellutil"
)

// LocalSSHCommand builds a local SSH command string for display/logging.
func LocalSSHCommand(cfg Config, cmd string) string {
	displayOpts := RunOptions{ConnectTimeout: 5 * time.Second}
	args := append([]string{"ssh"}, BuildSSHArgs(cfg, displayOpts)...)
	args = append(args, fmt.Sprintf("%s@%s", cfg.User, cfg.Host), "--", "bash", "-lc", shellutil.QuoteSingle(cmd))
	return strings.Join(args, " ")
}

// LocalSCPCommand builds a local SCP command string for display/logging.
func LocalSCPCommand(cfg Config, localPath, remotePath string) string {
	opts := SCPOptions{ConnectTimeout: 5 * time.Second, StrictHostKey: true}
	args := append([]string{"scp"}, BuildSCPArgs(cfg, opts)...)
	args = append(args, localPath, fmt.Sprintf("%s@%s:%s", cfg.User, cfg.Host, remotePath))
	return strings.Join(args, " ")
}

// FormatCommandForLog formats an SSH command for logging, redacting sensitive info.
func FormatCommandForLog(cfg Config, cmd string) string {
	// Build args with a cleared key path, then add the redacted key separately
	redactedCfg := cfg
	redactedCfg.KeyPath = ""
	parts := append([]string{"ssh"}, BuildSSHArgs(redactedCfg, DefaultRunOptions())...)
	if cfg.KeyPath != "" {
		parts = append(parts, "-i", "<redacted>")
	}
	target := fmt.Sprintf("%s@%s", cfg.User, cfg.Host)
	parts = append(parts, target, "--", cmd)
	return strings.Join(parts, " ")
}
