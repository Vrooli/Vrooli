package provider_lifecycle

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	"connectrpc.com/connect"

	"audio-tools/internal/capabilities"
	"audio-tools/internal/logx"

	plv1 "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/provider_lifecycle"
)

// dryRunHeader is the canonical request header signalling a dry-run
// invocation; mirrors github.com/vrooli/cli-core/cliutil.DryRunHeader
// (kept inline so the API doesn't pull cli-core as a dep).
const dryRunHeader = "X-Dry-Run"

// localProviderOrder pins ListLocalProviders' output order so the UI
// and CLI render deterministically regardless of the registry's
// internal map iteration.
var localProviderOrder = []string{
	"whisper-stt",
	"kokoro-tts",
	"speaker-verification",
	"ollama",
}

// localProviderActions defines which Action enum values each local
// provider advertises. Ollama is the only one that supports
// PULL_MODEL; everyone else gets START/STOP/RESTART/VIEW_LOGS.
var localProviderActions = map[string][]plv1.Action{
	"whisper-stt":          {plv1.Action_ACTION_START, plv1.Action_ACTION_STOP, plv1.Action_ACTION_RESTART, plv1.Action_ACTION_VIEW_LOGS},
	"kokoro-tts":           {plv1.Action_ACTION_START, plv1.Action_ACTION_STOP, plv1.Action_ACTION_RESTART, plv1.Action_ACTION_VIEW_LOGS},
	"speaker-verification": {plv1.Action_ACTION_START, plv1.Action_ACTION_STOP, plv1.Action_ACTION_RESTART, plv1.Action_ACTION_VIEW_LOGS},
	"ollama":               {plv1.Action_ACTION_START, plv1.Action_ACTION_STOP, plv1.Action_ACTION_RESTART, plv1.Action_ACTION_PULL_MODEL, plv1.Action_ACTION_VIEW_LOGS},
}

type connectHandler struct {
	deps Deps
}

// NewConnectHandler builds the Connect handler. All Deps fields are
// required; no fallbacks.
func NewConnectHandler(d Deps) *connectHandler {
	return &connectHandler{deps: d}
}

// ListLocalProviders enumerates the four local-tier providers
// audio-tools owns with their current process_state (derived from the
// in-process registry).
func (h *connectHandler) ListLocalProviders(ctx context.Context, _ *connect.Request[plv1.ListLocalProvidersRequest]) (*connect.Response[plv1.ListLocalProvidersResponse], error) {
	if h.deps.Registry == nil {
		return nil, connect.NewError(connect.CodeUnavailable, errors.New("capabilities registry not configured"))
	}
	states := h.deps.Registry.ResolveLiveness(ctx)
	stateByID := make(map[string]capabilities.Status, len(states))
	nameByID := make(map[string]string, len(states))
	for _, s := range states {
		stateByID[s.Def.ID] = s.Status
		nameByID[s.Def.ID] = s.Def.Name
	}

	out := make([]*plv1.LocalProvider, 0, len(localProviderOrder))
	for _, id := range localProviderOrder {
		slug, ok := capabilities.ResourceSlugForProviderID(id)
		if !ok {
			continue
		}
		out = append(out, &plv1.LocalProvider{
			ProviderId:       id,
			DisplayName:      defaultedName(nameByID[id], id),
			ResourceSlug:     slug,
			ProcessState:     processStateFor(stateByID[id]),
			SupportedActions: localProviderActions[id],
		})
	}
	return connect.NewResponse(&plv1.ListLocalProvidersResponse{Providers: out}), nil
}

// StartProvider wraps ResourceController.Start.
func (h *connectHandler) StartProvider(ctx context.Context, req *connect.Request[plv1.StartProviderRequest]) (*connect.Response[plv1.StartProviderResponse], error) {
	id := req.Msg.GetProviderId()
	result, err := h.runProviderAction(ctx, id, req.Header(), "StartProvider", "started", h.deps.Controller.Start)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(&plv1.StartProviderResponse{ProviderId: id, DryRun: result.dryRun, Message: result.message(id)}), nil
}

