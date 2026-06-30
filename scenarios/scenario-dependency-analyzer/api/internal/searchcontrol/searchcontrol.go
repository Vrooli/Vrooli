// Package searchcontrol hosts SDA's implementation of the SHARED, token-gated
// search control plane (search-hub.v1.control.SearchControlService): every SDA
// search leaf (.dependencies, .scenarios, .resources) speaks the same reindex +
// config-write RPCs so search-hub's sweep can drive index-time experiments and
// write a winning tuning back uniformly. WriteConfig/WriteCorpus select the leaf
// by provider_id; Reindex is scenario-scoped (the shared reconciler reconciles
// every corpus binding).
//
// SECURITY MODEL (token-only): the control token (Gate.Token) is the per-provider
// secret search-hub mints at first registration and is its only holder. Every
// control RPC carries it; the gate compares with subtle.ConstantTimeCompare and
// rejects any mismatch (an empty cached or presented token always denies).
package searchcontrol

import (
	"context"
	"crypto/subtle"
	"errors"
	"log"

	"connectrpc.com/connect"
	"github.com/gin-gonic/gin"

	"scenario-dependency-analyzer/internal/aisearch"

	aisearchpkg "github.com/vrooli/ai-go/search"
	searchregister "github.com/vrooli/searchregister-go"
	controlv1 "github.com/vrooli/vrooli/packages/proto/gen/go/search-hub/v1/control"
	controlconnect "github.com/vrooli/vrooli/packages/proto/gen/go/search-hub/v1/control/control_v1connect"
)

// Reindexer is the seam between the control handler and the aisearch reconcile
// job control. Satisfied by ServiceAdapter (wrapping the live multi-corpus
// aisearch service) in production and by a fake in tests.
type Reindexer interface {
	Reindex(ctx context.Context, scope string, dryRun bool) (jobID string, plannedUpserts, plannedDeletes int, err error)
	ReindexStatus(jobID string) (state string, processed, total int, errMsg string, ok bool)
	ReindexCancel(jobID string) bool
	// ApplyTuning swaps the live read engine for the provider's corpus to the new
	// query-time tuning IN PROCESS (no restart). An embed-recipe change is rejected
	// (SDA corpora share one embedder) — that takes effect on restart from the
	// search.json SSOT.
	ApplyTuning(ctx context.Context, providerID string, tuning aisearchpkg.TuningConfig) error
}

// ConfigWriter persists a new tuning block into the provider's search.json SSOT.
type ConfigWriter interface {
	WriteTuning(providerID string, tuning aisearchpkg.TuningConfig, dryRun bool) (effective aisearchpkg.TuningConfig, indexTimeChanged, written bool, err error)
}

// CorpusWriter persists a new tests corpus into the provider's search.json SSOT.
type CorpusWriter interface {
	WriteCorpus(providerID string, suite aisearchpkg.TestSuite, dryRun bool) (effective aisearchpkg.TestSuite, written bool, err error)
}

// Gate is the control-token guard every control RPC passes through. It is
// PER-PROVIDER aware: a provider-scoped RPC (WriteConfig/WriteCorpus) checks the
// presented token against that provider's minted token; a scenario-scoped RPC
// (Reindex/status/cancel) accepts any of the scenario's leaf tokens. It rejects
// (rather than silently degrading) because a control verb has no public fallback.
type Gate struct {
	Tokens *TokenStore
}

// authorizeProvider gates a provider-scoped RPC against that provider's token.
func (g *Gate) authorizeProvider(logger *log.Logger, providerID, presented string) error {
	if g == nil || g.Tokens == nil || presented == "" || !constantTimeEqual(presented, g.Tokens.Get(providerID)) {
		logger.Printf("[scenario-dependency-analyzer] search control denied: control token missing or mismatched for %q", providerID)
		return connect.NewError(connect.CodePermissionDenied, errors.New("search control plane is not available"))
	}
	return nil
}

// authorizeAny gates a scenario-scoped RPC (no provider_id) against any leaf token.
func (g *Gate) authorizeAny(logger *log.Logger, presented string) error {
	if g == nil || g.Tokens == nil || !g.Tokens.Match(presented) {
		logger.Printf("[scenario-dependency-analyzer] search control denied: control token missing or mismatched")
		return connect.NewError(connect.CodePermissionDenied, errors.New("search control plane is not available"))
	}
	return nil
}

// constantTimeEqual compares two tokens without leaking length-independent timing;
// an empty want always returns false (an un-minted token denies).
func constantTimeEqual(presented, want string) bool {
	if want == "" || presented == "" {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(presented), []byte(want)) == 1
}

