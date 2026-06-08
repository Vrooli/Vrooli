// Package searchcontrol hosts cli-health's implementation of the SHARED,
// token-gated search control plane (search-hub.v1.control.SearchControlService).
// It replaces the old cli-health-private ReindexService proto: every search
// provider now speaks the same reindex + config-write RPCs so search-hub's sweep
// can drive index-time experiments and write back a winning tuning uniformly.
//
// SECURITY MODEL (two independent layers; both must pass):
//   - per-environment flag (Gate.Enabled, default OFF via CLI_HEALTH_SEARCH_
//     CONTROL_ENABLED) — the whole control plane is dark unless an operator opts
//     the environment in. Defense in depth: a misconfigured token can do nothing
//     in an environment that never enabled the plane.
//   - control token (Gate.Token) — the per-provider secret search-hub minted at
//     first registration. Every RPC carries it (control_token field); a mismatch
//     is rejected with a constant-time compare. There is no token-free control
//     verb; public search (routing.Query without overrides) is the only unauthed
//     path and lives in the search handler, not here.
package searchcontrol

import (
	"context"
	"crypto/subtle"
	"errors"
	"log"

	"connectrpc.com/connect"

	"cli-health/internal/aisearch"

	aisearchpkg "github.com/vrooli/aisearch-go"
	searchregister "github.com/vrooli/searchregister-go"
	controlv1 "github.com/vrooli/vrooli/packages/proto/gen/go/search-hub/v1/control"
)

// Reindexer is the seam between the control handler and the aisearch reconcile
// job control. It is satisfied by ServiceAdapter (wrapping the live aisearch
// service) in production and by a fake in tests.
type Reindexer interface {
	Reindex(ctx context.Context, scope string, dryRun bool) (jobID string, plannedUpserts, plannedDeletes int, err error)
	ReindexStatus(jobID string) (state string, processed, total int, errMsg string, ok bool)
	ReindexCancel(jobID string) bool
	// ApplyTuning rebuilds the live engine for the new tuning and re-embeds the
	// corpus with the new recipe IN PROCESS — the index-time apply that takes
	// effect without a restart. It returns the reindex job id + planned drift so
	// the caller can poll ReindexStatus to terminal. A structural change the
	// collection schema guard rejects (dense↔hybrid) returns an error and leaves
	// the live engine untouched (no auto-drop).
	ApplyTuning(ctx context.Context, tuning aisearchpkg.TuningConfig) (jobID string, plannedUpserts, plannedDeletes int, err error)
}

// ConfigWriter is the seam that persists a new tuning block into the provider's
// search.json SSOT. FileConfigWriter is the production implementation (atomic
// write via aisearch-go); tests substitute a fake.
type ConfigWriter interface {
	// WriteTuning validates and persists tuning for providerID, returning the
	// tuning now in effect, whether an INDEX-TIME factor changed (the caller
	// reindexes when true), and whether the file was actually rewritten (false on
	// dryRun or a no-op).
	WriteTuning(providerID string, tuning aisearchpkg.TuningConfig, dryRun bool) (effective aisearchpkg.TuningConfig, indexTimeChanged, written bool, err error)
}

// Gate is the control-token guard every control RPC passes through. It mirrors
// the search handler's OverrideGate but rejects (rather than silently degrading)
// because a control verb has no public fallback. search-hub minted the token at
// registration and is its only holder, so the token alone gates the mutating
// plane; a provider that does not want to be tuned at all omits its control
// endpoints in search.json (no env flag).
type Gate struct {
	// Token returns the currently-cached control token ("" until search-hub
	// registration echoes one back). A "" want always denies.
	Token func() string
}

// authorize returns nil when the request may proceed, else a connect error. It
// logs the denial reason server-side but returns a uniform PermissionDenied so
// the wire response never reveals why (unregistered vs wrong token).
func (g *Gate) authorize(logger *log.Logger, presented string) error {
	if g == nil || !g.tokenMatches(presented) {
		logger.Printf("[cli-health] search control denied: control token missing or mismatched")
		return connect.NewError(connect.CodePermissionDenied, errors.New("search control plane is not available"))
	}
	return nil
}

func (g *Gate) tokenMatches(presented string) bool {
	want := ""
	if g.Token != nil {
		want = g.Token()
	}
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
	Gate         *Gate
}

type connectHandler struct {
	deps Deps
}

// NewConnectHandler builds the Connect handler.
func NewConnectHandler(d Deps) *connectHandler {
	if d.Logger == nil {
		d.Logger = log.Default()
	}
	return &connectHandler{deps: d}
}

func (h *connectHandler) Reindex(ctx context.Context, req *connect.Request[controlv1.ReindexRequest]) (*connect.Response[controlv1.ReindexResponse], error) {
	r := req.Msg
	if err := h.deps.Gate.authorize(h.deps.Logger, r.GetControlToken()); err != nil {
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
	if err := h.deps.Gate.authorize(h.deps.Logger, r.GetControlToken()); err != nil {
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
	if err := h.deps.Gate.authorize(h.deps.Logger, r.GetControlToken()); err != nil {
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
	if err := h.deps.Gate.authorize(h.deps.Logger, r.GetControlToken()); err != nil {
		return nil, err
	}
	if h.deps.ConfigWriter == nil {
		return nil, connect.NewError(connect.CodeUnimplemented, errors.New("config-write backend not configured"))
	}
	providerID := r.GetProviderId()
	if providerID == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("provider_id is required"))
	}

	// Validate the incoming tuning up front so a malformed factor is a clean
	// InvalidArgument, leaving WriteTuning's remaining failures to be NotFound
	// (unknown provider) or Internal (IO).
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

	// An index-time factor change requires re-embedding to take full effect.
	// ApplyTuning rebuilds the live engine with the new recipe and re-embeds the
	// corpus IN PROCESS (no restart) — the recipe-aware drift hash marks every
	// point drifted — returning a job the caller polls. A structural change the
	// schema guard rejects (dense↔hybrid) surfaces here without mutating the live
	// engine or dropping data; that arm needs a manual collection rebuild.
	if written && indexTimeChanged && h.deps.Reindexer != nil {
		jobID, _, _, rErr := h.deps.Reindexer.ApplyTuning(ctx, effective)
		if rErr != nil {
			return nil, connect.NewError(connect.CodeInternal, rErr)
		}
		resp.ReindexTriggered = true
		resp.ReindexJobId = jobID
	}
	return connect.NewResponse(resp), nil
}

// ServiceAdapter wraps *aisearch.Service to satisfy the Reindexer seam. (Moved
// verbatim from the deleted private reindex handler; the reconcile job control
// itself is unchanged — only the wire contract in front of it is now shared.)
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

// ApplyTuning delegates to the live service's in-process engine rebuild + re-embed.
func (a ServiceAdapter) ApplyTuning(ctx context.Context, tuning aisearchpkg.TuningConfig) (string, int, int, error) {
	return a.Service.ApplyTuning(ctx, tuning)
}

// FileConfigWriter persists tuning into a scenario's search.json via aisearch-go's
// atomic writer. Path is the absolute path to the scenario-owned search.json.
type FileConfigWriter struct{ Path string }

func (w FileConfigWriter) WriteTuning(providerID string, tuning aisearchpkg.TuningConfig, dryRun bool) (aisearchpkg.TuningConfig, bool, bool, error) {
	return aisearchpkg.WriteProviderTuning(w.Path, providerID, tuning, dryRun)
}
