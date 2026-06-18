package domains

import (
	"context"
	"errors"
	"path/filepath"
	"reflect"
	"testing"

	factsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/code-facts/v1/facts"
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

type failingResolver struct{}

func (failingResolver) ResolveScenarioURLDefault(context.Context, string) (string, error) {
	return "", errors.New("offline")
}
