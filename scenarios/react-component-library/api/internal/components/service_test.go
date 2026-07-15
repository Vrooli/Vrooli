package components_test

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"connectrpc.com/connect"
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

// TestFSServiceJSONReaderResolvesTemplateScenarioKey covers the template
// adoption key form "../templates/scenarios/<id>": it must resolve next to
// the scenarios root (so reapply against a vendored template copy can run
// style-fit) while still rejecting traversal inside the template id.
func TestFSServiceJSONReaderResolvesTemplateScenarioKey(t *testing.T) {
	repoRoot := t.TempDir()
	scenariosRoot := filepath.Join(repoRoot, "scenarios")
	serviceDir := filepath.Join(repoRoot, "templates", "scenarios", "react-vite", ".vrooli")
	require.NoError(t, os.MkdirAll(scenariosRoot, 0o755))
	require.NoError(t, os.MkdirAll(serviceDir, 0o755))
	payload := []byte(`{"generation":{"design":{"id":"vrooli-default"}}}`)
	require.NoError(t, os.WriteFile(filepath.Join(serviceDir, "service.json"), payload, 0o600))

	reader := components.NewFSServiceJSONReader(scenariosRoot)

	got, err := reader.Read(context.Background(), "../templates/scenarios/react-vite")
	require.NoError(t, err)
	require.Equal(t, payload, got)

	_, err = reader.Read(context.Background(), "../templates/scenarios/../../secrets")
	require.ErrorContains(t, err, "invalid scenario name")
	_, err = reader.Read(context.Background(), "../templates/scenarios/react-vite/nested")
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

// TestService_IngestScaffoldsCatalogMetadataContract asserts a fresh harvest
// lands catalog-complete: the manifest carries slot, category, and tags (slot
// and category defaulted when the harvester omits them), and every created
// version folder ships an examples.json stub. This is the contract that keeps
// harvested drafts indistinguishable from authored components at ingest time.
func TestService_IngestScaffoldsCatalogMetadataContract(t *testing.T) {
	repo := mocks.NewFakeRepository()
	root := t.TempDir()
	svc := components.NewServiceWithContent(repo, components.NewFSContentStore(root))
	components.SetScenarioSourceReader(svc, scenarioSourceReaderFunc(func(_ context.Context, _, _ string) ([]byte, error) {
		return []byte(`export default function Panel() { return <div className="rounded-control p-2">Panel</div>; }`), nil
	}))

	got, err := svc.IngestComponent(context.Background(), components.IngestComponentInput{
		Scenario: "web-console", SourceFile: "ui/src/components/Panel.tsx", Slug: "panel",
		DisplayName: "Panel", Tags: []string{"surface"},
		// Slot and Category deliberately omitted: the scaffold must default them.
	})
	require.NoError(t, err)

	manifestRaw, err := os.ReadFile(filepath.Join(root, got.ManifestPath))
	require.NoError(t, err)
	var mf struct {
		Slot     string   `json:"slot"`
		Category string   `json:"category"`
		Tags     []string `json:"tags"`
	}
	require.NoError(t, json.Unmarshal(manifestRaw, &mf))
	require.Equal(t, "ui-pattern", mf.Slot)
	require.Equal(t, "uncategorized", mf.Category)
	require.Equal(t, []string{"surface"}, mf.Tags)

	// Both the released baseline and the working draft carry the examples stub.
	for _, version := range []string{"0.1.0", got.DraftVersion} {
		examplesRaw, err := os.ReadFile(filepath.Join(root, "components", "panel", "versions", version, "examples.json"))
		require.NoError(t, err, "examples.json missing for version %s", version)
		var ef struct {
			Examples []struct {
				Name string `json:"name"`
			} `json:"examples"`
		}
		require.NoError(t, json.Unmarshal(examplesRaw, &ef))
		require.Len(t, ef.Examples, 1)
		require.Equal(t, "default", ef.Examples[0].Name)
	}
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

// TestService_IngestBlocksBehaviorLossUnlessAccepted is the permanent
// planted-error calibration for the origin-parity gate. It reconstructs the
// historical DrawerShell failure: a focus-trap hook reachable only through an
// app-alias import the harvest cannot carry, so the listener behavior is
// dropped. The gate must fail the harvest naming the dropped listener, and
// must only proceed when the caller explicitly accepts the loss — recording
// the named losses as an acknowledged parity report.
func TestService_IngestBlocksBehaviorLossUnlessAccepted(t *testing.T) {
	repo := mocks.NewFakeRepository()
	root := t.TempDir()
	svc := components.NewServiceWithContent(repo, components.NewFSContentStore(root))
	sources := map[string]string{
		"ui/src/components/DrawerShell.tsx": `import { useFocusTrap } from "@/hooks/useFocusTrap";
export function DrawerShell() { useFocusTrap(); return <div role="dialog" aria-modal="true" />; }`,
		"ui/src/hooks/useFocusTrap.ts": `export function useFocusTrap() { window.addEventListener("keydown", () => {}); }`,
	}
	components.SetScenarioSourceReader(svc, scenarioSourceReaderFunc(func(_ context.Context, _ string, sourceFile string) ([]byte, error) {
		body, ok := sources[sourceFile]
		if !ok {
			return nil, errors.New("not found: " + sourceFile)
		}
		return []byte(body), nil
	}))

	in := components.IngestComponentInput{Scenario: "web-console", SourceFile: "ui/src/components/DrawerShell.tsx", Slug: "drawer-shell", DisplayName: "Drawer Shell"}

	// (a) Without the override the harvest is blocked and names the loss.
	_, err := svc.IngestComponent(context.Background(), in)
	var loss components.ErrHarvestBehaviorLoss
	require.True(t, errors.As(err, &loss), "expected ErrHarvestBehaviorLoss, got %v", err)
	require.NotEmpty(t, loss.Findings)
	require.Contains(t, err.Error(), "addEventListener")
	require.Contains(t, err.Error(), "accept-behavior-loss")
	require.Equal(t, connect.CodeFailedPrecondition, connect.CodeOf(components.ToConnectError(err)))
	// A blocked harvest creates nothing.
	_, notFoundErr := svc.GetByLibraryID(context.Background(), "react-component-library:drawer-shell")
	require.True(t, errors.As(notFoundErr, &components.ErrComponentNotFound{}))

	// (b) With the override the harvest proceeds and records the losses.
	accepted := in
	accepted.AcceptBehaviorLoss = true
	got, err := svc.IngestComponent(context.Background(), accepted)
	require.NoError(t, err)
	require.NotEmpty(t, got.ParityReport.Findings)
	require.True(t, got.ParityReport.Acknowledged)

	draft, err := svc.GetVersion(context.Background(), got.Component.ID, got.DraftVersion)
	require.NoError(t, err)
	require.NotNil(t, draft.ParityReport)
	require.True(t, draft.ParityReport.Acknowledged)
	require.NotEmpty(t, draft.ParityReport.Findings)
	require.Contains(t, draft.ParityReport.Findings[0].Message, "addEventListener")

	// The acceptance is durable on the version's parity.json audit trail.
	raw, err := os.ReadFile(filepath.Join(root, "components", "drawer-shell", "versions", got.DraftVersion, "parity.json"))
	require.NoError(t, err)
	var onDisk components.IngestParityReport
	require.NoError(t, json.Unmarshal(raw, &onDisk))
	require.True(t, onDisk.Acknowledged)
	require.NotEmpty(t, onDisk.Findings)
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
