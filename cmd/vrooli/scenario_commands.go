package main

import (
	"fmt"
	"io"
	"strings"

	"github.com/vrooli/vrooli/internal/cli/scenariocli"
	"github.com/vrooli/vrooli/internal/lifecycle"
)

type (
	scenarioListPortOutput      = scenariocli.ListPortOutput
	scenarioListItemOutput      = scenariocli.ListItemOutput
	scenarioStatusItemOutput    = scenariocli.StatusItemOutput
	scenarioInfoOutput          = scenariocli.InfoOutput
	scenarioInfoScenarioData    = scenariocli.InfoScenarioData
	scenarioInfoRuntimeData     = scenariocli.InfoRuntimeData
	scenarioStatusSingleOutput  = scenariocli.StatusSingleOutput
	scenarioLifecycleItemOutput = scenariocli.LifecycleItemOutput
)

func parseScenarioNameAndJSON(command string, defaultJSON bool, args []string) (string, bool, error) {
	name, jsonFlag, err := parseOptionalScenarioNameAndJSON(command, defaultJSON, args)
	if err != nil {
		return "", false, err
	}
	if name == "" {
		return "", false, usageErrorf("scenario "+command, "scenario %s requires a scenario name", command)
	}
	return name, jsonFlag, nil
}

func parseOptionalScenarioNameAndJSON(command string, defaultJSON bool, args []string) (string, bool, error) {
	name := ""
	jsonFlag := defaultJSON
	for _, arg := range args {
		switch arg {
		case "--json":
			jsonFlag = true
		case "--help", "-h":
			return "", false, fmt.Errorf("usage requested")
		default:
			if strings.HasPrefix(arg, "-") {
				return "", false, unknownOptionError("scenario "+command, arg)
			}
			if name != "" {
				return "", false, usageErrorf("scenario "+command, "scenario %s accepts at most one scenario name", command)
			}
			name = arg
		}
	}
	return name, jsonFlag, nil
}

func parseScenarioStartArgs(defaultJSON bool, args []string) ([]string, lifecycle.StartOptions, bool, bool, error) {
	names := []string{}
	jsonFlag := defaultJSON
	openAfter := false
	opts := lifecycle.StartOptions{}

	for index := 0; index < len(args); index++ {
		arg := args[index]
		switch arg {
		case "--json":
			jsonFlag = true
		case "--open":
			openAfter = true
		case "--best-effort":
			opts.BestEffort = true
		case "--clean-stale":
			opts.CleanStale = true
		case "--path":
			if index+1 >= len(args) {
				return nil, lifecycle.StartOptions{}, false, false, usageErrorf("scenario start", "scenario start --path requires a value")
			}
			index++
			opts.CustomPath = args[index]
		case "--help", "-h":
			return nil, lifecycle.StartOptions{}, false, false, fmt.Errorf("usage requested")
		default:
			if strings.HasPrefix(arg, "-") {
				return nil, lifecycle.StartOptions{}, false, false, unknownOptionError("scenario start", arg)
			}
			names = append(names, arg)
		}
	}
	return names, opts, jsonFlag, openAfter, nil
}

func parseScenarioSingleStartArgs(command string, defaultJSON bool, args []string) (string, lifecycle.StartOptions, bool, bool, error) {
	names, opts, jsonFlag, openAfter, err := parseScenarioStartArgs(defaultJSON, args)
	if err != nil {
		return "", lifecycle.StartOptions{}, false, false, err
	}
	if len(names) == 0 {
		return "", lifecycle.StartOptions{}, false, false, usageErrorf("scenario "+command, "scenario %s requires a scenario name", command)
	}
	if len(names) > 1 {
		return "", lifecycle.StartOptions{}, false, false, usageErrorf("scenario "+command, "scenario %s accepts exactly one scenario name", command)
	}
	return names[0], opts, jsonFlag, openAfter, nil
}

func showScenarioHelp(w io.Writer) {
	_, _ = fmt.Fprintln(w, "Vrooli Scenario Commands")
	_, _ = fmt.Fprintln(w)
	_, _ = fmt.Fprintln(w, "Usage:")
	_, _ = fmt.Fprintln(w, "  vrooli scenario <subcommand> [options]")
	_, _ = fmt.Fprintln(w)
	renderCommandGroups(w, groupedScenarioCommands())
}
