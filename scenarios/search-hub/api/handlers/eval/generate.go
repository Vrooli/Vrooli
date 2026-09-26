package eval

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"

	"connectrpc.com/connect"
	"google.golang.org/protobuf/proto"

	"search-hub/internal/control"
	"search-hub/internal/corpusgen"
	internaleval "search-hub/internal/eval"

	registryv1 "github.com/vrooli/vrooli/packages/proto/gen/go/search-hub/v1/registry"

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
		applied, effective, aerr := h.applyCorpus(ctx, desc, resulting)
		if aerr != nil {
			return nil, aerr
		}
		resp.Applied = applied
		resp.Suite = effective
	}

	return connect.NewResponse(resp), nil
}

// PromoteCases flips reviewed candidate cases into the acceptance denominator.
// The mutation goes through the provider's search.json control plane, matching
// generate --apply, so the provider file remains the corpus source of truth and
// search-hub only mirrors the effective corpus returned by the provider.
func (h *connectHandler) PromoteCases(ctx context.Context, req *connect.Request[evalv1.PromoteCasesRequest]) (*connect.Response[evalv1.PromoteCasesResponse], error) {
	if req.Msg.GetAll() && len(req.Msg.GetCaseIds()) > 0 {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("use either --all or --case, not both"))
	}
	if !req.Msg.GetAll() && len(req.Msg.GetCaseIds()) == 0 {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("provide at least one case id or set all=true"))
	}

	suiteID := req.Msg.GetSuiteId()
	suite, err := h.deps.Store.GetSuite(ctx, suiteID)
	if err != nil {
		return nil, h.logged("eval.PromoteCases.getSuite", suiteID, err)
	}
	desc, err := h.deps.Providers.Get(ctx, suite.GetProviderId())
	if err != nil {
		h.deps.Logger.Printf("eval.PromoteCases(%q) resolve provider %q: %v", suiteID, suite.GetProviderId(), err)
		return nil, connect.NewError(connect.CodeFailedPrecondition,
			fmt.Errorf("provider %q is not registered (register it before promoting cases)", suite.GetProviderId()))
	}

	resulting, promoted, already, err := promoteCases(suite, req.Msg.GetCaseIds(), req.Msg.GetAll())
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	resp := &evalv1.PromoteCasesResponse{
		SuiteId:                suiteID,
		ProviderId:             suite.GetProviderId(),
		PromotedCaseIds:        promoted,
		AlreadyReviewedCaseIds: already,
	}
	if len(promoted) == 0 {
		resp.Suite = proto.Clone(suite).(*evalv1.EvalSuite)
		return connect.NewResponse(resp), nil
	}

	applied, effective, aerr := h.applyCorpus(ctx, desc, resulting)
	if aerr != nil {
		return nil, aerr
	}
	resp.Applied = applied
	resp.Suite = effective
	return connect.NewResponse(resp), nil
}

// applyCorpus persists the grown corpus to the provider's search.json SSOT through
// the token-gated WriteCorpus control RPC, then re-registers the returned,
// now-authoritative corpus into the eval store so the store mirror re-syncs with
// the file immediately (it would otherwise re-sync only on the scenario's next
// boot). The store is NEVER the apply target.
//
// INVARIANT: corpusMutationsGoThroughFile
//
//	A corpus mutation (generate --apply) is written to the scenario's search.json
//	via WriteCorpus, not to search-hub's store. The store is re-derived from the
//	file's effective corpus, so the file stays authoritative and no reverse drift
//	is possible.
func (h *connectHandler) applyCorpus(ctx context.Context, desc *registryv1.ProviderDescriptor, resulting *evalv1.EvalSuite) (applied bool, effective *evalv1.EvalSuite, err error) {
	if h.deps.Control == nil || h.deps.Tokens == nil {
		return false, nil, connect.NewError(connect.CodeUnimplemented,
			errors.New("corpus write-back is not configured on this server (preview is available; --apply needs the control plane)"))
	}
	providerID := desc.GetProviderId()
	token, terr := h.deps.Tokens.Token(ctx, providerID)
	if terr != nil {
		h.deps.Logger.Printf("eval.Generate(apply) token for %q: %v", providerID, terr)
		return false, nil, connect.NewError(connect.CodeFailedPrecondition,
			fmt.Errorf("resolve control token for %q: %w", providerID, terr))
	}

	wc, werr := h.deps.Control.WriteCorpus(ctx, desc, token, resulting, false)
	if werr != nil {
		if errors.Is(werr, control.ErrNoControlPlane) {
			return false, nil, connect.NewError(connect.CodeFailedPrecondition,
				fmt.Errorf("provider %q declares no control endpoint; a grown corpus cannot be written back to its search.json", providerID))
		}
		// permission / argument / not-found from the provider already carry a code;
		// surface them verbatim. A bare transport failure is logged + Unavailable.
		h.deps.Logger.Printf("eval.Generate(apply) WriteCorpus %q: %v", providerID, werr)
		return false, nil, werr
	}

	effective = wc.GetEffective()
	// Re-register the file's authoritative corpus into the eval cache so the store
	// mirrors the file without waiting for the scenario's next boot.
	if _, uerr := h.deps.Store.UpsertSuite(ctx, effective); uerr != nil {
		return false, nil, toConnectError(uerr)
	}
	return wc.GetWritten(), effective, nil
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

func promoteCases(suite *evalv1.EvalSuite, caseIDs []string, all bool) (*evalv1.EvalSuite, []string, []string, error) {
	merged := proto.Clone(suite).(*evalv1.EvalSuite)
	want := normalizeCaseIDs(caseIDs)
	if !all {
		if len(want) == 0 {
			return nil, nil, nil, errors.New("provide at least one non-empty case id")
		}
		for id := range want {
			if !suiteHasCase(merged, id) {
				return nil, nil, nil, fmt.Errorf("case %q is not in suite %q", id, suite.GetSuiteId())
			}
		}
	}

	promoted := []string{}
	already := []string{}
	for _, c := range merged.GetCases() {
		if c == nil || (!all && !want[c.GetCaseId()]) {
			continue
		}
		switch strings.TrimSpace(c.GetStatus()) {
		case "", "reviewed":
			if !all {
				already = append(already, c.GetCaseId())
			}
		case "candidate":
			c.Status = "reviewed"
			promoted = append(promoted, c.GetCaseId())
		default:
			return nil, nil, nil, fmt.Errorf("case %q has invalid status %q", c.GetCaseId(), c.GetStatus())
		}
	}
	sort.Strings(promoted)
	sort.Strings(already)
	return merged, promoted, already, nil
}

func normalizeCaseIDs(ids []string) map[string]bool {
	out := make(map[string]bool, len(ids))
	for _, raw := range ids {
		for _, part := range strings.Split(raw, ",") {
			id := strings.TrimSpace(part)
			if id != "" {
				out[id] = true
			}
		}
	}
	return out
}

func suiteHasCase(suite *evalv1.EvalSuite, id string) bool {
	for _, c := range suite.GetCases() {
		if c.GetCaseId() == id {
			return true
		}
	}
	return false
}
