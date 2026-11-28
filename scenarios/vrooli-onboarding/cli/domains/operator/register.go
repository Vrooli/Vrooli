package operator

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"

	onboardingselection "vrooli-onboarding/cli/domains/selection"
	"vrooli-onboarding/cli/internal/support"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliutil"
	operatorstatev1 "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-onboarding/v1/operatorstate"
	permissionsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-onboarding/v1/permissions"
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
	updateControlPermissionID      = "onboarding/install-posture/update-control"
)

// Register exposes the V2 onboarding control plane. It owns no state: the API
// persists operator decisions and derives scenarios/readiness from manifests.
func Register(core *cliapp.ScenarioApp) cliapp.SubcommandGroup {
	return cliapp.SubcommandGroup{Name: "operator", Description: "Inspect and commit V2 operator state", NeedsAPI: true, Subcommands: []cliapp.Command{
		{Name: "show", Description: "Show persisted operator state", Run: func(args []string) error { return runShow(core, args) }},
		{Name: "patch", Description: "Apply a masked operator-state patch from --body-file", Run: func(args []string) error { return runPatch(core, args) }},
		{Name: "update-control", Description: "Change host update control through its typed owner permission", Run: func(args []string) error { return runUpdateControl(core, args) }},
		{Name: "scenarios", Description: "Show manifest-derived scenario choices", Run: func(args []string) error { return onboardingselection.List(core, args) }},
		{Name: "readiness", Description: "Show metadata-safe composed readiness", Run: func(args []string) error { return support.GetJSON(core, "operator", args, "/v2/readiness") }},
	}}
}

func runUpdateControl(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("operator update-control")
	mode := fs.String("mode", "", "Host update mode: observe or guard")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	return writeUpdateControl(core, *mode, *jsonOutput)
}

func runManifestUpdateControl(ctx cliapp.RunContext) error {
	return writeUpdateControl(ctx.Core(), ctx.Flag("mode"), ctx.JSON())
}

func writeUpdateControl(core *cliapp.ScenarioApp, mode string, jsonOutput bool) error {
	if mode != "observe" && mode != "guard" {
		return fmt.Errorf("update-control mode must be observe or guard; own is reserved")
	}
	read := new(permissionsv1.ReadPermissionResponse)
	readRequest := &permissionsv1.ReadPermissionRequest{PermissionId: updateControlPermissionID, Owner: "vrooli-onboarding"}
	if err := support.RequestProto(core, "/vrooli.vrooli_onboarding.v1.permissions.PermissionRegistryService/ReadPermission", readRequest, read, "update-control permission"); err != nil {
		return fmt.Errorf("read update control: %w", err)
	}
	if read.GetRevision() == "" {
		return fmt.Errorf("update-control owner returned no source revision")
	}
	requestID := make([]byte, 16)
	if _, err := rand.Read(requestID); err != nil {
		return fmt.Errorf("create update-control request id: %w", err)
	}
	payload, _ := json.Marshal(map[string]string{"value": mode})
	write := new(permissionsv1.WritePermissionResponse)
	writeRequest := &permissionsv1.WritePermissionRequest{PermissionId: updateControlPermissionID, Owner: "vrooli-onboarding", ExpectedRevision: read.GetRevision(), IdempotencyKey: "cli-update-control:" + hex.EncodeToString(requestID), ValueJson: payload}
	if err := support.RequestProto(core, "/vrooli.vrooli_onboarding.v1.permissions.PermissionRegistryService/WritePermission", writeRequest, write, "update-control permission"); err != nil {
		return fmt.Errorf("write update control: %w", err)
	}
	if jsonOutput {
		return support.PrintProto(os.Stdout, write, "update-control receipt")
	}
	return cliapp.RenderMutationReport(os.Stdout, cliapp.MutationReport{Result: []string{"Host update control saved with a durable owner receipt"}, NextCommand: []string{support.CLIName + " operator show"}})
}

