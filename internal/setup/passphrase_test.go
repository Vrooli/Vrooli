package setup

import (
	"bytes"
	"strings"
	"testing"

	"github.com/vrooli/vrooli/internal/operatorcapability"
	"github.com/vrooli/vrooli/internal/testenv"
)

func TestCredentialStoreInputIsQueuedWithoutReadingASecret(t *testing.T) {
	testenv.SetIdentityEnv(t, map[string]string{"HOME": t.TempDir()})

	var output bytes.Buffer
	if err := enqueueCredentialStoreInput(false, &output); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(output.String(), "vrooli-onboarding") {
		t.Fatalf("handoff output = %q", output.String())
	}
	requests, err := operatorcapability.Load()
	if err != nil {
		t.Fatal(err)
	}
	if len(requests.Requests) != 1 || requests.Requests[0].ID != "credential-store-passphrase" {
		t.Fatalf("queued requests = %+v", requests)
	}
}
