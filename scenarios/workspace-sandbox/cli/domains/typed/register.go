package typed

import (
	"context"
	"fmt"

	"connectrpc.com/connect"
	"github.com/vrooli/cli-core/cliapp"
	workspacev1 "github.com/vrooli/vrooli/packages/proto/gen/go/workspace-sandbox/v1/workspace"
	workspaceconnect "github.com/vrooli/vrooli/packages/proto/gen/go/workspace-sandbox/v1/workspace/workspaceconnect"
)

func Register(core *cliapp.ScenarioApp, manifest []byte) ([]cliapp.SubcommandGroup, error) {
	httpClient, baseURL := cliapp.NewConnectHTTPClient(core)
	h := &handlers{client: workspaceconnect.NewWorkspaceSandboxServiceClient(httpClient, baseURL)}
	sandboxGroup, err := cliapp.LoadFromManifestPrimitives(manifest, "sandbox", map[string]cliapp.PrimitiveHandler{
		"WorkspaceSandboxService.CreateSandbox": cliapp.ProtoMutation(h.create, h.createReport),
	})
	if err != nil {
		return nil, err
	}
	changeGroup, err := cliapp.LoadFromManifestPrimitives(manifest, "change", map[string]cliapp.PrimitiveHandler{
		"WorkspaceSandboxService.CaptureReviewSnapshot":     cliapp.ProtoMutation(h.captureReview, h.captureReviewReport),
		"WorkspaceSandboxService.MaterializeReviewSnapshot": cliapp.ProtoMutation(h.materializeReview, h.materializeReviewReport),
		"WorkspaceSandboxService.GetReviewSnapshot":         cliapp.ProtoList(h.showReview, h.showReviewReport),
		"WorkspaceSandboxService.GetReviewFile":             cliapp.ProtoListEmitUnpopulatedJSON(h.reviewFile, h.reviewFileReport),
		"WorkspaceSandboxService.GetSandboxDiff":            cliapp.ProtoList(h.diff, h.diffReport),
		"WorkspaceSandboxService.PromoteSandbox":            cliapp.ProtoMutation(h.promote, h.promoteReport),
	})
	if err != nil {
		return nil, err
	}
	return []cliapp.SubcommandGroup{sandboxGroup, changeGroup}, nil
}

type handlers struct {
	client workspaceconnect.WorkspaceSandboxServiceClient
}

func (h *handlers) create(ctx cliapp.OperationContext) (*workspacev1.CreateSandboxResponse, error) {
	response, err := h.client.CreateSandbox(context.Background(), connect.NewRequest(&workspacev1.CreateSandboxRequest{
		Name:           ctx.Flag("name"),
		ScopePath:      ctx.Flag("scope-path"),
		ProjectRoot:    ctx.Flag("project-root"),
		Owner:          ctx.Flag("owner"),
		ReservedPaths:  ctx.FlagValues("reserved-path"),
		IdempotencyKey: ctx.Flag("idempotency-key"),
	}))
	if err != nil {
		return nil, cliapp.WrapAPIError("create sandbox", err, nil)
	}
	return response.Msg, nil
}

func (*handlers) createReport(_ cliapp.OperationContext, response *workspacev1.CreateSandboxResponse) cliapp.MutationReport {
	if response.GetSandbox() == nil {
		return cliapp.MutationReport{Result: []string{"Sandbox creation returned no sandbox."}}
	}
	sandbox := response.GetSandbox()
	return cliapp.MutationReport{
		Result: []string{fmt.Sprintf("Sandbox created: %s", sandbox.GetSandboxId())},
		Changes: []string{
			"Status: " + sandbox.GetStatus(),
			"Workspace: " + sandbox.GetWorkspaceRoot(),
			"Isolation: " + sandbox.GetIsolationMode(),
		},
	}
}

func (h *handlers) diff(ctx cliapp.OperationContext) (*workspacev1.GetSandboxDiffResponse, error) {
	response, err := h.client.GetSandboxDiff(context.Background(), connect.NewRequest(&workspacev1.GetSandboxDiffRequest{SandboxId: ctx.Positional("sandbox_id")}))
	if err != nil {
		return nil, cliapp.WrapAPIError("get sandbox diff", err, nil)
	}
	return response.Msg, nil
}

func (*handlers) diffReport(_ cliapp.OperationContext, response *workspacev1.GetSandboxDiffResponse) cliapp.ListReport {
	rows := make([]string, 0, len(response.GetFiles()))
	for _, file := range response.GetFiles() {
		rows = append(rows, fmt.Sprintf("%s %s (%d bytes)", file.GetChangeType(), file.GetPath(), file.GetSize()))
	}
	stats := response.GetStats()
	summary := []string{fmt.Sprintf("Sandbox: %s", response.GetSandboxId()), fmt.Sprintf("Files changed: %d", stats.GetFilesChanged())}
	if response.GetPatchSha256() != "" {
		summary = append(summary, "Patch SHA-256: "+response.GetPatchSha256())
	}
	return cliapp.ListReport{Summary: summary, ResultsHeading: "Changes", Results: rows}
}

