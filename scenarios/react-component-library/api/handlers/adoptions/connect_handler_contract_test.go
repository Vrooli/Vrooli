package adoptions

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	adoptionsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/react-component-library/v1/adoptions"
	inventoryv1 "github.com/vrooli/vrooli/packages/proto/gen/go/ui-health/v1/inventory"
	"react-component-library/internal/adoptions"
	"react-component-library/internal/catalogcoverage"
	"react-component-library/internal/components"
	"react-component-library/internal/deps"
)

// handlerService embeds the full application interface so each contract test
// only has to implement the RPC's seam. An unexpected service call remains a
// nil-function panic, making accidental widening of a test immediately clear.
type handlerService struct {
	adoptions.Service
	list       func(context.Context, adoptions.ListQuery) ([]adoptions.Adoption, error)
	effective  func(context.Context, string, int) ([]adoptions.EffectiveAdoption, error)
	preflight  func(context.Context, adoptions.PreflightInput) (adoptions.PreflightResult, error)
	sync       func(context.Context, adoptions.TokenSyncInput) (adoptions.TokenSyncResult, error)
	prune      func(context.Context, adoptions.TokenPruneInput) (adoptions.TokenPruneResult, error)
	refresh    func(context.Context, string) ([]adoptions.Adoption, adoptions.RefreshSummary, error)
	forkReport func(context.Context, string, bool) ([]adoptions.Adoption, adoptions.RefreshSummary, error)
	reconcile  func(context.Context, adoptions.ReconcileInput) (adoptions.ReconcileResult, error)
	reconverge func(context.Context, adoptions.ReconvergeInput) (adoptions.ReconvergeResult, error)
	discover   func(context.Context, adoptions.DiscoverInput) (adoptions.DiscoverResult, error)
	confirm    func(context.Context, adoptions.ConfirmDiscoveryInput) (adoptions.ConfirmDiscoveryResult, error)
	delete     func(context.Context, string, bool) (adoptions.DeleteResult, error)
	link       func(context.Context, adoptions.LinkInput) (adoptions.LinkResult, error)
	eject      func(context.Context, adoptions.EjectInput) (adoptions.ApplyResult, error)
	batch      func(context.Context, adoptions.BatchApplyInput) (adoptions.BatchApplyResult, error)
	reapply    func(context.Context, adoptions.ReapplyInput) (adoptions.Adoption, string, error)
}

func (s handlerService) List(ctx context.Context, q adoptions.ListQuery) ([]adoptions.Adoption, error) {
	return s.list(ctx, q)
}

func (s handlerService) ListEffective(ctx context.Context, id string, limit int) ([]adoptions.EffectiveAdoption, error) {
	return s.effective(ctx, id, limit)
}

func (s handlerService) Preflight(ctx context.Context, in adoptions.PreflightInput) (adoptions.PreflightResult, error) {
	return s.preflight(ctx, in)
}

func (s handlerService) SyncScenarioTokens(ctx context.Context, in adoptions.TokenSyncInput) (adoptions.TokenSyncResult, error) {
	return s.sync(ctx, in)
}

func (s handlerService) PruneScenarioTokens(ctx context.Context, in adoptions.TokenPruneInput) (adoptions.TokenPruneResult, error) {
	return s.prune(ctx, in)
}

func (s handlerService) Refresh(ctx context.Context, id string) ([]adoptions.Adoption, adoptions.RefreshSummary, error) {
	return s.refresh(ctx, id)
}

func (s handlerService) ForkReport(ctx context.Context, id string, apply bool) ([]adoptions.Adoption, adoptions.RefreshSummary, error) {
	return s.forkReport(ctx, id, apply)
}

func (s handlerService) Reconcile(ctx context.Context, in adoptions.ReconcileInput) (adoptions.ReconcileResult, error) {
	return s.reconcile(ctx, in)
}

func (s handlerService) Reconverge(ctx context.Context, in adoptions.ReconvergeInput) (adoptions.ReconvergeResult, error) {
	return s.reconverge(ctx, in)
}

func (s handlerService) Discover(ctx context.Context, in adoptions.DiscoverInput) (adoptions.DiscoverResult, error) {
	return s.discover(ctx, in)
}

