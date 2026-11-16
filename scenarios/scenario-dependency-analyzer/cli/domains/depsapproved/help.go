package depsapproved

import (
	"fmt"
	"os"
	"slices"
	"strings"

	"github.com/vrooli/cli-core/cliapp"
	"scenario-dependency-analyzer/cli/internal/support"
)

// commandDoc is what a legacy command's help says beyond its flags. The flags are
// read from the command's own flag set (support.DescribeFlags), so they cannot
// drift from what it accepts; the usage line and an example are written here, once,
// and the command's usage error reads the same text.
type commandDoc struct {
	Usage   string
	Example string
}

// commandDocs is keyed "<group> <command>". A test fails for any registered
// command of these groups that has no entry, no example, or no flag.
var commandDocs = map[string]commandDoc{
	"deps-approved check":            {"deps check one-version-per-shared-package [--packages <module,...>] [--include-indirect=false] [--json]", "deps check one-version-per-shared-package --packages modernc.org/sqlite,modernc.org/libc"},
	"deps-approved list":             {"deps approved list [--ecosystem npm|go] [--state approved] [--surface ui] [--use-case <tag>] [--json]", "deps approved list --ecosystem npm --state approved"},
	"deps-approved search":           {"deps approved search <query> [--ecosystem npm|go|pip] [--framework react] [--surface ui|api|cli|playwright-driver|tools/<package>] [--state approved|denied] [--limit 20] [--json]", "deps approved search markdown --ecosystem npm --surface ui"},
	"deps-approved explain":          {"deps approved explain <ecosystem>/<package> [--json]", "deps approved explain npm/react"},
	"deps-approved validate":         {"deps approved validate <scenario> [--policy-mode advisory|strict|review_gate] [--json]\n  deps approved validate --all [--policy-mode advisory|strict|review_gate] [--json]", "deps approved validate notification-hub --policy-mode strict"},
	"deps-approved triage":           {"deps approved triage [--section security|seeding|ranges|hotspots|expired] [--ecosystem npm|go] [--package <name>] [--limit 20] [--json]", "deps approved triage --section security --ecosystem npm"},
	"deps-approved findings":         {"deps approved findings [--scenario <name>] [--ecosystem npm|go] [--package <name>] [--severity WARNING|ERROR] [--class <finding-class>] [--json]", "deps approved findings --scenario notification-hub --severity ERROR"},
	"deps-approved usage":            {"deps approved usage <ecosystem>/<package> [--policy-mode advisory|strict|review_gate] [--json]", "deps approved usage npm/express"},
	"deps-approved upsert":           {"deps approved upsert --file <record.json> [--apply] [--json]", "deps approved upsert --file record.json --apply"},
	"deps-approved propose-records":  {"deps approved propose-records [--top-unrecorded 25] [--ecosystem npm|go] [--package <name>] [--json]", "deps approved propose-records --top-unrecorded 10 --ecosystem npm"},
	"deps-approved upsert-batch":     {"deps approved upsert-batch --file <proposals.json> [--apply|--dry-run] [--json]", "deps approved upsert-batch --file proposals.json --apply"},
	"deps-approved security-gaps":    {"deps approved security-gaps [--ecosystem npm|go] [--package <name>] [--scenario <name>] [--vulnerability <id>] [--minimum-severity high] [--limit 25] [--json]", "deps approved security-gaps --ecosystem npm --minimum-severity high"},
	"deps-approved approve-observed": {"deps approved approve-observed <ecosystem>/<package> [--from-findings] [--range-strategy observed|major_line|wildcard] [--apply] [--json]", "deps approved approve-observed npm/qrcode.react --from-findings --apply"},
	"deps-approved widen-range":      {"deps approved widen-range <ecosystem>/<package> --to-major-line [--apply] [--json]", "deps approved widen-range npm/react --to-major-line --apply"},
	"deps-approved approve":          {"deps approved approve <ecosystem>/<package> --rationale <text> [--range <range>] [--range-policy exact|major_line|minimum|dev_tooling] [--scenarios a,b] [--apply] [--json]", "deps approved approve npm/xlsx --range 0.20.3 --range-policy exact --rationale 'spreadsheet preview in file-viewer' --apply"},
	"deps-approved deny":             {"deps approved deny <ecosystem>/<package> --reason <text> [--replacement <text>] [--apply] [--json]", "deps approved deny npm/left-pad --reason 'unmaintained' --replacement 'native padStart' --apply"},
	"deps-approved deny-vulnerable":  {"deps approved deny-vulnerable <ecosystem>/<package> --vulnerability <id> [--affected-range <range>] [--fixed-range <range>] [--apply] [--json]", "deps approved deny-vulnerable npm/body-parser --vulnerability GHSA-27r3-jxmh-p828 --affected-range '<1.20.9' --fixed-range '>=1.20.9'"},
	"deps-approved remediate":        {"deps approved remediate <ecosystem>/<package> --vulnerability <id> [--json]", "deps approved remediate npm/body-parser --vulnerability GHSA-27r3-jxmh-p828"},
	"deps declare-runtime":           {"deps declare-runtime --scenario <consumer> --dependency <source> --description <text> --rationale <text> --degraded-behavior <text> [--apply --expected-sha256 <before-sha256>] [--json]", "deps declare-runtime --scenario portal --dependency search-hub --description 'optional search' --rationale 'runtime HTTP only' --degraded-behavior 'search box hidden'"},
	"deps resync":                    {"deps resync --scenario <name> --surface <ui|cli|api> [--apply] [--json]", "deps resync --scenario notification-hub --surface ui --apply"},
	"deps reconcile":                 {"deps reconcile (--scenario <name> | --module <dir> | --all | --all-modules) [--surface api|cli|ui] [--apply | --check] [--json]", "deps reconcile --scenario portal --surface api --check"},
	"deps vendor":                    {"deps vendor --module <repo-relative-go-module> [--preserve <vendor-subdir,...>] --apply [--json]", "deps vendor --module scenarios/portal/api --apply"},
}

