package main

import (
	"fmt"
	"strings"
)

var logAddCommands = map[string]string{
	"decision": "decision-add",
	"finding":  "finding-add",
	"bug":      "bug-add",
	"record":   "record-add",
	"note":     "note-add",
}

func planManagerCommandHint(args []string) string {
	args = trimArgs(args)
	if len(args) == 0 {
		return ""
	}
	if len(args) >= 3 && args[0] == "log" && args[2] == "add" {
		if canonical, ok := logAddCommands[args[1]]; ok {
			return fmt.Sprintf("Did you mean:\n  plan-manager log %s <plan-or-execution> --title <title>", canonical)
		}
	}
	if len(args) >= 2 && args[0] == "exec" {
		if canonical, ok := movedExecLogCommand(args[1]); ok {
			return fmt.Sprintf("This command moved to the log ledger.\n\nUse:\n  plan-manager log %s <plan-or-execution> --title <title>", canonical)
		}
	}
	if len(args) >= 2 && args[0] == "log" && looksLikeBraceLogAdd(args[1]) {
		return "Brace expansion is shell syntax, not a plan-manager command. Use one of:\n" +
			"  plan-manager log decision-add <plan-or-execution> --title <title>\n" +
			"  plan-manager log finding-add <plan-or-execution> --title <title>\n" +
			"  plan-manager log bug-add <plan-or-execution> --title <title>\n" +
			"  plan-manager log record-add <plan-or-execution> --title <title>\n" +
			"  plan-manager log note-add <plan-or-execution> --title <title>"
	}
	return ""
}

func movedExecLogCommand(command string) (string, bool) {
	command = strings.TrimSpace(command)
	for _, canonical := range logAddCommands {
		if command == canonical {
			return canonical, true
		}
	}
	return "", false
}

func looksLikeBraceLogAdd(command string) bool {
	command = strings.TrimSpace(command)
	return strings.Contains(command, "{") &&
		strings.Contains(command, "}") &&
		strings.HasSuffix(command, "-add") &&
		strings.Contains(command, "decision") &&
		strings.Contains(command, "finding") &&
		strings.Contains(command, "bug") &&
		strings.Contains(command, "record") &&
		strings.Contains(command, "note")
}

func trimArgs(args []string) []string {
	out := make([]string, 0, len(args))
	for _, arg := range args {
		if trimmed := strings.TrimSpace(arg); trimmed != "" {
			out = append(out, trimmed)
		}
	}
	return out
}
