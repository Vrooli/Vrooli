package permissions

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"

	"github.com/vrooli/cli-core/cliapp"
	permissionsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-onboarding/v1/permissions"
	"vrooli-onboarding/cli/internal/support"
)

const writeProcedure = "/vrooli.vrooli_onboarding.v1.permissions.PermissionRegistryService/WritePermission"
const listProcedure = "/vrooli.vrooli_onboarding.v1.permissions.PermissionRegistryService/ListPermissions"
const readProcedure = "/vrooli.vrooli_onboarding.v1.permissions.PermissionRegistryService/ReadPermission"

// ManifestHandlers keeps arbitrary permission payloads in a caller-owned JSON
// file while sending the same typed request used by UI and API clients.
func ManifestHandlers(core *cliapp.ScenarioApp) map[string]cliapp.PrimitiveHandler {
	return map[string]cliapp.PrimitiveHandler{
		"permissions.list":                          listPermissionsHandler(),
		"PermissionRegistryService.ListPermissions": listPermissionsHandler(),
		"permissions.read":                          readPermissionHandler(),
		"PermissionRegistryService.ReadPermission":  readPermissionHandler(),
		"permissions.write":                         cliapp.ExternalDelegation(runWrite),
		"PermissionRegistryService.WritePermission": cliapp.ExternalDelegation(runWrite),
	}
}

func listPermissionsHandler() cliapp.PrimitiveHandler {
	return cliapp.ProtoList(
		func(ctx cliapp.OperationContext) (*permissionsv1.ListPermissionsResponse, error) {
			target, err := targetFromFlags(ctx)
			if err != nil {
				return nil, err
			}
			request := &permissionsv1.ListPermissionsRequest{Owner: ctx.Flag("owner"), Target: target}
			response := new(permissionsv1.ListPermissionsResponse)
			if err := support.RequestProto(ctx.Core(), listProcedure, request, response, "onboarding permission list"); err != nil {
				return nil, err
			}
			return response, nil
		},
		func(_ cliapp.OperationContext, response *permissionsv1.ListPermissionsResponse) cliapp.ListReport {
			return cliapp.ListReport{Summary: []string{fmt.Sprintf("%d permission(s).", len(response.GetPermissions()))}, ResultsHeading: "Permissions", Results: []string{"Use --json for owner, scope, source, revision, and target details."}}
		},
	)
}

func readPermissionHandler() cliapp.PrimitiveHandler {
	return cliapp.ProtoList(
		func(ctx cliapp.OperationContext) (*permissionsv1.ReadPermissionResponse, error) {
			target, err := targetFromFlags(ctx)
			if err != nil {
				return nil, err
			}
			request := &permissionsv1.ReadPermissionRequest{PermissionId: ctx.Flag("permission-id"), Owner: ctx.Flag("owner"), Target: target}
			response := new(permissionsv1.ReadPermissionResponse)
			if err := support.RequestProto(ctx.Core(), readProcedure, request, response, "onboarding permission read"); err != nil {
				return nil, err
			}
			return response, nil
		},
		func(_ cliapp.OperationContext, _ *permissionsv1.ReadPermissionResponse) cliapp.ListReport {
			return cliapp.ListReport{Summary: []string{"Permission value read from its owner."}, ResultsHeading: "Permission", Results: []string{"Use --json for the typed owner value and source revision."}}
		},
	)
}

func targetFromFlags(ctx cliapp.OperationContext) (*permissionsv1.PermissionTarget, error) {
	machineID, nodeID, generationText := ctx.Flag("machine-id"), ctx.Flag("node-id"), ctx.Flag("enrollment-generation")
	if machineID == "" && nodeID == "" && generationText == "" {
		return nil, nil
	}
	if machineID == "" || nodeID == "" || generationText == "" {
		return nil, fmt.Errorf("target-bound permissions require --machine-id, --node-id, and --enrollment-generation together")
	}
	generation, err := strconv.ParseUint(generationText, 10, 64)
	if err != nil || generation == 0 {
		return nil, fmt.Errorf("--enrollment-generation must be a positive integer")
	}
	return &permissionsv1.PermissionTarget{MachineId: machineID, NodeId: nodeID, EnrollmentGeneration: generation}, nil
}

func runWrite(ctx cliapp.RunContext) error {
	path := ctx.Flag("body-file")
	if path == "" {
		return fmt.Errorf("permissions write requires --body-file")
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read permission write file: %w", err)
	}
	request, err := parseWriteRequest(data)
	if err != nil {
		return err
	}
	response := new(permissionsv1.WritePermissionResponse)
	if err := support.RequestProto(ctx.Core(), writeProcedure, request, response, "onboarding permission"); err != nil {
		return err
	}
	if ctx.JSON() {
		return support.PrintProto(ctx.Stdout(), response, "permission receipt")
	}
	return cliapp.RenderMutationReport(ctx.Stdout(), cliapp.MutationReport{Result: permissionWriteResult(response.GetReceipt()), NextCommand: []string{support.CLIName + " permissions list"}})
}

func permissionWriteResult(receipt *permissionsv1.PermissionReceipt) []string {
	if receipt == nil {
		return []string{"Permission write returned no receipt."}
	}
	if receipt.GetOwnerState() != "" {
		return []string{fmt.Sprintf("Permission outcome: %s.", receipt.GetOutcome()), fmt.Sprintf("Owner execution state: %s.", receipt.GetOwnerState())}
	}
	return []string{"Permission write committed with a durable owner receipt."}
}

func parseWriteRequest(data []byte) (*permissionsv1.WritePermissionRequest, error) {
	var input struct {
		PermissionID     string `json:"permission_id"`
		Owner            string `json:"owner"`
		ExpectedRevision string `json:"expected_revision"`
		IdempotencyKey   string `json:"idempotency_key"`
		Target           *struct {
			MachineID            string          `json:"machine_id"`
			NodeID               string          `json:"node_id"`
			EnrollmentGeneration json.RawMessage `json:"enrollment_generation"`
		} `json:"target"`
		Value json.RawMessage `json:"value"`
	}
	if err := json.Unmarshal(data, &input); err != nil {
		return nil, fmt.Errorf("decode permission write file: %w", err)
	}
	if input.PermissionID == "" || input.Owner == "" || input.ExpectedRevision == "" || input.IdempotencyKey == "" || len(input.Value) == 0 {
		return nil, fmt.Errorf("permission write requires permission_id, owner, expected_revision, idempotency_key, and value")
	}
	var target *permissionsv1.PermissionTarget
	if input.Target != nil {
		var generation uint64
		if len(input.Target.EnrollmentGeneration) > 0 {
			if err := json.Unmarshal(input.Target.EnrollmentGeneration, &generation); err != nil {
				var value string
				if stringErr := json.Unmarshal(input.Target.EnrollmentGeneration, &value); stringErr != nil {
					return nil, fmt.Errorf("decode target enrollment_generation: %w", err)
				}
				parsed, parseErr := strconv.ParseUint(value, 10, 64)
				if parseErr != nil {
					return nil, fmt.Errorf("decode target enrollment_generation: %w", parseErr)
				}
				generation = parsed
			}
		}
		target = &permissionsv1.PermissionTarget{MachineId: input.Target.MachineID, NodeId: input.Target.NodeID, EnrollmentGeneration: generation}
	}
	return &permissionsv1.WritePermissionRequest{PermissionId: input.PermissionID, Owner: input.Owner, Target: target, ExpectedRevision: input.ExpectedRevision, IdempotencyKey: input.IdempotencyKey, ValueJson: append([]byte(nil), input.Value...)}, nil
}
