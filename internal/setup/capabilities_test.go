package setup

import (
	"bytes"
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/vrooli/vrooli/internal/operatorcapability"
	"github.com/vrooli/vrooli/internal/operatorinput"
)

func TestDiscoverAndQueueCapabilitiesPersistsOnlyMetadata(t *testing.T) {
	if err := operatorinput.Replace(nil); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = operatorinput.Replace(nil) })

	descriptor := operatorcapability.Descriptor{
		Version: operatorcapability.ContractVersion, ID: "test-escrow", Owner: "test-owner", Title: "Test escrow",
		Inputs: []operatorcapability.InputDescriptor{
			{ID: "sink", Kind: operatorcapability.KindPath, Label: "Sink", Required: true},
			{ID: "passphrase", Kind: operatorcapability.KindSecret, Label: "Passphrase", Required: true},
		},
		Policy:   operatorcapability.Policy{Idempotent: true},
		Evidence: operatorcapability.EvidenceContract{SecretFree: true},
	}
	var output bytes.Buffer
	err := discoverAndQueueCapabilities(context.Background(), func(context.Context, string, string) ([]operatorcapability.Status, error) {
		return []operatorcapability.Status{{Descriptor: descriptor, MissingInputs: []string{"sink", "passphrase"}}}, nil
	}, "root", "home", &output)
	if err != nil {
		t.Fatal(err)
	}
	queue, err := operatorinput.Load()
	if err != nil {
		t.Fatal(err)
	}
	if len(queue.Requests) != 2 || queue.Requests[0].ID != "test-escrow:sink" || queue.Requests[1].ID != "test-escrow:passphrase" {
		t.Fatalf("queued requests = %+v", queue.Requests)
	}
	encoded, err := json.Marshal(queue)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(encoded), "secret-answer") || strings.Contains(output.String(), "secret-answer") {
		t.Fatalf("secret answer appeared in setup evidence: %s / %s", encoded, output.String())
	}
}
