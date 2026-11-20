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
	"google.golang.org/protobuf/proto"

	"audio-tools/internal/capabilities"
	"audio-tools/internal/logx"

	plv1 "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/provider_lifecycle"
)

// dryRunHeader is the canonical request header signalling a dry-run
// invocation; mirrors github.com/vrooli/cli-core/cliutil.DryRunHeader
// (kept inline so the API doesn't pull cli-core as a dep).
const dryRunHeader = "X-Dry-Run"

// actionsForProvider translates the generated provider-catalogue action
// vocabulary into the provider_lifecycle proto enum. The set of actions is
// owned by providercatalog; this function only translates enum values.
func actionsForProvider(id string) []plv1.Action {
	names := capabilities.ActionsForProvider(id)
	out := make([]plv1.Action, 0, len(names))
	for _, name := range names {
		if action, ok := lifecycleAction(name); ok {
			out = append(out, action)
		}
	}
	return out
}

func lifecycleAction(name string) (plv1.Action, bool) {
	switch name {
	case "start":
		return plv1.Action_ACTION_START, true
	case "stop":
		return plv1.Action_ACTION_STOP, true
	case "restart":
		return plv1.Action_ACTION_RESTART, true
	case "view_logs":
		return plv1.Action_ACTION_VIEW_LOGS, true
	case "pull_model":
		return plv1.Action_ACTION_PULL_MODEL, true
	default:
		return plv1.Action_ACTION_UNSPECIFIED, false
	}
}

type connectHandler struct {
	deps        Deps
	idempotency *IdempotencyStore
}

// NewConnectHandler builds the Connect handler. All Deps fields are
// required; no fallbacks. The idempotency store is the one exception:
// production gets the default in-memory store when Deps.Idempotency is
// nil, so wiring requires zero extra configuration.
func NewConnectHandler(d Deps) *connectHandler {
	store := d.Idempotency
	if store == nil {
		store = NewIdempotencyStore(IdempotencyOptions{})
	}
	return &connectHandler{deps: d, idempotency: store}
}

// ListLocalProviders enumerates the active local-tier providers that own a
// lifecycle row, with their current process_state (derived from the in-process
// registry). The order and membership come from the generated provider
// catalogue, deduplicated by resource slug.
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

	order := capabilities.LocalProviderIDs()
	out := make([]*plv1.LocalProvider, 0, len(order))
	for _, id := range order {
		slug, ok := capabilities.ResourceSlugForProviderID(id)
		if !ok {
			continue
		}
		out = append(out, &plv1.LocalProvider{
			ProviderId:       id,
			DisplayName:      defaultedName(nameByID[id], id),
			ResourceSlug:     slug,
			ProcessState:     processStateFor(stateByID[id]),
			SupportedActions: actionsForProvider(id),
		})
	}
	return connect.NewResponse(&plv1.ListLocalProvidersResponse{Providers: out}), nil
}

// StartProvider wraps ResourceController.Start.
func (h *connectHandler) StartProvider(ctx context.Context, req *connect.Request[plv1.StartProviderRequest]) (*connect.Response[plv1.StartProviderResponse], error) {
	id := req.Msg.GetProviderId()
	msg, err := runProviderAction(h, ctx, id, req.Header(), "StartProvider", "started", h.deps.Controller.Start,
		func(result providerActionResult) *plv1.StartProviderResponse {
			return &plv1.StartProviderResponse{ProviderId: id, DryRun: result.dryRun, Message: result.message(id)}
		})
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(msg), nil
}

// StopProvider wraps ResourceController.Stop.
func (h *connectHandler) StopProvider(ctx context.Context, req *connect.Request[plv1.StopProviderRequest]) (*connect.Response[plv1.StopProviderResponse], error) {
	id := req.Msg.GetProviderId()
	msg, err := runProviderAction(h, ctx, id, req.Header(), "StopProvider", "stopped", h.deps.Controller.Stop,
		func(result providerActionResult) *plv1.StopProviderResponse {
			return &plv1.StopProviderResponse{ProviderId: id, DryRun: result.dryRun, Message: result.message(id)}
		})
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(msg), nil
}

