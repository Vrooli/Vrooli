// Package safetyadmission describes whether a command may be admitted to an
// automated Git Control Tower validation run.
package safetyadmission

import "strings"

type Effect string

const (
	EffectRead          Effect = "pure_read"
	EffectArtifactWrite Effect = "artifact_write"
	EffectExternalWrite Effect = "external_write"
	EffectRepoMutation  Effect = "repository_mutation"
	EffectUnknown       Effect = "unknown"
)

type Command struct {
	Owner  string   `json:"owner"`
	Argv   []string `json:"argv"`
	Effect Effect   `json:"effect"`
	Reason string   `json:"reason"`
}

type Report struct {
	SchemaVersion string    `json:"schema_version"`
	Admitted      bool      `json:"admitted"`
	Commands      []Command `json:"commands"`
	Findings      []string  `json:"findings"`
}

func Classify(argv []string) (Effect, string) {
	if len(argv) == 0 {
		return EffectUnknown, "empty command"
	}
	args := append([]string(nil), argv...)
	if args[0] == "git" {
		args = args[1:]
	}
	for len(args) > 0 && (args[0] == "-C" || args[0] == "--git-dir" || args[0] == "--work-tree") {
		if len(args) < 2 {
			return EffectUnknown, "git option has no value"
		}
		args = args[2:]
	}
	if len(args) == 0 {
		return EffectUnknown, "git command is missing"
	}
	switch args[0] {
	case "status", "diff", "log", "show", "cat-file", "rev-parse", "ls-files", "ls-tree", "describe":
		return EffectRead, "git inspection command"
	case "branch":
		for _, arg := range args[1:] {
			if strings.HasPrefix(arg, "-") && (strings.Contains(arg, "-d") || strings.Contains(arg, "-m") || strings.Contains(arg, "-c") || strings.Contains(arg, "-f")) {
				return EffectRepoMutation, "branch command contains a mutation flag"
			}
		}
		return EffectRead, "git branch inspection command"
	case "init", "add", "commit", "checkout", "switch", "reset", "restore", "clean", "merge", "rebase", "stash", "worktree", "mv", "rm", "update-index", "config":
		return EffectRepoMutation, "git command changes repository state"
	case "push", "pull", "fetch", "clone", "remote":
		return EffectExternalWrite, "git command crosses or changes an external repository boundary"
	default:
		return EffectUnknown, "unclassified command"
	}
}

func NewReport(commands []Command) Report {
	report := Report{SchemaVersion: "gct.safety-admission.v1", Admitted: true}
	for _, command := range commands {
		report.Commands = append(report.Commands, command)
		if command.Effect != EffectRead {
			report.Admitted = false
			report.Findings = append(report.Findings, command.Owner+": "+command.Reason)
		}
	}
	return report
}