func (s handlerService) ConfirmDiscovery(ctx context.Context, in adoptions.ConfirmDiscoveryInput) (adoptions.ConfirmDiscoveryResult, error) {
	return s.confirm(ctx, in)
}

func (s handlerService) DeleteWithOptions(ctx context.Context, id string, confirm bool) (adoptions.DeleteResult, error) {
	return s.delete(ctx, id, confirm)
}

func (s handlerService) Link(ctx context.Context, in adoptions.LinkInput) (adoptions.LinkResult, error) {
	return s.link(ctx, in)
}

func (s handlerService) Eject(ctx context.Context, in adoptions.EjectInput) (adoptions.ApplyResult, error) {
	return s.eject(ctx, in)
}

func (s handlerService) BatchApply(ctx context.Context, in adoptions.BatchApplyInput) (adoptions.BatchApplyResult, error) {
	return s.batch(ctx, in)
}

func (s handlerService) Reapply(ctx context.Context, in adoptions.ReapplyInput) (adoptions.Adoption, string, error) {
	return s.reapply(ctx, in)
}

func TestConnectHandlerListAndProjectionContracts(t *testing.T) {
	ctx := context.Background()
	rows := []adoptions.Adoption{{ID: "a1", ComponentID: "cmp", Scenario: "demo", AdoptedPath: "Button.tsx", CreatedAt: timeForHandlerTests(), Files: []adoptions.AdoptionFile{{LibraryPath: "Button.tsx", AdoptedPath: "Button.tsx", SourceAssetID: "asset"}}}}
	svc := handlerService{
		list: func(_ context.Context, q adoptions.ListQuery) ([]adoptions.Adoption, error) {
			require.Equal(t, adoptions.ListQuery{ComponentID: "cmp", Scenario: "demo", Limit: 7}, q)
			return rows, nil
		},
		effective: func(_ context.Context, id string, limit int) ([]adoptions.EffectiveAdoption, error) {
			require.Equal(t, "cmp", id)
			require.Equal(t, 3, limit)
			return []adoptions.EffectiveAdoption{{SourceAssetID: "asset", SourceLibraryID: "rcl:Button", SourceVersion: "1", Mediated: true, ParentAdoption: rows[0]}}, nil
		},
	}
	h := NewConnectHandler(Deps{Service: svc})
	got, err := h.ListAdoptions(ctx, connect.NewRequest(&adoptionsv1.ListAdoptionsRequest{ComponentId: "cmp", Scenario: "demo", Limit: 7}))
	require.NoError(t, err)
	require.Len(t, got.Msg.Adoptions, 1)
	require.Equal(t, "a1", got.Msg.Adoptions[0].Id)
	require.Len(t, got.Msg.Adoptions[0].Files, 1)
	gotEffective, err := h.ListEffectiveAdoptions(ctx, connect.NewRequest(&adoptionsv1.ListEffectiveAdoptionsRequest{ComponentId: "cmp", Limit: 3}))
	require.NoError(t, err)
	require.Len(t, gotEffective.Msg.Adoptions, 1)
	require.True(t, gotEffective.Msg.Adoptions[0].Mediated)
	require.Equal(t, "a1", gotEffective.Msg.Adoptions[0].ParentAdoption.Id)
}

func TestConnectHandlerListAndPreflightErrors(t *testing.T) {
	want := errors.New("repository unavailable")
	h := NewConnectHandler(Deps{Service: handlerService{
		list: func(context.Context, adoptions.ListQuery) ([]adoptions.Adoption, error) { return nil, want },
		preflight: func(context.Context, adoptions.PreflightInput) (adoptions.PreflightResult, error) {
			return adoptions.PreflightResult{}, want
		},
	}})
	_, err := h.ListAdoptions(context.Background(), connect.NewRequest(&adoptionsv1.ListAdoptionsRequest{}))
	require.Error(t, err)
	require.Equal(t, connect.CodeInternal, connect.CodeOf(err))
	_, err = h.PreflightAdoption(context.Background(), connect.NewRequest(&adoptionsv1.PreflightAdoptionRequest{}))
	require.Error(t, err)
	require.Equal(t, connect.CodeInternal, connect.CodeOf(err))
}

