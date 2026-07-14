package components_test

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"react-component-library/internal/components"
	"react-component-library/internal/components/mocks"
)

func TestService_UpsertRejectsBlankLibraryID(t *testing.T) {
	repo := mocks.NewFakeRepository()
	svc := components.NewService(repo)

	_, err := svc.Upsert(context.Background(), components.UpsertInput{LibraryID: "   "})
	var bad components.ErrInvalidHeader
	require.True(t, errors.As(err, &bad))
	require.Equal(t, int64(0), repo.UpsertCalls.Load())
}

func TestService_ListAppliesDefaultLimit(t *testing.T) {
	repo := mocks.NewFakeRepository()
	svc := components.NewService(repo)
	ctx := context.Background()

	_, err := svc.Upsert(ctx, components.UpsertInput{LibraryID: "a"})
	require.NoError(t, err)
	_, err = svc.Upsert(ctx, components.UpsertInput{LibraryID: "b"})
	require.NoError(t, err)

	got, err := svc.List(ctx, components.SearchQuery{Limit: 0})
	require.NoError(t, err)
	require.Len(t, got, 2, "default limit should fetch all seeded rows")
}

func TestService_GetByLibraryIDPropagatesNotFound(t *testing.T) {
	repo := mocks.NewFakeRepository()
	svc := components.NewService(repo)
	_, err := svc.GetByLibraryID(context.Background(), "missing")
	var nf components.ErrComponentNotFound
	require.True(t, errors.As(err, &nf))
}

func TestService_ValidateStyleFitFoldsAffinityVerdicts(t *testing.T) {
	tests := []struct {
		name     string
		style    string
		wantKind components.StyleFitVerdictKind
		wantAff  components.DesignAffinity
		wantText string
	}{
		{
			name:     "native is ok",
			style:    "vrooli-default",
			wantKind: components.StyleFitVerdictOK,
			wantAff:  components.DesignAffinityNative,
			wantText: "token-native baseline",
		},
		{
			name:     "compatible is ok with info detail",
			style:    "vrooli-conversion-landing",
			wantKind: components.StyleFitVerdictOK,
			wantAff:  components.DesignAffinityCompatible,
			wantText: "compatible",
		},
		{
			name:     "discouraged is warn",
			style:    "vrooli-data-dense",
			wantKind: components.StyleFitVerdictWarn,
			wantAff:  components.DesignAffinityDiscouraged,
			wantText: "too sparse",
		},
		{
			name:     "undeclared style is info",
			style:    "vrooli-editorial",
			wantKind: components.StyleFitVerdictInfo,
			wantText: "declares no affinity",
		},
		{
			name:     "missing scenario style is warn",
			style:    "",
			wantKind: components.StyleFitVerdictWarn,
			wantText: "does not declare generation.design.id",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := mocks.NewFakeRepository()
			c := seedStyleFitComponent(t, repo)
			svc := components.NewServiceWithScenarioReader(repo, serviceJSONReaderFunc(func(context.Context, string) ([]byte, error) {
				if tt.style == "" {
					return []byte(`{"generation":{"design":{}}}`), nil
				}
				return []byte(`{"generation":{"design":{"id":"` + tt.style + `"}}}`), nil
			}))

			got, err := svc.ValidateStyleFit(context.Background(), c.ID, "1.0.0", "demo")
			require.NoError(t, err)
			require.Equal(t, tt.wantKind, got.Kind)
			require.Equal(t, tt.wantAff, got.Affinity)
			require.Equal(t, c.ID, got.ComponentID)
			require.Equal(t, "1.0.0", got.Version)
			require.Equal(t, "demo", got.Scenario)
			require.Equal(t, tt.style, got.ScenarioStyle)
			require.Contains(t, got.Detail, tt.wantText)
		})
	}
}

func TestService_ValidateStyleFitRequiresScenarioReader(t *testing.T) {
	repo := mocks.NewFakeRepository()
	c := seedStyleFitComponent(t, repo)
	svc := components.NewService(repo)

	_, err := svc.ValidateStyleFit(context.Background(), c.ID, "1.0.0", "demo")
	require.ErrorContains(t, err, "service.json reader not configured")
}

func TestFSServiceJSONReaderGuardsTraversal(t *testing.T) {
	reader := components.NewFSServiceJSONReader(t.TempDir())
	_, err := reader.Read(context.Background(), "../demo")
	require.ErrorContains(t, err, "invalid scenario name")
}

