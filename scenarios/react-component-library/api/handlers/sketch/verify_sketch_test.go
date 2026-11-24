package sketch

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	sharedv1 "github.com/vrooli/vrooli/packages/proto/gen/go/react-component-library/v1/shared"
	sketchv1 "github.com/vrooli/vrooli/packages/proto/gen/go/react-component-library/v1/sketch"
	"react-component-library/internal/availability"
	"react-component-library/internal/reconcile"
	internal "react-component-library/internal/sketch"
)

type emptyScenarioScanner struct{}

func (emptyScenarioScanner) ScanScenario(context.Context, string) ([]reconcile.ObservedFile, error) {
	return nil, nil
}

type failingScenarioScanner struct{ err error }

func (s failingScenarioScanner) ScanScenario(context.Context, string) ([]reconcile.ObservedFile, error) {
	return nil, s.err
}

func TestVerifySketchReportsRequiredDeclaredRegionsWithoutInventory(t *testing.T) {
	root := t.TempDir()
	page := filepath.Join(root, "scenarios", "demo", "experience", "pages", "home.json")
	require.NoError(t, os.MkdirAll(filepath.Dir(page), 0o700))
	require.NoError(t, os.WriteFile(page, []byte(`{"regions":[{"id":"hero","purpose":"Required hero region","component":{"library":{"component":"Hero","version":"1.0.0"}}}]}`), 0o600))

	h := NewConnectHandler(Deps{
		Store:      internal.NewStore(root),
		Reconciler: &reconcile.Resolver{ScenariosRoot: filepath.Join(root, "scenarios"), Scanner: emptyScenarioScanner{}},
	})
	response, err := h.VerifySketch(context.Background(), connect.NewRequest(&sketchv1.VerifySketchRequest{
		Target: &sharedv1.SketchTarget{Scenario: "demo", Page: "home"},
	}))
	require.NoError(t, err)
	require.NotNil(t, response.Msg.GetCoverage())
	require.Equal(t, int32(1), response.Msg.GetCoverage().GetMissing())
	require.Equal(t, "applicable", response.Msg.GetCoverage().GetStatus())
	require.Len(t, response.Msg.GetRegions(), 1)
	require.Equal(t, "hero", response.Msg.GetRegions()[0].GetRegion())
	require.False(t, response.Msg.GetRegions()[0].GetProven())
	require.NotEmpty(t, response.Msg.GetFindings())
}

func TestVerifySketchReportsMissingDependenciesAsUnavailable(t *testing.T) {
	h := NewConnectHandler(Deps{Store: internal.NewStore(t.TempDir())})
	_, err := h.VerifySketch(context.Background(), connect.NewRequest(&sketchv1.VerifySketchRequest{
		Target: &sharedv1.SketchTarget{Scenario: "demo", Page: "home"},
	}))
	require.Equal(t, connect.CodeUnimplemented, connect.CodeOf(err))
}

func TestVerifySketchRejectsInvalidTarget(t *testing.T) {
	h := NewConnectHandler(Deps{Store: internal.NewStore(t.TempDir()), Reconciler: &reconcile.Resolver{Scanner: emptyScenarioScanner{}}})
	_, err := h.VerifySketch(context.Background(), connect.NewRequest(&sketchv1.VerifySketchRequest{Target: &sharedv1.SketchTarget{Scenario: "", Page: "home"}}))
	require.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))
}

func TestVerifySketchSurfacesScannerFailure(t *testing.T) {
	root := t.TempDir()
	page := filepath.Join(root, "scenarios", "demo", "experience", "pages", "home.json")
	require.NoError(t, os.MkdirAll(filepath.Dir(page), 0o700))
	require.NoError(t, os.WriteFile(page, []byte(`{"regions":[]}`), 0o600))
	h := NewConnectHandler(Deps{
		Store:      internal.NewStore(root),
		Reconciler: &reconcile.Resolver{ScenariosRoot: filepath.Join(root, "scenarios"), Scanner: failingScenarioScanner{err: errors.New("inventory unavailable")}},
	})
	_, err := h.VerifySketch(context.Background(), connect.NewRequest(&sketchv1.VerifySketchRequest{
		Target: &sharedv1.SketchTarget{Scenario: "demo", Page: "home"},
	}))
	require.Equal(t, connect.CodeInternal, connect.CodeOf(err))
}