func TestConnectHandlerPreflightAndTokenResults(t *testing.T) {
	h := NewConnectHandler(Deps{Service: handlerService{
		preflight: func(_ context.Context, in adoptions.PreflightInput) (adoptions.PreflightResult, error) {
			require.Equal(t, adoptions.PreflightInput{ComponentID: "cmp", Scenario: "demo", Version: "2"}, in)
			return adoptions.PreflightResult{ComponentID: "cmp", Scenario: "demo", Version: "2", CatalogAssetID: "asset", CatalogLatestVersion: "3", Tokens: adoptions.TokenVerdict{Required: []string{"--accent"}, Defined: []string{"--accent"}}, Verdict: adoptions.AdoptionVerdict{Version: components.VersionStatusReleased, Maturity: adoptions.MaturityVerdict{Achieved: "tested", Floor: "tested"}, Warnings: []string{"review"}}}, nil
		},
		sync: func(_ context.Context, in adoptions.TokenSyncInput) (adoptions.TokenSyncResult, error) {
			require.Equal(t, adoptions.TokenSyncInput{Scenario: "demo", DryRun: true, CandidateComponentID: "cmp", CandidateVersion: "2"}, in)
			return adoptions.TokenSyncResult{Scenario: "demo", Added: []string{"--accent"}, Changed: true}, nil
		},
		prune: func(_ context.Context, in adoptions.TokenPruneInput) (adoptions.TokenPruneResult, error) {
			require.Equal(t, adoptions.TokenPruneInput{Scenario: "demo", Apply: true}, in)
			return adoptions.TokenPruneResult{Scenario: "demo", Removed: []string{"--old"}, Retained: []string{"--used"}, Changed: true}, nil
		},
	}})
	ctx := context.Background()
	pre, err := h.PreflightAdoption(ctx, connect.NewRequest(&adoptionsv1.PreflightAdoptionRequest{ComponentId: "cmp", Scenario: "demo", Version: "2"}))
	require.NoError(t, err)
	require.Equal(t, "asset", pre.Msg.CatalogAssetId)
	require.Equal(t, []string{"--accent"}, pre.Msg.RequiredTokens)
	require.Equal(t, []string{"review"}, pre.Msg.Warnings)
	synced, err := h.SyncScenarioTokens(ctx, connect.NewRequest(&adoptionsv1.SyncScenarioTokensRequest{Scenario: "demo", DryRun: true, CandidateComponentId: "cmp", CandidateVersion: "2"}))
	require.NoError(t, err)
	require.True(t, synced.Msg.Changed)
	require.Equal(t, []string{"--accent"}, synced.Msg.Added)
	pruned, err := h.PruneScenarioTokens(ctx, connect.NewRequest(&adoptionsv1.PruneScenarioTokensRequest{Scenario: "demo", Apply: true}))
	require.NoError(t, err)
	require.Equal(t, []string{"--old"}, pruned.Msg.Removed)
	require.Equal(t, []string{"--used"}, pruned.Msg.Retained)
}

