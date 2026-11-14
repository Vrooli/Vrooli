package measures

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/vrooli/cli-core/cliapp"

	sharedmeasurepb "github.com/vrooli/vrooli/packages/proto/gen/go/measures/v1"
)

var windowValues = []string{"this_week", "last_7d", "last_30d", "this_month", "last_month", "this_quarter"}

// declaration is the part of a registry declaration the CLI renders and uses
// to resolve a measure id. The API's registry is the only list of measures.
type declaration struct {
	Name      string   `json:"name"`
	Domain    string   `json:"domain"`
	Intent    string   `json:"intent"`
	Questions []string `json:"questions"`
	Params    map[string]struct {
		Default string `json:"default"`
	} `json:"params"`
}

type measureResult struct {
	Value  string              `json:"value"`
	Fields []map[string]string `json:"fields"`
}

// Register exposes the API's measure registry: `list` prints every declared
// measure and `get` executes one. No per-measure verbs exist; a new measure
// appears here the moment the API declares it.
func Register() cliapp.SubcommandGroup {
	return cliapp.SubcommandGroup{Name: "measures", Description: "Declared measures: list them, get one by id", NeedsAPI: true, Subcommands: []cliapp.Command{
		listCommand(),
		getCommand(),
		cohortDefinitionCommand(),
		selectCohortCommand(),
		cohortReportCommand(),
	}}
}

func listDeclarations(ctx cliapp.OperationContext) ([]byte, []declaration, error) {
	body, err := ctx.Core().GetRoot("measures/declarations", nil)
	if err != nil {
		return nil, nil, err
	}
	var declarations []declaration
	if err := json.Unmarshal(body, &declarations); err != nil {
		return nil, nil, fmt.Errorf("decode measure declarations: %w", err)
	}
	return body, declarations, nil
}

func listCommand() cliapp.Command {
	return cliapp.Command{
		Name: "list", NeedsAPI: true, Description: "List every declared measure with its question and parameters",
		Usage:    "agent-manager measures list [--json]",
		HelpText: "Ids are domain.name; `measures get` also accepts the bare name with dashes, e.g. run-volume.",
		RunCtx: func(ctx cliapp.RunContext) error {
			body, declarations, err := listDeclarations(ctx)
			if err != nil {
				return err
			}
			if ctx.JSON() {
				_, err = ctx.Stdout().Write(append(body, '\n'))
				return err
			}
			for _, d := range declarations {
				params := make([]string, 0, len(d.Params))
				for name := range d.Params {
					params = append(params, name)
				}
				sort.Strings(params)
				fmt.Fprintf(ctx.Stdout(), "%-40s %s [%s]\n", d.Name, d.Intent, strings.Join(params, ","))
			}
			return nil
		},
	}
}

func getCommand() cliapp.Command {
	return cliapp.Command{
		Name: "get", NeedsAPI: true, Description: "Execute one declared measure",
		Usage:    "agent-manager measures get <id> [--window TOKEN] [--param name=value]... [--json]",
		HelpText: "Run `measures list` for ids and the parameters each accepts. --window is the common time-window parameter; every other parameter is --param name=value (repeatable), e.g. --param group_by=executable --param view=incurred for friction.token_attribution.",
		Args: cliapp.ArgSchema{
			Positionals: []cliapp.Positional{{Name: "id", Required: true, Description: "Measure id (domain.name) or bare name"}},
			Flags: []cliapp.Flag{
				{Name: "window", Description: "Analytical time window", Values: windowValues},
				{Name: "param", Repeated: true, Description: "Measure parameter as name=value"},
			},
		},
		RunCtx: func(ctx cliapp.RunContext) error {
			_, declarations, err := listDeclarations(ctx)
			if err != nil {
				return err
			}
			chosen, err := resolveDeclaration(ctx.Positional("id"), declarations)
			if err != nil {
				return err
			}
			id := chosen.Name
			params, err := buildParams(chosen, ctx.Flag("window"), ctx.FlagValues("param"))
			if err != nil {
				return err
			}
			body, err := ctx.Core().RequestRoot("POST", "measures/execute", nil, map[string]any{"measure": id, "params": params})
			if err != nil {
				return err
			}
			if ctx.JSON() {
				_, err = ctx.Stdout().Write(append(body, '\n'))
				return err
			}
			var result measureResult
			if err := json.Unmarshal(body, &result); err != nil {
				return err
			}
			renderResult(ctx, id, result)
			return nil
		},
	}
}

// buildParams merges --window and --param pairs, then fills the declaration's
// defaults: the registry validates parameters but does not apply them.
func buildParams(chosen declaration, window string, pairs []string) (map[string]string, error) {
	params := map[string]string{}
	if window != "" {
		params["window"] = window
	}
	for _, pair := range pairs {
		name, value, ok := strings.Cut(pair, "=")
		if !ok || name == "" {
			return nil, fmt.Errorf("--param %q must be name=value", pair)
		}
		params[name] = value
	}
	for name, param := range chosen.Params {
		if _, given := params[name]; !given && param.Default != "" {
			params[name] = param.Default
		}
	}
	return params, nil
}

// resolveDeclaration accepts a full id or the bare name (dashes for
// underscores) and refuses an ambiguous bare name.
func resolveDeclaration(input string, declarations []declaration) (declaration, error) {
	var matches []declaration
	for _, d := range declarations {
		_, bare, _ := strings.Cut(d.Name, ".")
		if d.Name == input {
			return d, nil
		}
		if strings.ReplaceAll(bare, "_", "-") == strings.ReplaceAll(input, "_", "-") {
			matches = append(matches, d)
		}
	}
	switch len(matches) {
	case 1:
		return matches[0], nil
	case 0:
		return declaration{}, fmt.Errorf("unknown measure %q; run `agent-manager measures list`", input)
	}
	names := make([]string, len(matches))
	for i, d := range matches {
		names[i] = d.Name
	}
	return declaration{}, fmt.Errorf("measure %q is ambiguous: %s", input, strings.Join(names, ", "))
}

func renderResult(ctx cliapp.RunContext, id string, result measureResult) {
	out := ctx.Stdout()
	if result.Value != "" {
		fmt.Fprintf(out, "%s: %s\n", id, result.Value)
	} else {
		fmt.Fprintf(out, "%s: %d row(s)\n", id, len(result.Fields))
	}
	for _, field := range result.Fields {
		keys := make([]string, 0, len(field))
		for key := range field {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		parts := make([]string, 0, len(keys))
		for _, key := range keys {
			parts = append(parts, key+"="+field[key])
		}
		fmt.Fprintln(out, "  "+strings.Join(parts, " "))
	}
}

func parseWindow(value string) (*sharedmeasurepb.TimeWindow, error) {
	if value == "" {
		return nil, nil
	}
	key := "TIME_WINDOW_TOKEN_" + strings.ReplaceAll(strings.ToUpper(value), "-", "_")
	number, ok := sharedmeasurepb.TimeWindowToken_value[key]
	if !ok || number == int32(sharedmeasurepb.TimeWindowToken_TIME_WINDOW_TOKEN_UNSPECIFIED) {
		return nil, fmt.Errorf("unsupported time window %q", value)
	}
	return &sharedmeasurepb.TimeWindow{Window: &sharedmeasurepb.TimeWindow_Token{Token: sharedmeasurepb.TimeWindowToken(number)}}, nil
}
