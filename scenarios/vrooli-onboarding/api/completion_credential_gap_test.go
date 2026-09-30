package main

import (
	"strings"
	"testing"
)

// TestCredentialGapNamesAnAbsentStore pins the misdirection seen on minimouse:
// with no credential store on the machine, every credential read "the
// credential backend could not answer for this address" and told the operator
// to retry, while the only fix was creating the store.
func TestCredentialGapNamesAnAbsentStore(t *testing.T) {
	absent := credentialReadiness{Status: "unsupported", ProviderState: "absent"}
	if reason := credentialGapReason(absent); !strings.Contains(reason, "no credential store exists") {
		t.Fatalf("reason = %q, want it to name the absent store", reason)
	}
	remediation := credentialGapRemediation(absent)
	for _, want := range []string{"Protect the encrypted credential store", "vrooli credentials store init"} {
		if !strings.Contains(remediation, want) {
			t.Fatalf("remediation = %q, want it to contain %q", remediation, want)
		}
	}
}

func TestCredentialGapCarriesAnUnavailableStoresOwnReason(t *testing.T) {
	unavailable := credentialReadiness{Status: "unsupported", ProviderState: "unavailable", ProviderDetail: "the login keyring is locked"}
	if reason := credentialGapReason(unavailable); !strings.Contains(reason, "the login keyring is locked") {
		t.Fatalf("reason = %q, want the store's own detail", reason)
	}
	if remediation := credentialGapRemediation(unavailable); !strings.Contains(remediation, "vrooli credentials doctor") {
		t.Fatalf("remediation = %q, want the doctor path for an unavailable store", remediation)
	}
}

func TestCredentialGapForAnUnsetValueIsUnchanged(t *testing.T) {
	if reason := credentialGapReason(credentialReadiness{Status: "missing", ProviderState: "available"}); reason != "the credential is declared and not configured" {
		t.Fatalf("reason = %q", reason)
	}
}
