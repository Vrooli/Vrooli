package main

import (
	"context"
	"encoding/json"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"testing"

	"github.com/vrooli/vrooli/internal/operatorcapability"
)

func TestV2CapabilityRoutesUseMetadataOnlyGenericControlPlaneContract(t *testing.T) {
	// Isolate the runtime home. Reading the operator's real one makes the test
	// depend on host state it does not own; on this host that queue file was
	// left root-owned by an elevated setup run, so the reconcile step failed
	// with a permission error that had nothing to do with the contract.
	t.Setenv("HOME", t.TempDir())
	previous := controlPlaneCommand
	var commands [][]string
	controlPlaneCommand = func(ctx context.Context, name string, args ...string) *exec.Cmd {
		commands = append(commands, append([]string{name}, args...))
		output := `[]`
		if len(args) > 1 {
			switch args[1] {
			case "status":
				output = `[{"descriptor":{"version":"operator-capability/v1","id":"demo","owner":"demo.owner","title":"Demo action","policy":{"requires_confirmation":true,"idempotent":true,"retryable":true},"evidence":{"secret_free":true}},"state":"needs_operator_input","missing_inputs":["destination"],"updated_at":"2026-08-19T00:00:00Z"}]`
			case "preview":
				output = `{"capability_id":"demo","plan_id":"demo-plan","state":"ready_to_preview","mutations":[{"id":"write","summary":"write verified metadata","reversible":true}]}`
			case "apply":
				output = `{"capability_id":"demo","state":"ready","outcome":"complete","retryable":true,"evidence":[{"kind":"demo","artifact_identity":"demo-artifact","observed_at":"2026-08-19T00:00:00Z","verified":true}]}`
			case "verify":
				output = `[{"capability_id":"demo","kind":"demo","artifact_identity":"demo-artifact","target_id":"local","operation":"readiness-check","observed_at":"2026-08-19T00:00:00Z","expires_at":"2026-08-20T00:00:00Z","verified":true,"effect_class":"read_only"}]`
			}
		}
		return exec.CommandContext(ctx, "printf", "%s", output)
	}
	t.Cleanup(func() { controlPlaneCommand = previous })
	status := doPost(t, NewServer(), "/vrooli.vrooli_onboarding.v1.capabilities.CapabilitiesService/ListCapabilities", `{"target":"local"}`)
	if status.Code != http.StatusOK || !strings.Contains(status.Body.String(), `"id":"demo"`) || strings.Contains(status.Body.String(), "sensitive") {
		t.Fatalf("status = %d %s", status.Code, status.Body.String())
	}
	preview := doPost(t, NewServer(), "/vrooli.vrooli_onboarding.v1.capabilities.CapabilitiesService/PreviewCapability", `{"target":"local","action":{"capabilityId":"demo","inputs":{"destination":"/approved","secret":"sensitive"}}}`)
	if preview.Code != http.StatusOK || !strings.Contains(preview.Body.String(), "write verified metadata") || strings.Contains(preview.Body.String(), "sensitive") {
		t.Fatalf("preview = %d %s", preview.Code, preview.Body.String())
	}
	if unconfirmed := doPost(t, NewServer(), "/vrooli.vrooli_onboarding.v1.capabilities.CapabilitiesService/ApplyCapability", `{"target":"local","action":{"capabilityId":"demo","inputs":{"destination":"/approved"}}}`); unconfirmed.Code != http.StatusBadRequest {
		t.Fatalf("unconfirmed apply = %d %s", unconfirmed.Code, unconfirmed.Body.String())
	}
	result := doPost(t, NewServer(), "/vrooli.vrooli_onboarding.v1.capabilities.CapabilitiesService/ApplyCapability", `{"target":"local","action":{"capabilityId":"demo","confirm":true,"inputs":{"destination":"/approved"}}}`)
	if result.Code != http.StatusOK || !strings.Contains(result.Body.String(), `"outcome":"complete"`) || strings.Contains(result.Body.String(), "sensitive") {
		t.Fatalf("apply = %d %s", result.Code, result.Body.String())
	}
	verified := doPost(t, NewServer(), "/vrooli.vrooli_onboarding.v1.capabilities.CapabilitiesService/VerifyCapability", `{"target":"local","verification":{"capabilityId":"demo","targetId":"local","operation":"readiness-check","effectClass":"read_only"}}`)
	if verified.Code != http.StatusOK || !strings.Contains(verified.Body.String(), `"verified":true`) {
		t.Fatalf("verify = %d %s", verified.Code, verified.Body.String())
	}
	for _, command := range commands {
		if len(command) < 3 || command[0] != "vrooli" || command[1] != "capability" {
			t.Fatalf("capability route invoked non-control-plane command: %q", command)
		}
	}
}

func TestCapabilityApplyPassesTargetToControlPlaneOwner(t *testing.T) {
	capture := t.TempDir() + "/action.json"
	previous := controlPlaneCommand
	controlPlaneCommand = func(ctx context.Context, _ string, _ ...string) *exec.Cmd {
		return exec.CommandContext(ctx, "sh", "-c", `tee "$1" >/dev/null; printf '%s' '{"capability_id":"demo","state":"ready","outcome":"complete"}'`, "capability-target-test", capture)
	}
	t.Cleanup(func() { controlPlaneCommand = previous })

	result, err := (controlPlaneExecutor{}).applyCapability(context.Background(), operatorcapability.ActionRequest{CapabilityID: "demo", TargetID: "node-7", IdempotencyKey: "demo-node-7", Confirm: true})
	if err != nil {
		t.Fatal(err)
	}
	if result.State != "ready" {
		t.Fatalf("result = %+v", result)
	}
	data, err := os.ReadFile(capture)
	if err != nil {
		t.Fatal(err)
	}
	var payload map[string]any
	if err := json.Unmarshal(data, &payload); err != nil {
		t.Fatalf("decode owner payload: %v; payload=%s", err, data)
	}
	if payload["target_id"] != "node-7" {
		t.Fatalf("owner target_id = %#v, want node-7; payload=%s", payload["target_id"], data)
	}
}
