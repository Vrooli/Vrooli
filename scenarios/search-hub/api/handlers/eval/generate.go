package eval

import (
	"context"
	"errors"
	"fmt"

	"connectrpc.com/connect"
	"google.golang.org/protobuf/proto"

	"search-hub/internal/corpusgen"
	internaleval "search-hub/internal/eval"

	evalv1 "github.com/vrooli/vrooli/packages/proto/gen/go/search-hub/v1/eval"
)

// Generate proposes machine-generated golden cases for a suite by sampling the
// provider's index and inverting items to queries (+ optional hard negatives),
// de-duped against the existing corpus. PREVIEW by default: the proposals are
// returned and nothing is persisted. With apply=true the proposals are appended
// to the suite (each marked tags:["generated"], so the sweep holds them out of
// tuning) and the suite is upserted.
//
// A failure here is almost always a caller-correctable precondition (no such
// suite, the suite's provider is unregistered, or the index could not be
// sampled) — not an internal fault. The resulting corpus's adequacy is always
// computed and returned (warn-level) so an operator sees whether the augmented
// corpus is now adequate.
func (h *connectHandler) Generate(ctx context.Context, req *connect.Request[evalv1.GenerateRequest]) (*connect.Response[evalv1.GenerateResponse], error) {
	if h.deps.Generator == nil {
		return nil, connect.NewError(connect.CodeUnimplemented, errors.New("corpus generation is not configured on this server"))
	}
	suiteID := req.Msg.GetSuiteId()
	suite, err := h.deps.Store.GetSuite(ctx, suiteID)
	if err != nil {
		return nil, h.logged("eval.Generate.getSuite", suiteID, err)
	}
	desc, err := h.deps.Providers.Get(ctx, suite.GetProviderId())
	if err != nil {
		h.deps.Logger.Printf("eval.Generate(%q) resolve provider %q: %v", suiteID, suite.GetProviderId(), err)
		return nil, connect.NewError(connect.CodeFailedPrecondition,
			fmt.Errorf("provider %q is not registered (register it before generating cases)", suite.GetProviderId()))
	}

	res, err := h.deps.Generator.Generate(ctx, suite, desc, corpusgen.Options{
		Count:     int(req.Msg.GetCount()),
		Negatives: req.Msg.GetNegatives(),
	})
	if err != nil {
		h.deps.Logger.Printf("eval.Generate(%q): %v", suiteID, err)
		return nil, connect.NewError(connect.CodeFailedPrecondition, err)
	}

	proposed := toGeneratedCases(res)

	// The would-be corpus (existing + proposals) — used for adequacy on both the
	// preview and the apply path, so the reported adequacy always reflects the
	// corpus the operator would end up with.
	resulting := mergeProposals(suite, res)

	resp := &evalv1.GenerateResponse{
		SuiteId:    suiteID,
		ProviderId: suite.GetProviderId(),
		Proposed:   proposed,
		Adequacy:   internaleval.CheckAdequacy(resulting, res.Strata),
		Summary:    res.Summary(),
	}

	if req.Msg.GetApply() && len(proposed) > 0 {
		if _, err := h.deps.Store.UpsertSuite(ctx, resulting); err != nil {
			// A validation failure on the merged suite is caller-facing; anything
			// else is an opaque internal fault.
			return nil, toConnectError(err)
		}
		resp.Applied = true
		resp.Suite = resulting
	}

	return connect.NewResponse(resp), nil
}

// toGeneratedCases converts corpusgen proposals into the wire shape.
func toGeneratedCases(res *corpusgen.Result) []*evalv1.GeneratedCase {
	out := make([]*evalv1.GeneratedCase, 0, len(res.Proposed))
	for _, p := range res.Proposed {
		out = append(out, &evalv1.GeneratedCase{
			Case:     p.Case,
			SourceId: p.SourceID,
			Stratum:  p.Stratum,
		})
	}
	return out
}

// mergeProposals returns a deep copy of suite with the proposed cases appended,
// leaving the input suite untouched (the preview path must not mutate stored
// state). updated_at is left for the store's Normalize/Upsert to stamp.
func mergeProposals(suite *evalv1.EvalSuite, res *corpusgen.Result) *evalv1.EvalSuite {
	merged := proto.Clone(suite).(*evalv1.EvalSuite)
	for _, p := range res.Proposed {
		merged.Cases = append(merged.Cases, proto.Clone(p.Case).(*evalv1.EvalCase))
	}
	return merged
}
