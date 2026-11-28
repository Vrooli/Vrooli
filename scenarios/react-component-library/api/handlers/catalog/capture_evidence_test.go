package catalog

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	apidb "github.com/vrooli/api-core/database"
	dbtest "github.com/vrooli/api-core/databasetest"
	catalogv1 "github.com/vrooli/vrooli/packages/proto/gen/go/react-component-library/v1/catalog"
	"react-component-library/internal/catalogcoverage"
	"react-component-library/internal/components"
	componentmocks "react-component-library/internal/components/mocks"
	"react-component-library/internal/componenttests"
)

func TestCaptureEvidenceRejectsOutOfRangeCatalogBatchOffset(t *testing.T) {
	repoRoot := t.TempDir()
	scenarioRoot := filepath.Join(repoRoot, "scenarios", "react-component-library")
	catalogRoot := filepath.Join(scenarioRoot, "catalog")
	require.NoError(t, os.MkdirAll(filepath.Join(catalogRoot, "assets"), 0o755))
	require.NoError(t, os.MkdirAll(filepath.Join(scenarioRoot, "library"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(catalogRoot, "config.json"), []byte(`{"domains":[]}`), 0o600))

	h := &handler{
		repoRoot:      repoRoot,
		captureRunner: &componenttests.Runner{Assets: captureDependencyReader{}, Executor: captureExecutor{}},
	}
	_, err := h.CaptureEvidence(context.Background(), connect.NewRequest(&catalogv1.CaptureEvidenceRequest{
		All: true, Offset: 1, Limit: 1,
	}))

	require.Error(t, err)
	require.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))
	require.Contains(t, err.Error(), "outside catalog range 0..0")
	require.NoFileExists(t, filepath.Join(scenarioRoot, "captures", "catalog", "all", "capture-manifest.json"))
}

func TestCaptureEvidenceRejectsAssetIDPathSegments(t *testing.T) {
	repoRoot := t.TempDir()
	scenarioRoot := filepath.Join(repoRoot, "scenarios", "react-component-library")
	h := &handler{
		repoRoot:      repoRoot,
		captureRunner: &componenttests.Runner{Assets: captureDependencyReader{}, Executor: captureExecutor{}},
	}

	_, err := h.CaptureEvidence(context.Background(), connect.NewRequest(&catalogv1.CaptureEvidenceRequest{
		AssetId: "../outside",
	}))

	require.Error(t, err)
	require.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))
	require.NoDirExists(t, filepath.Join(scenarioRoot, "captures"))
}

