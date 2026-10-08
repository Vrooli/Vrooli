package metrics

import (
	"os"
	"regexp"
	"strings"
)

const (
	maxPositionalArgs   = 8
	maxPositionalArgLen = 64
	redactedPlaceholder = "<redacted>"
)

// secretKVPattern matches KEY=VALUE where KEY suggests a secret. Case-insensitive.
var secretKVPattern = regexp.MustCompile(`(?i)^(token|secret|key|password|credential)s?=`)

// RedactArgs returns a sanitized copy of args safe to persist.
//   - Anything after a literal "--" is dropped (passthrough payload may carry
//     secrets intended for a child process).
//   - Flag values are stripped: --token=abc becomes --token.
//   - Positional args are capped at maxPositionalArgs, each trimmed to
//     maxPositionalArgLen bytes.
//   - Positional args matching a secret KEY=VALUE pattern become <redacted>.
func RedactArgs(args []string) []string {
	if len(args) == 0 {
		return nil
	}
	out := make([]string, 0, len(args))
	positional := 0
	for _, arg := range args {
		if arg == "--" {
			break
		}
		if strings.HasPrefix(arg, "-") {
			if idx := strings.IndexByte(arg, '='); idx >= 0 {
				out = append(out, arg[:idx])
			} else {
				out = append(out, arg)
			}
			continue
		}
		if positional >= maxPositionalArgs {
			continue
		}
		positional++
		if secretKVPattern.MatchString(arg) {
			out = append(out, redactedPlaceholder)
			continue
		}
		if len(arg) > maxPositionalArgLen {
			out = append(out, arg[:maxPositionalArgLen])
			continue
		}
		out = append(out, arg)
	}
	return out
}

// ClassifyError returns a short category string for err. Empty when err is nil.
// Callers with richer category information (e.g. rootcli error categories) can
// pass a pre-computed string instead of calling this.
func ClassifyError(err error) string {
	if err == nil {
		return ""
	}
	return "error"
}

// Hostname returns the host name for stamping events, or "" if unavailable.
func Hostname() string {
	h, err := os.Hostname()
	if err != nil {
		return ""
	}
	return h
}
