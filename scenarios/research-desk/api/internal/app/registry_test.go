package app_test

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"

	"research-desk/internal/app"
)

// TestAllEndpoints_NonEmpty pins the smoke contract: at minimum the
// health endpoint is registered. Failing this means the registry
// didn't import a handler package or the handler dropped its
// Endpoints slice.

func TestAllEndpoints_NonEmpty(t *testing.T) {
	got := app.AllEndpoints()
	require.NotEmpty(t, got, "AllEndpoints must include at least the health endpoint")
}

// TestAllEndpoints_StableOrder pins the stable-order contract that the
// diff-exit-code CI check on .vrooli/endpoints.json depends on. Two
// consecutive calls must return slices with the same element order;
// otherwise codegen output churns and the diff gate becomes noisy.
func TestAllEndpoints_StableOrder(t *testing.T) {
	a := app.AllEndpoints()
	b := app.AllEndpoints()
	require.Equal(t, len(a), len(b), "non-deterministic length")
	for i := range a {
		require.Equal(t, a[i].ID, b[i].ID,
			"endpoint order changed between calls at index %d (%s vs %s)",
			i, a[i].ID, b[i].ID)
	}
}

// TestProtoConnectParity enforces the proto/Connect-RPC anti-drift
// contract globally: every `rpc Foo(...)` method declared in any proto
// FileDescriptor returned by AllProtoFiles() must have exactly one
// matching EndpointDescriptor in AllEndpoints() whose Path equals
// "/" + service.FullName + "/" + method.Name.

// On failure the message names the proto method, the expected path,
// and the module, so the fix is mechanical: either add the matching
// EndpointDescriptor (referencing the generated *Procedure constant
// from the *v1connect package) or remove the rpc from the proto file.
func TestProtoConnectParity(t *testing.T) {
	endpoints := app.AllEndpoints()
	byPath := make(map[string]int, len(endpoints))
	for _, ep := range endpoints {
		byPath[ep.Path]++
	}

	files := app.AllProtoFiles()
	require.NotEmpty(t, files,
		"AllProtoFiles() returned no entries; every Connect-mounted "+
			"domain module must be registered there")

	for _, entry := range files {
		services := entry.File.Services()
		require.NotZero(t, services.Len(),
			"module %q: proto file declares no services", entry.Module)

		for s := 0; s < services.Len(); s++ {
			svc := services.Get(s)
			methods := svc.Methods()
			for m := 0; m < methods.Len(); m++ {
				method := methods.Get(m)
				wantPath := fmt.Sprintf("/%s/%s", svc.FullName(), method.Name())
				count := byPath[wantPath]
				require.Equal(t, 1, count,
					"module %q: proto method %s.%s (expected EndpointDescriptor.Path %q) "+
						"must have exactly one matching entry in AllEndpoints(); found %d. "+
						"Fix: either add an EndpointDescriptor referencing the generated "+
						"*Procedure constant, or remove the rpc from the proto file.",
					entry.Module, svc.FullName(), method.Name(), wantPath, count)
			}
		}
	}
}
