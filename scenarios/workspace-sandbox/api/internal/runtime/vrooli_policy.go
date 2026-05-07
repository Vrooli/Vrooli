package runtime

import (
	"path/filepath"
	"strings"
)

const (
	VrooliPolicyDestructiveMaintenanceBlocked = "destructive_vrooli_maintenance_blocked"
	vrooliPolicyDestructiveMessage            = "refusing host-wide Vrooli maintenance from an agent sandbox"
)

type VrooliCommandDecision struct {
	Allowed              bool
	Code                 string
	Reason               string
	SuggestedAlternative string
}

func EvaluateVrooliCommandPolicy(command string, args []string) VrooliCommandDecision {
	argv := append([]string{command}, args...)
	if containsDestructiveVrooliSequence(normalizeVrooliCommand(argv)) {
		return VrooliCommandDecision{
			Allowed:              false,
			Code:                 VrooliPolicyDestructiveMaintenanceBlocked,
			Reason:               vrooliPolicyDestructiveMessage,
			SuggestedAlternative: "use read-only status commands or ask the operator to run cleanup outside the agent session",
		}
	}
	return VrooliCommandDecision{Allowed: true}
}

func normalizeVrooliCommand(argv []string) []string {
	argv = trimEmpty(argv)
	if len(argv) == 0 {
		return nil
	}
	switch commandBase(argv[0]) {
	case "env":
		rest := skipEnvAssignments(argv[1:])
		if len(rest) > 0 {
			return normalizeVrooliCommand(rest)
		}
	case "sh", "bash":
		if len(argv) >= 3 && (argv[1] == "-c" || argv[1] == "-lc") {
			return shellWords(argv[2])
		}
	}
	return argv
}

func skipEnvAssignments(args []string) []string {
	for len(args) > 0 {
		if strings.HasPrefix(args[0], "-") {
			args = args[1:]
			continue
		}
		if key, _, ok := strings.Cut(args[0], "="); ok && key != "" && !strings.ContainsAny(key, "/ \t") {
			args = args[1:]
			continue
		}
		return args
	}
	return args
}

func containsDestructiveVrooliSequence(words []string) bool {
	for i := 0; i < len(words); i++ {
		if commandBase(words[i]) != "vrooli" {
			continue
		}
		rest := commandArgsUntilBoundary(words[i+1:])
		if destructiveVrooliArgs(rest) {
			return true
		}
	}
	return false
}

func destructiveVrooliArgs(args []string) bool {
	args = skipGlobalFlags(args)
	if len(args) == 0 {
		return false
	}
	switch args[0] {
	case "cleanup":
		if len(args) < 2 {
			return false
		}
		if args[1] == "locks" {
			return true
		}
		return args[1] == "orphans" && !hasFlag(args[2:], "--dry-run")
	case "orphans":
		return len(args) > 1 && args[1] == "kill" && !hasFlag(args[2:], "--dry-run")
	case "locks":
		return len(args) > 1 && args[1] == "clean"
	case "stop", "stop-all":
		return true
	case "scenario":
		return len(args) > 1 && (args[1] == "stop-all" || args[1] == "stop")
	case "resource":
		return len(args) > 1 && (args[1] == "stop-all" || args[1] == "stop")
	}
	return false
}

func commandArgsUntilBoundary(words []string) []string {
	out := make([]string, 0, len(words))
	for _, word := range words {
		switch word {
		case ";", "&&", "||", "|":
			return out
		default:
			out = append(out, word)
		}
	}
	return out
}

func skipGlobalFlags(args []string) []string {
	for len(args) > 0 && strings.HasPrefix(args[0], "-") {
		if args[0] == "--no-stale-check" || args[0] == "--json" || args[0] == "--verbose" || args[0] == "--no-color" {
			args = args[1:]
			continue
		}
		return args
	}
	return args
}

func hasFlag(args []string, flag string) bool {
	for _, arg := range args {
		if arg == flag {
			return true
		}
	}
	return false
}

func trimEmpty(args []string) []string {
	out := make([]string, 0, len(args))
	for _, arg := range args {
		if strings.TrimSpace(arg) != "" {
			out = append(out, arg)
		}
	}
	return out
}

func commandBase(command string) string {
	return filepath.Base(strings.TrimSpace(command))
}

func shellWords(command string) []string {
	var words []string
	var b strings.Builder
	var quote rune
	flush := func() {
		if b.Len() > 0 {
			words = append(words, b.String())
			b.Reset()
		}
	}
	for _, r := range command {
		if quote != 0 {
			if r == quote {
				quote = 0
				continue
			}
			b.WriteRune(r)
			continue
		}
		switch r {
		case '\'', '"':
			quote = r
		case ' ', '\t', '\n':
			flush()
		case ';', '|', '&':
			flush()
			words = append(words, string(r))
		default:
			b.WriteRune(r)
		}
	}
	flush()
	return words
}