func TestService_IngestComponentCreatesIndexedDraftAndReportsFindings(t *testing.T) {
	repo := mocks.NewFakeRepository()
	root := t.TempDir()
	svc := components.NewServiceWithContent(repo, components.NewFSContentStore(root))
	components.SetScenarioSourceReader(svc, scenarioSourceReaderFunc(func(_ context.Context, scenario, sourceFile string) ([]byte, error) {
		require.Equal(t, "web-console", scenario)
		require.Equal(t, "ui/src/components/DrawerShell.tsx", sourceFile)
		return []byte(`import { useNavigate } from "react-router-dom";
export default function DrawerShell() { const navigate = useNavigate(); return <div className="bg-red-500" onClick={() => navigate("/")} />; }`), nil
	}))

	got, err := svc.IngestComponent(context.Background(), components.IngestComponentInput{
		Scenario: "web-console", SourceFile: "ui/src/components/DrawerShell.tsx", Slug: "drawer-shell",
		DisplayName: "Drawer Shell", Tags: []string{"overlay"},
	})
	require.NoError(t, err)
	require.Equal(t, "0.1.0-draft.1", got.DraftVersion)
	require.Equal(t, "react-component-library:drawer-shell", got.Component.LibraryID)
	require.Equal(t, got.DraftVersion, got.Component.DraftVersion)
	require.FileExists(t, filepath.Join(root, got.ManifestPath))
	require.FileExists(t, filepath.Join(root, got.SourcePath))
	require.Contains(t, got.ChecklistPath, "de-scenario-ification")
	require.Len(t, got.Findings, 2)
}

func TestService_IngestComponentCopiesRelativeImportClosureAsOneVersionUnit(t *testing.T) {
	repo := mocks.NewFakeRepository()
	root := t.TempDir()
	svc := components.NewServiceWithContent(repo, components.NewFSContentStore(root))
	sources := map[string]string{
		"ui/src/components/DrawerShell.tsx": `import { useFocusTrap } from "../hooks/useFocusTrap";
import { useEscapeKey } from "../hooks/useEscapeKey";
export function DrawerShell() { useFocusTrap(); useEscapeKey(); return <div role="dialog" aria-modal="true" />; }`,
		"ui/src/hooks/useFocusTrap.ts": `export function useFocusTrap() { window.addEventListener("keydown", () => {}); }`,
		"ui/src/hooks/useEscapeKey.ts": `export function useEscapeKey() { return undefined; }`,
	}
	components.SetScenarioSourceReader(svc, scenarioSourceReaderFunc(func(_ context.Context, scenario, sourceFile string) ([]byte, error) {
		require.Equal(t, "web-console", scenario)
		body, ok := sources[sourceFile]
		if !ok {
			return nil, errors.New("not found")
		}
		return []byte(body), nil
	}))

	got, err := svc.IngestComponent(context.Background(), components.IngestComponentInput{
		Scenario: "web-console", SourceFile: "ui/src/components/DrawerShell.tsx", Slug: "drawer-shell", DisplayName: "DrawerShell",
	})
	require.NoError(t, err)
	draft, err := svc.GetVersion(context.Background(), got.Component.ID, got.DraftVersion)
	require.NoError(t, err)
	require.Len(t, draft.Files, 3)
	require.NotNil(t, draft.ParityReport)
	require.Empty(t, draft.ParityReport.Findings)
	require.Equal(t, []string{"DrawerShell.tsx", "useEscapeKey.ts", "useFocusTrap.ts"}, draft.ParityReport.OriginFiles)
	require.Equal(t, []string{"DrawerShell.tsx", "useEscapeKey.ts", "useFocusTrap.ts"}, []string{draft.Files[0].Path, draft.Files[1].Path, draft.Files[2].Path})
	require.True(t, draft.Files[0].IsEntry)
	require.Contains(t, draft.Files[0].Content, `from "./useFocusTrap"`)
	require.FileExists(t, filepath.Join(root, "components", "drawer-shell", "versions", got.DraftVersion, "useFocusTrap.ts"))
	require.FileExists(t, filepath.Join(root, "components", "drawer-shell", "versions", got.DraftVersion, "useEscapeKey.ts"))

	reharvested, err := svc.IngestComponent(context.Background(), components.IngestComponentInput{
		Scenario: "web-console", SourceFile: "ui/src/components/DrawerShell.tsx", Slug: "drawer-shell", DisplayName: "DrawerShell", Version: "1.0.0",
	})
	require.NoError(t, err)
	require.Equal(t, "1.0.0-draft.1", reharvested.DraftVersion)
	require.Equal(t, got.Component.ID, reharvested.Component.ID)
	require.FileExists(t, filepath.Join(root, "components", "drawer-shell", "versions", reharvested.DraftVersion, "useFocusTrap.ts"))
}

