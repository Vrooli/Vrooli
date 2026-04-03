package scenarios

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/gorilla/mux"

	apipb "github.com/vrooli/vrooli/packages/proto/gen/go/swarm-manager/v1/api"
	"swarm-manager/internal/apierr"
	"swarm-manager/internal/httputil"
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
	startTimeout   time.Duration
	stopTimeout    time.Duration
	restartTimeout time.Duration
}

// NewCLILifecycle creates a CLI-backed lifecycle controller.
func NewCLILifecycle() *CLILifecycle {
	return &CLILifecycle{
		startTimeout:   defaultStartTimeout,
		stopTimeout:    defaultStopTimeout,
		restartTimeout: defaultRestartTimeout,
	}
}

// Start starts a scenario using the Vrooli CLI.
func (c *CLILifecycle) Start(ctx context.Context, name string) error {
	trimmed := strings.TrimSpace(name)
	if trimmed == "" {
		return errScenarioNameRequired
	}
	_, err := executeVrooliCommand(ctx, c.startTimeout, "scenario", "start", trimmed)
	return err
}

// Stop stops a scenario using the Vrooli CLI.
func (c *CLILifecycle) Stop(ctx context.Context, name string) error {
	trimmed := strings.TrimSpace(name)
	if trimmed == "" {
		return errScenarioNameRequired
	}
	_, err := executeVrooliCommand(ctx, c.stopTimeout, "scenario", "stop", trimmed)
	return err
}

// Restart restarts a scenario using the Vrooli CLI.
func (c *CLILifecycle) Restart(ctx context.Context, name string) error {
	trimmed := strings.TrimSpace(name)
	if trimmed == "" {
		return errScenarioNameRequired
	}
	_, err := executeVrooliCommand(ctx, c.restartTimeout, "scenario", "restart", trimmed)
	return err
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

	scenario, err := h.loadScenario(r.Context(), name)
	if err != nil {
		apierr.MapError(w, "[scenarios] "+action, apierr.Internal("failed to load scenario"))
		return
	}

	resp := &apipb.ScenarioResponse{Scenario: scenarioToProto(scenario)}
	if err := httputil.ProtoJSON(w, resp); err != nil {
		apierr.MapError(w, "[scenarios] "+action, apierr.Internal("failed to encode response"))
	}

	// Dispatch graph event for scenario status change.
	if h.eventDispatcher != nil {
		h.eventDispatcher.DispatchNodeUpdate("Scenario", "scenario/"+name, map[string]any{
			"name":   scenario.Name,
			"status": string(scenario.Status),
		})
		h.eventDispatcher.DispatchInvalidate("topology", "operations")
	}
}