// StopProvider wraps ResourceController.Stop.
func (h *connectHandler) StopProvider(ctx context.Context, req *connect.Request[plv1.StopProviderRequest]) (*connect.Response[plv1.StopProviderResponse], error) {
	id := req.Msg.GetProviderId()
	result, err := h.runProviderAction(ctx, id, req.Header(), "StopProvider", "stopped", h.deps.Controller.Stop)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(&plv1.StopProviderResponse{ProviderId: id, DryRun: result.dryRun, Message: result.message(id)}), nil
}

// RestartProvider wraps ResourceController.Restart.
func (h *connectHandler) RestartProvider(ctx context.Context, req *connect.Request[plv1.RestartProviderRequest]) (*connect.Response[plv1.RestartProviderResponse], error) {
	id := req.Msg.GetProviderId()
	result, err := h.runProviderAction(ctx, id, req.Header(), "RestartProvider", "restarted", h.deps.Controller.Restart)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(&plv1.RestartProviderResponse{ProviderId: id, DryRun: result.dryRun, Message: result.message(id)}), nil
}

type providerActionResult struct {
	dryRun bool
	verb   string
}

func (r providerActionResult) message(id string) string {
	if r.dryRun {
		return "dry run; no action taken"
	}
	return fmt.Sprintf("%s %s", id, r.verb)
}

func (h *connectHandler) runProviderAction(ctx context.Context, id string, headers http.Header, operation, verb string, action func(context.Context, string) error) (providerActionResult, error) {
	slug, err := h.validateLocalProvider(id)
	if err != nil {
		return providerActionResult{}, err
	}
	logIdempotency(h.deps.Logger, operation, id, headers)
	if isDryRun(headers) {
		return providerActionResult{dryRun: true, verb: verb}, nil
	}
	if err := action(ctx, slug); err != nil {
		return providerActionResult{}, h.mapControllerErr(ctx, operation, err)
	}
	if h.deps.InvalidateEngineCache != nil {
		h.deps.InvalidateEngineCache()
	}
	h.scheduleResolveForce()
	return providerActionResult{verb: verb}, nil
}

// PullModel is ollama-only.
func (h *connectHandler) PullModel(ctx context.Context, req *connect.Request[plv1.PullModelRequest]) (*connect.Response[plv1.PullModelResponse], error) {
	id := req.Msg.GetProviderId()
	model := req.Msg.GetModelName()
	if _, ok := capabilities.ResourceSlugForProviderID(id); !ok {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("unknown provider_id %q", id))
	}
	if !capabilities.SupportsPullModel(id) {
		return nil, connect.NewError(connect.CodeFailedPrecondition, fmt.Errorf("provider %q does not support pull_model", id))
	}
	if model == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("model_name is required"))
	}
	if h.deps.Controller == nil {
		return nil, connect.NewError(connect.CodeUnavailable, errors.New("resource controller not configured"))
	}
	logIdempotency(h.deps.Logger, "PullModel", id+"/"+model, req.Header())
	if isDryRun(req.Header()) {
		return connect.NewResponse(&plv1.PullModelResponse{ProviderId: id, ModelName: model, DryRun: true, Message: "dry run; no action taken"}), nil
	}
	if err := h.deps.Controller.PullModel(ctx, model); err != nil {
		return nil, h.mapControllerErr(ctx, "PullModel", err)
	}
	return connect.NewResponse(&plv1.PullModelResponse{ProviderId: id, ModelName: model, Message: fmt.Sprintf("model %q pulled on %s", model, id)}), nil
}

// GetProviderLogs streams log lines until the controller closes the
// reader, the context is cancelled, or stream.Send fails.
func (h *connectHandler) GetProviderLogs(ctx context.Context, req *connect.Request[plv1.GetProviderLogsRequest], stream *connect.ServerStream[plv1.LogLine]) error {
	id := req.Msg.GetProviderId()
	slug, err := h.validateLocalProvider(id)
	if err != nil {
		return err
	}
	reader, logErr := h.deps.Controller.Logs(ctx, slug, req.Msg.GetFollow(), int(req.Msg.GetTailLines()))
	if logErr != nil {
		return h.mapControllerErr(ctx, "GetProviderLogs", logErr)
	}
	defer reader.Close()

	scanner := bufio.NewScanner(reader)
	// Allow long lines (tracebacks, JSON dumps) up to 1 MiB.
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for scanner.Scan() {
		if err := ctx.Err(); err != nil {
			return nil
		}
		line := scanner.Text()
		if err := stream.Send(&plv1.LogLine{
			Line:     line,
			TsUnixMs: h.deps.Clock.Now().UnixMilli(),
			Stream:   plv1.LogStream_LOG_STREAM_STDOUT,
		}); err != nil {
			return err
		}
	}
	if err := scanner.Err(); err != nil && !errors.Is(err, io.EOF) {
		return connect.NewError(connect.CodeInternal, fmt.Errorf("read logs: %w", err))
	}
	return nil
}

