package scenarios

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"time"

	"swarm-manager/internal/apierr"
	"swarm-manager/internal/httputil"

	"github.com/gorilla/mux"

	vroolicli "github.com/vrooli/vrooli-cli-go"
	apipb "github.com/vrooli/vrooli/packages/proto/gen/go/swarm-manager/v1/api"
)

// DOC: docs/concepts/ARCHITECTURE.md#key-flows
// DOC: docs/internal/SEAMS.md

const (
	defaultStartTimeout   = 60 * time.Second
	defaultStopTimeout    = 20 * time.Second
	defaultRestartTimeout = 90 * time.Second
)

// Lifecycle controls scenario start/stop/restart operations.
type Lifecycle interface {
	Start(ctx context.Context, name string) error
	Stop(ctx context.Context, name string) error
	Restart(ctx context.Context, name string) error
}

// CLILifecycle executes lifecycle actions via the Vrooli CLI.
type CLILifecycle struct {
	client         *vroolicli.Client
	startTimeout   time.Duration
	stopTimeout    time.Duration
	restartTimeout time.Duration
}

// NewCLILifecycle creates a CLI-backed lifecycle controller.
func NewCLILifecycle() *CLILifecycle {
	return &CLILifecycle{
		client:         vroolicli.New(),
		startTimeout:   defaultStartTimeout,
		stopTimeout:    defaultStopTimeout,
		restartTimeout: defaultRestartTimeout,
	}
}

// run executes a lifecycle verb under a per-action timeout. The bounded context
// is honored by the client (which only applies its own default when the caller
// supplies no deadline), so each verb keeps its distinct budget. OutputCombined
// surfaces any stderr text the CLI prints as part of the error.
func (c *CLILifecycle) run(ctx context.Context, timeout time.Duration, name, verb string) error {
	trimmed := strings.TrimSpace(name)
	if trimmed == "" {
		return errScenarioNameRequired
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	_, err := c.client.OutputCombined(ctx, "scenario", verb, trimmed)
	return err
}

// Start starts a scenario using the Vrooli CLI.
func (c *CLILifecycle) Start(ctx context.Context, name string) error {
	return c.run(ctx, c.startTimeout, name, "start")
}

// Stop stops a scenario using the Vrooli CLI.
func (c *CLILifecycle) Stop(ctx context.Context, name string) error {
	return c.run(ctx, c.stopTimeout, name, "stop")
}

// Restart restarts a scenario using the Vrooli CLI.
func (c *CLILifecycle) Restart(ctx context.Context, name string) error {
	return c.run(ctx, c.restartTimeout, name, "restart")
}

// Start starts a scenario via the Vrooli CLI.
func (h *Handler) Start(w http.ResponseWriter, r *http.Request) {
	h.handleLifecycleAction(w, r, "start")
}

// Stop stops a scenario via the Vrooli CLI.
func (h *Handler) Stop(w http.ResponseWriter, r *http.Request) {
	h.handleLifecycleAction(w, r, "stop")
}

// Restart restarts a scenario via the Vrooli CLI.
func (h *Handler) Restart(w http.ResponseWriter, r *http.Request) {
	h.handleLifecycleAction(w, r, "restart")
}

func (h *Handler) handleLifecycleAction(w http.ResponseWriter, r *http.Request, action string) {
	vars := mux.Vars(r)
	name := strings.TrimSpace(vars["name"])
	if name == "" {
		apierr.MapError(w, "[scenarios] "+action, apierr.BadRequest("name is required"))
		return
	}
	if h.lifecycle == nil {
		apierr.MapError(w, "[scenarios] "+action, apierr.Internal("scenario lifecycle is unavailable"))
		return
	}

	_, found, err := h.findScenarioSource(r.Context(), name)
	if err != nil {
		apierr.MapError(w, "[scenarios] "+action, apierr.Internal("failed to load scenarios from CLI"))
		return
	}
	if !found {
		apierr.MapError(w, "", apierr.NotFound("scenario not found"))
		return
	}

	var actionErr error
	switch action {
	case "start":
		actionErr = h.lifecycle.Start(r.Context(), name)
	case "stop":
		actionErr = h.lifecycle.Stop(r.Context(), name)
	case "restart":
		actionErr = h.lifecycle.Restart(r.Context(), name)
	default:
		apierr.MapError(w, "[scenarios] "+action, apierr.BadRequest("unsupported action"))
		return
	}
	if actionErr != nil {
		if errors.Is(actionErr, errScenarioNameRequired) {
			apierr.MapError(w, "[scenarios] "+action, apierr.BadRequest("name is required"))
			return
		}
		apierr.MapError(w, "[scenarios] "+action, apierr.Internal("failed to %s scenario", action))
		return
	}
	h.invalidateCatalog()

	scenario, err := h.loadScenario(r.Context(), name)
	if err != nil {
		apierr.MapError(w, "[scenarios] "+action, apierr.Internal("failed to load scenario"))
		return
	}

	resp := &apipb.ScenarioResponse{Scenario: scenarioToProto(scenario, nil)}
	if err := httputil.ProtoJSON(w, resp); err != nil {
		apierr.MapError(w, "[scenarios] "+action, apierr.Internal("failed to encode response"))
	}

	// Dispatch graph event for scenario status change.
	if h.eventDispatcher != nil {
		h.eventDispatcher.DispatchNodeUpdate("Scenario", "scenario/"+name, map[string]any{
			"name":   scenario.Name,
			"status": string(scenario.Status),
		})
		h.eventDispatcher.DispatchInvalidate("topology")
	}
}
