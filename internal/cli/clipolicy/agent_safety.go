package clipolicy

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/vrooli/vrooli/internal/agentcontext"
	"github.com/vrooli/vrooli/internal/vroolierr"
)

const (
	agentSafetyStopAll = "stop-all"
)

const (
	agentSafetyParameterA = 2
)

const (
	CodeDestructiveVrooliMaintenanceBlocked = "destructive_vrooli_maintenance_blocked"
	destructiveMaintenanceMessage           = "refusing host-wide Vrooli maintenance from an agent sandbox"
)

type CommandDecision struct {
	Allowed              bool
	Code                 string
	Reason               string
	SuggestedAlternative string
}

func AllowCommand() CommandDecision {
	return CommandDecision{Allowed: true}
}

func DenyDestructiveVrooliMaintenance() CommandDecision {
	return CommandDecision{
		Allowed:              false,
		Code:                 CodeDestructiveVrooliMaintenanceBlocked,
		Reason:               destructiveMaintenanceMessage,
		SuggestedAlternative: "use read-only status commands or ask the operator to run cleanup outside the agent session",
	}
}

func NewCommandPolicyError(decision CommandDecision) error {
	if decision.Allowed {
		return nil
	}
	hint := decision.SuggestedAlternative
	if hint == "" {
		hint = GeneralHelpHint
	}
	return &vroolierr.Error{
		Err:      fmt.Errorf("%s", decision.Reason),
		Category: ErrorCategoryEnvironment,
		Hint:     hint,
		Code:     decision.Code,
	}
}

func ClassifyAgentCommand(argv []string, env []string) CommandDecision {
	if !agentcontext.IsAgentControlled(env) {
		return AllowCommand()
	}
	return ClassifySandboxVrooliCommand(argv)
}

func ClassifySandboxVrooliCommand(argv []string) CommandDecision {
	normalized := normalizeCommand(argv)
	if len(normalized) == 0 {
		return AllowCommand()
	}
	if containsDestructiveVrooliSequence(normalized) {
		return DenyDestructiveVrooliMaintenance()
	}
	return AllowCommand()
}

func normalizeCommand(argv []string) []string {
	argv = trimEmpty(argv)
	if len(argv) == 0 {
		return nil
	}
	name := commandBase(argv[0])
	switch name {
	case "env":
		rest := skipEnvAssignments(argv[1:])
		if len(rest) > 0 {
			return normalizeCommand(rest)
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
		if len(args) < agentSafetyParameterA {
			return false
		}
		switch args[1] {
		case "locks":
			return true
		case "orphans":
			return !hasFlag(args[2:], "--dry-run")
		}
	case "orphans":
		return len(args) > 1 && args[1] == "kill" && !hasFlag(args[2:], "--dry-run")
	case "locks":
		return len(args) > 1 && args[1] == "clean"
	case "stop", agentSafetyStopAll:
		return true
	case "scenario":
		return len(args) > 1 && args[1] == agentSafetyStopAll
	case "resource":
		return len(args) > 1 && (args[1] == agentSafetyStopAll || args[1] == "stop")
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
		if b.Len() == 0 {
			return
		}
		words = append(words, b.String())
		b.Reset()
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
		case ';', '|':
			flush()
			words = append(words, string(r))
		case '&':
			flush()
			words = append(words, string(r))
		default:
			b.WriteRune(r)
		}
	}
	flush()
	return words
}
