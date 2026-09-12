package selection

import (
	"fmt"
	"net/url"
	"os"
	"strings"

	"vrooli-onboarding/cli/internal/support"

	"github.com/vrooli/cli-core/cliapp"
	selectionv1 "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-onboarding/v1/selection"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

const (
	ListScenariosProcedure        = "/vrooli.vrooli_onboarding.v1.selection.SelectionService/ListScenarios"
	GetCoreSetProcedure           = "/vrooli.vrooli_onboarding.v1.selection.SelectionService/GetCoreSet"
	GetRecommendationProcedure    = "/vrooli.vrooli_onboarding.v1.selection.SelectionService/GetRecommendation"
	AcceptRecommendationProcedure = "/vrooli.vrooli_onboarding.v1.selection.SelectionService/AcceptRecommendation"
	GetClosureProcedure           = "/vrooli.vrooli_onboarding.v1.selection.SelectionService/GetClosure"
	GetUnionProcedure             = "/vrooli.vrooli_onboarding.v1.selection.SelectionService/GetUnion"
	CreateHandoffProcedure        = "/vrooli.vrooli_onboarding.v1.selection.SelectionService/CreateHandoff"
	GetHandoffProcedure           = "/vrooli.vrooli_onboarding.v1.selection.SelectionService/GetHandoff"
)

func Request(core *cliapp.ScenarioApp, procedure string, message, response proto.Message) error {
	return support.RequestProto(core, procedure, message, response, "selection")
}

func printJSON(message proto.Message) error {
	return support.PrintProto(os.Stdout, message, "selection")
}

func targetFlags(name string, args []string) (*string, *bool, error) {
	fs := support.NewFlagSet(name)
	target := fs.String("target", "local", "Onboarding target scenario")
	jsonOutput := fs.Bool("json", false, "Render the typed response as JSON")
	if err := support.ParseFlags(fs, args); err != nil {
		return nil, nil, err
	}
	if strings.TrimSpace(*target) == "" {
		return nil, nil, fmt.Errorf("--target is required")
	}
	return target, jsonOutput, nil
}

func List(core *cliapp.ScenarioApp, args []string) error {
	target, jsonOutput, err := targetFlags("scenarios list", args)
	if err != nil {
		return err
	}
	response := &selectionv1.ListScenariosResponse{}
	if err := Request(core, ListScenariosProcedure, &selectionv1.ListScenariosRequest{Target: *target}, response); err != nil {
		return err
	}
	if *jsonOutput {
		return printJSON(response)
	}
	for _, scenario := range response.GetScenarios() {
		_, _ = fmt.Fprintf(os.Stdout, "%s\n", scenario.GetName())
	}
	return nil
}

func Closure(core *cliapp.ScenarioApp, args []string) error {
	target, jsonOutput, err := targetFlags("closure", args)
	if err != nil {
		return err
	}
	response := &selectionv1.GetClosureResponse{}
	if err := Request(core, GetClosureProcedure, &selectionv1.GetClosureRequest{Target: *target}, response); err != nil {
		return err
	}
	if *jsonOutput {
		return printJSON(response)
	}
	_, err = fmt.Fprintf(os.Stdout, "Closure: %d scenario(s), %d resource(s)\n", len(response.GetScenarios()), len(response.GetResources()))
	return err
}

func UnionExport(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("union export")
	output := fs.String("output", "", "Output JSON path")
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	if strings.TrimSpace(*output) == "" {
		return fmt.Errorf("--output is required")
	}
	response := &selectionv1.GetUnionResponse{}
	if err := Request(core, GetUnionProcedure, &selectionv1.GetUnionRequest{Target: "local"}, response); err != nil {
		return err
	}
	body, err := (protojson.MarshalOptions{Multiline: true, Indent: "  "}).Marshal(response)
	if err != nil {
		return err
	}
	if err := support.WriteOutput(*output, append(body, '\n')); err != nil {
		return err
	}
	_, err = fmt.Fprintln(os.Stdout, "Deployment union exported to", *output)
	return err
}

// ManifestHandlers retains the explicit output-file behavior for the union
// command while its data still comes from the typed SelectionService RPC.
func ManifestHandlers(core *cliapp.ScenarioApp) map[string]cliapp.PrimitiveHandler {
	return map[string]cliapp.PrimitiveHandler{
		"union.export": cliapp.ExternalDelegation(func(ctx cliapp.RunContext) error {
			return UnionExport(core, ctx.Args())
		}),
	}
}

func CoreSet(core *cliapp.ScenarioApp, seed []string) (*selectionv1.GetCoreSetResponse, error) {
	return CoreSetForTarget(core, seed, "local")
}

func CoreSetForTarget(core *cliapp.ScenarioApp, seed []string, target string) (*selectionv1.GetCoreSetResponse, error) {
	response := &selectionv1.GetCoreSetResponse{}
	query := url.Values{}
	_ = query
	if err := Request(core, GetCoreSetProcedure, &selectionv1.GetCoreSetRequest{Target: target, Seed: seed}, response); err != nil {
		return nil, err
	}
	return response, nil
}
