package validation

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"

	internalplans "plan-manager/internal/plans"

	factsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/code-facts/v1/facts"
	factsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/code-facts/v1/facts/facts_v1connect"
)

type staticResolver struct {
	baseURL string
	err     error
}

func (r staticResolver) ResolveScenarioURLDefault(context.Context, string) (string, error) {
	if r.err != nil {
		return "", r.err
	}
	return r.baseURL, nil
}

type fakeCodeFactsService struct {
	report *factsv1.CodeFactsReport
	err    error
}

func (s fakeCodeFactsService) DescribeCodeFacts(_ context.Context, _ *connect.Request[factsv1.DescribeCodeFactsRequest]) (*connect.Response[factsv1.CodeFactsReport], error) {
	if s.err != nil {
		return nil, s.err
	}
	return connect.NewResponse(s.report), nil
}

func (s fakeCodeFactsService) DescribeFleetImports(context.Context, *connect.Request[factsv1.DescribeFleetImportsRequest]) (*connect.Response[factsv1.DescribeFleetImportsResponse], error) {
	return connect.NewResponse(&factsv1.DescribeFleetImportsResponse{}), nil
}

func (s fakeCodeFactsService) ListSurfaces(context.Context, *connect.Request[factsv1.ListSurfacesRequest]) (*connect.Response[factsv1.ListSurfacesResponse], error) {
	return connect.NewResponse(&factsv1.ListSurfacesResponse{}), nil
}

func (s fakeCodeFactsService) CheckProtoAdoption(context.Context, *connect.Request[factsv1.CheckProtoAdoptionRequest]) (*connect.Response[factsv1.ProofReport], error) {
	return connect.NewResponse(&factsv1.ProofReport{}), nil
}

func (s fakeCodeFactsService) CheckEndpointProof(context.Context, *connect.Request[factsv1.CheckEndpointProofRequest]) (*connect.Response[factsv1.ProofReport], error) {
	return connect.NewResponse(&factsv1.ProofReport{}), nil
}

func (s fakeCodeFactsService) GetCacheStatus(context.Context, *connect.Request[factsv1.GetCacheStatusRequest]) (*connect.Response[factsv1.CacheStatus], error) {
	return connect.NewResponse(&factsv1.CacheStatus{}), nil
}

func (s fakeCodeFactsService) InspectCache(context.Context, *connect.Request[factsv1.InspectCacheRequest]) (*connect.Response[factsv1.CacheStatus], error) {
	return connect.NewResponse(&factsv1.CacheStatus{}), nil
}

func (s fakeCodeFactsService) ClearCache(context.Context, *connect.Request[factsv1.ClearCacheRequest]) (*connect.Response[factsv1.ClearCacheResponse], error) {
	return connect.NewResponse(&factsv1.ClearCacheResponse{}), nil
}

func newCodeFactsTestServer(t *testing.T, svc fakeCodeFactsService) *httptest.Server {
	t.Helper()
	mux := http.NewServeMux()
	path, handler := factsconnect.NewCodeFactsServiceHandler(svc)
	mux.Handle(path, handler)
	return httptest.NewServer(mux)
}

// [REQ:PM-REF-001]
func TestCodeFactsReferenceResolverUsesConnectEvidence(t *testing.T) {
	root := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(root, "scenarios", "alpha"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(root, "scenarios", "alpha", "x.go"), []byte("package alpha\n"), 0o644))
	server := newCodeFactsTestServer(t, fakeCodeFactsService{report: &factsv1.CodeFactsReport{
		Target: &factsv1.TargetContext{RootPath: filepath.Join(root, "scenarios", "alpha", "x.go")},
		Evidence: []*factsv1.Evidence{{
			Status:   factsv1.EvidenceStatus_EVIDENCE_STATUS_PROVEN,
			Analyzer: "code-facts.target-resolver",
		}},
	}})
	defer server.Close()

	resolver := newCodeFactsReferenceResolver(root)
	resolver.resolver = staticResolver{baseURL: server.URL}
	got, err := resolver.Resolve(context.Background(), internalplans.Reference{
		Kind:   internalplans.ReferenceCode,
		Target: "scenarios/alpha/x.go",
	})
	require.NoError(t, err)
	require.Equal(t, internalplans.ResolutionResolved, got.Resolution)
	require.Equal(t, "resolved by code-facts", got.Note)
}

func TestCodeFactsReferenceResolverFallsBackWhenDependencyDown(t *testing.T) {
	root := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(root, "scenarios", "alpha"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(root, "scenarios", "alpha", "x.go"), []byte("package alpha\n"), 0o644))

	resolver := newCodeFactsReferenceResolver(root)
	resolver.resolver = staticResolver{err: errors.New("code-facts down")}
	got, err := resolver.Resolve(context.Background(), internalplans.Reference{
		Kind:   internalplans.ReferenceCode,
		Target: "scenarios/alpha/x.go",
	})
	require.NoError(t, err)
	require.Equal(t, internalplans.ResolutionResolved, got.Resolution)
	require.Contains(t, got.Note, "filesystem floor used")
}
