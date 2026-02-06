package shellutil

import (
	"fmt"
	"path"
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

// VrooliCommand wraps a vrooli command with PATH setup for SSH non-interactive sessions.
// SSH non-interactive commands don't source .bashrc, so we need to set up PATH explicitly.
func VrooliCommand(workdir, cmd string) string {
	pathSetup := `export PATH="$HOME/.local/bin:$HOME/bin:/usr/local/bin:$PATH"`
	return fmt.Sprintf("%s && cd %s && %s", pathSetup, QuoteSingle(workdir), cmd)
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
