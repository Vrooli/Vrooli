package experienceimport

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	workspacev1 "github.com/vrooli/vrooli/packages/proto/gen/go/react-component-library/v1/workspace"
	"react-component-library/internal/workspaces"
)

type AtomicWorkspaceImporter interface {
	CreateWorkspaceWithConversionReceipt(context.Context, *workspacev1.Workspace, workspaces.ConversionReceiptInput) (*workspacev1.Workspace, workspaces.ConversionReceipt, error)
	GetConversionReceipt(context.Context, string, string) (workspaces.ConversionReceipt, error)
}

// AdoptPage converts and atomically persists an experience page and immutable
// source recovery receipt. Exact retries return the original workspace and
// receipt. A changed source page is rejected for explicit reconciliation.
func AdoptPage(ctx context.Context, importer AtomicWorkspaceImporter, scenario, page string, source []byte) (*workspacev1.Workspace, workspaces.ConversionReceipt, bool, error) {
	if importer == nil {
		return nil, workspaces.ConversionReceipt{}, false, fmt.Errorf("workspace importer is required")
	}
	result, err := ConvertPage(scenario, page, source)
	if err != nil {
		return nil, workspaces.ConversionReceipt{}, false, err
	}
	if err := VerifyResult(result); err != nil {
		return nil, workspaces.ConversionReceipt{}, false, fmt.Errorf("verify conversion before persistence: %w", err)
	}
	encodedReceipt, err := json.Marshal(result.Receipt)
	if err != nil {
		return nil, workspaces.ConversionReceipt{}, false, fmt.Errorf("encode conversion receipt: %w", err)
	}
	input := workspaces.ConversionReceiptInput{
		Scenario: scenario, Page: page, WorkspaceID: result.Workspace.GetId(),
		SourceHash: result.Receipt.SourceHash, TargetHash: result.Receipt.TargetHash,
		ReceiptJSON: string(encodedReceipt),
	}
	_, lookupErr := importer.GetConversionReceipt(ctx, scenario, page)
	alreadyAdopted := lookupErr == nil
	if lookupErr != nil && !errors.Is(lookupErr, workspaces.ErrNotFound) {
		return nil, workspaces.ConversionReceipt{}, false, fmt.Errorf("read existing conversion receipt: %w", lookupErr)
	}
	created, receipt, err := importer.CreateWorkspaceWithConversionReceipt(ctx, result.Workspace, input)
	if err != nil {
		return nil, workspaces.ConversionReceipt{}, false, fmt.Errorf("persist workspace and conversion receipt: %w", err)
	}
	return created, receipt, alreadyAdopted, nil
}