func TestIngestBehaviorInventoryFlagsHistoricalFocusTrapLoss(t *testing.T) {
	origin := `import { useFocusTrap } from "../hooks/useFocusTrap";
export function Drawer() { return <div role="dialog" aria-modal="true" onKeyDown={() => {}} /> }`
	harvested := `export function Drawer() { return <div role="dialog" /> }`
	findings := components.BehaviorLossFindings(origin, harvested, "Drawer.tsx")
	require.Len(t, findings, 3)
	require.Contains(t, findings[0].Code, "behavior-lost")
}

func TestService_CreateComponentVersionRequiresExplicitParityWaiver(t *testing.T) {
	repo := mocks.NewFakeRepository()
	root := t.TempDir()
	svc := components.NewServiceWithContent(repo, components.NewFSContentStore(root))
	components.SetScenarioSourceReader(svc, scenarioSourceReaderFunc(func(context.Context, string, string) ([]byte, error) {
		return []byte(`export function DrawerShell() { return <div /> }`), nil
	}))
	created, err := svc.IngestComponent(context.Background(), components.IngestComponentInput{Scenario: "web-console", SourceFile: "ui/src/components/DrawerShell.tsx", Slug: "drawer-shell"})
	require.NoError(t, err)
	report := components.IngestParityReport{Findings: []components.IngestFinding{{Code: "behavior-lost", Message: "fixture loss"}}}
	raw, err := json.Marshal(report)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(filepath.Join(root, "components", "drawer-shell", "versions", created.DraftVersion, "parity.json"), raw, 0o600))
	_, err = components.NewIndexer(repo, root, nil).Run(context.Background())
	require.NoError(t, err)
	c, err := svc.GetByLibraryID(context.Background(), created.Component.LibraryID)
	require.NoError(t, err)
	_, err = svc.CreateComponentVersion(context.Background(), components.CreateComponentVersionInput{ComponentID: c.ID, Version: "1.0.0", FromVersion: created.DraftVersion, Intent: components.VersionIntentRelease})
	var waiver components.ErrParityWaiverRequired
	require.True(t, errors.As(err, &waiver))
	got, err := svc.CreateComponentVersion(context.Background(), components.CreateComponentVersionInput{ComponentID: c.ID, Version: "1.0.0", FromVersion: created.DraftVersion, Intent: components.VersionIntentRelease, AcknowledgeParityWaiver: true})
	require.NoError(t, err)
	require.NotNil(t, got.Version.ParityReport)
	require.True(t, got.Version.ParityReport.Acknowledged)
}

func seedStyleFitComponent(t *testing.T, repo *mocks.FakeRepository) components.Component {
	t.Helper()
	c, err := repo.UpsertManifest(context.Background(), components.IndexManifestInput{
		Manifest: components.ComponentManifest{
			LibraryID:     "react-component-library:Button",
			DisplayName:   "Button",
			LatestVersion: "1.0.0",
			DesignStyles: []components.ComponentDesignAffinity{
				{StyleID: "vrooli-default", Affinity: components.DesignAffinityNative, Reason: "token-native baseline"},
				{StyleID: "vrooli-conversion-landing", Affinity: components.DesignAffinityCompatible},
				{StyleID: "vrooli-data-dense", Affinity: components.DesignAffinityDiscouraged, Reason: "too sparse"},
			},
		},
		Versions: []components.ComponentVersion{{Version: "1.0.0"}},
	})
	require.NoError(t, err)
	return c
}

type serviceJSONReaderFunc func(context.Context, string) ([]byte, error)

func (f serviceJSONReaderFunc) Read(ctx context.Context, scenario string) ([]byte, error) {
	return f(ctx, scenario)
}

type scenarioSourceReaderFunc func(context.Context, string, string) ([]byte, error)

func (f scenarioSourceReaderFunc) Read(ctx context.Context, scenario, sourceFile string) ([]byte, error) {
	return f(ctx, scenario, sourceFile)
}
