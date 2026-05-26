package manifest

import (
	"context"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"

	manifestv1 "github.com/vrooli/vrooli/packages/proto/gen/go/architecture-cartographer/v1/manifest"
	manifestconnect "github.com/vrooli/vrooli/packages/proto/gen/go/architecture-cartographer/v1/manifest/manifest_v1connect"

	"github.com/vrooli/cli-core/cliapp"
	cliapptest "github.com/vrooli/cli-core/cliapptest"

	clitest "architecture-cartographer/cli/internal/testutil"
)

type fakeService struct {
	manifestconnect.UnimplementedManifestServiceHandler

	mu           sync.Mutex
	validateReqs []*manifestv1.ValidateManifestRequest
	validateResp *manifestv1.ValidateManifestResponse
	getResp      *manifestv1.GetManifestResponse
	domainsResp  *manifestv1.ListDomainsResponse
}

func (s *fakeService) ValidateManifest(_ context.Context, req *connect.Request[manifestv1.ValidateManifestRequest]) (*connect.Response[manifestv1.ValidateManifestResponse], error) {
	s.mu.Lock()
	s.validateReqs = append(s.validateReqs, req.Msg)
	s.mu.Unlock()
	return connect.NewResponse(s.validateResp), nil
}

func (s *fakeService) GetManifest(_ context.Context, _ *connect.Request[manifestv1.GetManifestRequest]) (*connect.Response[manifestv1.GetManifestResponse], error) {
	return connect.NewResponse(s.getResp), nil
}

func (s *fakeService) ListDomains(_ context.Context, _ *connect.Request[manifestv1.ListDomainsRequest]) (*connect.Response[manifestv1.ListDomainsResponse], error) {
	return connect.NewResponse(s.domainsResp), nil
}

func connectAPI(t *testing.T, svc *fakeService) http.Handler {
	t.Helper()
	path, handler := manifestconnect.NewManifestServiceHandler(svc)
	mux := http.NewServeMux()
	mux.Handle(path, handler)
	return mux
}

func TestValidate_SubmitsFileBytesAndRendersGate(t *testing.T) {
	svc := &fakeService{validateResp: &manifestv1.ValidateManifestResponse{
		Valid: false,
		Diagnostics: []*manifestv1.Diagnostic{
			{Severity: manifestv1.DiagnosticSeverity_DIAGNOSTIC_SEVERITY_ERROR, Code: "MANIFEST_UNKNOWN_DOMAIN", Message: "unknown domain", Path: "domains[0]"},
		},
	}}
	core := clitest.NewTestApp(t, connectAPI(t, svc))
	h := newHandlers(core)

	dir := t.TempDir()
	file := filepath.Join(dir, "manifest.yaml")
	require.NoError(t, os.WriteFile(file, []byte("scenario: demo\n"), 0o644))

	ctx, out := cliapptest.NewCapturedRunContext(core, validateSchema(), cliapptest.TestRunContextOptions{
		Positionals: map[string]string{"scenario": "demo", "file": file},
	})

	require.NoError(t, h.validate(ctx))
	require.Len(t, svc.validateReqs, 1)
	require.Equal(t, []byte("scenario: demo\n"), svc.validateReqs[0].GetSource())
	require.Equal(t, "application/yaml", svc.validateReqs[0].GetContentType())

	body := out.String()
	require.Contains(t, body, "INVALID")
	require.Contains(t, body, "MANIFEST_UNKNOWN_DOMAIN")
}

func TestValidate_NoFileLeavesSourceEmpty(t *testing.T) {
	svc := &fakeService{validateResp: &manifestv1.ValidateManifestResponse{Valid: true}}
	core := clitest.NewTestApp(t, connectAPI(t, svc))
	h := newHandlers(core)
	ctx, out := cliapptest.NewCapturedRunContext(core, validateSchema(), cliapptest.TestRunContextOptions{
		Positionals: map[string]string{"scenario": "demo"},
	})

	require.NoError(t, h.validate(ctx))
	require.Len(t, svc.validateReqs, 1)
	require.Empty(t, svc.validateReqs[0].GetSource())
	require.Contains(t, out.String(), "VALID")
}

func TestListDomains_RendersTable(t *testing.T) {
	svc := &fakeService{domainsResp: &manifestv1.ListDomainsResponse{
		Domains: []*manifestv1.DomainSpec{
			{Name: "graph", Paths: []string{"api/internal/graph/**"}, AllowedDependencies: []string{"manifest"}},
		},
	}}
	core := clitest.NewTestApp(t, connectAPI(t, svc))
	h := newHandlers(core)
	ctx, out := cliapptest.NewCapturedRunContext(core, cliapp.ArgSchema{
		Positionals: []cliapp.Positional{{Name: "scenario", Required: true}},
	}, cliapptest.TestRunContextOptions{
		Positionals: map[string]string{"scenario": "demo"},
	})

	require.NoError(t, h.listDomains(ctx))
	body := out.String()
	require.Contains(t, body, "1 declared domain(s)")
	require.Contains(t, body, "graph")
}

func validateSchema() cliapp.ArgSchema {
	return cliapp.ArgSchema{
		Positionals: []cliapp.Positional{
			{Name: "scenario", Required: true},
			{Name: "file"},
		},
	}
}
