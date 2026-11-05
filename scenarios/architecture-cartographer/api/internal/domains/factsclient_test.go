package domains

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"reflect"
	"testing"

	"connectrpc.com/connect"
	factsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/code-facts/v1/facts"
	factsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/code-facts/v1/facts/facts_v1connect"
)

func TestInventoryFromCodeFacts(t *testing.T) {
	report := &factsv1.CodeFactsReport{
		Surfaces: []*factsv1.Surface{{
			Id:     "api",
			Kind:   factsv1.SurfaceKind_SURFACE_KIND_API,
			Path:   "/repo/scenarios/demo/api",
			Status: factsv1.SurfaceStatus_SURFACE_STATUS_KNOWN,
		}},
		ParseUnits: []*factsv1.ParseUnit{{
			Id:         "api",
			Language:   "go",
			RootPath:   "/repo/scenarios/demo/api",
			ConfigPath: "/repo/scenarios/demo/api/go.mod",
			Status:     factsv1.EvidenceStatus_EVIDENCE_STATUS_PROVEN,
		}},
	}
	inv := inventoryFromCodeFacts(report)
	if !reflect.DeepEqual(inv.Surfaces, []Surface{{
		ID:     "api",
		Kind:   "api",
		Path:   "/repo/scenarios/demo/api",
		Status: "known",
	}}) {
		t.Fatalf("surfaces = %+v", inv.Surfaces)
	}
	if !reflect.DeepEqual(inv.ParseUnits, []ParseUnit{{
		ID:         "api",
		Language:   "go",
		RootPath:   "/repo/scenarios/demo/api",
		ConfigPath: "/repo/scenarios/demo/api/go.mod",
		Status:     "proven",
	}}) {
		t.Fatalf("parse units = %+v", inv.ParseUnits)
	}
}

func TestCodeFactsSurfaceProviderFallbackWarning(t *testing.T) {
	dir := t.TempDir()
	mkdirs(t, dir, "api/internal/graph")
	provider := NewCodeFactsSurfaceProvider(failingResolver{}, nil, NewLocalSurfaceProvider())
	inv, err := provider.Inspect(context.Background(), dir)
	if err != nil {
		t.Fatalf("inspect: %v", err)
	}
	if len(inv.Warnings) != 1 || inv.Warnings[0].Kind != "code_facts.unavailable" {
		t.Fatalf("warnings = %+v", inv.Warnings)
	}
	api, ok := surfaceByID(inv, "api")
	if !ok || api.Path != filepath.Join(dir, "api") {
		t.Fatalf("api surface = %+v ok=%v", api, ok)
	}
}

func TestCodeFactsSurfaceProviderRequestsSliceProofFamilies(t *testing.T) {
	var got []factsv1.FactFamily
	svc := captureFactsService{onDescribe: func(req *factsv1.DescribeCodeFactsRequest) {
		got = append([]factsv1.FactFamily(nil), req.GetInclude()...)
	}}
	mux := http.NewServeMux()
	path, handler := factsconnect.NewCodeFactsServiceHandler(svc)
	mux.Handle(path, handler)
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)

	provider := NewCodeFactsSurfaceProvider(staticResolver{url: server.URL}, server.Client(), NewLocalSurfaceProvider())
	_, err := provider.Inspect(context.Background(), t.TempDir())
	if err != nil {
		t.Fatalf("inspect: %v", err)
	}
	want := []factsv1.FactFamily{
		factsv1.FactFamily_FACT_FAMILY_SURFACES,
		factsv1.FactFamily_FACT_FAMILY_PARSE_UNITS,
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("include = %v, want %v", got, want)
	}
}

type failingResolver struct{}

func (failingResolver) ResolveScenarioURLDefault(context.Context, string) (string, error) {
	return "", errors.New("offline")
}

type staticResolver struct {
	url string
}

func (r staticResolver) ResolveScenarioURLDefault(context.Context, string) (string, error) {
	return r.url, nil
}

type captureFactsService struct {
	factsconnect.UnimplementedCodeFactsServiceHandler
	onDescribe func(*factsv1.DescribeCodeFactsRequest)
}

func (s captureFactsService) DescribeCodeFacts(_ context.Context, req *connect.Request[factsv1.DescribeCodeFactsRequest]) (*connect.Response[factsv1.CodeFactsReport], error) {
	if s.onDescribe != nil {
		s.onDescribe(req.Msg)
	}
	return connect.NewResponse(&factsv1.CodeFactsReport{}), nil
}
