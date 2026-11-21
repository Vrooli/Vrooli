package runtimeapp

import (
	"testing"

	"github.com/vrooli/vrooli/internal/stateformat"
)

func TestSharedModuleOwnerDescriptorsAreContractOnly(t *testing.T) {
	for _, tc := range []struct {
		name    string
		provide func() (stateformat.Store, error)
		pinned  string
	}{
		{"tool-installation", toolInstallationStateFormat, toolInstallationSchemaDigest},
		{"break-glass", breakGlassStateFormat, breakGlassSchemaDigest},
		{"credential-recovery", credentialRecoveryStateFormat, credentialRecoverySchemaDigest},
		{"harness-lifecycle", harnessLifecycleStateFormat, harnessLifecycleSchemaDigest},
	} {
		t.Run(tc.name, func(t *testing.T) {
			store, err := tc.provide()
			if err != nil {
				t.Fatal(err)
			}
			if err := stateformat.SortStores([]stateformat.Store{store}); err != nil {
				t.Fatalf("state format is invalid: %v", err)
			}
			if got := store.Schemas[store.Write]; got != tc.pinned {
				t.Fatalf("schema digest = %s, want pinned %s", got, tc.pinned)
			}
			if !store.ContractOnly || len(store.StatePaths) != 0 {
				t.Fatalf("owner writes outside the runtime home, so it must be contract-only: %+v", store)
			}
		})
	}
}