// Deps wires the seams the control handler needs.
type Deps struct {
	Logger       *log.Logger
	Reindexer    Reindexer
	ConfigWriter ConfigWriter
	CorpusWriter CorpusWriter
	Gate         *Gate
}

type connectHandler struct {
	deps Deps
}

// RegisterConnectRoutes mounts the SearchControlService Connect handler on the
// gin router at its generated procedure paths.
func RegisterConnectRoutes(router *gin.Engine, deps Deps) {
	if deps.Logger == nil {
		deps.Logger = log.Default()
	}
	pattern, handler := controlconnect.NewSearchControlServiceHandler(&connectHandler{deps: deps})
	router.Any(pattern+"*path", gin.WrapH(handler))
}

func (h *connectHandler) Reindex(ctx context.Context, req *connect.Request[controlv1.ReindexRequest]) (*connect.Response[controlv1.ReindexResponse], error) {
	r := req.Msg
	if err := h.deps.Gate.authorizeAny(h.deps.Logger, r.GetControlToken()); err != nil {
		return nil, err
	}
	if h.deps.Reindexer == nil {
		return nil, connect.NewError(connect.CodeUnimplemented, errors.New("reindex backend not configured"))
	}
	jobID, up, del, err := h.deps.Reindexer.Reindex(ctx, r.GetScope(), r.GetDryRun())
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&controlv1.ReindexResponse{
		JobId:          jobID,
		PlannedUpserts: int32(up),
		PlannedDeletes: int32(del),
		DryRun:         r.GetDryRun(),
	}), nil
}

func (h *connectHandler) ReindexStatus(_ context.Context, req *connect.Request[controlv1.ReindexStatusRequest]) (*connect.Response[controlv1.ReindexStatusResponse], error) {
	r := req.Msg
	if err := h.deps.Gate.authorizeAny(h.deps.Logger, r.GetControlToken()); err != nil {
		return nil, err
	}
	if h.deps.Reindexer == nil {
		return nil, connect.NewError(connect.CodeUnimplemented, errors.New("reindex backend not configured"))
	}
	state, processed, total, errMsg, ok := h.deps.Reindexer.ReindexStatus(r.GetJobId())
	if !ok {
		return nil, connect.NewError(connect.CodeNotFound, errors.New("reindex job not found"))
	}
	return connect.NewResponse(&controlv1.ReindexStatusResponse{
		JobId:     r.GetJobId(),
		State:     state,
		Processed: int32(processed),
		Total:     int32(total),
		Error:     errMsg,
	}), nil
}

func (h *connectHandler) ReindexCancel(_ context.Context, req *connect.Request[controlv1.ReindexCancelRequest]) (*connect.Response[controlv1.ReindexCancelResponse], error) {
	r := req.Msg
	if err := h.deps.Gate.authorizeAny(h.deps.Logger, r.GetControlToken()); err != nil {
		return nil, err
	}
	if h.deps.Reindexer == nil {
		return nil, connect.NewError(connect.CodeUnimplemented, errors.New("reindex backend not configured"))
	}
	cancelled := h.deps.Reindexer.ReindexCancel(r.GetJobId())
	return connect.NewResponse(&controlv1.ReindexCancelResponse{
		JobId:     r.GetJobId(),
		Cancelled: cancelled,
	}), nil
}

func (h *connectHandler) WriteConfig(ctx context.Context, req *connect.Request[controlv1.WriteConfigRequest]) (*connect.Response[controlv1.WriteConfigResponse], error) {
	r := req.Msg
	providerID := r.GetProviderId()
	if providerID == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("provider_id is required"))
	}
	if err := h.deps.Gate.authorizeProvider(h.deps.Logger, providerID, r.GetControlToken()); err != nil {
		return nil, err
	}
	if h.deps.ConfigWriter == nil {
		return nil, connect.NewError(connect.CodeUnimplemented, errors.New("config-write backend not configured"))
	}

	cfg := searchregister.TuningFromProto(r.GetTuning()).WithDefaults()
	if err := cfg.Validate(); err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	effective, indexTimeChanged, written, err := h.deps.ConfigWriter.WriteTuning(providerID, cfg, r.GetDryRun())
	if err != nil {
		var notIn aisearchpkg.ErrProviderNotInFile
		if errors.As(err, &notIn) {
			return nil, connect.NewError(connect.CodeNotFound, err)
		}
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	resp := &controlv1.WriteConfigResponse{
		Written:   written,
		Effective: searchregister.TuningToProto(effective),
	}

	// On a written change, swap the live read engine so the new tuning takes
	// effect without a restart. A query-time-only change (rerank/floor) applies
	// cleanly; an index-time (embed-recipe) change is rejected by ApplyTuning and
	// surfaced as an error so the operator knows a restart is required.
	if written && h.deps.Reindexer != nil {
		if err := h.deps.Reindexer.ApplyTuning(ctx, providerID, effective); err != nil {
			if indexTimeChanged {
				return nil, connect.NewError(connect.CodeFailedPrecondition, err)
			}
			return nil, connect.NewError(connect.CodeInternal, err)
		}
	}
	return connect.NewResponse(resp), nil
}