func TestConnectHandlerMutationProjectionContracts(t *testing.T) {
	row := adoptions.Adoption{ID: "a1", ComponentID: "cmp", Scenario: "demo", AdoptedPath: "Button.tsx"}
	h := NewConnectHandler(Deps{Service: handlerService{
		link: func(_ context.Context, in adoptions.LinkInput) (adoptions.LinkResult, error) {
			require.Equal(t, "pkg", in.ImportSubpath)
			return adoptions.LinkResult{Adoption: row, PackagePath: "packages/rcl", ImportSubpath: "pkg", UpdatedFiles: []string{"package.json"}}, nil
		},
		eject: func(_ context.Context, in adoptions.EjectInput) (adoptions.ApplyResult, error) {
			require.Equal(t, "operator fork", in.Reason)
			return adoptions.ApplyResult{Adoption: row, WrittenPath: "Button.tsx", CopiedAssets: []string{"icon.svg"}}, nil
		},
		batch: func(_ context.Context, in adoptions.BatchApplyInput) (adoptions.BatchApplyResult, error) {
			require.Len(t, in.Items, 1)
			require.Equal(t, "Button.tsx", in.Items[0].AdoptedPath)
			return adoptions.BatchApplyResult{Results: []adoptions.ApplyResult{{Adoption: row, WrittenPath: "Button.tsx", ImportSites: []string{"App.tsx"}}}, SharedDependencies: []string{"asset:icon"}}, nil
		},
		reapply: func(_ context.Context, in adoptions.ReapplyInput) (adoptions.Adoption, string, error) {
			require.True(t, in.DryRun)
			return row, "Button.tsx", nil
		},
		delete: func(_ context.Context, id string, confirmed bool) (adoptions.DeleteResult, error) {
			require.Equal(t, "a1", id)
			require.True(t, confirmed)
			return adoptions.DeleteResult{RemovedFiles: []string{"Button.tsx"}}, nil
		},
	}})
	ctx := context.Background()
	linked, err := h.LinkAdoption(ctx, connect.NewRequest(&adoptionsv1.LinkAdoptionRequest{ComponentId: "cmp", Scenario: "demo", ImportSubpath: "pkg"}))
	require.NoError(t, err)
	require.Equal(t, "packages/rcl", linked.Msg.PackagePath)
	require.Equal(t, []string{"package.json"}, linked.Msg.UpdatedFiles)
	ejected, err := h.EjectAdoption(ctx, connect.NewRequest(&adoptionsv1.EjectAdoptionRequest{ComponentId: "cmp", Scenario: "demo", Reason: "operator fork"}))
	require.NoError(t, err)
	require.Equal(t, []string{"icon.svg"}, ejected.Msg.CopiedAssets)
	batch, err := h.BatchApplyAdoptions(ctx, connect.NewRequest(&adoptionsv1.BatchApplyAdoptionsRequest{Items: []*adoptionsv1.BatchApplyItem{{ComponentId: "cmp", Scenario: "demo", AdoptedPath: "Button.tsx"}, nil}}))
	require.NoError(t, err)
	require.Len(t, batch.Msg.Results, 1)
	require.Equal(t, []string{"asset:icon"}, batch.Msg.SharedDependencies)
	reapplied, err := h.ReapplyAdoption(ctx, connect.NewRequest(&adoptionsv1.ReapplyAdoptionRequest{Id: "a1", DryRun: true}))
	require.NoError(t, err)
	require.Equal(t, "Button.tsx", reapplied.Msg.WrittenPath)
	deleted, err := h.DeleteAdoption(ctx, connect.NewRequest(&adoptionsv1.DeleteAdoptionRequest{Id: "a1", ConfirmRemoveFiles: true}))
	require.NoError(t, err)
	require.Equal(t, []string{"Button.tsx"}, deleted.Msg.RemovedFiles)
}

