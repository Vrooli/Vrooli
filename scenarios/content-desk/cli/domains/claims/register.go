// Package claims exposes the reusable claim-evidence lifecycle for operators and agents.
package claims

import (
	"context"
	"fmt"
	"strconv"

	"connectrpc.com/connect"
	"github.com/vrooli/cli-core/cliapp"
	claimsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/content-desk/v1/claims"
	claimsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/content-desk/v1/claims/claims_v1connect"
)

const GroupName = "claims"

type handlers struct {
	client claimsconnect.ClaimsServiceClient
}

func newHandlers(core *cliapp.ScenarioApp) *handlers {
	httpClient, baseURL := cliapp.NewConnectHTTPClient(core)
	return &handlers{client: claimsconnect.NewClaimsServiceClient(httpClient, baseURL)}
}

func (h *handlers) listCall(_ cliapp.OperationContext) (*claimsv1.ListClaimsResponse, error) {
	response, err := h.client.ListClaims(context.Background(), connect.NewRequest(&claimsv1.ListClaimsRequest{}))
	if err != nil {
		return nil, cliapp.WrapAPIError("list claims", err, nil)
	}
	if response == nil || response.Msg == nil {
		return nil, fmt.Errorf("server returned no claims response")
	}
	return response.Msg, nil
}

func (h *handlers) listReport(_ cliapp.OperationContext, message *claimsv1.ListClaimsResponse) cliapp.ListReport {
	results := make([]string, 0, len(message.Claims))
	for _, claim := range message.Claims {
		results = append(results, fmt.Sprintf("%s — %s (%s)", claim.Id, claim.Statement, claim.VerificationStatus))
	}
	return cliapp.ListReport{Summary: []string{fmt.Sprintf("Found %d claim(s).", len(message.Claims))}, ResultsHeading: "Claims", Results: results}
}

func (h *handlers) listDraftCall(ctx cliapp.OperationContext) (*claimsv1.ListDraftClaimsResponse, error) {
	response, err := h.client.ListDraftClaims(context.Background(), connect.NewRequest(&claimsv1.ListDraftClaimsRequest{DraftId: ctx.Positional("draft-id")}))
	if err != nil {
		return nil, cliapp.WrapAPIError("list draft claims", err, nil)
	}
	if response == nil || response.Msg == nil {
		return nil, fmt.Errorf("server returned no draft claims response")
	}
	return response.Msg, nil
}
func (h *handlers) listDraftReport(_ cliapp.OperationContext, message *claimsv1.ListDraftClaimsResponse) cliapp.ListReport {
	results := make([]string, 0, len(message.Claims))
	for _, claim := range message.Claims {
		results = append(results, fmt.Sprintf("%s — %s (%s)", claim.Id, claim.Statement, claim.VerificationStatus))
	}
	return cliapp.ListReport{Summary: []string{fmt.Sprintf("Found %d cited claim(s).", len(message.Claims))}, ResultsHeading: "Cited claims", Results: results}
}

func (h *handlers) createCall(ctx cliapp.OperationContext) (*claimsv1.CreateClaimResponse, error) {
	response, err := h.client.CreateClaim(context.Background(), connect.NewRequest(&claimsv1.CreateClaimRequest{Statement: ctx.Flag("statement"), Kind: ctx.Flag("kind"), EvidenceKind: ctx.Flag("evidence-kind"), Reference: ctx.Flag("reference"), Command: ctx.Flag("command"), ExpectedResult: ctx.Flag("expected-result"), ObservedAt: ctx.Flag("observed-at")}))
	if err != nil {
		return nil, cliapp.WrapAPIError("create claim", err, nil)
	}
	if response == nil || response.Msg == nil || response.Msg.Claim == nil {
		return nil, fmt.Errorf("server returned no claim")
	}
	return response.Msg, nil
}

func (h *handlers) createReport(_ cliapp.OperationContext, message *claimsv1.CreateClaimResponse) cliapp.MutationReport {
	return cliapp.MutationReport{Result: []string{fmt.Sprintf("Created claim %s.", message.Claim.Id)}, Changes: []string{fmt.Sprintf("kind=%s status=%s", message.Claim.Kind, message.Claim.VerificationStatus)}}
}

