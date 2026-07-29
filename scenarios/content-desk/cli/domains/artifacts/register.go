// Package artifacts exposes read-only draft visibility for operators and agents.
package artifacts

import (
	"context"
	"fmt"

	"connectrpc.com/connect"
	"github.com/vrooli/cli-core/cliapp"
	artifactsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/content-desk/v1/artifacts"
	artifactsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/content-desk/v1/artifacts/artifacts_v1connect"
)

const GroupName = "artifacts"

type handlers struct {
	client artifactsconnect.ArtifactsServiceClient
}

func newHandlers(core *cliapp.ScenarioApp) *handlers {
	httpClient, baseURL := cliapp.NewConnectHTTPClient(core)
	return &handlers{client: artifactsconnect.NewArtifactsServiceClient(httpClient, baseURL)}
}

func (h *handlers) listCall(_ cliapp.OperationContext) (*artifactsv1.ListDraftsResponse, error) {
	response, err := h.client.ListDrafts(context.Background(), connect.NewRequest(&artifactsv1.ListDraftsRequest{}))
	if err != nil {
		return nil, cliapp.WrapAPIError("list drafts", err, nil)
	}
	if response == nil || response.Msg == nil {
		return nil, fmt.Errorf("server returned no drafts response")
	}
	return response.Msg, nil
}

func (h *handlers) listReport(_ cliapp.OperationContext, message *artifactsv1.ListDraftsResponse) cliapp.ListReport {
	results := make([]string, 0, len(message.Drafts))
	for _, draft := range message.Drafts {
		results = append(results, fmt.Sprintf("%s — campaign=%s status=%s", draft.Id, draft.CampaignId, draft.Status))
	}
	return cliapp.ListReport{Summary: []string{fmt.Sprintf("Found %d draft(s).", len(message.Drafts))}, ResultsHeading: "Drafts", Results: results}
}

func (h *handlers) createCall(ctx cliapp.OperationContext) (*artifactsv1.CreateDraftResponse, error) {
	response, err := h.client.CreateDraft(context.Background(), connect.NewRequest(&artifactsv1.CreateDraftRequest{CampaignId: ctx.Flag("campaign"), PostTypeId: ctx.Flag("post-type"), Body: ctx.Flag("body"), Channel: ctx.Flag("channel"), Format: ctx.Flag("format"), Lane: ctx.Flag("lane"), Sku: ctx.Flag("sku")}))
	if err != nil {
		return nil, cliapp.WrapAPIError("create draft", err, nil)
	}
	if response == nil || response.Msg == nil || response.Msg.Draft == nil {
		return nil, fmt.Errorf("server returned no draft")
	}
	return response.Msg, nil
}

func (h *handlers) createReport(_ cliapp.OperationContext, message *artifactsv1.CreateDraftResponse) cliapp.MutationReport {
	return cliapp.MutationReport{Result: []string{fmt.Sprintf("Created draft %s.", message.Draft.Id)}, Changes: []string{fmt.Sprintf("campaign=%s status=%s", message.Draft.CampaignId, message.Draft.Status)}}
}

func (h *handlers) updateBodyCall(ctx cliapp.OperationContext) (*artifactsv1.UpdateDraftBodyResponse, error) {
	response, err := h.client.UpdateDraftBody(context.Background(), connect.NewRequest(&artifactsv1.UpdateDraftBodyRequest{Id: ctx.Positional("id"), Body: ctx.Flag("body")}))
	if err != nil {
		return nil, cliapp.WrapAPIError("save draft revision", err, nil)
	}
	if response == nil || response.Msg == nil || response.Msg.Draft == nil {
		return nil, fmt.Errorf("server returned no revised draft")
	}
	return response.Msg, nil
}

func (h *handlers) updateBodyReport(_ cliapp.OperationContext, message *artifactsv1.UpdateDraftBodyResponse) cliapp.MutationReport {
	return cliapp.MutationReport{Result: []string{fmt.Sprintf("Saved revision for draft %s.", message.Draft.Id)}}
}