func TestConnectHandlerRefreshAndIntakeProjectionContracts(t *testing.T) {
	rows := []adoptions.Adoption{{ID: "a1", Scenario: "demo", LibraryID: "rcl:Button", ForkStatus: adoptions.ForkStatusDeclared}}
	h := NewConnectHandler(Deps{Service: handlerService{
		refresh: func(_ context.Context, id string) ([]adoptions.Adoption, adoptions.RefreshSummary, error) {
			require.Equal(t, "cmp", id)
			return rows, adoptions.RefreshSummary{LibraryCurrent: 1, LocalClean: 1}, nil
		},
		forkReport: func(_ context.Context, id string, apply bool) ([]adoptions.Adoption, adoptions.RefreshSummary, error) {
			require.Equal(t, "cmp", id)
			require.True(t, apply)
			return rows, adoptions.RefreshSummary{}, nil
		},
		reconcile: func(_ context.Context, in adoptions.ReconcileInput) (adoptions.ReconcileResult, error) {
			require.True(t, in.Apply)
			return adoptions.ReconcileResult{Scanned: 2, Created: 1, Findings: []adoptions.ReconcileFinding{{Scenario: "demo", AdoptedPath: "Button.tsx", LibraryID: "rcl:Button", Version: "1"}}}, nil
		},
		reconverge: func(_ context.Context, in adoptions.ReconvergeInput) (adoptions.ReconvergeResult, error) {
			require.Equal(t, adoptions.ReconvergeInput{Scenario: "demo", Apply: true}, in)
			return adoptions.ReconvergeResult{Behind: 1, Outcomes: []adoptions.ReconvergeOutcome{{AdoptionID: "a1", Scenario: "demo", Action: adoptions.ReconvergeActionReapplied, Files: []adoptions.ReconvergeFileOutcome{{LibraryPath: "Button.tsx", AdoptedPath: "Button.tsx", LocalStatus: adoptions.LocalStatusClean}}}}}, nil
		},
		discover: func(_ context.Context, in adoptions.DiscoverInput) (adoptions.DiscoverResult, error) {
			require.Equal(t, "demo", in.Scenario)
			return adoptions.DiscoverResult{Scanned: 1, MinSimilarity: .8, Candidates: []adoptions.DiscoveryCandidate{{Scenario: "demo", AdoptedPath: "Button.tsx", ComponentID: "cmp", Similarity: .9, Evidence: []string{"same markup"}}}}, nil
		},
		confirm: func(_ context.Context, in adoptions.ConfirmDiscoveryInput) (adoptions.ConfirmDiscoveryResult, error) {
			require.Equal(t, "cmp", in.ComponentID)
			return adoptions.ConfirmDiscoveryResult{Adoption: rows[0], WrittenPath: "Button.tsx", Similarity: .9}, nil
		},
	}})
	ctx := context.Background()
	refreshed, err := h.RefreshAdoptions(ctx, connect.NewRequest(&adoptionsv1.RefreshAdoptionsRequest{ComponentId: "cmp"}))
	require.NoError(t, err)
	require.Equal(t, int32(1), refreshed.Msg.LibraryCurrent)
	require.Empty(t, refreshed.Msg.ForkDispositionCounts)
	forked, err := h.ForkReportAdoptions(ctx, connect.NewRequest(&adoptionsv1.RefreshAdoptionsRequest{ComponentId: "cmp", Apply: true}))
	require.NoError(t, err)
	require.Equal(t, int32(1), forked.Msg.ForkDispositionCounts["declared-fork"])
	require.Equal(t, int32(1), forked.Msg.ForkScenarioCounts["demo"])
	reconciled, err := h.ReconcileAdoptions(ctx, connect.NewRequest(&adoptionsv1.ReconcileAdoptionsRequest{Apply: true}))
	require.NoError(t, err)
	require.Equal(t, int32(1), reconciled.Msg.Created)
	require.Equal(t, "Button.tsx", reconciled.Msg.Findings[0].AdoptedPath)
	reconverged, err := h.ReconvergeAdoptions(ctx, connect.NewRequest(&adoptionsv1.ReconvergeAdoptionsRequest{Scenario: "demo", Apply: true}))
	require.NoError(t, err)
	require.Equal(t, int32(1), reconverged.Msg.Behind)
	require.Equal(t, adoptionsv1.ReconvergeAction_RECONVERGE_ACTION_REAPPLIED, reconverged.Msg.Outcomes[0].Action)
	require.Len(t, reconverged.Msg.Outcomes[0].Files, 1)
	discovered, err := h.DiscoverAdoptions(ctx, connect.NewRequest(&adoptionsv1.DiscoverAdoptionsRequest{Scenario: "demo"}))
	require.NoError(t, err)
	require.Equal(t, .8, discovered.Msg.MinSimilarity)
	require.Equal(t, []string{"same markup"}, discovered.Msg.Candidates[0].Evidence)
	confirmed, err := h.ConfirmDiscovery(ctx, connect.NewRequest(&adoptionsv1.ConfirmDiscoveryRequest{ComponentId: "cmp"}))
	require.NoError(t, err)
	require.Equal(t, .9, confirmed.Msg.Similarity)
	require.Equal(t, "Button.tsx", confirmed.Msg.WrittenPath)
}