// ManifestHandlers retains the file-backed merge-patch behavior while the
// manifest remains the command and governance source of truth. The handler
// still sends a typed PatchOperatorState request after deriving its field mask.
func ManifestHandlers(core *cliapp.ScenarioApp) map[string]cliapp.PrimitiveHandler {
	return map[string]cliapp.PrimitiveHandler{
		"operator.patch":          cliapp.ExternalDelegation(runManifestPatch),
		"operator.update-control": cliapp.ExternalDelegation(runManifestUpdateControl),
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
	result, err := applyOperatorPatch(ctx.Core(), state, raw)
	if err != nil {
		return err
	}
	if ctx.JSON() {
		return printStateTo(ctx.Stdout(), result)
	}
	return cliapp.RenderMutationReport(ctx.Stdout(), cliapp.MutationReport{Result: []string{"Operator state and typed permissions committed"}, NextCommand: []string{support.CLIName + " operator readiness", support.CLIName + " operator scenarios"}})
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
	result, err := applyOperatorPatch(core, state, raw)
	if err != nil {
		return err
	}
	if *jsonOutput {
		return printState(result)
	}
	return cliapp.RenderMutationReport(os.Stdout, cliapp.MutationReport{Result: []string{"Operator state and typed permissions committed"}, NextCommand: []string{support.CLIName + " operator readiness", support.CLIName + " operator scenarios"}})
}

func applyOperatorPatch(core *cliapp.ScenarioApp, state *operatorstatev1.OperatorState, raw map[string]json.RawMessage) (*operatorstatev1.OperatorState, error) {
	for _, field := range []string{"trust_posture", "update_control", "session_mode"} {
		if _, exists := raw[field]; exists {
			switch field {
			case "update_control":
				return nil, fmt.Errorf("update_control must use `vrooli-onboarding operator update-control --mode observe|guard`")
			case "session_mode":
				return nil, fmt.Errorf("session_mode is enrollment-owned; use the Bridge operator-session enrollment flow")
			default:
				return nil, fmt.Errorf("trust_posture is control-plane-owned and cannot be changed through onboarding")
			}
		}
	}
	choices := state.GetHostSafeguards()
	hostTools := state.GetHostTools()
	coreAuthority := state.GetCore()
	delete(raw, "host_safeguards")
	delete(raw, "host_tools")
	delete(raw, "core")
	state.HostSafeguards = nil
	state.HostTools = nil
	state.Core = nil
	if len(raw) == 0 && len(choices) == 0 && len(hostTools) == 0 && coreAuthority == nil {
		return nil, fmt.Errorf("operator patch contains no writable fields")
	}
	var result *operatorstatev1.OperatorState
	if len(raw) > 0 {
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
			return nil, err
		}
		result = response.GetState()
	}
	if len(choices) > 0 {
		if err := writeSafeguards(core, choices); err != nil {
			return nil, fmt.Errorf("operator fields were saved, but a host safeguard permission failed: %w", err)
		}
		response := new(operatorstatev1.GetOperatorStateResponse)
		if err := request(core, operatorStateGetProcedure, &operatorstatev1.GetOperatorStateRequest{Target: "local"}, response); err != nil {
			return nil, err
		}
		result = response.GetState()
	}
	if len(hostTools) > 0 {
		if err := writeHostTools(core, hostTools); err != nil {
			return nil, fmt.Errorf("operator fields were saved, but a host tool permission failed: %w", err)
		}
		response := new(operatorstatev1.GetOperatorStateResponse)
		if err := request(core, operatorStateGetProcedure, &operatorstatev1.GetOperatorStateRequest{Target: "local"}, response); err != nil {
			return nil, err
		}
		result = response.GetState()
	}
	if coreAuthority != nil {
		if err := writeCoreAuthority(core, coreAuthority); err != nil {
			return nil, fmt.Errorf("operator fields were saved, but core authority failed: %w", err)
		}
		response := new(operatorstatev1.GetOperatorStateResponse)
		if err := request(core, operatorStateGetProcedure, &operatorstatev1.GetOperatorStateRequest{Target: "local"}, response); err != nil {
			return nil, err
		}
		result = response.GetState()
	}
	return result, nil
}

func writeCoreAuthority(core *cliapp.ScenarioApp, authority *operatorstatev1.CoreSet) error {
	if authority == nil {
		return fmt.Errorf("core authority is empty")
	}
	permissionID := "onboarding/core-authority"
	read := new(permissionsv1.ReadPermissionResponse)
	if err := support.RequestProto(core, "/vrooli.vrooli_onboarding.v1.permissions.PermissionRegistryService/ReadPermission", &permissionsv1.ReadPermissionRequest{PermissionId: permissionID, Owner: "vrooli-onboarding"}, read, "core authority permission"); err != nil {
		return fmt.Errorf("read core authority: %w", err)
	}
	var existing map[string]json.RawMessage
	if err := json.Unmarshal(read.GetValueJson(), &existing); err != nil {
		return fmt.Errorf("decode current core authority: %w", err)
	}
	payloadValue := map[string]json.RawMessage{}
	for key, value := range existing {
		payloadValue[key] = value
	}
	seed, err := json.Marshal(authority.GetSeed())
	if err != nil {
		return err
	}
	trustedBase, err := json.Marshal(authority.GetTrustedBase())
	if err != nil {
		return err
	}
	if _, ok := payloadValue["important"]; !ok {
		payloadValue["important"] = json.RawMessage("[]")
	}
	payloadValue["seed"], payloadValue["trusted_base"] = seed, trustedBase
	payload, err := json.Marshal(payloadValue)
	if err != nil {
		return err
	}
	var requestID [16]byte
	if _, err := rand.Read(requestID[:]); err != nil {
		return fmt.Errorf("create core authority request id: %w", err)
	}
	request := &permissionsv1.WritePermissionRequest{PermissionId: permissionID, Owner: "vrooli-onboarding", ExpectedRevision: read.GetRevision(), IdempotencyKey: "cli-core-authority:" + hex.EncodeToString(requestID[:]), ValueJson: payload}
	response := new(permissionsv1.WritePermissionResponse)
	if err := support.RequestProto(core, "/vrooli.vrooli_onboarding.v1.permissions.PermissionRegistryService/WritePermission", request, response, "core authority permission"); err != nil {
		return fmt.Errorf("write core authority: %w", err)
	}
	return nil
}