func TestVerifySketchSurfacesAvailabilityFailure(t *testing.T) {
	root := t.TempDir()
	page := filepath.Join(root, "scenarios", "demo", "experience", "pages", "home.json")
	require.NoError(t, os.MkdirAll(filepath.Dir(page), 0o700))
	require.NoError(t, os.WriteFile(page, []byte(`{"regions":[]}`), 0o600))
	h := NewConnectHandler(Deps{
		Store:        internal.NewStore(root),
		Reconciler:   &reconcile.Resolver{ScenariosRoot: filepath.Join(root, "scenarios"), Scanner: emptyScenarioScanner{}},
		Availability: func(context.Context) (*availability.Snapshot, error) { return nil, errors.New("catalog unavailable") },
	})
	_, err := h.VerifySketch(context.Background(), connect.NewRequest(&sketchv1.VerifySketchRequest{
		Target: &sharedv1.SketchTarget{Scenario: "demo", Page: "home"},
	}))
	require.Equal(t, connect.CodeInternal, connect.CodeOf(err))
}

func TestMapCandidateRegionsCreatesImmutableChildForDeclaredPort(t *testing.T) {
	root := t.TempDir()
	page := filepath.Join(root, "scenarios", "demo", "experience", "pages", "home.json")
	require.NoError(t, os.MkdirAll(filepath.Dir(page), 0o700))
	require.NoError(t, os.WriteFile(page, []byte(`{"regions":[{"id":"hero"}]}`), 0o600))
	store := internal.NewStore(root)
	base, err := store.Read("demo", "home")
	require.NoError(t, err)
	candidate, err := store.SaveCandidate("demo", "home", "landing", base.ContentHash, internal.Document{
		Regions: []internal.Region{{ID: "hero"}},
		Render:  &internal.RenderSettings{TemplateExport: "LandingPage", Regions: []internal.RenderRegion{{ID: "slot-main", TemplateRegion: "main", Slot: []string{"regions", "main"}}}},
	})
	require.NoError(t, err)
	writes := 0
	h := NewConnectHandler(Deps{Store: store, RecordWrite: func(context.Context) { writes++ }})
	response, err := h.MapCandidateRegions(context.Background(), connect.NewRequest(&sketchv1.MapCandidateRegionsRequest{
		Candidate: &sharedv1.CandidateReference{Scenario: "demo", DesignId: candidate.DesignID, Hash: candidate.Hash},
		Mappings:  []*sketchv1.CandidateRegionMapping{{Region: "hero", TemplateRegion: "main"}},
	}))
	require.NoError(t, err)
	require.Equal(t, "hero", response.Msg.GetSketch().GetRender().GetRegions()[0].GetId())
	require.NotEqual(t, candidate.Hash, response.Msg.GetCandidate().GetHash())
	require.Equal(t, 1, writes)
	_, err = store.ReadCandidate("demo", candidate.DesignID, candidate.Hash)
	require.NoError(t, err, "mapping must preserve its immutable parent candidate")
}

func TestMapCandidateRegionsRejectsUnknownPort(t *testing.T) {
	root := t.TempDir()
	page := filepath.Join(root, "scenarios", "demo", "experience", "pages", "home.json")
	require.NoError(t, os.MkdirAll(filepath.Dir(page), 0o700))
	require.NoError(t, os.WriteFile(page, []byte(`{"regions":[{"id":"hero"}]}`), 0o600))
	store := internal.NewStore(root)
	base, err := store.Read("demo", "home")
	require.NoError(t, err)
	candidate, err := store.SaveCandidate("demo", "home", "landing", base.ContentHash, internal.Document{Regions: []internal.Region{{ID: "hero"}}})
	require.NoError(t, err)
	h := NewConnectHandler(Deps{Store: store})
	_, err = h.MapCandidateRegions(context.Background(), connect.NewRequest(&sketchv1.MapCandidateRegionsRequest{
		Candidate: &sharedv1.CandidateReference{Scenario: "demo", DesignId: candidate.DesignID, Hash: candidate.Hash},
		Mappings:  []*sketchv1.CandidateRegionMapping{{Region: "hero", TemplateRegion: "unknown"}},
	}))
	require.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))
}