func (h *handlers) submitReleaseCall(ctx cliapp.OperationContext) (*artifactsv1.SubmitReleaseDraftResponse, error) {
	response, err := h.client.SubmitReleaseDraft(context.Background(), connect.NewRequest(&artifactsv1.SubmitReleaseDraftRequest{Id: ctx.Positional("id"), IdentityId: ctx.Flag("identity"), Lane: ctx.Flag("lane"), IdempotencyKey: ctx.Flag("idempotency-key")}))
	if err != nil {
		return nil, cliapp.WrapAPIError("submit release", err, nil)
	}
	if response == nil || response.Msg == nil || response.Msg.Draft == nil {
		return nil, fmt.Errorf("server returned no release submission")
	}
	return response.Msg, nil
}

func (h *handlers) submitReleaseReport(_ cliapp.OperationContext, message *artifactsv1.SubmitReleaseDraftResponse) cliapp.MutationReport {
	return cliapp.MutationReport{Result: []string{fmt.Sprintf("Submitted draft %s to Channel Manager.", message.Draft.Id)}, Changes: []string{fmt.Sprintf("release=%s action=%s status=%s", message.ReleaseId, message.ActionId, message.ReleaseStatus)}}
}

func (h *handlers) transitionCall(ctx cliapp.OperationContext) (*artifactsv1.TransitionDraftResponse, error) {
	response, err := h.client.TransitionDraft(context.Background(), connect.NewRequest(&artifactsv1.TransitionDraftRequest{Id: ctx.Positional("id"), Event: ctx.Flag("event")}))
	if err != nil {
		return nil, cliapp.WrapAPIError("transition draft", err, nil)
	}
	if response == nil || response.Msg == nil || response.Msg.Draft == nil {
		return nil, fmt.Errorf("server returned no transitioned draft")
	}
	return response.Msg, nil
}

func (h *handlers) transitionReport(_ cliapp.OperationContext, message *artifactsv1.TransitionDraftResponse) cliapp.MutationReport {
	return cliapp.MutationReport{Result: []string{fmt.Sprintf("Draft %s is now %s.", message.Draft.Id, message.Draft.Status)}}
}

func (h *handlers) approveCall(ctx cliapp.OperationContext) (*artifactsv1.ApproveDraftResponse, error) {
	response, err := h.client.ApproveDraft(context.Background(), connect.NewRequest(&artifactsv1.ApproveDraftRequest{Id: ctx.Positional("id")}))
	if err != nil {
		return nil, cliapp.WrapAPIError("approve draft", err, nil)
	}
	if response == nil || response.Msg == nil || response.Msg.Draft == nil {
		return nil, fmt.Errorf("server returned no approved draft")
	}
	return response.Msg, nil
}

func (h *handlers) approveReport(_ cliapp.OperationContext, message *artifactsv1.ApproveDraftResponse) cliapp.MutationReport {
	return cliapp.MutationReport{Result: []string{fmt.Sprintf("Draft %s approved.", message.Draft.Id)}}
}

func Register(core *cliapp.ScenarioApp, manifest []byte) (cliapp.SubcommandGroup, error) {
	h := newHandlers(core)
	group, err := cliapp.LoadFromManifestPrimitives(manifest, GroupName, map[string]cliapp.PrimitiveHandler{
		"ArtifactsService.ApproveDraft":       cliapp.ProtoMutation(h.approveCall, h.approveReport),
		"ArtifactsService.CreateDraft":        cliapp.ProtoMutation(h.createCall, h.createReport),
		"ArtifactsService.ListDrafts":         cliapp.ProtoList(h.listCall, h.listReport),
		"ArtifactsService.TransitionDraft":    cliapp.ProtoMutation(h.transitionCall, h.transitionReport),
		"ArtifactsService.UpdateDraftBody":    cliapp.ProtoMutation(h.updateBodyCall, h.updateBodyReport),
		"ArtifactsService.SubmitReleaseDraft": cliapp.ProtoMutation(h.submitReleaseCall, h.submitReleaseReport),
	})
	if err != nil {
		return cliapp.SubcommandGroup{}, fmt.Errorf("artifacts: load from manifest: %w", err)
	}
	return group, nil
}
