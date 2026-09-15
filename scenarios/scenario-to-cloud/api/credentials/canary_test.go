package credentials

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"scenario-to-cloud/domain"
)

// [REQ:STC-P0-032] P13-A07 / SECRET-05: after a full lifecycle (materialize,
// rotate with an operator value, break-glass, revoke, recovery) driven with
// synthetic canaries, no retained surface holds a canary: the ledger dump,
// every operation's receipts and JSON projection, the service log, the argv
// sent to the target, the binding list clients read, and the ingest payload
// metadata. Standard-input payloads are the one channel a value may use and
// are deliberately not a retained surface.
func TestCanaryValuesNeverReachRetainedSurfaces(t *testing.T) {
	h := newHarness(t)
	canaries := []string{"canary-generated-7d1f0b3e4a", "canary-operator-token-2c9e8f11", "canary-rotated-token-5b7a9d02", "canary-passphrase-0e4d6c8b"}
	scanner := NewScanner(canaries...)
	minted := 0
	h.svc.Generate = func(domain.CredentialBinding) (string, error) {
		minted++
		value := fmt.Sprintf("%s-%d", canaries[0], minted)
		scanner.Add(value)
		return value, nil
	}
	h.svc.Providers[domain.CredentialClassExternalAPICredential] = &ExternalAPIProvider{Name: "mailer", Probe: func(context.Context, string) error { return nil }, RevokeAPI: func(context.Context, domain.CredentialBinding, domain.CredentialVersion) error { return nil }}

	ctx := context.Background()
	mailer := h.bindingFor("api-token")
	result := h.materialize(map[string]string{mailer.ID: canaries[1]})
	_ = scanner.RecordJSON("materialize-result", result)

	rotation, err := h.svc.Rotate(ctx, RotateRequest{Target: h.targetRef(), DeploymentID: "dep-1", BindingID: mailer.ID, Value: canaries[2]})
	if err != nil {
		t.Fatalf("rotate: %v", err)
	}
	h.mustState(rotation, domain.RotationComplete)

	store := h.bindingFor("password")
	glass, err := h.svc.BreakGlass(ctx, BreakGlassRequest{Target: h.targetRef(), DeploymentID: "dep-1", BindingID: store.ID, Scope: "incident", Operator: "operator:alice", Window: time.Minute, Confirmation: ExpectedBreakGlassConfirmation(store.ID)})
	if err != nil {
		t.Fatalf("break-glass: %v", err)
	}
	h.clock = h.clock.Add(time.Hour)
	if _, err := h.svc.SweepBreakGlass(ctx, "dep-1", h.targetRef()); err != nil {
		t.Fatalf("sweep: %v", err)
	}
	if _, err := h.svc.Revoke(ctx, RevokeBindingRequest{Target: h.targetRef(), DeploymentID: "dep-1", BindingID: h.bindingFor("signing-key").ID}); err != nil {
		t.Fatalf("revoke: %v", err)
	}
	bindings, _ := h.store.ListBindings(ctx, "dep-1")
	covers := make([]domain.CredentialDescriptor, 0, len(bindings))
	for _, b := range bindings {
		covers = append(covers, b.Descriptor)
	}
	replacement := newFakeTarget()
	h.svc.Recovery = &fakeRecovery{covers: covers, restoreTo: replacement, bindings: bindings}
	h.svc.Distributor = &fakeDistributor{target: replacement}
	if _, err := h.svc.Recover(ctx, RecoverRequest{DeploymentID: "dep-1", NewTarget: h.targetRef(), BundleRef: "/root/recovery.bundle", Passphrase: canaries[3]}); err != nil {
		t.Fatalf("recover: %v", err)
	}
	_ = glass

	// Retained surfaces.
	_ = scanner.RecordJSON("ledger-dump", h.store.Dump())
	rotations, _ := h.store.ListRotations(ctx, "dep-1")
	for _, r := range rotations {
		_ = scanner.RecordJSON("rotation-"+r.ID, r)
	}
	views, _ := h.svc.ListBindings(ctx, "dep-1")
	_ = scanner.RecordJSON("binding-list", views)
	scanner.RecordString("service-log", h.logs.String())
	scanner.RecordString("target-argv", h.argvJoined())
	scanner.RecordString("replacement-argv", func() string {
		out := ""
		for _, argv := range replacement.argv {
			out += fmt.Sprint(argv) + "\n"
		}
		return out
	}())
	for _, in := range h.target.stdin {
		var payload IngestPayload
		if json.Unmarshal([]byte(in), &payload) == nil {
			payload.Value = ""
			_ = scanner.RecordJSON("ingest-payload-metadata", payload)
		}
	}
	if len(scanner.Surfaces()) < 8 {
		t.Fatalf("too few surfaces recorded: %v", scanner.Surfaces())
	}
	if findings := scanner.Scan(); len(findings) != 0 {
		for _, f := range findings {
			t.Error(f.String())
		}
		t.FailNow()
	}
	if minted == 0 {
		t.Fatal("no generated canary was minted; the scan proved nothing")
	}
}

// The scanner itself must find a planted canary, or a clean scan proves nothing.
func TestScannerFindsPlantedCanary(t *testing.T) {
	scanner := NewScanner("canary-planted-1234abcd")
	scanner.RecordString("clean", "nothing here")
	_ = scanner.RecordJSON("dirty", map[string]any{"details": map[string]any{"value": "prefix canary-planted-1234abcd suffix"}})
	findings := scanner.Scan()
	if len(findings) != 1 || findings[0].Surface != "dirty" {
		t.Fatalf("findings = %v", findings)
	}
	if findings[0].String() == "" || findings[0].String() == findings[0].Canary {
		t.Fatal("finding message is unusable or repeats the value")
	}
	if got := redact("canary-planted-1234abcd"); got != "cana***************abcd" {
		t.Fatalf("redact = %s", got)
	}
}
