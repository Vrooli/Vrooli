package protosurface

import (
	"os"
	"path/filepath"
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
	var sawTemplate bool
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
				if a.Name == "template" && a.Value == "react-vite/example" {
					sawTemplate = true
				}
			}
		}
	}
	require.True(t, sawNotes)
	require.True(t, sawStability)
	require.True(t, sawTemplate)

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
		FromFile:   "proto-health/v1/notes/notes.proto",
		ToFile:     "measures/v1/measures.proto",
		FromDomain: "notes",
	})
	require.Contains(t, surface.AdoptionSignals, AdoptionSignal{
		Name:    "api_go_mod_replace",
		Present: true,
		Detail:  "api/go.mod references the shared packages/proto module",
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
