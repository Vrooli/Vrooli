package manifest_test

import (
	"context"
	"net/http"
	"testing"

	manifesth "architecture-cartographer/handlers/manifest"
	"architecture-cartographer/internal/manifest"
	"architecture-cartographer/internal/manifest/mocks"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	manifestv1 "github.com/vrooli/vrooli/packages/proto/gen/go/architecture-cartographer/v1/manifest"
	"github.com/vrooli/vrooli/packages/proto/gen/go/architecture-cartographer/v1/manifest/manifest_v1connect"
)

const yamlBody = `
manifest_version: v1
scenario: demo
domains:
  - name: graph
    paths:
      - api/internal/graph/**
`

func newHandler(t *testing.T) (*manifesth.Handler, *mocks.FakeRepository) {
	t.Helper()
	repo := &mocks.FakeRepository{}
	svc := manifest.NewService(repo)
	return manifesth.NewHandler(svc), repo
}

func TestHandler_ValidateManifest_HappyPath(t *testing.T) {
	h, repo := newHandler(t)
	resp, err := h.ValidateManifest(context.Background(), connect.NewRequest(&manifestv1.ValidateManifestRequest{
		Scenario:    "demo",
		Source:      []byte(yamlBody),
		ContentType: "application/yaml",
	}))
	require.NoError(t, err)
	require.True(t, resp.Msg.GetValid(), "expected valid manifest; diags=%+v", resp.Msg.GetDiagnostics())
	require.Equal(t, "demo", resp.Msg.GetManifest().GetScenario())
	require.Equal(t, int64(1), repo.SaveCalls.Load())
}

func TestHandler_ValidateManifest_RejectsMissingScenario(t *testing.T) {
	h, _ := newHandler(t)
	_, err := h.ValidateManifest(context.Background(), connect.NewRequest(&manifestv1.ValidateManifestRequest{
		Source: []byte(yamlBody),
	}))
	require.Error(t, err)
	var ce *connect.Error
	require.ErrorAs(t, err, &ce)
	require.Equal(t, connect.CodeInvalidArgument, ce.Code())
}

func TestHandler_ValidateManifest_DiagnosticsOnInvalid(t *testing.T) {
	h, _ := newHandler(t)
	// Unknown allowed_dependencies reference produces a structural error.
	bad := `
manifest_version: v1
scenario: demo
domains:
  - name: graph
    paths: ["api/internal/graph/**"]
    allowed_dependencies: [nope]
`
	resp, err := h.ValidateManifest(context.Background(), connect.NewRequest(&manifestv1.ValidateManifestRequest{
		Scenario:    "demo",
		Source:      []byte(bad),
		ContentType: "application/yaml",
	}))
	require.Error(t, err, "invalid manifest should surface a Connect error")
	var ce *connect.Error
	require.ErrorAs(t, err, &ce)
	require.Equal(t, connect.CodeInvalidArgument, ce.Code())
	require.NotNil(t, resp, "envelope should still carry diagnostics")
	require.False(t, resp.Msg.GetValid())
	require.NotEmpty(t, resp.Msg.GetDiagnostics())
}

func TestHandler_GetManifest_NotFound(t *testing.T) {
	h, _ := newHandler(t)
	_, err := h.GetManifest(context.Background(), connect.NewRequest(&manifestv1.GetManifestRequest{Scenario: "missing"}))
	require.Error(t, err)
	var ce *connect.Error
	require.ErrorAs(t, err, &ce)
	require.Equal(t, connect.CodeNotFound, ce.Code())
}

func TestHandler_ListDomains_AfterValidate(t *testing.T) {
	h, _ := newHandler(t)
	_, err := h.ValidateManifest(context.Background(), connect.NewRequest(&manifestv1.ValidateManifestRequest{
		Scenario:    "demo",
		Source:      []byte(yamlBody),
		ContentType: "application/yaml",
	}))
	require.NoError(t, err)

	resp, err := h.ListDomains(context.Background(), connect.NewRequest(&manifestv1.ListDomainsRequest{Scenario: "demo"}))
	require.NoError(t, err)
	require.Len(t, resp.Msg.GetDomains(), 1)
	require.Equal(t, "graph", resp.Msg.GetDomains()[0].GetName())
}

func TestHandler_InterfaceSatisfied(t *testing.T) {
	var _ manifest_v1connect.ManifestServiceHandler = (*manifesth.Handler)(nil)
}

// Sanity-check the no-op fallback for the unused Content-Type detection.
func TestHandler_ValidateManifest_AcceptsUnsetContentType(t *testing.T) {
	h, _ := newHandler(t)
	resp, err := h.ValidateManifest(context.Background(), connect.NewRequest(&manifestv1.ValidateManifestRequest{
		Scenario: "demo",
		Source:   []byte(yamlBody),
	}))
	require.NoError(t, err)
	require.True(t, resp.Msg.GetValid())
}

func TestHandler_ValidateManifest_RejectsEmptySource(t *testing.T) {
	h, _ := newHandler(t)
	_, err := h.ValidateManifest(context.Background(), connect.NewRequest(&manifestv1.ValidateManifestRequest{
		Scenario: "demo",
	}))
	require.Error(t, err)
	var ce *connect.Error
	require.ErrorAs(t, err, &ce)
	require.Equal(t, connect.CodeInvalidArgument, ce.Code())
}

// Avoid an unused-import lint when http isn't otherwise referenced.
var _ = http.MethodPost
