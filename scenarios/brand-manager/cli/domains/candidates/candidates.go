// Package candidates is the CLI surface for logo candidates.
package candidates

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"connectrpc.com/connect"
	"github.com/vrooli/cli-core/cliapp"

	candsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/brand-manager/v1/candidates"
	candsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/brand-manager/v1/candidates/candidates_v1connect"
)

// GroupName is the command group name.
const GroupName = "candidates"

// longRPCTimeout covers explore, refine and pick, which wait on image
// generation, vectorize and rasterize jobs. The server bounds an explore round
// at 15 minutes and keeps every stored candidate even if this client gives up.
const longRPCTimeout = 20 * time.Minute

type handlers struct {
	client candsconnect.CandidatesServiceClient
	// long is the same service with longRPCTimeout, for the image-job RPCs.
	long candsconnect.CandidatesServiceClient
}

func newHandlers(core *cliapp.ScenarioApp) *handlers {
	httpClient, baseURL := cliapp.NewConnectHTTPClient(core)
	longClient, longBase := cliapp.NewConnectHTTPClientWithTimeout(core, longRPCTimeout)
	return &handlers{
		client: candsconnect.NewCandidatesServiceClient(httpClient, baseURL),
		long:   candsconnect.NewCandidatesServiceClient(longClient, longBase),
	}
}

// Register builds the candidates command group.
func Register(core *cliapp.ScenarioApp) cliapp.SubcommandGroup {
	h := newHandlers(core)
	cmd := func(name, desc string, positionals []cliapp.Positional, flags []cliapp.Flag, run func(cliapp.RunContext) error) cliapp.Command {
		return cliapp.Command{Name: name, Description: desc, NeedsAPI: true, Args: cliapp.ArgSchema{Positionals: positionals, Flags: flags}, RunCtx: run}
	}
	return cliapp.SubcommandGroup{
		Name:        GroupName,
		Description: "Explore, import, refine, pick and reject logo candidates",
		NeedsAPI:    true,
		Subcommands: []cliapp.Command{
			cmd("explore", "Explore logo candidates (concepts × variations)", nil, []cliapp.Flag{
				{Name: "brand-id", Required: true, Description: "Brand id"},
				{Name: "brief", Description: "What the product is"},
				{Name: "concept", Repeated: true, Description: "A concept to explore (repeatable)"},
				{Name: "variations", Description: "Variations per concept (1-4)"},
				{Name: "style-reference-brand", Description: "Match this brand's approved mark (id or slug), e.g. the product line's first product"},
				{Name: "style-reference-asset", Description: "Match this asset's style (ignored with --style-reference-brand)"},
				{Name: "role", Description: "Override the OpenRouter role (default: illustration role; edit role with a style reference)"},
				{Name: "prefer-vector", Bool: true, Description: "Render with the flat SVG-native role instead of the illustration role"},
				{Name: "byok", Bool: true, Description: "Deprecated: the cloud tier is permitted unless --fallback-policy local_only"},
				{Name: "quality-policy", Description: "quality|balanced|fast (default quality)"},
				{Name: "fallback-policy", Description: "any|cloud_allowed|local_only (default any)"},
			}, h.explore),
			cmd("import", "Import an existing asset as a candidate", nil, []cliapp.Flag{
				{Name: "brand-id", Required: true, Description: "Brand id"},
				{Name: "asset-id", Required: true, Description: "Existing asset id"},
				{Name: "concept", Description: "Concept label"},
				{Name: "parent", Description: "Parent candidate id"},
				{Name: "note", Description: "Note"},
			}, h.importCandidate),
			cmd("list", "List candidates for a brand", nil, []cliapp.Flag{
				{Name: "brand-id", Required: true, Description: "Brand id"},
				{Name: "status", Description: "PROPOSED|PICKED|REJECTED|SUPERSEDED"},
				{Name: "limit", Description: "Maximum rows"},
				{Name: "offset", Description: "Rows to skip"},
			}, h.list),
			cmd("get", "Get one candidate", []cliapp.Positional{{Name: "id", Required: true, Description: "Candidate id"}}, nil, h.get),
			cmd("refine", "Refine a candidate into a new child", []cliapp.Positional{{Name: "candidate-id", Required: true, Description: "Parent candidate id"}}, []cliapp.Flag{
				{Name: "instruction", Description: "Instruction edit"},
				{Name: "mask-asset", Description: "Mask asset id for object removal"},
				{Name: "remove-background", Bool: true, Description: "Remove the background"},
				{Name: "vectorize", Bool: true, Description: "Vectorize into SVG"},
				{Name: "keep-color", Repeated: true, Description: "Vectorize: pin a colour"},
				{Name: "inset-px", Description: "Vectorize: clip inset px"},
				{Name: "tolerance-px", Description: "Vectorize: simplification tolerance px"},
			}, h.refine),
			cmd("pick", "Pick a candidate as the brand mark", []cliapp.Positional{{Name: "candidate-id", Required: true, Description: "Candidate id"}}, nil, h.pick),
			cmd("reject", "Reject a candidate", []cliapp.Positional{{Name: "candidate-id", Required: true, Description: "Candidate id"}}, []cliapp.Flag{{Name: "note", Description: "Verdict note"}}, h.reject),
			cmd("restore", "Restore a rejected candidate", []cliapp.Positional{{Name: "candidate-id", Required: true, Description: "Candidate id"}}, nil, h.restore),
		},
	}
}

