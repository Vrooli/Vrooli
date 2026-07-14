package ssh

import (
	"fmt"
	"strings"
)

// FormatCommandForLog renders an SSH command for logging with the key path
// redacted so no key material or paths leak into logs.
func FormatCommandForLog(cfg Config, cmd string) string {
	redactedCfg := cfg
	redactedCfg.KeyPath = ""
	parts := append([]string{"ssh"}, buildSSHArgs(redactedCfg, DefaultRunOptions())...)
	if cfg.KeyPath != "" {
		parts = append(parts, "-i", "<redacted>")
	}
	target := fmt.Sprintf("%s@%s", cfg.User, cfg.Host)
	parts = append(parts, target, "--", cmd)
	return strings.Join(parts, " ")
}