func writeSafeguards(core *cliapp.ScenarioApp, choices map[string]*operatorstatev1.OptInChoice) error {
	for name, choice := range choices {
		if choice == nil {
			return fmt.Errorf("host safeguard %q choice is empty", name)
		}
		value := map[string]any{"opted_in": choice.GetOptedIn()}
		if choice.GetConfig() != nil && len(choice.GetConfig().GetFields()) > 0 {
			config := make(map[string]any, len(choice.GetConfig().GetFields()))
			for key, entry := range choice.GetConfig().GetFields() {
				if entry == nil {
					config[key] = nil
				} else {
					config[key] = entry.AsInterface()
				}
			}
			value["config"] = config
		}
		payload, err := json.Marshal(value)
		if err != nil {
			return fmt.Errorf("encode host safeguard %q: %w", name, err)
		}
		permissionID := "onboarding/host-safeguard/" + name
		read := new(permissionsv1.ReadPermissionResponse)
		if err := support.RequestProto(core, "/vrooli.vrooli_onboarding.v1.permissions.PermissionRegistryService/ReadPermission", &permissionsv1.ReadPermissionRequest{PermissionId: permissionID, Owner: "vrooli-onboarding"}, read, "host safeguard permission"); err != nil {
			return fmt.Errorf("read host safeguard %s: %w", name, err)
		}
		var requestID [16]byte
		if _, err := rand.Read(requestID[:]); err != nil {
			return fmt.Errorf("create host safeguard request id: %w", err)
		}
		write := new(permissionsv1.WritePermissionResponse)
		request := &permissionsv1.WritePermissionRequest{PermissionId: permissionID, Owner: "vrooli-onboarding", ExpectedRevision: read.GetRevision(), IdempotencyKey: "cli-operator-patch:" + hex.EncodeToString(requestID[:]), ValueJson: payload}
		if err := support.RequestProto(core, "/vrooli.vrooli_onboarding.v1.permissions.PermissionRegistryService/WritePermission", request, write, "host safeguard permission"); err != nil {
			return fmt.Errorf("write host safeguard %s: %w", name, err)
		}
	}
	return nil
}

func writeHostTools(core *cliapp.ScenarioApp, choices map[string]*operatorstatev1.OptInChoice) error {
	for name, choice := range choices {
		if choice == nil {
			return fmt.Errorf("host tool %q choice is empty", name)
		}
		payload, err := json.Marshal(map[string]bool{"opted_in": choice.GetOptedIn()})
		if err != nil {
			return err
		}
		permissionID := "onboarding/host-tool/selected-closure/" + name
		read := new(permissionsv1.ReadPermissionResponse)
		if err := support.RequestProto(core, "/vrooli.vrooli_onboarding.v1.permissions.PermissionRegistryService/ReadPermission", &permissionsv1.ReadPermissionRequest{PermissionId: permissionID, Owner: "vrooli-onboarding"}, read, "host tool permission"); err != nil {
			return fmt.Errorf("read host tool %s: %w", name, err)
		}
		var requestID [16]byte
		if _, err := rand.Read(requestID[:]); err != nil {
			return fmt.Errorf("create host tool request id: %w", err)
		}
		write := new(permissionsv1.WritePermissionResponse)
		request := &permissionsv1.WritePermissionRequest{PermissionId: permissionID, Owner: "vrooli-onboarding", ExpectedRevision: read.GetRevision(), IdempotencyKey: "cli-operator-patch:" + hex.EncodeToString(requestID[:]), ValueJson: payload}
		if err := support.RequestProto(core, "/vrooli.vrooli_onboarding.v1.permissions.PermissionRegistryService/WritePermission", request, write, "host tool permission"); err != nil {
			return fmt.Errorf("write host tool %s: %w", name, err)
		}
	}
	return nil
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