func (h *handlers) explore(ctx cliapp.RunContext) error {
	resp, err := h.long.ExploreCandidates(context.Background(), connect.NewRequest(&candsv1.ExploreCandidatesRequest{
		BrandId:               ctx.Flag("brand-id"),
		Brief:                 ctx.Flag("brief"),
		Concepts:              ctx.FlagValues("concept"),
		Variations:            int32(atoiOrZero(ctx.Flag("variations"))),
		PreferVector:          ctx.BoolFlag("prefer-vector"),
		AllowByok:             ctx.BoolFlag("byok"),
		QualityPolicy:         ctx.Flag("quality-policy"),
		FallbackPolicy:        ctx.Flag("fallback-policy"),
		Role:                  ctx.Flag("role"),
		StyleReferenceBrand:   ctx.Flag("style-reference-brand"),
		StyleReferenceAssetId: ctx.Flag("style-reference-asset"),
	}))
	if err != nil {
		return cliapp.WrapAPIError("explore candidates", err, nil)
	}
	report := candidatesReport(resp.Msg.GetCandidates())
	for _, w := range resp.Msg.GetWarnings() {
		report.Summary = append(report.Summary, "warning: "+w)
	}
	return cliapp.RenderProtoList(ctx, resp.Msg, report)
}

func (h *handlers) importCandidate(ctx cliapp.RunContext) error {
	resp, err := h.client.ImportCandidate(context.Background(), connect.NewRequest(&candsv1.ImportCandidateRequest{
		BrandId:  ctx.Flag("brand-id"),
		AssetId:  ctx.Flag("asset-id"),
		Concept:  ctx.Flag("concept"),
		ParentId: ctx.Flag("parent"),
		Note:     ctx.Flag("note"),
	}))
	if err != nil {
		return cliapp.WrapAPIError("import candidate", err, nil)
	}
	return cliapp.RenderProtoList(ctx, resp.Msg, candidatesReport([]*candsv1.LogoCandidate{resp.Msg.GetCandidate()}))
}

func (h *handlers) list(ctx cliapp.RunContext) error {
	resp, err := h.client.ListCandidates(context.Background(), connect.NewRequest(&candsv1.ListCandidatesRequest{
		BrandId: ctx.Flag("brand-id"),
		Status:  statusFromString(ctx.Flag("status")),
		Limit:   int32(atoiOrZero(ctx.Flag("limit"))),
		Offset:  int32(atoiOrZero(ctx.Flag("offset"))),
	}))
	if err != nil {
		return cliapp.WrapAPIError("list candidates", err, nil)
	}
	return cliapp.RenderProtoList(ctx, resp.Msg, candidatesReport(resp.Msg.GetCandidates()))
}

func (h *handlers) get(ctx cliapp.RunContext) error {
	resp, err := h.client.GetCandidate(context.Background(), connect.NewRequest(&candsv1.GetCandidateRequest{Id: ctx.Positional("id")}))
	if err != nil {
		return cliapp.WrapAPIError("get candidate", err, nil)
	}
	c := resp.Msg.GetCandidate()
	report := candidatesReport([]*candsv1.LogoCandidate{c})
	report.Results = append(report.Results,
		"  origin: "+c.GetOrigin().String(),
		"  media: "+c.GetMediaType(),
		"  role: "+c.GetRole(),
		"  model: "+c.GetModel(),
		"  parent: "+c.GetParentId(),
		"  note: "+c.GetNote(),
		"  prompt: "+c.GetPrompt(),
	)
	return cliapp.RenderProtoList(ctx, resp.Msg, report)
}

