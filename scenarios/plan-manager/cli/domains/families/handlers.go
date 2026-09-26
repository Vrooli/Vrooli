package families

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"connectrpc.com/connect"
	"github.com/vrooli/cli-core/cliapp"
	familiesv1 "github.com/vrooli/vrooli/packages/proto/gen/go/plan-manager/v1/families"
	familiesconnect "github.com/vrooli/vrooli/packages/proto/gen/go/plan-manager/v1/families/families_v1connect"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

type handlers struct {
	client familiesconnect.FamiliesServiceClient
}

func newHandlers(core *cliapp.ScenarioApp) *handlers {
	httpClient, baseURL := cliapp.NewConnectHTTPClient(core)
	return &handlers{client: familiesconnect.NewFamiliesServiceClient(httpClient, baseURL)}
}
func number(ctx cliapp.RunContext, name string) (uint64, error) {
	raw := strings.TrimSpace(ctx.Flag(name))
	if raw == "" {
		return 0, nil
	}
	n, err := strconv.ParseUint(raw, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("--%s must be a non-negative integer", name)
	}
	return n, nil
}
func decode(raw, label string, target proto.Message) error {
	if err := protojson.Unmarshal([]byte(raw), target); err != nil {
		return fmt.Errorf("parse --%s JSON: %w", label, err)
	}
	return nil
}
func renderMutation(ctx cliapp.RunContext, msg proto.Message, text string) error {
	return cliapp.RenderProtoMutation(ctx, msg, cliapp.MutationReport{Result: []string{text}})
}

