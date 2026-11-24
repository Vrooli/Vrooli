package workspaces

import (
	"context"
	"errors"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"
	workspacev1 "github.com/vrooli/vrooli/packages/proto/gen/go/react-component-library/v1/workspace"
)

func TestConversionReceiptCreateIsIdempotentAndImmutable(t *testing.T) {
	repo := newTestRepository(t)
	ctx := context.Background()
	input := ConversionReceiptInput{Scenario: "demo", Page: "home", WorkspaceID: "ws_demo", SourceHash: "source", TargetHash: "target", ReceiptJSON: `{"source":"exact"}`}

	created, err := repo.CreateConversionReceipt(ctx, input)
	require.NoError(t, err)
	require.NotEmpty(t, created.ID)
	require.Equal(t, "2026-11-19T12:00:00.000Z", created.CreatedAt)

	repeated, err := repo.CreateConversionReceipt(ctx, input)
	require.NoError(t, err)
	require.Equal(t, created, repeated)

	changed := input
	changed.TargetHash = "new-target"
	_, err = repo.CreateConversionReceipt(ctx, changed)
	var conflict *ConversionReceiptConflictError
	require.ErrorAs(t, err, &conflict)
	require.True(t, errors.Is(err, ErrConversionReceiptConflict))
	require.Equal(t, created, conflict.Existing)

	fetched, err := repo.GetConversionReceipt(ctx, "demo", "home")
	require.NoError(t, err)
	require.Equal(t, created, fetched)
	_, err = repo.GetConversionReceipt(ctx, "demo", "missing")
	require.ErrorIs(t, err, ErrNotFound)
}

func TestCreateWorkspaceWithConversionReceiptIsAtomicAndRepeatSafe(t *testing.T) {
	repo := newTestRepository(t)
	ctx := context.Background()
	workspace := &workspacev1.Workspace{Id: "ws_converted", Name: "Converted", Provenance: workspacev1.WorkspaceProvenance_WORKSPACE_PROVENANCE_IMPORTED, BindingScenario: "demo", BindingPage: "home", ThemeJson: "{}", OptionsJson: "{}"}
	input := ConversionReceiptInput{Scenario: "demo", Page: "home", WorkspaceID: workspace.Id, SourceHash: "source", TargetHash: "target", ReceiptJSON: `{"workspaceId":"ws_converted"}`}
	created, receipt, err := repo.CreateWorkspaceWithConversionReceipt(ctx, workspace, input)
	require.NoError(t, err)
	require.Equal(t, workspace.Id, created.GetId())
	require.NotEmpty(t, receipt.ID)

	repeatedWorkspace, repeatedReceipt, err := repo.CreateWorkspaceWithConversionReceipt(ctx, workspace, input)
	require.NoError(t, err)
	require.Equal(t, receipt, repeatedReceipt)
	require.Equal(t, created.GetId(), repeatedWorkspace.GetId())

	changed := input
	changed.SourceHash = "different-source"
	_, _, err = repo.CreateWorkspaceWithConversionReceipt(ctx, workspace, changed)
	var conflict *ConversionReceiptConflictError
	require.ErrorAs(t, err, &conflict)
	require.Equal(t, receipt, conflict.Existing)
}

func TestConcurrentWorkspaceAndReceiptRetriesResolveToOneCommittedPair(t *testing.T) {
	repo := newTestRepository(t)
	workspace := &workspacev1.Workspace{Id: "ws_concurrent", Name: "Concurrent", Provenance: workspacev1.WorkspaceProvenance_WORKSPACE_PROVENANCE_IMPORTED, BindingScenario: "demo", BindingPage: "home", ThemeJson: "{}", OptionsJson: "{}"}
	input := ConversionReceiptInput{Scenario: "demo", Page: "home", WorkspaceID: workspace.Id, SourceHash: "source", TargetHash: "target", ReceiptJSON: `{"workspaceId":"ws_concurrent"}`}
	start := make(chan struct{})
	type result struct {
		workspace *workspacev1.Workspace
		receipt   ConversionReceipt
		err       error
	}
	results := make(chan result, 2)
	var workers sync.WaitGroup
	for range 2 {
		workers.Add(1)
		go func() {
			defer workers.Done()
			<-start
			created, receipt, err := repo.CreateWorkspaceWithConversionReceipt(context.Background(), workspace, input)
			results <- result{workspace: created, receipt: receipt, err: err}
		}()
	}
	close(start)
	workers.Wait()
	close(results)
	var first *result
	for got := range results {
		require.NoError(t, got.err)
		require.NotNil(t, got.workspace)
		require.NotEmpty(t, got.receipt.ID)
		if first == nil {
			copy := got
			first = &copy
			continue
		}
		require.Equal(t, first.workspace.GetId(), got.workspace.GetId())
		require.Equal(t, first.receipt.ID, got.receipt.ID)
	}
}

func TestConversionReceiptRequiresValidJSON(t *testing.T) {
	repo := newTestRepository(t)
	_, err := repo.CreateConversionReceipt(context.Background(), ConversionReceiptInput{
		Scenario: "demo", Page: "home", WorkspaceID: "ws_demo", SourceHash: "source", TargetHash: "target", ReceiptJSON: "not-json",
	})
	require.ErrorIs(t, err, ErrInvalid)
}