func (h *handlers) refine(ctx cliapp.RunContext) error {
	req := &candsv1.RefineCandidateRequest{CandidateId: ctx.Positional("candidate-id")}
	switch {
	case ctx.BoolFlag("vectorize"):
		req.Action = &candsv1.RefineCandidateRequest_Vectorize{Vectorize: &candsv1.VectorizeOptions{
			KeepColors:                 ctx.FlagValues("keep-color"),
			ClipToLargestRoundedRegion: true,
			InsetPx:                    f64OrZero(ctx.Flag("inset-px")),
			TolerancePx:                f64OrZero(ctx.Flag("tolerance-px")),
		}}
	case ctx.BoolFlag("remove-background"):
		req.Action = &candsv1.RefineCandidateRequest_RemoveBackground{RemoveBackground: true}
	case ctx.Flag("mask-asset") != "":
		req.Action = &candsv1.RefineCandidateRequest_MaskAssetId{MaskAssetId: ctx.Flag("mask-asset")}
	default:
		req.Action = &candsv1.RefineCandidateRequest_Instruction{Instruction: ctx.Flag("instruction")}
	}
	resp, err := h.long.RefineCandidate(context.Background(), connect.NewRequest(req))
	if err != nil {
		return cliapp.WrapAPIError("refine candidate", err, nil)
	}
	return cliapp.RenderProtoList(ctx, resp.Msg, candidatesReport([]*candsv1.LogoCandidate{resp.Msg.GetCandidate()}))
}

func (h *handlers) pick(ctx cliapp.RunContext) error {
	resp, err := h.long.PickCandidate(context.Background(), connect.NewRequest(&candsv1.PickCandidateRequest{CandidateId: ctx.Positional("candidate-id")}))
	if err != nil {
		return cliapp.WrapAPIError("pick candidate", err, nil)
	}
	return cliapp.RenderProtoMutation(ctx, resp.Msg, cliapp.MutationReport{
		Result:  []string{fmt.Sprintf("Picked candidate %s (mark asset %s)", resp.Msg.GetCandidate().GetId(), resp.Msg.GetMarkAssetId())},
		Changes: []string{fmt.Sprintf("vectorized=%t", resp.Msg.GetVectorized())},
	})
}

func (h *handlers) reject(ctx cliapp.RunContext) error {
	resp, err := h.client.RejectCandidate(context.Background(), connect.NewRequest(&candsv1.RejectCandidateRequest{CandidateId: ctx.Positional("candidate-id"), Note: ctx.Flag("note")}))
	if err != nil {
		return cliapp.WrapAPIError("reject candidate", err, nil)
	}
	return cliapp.RenderProtoList(ctx, resp.Msg, candidatesReport([]*candsv1.LogoCandidate{resp.Msg.GetCandidate()}))
}

func (h *handlers) restore(ctx cliapp.RunContext) error {
	resp, err := h.client.RestoreCandidate(context.Background(), connect.NewRequest(&candsv1.RestoreCandidateRequest{CandidateId: ctx.Positional("candidate-id")}))
	if err != nil {
		return cliapp.WrapAPIError("restore candidate", err, nil)
	}
	return cliapp.RenderProtoList(ctx, resp.Msg, candidatesReport([]*candsv1.LogoCandidate{resp.Msg.GetCandidate()}))
}

// candidatesReport is the human view; --json emits the typed response instead.
func candidatesReport(list []*candsv1.LogoCandidate) cliapp.ListReport {
	rows := make([]string, 0, len(list))
	for _, c := range list {
		status := strings.TrimPrefix(c.GetStatus().String(), "CANDIDATE_STATUS_")
		origin := strings.TrimPrefix(c.GetOrigin().String(), "CANDIDATE_ORIGIN_")
		rows = append(rows, fmt.Sprintf("%s  %-10s %-18s %-26s %s", c.GetId(), status, origin, truncate(c.GetConcept(), 26), c.GetModel()))
	}
	return cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("%d candidate(s).", len(list))},
		ResultsHeading: "Candidates",
		Results:        rows,
		RetrievalHints: []string{"`candidates get <id>` — prompt, role, model and lineage", "`candidates pick <id>` — promote the chosen mark"},
	}
}

func truncate(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return string(r[:n-1]) + "…"
}

func statusFromString(s string) candsv1.CandidateStatus {
	switch s {
	case "PROPOSED":
		return candsv1.CandidateStatus_CANDIDATE_STATUS_PROPOSED
	case "PICKED":
		return candsv1.CandidateStatus_CANDIDATE_STATUS_PICKED
	case "REJECTED":
		return candsv1.CandidateStatus_CANDIDATE_STATUS_REJECTED
	case "SUPERSEDED":
		return candsv1.CandidateStatus_CANDIDATE_STATUS_SUPERSEDED
	default:
		return candsv1.CandidateStatus_CANDIDATE_STATUS_UNSPECIFIED
	}
}

func atoiOrZero(s string) int {
	n, err := strconv.Atoi(s)
	if err != nil {
		return 0
	}
	return n
}

func f64OrZero(s string) float64 {
	v, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return 0
	}
	return v
}
