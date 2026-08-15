package setup

import (
	"bytes"
	"strings"
	"testing"

	"github.com/vrooli/vrooli/internal/operatorinput"
)

func TestCredentialStoreInputIsQueuedWithoutReadingASecret(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	var output bytes.Buffer
	if err := enqueueCredentialStoreInput(false, &output); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(output.String(), "vrooli-onboarding") {
		t.Fatalf("handoff output = %q", output.String())
	}
	requests, err := operatorinput.Load()
	if err != nil {
		t.Fatal(err)
	}
	if len(requests.Requests) != 1 || requests.Requests[0].ID != "credential-store-passphrase" {
		t.Fatalf("queued requests = %+v", requests)
	}
}
