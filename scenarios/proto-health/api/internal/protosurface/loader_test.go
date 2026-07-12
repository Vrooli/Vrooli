package protosurface

import (
	"os"
	"path/filepath"
	"sort"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDescriptorLoaderLoadScenario(t *testing.T) {
	repoRoot := findRepoRoot(t)
	descriptorPath := filepath.Join(repoRoot, "packages", "proto", "gen", "descriptor", "image.binpb")
	loader, err := NewDescriptorLoaderFromFile(repoRoot, descriptorPath)
	require.NoError(t, err)

	surface, err := loader.LoadScenario("proto-health")
	require.NoError(t, err)

	require.Equal(t, "proto-health", surface.Scenario)
	require.NotEmpty(t, surface.Files)
	require.NotEmpty(t, surface.Messages)
	require.Equal(t, TransportWorldConnect, surface.TransportWorld)

	var sawValidation bool
	var sawStability bool
	for _, f := range surface.Files {
		if f.Path == "proto-health/v1/validation/validation.proto" {
			sawValidation = true
			require.Equal(t, "v1", f.Version)
			require.Equal(t, "validation", f.Domain)
			require.Equal(t, "stable", f.Stability)
			for _, a := range f.Annotations {
				if a.Name == "stability" && a.Value == "stable" {
					sawStability = true
				}
			}
		}
	}
	require.True(t, sawValidation)
	require.True(t, sawStability)

	var sawRPC bool
	for _, svc := range surface.Services {
		if svc.FullName != "vrooli.proto_health.v1.validation.ProtoHealthService" {
			continue
		}
		for _, rpc := range svc.RPCs {
			if rpc.Name == "DescribeScenarioProtos" {
				sawRPC = true
				require.Equal(t, "vrooli.proto_health.v1.validation.DescribeScenarioProtosRequest", rpc.Input)
				require.Equal(t, TransportKindConnect, rpc.Transport)
			}
		}
	}
	require.True(t, sawRPC)

	var sawMapEntry bool
	for _, msg := range surface.Messages {
		if msg.FullName == "vrooli.proto_health.v1.shared.ErrorEnvelope.DetailsEntry" {
			sawMapEntry = true
			require.True(t, msg.IsMapEntry)
		}
		if msg.FullName == "vrooli.proto_health.v1.shared.ErrorEnvelope" {
			require.False(t, msg.IsMapEntry)
		}
	}
	require.True(t, sawMapEntry)

	// proto-health's only REST exception is the ops_probe health endpoint;
	// every domain RPC is Connect, so no other exception should appear.
	require.Contains(t, surface.RESTExceptions, RESTExceptionEndpoint{
		EndpointID:             "health",
		Path:                   "/health",
		Method:                 "GET",
		Domain:                 "system",
		Reason:                 "ops_probe",
		HasPayloadDeclarations: true,
	})
}

func TestDescriptorLoaderListScenarios(t *testing.T) {
	repoRoot := findRepoRoot(t)
	descriptorPath := filepath.Join(repoRoot, "packages", "proto", "gen", "descriptor", "image.binpb")
	loader, err := NewDescriptorLoaderFromFile(repoRoot, descriptorPath)
	require.NoError(t, err)

	scenarios, err := loader.ListScenarios()
	require.NoError(t, err)

	require.Contains(t, scenarios, "proto-health")
	require.Contains(t, scenarios, "code-facts")
	sorted := append([]string{}, scenarios...)
	sort.Strings(sorted)
	require.Equal(t, sorted, scenarios)
}

func TestApplyTransportFactsAtScenarioPathUsesPhysicalWorkspace(t *testing.T) {
	scenarioPath := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(scenarioPath, ".vrooli"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(scenarioPath, ".vrooli", "endpoints.json"), []byte(`{
  "endpoints": [{
    "id": "notes-list",
    "path": "/example.v1.notes.NotesService/ListNotes",
    "method": "POST",
    "category": "notes"
  }]
}`), 0o644))
	surface := Surface{
		Scenario: "example",
		Services: []Service{{
			FilePath: "example/v1/notes/notes.proto",
			FullName: "example.v1.notes.NotesService",
			RPCs:     []RPC{{Name: "ListNotes"}},
		}},
	}

	ApplyTransportFactsAtScenarioPath(&surface, scenarioPath)

	require.Equal(t, TransportWorldConnect, surface.TransportWorld)
	require.Equal(t, TransportKindConnect, surface.Services[0].RPCs[0].Transport)
}

func findRepoRoot(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	require.NoError(t, err)
	for {
		if _, err := os.Stat(filepath.Join(dir, "packages", "proto", "buf.yaml")); err == nil {
			return dir
		}
		next := filepath.Dir(dir)
		require.NotEqual(t, dir, next, "repo root not found from %s", dir)
		dir = next
	}
}