// RestartProvider wraps ResourceController.Restart.
func (h *connectHandler) RestartProvider(ctx context.Context, req *connect.Request[plv1.RestartProviderRequest]) (*connect.Response[plv1.RestartProviderResponse], error) {
	id := req.Msg.GetProviderId()
	msg, err := runProviderAction(h, ctx, id, req.Header(), "RestartProvider", "restarted", h.deps.Controller.Restart,
		func(result providerActionResult) *plv1.RestartProviderResponse {
			return &plv1.RestartProviderResponse{ProviderId: id, DryRun: result.dryRun, Message: result.message(id)}
		})
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(msg), nil
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

// runProviderAction is the shared Start/Stop/Restart flow: validate the
// provider, honor dry-run, then execute the controller action at most once
// per (Idempotency-Key, operation, provider).
func runProviderAction[T proto.Message](
	h *connectHandler,
	ctx context.Context,
	id string,
	headers http.Header,
	operation, verb string,
	action func(context.Context, string) error,
	build func(providerActionResult) T,
) (T, error) {
	var zero T
	slug, err := h.validateLocalProvider(id)
	if err != nil {
		return zero, err
	}
	requestKey := headers.Get(idempotencyKeyHeader)

	// Dry runs never reach the controller and never enter the store, so a
	// dry-run response can never satisfy a real request (or vice versa).
	if isDryRun(headers) {
		logIdempotency(h.deps.Logger, operation, id, requestKey, outcomeBypass)
		return build(providerActionResult{dryRun: true, verb: verb}), nil
	}

	msg, outcome, dedupErr := executeIdempotent(h, ctx, requestKey, operation, id, func(ctx context.Context) (T, error) {
		result, actionErr := h.applyProviderAction(ctx, slug, operation, verb, action)
		if actionErr != nil {
			return zero, actionErr
		}
		return build(result), nil
	})
	logIdempotency(h.deps.Logger, operation, id, requestKey, outcome)
	if dedupErr != nil {
		return zero, h.mapDedupErr(ctx, operation, dedupErr)
	}
	return msg, nil
}

// mapDedupErr keeps a deduplicated action's error contract intact. A leader
// already returns a *connect.Error from mapControllerErr; a joiner can instead
// surface its own context.Canceled/DeadlineExceeded, which must still reach the
// caller as Canceled/DeadlineExceeded rather than Connect's generic Unknown.
func (h *connectHandler) mapDedupErr(ctx context.Context, operation string, err error) error {
	var connectErr *connect.Error
	if errors.As(err, &connectErr) {
		return err
	}
	return h.mapControllerErr(ctx, operation, err)
}

// applyProviderAction runs the controller shell-out and its post-success
// cache busting. Only the leader of an idempotent request reaches here.
func (h *connectHandler) applyProviderAction(ctx context.Context, slug, operation, verb string, action func(context.Context, string) error) (providerActionResult, error) {
	if err := action(ctx, slug); err != nil {
		return providerActionResult{}, h.mapControllerErr(ctx, operation, err)
	}
	if h.deps.InvalidateEngineCache != nil {
		h.deps.InvalidateEngineCache()
	}
	h.scheduleResolveForce()
	return providerActionResult{verb: verb}, nil
}

// executeIdempotent routes fn through the dedup store. An empty key or a
// nil store executes fn directly (pre-dedup behavior). Returned messages
// are always clones, never shared state.
func executeIdempotent[T proto.Message](
	h *connectHandler,
	ctx context.Context,
	requestKey, operation, target string,
	fn func(context.Context) (T, error),
) (T, idempotencyOutcome, error) {
	var zero T
	if requestKey == "" || h.idempotency == nil {
		msg, err := fn(ctx)
		return msg, outcomeBypass, err
	}

	msg, outcome, err := h.idempotency.Do(ctx,
		idempotencyKey{key: requestKey, operation: operation, target: target, dryRun: false},
		func(ctx context.Context) (proto.Message, error) {
			return fn(ctx)
		})
	if err != nil {
		return zero, outcome, err
	}
	typed, ok := msg.(T)
	if !ok {
		return zero, outcome, connect.NewError(connect.CodeInternal, fmt.Errorf("%s: idempotency store returned %T", operation, msg))
	}
	return typed, outcome, nil
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
	requestKey := req.Header().Get(idempotencyKeyHeader)
	target := id + "/" + model
	if isDryRun(req.Header()) {
		logIdempotency(h.deps.Logger, "PullModel", target, requestKey, outcomeBypass)
		return connect.NewResponse(&plv1.PullModelResponse{ProviderId: id, ModelName: model, DryRun: true, Message: "dry run; no action taken"}), nil
	}
	msg, outcome, err := executeIdempotent(h, ctx, requestKey, "PullModel", target, func(ctx context.Context) (*plv1.PullModelResponse, error) {
		if pullErr := h.deps.Controller.PullModel(ctx, model); pullErr != nil {
			return nil, h.mapControllerErr(ctx, "PullModel", pullErr)
		}
		return &plv1.PullModelResponse{ProviderId: id, ModelName: model, Message: fmt.Sprintf("model %q pulled on %s", model, id)}, nil
	})
	logIdempotency(h.deps.Logger, "PullModel", target, requestKey, outcome)
	if err != nil {
		return nil, h.mapDedupErr(ctx, "PullModel", err)
	}
	return connect.NewResponse(msg), nil
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
// Canceled; deadline → DeadlineExceeded; everything else → Internal.
func (h *connectHandler) mapControllerErr(ctx context.Context, op string, err error) *connect.Error {
	if errors.Is(err, capabilities.ErrControllerUnavailable) {
		return connect.NewError(connect.CodeUnavailable, err)
	}
	if errors.Is(ctx.Err(), context.Canceled) || errors.Is(err, context.Canceled) {
		return connect.NewError(connect.CodeCanceled, err)
	}
	if errors.Is(ctx.Err(), context.DeadlineExceeded) || errors.Is(err, context.DeadlineExceeded) {
		return connect.NewError(connect.CodeDeadlineExceeded, err)
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

// logIdempotency emits a best-effort log line describing how the dedup
// store handled the request. outcome is "miss" (leader executed), "join"
// (shared an in-flight execution), "hit" (replayed a cached result), or
// "bypass" (empty key or dry run: no dedup). Requests without an
// Idempotency-Key are not logged, preserving the pre-dedup noise level.
func logIdempotency(logger logx.Logger, op, target, key string, outcome idempotencyOutcome) {
	if key == "" {
		return
	}
	logger.Printf("provider_lifecycle %s idempotency_key=%q target=%q outcome=%s", op, key, target, outcome)
}
