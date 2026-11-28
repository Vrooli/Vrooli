package sketch

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	shared "github.com/vrooli/vrooli/packages/proto/gen/go/react-component-library/v1/shared"
	sketchv1 "github.com/vrooli/vrooli/packages/proto/gen/go/react-component-library/v1/sketch"
	"react-component-library/internal/availability"
	"react-component-library/internal/catalogcoverage"
	"react-component-library/internal/components"
	componentmocks "react-component-library/internal/components/mocks"
	internal "react-component-library/internal/sketch"
)

func TestRefineCandidateRequiresSketchAndRecordsOnlyARealRefinement(t *testing.T) {
	root := t.TempDir()
	pagePath := filepath.Join(root, "scenarios", "demo", "experience", "pages", "home.json")
	require.NoError(t, os.MkdirAll(filepath.Dir(pagePath), 0o700))
	require.NoError(t, os.WriteFile(pagePath, []byte(`{"regions":[]}`), 0o600))
	store := internal.NewStore(root)
	base, err := store.Read("demo", "home")
	require.NoError(t, err)
	original := internal.Document{Regions: []internal.Region{{ID: "hero", Elements: []string{"heading"}}}}
	candidate, err := store.SaveCandidate("demo", "home", "refinement-test", base.ContentHash, original)
	require.NoError(t, err)

	writes := 0
	h := NewConnectHandler(Deps{Store: store, RecordWrite: func(context.Context) { writes++ }})
	_, err = h.RefineCandidate(context.Background(), connect.NewRequest(&sketchv1.RefineCandidateRequest{}))
	require.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))
	require.Zero(t, writes)

	ref := &shared.CandidateReference{Scenario: "demo", DesignId: candidate.DesignID, Hash: candidate.Hash}
	unchanged, err := h.RefineCandidate(context.Background(), connect.NewRequest(&sketchv1.RefineCandidateRequest{
		Candidate: ref, Sketch: toProto(original), RequestedRegions: []string{"hero"}, Reason: "Keep the reviewed hero layout",
	}))
	require.NoError(t, err)
	require.Equal(t, "unchanged", unchanged.Msg.GetStatus())
	require.Zero(t, writes, "no-op refinements must not record a write")

	refinedDoc := internal.Document{Regions: []internal.Region{{ID: "hero", Elements: []string{"heading", "supporting-copy"}}}}
	refined, err := h.RefineCandidate(context.Background(), connect.NewRequest(&sketchv1.RefineCandidateRequest{
		Candidate: ref, Sketch: toProto(refinedDoc), RequestedRegions: []string{"hero"}, Reason: "Add supporting copy to the hero",
	}))
	require.NoError(t, err)
	require.Equal(t, "refined", refined.Msg.GetStatus())
	require.EqualValues(t, 1, refined.Msg.GetRound())
	require.EqualValues(t, 3, refined.Msg.GetBudget())
	require.Equal(t, 1, writes)
	require.NotEqual(t, candidate.Hash, refined.Msg.GetResult().GetCandidate().GetHash())
}

func TestPlaceCandidateAssetValidatesAvailabilityBeforeResolvingPublishedSources(t *testing.T) {
	root := t.TempDir()
	pagePath := filepath.Join(root, "scenarios", "demo", "experience", "pages", "home.json")
	require.NoError(t, os.MkdirAll(filepath.Dir(pagePath), 0o700))
	require.NoError(t, os.WriteFile(pagePath, []byte(`{"regions":[]}`), 0o600))
	store := internal.NewStore(root)
	base, err := store.Read("demo", "home")
	require.NoError(t, err)
	candidate, err := store.SaveCandidate("demo", "home", "placement-test", base.ContentHash, internal.Document{})
	require.NoError(t, err)
	ref := &shared.CandidateReference{Scenario: "demo", DesignId: candidate.DesignID, Hash: candidate.Hash}

	withoutDependencies := NewConnectHandler(Deps{Store: store})
	_, err = withoutDependencies.PlaceCandidateAsset(context.Background(), connect.NewRequest(&sketchv1.PlaceCandidateAssetRequest{Candidate: ref}))
	require.Equal(t, connect.CodeUnavailable, connect.CodeOf(err))

	withDependencies := NewConnectHandler(Deps{
		Store: store,
		Availability: func(context.Context) (*availability.Snapshot, error) {
			return &availability.Snapshot{}, nil
		},
		Catalog: func(context.Context) ([]catalogcoverage.Asset, error) { return nil, nil },
		Assets:  components.NewService(componentmocks.NewFakeRepository()),
	})
	_, err = withDependencies.PlaceCandidateAsset(context.Background(), connect.NewRequest(&sketchv1.PlaceCandidateAssetRequest{Candidate: ref}))
	require.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err), "exact version is required before any published source lookup")
}

type candidatePublishedAssets struct {
	asset   components.Component
	version components.ComponentVersion
}

