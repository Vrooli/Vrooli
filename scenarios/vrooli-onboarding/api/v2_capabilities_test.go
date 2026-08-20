package main

import (
	"context"
	"net/http"
	"os/exec"
	"strings"
	"testing"
)

func TestV2CapabilityRoutesUseMetadataOnlyGenericControlPlaneContract(t *testing.T) {
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
			}
		}
		return exec.CommandContext(ctx, "printf", "%s", output)
	}
	t.Cleanup(func() { controlPlaneCommand = previous })
	status := doGet(t, NewServer(), "/api/v2/capabilities")
	if status.Code != http.StatusOK || !strings.Contains(status.Body.String(), `"id":"demo"`) || strings.Contains(status.Body.String(), "sensitive") {
		t.Fatalf("status = %d %s", status.Code, status.Body.String())
	}
	preview := doPost(t, NewServer(), "/api/v2/capabilities/preview", `{"capability_id":"demo","inputs":{"destination":"/approved","secret":"sensitive"}}`)
	if preview.Code != http.StatusOK || !strings.Contains(preview.Body.String(), "write verified metadata") || strings.Contains(preview.Body.String(), "sensitive") {
		t.Fatalf("preview = %d %s", preview.Code, preview.Body.String())
	}
	if unconfirmed := doPost(t, NewServer(), "/api/v2/capabilities/apply", `{"capability_id":"demo","inputs":{"destination":"/approved"}}`); unconfirmed.Code != http.StatusBadRequest {
		t.Fatalf("unconfirmed apply = %d %s", unconfirmed.Code, unconfirmed.Body.String())
	}
	result := doPost(t, NewServer(), "/api/v2/capabilities/apply", `{"capability_id":"demo","confirm":true,"inputs":{"destination":"/approved"}}`)
	if result.Code != http.StatusOK || !strings.Contains(result.Body.String(), `"outcome":"complete"`) || strings.Contains(result.Body.String(), "sensitive") {
		t.Fatalf("apply = %d %s", result.Code, result.Body.String())
	}
	for _, command := range commands {
		if len(command) < 3 || command[0] != "vrooli" || command[1] != "capability" {
			t.Fatalf("capability route invoked non-control-plane command: %q", command)
		}
	}
}