func TestConnectHandlerServiceErrorsPreserveConnectCodes(t *testing.T) {
	want := errors.New("database failure")
	h := NewConnectHandler(Deps{Service: handlerService{
		effective: func(context.Context, string, int) ([]adoptions.EffectiveAdoption, error) { return nil, want },
		sync: func(context.Context, adoptions.TokenSyncInput) (adoptions.TokenSyncResult, error) {
			return adoptions.TokenSyncResult{}, want
		},
		prune: func(context.Context, adoptions.TokenPruneInput) (adoptions.TokenPruneResult, error) {
			return adoptions.TokenPruneResult{}, want
		},
		refresh: func(context.Context, string) ([]adoptions.Adoption, adoptions.RefreshSummary, error) {
			return nil, adoptions.RefreshSummary{}, want
		},
		reconcile: func(context.Context, adoptions.ReconcileInput) (adoptions.ReconcileResult, error) {
			return adoptions.ReconcileResult{}, want
		},
		reconverge: func(context.Context, adoptions.ReconvergeInput) (adoptions.ReconvergeResult, error) {
			return adoptions.ReconvergeResult{}, want
		},
		discover: func(context.Context, adoptions.DiscoverInput) (adoptions.DiscoverResult, error) {
			return adoptions.DiscoverResult{}, want
		},
		confirm: func(context.Context, adoptions.ConfirmDiscoveryInput) (adoptions.ConfirmDiscoveryResult, error) {
			return adoptions.ConfirmDiscoveryResult{}, want
		},
		delete: func(context.Context, string, bool) (adoptions.DeleteResult, error) {
			return adoptions.DeleteResult{}, want
		},
		link: func(context.Context, adoptions.LinkInput) (adoptions.LinkResult, error) {
			return adoptions.LinkResult{}, want
		},
		eject: func(context.Context, adoptions.EjectInput) (adoptions.ApplyResult, error) {
			return adoptions.ApplyResult{}, want
		},
		batch: func(context.Context, adoptions.BatchApplyInput) (adoptions.BatchApplyResult, error) {
			return adoptions.BatchApplyResult{}, want
		},
		reapply: func(context.Context, adoptions.ReapplyInput) (adoptions.Adoption, string, error) {
			return adoptions.Adoption{}, "", want
		},
	}})
	ctx := context.Background()
	cases := []struct {
		name string
		call func() error
	}{
		{"effective", func() error {
			_, e := h.ListEffectiveAdoptions(ctx, connect.NewRequest(&adoptionsv1.ListEffectiveAdoptionsRequest{}))
			return e
		}},
		{"sync", func() error {
			_, e := h.SyncScenarioTokens(ctx, connect.NewRequest(&adoptionsv1.SyncScenarioTokensRequest{}))
			return e
		}},
		{"prune", func() error {
			_, e := h.PruneScenarioTokens(ctx, connect.NewRequest(&adoptionsv1.PruneScenarioTokensRequest{}))
			return e
		}},
		{"refresh", func() error {
			_, e := h.RefreshAdoptions(ctx, connect.NewRequest(&adoptionsv1.RefreshAdoptionsRequest{}))
			return e
		}},
		{"reconcile", func() error {
			_, e := h.ReconcileAdoptions(ctx, connect.NewRequest(&adoptionsv1.ReconcileAdoptionsRequest{}))
			return e
		}},
		{"reconverge", func() error {
			_, e := h.ReconvergeAdoptions(ctx, connect.NewRequest(&adoptionsv1.ReconvergeAdoptionsRequest{}))
			return e
		}},
		{"discover", func() error {
			_, e := h.DiscoverAdoptions(ctx, connect.NewRequest(&adoptionsv1.DiscoverAdoptionsRequest{}))
			return e
		}},
		{"confirm", func() error {
			_, e := h.ConfirmDiscovery(ctx, connect.NewRequest(&adoptionsv1.ConfirmDiscoveryRequest{}))
			return e
		}},
		{"delete", func() error {
			_, e := h.DeleteAdoption(ctx, connect.NewRequest(&adoptionsv1.DeleteAdoptionRequest{}))
			return e
		}},
		{"link", func() error {
			_, e := h.LinkAdoption(ctx, connect.NewRequest(&adoptionsv1.LinkAdoptionRequest{}))
			return e
		}},
		{"eject", func() error {
			_, e := h.EjectAdoption(ctx, connect.NewRequest(&adoptionsv1.EjectAdoptionRequest{}))
			return e
		}},
		{"batch", func() error {
			_, e := h.BatchApplyAdoptions(ctx, connect.NewRequest(&adoptionsv1.BatchApplyAdoptionsRequest{}))
			return e
		}},
		{"reapply", func() error {
			_, e := h.ReapplyAdoption(ctx, connect.NewRequest(&adoptionsv1.ReapplyAdoptionRequest{}))
			return e
		}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.call()
			require.Error(t, err)
			require.Equal(t, connect.CodeInternal, connect.CodeOf(err))
		})
	}
}

