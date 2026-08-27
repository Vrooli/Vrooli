package scenariohandlers

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/vrooli/vrooli/internal/cli/rootcli"
	"github.com/vrooli/vrooli/internal/repocontractmeta"
	"github.com/vrooli/vrooli/internal/scenariostale"
)

const (
	staleCheckParameterA = 3
)

// staleCheckEnvVar lets tests and CI disable the warning without threading the
// --no-stale-check global flag through every caller.
//
//nolint:unused // reached only via generic scenariohandlers; unused linter can't trace through instantiations.
const staleCheckEnvVar = "VROOLI_NO_SCENARIO_STALE_CHECK"

// emitScenarioStaleWarning runs a best-effort stale check against the named
// scenario and writes a warning to stderr when the scenario's binary is stale
// relative to its Go source. It is silent for all other outcomes (fresh,
// rebuild-detected, initial baseline, no sources). Errors during the check are
// swallowed: staleness reporting must never block a command.
//
//nolint:unused // reached only via generic scenariohandlers; unused linter can't trace through instantiations.
func emitScenarioStaleWarning(stderr io.Writer, root, scenarioName string, globals rootcli.GlobalOptions) {
	if globals.NoStaleCheck {
		return
	}
	if strings.EqualFold(strings.TrimSpace(os.Getenv(staleCheckEnvVar)), "1") {
		return
	}
	scenarioName = strings.TrimSpace(scenarioName)
	if scenarioName == "" {
		return
	}
	if strings.ContainsAny(scenarioName, "/\\") {
		return
	}
	scenarioDir := filepath.Join(strings.TrimSpace(root), repocontractmeta.ScenarioDir, scenarioName)
	info, err := os.Stat(scenarioDir)
	if err != nil || !info.IsDir() {
		return
	}
	result, err := scenariostale.Check(scenarioDir, scenarioName, scenariostale.Options{})
	if err != nil {
		return
	}
	if result.Status != scenariostale.StatusStale {
		return
	}
	if stderr == nil {
		return
	}
	fmt.Fprint(stderr, scenariostale.FormatWarning(result))
}

// extractCompletenessScenario pulls the target scenario name out of the raw
// argument list forwarded to the scenario-completeness-scoring CLI. It returns
// an empty string when the subcommand does not operate on a named scenario.
//
// Supported shapes (after stripping recognised global flags):
//
//	score get <name>
//	score calculate <name>
//	score validation <name>
//	score history <name>
//	score trends <name>
//	score recommend <name>
//
//nolint:unused // reached only via generic scenariohandlers; unused linter can't trace through instantiations.
func extractCompletenessScenario(args []string) string {
	positional := stripCompletenessFlags(args)
	if len(positional) < staleCheckParameterA {
		return ""
	}
	if positional[0] != "score" {
		return ""
	}
	switch positional[1] {
	case "get", "calculate", "validation", "history", "trends", "recommend":
		name := strings.TrimSpace(positional[2])
		if strings.HasPrefix(name, "-") {
			return ""
		}
		return name
	}
	return ""
}

// stripCompletenessFlags removes flag tokens (and flag values) from args so the
// remaining positional list is easy to match. It only knows about the flags
// advertised by the completeness CLI's global help; unrecognised flags are
// kept as-is so we don't accidentally drop a positional that happens to
// resemble one.
//
//nolint:unused // reached only via generic scenariohandlers; unused linter can't trace through instantiations.
func stripCompletenessFlags(args []string) []string {
	valuedFlags := map[string]struct{}{
		"--api-base": {},
	}
	out := make([]string, 0, len(args))
	for i := 0; i < len(args); i++ {
		arg := args[i]
		if eq := strings.Index(arg, "="); eq > 0 && strings.HasPrefix(arg, "-") {
			continue
		}
		if _, ok := valuedFlags[arg]; ok {
			i++
			continue
		}
		if strings.HasPrefix(arg, "-") {
			continue
		}
		out = append(out, arg)
	}
	return out
}

// firstPositionalArg returns the first non-flag token in args, or an empty
// string if none exists. Used to infer the scenario name from pass-through
// subcommand arguments.
//
//nolint:unused // reached only via generic scenariohandlers; unused linter can't trace through instantiations.
func firstPositionalArg(args []string) string {
	for _, arg := range args {
		if arg == "" || strings.HasPrefix(arg, "-") {
			continue
		}
		return strings.TrimSpace(arg)
	}
	return ""
}