func (h *connectHandler) WriteCorpus(_ context.Context, req *connect.Request[controlv1.WriteCorpusRequest]) (*connect.Response[controlv1.WriteCorpusResponse], error) {
	r := req.Msg
	providerID := r.GetProviderId()
	if providerID == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("provider_id is required"))
	}
	if err := h.deps.Gate.authorizeProvider(h.deps.Logger, providerID, r.GetControlToken()); err != nil {
		return nil, err
	}
	if h.deps.CorpusWriter == nil {
		return nil, connect.NewError(connect.CodeUnimplemented, errors.New("corpus-write backend not configured"))
	}

	suite := searchregister.SuiteFromProto(r.GetCorpus())
	if err := suite.Validate(); err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	effective, written, err := h.deps.CorpusWriter.WriteCorpus(providerID, suite, r.GetDryRun())
	if err != nil {
		var notIn aisearchpkg.ErrProviderNotInFile
		if errors.As(err, &notIn) {
			return nil, connect.NewError(connect.CodeNotFound, err)
		}
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	return connect.NewResponse(&controlv1.WriteCorpusResponse{
		Written:   written,
		Effective: searchregister.SuiteToProto(providerID, effective),
	}), nil
}

var _ controlconnect.SearchControlServiceHandler = (*connectHandler)(nil)

// ServiceAdapter wraps *aisearch.Service to satisfy the Reindexer seam, mapping
// provider_id to the corpus ApplyTuning targets.
type ServiceAdapter struct{ Service *aisearch.Service }

func (a ServiceAdapter) Reindex(ctx context.Context, scope string, dryRun bool) (string, int, int, error) {
	job, err := a.Service.Reindex(ctx, scope, dryRun)
	if err != nil {
		return "", 0, 0, err
	}
	exp := a.Service.JobExport(job)
	up, _ := exp["planned_upserts"].(int)
	del, _ := exp["planned_deletes"].(int)
	return job.ID, up, del, nil
}

func (a ServiceAdapter) ReindexStatus(jobID string) (string, int, int, string, bool) {
	job, ok := a.Service.ReindexStatus(jobID)
	if !ok {
		return "", 0, 0, "", false
	}
	exp := a.Service.JobExport(job)
	state, _ := exp["state"].(string)
	processed, _ := exp["processed"].(int)
	total, _ := exp["total"].(int)
	errMsg, _ := exp["error"].(string)
	return state, processed, total, errMsg, true
}

func (a ServiceAdapter) ReindexCancel(jobID string) bool { return a.Service.ReindexCancel(jobID) }

func (a ServiceAdapter) ApplyTuning(ctx context.Context, providerID string, tuning aisearchpkg.TuningConfig) error {
	corpus, ok := aisearch.CorpusForProvider(providerID)
	if !ok {
		return errors.New("unknown provider_id for this scenario")
	}
	return a.Service.ApplyTuning(ctx, corpus, tuning)
}

// FileConfigWriter persists tuning into the scenario's search.json via the shared
// engine's atomic writer. Path is the absolute path to search.json.
type FileConfigWriter struct{ Path string }

func (w FileConfigWriter) WriteTuning(providerID string, tuning aisearchpkg.TuningConfig, dryRun bool) (aisearchpkg.TuningConfig, bool, bool, error) {
	return aisearchpkg.WriteProviderTuning(w.Path, providerID, tuning, dryRun)
}

// FileCorpusWriter persists a tests corpus into the scenario's search.json.
//
// INVARIANT: onlyScenarioWritesItsFile — search-hub never writes a scenario's
// search.json directly; every write-back arrives through the token-gated RPCs and
// lands HERE, in the scenario's own process.
type FileCorpusWriter struct{ Path string }

func (w FileCorpusWriter) WriteCorpus(providerID string, suite aisearchpkg.TestSuite, dryRun bool) (aisearchpkg.TestSuite, bool, error) {
	return aisearchpkg.WriteProviderCorpus(w.Path, providerID, suite, dryRun)
}