func TestCaptureEvidenceReportsAssetsWithoutImplementations(t *testing.T) {
	repoRoot := t.TempDir()
	scenarioRoot := filepath.Join(repoRoot, "scenarios", "react-component-library")
	catalogRoot := filepath.Join(scenarioRoot, "catalog")
	assetsDir := filepath.Join(catalogRoot, "assets", "controls")
	require.NoError(t, os.MkdirAll(assetsDir, 0o755))
	require.NoError(t, os.MkdirAll(filepath.Join(scenarioRoot, "library"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(catalogRoot, "config.json"), []byte(`{"domains":[{"id":"controls","order":10}]}`), 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(assetsDir, "future-control.json"), []byte(`{
		"kind":"catalog-asset",
		"asset":{"id":"react-component-library:future-control","name":"Future Control","kind":"component","domain":"controls","targets":["react-vite"]}
	}`), 0o600))
	h := &handler{
		repoRoot:      repoRoot,
		captureRunner: &componenttests.Runner{Assets: captureDependencyReader{}, Executor: captureExecutor{}},
	}

	response, err := h.CaptureEvidence(context.Background(), connect.NewRequest(&catalogv1.CaptureEvidenceRequest{
		AssetId: "react-component-library:future-control",
	}))

	require.NoError(t, err)
	require.Equal(t, "react-component-library:future-control", response.Msg.GetAssetId())
	require.Equal(t, []string{"react-component-library:future-control"}, response.Msg.GetMissingContractAssets())
	require.Zero(t, response.Msg.GetRowsWritten())
	require.True(t, response.Msg.GetComplete())
	require.FileExists(t, filepath.Join(response.Msg.GetCaptureDirectory(), "capture-manifest.json"))
}

type successfulCaptureExecutor struct{}

func (successfulCaptureExecutor) ExecuteStory(context.Context, string, string, string) (componenttests.StoryExecution, error) {
	return componenttests.StoryExecution{
		Passed: true, AccessibilityJSON: `{"status":"complete"}`,
		Artifacts: []componenttests.Artifact{{Kind: "screenshot", Reference: "capture://card/default"}},
	}, nil
}

func TestCaptureEvidenceRunsPublishedStoriesAndPersistsCaptureAndVisualEvidence(t *testing.T) {
	ctx := context.Background()
	repoRoot := t.TempDir()
	scenarioRoot := filepath.Join(repoRoot, "scenarios", "react-component-library")
	catalogRoot := filepath.Join(scenarioRoot, "catalog")
	assetDir := filepath.Join(catalogRoot, "assets", "controls")
	libraryRoot := filepath.Join(scenarioRoot, "library")
	versionDir := filepath.Join(libraryRoot, "components", "Card", "versions", "1.0.0")
	inputDir := filepath.Join(libraryRoot, "components", "Input", "versions", "1.0.0")
	require.NoError(t, os.MkdirAll(assetDir, 0o755))
	require.NoError(t, os.MkdirAll(versionDir, 0o755))
	require.NoError(t, os.MkdirAll(inputDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(catalogRoot, "config.json"), []byte(`{"domains":[{"id":"controls","order":1}]}`), 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(assetDir, "card.json"), []byte(`{"kind":"catalog-asset","asset":{"id":"controls.card","name":"Card","kind":"component","domain":"controls","targets":["react-vite"]}}`), 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(assetDir, "input.json"), []byte(`{"kind":"catalog-asset","asset":{"id":"controls.input","name":"Input","kind":"component","domain":"controls","targets":["react-vite"]}}`), 0o600))
	manifest := `{"libraryId":"react-component-library:Card","catalogId":"controls.card","latest":"1.0.0","kind":"component"}`
	require.NoError(t, os.WriteFile(filepath.Join(libraryRoot, "components", "Card", "component.json"), []byte(manifest), 0o600))
	source := `/**
 * @libraryId react-component-library:Card
 * @version 1.0.0
 */
export const Card = () => <article>Card</article>;
`
	require.NoError(t, os.WriteFile(filepath.Join(versionDir, "Card.tsx"), []byte(source), 0o600))
	story := `{"schemaVersion":5,"kind":"component","args":{"fields":[]},"environment":{"fixtures":[]},"stories":[{"id":"default","name":"Default","role":"anatomy","args":{}}]}`
	require.NoError(t, os.WriteFile(filepath.Join(versionDir, "story.json"), []byte(story), 0o600))
	inputManifest := `{"libraryId":"react-component-library:Input","catalogId":"controls.input","latest":"1.0.0","kind":"component"}`
	require.NoError(t, os.WriteFile(filepath.Join(libraryRoot, "components", "Input", "component.json"), []byte(inputManifest), 0o600))
	inputSource := strings.ReplaceAll(source, "Card", "Input")
	require.NoError(t, os.WriteFile(filepath.Join(inputDir, "Input.tsx"), []byte(inputSource), 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(inputDir, "story.json"), []byte(story), 0o600))

	repository := componentmocks.NewFakeRepository()
	indexer := components.NewIndexer(repository, libraryRoot, nil)
	_, err := indexer.IndexManifest(ctx, "components/Card/component.json")
	require.NoError(t, err)
	_, err = indexer.IndexManifest(ctx, "components/Input/component.json")
	require.NoError(t, err)
	assets := components.NewService(repository)
	database := dbtest.NewSQLite(t)
	require.NoError(t, apidb.EnsureSchemas(ctx, database, apidb.SchemaProviderFunc(catalogcoverage.Schema)))
	store := catalogcoverage.NewEvidenceStore(database)
	h := &handler{
		repoRoot: repoRoot, evidence: store,
		captureRunner: &componenttests.Runner{Assets: assets, Stories: assets, Executor: successfulCaptureExecutor{}},
	}

	first, err := h.CaptureEvidence(ctx, connect.NewRequest(&catalogv1.CaptureEvidenceRequest{All: true, Offset: 0, Limit: 1}))
	require.NoError(t, err)
	require.Equal(t, "all", first.Msg.GetAssetId())
	require.False(t, first.Msg.GetComplete())
	require.EqualValues(t, 1, first.Msg.GetNextOffset())
	require.Empty(t, first.Msg.GetMissingContractAssets())
	require.EqualValues(t, 1, first.Msg.GetRowsWritten())

	second, err := h.CaptureEvidence(ctx, connect.NewRequest(&catalogv1.CaptureEvidenceRequest{All: true, Offset: 1, Limit: 1}))
	require.NoError(t, err)
	require.True(t, second.Msg.GetComplete())
	require.EqualValues(t, 2, second.Msg.GetNextOffset())
	require.Empty(t, second.Msg.GetMissingContractAssets())
	require.EqualValues(t, 1, second.Msg.GetRowsWritten())
	manifestPath := filepath.Join(second.Msg.GetCaptureDirectory(), "capture-manifest.json")
	require.FileExists(t, manifestPath)
	manifestBytes, err := os.ReadFile(manifestPath)
	require.NoError(t, err)
	require.Contains(t, string(manifestBytes), `"result": "passed"`)
	require.Contains(t, string(manifestBytes), "capture://card/default")
	require.Contains(t, string(manifestBytes), "controls.input")

	persisted, err := store.List(ctx)
	require.NoError(t, err)
	require.Len(t, persisted, 2)
	ids := map[string]bool{}
	for _, row := range persisted {
		ids[row.AssetID] = true
		require.Equal(t, "visual", row.Gate)
		require.Equal(t, "pass", row.Result)
		require.Contains(t, row.MeasurementJSON, "browser-automation-studio.ux-capture.v1")
	}
	require.True(t, ids["controls.card"])
	require.True(t, ids["controls.input"])
}