func (h *handlers) create(ctx cliapp.RunContext) error {
	max, err := number(ctx, "max-parallel")
	if err != nil {
		return err
	}
	resp, err := h.client.CreateFamily(context.Background(), connect.NewRequest(&familiesv1.CreateFamilyRequest{Slug: ctx.Flag("slug"), Outcome: ctx.Flag("outcome"), SharedContext: ctx.Flag("context"), Policy: &familiesv1.FamilyPolicy{MaximumParallelPlans: uint32(max), UnknownInteractionsSequential: true, RequireReviewBeforeLaunch: true}}))
	if err != nil {
		return cliapp.WrapAPIError("create family", err, nil)
	}
	return renderMutation(ctx, resp.Msg, "Family created: "+resp.Msg.GetFamily().GetFamilyId())
}
func (h *handlers) get(ctx cliapp.RunContext) error {
	resp, err := h.client.GetFamily(context.Background(), connect.NewRequest(&familiesv1.GetFamilyRequest{FamilyId: ctx.Positional("family")}))
	if err != nil {
		return cliapp.WrapAPIError("get family", err, nil)
	}
	return cliapp.RenderProtoList(ctx, resp.Msg, cliapp.ListReport{Summary: []string{"Family loaded."}})
}
func (h *handlers) list(ctx cliapp.RunContext) error {
	size, err := number(ctx, "page-size")
	if err != nil {
		return err
	}
	resp, err := h.client.ListFamilies(context.Background(), connect.NewRequest(&familiesv1.ListFamiliesRequest{PageSize: uint32(size), PageToken: ctx.Flag("page-token")}))
	if err != nil {
		return cliapp.WrapAPIError("list families", err, nil)
	}
	return cliapp.RenderProtoList(ctx, resp.Msg, cliapp.ListReport{Summary: []string{fmt.Sprintf("Found %d family record(s).", len(resp.Msg.GetFamilies()))}})
}
func (h *handlers) update(ctx cliapp.RunContext) error {
	rev, err := number(ctx, "revision")
	if err != nil {
		return err
	}
	resp, err := h.client.UpdateFamily(context.Background(), connect.NewRequest(&familiesv1.UpdateFamilyRequest{FamilyId: ctx.Positional("family"), ExpectedRevision: rev, Outcome: ctx.Flag("outcome"), SharedContext: ctx.Flag("context")}))
	if err != nil {
		return cliapp.WrapAPIError("update family", err, nil)
	}
	return renderMutation(ctx, resp.Msg, "Family updated.")
}
func (h *handlers) putMember(ctx cliapp.RunContext) error {
	rev, err := number(ctx, "revision")
	if err != nil {
		return err
	}
	value := &familiesv1.FamilyMember{}
	if err = decode(ctx.Flag("member-json"), "member-json", value); err != nil {
		return err
	}
	resp, err := h.client.PutMember(context.Background(), connect.NewRequest(&familiesv1.PutMemberRequest{FamilyId: ctx.Positional("family"), ExpectedRevision: rev, Member: value}))
	if err != nil {
		return cliapp.WrapAPIError("put family member", err, nil)
	}
	return renderMutation(ctx, resp.Msg, "Family member stored.")
}
func (h *handlers) removeMember(ctx cliapp.RunContext) error {
	rev, err := number(ctx, "revision")
	if err != nil {
		return err
	}
	resp, err := h.client.RemoveMember(context.Background(), connect.NewRequest(&familiesv1.RemoveMemberRequest{FamilyId: ctx.Positional("family"), ExpectedRevision: rev, PlanId: ctx.Positional("plan")}))
	if err != nil {
		return cliapp.WrapAPIError("remove family member", err, nil)
	}
	return renderMutation(ctx, resp.Msg, "Family member removed.")
}
func (h *handlers) putClaim(ctx cliapp.RunContext) error {
	rev, err := number(ctx, "revision")
	if err != nil {
		return err
	}
	value := &familiesv1.ResourceClaim{}
	if err = decode(ctx.Flag("claim-json"), "claim-json", value); err != nil {
		return err
	}
	resp, err := h.client.PutClaim(context.Background(), connect.NewRequest(&familiesv1.PutClaimRequest{FamilyId: ctx.Positional("family"), ExpectedRevision: rev, Claim: value}))
	if err != nil {
		return cliapp.WrapAPIError("put family claim", err, nil)
	}
	return renderMutation(ctx, resp.Msg, "Resource claim stored.")
}
func (h *handlers) removeClaim(ctx cliapp.RunContext) error {
	rev, err := number(ctx, "revision")
	if err != nil {
		return err
	}
	resp, err := h.client.RemoveClaim(context.Background(), connect.NewRequest(&familiesv1.RemoveClaimRequest{FamilyId: ctx.Positional("family"), ExpectedRevision: rev, ClaimId: ctx.Positional("claim")}))
	if err != nil {
		return cliapp.WrapAPIError("remove family claim", err, nil)
	}
	return renderMutation(ctx, resp.Msg, "Resource claim removed.")
}
func edges(values []string, label string) ([]*familiesv1.FamilyEdge, error) {
	out := []*familiesv1.FamilyEdge{}
	for _, raw := range values {
		value := &familiesv1.FamilyEdge{}
		if err := decode(raw, label, value); err != nil {
			return nil, err
		}
		out = append(out, value)
	}
	return out, nil
}
func (h *handlers) propose(ctx cliapp.RunContext) error {
	rev, err := number(ctx, "revision")
	if err != nil {
		return err
	}
	values, err := edges(ctx.FlagValues("edge-json"), "edge-json")
	if err != nil {
		return err
	}
	resp, err := h.client.ProposeGraph(context.Background(), connect.NewRequest(&familiesv1.ProposeGraphRequest{FamilyId: ctx.Positional("family"), ExpectedRevision: rev, ExplicitEdges: values}))
	if err != nil {
		return cliapp.WrapAPIError("propose family graph", err, nil)
	}
	return renderMutation(ctx, resp.Msg, "Graph revision proposed; review is required before launch.")
}
func decision(raw string) (familiesv1.ReviewDecision, error) {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "approve", "approved":
		return familiesv1.ReviewDecision_REVIEW_DECISION_APPROVED, nil
	case "correct", "corrected":
		return familiesv1.ReviewDecision_REVIEW_DECISION_CORRECTED, nil
	case "reject", "rejected":
		return familiesv1.ReviewDecision_REVIEW_DECISION_REJECTED, nil
	default:
		return 0, errors.New("--decision must be approved, corrected, or rejected")
	}
}
func (h *handlers) review(ctx cliapp.RunContext) error {
	rev, err := number(ctx, "revision")
	if err != nil {
		return err
	}
	graphRev, err := number(ctx, "graph-revision")
	if err != nil {
		return err
	}
	d, err := decision(ctx.Flag("decision"))
	if err != nil {
		return err
	}
	corrected, err := edges(ctx.FlagValues("edge-json"), "edge-json")
	if err != nil {
		return err
	}
	resp, err := h.client.ReviewGraph(context.Background(), connect.NewRequest(&familiesv1.ReviewGraphRequest{FamilyId: ctx.Positional("family"), ExpectedRevision: rev, GraphRevision: graphRev, Decision: d, Reviewer: ctx.Flag("reviewer"), Rationale: ctx.Flag("rationale"), CorrectedEdges: corrected}))
	if err != nil {
		return cliapp.WrapAPIError("review family graph", err, nil)
	}
	return renderMutation(ctx, resp.Msg, "Graph review recorded immutably.")
}
func (h *handlers) frontier(ctx cliapp.RunContext) error {
	resp, err := h.client.GetFrontier(context.Background(), connect.NewRequest(&familiesv1.GetFrontierRequest{FamilyId: ctx.Positional("family")}))
	if err != nil {
		return cliapp.WrapAPIError("get family frontier", err, nil)
	}
	return cliapp.RenderProtoList(ctx, resp.Msg, cliapp.ListReport{Summary: []string{fmt.Sprintf("Launchable: %t.", resp.Msg.GetLaunchable())}})
}

func (h *handlers) render(ctx cliapp.RunContext) error {
	resp, err := h.client.RenderFamily(context.Background(), connect.NewRequest(&familiesv1.RenderFamilyRequest{FamilyId: ctx.Positional("family")}))
	if err != nil {
		return cliapp.WrapAPIError("render family", err, nil)
	}
	if ctx.JSON() {
		return cliapp.PrintProtoJSON(ctx.Stdout(), resp.Msg)
	}
	fmt.Fprint(ctx.Stdout(), resp.Msg.GetMarkdown())
	return nil
}