func TestSuggestionScenarioAndTokenHelpers(t *testing.T) {
	root := t.TempDir()
	require.NoError(t, os.Mkdir(filepath.Join(root, "beta"), 0o755))
	require.NoError(t, os.Mkdir(filepath.Join(root, "alpha"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(root, "file"), nil, 0o600))
	h := NewConnectHandler(Deps{ScenariosRoot: root})
	names, err := h.suggestionScenarios("")
	require.NoError(t, err)
	require.Equal(t, []string{"alpha", "beta"}, names)
	names, err = h.suggestionScenarios("  demo ")
	require.NoError(t, err)
	require.Equal(t, []string{"demo"}, names)
	_, err = h.suggestionScenarios("../demo")
	require.Error(t, err)
	_, err = NewConnectHandler(Deps{ScenariosRoot: filepath.Join(root, "missing")}).suggestionScenarios("")
	require.Error(t, err)
	require.Equal(t, "drawershell", normalizeSuggestionToken("Drawer-Shell"))
	component := components.Component{Slug: "DrawerShell", DisplayName: "Drawer Shell"}
	require.Equal(t, "drawer", suggestionMatch(component, "Drawer panel", "src/Drawer.tsx"))
	require.Equal(t, []string{"DrawerShell", "Drawer Shell", "drawer", "drawer"}, suggestionTerms(component))
	require.Equal(t, "2.0", firstVersion(" ", " 2.0 ", "1.0"))
	require.Empty(t, firstVersion(" ", ""))
	kept := activeSuggestionComponents([]components.Component{{ID: "keep", CatalogID: "active"}, {ID: "old", CatalogID: "old"}, {ID: "unknown", CatalogID: "missing"}, {ID: "legacy"}}, []catalogcoverage.Asset{{ID: "active", Maturity: "ready"}, {ID: "old", Maturity: "deprecated"}})
	require.Equal(t, []string{"keep", "legacy"}, []string{kept[0].ID, kept[1].ID})
}

func TestConnectHandlerScenarioInventoryAndUnconfiguredSuggestions(t *testing.T) {
	root := t.TempDir()
	require.NoError(t, os.Mkdir(filepath.Join(root, "zebra"), 0o755))
	require.NoError(t, os.Mkdir(filepath.Join(root, "alpha"), 0o755))
	h := NewConnectHandler(Deps{Service: handlerService{}, ScenariosRoot: root})
	listed, err := h.ListScenarios(context.Background(), connect.NewRequest(&adoptionsv1.ListScenariosRequest{}))
	require.NoError(t, err)
	require.Equal(t, []string{"alpha", "zebra"}, []string{listed.Msg.Scenarios[0].Name, listed.Msg.Scenarios[1].Name})
	require.Equal(t, "alpha", listed.Msg.Scenarios[0].DisplayName)

	_, err = NewConnectHandler(Deps{Service: handlerService{}, ScenariosRoot: filepath.Join(root, "absent")}).ListScenarios(context.Background(), connect.NewRequest(&adoptionsv1.ListScenariosRequest{}))
	require.Error(t, err)
	require.Equal(t, connect.CodeInternal, connect.CodeOf(err))
	_, err = h.SuggestAdoptions(context.Background(), connect.NewRequest(&adoptionsv1.SuggestAdoptionsRequest{}))
	require.Error(t, err)
	require.Equal(t, connect.CodeUnimplemented, connect.CodeOf(err))
}

type suggestionComponents struct {
	components.Service
	t    *testing.T
	rows []components.Component
}

func (s suggestionComponents) List(_ context.Context, q components.SearchQuery) ([]components.Component, error) {
	require.Equal(s.t, 500, q.Limit)
	return s.rows, nil
}

func (s suggestionComponents) ValidateStyleFit(_ context.Context, id, version, scenario string) (components.StyleFitVerdict, error) {
	require.Equal(s.t, "cmp", id)
	require.Equal(s.t, "2.0", version)
	require.Equal(s.t, "demo", scenario)
	return components.StyleFitVerdict{Detail: "fits the scenario style"}, nil
}

type suggestionDependencies struct {
	deps.Service
	t *testing.T
}

func (s suggestionDependencies) ValidateAdoption(_ context.Context, id, version, scenario string) (deps.Verdict, error) {
	require.Equal(s.t, "cmp", id)
	require.Equal(s.t, "2.0", version)
	require.Equal(s.t, "demo", scenario)
	return deps.Verdict{Kind: deps.VerdictWarn}, nil
}

type suggestionInventory struct {
	InventoryScanner
	t *testing.T
}

func (s suggestionInventory) ScanScenario(_ context.Context, req *connect.Request[inventoryv1.ScanScenarioRequest]) (*connect.Response[inventoryv1.ScanScenarioResponse], error) {
	require.Equal(s.t, "demo", req.Msg.Scenario)
	return connect.NewResponse(&inventoryv1.ScanScenarioResponse{Surfaces: []*inventoryv1.SurfaceRecord{{DisplayName: "Button controls", FilePath: "src/Button.tsx"}}}), nil
}

func TestSuggestAdoptionsReturnsExplainableHeuristicCandidate(t *testing.T) {
	root := t.TempDir()
	assets := filepath.Join(root, "react-component-library", "catalog", "assets")
	require.NoError(t, os.MkdirAll(assets, 0o755))
	service := handlerService{
		list: func(_ context.Context, q adoptions.ListQuery) ([]adoptions.Adoption, error) {
			require.Equal(t, adoptions.ListQuery{Scenario: "demo", Limit: 1024}, q)
			return nil, nil
		},
		effective: func(_ context.Context, id string, limit int) ([]adoptions.EffectiveAdoption, error) {
			require.Equal(t, "cmp", id)
			require.Equal(t, 1024, limit)
			return nil, nil
		},
	}
	h := NewConnectHandler(Deps{
		Service: service,
		Components: suggestionComponents{t: t, rows: []components.Component{{
			ID: "cmp", Slug: "Button", DisplayName: "Button", LibraryID: "rcl:Button", LatestVersion: "2.0",
		}}},
		Dependencies: suggestionDependencies{t: t}, Inventory: suggestionInventory{t: t}, ScenariosRoot: root,
	})
	got, err := h.SuggestAdoptions(context.Background(), connect.NewRequest(&adoptionsv1.SuggestAdoptionsRequest{Scenario: " demo "}))
	require.NoError(t, err)
	require.Len(t, got.Msg.Suggestions, 1)
	suggestion := got.Msg.Suggestions[0]
	require.Equal(t, "demo", suggestion.Scenario)
	require.Equal(t, "cmp", suggestion.ComponentId)
	require.Equal(t, "src/Button.tsx", suggestion.InventoryPath)
	require.Equal(t, adoptionsv1.RecommendationClass_RECOMMENDATION_CLASS_HEURISTIC, suggestion.Classification)
	require.Equal(t, []string{
		"heuristic inventory candidate; review before using the separate adoption action",
		"inventory surface \"src/Button.tsx\" matches Button",
		"fits the scenario style",
		"dependency compatibility has warnings",
	}, suggestion.Reasons)
}

func TestSuggestAdoptionsSkipsComponentsPresentThroughAnotherAdoption(t *testing.T) {
	root := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(root, "react-component-library", "catalog", "assets"), 0o755))
	service := handlerService{
		list: func(context.Context, adoptions.ListQuery) ([]adoptions.Adoption, error) { return nil, nil },
		effective: func(_ context.Context, id string, limit int) ([]adoptions.EffectiveAdoption, error) {
			require.Equal(t, "cmp", id)
			require.Equal(t, 1024, limit)
			return []adoptions.EffectiveAdoption{{ParentAdoption: adoptions.Adoption{Scenario: "demo"}}}, nil
		},
	}
	h := NewConnectHandler(Deps{
		Service: service,
		Components: suggestionComponents{t: t, rows: []components.Component{{
			ID: "cmp", Slug: "Button", DisplayName: "Button", LibraryID: "rcl:Button", LatestVersion: "2.0",
		}}},
		Dependencies: suggestionDependencies{t: t}, Inventory: suggestionInventory{t: t}, ScenariosRoot: root,
	})
	got, err := h.SuggestAdoptions(context.Background(), connect.NewRequest(&adoptionsv1.SuggestAdoptionsRequest{Scenario: "demo"}))
	require.NoError(t, err)
	require.Empty(t, got.Msg.Suggestions)
}

func timeForHandlerTests() time.Time { return time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC) }
