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

	var sawNotes bool
	var sawStability bool
	for _, f := range surface.Files {
		if f.Path == "proto-health/v1/notes/notes.proto" {
			sawNotes = true
			require.Equal(t, "v1", f.Version)
			require.Equal(t, "notes", f.Domain)
			require.Equal(t, "stable", f.Stability)
			for _, a := range f.Annotations {
				if a.Name == "stability" && a.Value == "stable" {
					sawStability = true
				}
			}
		}
	}
	require.True(t, sawNotes)
	require.True(t, sawStability)

	var sawRPC bool
	for _, svc := range surface.Services {
		if svc.FullName != "vrooli.proto_health.v1.notes.NotesService" {
			continue
		}
		for _, rpc := range svc.RPCs {
			if rpc.Name == "ListNotes" {
				sawRPC = true
				require.Equal(t, "vrooli.proto_health.v1.notes.ListNotesRequest", rpc.Input)
				require.Equal(t, TransportKindConnect, rpc.Transport)
			}
		}
	}
	require.True(t, sawRPC)

	require.Contains(t, surface.CrossScenarioImports, Import{
		FromFile:     "proto-health/v1/notes/notes.proto",
		ToFile:       "measures/v1/measures.proto",
		FromScenario: "proto-health",
		ToScenario:   "measures",
		FromPackage:  "vrooli.proto_health.v1.notes",
		ToPackage:    "vrooli.measures.v1",
		FromVersion:  "v1",
		ToVersion:    "v1",
		FromDomain:   "notes",
		ToDomain:     "measures",
		Kind:         ImportKindCrossScenario,
	})
	require.Contains(t, surface.RESTExceptions, RESTExceptionEndpoint{
		EndpointID:             "notes_attach",
		Path:                   "/api/v1/notes/{id}/attachments",
		Method:                 "POST",
		Domain:                 "notes",
		Reason:                 "multipart_upload",
		HasPayloadDeclarations: true,
	})
	require.Contains(t, surface.RESTExceptionPayloads, RESTExceptionPayloadRef{
		EndpointID:    "notes_attach",
		Path:          "/api/v1/notes/{id}/attachments",
		Method:        "POST",
		Domain:        "notes",
		Reason:        "multipart_upload",
		Role:          RESTPayloadRoleResponse,
		ProtoFullName: "vrooli.proto_health.v1.notes.UploadAttachmentResponse",
		Transport:     "json",
		Conformance:   "protojson",
		ProofStatus:   RESTPayloadProofNotEvaluated,
	})
	require.Contains(t, surface.RESTExceptionRefs, RESTExceptionRef{
		EndpointID: "notes_attach",
		Path:       "/api/v1/notes/{id}/attachments",
		Method:     "POST",
		Domain:     "notes",
		Message:    "UploadAttachmentResponse",
		FullName:   "vrooli.proto_health.v1.notes.UploadAttachmentResponse",
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
