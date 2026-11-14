package scenariocli

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"

	repocontract "github.com/vrooli/repo-contract-go"
	"github.com/vrooli/vrooli/internal/cli/clipolicy"
	"github.com/vrooli/vrooli/internal/cli/commandtree"
	"github.com/vrooli/vrooli/internal/process"
	"github.com/vrooli/vrooli/internal/repocontractmeta"
)

type LogOptions struct {
	Follow      bool
	ForceFollow bool
	StepName    string
	Runtime     bool
	Lifecycle   bool
	Previous    bool
	Clean       bool
	Tail        int
}

var ErrScenarioLogsUsage = errors.New("scenario logs requires a scenario name")

func ParseLogsArgs(args []string) (string, LogOptions, error) {
	parsed, err := commandtree.ParseArgs("scenario logs", scenarioLogsHelpText(), scenarioLogsArgSchema(), args)
	if err != nil {
		if _, ok := err.(interface{ HelpText() string }); ok {
			return "", LogOptions{}, ErrScenarioLogsUsage
		}
		return "", LogOptions{}, fmt.Errorf("parse scenario logs arguments: %w", err)
	}
	tail, err := parseTailValue(parsed.FlagValue("--tail"))
	if err != nil {
		return "", LogOptions{}, fmt.Errorf("parse scenario logs tail: %w", err)
	}
	// A variant files its logs under its instance slug ("scenario@variant"),
	// so --instance resolves to the same address the reader looks up. Without
	// this the flag was rejected as an unknown option while its sibling
	// commands accepted it.
	name, err := resolveAddressArg("logs", parsed.Positionals[0], parsed.FlagValue("--instance"), "")
	if err != nil {
		return "", LogOptions{}, err
	}
	// Logs are read from this host's log directory, so a node address has no
	// reader. Refuse it by name instead of letting "node/scenario" reach the
	// path builder, which would reject it as an invalid selector.
	if strings.Contains(name, "/") {
		return "", LogOptions{}, clipolicy.UsageErrorf("scenario logs", "remote logs are not supported; run `vrooli scenario logs` on that node")
	}
	return name, LogOptions{
		Follow:      parsed.HasFlag("--follow"),
		ForceFollow: parsed.HasFlag("--force-follow"),
		StepName:    parsed.FlagValue("--step"),
		Runtime:     parsed.HasFlag("--runtime"),
		Lifecycle:   parsed.HasFlag("--lifecycle"),
		Previous:    parsed.HasFlag("--previous"),
		Clean:       parsed.HasFlag("--clean"),
		Tail:        tail,
	}, nil
}

func parseTailValue(value string) (int, error) {
	if value == "" {
		return 0, nil
	}
	tail, err := strconv.Atoi(value)
	if err != nil {
		return 0, fmt.Errorf("scenario logs --tail must be a positive integer")
	}
	if tail <= 0 {
		return 0, fmt.Errorf("scenario logs --tail must be greater than zero")
	}
	return tail, nil
}

func ShowLogsUsage(w io.Writer) error {
	home, _ := process.HomeDir()
	commandtree.WriteHelp(w, scenarioLogsHelpText())
	if home == "" {
		return ErrScenarioLogsUsage
	}
	logsDir, err := repocontract.RuntimeHomeEntryPath(home, repocontract.HomeKeyLogs)
	if err != nil {
		return err
	}
	logsRoot := filepath.Join(logsDir, repocontractmeta.ScenarioDir)
	entries, err := os.ReadDir(logsRoot)
	if err != nil {
		if os.IsNotExist(err) {
			return ErrScenarioLogsUsage
		}
		return err
	}
	names := make([]string, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() {
			names = append(names, entry.Name())
		}
	}
	slices.Sort(names)
	_, _ = fmt.Fprintln(w)
	_, _ = fmt.Fprintln(w, "Available scenarios with logs:")
	if len(names) == 0 {
		_, _ = fmt.Fprintln(w, "  (none found)")
	} else {
		for _, name := range names {
			_, _ = fmt.Fprintf(w, "  %s\n", name)
		}
	}
	return ErrScenarioLogsUsage
}

func scenarioLogsArgSchema() commandtree.ArgSchema {
	return commandtree.ArgSchema{
		Positionals: []commandtree.PositionalArg{{Name: "scenario name", Required: true}},
		Options: []commandtree.OptionArg{
			{Name: "--follow", Aliases: []string{"-f"}, Description: "Follow log output in real time"},
			{Name: "--force-follow", Description: "Stream even in non-interactive environments"},
			{Name: "--step", ValueName: "name", Description: "View a specific background step log"},
			{Name: "--runtime", Description: "View all background process logs"},
			{Name: "--lifecycle", Description: "View lifecycle log (default)"},
			{Name: "--previous", Description: "View the previous step log backup (.log.bak)"},
			{Name: "--tail", ValueName: "lines", Description: "Show the last N lines"},
			{Name: "--clean", Description: "Remove orphaned background logs"},
			instanceOption(),
		},
	}
}

func scenarioLogsHelpText() string {
	return commandtree.HelpText("", "vrooli scenario logs", "View logs for a scenario.", commandtree.Help{}, scenarioLogsArgSchema())
}
