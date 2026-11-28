package validationbroker

import (
	"context"
	"fmt"
	"net/http"

	"connectrpc.com/connect"
	"github.com/vrooli/api-core/discovery"
	bountycases "github.com/vrooli/vrooli/packages/proto/gen/go/client-rehearsal/v1/bountycases"
	bountycasesconnect "github.com/vrooli/vrooli/packages/proto/gen/go/client-rehearsal/v1/bountycases/bountycases_v1connect"
	validationv1 "github.com/vrooli/vrooli/packages/proto/gen/go/test-genie/v1/validation"
	workspace "github.com/vrooli/vrooli/packages/proto/gen/go/workspace-sandbox/v1/workspace"
	workspaceconnect "github.com/vrooli/vrooli/packages/proto/gen/go/workspace-sandbox/v1/workspace/workspaceconnect"
)

// LiveWorkspaceSandboxSealOwner delegates every path resolution and integrity
// check to Workspace Sandbox. The returned path is consumed only inside the
// Test Genie identity resolver and is never serialized to callers.
type LiveWorkspaceSandboxSealOwner struct{}

func (LiveWorkspaceSandboxSealOwner) client(ctx context.Context) (workspaceconnect.WorkspaceSandboxServiceClient, error) {
	url, err := discovery.ResolveScenarioURLDefault(ctx, "workspace-sandbox")
	if err != nil {
		return nil, fmt.Errorf("resolve workspace-sandbox URL: %w", err)
	}
	return workspaceconnect.NewWorkspaceSandboxServiceClient(http.DefaultClient, url), nil
}

func (o LiveWorkspaceSandboxSealOwner) ResolveReplaySeal(ctx context.Context, ref *validationv1.WorkspaceSandboxSealReference) (string, error) {
	client, err := o.client(ctx)
	if err != nil {
		return "", err
	}
	response, err := client.ResolveReplaySeal(ctx, connect.NewRequest(&workspace.ResolveReplaySealRequest{CaseId: ref.GetCaseId(), Role: workspace.ReplaySealRole(ref.GetRole()), SealDigest: ref.GetSealDigest()}))
	if err != nil {
		return "", err
	}
	if !response.Msg.GetSuccess() || response.Msg.GetCaseId() != ref.GetCaseId() || response.Msg.GetRole() != workspace.ReplaySealRole(ref.GetRole()) || response.Msg.GetSealDigest() != ref.GetSealDigest() || response.Msg.GetWorkspaceRoot() == "" {
		return "", fmt.Errorf("workspace-sandbox returned a mismatched replay seal")
	}
	return response.Msg.GetWorkspaceRoot(), nil
}

func (o LiveWorkspaceSandboxSealOwner) VerifyReplaySeal(ctx context.Context, ref *validationv1.WorkspaceSandboxSealReference) error {
	client, err := o.client(ctx)
	if err != nil {
		return err
	}
	response, err := client.VerifyReplaySeal(ctx, connect.NewRequest(&workspace.VerifyReplaySealRequest{CaseId: ref.GetCaseId(), Role: workspace.ReplaySealRole(ref.GetRole()), SealDigest: ref.GetSealDigest()}))
	if err != nil {
		return err
	}
	if !response.Msg.GetSuccess() || !response.Msg.GetIntact() || response.Msg.GetTamperDetected() || response.Msg.GetCaseId() != ref.GetCaseId() || response.Msg.GetRole() != workspace.ReplaySealRole(ref.GetRole()) || response.Msg.GetSealDigest() != ref.GetSealDigest() {
		return fmt.Errorf("workspace-sandbox replay seal is not intact")
	}
	return nil
}

// LiveFrozenEvaluatorBundleOwner obtains bounded evaluator bytes from the
// client-rehearsal registry. It does not reveal the registry's retained path.
type LiveFrozenEvaluatorBundleOwner struct{}

func (LiveFrozenEvaluatorBundleOwner) ResolveFrozenEvaluatorBundle(ctx context.Context, ref *validationv1.FrozenEvaluatorBundleReference) (FrozenEvaluatorBundle, error) {
	url, err := discovery.ResolveScenarioURLDefault(ctx, "client-rehearsal")
	if err != nil {
		return FrozenEvaluatorBundle{}, fmt.Errorf("resolve client-rehearsal URL: %w", err)
	}
	client := bountycasesconnect.NewBountyCaseServiceClient(http.DefaultClient, url)
	response, err := client.GetFrozenEvaluatorBundle(ctx, connect.NewRequest(&bountycases.GetFrozenEvaluatorBundleRequest{EvaluatorBundleId: ref.GetEvaluatorBundleId()}))
	if err != nil {
		return FrozenEvaluatorBundle{}, err
	}
	owner := response.Msg.GetEvaluatorBundle()
	if owner == nil {
		return FrozenEvaluatorBundle{}, fmt.Errorf("client-rehearsal returned no frozen evaluator bundle")
	}
	bundle := FrozenEvaluatorBundle{
		EvaluatorBundleID: owner.GetEvaluatorBundleId(),
		CaseID:            owner.GetCaseId(),
		EntryPoint:        owner.GetEntrypoint(),
		EvaluatorDigest:   owner.GetEvaluatorDigest(),
		ContentDigest:     owner.GetContentDigest(),
	}
	for _, file := range owner.GetFiles() {
		if file == nil {
			return FrozenEvaluatorBundle{}, fmt.Errorf("client-rehearsal returned a nil evaluator file")
		}
		bundle.Files = append(bundle.Files, FrozenEvaluatorFile{RelativePath: file.GetRelativePath(), Content: append([]byte(nil), file.GetContent()...), SHA256: file.GetSha256()})
	}
	return bundle, nil
}
