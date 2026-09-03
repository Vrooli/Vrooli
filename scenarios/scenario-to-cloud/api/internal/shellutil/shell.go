package shellutil

import (
	"fmt"
	"path"
	"strings"
)

const remoteVrooliRelativePath = ".vrooli/bin/vrooli"

// QuoteSingle quotes a string for safe use in single-quoted shell contexts.
// This is the standard way to quote strings for SSH command arguments.
func QuoteSingle(s string) string {
	if s == "" {
		return "''"
	}
	return "'" + strings.ReplaceAll(s, "'", `'"'"'`) + "'"
}

// RemoteVrooliPath returns the deployment-local vrooli binary path for a VPS workdir.
func RemoteVrooliPath(workdir string) string {
	return SafeRemoteJoin(workdir, remoteVrooliRelativePath)
}

// QuotedRemoteVrooliPath returns the shell-quoted deployment-local vrooli binary path.
func QuotedRemoteVrooliPath(workdir string) string {
	return QuoteSingle(RemoteVrooliPath(workdir))
}

// VrooliCommand wraps a remote vrooli command with PATH setup for SSH non-interactive sessions.
// SSH non-interactive commands don't source shell profiles, so we make the deployment-local
// binary location explicit because the VPS install is a sealed artifact.
func VrooliCommand(workdir, cmd string) string {
	pathSetup := `export PATH="$HOME/.vrooli/bin:$HOME/.local/bin:$HOME/bin:/usr/local/bin:$PATH"`
	trimmed := strings.TrimSpace(cmd)
	switch {
	case trimmed == "vrooli":
		trimmed = ""
	case strings.HasPrefix(trimmed, "vrooli "):
		trimmed = strings.TrimSpace(strings.TrimPrefix(trimmed, "vrooli "))
	}

	base := fmt.Sprintf("%s && cd %s && %s", pathSetup, QuoteSingle(workdir), QuotedRemoteVrooliPath(workdir))
	if trimmed == "" {
		return base
	}
	return fmt.Sprintf("%s %s", base, trimmed)
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