func (h *handlers) citeCall(ctx cliapp.OperationContext) (*claimsv1.CiteClaimResponse, error) {
	start, err := strconv.ParseInt(ctx.Flag("span-start"), 10, 32)
	if err != nil {
		return nil, fmt.Errorf("parse span-start: %w", err)
	}
	end, err := strconv.ParseInt(ctx.Flag("span-end"), 10, 32)
	if err != nil {
		return nil, fmt.Errorf("parse span-end: %w", err)
	}
	response, err := h.client.CiteClaim(context.Background(), connect.NewRequest(&claimsv1.CiteClaimRequest{DraftId: ctx.Flag("draft"), ClaimId: ctx.Flag("claim"), SpanStart: int32(start), SpanEnd: int32(end), Body: ctx.Flag("body")}))
	if err != nil {
		return nil, cliapp.WrapAPIError("cite claim", err, nil)
	}
	if response == nil || response.Msg == nil {
		return nil, fmt.Errorf("server returned no citation response")
	}
	return response.Msg, nil
}

func (h *handlers) citeReport(_ cliapp.OperationContext, _ *claimsv1.CiteClaimResponse) cliapp.MutationReport {
	return cliapp.MutationReport{Result: []string{"Attached claim citation."}}
}

func (h *handlers) verifyCall(ctx cliapp.OperationContext) (*claimsv1.VerifyClaimResponse, error) {
	response, err := h.client.VerifyClaim(context.Background(), connect.NewRequest(&claimsv1.VerifyClaimRequest{Id: ctx.Positional("id")}))
	if err != nil {
		return nil, cliapp.WrapAPIError("verify claim", err, nil)
	}
	if response == nil || response.Msg == nil || response.Msg.Claim == nil {
		return nil, fmt.Errorf("server returned no verified claim")
	}
	return response.Msg, nil
}

func (h *handlers) verifyReport(_ cliapp.OperationContext, message *claimsv1.VerifyClaimResponse) cliapp.MutationReport {
	return cliapp.MutationReport{Result: []string{fmt.Sprintf("Claim %s is %s.", message.Claim.Id, message.Claim.VerificationStatus)}}
}

func (h *handlers) sweepCall(_ cliapp.OperationContext) (*claimsv1.SweepClaimsResponse, error) {
	response, err := h.client.SweepClaims(context.Background(), connect.NewRequest(&claimsv1.SweepClaimsRequest{}))
	if err != nil {
		return nil, cliapp.WrapAPIError("sweep claims", err, nil)
	}
	if response == nil || response.Msg == nil {
		return nil, fmt.Errorf("server returned no sweep response")
	}
	return response.Msg, nil
}

func (h *handlers) sweepReport(_ cliapp.OperationContext, message *claimsv1.SweepClaimsResponse) cliapp.MutationReport {
	return cliapp.MutationReport{Result: []string{fmt.Sprintf("Verified %d check-backed claim(s).", len(message.Claims))}}
}

func Register(core *cliapp.ScenarioApp, manifest []byte) (cliapp.SubcommandGroup, error) {
	h := newHandlers(core)
	group, err := cliapp.LoadFromManifestPrimitives(manifest, GroupName, map[string]cliapp.PrimitiveHandler{
		"ClaimsService.ListClaims":      cliapp.ProtoList(h.listCall, h.listReport),
		"ClaimsService.ListDraftClaims": cliapp.ProtoList(h.listDraftCall, h.listDraftReport),
		"ClaimsService.CreateClaim":     cliapp.ProtoMutation(h.createCall, h.createReport),
		"ClaimsService.CiteClaim":       cliapp.ProtoMutation(h.citeCall, h.citeReport),
		"ClaimsService.VerifyClaim":     cliapp.ProtoMutation(h.verifyCall, h.verifyReport),
		"ClaimsService.SweepClaims":     cliapp.ProtoMutation(h.sweepCall, h.sweepReport),
	})
	if err != nil {
		return cliapp.SubcommandGroup{}, fmt.Errorf("claims: load from manifest: %w", err)
	}
	return group, nil
}
