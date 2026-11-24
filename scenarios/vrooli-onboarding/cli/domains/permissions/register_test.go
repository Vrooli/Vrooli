package permissions

import (
	"bytes"
	"testing"

	permissionsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-onboarding/v1/permissions"

	"github.com/vrooli/cli-core/cliapp"
)

func TestParseWriteRequestPreservesExactJSONValueAndBinding(t *testing.T) {
	const raw = `{"permission_id":"onboarding/host-safeguard/firewall","owner":"vrooli-onboarding","expected_revision":"rev-8","idempotency_key":"request-9","target":{"machine_id":"machine-a","node_id":"node-a","enrollment_generation":"4"},"value":{"config_key":"enabled","value":true}}`
	request, err := parseWriteRequest([]byte(raw))
	if err != nil {
		t.Fatal(err)
	}
	wantValue := `{"config_key":"enabled","value":true}`
	if request.GetPermissionId() != "onboarding/host-safeguard/firewall" || request.GetOwner() != "vrooli-onboarding" || request.GetExpectedRevision() != "rev-8" || request.GetIdempotencyKey() != "request-9" || string(request.GetValueJson()) != wantValue {
		t.Fatalf("parsed request=%+v value=%s", request, request.GetValueJson())
	}
	if request.GetTarget().GetMachineId() != "machine-a" || request.GetTarget().GetNodeId() != "node-a" || request.GetTarget().GetEnrollmentGeneration() != 4 {
		t.Fatalf("target binding=%+v", request.GetTarget())
	}
}

func TestPermissionWriteResultKeepsOwnerExecutionStateVisible(t *testing.T) {
	got := permissionWriteResult(&permissionsv1.PermissionReceipt{Outcome: "revoked", OwnerState: "pending"})
	if len(got) != 2 || got[0] != "Permission outcome: revoked." || got[1] != "Owner execution state: pending." {
		t.Fatalf("pending receipt summary=%v", got)
	}
	confirmed := permissionWriteResult(&permissionsv1.PermissionReceipt{Outcome: "revoked", OwnerState: "confirmed"})
	if len(confirmed) != 2 || confirmed[1] != "Owner execution state: confirmed." {
		t.Fatalf("confirmed receipt summary=%v", confirmed)
	}
}

func TestTargetFromFlagsRequiresCompleteTargetAndBindsGeneration(t *testing.T) {
	schema := cliapp.ArgSchema{Flags: []cliapp.Flag{
		{Name: "machine-id"}, {Name: "node-id"}, {Name: "enrollment-generation"},
	}}
	ctx, err := cliapp.NewTestRunContextFromArgs(schema, []string{"--machine-id", "machine-a", "--node-id", "node-a", "--enrollment-generation", "7"}, nil, &bytes.Buffer{}, &bytes.Buffer{})
	if err != nil {
		t.Fatal(err)
	}
	target, err := targetFromFlags(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if target.GetMachineId() != "machine-a" || target.GetNodeId() != "node-a" || target.GetEnrollmentGeneration() != 7 {
		t.Fatalf("target binding = %+v", target)
	}

	partial, err := cliapp.NewTestRunContextFromArgs(schema, []string{"--machine-id", "machine-a"}, nil, &bytes.Buffer{}, &bytes.Buffer{})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := targetFromFlags(partial); err == nil {
		t.Fatal("partial target binding was accepted")
	}
	zero, err := cliapp.NewTestRunContextFromArgs(schema, []string{"--machine-id", "machine-a", "--node-id", "node-a", "--enrollment-generation", "0"}, nil, &bytes.Buffer{}, &bytes.Buffer{})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := targetFromFlags(zero); err == nil {
		t.Fatal("zero target generation was accepted")
	}
}

func TestPermissionReadsRetainTypedListPrimitiveEvidence(t *testing.T) {
	for name, handler := range map[string]cliapp.PrimitiveHandler{"list": listPermissionsHandler(), "read": readPermissionHandler()} {
		if handler.Primitive() != cliapp.PrimitiveProtoList {
			t.Fatalf("%s primitive = %q", name, handler.Primitive())
		}
	}
}

func TestParseWriteRequestRequiresAllMutationPreconditions(t *testing.T) {
	_, err := parseWriteRequest([]byte(`{"permission_id":"permission","owner":"owner","expected_revision":"rev","idempotency_key":"key"}`))
	if err == nil {
		t.Fatal("write without value was accepted")
	}
	_, err = parseWriteRequest([]byte(`{"permission_id":"permission","owner":"owner","value":{}}`))
	if err == nil {
		t.Fatal("write without revision/idempotency key was accepted")
	}
}