func (h *handlers) promote(ctx cliapp.OperationContext) (*workspacev1.PromoteSandboxResponse, error) {
	response, err := h.client.PromoteSandbox(context.Background(), connect.NewRequest(&workspacev1.PromoteSandboxRequest{
		SandboxId:            ctx.Positional("sandbox_id"),
		Mode:                 ctx.Flag("mode"),
		Actor:                ctx.Flag("actor"),
		CommitMessage:        ctx.Flag("commit-message"),
		CreateCommit:         ctx.BoolFlag("create-commit"),
		Force:                ctx.BoolFlag("force"),
		OverrideAcceptance:   ctx.BoolFlag("override-acceptance"),
		Confirm:              ctx.BoolFlag("confirm"),
		ExpectedPatchSha256:  ctx.Flag("expected-patch-sha256"),
		ReviewRequestId:      ctx.Flag("review-request-id"),
		ExpectedReviewSha256: ctx.Flag("expected-review-sha256"),
	}))
	if err != nil {
		return nil, cliapp.WrapAPIError("promote sandbox", err, nil)
	}
	return response.Msg, nil
}

func (h *handlers) captureReview(ctx cliapp.OperationContext) (*workspacev1.ReviewSnapshotResponse, error) {
	response, err := h.client.CaptureReviewSnapshot(context.Background(), connect.NewRequest(&workspacev1.CaptureReviewSnapshotRequest{SandboxId: ctx.Positional("sandbox_id"), RequestId: ctx.Positional("request_id"), Paths: ctx.FlagValues("path")}))
	if err != nil {
		return nil, cliapp.WrapAPIError("capture review input", err, nil)
	}
	return response.Msg, nil
}

func (h *handlers) showReview(ctx cliapp.OperationContext) (*workspacev1.ReviewSnapshotResponse, error) {
	response, err := h.client.GetReviewSnapshot(context.Background(), connect.NewRequest(&workspacev1.GetReviewSnapshotRequest{SandboxId: ctx.Positional("sandbox_id"), RequestId: ctx.Positional("request_id")}))
	if err != nil {
		return nil, cliapp.WrapAPIError("read review input", err, nil)
	}
	return response.Msg, nil
}

func (*handlers) captureReviewReport(_ cliapp.OperationContext, response *workspacev1.ReviewSnapshotResponse) cliapp.MutationReport {
	snapshot := response.GetSnapshot()
	return cliapp.MutationReport{Result: []string{"Retained review: " + snapshot.GetRequestId(), "Source SHA-256: " + snapshot.GetSha256()}, Changes: []string{fmt.Sprintf("Candidate files: %d; retained input bytes: %d", len(snapshot.GetAfter()), snapshot.GetInputBytes())}}
}

func (h *handlers) materializeReview(ctx cliapp.OperationContext) (*workspacev1.ReviewWorkspace, error) {
	response, err := h.client.MaterializeReviewSnapshot(context.Background(), connect.NewRequest(&workspacev1.MaterializeReviewSnapshotRequest{SandboxId: ctx.Positional("sandbox_id"), RequestId: ctx.Positional("request_id"), ExpectedSha256: ctx.Flag("expected-sha256")}))
	if err != nil {
		return nil, cliapp.WrapAPIError("materialize review workspace", err, nil)
	}
	return response.Msg, nil
}

func (*handlers) materializeReviewReport(_ cliapp.OperationContext, response *workspacev1.ReviewWorkspace) cliapp.MutationReport {
	return cliapp.MutationReport{Result: []string{"Review input: " + response.GetRoot(), "Source SHA-256: " + response.GetSha256()}, Changes: []string{"Derived only from retained evidence. Bind the root read-only for the reviewer."}}
}

func (*handlers) showReviewReport(_ cliapp.OperationContext, response *workspacev1.ReviewSnapshotResponse) cliapp.ListReport {
	snapshot := response.GetSnapshot()
	rows := make([]string, 0, len(snapshot.GetAfter()))
	for _, file := range snapshot.GetAfter() {
		rows = append(rows, fmt.Sprintf("%s %s (%d bytes, mode %d)", file.GetSha256(), file.GetPath(), file.GetSize(), file.GetMode()))
	}
	return cliapp.ListReport{Summary: []string{"Source SHA-256: " + snapshot.GetSha256(), "Patch SHA-256: " + snapshot.GetPatchSha256()}, ResultsHeading: "Retained candidate", Results: rows}
}

func (h *handlers) reviewFile(ctx cliapp.OperationContext) (*workspacev1.GetReviewFileResponse, error) {
	response, err := h.client.GetReviewFile(context.Background(), connect.NewRequest(&workspacev1.GetReviewFileRequest{SandboxId: ctx.Positional("sandbox_id"), RequestId: ctx.Positional("request_id"), Side: ctx.Flag("side"), Path: ctx.Flag("path")}))
	if err != nil {
		return nil, cliapp.WrapAPIError("read review file", err, nil)
	}
	return response.Msg, nil
}

func (*handlers) reviewFileReport(_ cliapp.OperationContext, response *workspacev1.GetReviewFileResponse) cliapp.ListReport {
	return cliapp.ListReport{Summary: []string{fmt.Sprintf("Retained bytes: %d", len(response.GetContent())), "Use --json for binary-safe base64 content."}}
}

func (*handlers) promoteReport(_ cliapp.OperationContext, response *workspacev1.PromoteSandboxResponse) cliapp.MutationReport {
	return cliapp.MutationReport{
		Result: []string{fmt.Sprintf("Sandbox promotion success=%t applied=%d failed=%d", response.GetSuccess(), response.GetApplied(), response.GetFailed())},
		Changes: []string{
			fmt.Sprintf("Remaining: %d", response.GetRemaining()),
			"Commit: " + response.GetCommitHash(),
		},
	}
}