func (c candidatePublishedAssets) Get(_ context.Context, id string) (components.Component, error) {
	if id == c.asset.ID || id == c.asset.CatalogID || id == c.asset.LibraryID {
		return c.asset, nil
	}
	return components.Component{}, os.ErrNotExist
}

func (c candidatePublishedAssets) GetByLibraryID(ctx context.Context, id string) (components.Component, error) {
	return c.Get(ctx, id)
}

func (c candidatePublishedAssets) GetVersion(_ context.Context, componentID, version string) (components.ComponentVersion, error) {
	if componentID != c.asset.ID || version != c.version.Version {
		return components.ComponentVersion{}, os.ErrNotExist
	}
	return c.version, nil
}

func (c candidatePublishedAssets) List(context.Context, components.SearchQuery) ([]components.Component, error) {
	return []components.Component{c.asset}, nil
}

func TestPlaceCandidateAssetCreatesImmutableCandidateFromPublishedStory(t *testing.T) {
	root := t.TempDir()
	pagePath := filepath.Join(root, "scenarios", "demo", "experience", "pages", "home.json")
	require.NoError(t, os.MkdirAll(filepath.Dir(pagePath), 0o700))
	require.NoError(t, os.WriteFile(pagePath, []byte(`{"regions":[]}`), 0o600))
	store := internal.NewStore(root)
	base, err := store.Read("demo", "home")
	require.NoError(t, err)
	doc := internal.Document{
		Template: &internal.AssetRef{Asset: "templates.page", Version: "1.0.0"},
		Regions:  []internal.Region{{ID: "records"}},
		Render: &internal.RenderSettings{
			TemplateExport: "Page",
			Regions:        []internal.RenderRegion{{ID: "records", TemplateRegion: "body", Slot: []string{"regions", "body"}}},
			Bindings:       map[string]any{"$template": map[string]any{}},
		},
	}
	candidate, err := store.SaveCandidate("demo", "home", "asset-placement", base.ContentHash, doc)
	require.NoError(t, err)

	story := `{"schemaVersion":5,"kind":"component","args":{"fields":[]},"environment":{"fixtures":[]},"stories":[{"id":"ready","name":"Ready","role":"anatomy","args":{}}]}`
	storyHash := sha256.Sum256([]byte(story))
	source := "export const Card = () => null;"
	sourceHash := sha256.Sum256([]byte(source))
	asset := components.Component{ID: "card-row", CatalogID: "data-display.card", LibraryID: "react-component-library:Card", Slug: "Card", LatestVersion: "1.0.0"}
	version := components.ComponentVersion{
		ComponentID: "card-row", Version: "1.0.0", Status: components.VersionStatusReleased,
		DependencyLockPresent: true, SourcePath: "Card.tsx", Content: source, ContentSHA256: hex.EncodeToString(sourceHash[:]),
		Files: []components.ComponentVersionFile{{Path: "story.json", Content: story, ContentSHA256: hex.EncodeToString(storyHash[:])}},
	}
	published := candidatePublishedAssets{asset: asset, version: version}
	snapshot, err := availability.NewSnapshot(context.Background(), published, func(context.Context, string, string) (string, error) { return "build-hash", nil })
	require.NoError(t, err)
	writes := 0
	h := NewConnectHandler(Deps{
		Store:        store,
		Availability: func(context.Context) (*availability.Snapshot, error) { return snapshot, nil },
		Catalog: func(context.Context) ([]catalogcoverage.Asset, error) {
			return []catalogcoverage.Asset{
				{ID: "templates.page", Kind: "page-template", RegionAccepts: map[string]string{"body": "data-display"}},
				{ID: "data-display.card", Kind: "component", Domain: "data-display"},
			}, nil
		},
		Assets:      published,
		RecordWrite: func(context.Context) { writes++ },
	})
	placed, err := h.PlaceCandidateAsset(context.Background(), connect.NewRequest(&sketchv1.PlaceCandidateAssetRequest{
		Candidate: &shared.CandidateReference{Scenario: "demo", DesignId: candidate.DesignID, Hash: candidate.Hash},
		Region:    "records", Asset: &shared.AssetReference{Asset: "data-display.card", Version: "1.0.0"}, Story: "ready",
	}))
	require.NoError(t, err)
	require.Equal(t, 1, writes)
	require.NotEqual(t, candidate.Hash, placed.Msg.GetCandidate().GetHash())
	require.Equal(t, "data-display.card", placed.Msg.GetSketch().GetPlacements()[0].GetFills().GetAsset())
	require.Equal(t, "Card", placed.Msg.GetSketch().GetRender().GetRegions()[0].GetExport())
	require.Equal(t, "ready", placed.Msg.GetSketch().GetRender().GetRegions()[0].GetStory())
	current, err := store.Read("demo", "home")
	require.NoError(t, err)
	require.Equal(t, base.ContentHash, current.ContentHash, "placing an asset in a candidate must not mutate the active page")
}