// validateLocalProvider checks the provider_id is a local-tier
// provider we own, and asserts the controller is configured. Returns
// the resource slug on success.
func (h *connectHandler) validateLocalProvider(id string) (string, *connect.Error) {
	if id == "" {
		return "", connect.NewError(connect.CodeInvalidArgument, errors.New("provider_id is required"))
	}
	slug, ok := capabilities.ResourceSlugForProviderID(id)
	if !ok {
		// Distinguish "known but not local" from "unknown" so callers
		// get an actionable code.
		if id == "openrouter" || id == "audio-tools" {
			return "", connect.NewError(connect.CodeFailedPrecondition, fmt.Errorf("provider %q is not a local-tier provider; lifecycle actions only apply to local resources", id))
		}
		return "", connect.NewError(connect.CodeNotFound, fmt.Errorf("unknown provider_id %q", id))
	}
	if h.deps.Registry != nil {
		for _, state := range h.deps.Registry.ResolveLiveness(context.Background()) {
			if state.Def.ID != id || state.Def.Platform.Support != capabilities.PlatformUnsupported {
				continue
			}
			reason := state.Def.Platform.Reason
			if reason == "" {
				reason = "the provider is unsupported on this platform"
			}
			return "", connect.NewError(connect.CodeFailedPrecondition, fmt.Errorf("provider %q cannot be started: %s", id, reason))
		}
	}
	if h.deps.Controller == nil {
		return "", connect.NewError(connect.CodeUnavailable, errors.New("resource controller not configured"))
	}
	return slug, nil
}

// mapControllerErr converts controller-layer errors to typed Connect
// errors. ErrControllerUnavailable → Unavailable; context cancel →
// Canceled; everything else → Internal.
func (h *connectHandler) mapControllerErr(ctx context.Context, op string, err error) *connect.Error {
	if errors.Is(err, capabilities.ErrControllerUnavailable) {
		return connect.NewError(connect.CodeUnavailable, err)
	}
	if errors.Is(ctx.Err(), context.Canceled) || errors.Is(err, context.Canceled) {
		return connect.NewError(connect.CodeCanceled, err)
	}
	h.deps.Logger.Printf("provider_lifecycle %s failed: %v", op, err)
	return connect.NewError(connect.CodeInternal, err)
}

// scheduleResolveForce kicks a background cache-bust so the next
// GetProviderHealth call reflects the new process state without
// waiting for the registry TTL boundary. Uses a fresh context so the
// goroutine survives after this handler returns.
func (h *connectHandler) scheduleResolveForce() {
	if h.deps.Registry == nil {
		return
	}
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		h.deps.Registry.ResolveForce(ctx)
	}()
}

func defaultedName(name, fallback string) string {
	if name == "" {
		return fallback
	}
	return name
}

func processStateFor(s capabilities.Status) plv1.ProcessState {
	switch s {
	case capabilities.StatusAvailable:
		return plv1.ProcessState_PROCESS_STATE_RUNNING
	case capabilities.StatusUnavailable:
		return plv1.ProcessState_PROCESS_STATE_STOPPED
	}
	return plv1.ProcessState_PROCESS_STATE_UNKNOWN
}

// isDryRun reads the canonical X-Dry-Run header. Reuses cli-core's
// header constant so request- and server-side stay in lockstep.
func isDryRun(h interface{ Get(string) string }) bool {
	return h.Get(dryRunHeader) == "true"
}

// logIdempotency emits a best-effort log line carrying the
// Idempotency-Key header. Phase 2 does NOT dedup; this is observability
// only. TODO(phase3): wire a real dedup store.
func logIdempotency(logger logx.Logger, op, target string, h interface{ Get(string) string }) {
	if key := h.Get("Idempotency-Key"); key != "" {
		logger.Printf("provider_lifecycle %s idempotency_key=%q target=%q (best-effort; no dedup yet)", op, key, target)
	}
}