// usageError is a command's usage failure, from the same text its help prints.
func usageError(key string) error {
	return fmt.Errorf("usage: %s %s", support.AppName, strings.ReplaceAll(commandDocs[key].Usage, "\n  ", "\n  "+support.AppName+" "))
}

// documentCommands gives every legacy (Run) command of a group the usage line
// from commandDocs and, when help for that command was asked for, an Options
// block read from its flag set and an example. Commands with an ArgSchema render
// their own help and are left alone.
func documentCommands(group string, commands []cliapp.Command) []cliapp.Command {
	for i, cmd := range commands {
		if cmd.RunCtx != nil || cmd.Run == nil {
			continue
		}
		doc, ok := commandDocs[group+" "+cmd.Name]
		if !ok {
			continue
		}
		commands[i].Usage = support.AppName + " " + strings.ReplaceAll(doc.Usage, "\n  ", "\n  "+support.AppName+" ")
		if helpRequestedFor(cmd.Name) {
			commands[i].HelpText = helpBody(cmd, doc)
		}
	}
	return commands
}

// helpBody is the Options block and example for one legacy command.
func helpBody(cmd cliapp.Command, doc commandDoc) string {
	var b strings.Builder
	b.WriteString("Options:\n")
	for _, flag := range support.DescribeFlags(cmd.Run) {
		synopsis := "--" + flag.Name
		if !flag.Bool {
			synopsis += " <value>"
		}
		line := fmt.Sprintf("  %-26s %s", synopsis, flag.Usage)
		if !flag.Bool && flag.Default != "" {
			line += fmt.Sprintf(" (default: %s)", flag.Default)
		}
		b.WriteString(line + "\n")
	}
	if doc.Example != "" {
		fmt.Fprintf(&b, "\nExample:\n  %s %s\n", support.AppName, doc.Example)
	}
	if cmd.HelpText != "" {
		b.WriteString("\n" + cmd.HelpText + "\n")
	}
	return strings.TrimRight(b.String(), "\n")
}

// helpRequestedFor reports whether this process was invoked to show help for the
// named command, so the flag probe runs only then.
func helpRequestedFor(name string) bool {
	args := os.Args[1:]
	return slices.Contains(args, name) && (slices.Contains(args, "--help") || slices.Contains(args, "-h"))
}
