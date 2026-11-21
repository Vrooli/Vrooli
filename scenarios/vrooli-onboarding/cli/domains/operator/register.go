package operator

import (
	"encoding/json"
	"fmt"
	"io"
	"os"

	onboardingselection "vrooli-onboarding/cli/domains/selection"
	"vrooli-onboarding/cli/internal/support"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliutil"
	operatorstatev1 "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-onboarding/v1/operatorstate"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/fieldmaskpb"
)

// Manifest bound procedure addresses retained here make the CLI surface
// contract searchable while the manifest runtime owns their invocation.
const (
	operatorInputsListProcedure    = "/vrooli.vrooli_onboarding.v1.operatorinputs.OperatorInputsService/ListOperatorInputs"
	operatorInputsResolveProcedure = "/vrooli.vrooli_onboarding.v1.operatorinputs.OperatorInputsService/ResolveOperatorInputs"
	operatorStateGetProcedure      = "/vrooli.vrooli_onboarding.v1.operatorstate.OperatorStateService/GetOperatorState"
	operatorStatePatchProcedure    = "/vrooli.vrooli_onboarding.v1.operatorstate.OperatorStateService/PatchOperatorState"
)

// Register exposes the V2 onboarding control plane. It owns no state: the API
// persists operator decisions and derives scenarios/readiness from manifests.
func Register(core *cliapp.ScenarioApp) cliapp.SubcommandGroup {
	return cliapp.SubcommandGroup{Name: "operator", Description: "Inspect and commit V2 operator state", NeedsAPI: true, Subcommands: []cliapp.Command{
		{Name: "show", Description: "Show persisted operator state", Run: func(args []string) error { return runShow(core, args) }},
		{Name: "patch", Description: "Apply a masked operator-state patch from --body-file", Run: func(args []string) error { return runPatch(core, args) }},
		{Name: "scenarios", Description: "Show manifest-derived scenario choices", Run: func(args []string) error { return onboardingselection.List(core, args) }},
		{Name: "readiness", Description: "Show metadata-safe composed readiness", Run: func(args []string) error { return support.GetJSON(core, "operator", args, "/v2/readiness") }},
	}}
}

// ManifestHandlers retains the file-backed merge-patch behavior while the
// manifest remains the command and governance source of truth. The handler
// still sends a typed PatchOperatorState request after deriving its field mask.
func ManifestHandlers(core *cliapp.ScenarioApp) map[string]cliapp.PrimitiveHandler {
	return map[string]cliapp.PrimitiveHandler{
		"operator.patch": cliapp.ExternalDelegation(runManifestPatch),
	}
}

func runManifestPatch(ctx cliapp.RunContext) error {
	body, err := support.ReadJSONFile(ctx.Flag("body-file"), true)
	if err != nil {
		return err
	}
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(body, &raw); err != nil || len(raw) == 0 {
		return fmt.Errorf("operator patch must be a JSON object")
	}
	state := new(operatorstatev1.OperatorState)
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(body, state); err != nil {
		return fmt.Errorf("decode operator state patch: %w", err)
	}
	paths := make([]string, 0, len(raw))
	for key := range raw {
		if key == "$schema" {
			paths = append(paths, "schema")
			continue
		}
		paths = append(paths, fieldMaskPath(key))
	}
	response := new(operatorstatev1.PatchOperatorStateResponse)
	if err := request(ctx.Core(), operatorStatePatchProcedure, &operatorstatev1.PatchOperatorStateRequest{Target: "local", State: state, UpdateMask: &fieldmaskpb.FieldMask{Paths: paths}}, response); err != nil {
		return err
	}
	if ctx.JSON() {
		return printStateTo(ctx.Stdout(), response.GetState())
	}
	return cliapp.RenderMutationReport(ctx.Stdout(), cliapp.MutationReport{Result: []string{"Operator state committed atomically"}, NextCommand: []string{support.CLIName + " operator readiness", support.CLIName + " operator scenarios"}})
}

func runShow(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("operator show")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	response := new(operatorstatev1.GetOperatorStateResponse)
	if err := request(core, operatorStateGetProcedure, &operatorstatev1.GetOperatorStateRequest{Target: "local"}, response); err != nil {
		return err
	}
	_ = jsonOutput
	return printState(response.GetState())
}

func runPatch(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("operator patch")
	bodyFile := fs.String("body-file", "", "Path to a JSON operator-state patch")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	body, err := support.ReadJSONFile(*bodyFile, true)
	if err != nil {
		return err
	}
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(body, &raw); err != nil || len(raw) == 0 {
		return fmt.Errorf("operator patch must be a JSON object")
	}
	state := new(operatorstatev1.OperatorState)
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(body, state); err != nil {
		return fmt.Errorf("decode operator state patch: %w", err)
	}
	paths := make([]string, 0, len(raw))
	for key := range raw {
		if key == "$schema" {
			paths = append(paths, "schema")
			continue
		}
		paths = append(paths, fieldMaskPath(key))
	}
	response := new(operatorstatev1.PatchOperatorStateResponse)
	if err := request(core, operatorStatePatchProcedure, &operatorstatev1.PatchOperatorStateRequest{Target: "local", State: state, UpdateMask: &fieldmaskpb.FieldMask{Paths: paths}}, response); err != nil {
		return err
	}
	if *jsonOutput {
		return printState(response.GetState())
	}
	return cliapp.RenderMutationReport(os.Stdout, cliapp.MutationReport{Result: []string{"Operator state committed atomically"}, NextCommand: []string{support.CLIName + " operator readiness", support.CLIName + " operator scenarios"}})
}

func request(core *cliapp.ScenarioApp, procedure string, message, response proto.Message) error {
	return support.RequestProto(core, procedure, message, response, "operator state")
}

func printState(state *operatorstatev1.OperatorState) error {
	return printStateTo(os.Stdout, state)
}

func printStateTo(writer io.Writer, state *operatorstatev1.OperatorState) error {
	if state == nil {
		state = new(operatorstatev1.OperatorState)
	}
	return support.PrintProto(writer, state, "operator state")
}

func fieldMaskPath(value string) string {
	// FieldMask paths use the protobuf field spelling, not the JSON lowerCamel
	// spelling. protojson rejects a lowerCamel path when it cannot round-trip it
	// losslessly, which made manifest-driven operator patches fail for
	// host_tools and host_safeguards even though the state payload decoded.
	return value
}
