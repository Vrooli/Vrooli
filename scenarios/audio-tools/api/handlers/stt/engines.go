package stt

import (
	"context"
	"os"
	"path/filepath"
	"strings"

	"connectrpc.com/connect"

	"audio-tools/internal/sttengine"

	sttv1 "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/stt"
)

// ListEngines projects the engine-capability manifest into the wire shape the
// admin picker + CLI consume, resolving each engine's runtime availability and
// marking the active selection. The engine catalog is manifest-derived — this
// handler never hardcodes an engine list.
func (h *connectHandler) ListEngines(ctx context.Context, _ *connect.Request[sttv1.ListEnginesRequest]) (*connect.Response[sttv1.ListEnginesResponse], error) {
	resp := &sttv1.ListEnginesResponse{}
	if h.deps.Registry == nil {
		return connect.NewResponse(resp), nil
	}
	active := h.resolveStreamPipelineConfig(ctx).EngineID
	if active == "" {
		active = h.deps.Registry.DefaultEngineID()
	}
	for _, e := range h.deps.Registry.Engines() {
		resp.Engines = append(resp.Engines, &sttv1.EngineInfo{
			Id:              e.ID,
			DisplayName:     e.DisplayName,
			Kind:            e.Kind,
			Available:       h.engineAvailable(ctx, e),
			NativeStreaming: e.Provides.NativeStreaming,
			IsActive:        e.ID == active,
		})
	}
	return connect.NewResponse(resp), nil
}

// engineAvailable probes whether an engine can serve right now. Availability is
// inherently per-resource (each backing service has its own health surface),
// but the mapping from engine id to liveness probe lives in the chain (which
// owns the Local-tier providers) — so this stays manifest-driven with no
// per-resource branching here. Non-local engines (BYOK/Vrooli) are reported
// available; their per-request credential gating happens at stream time.
func (h *connectHandler) engineAvailable(ctx context.Context, e sttengine.Engine) bool {
	if e.Kind != sttengine.KindLocalResource {
		return true
	}
	return h.deps.Chain != nil && h.deps.Chain.LocalEngineAvailable(ctx, e.ID)
}

// GetEngineSwitchImpact reports the shared-resource impact of switching away
// from from_engine_id: which OTHER scenarios still declare that engine's
// backing resource, whether it is therefore safe to stop, and the exact
// operator command to stop it. audio-tools NEVER stops a shared resource
// itself — this is purely informational so the picker can prompt the operator.
func (h *connectHandler) GetEngineSwitchImpact(_ context.Context, req *connect.Request[sttv1.GetEngineSwitchImpactRequest]) (*connect.Response[sttv1.GetEngineSwitchImpactResponse], error) {
	resp := &sttv1.GetEngineSwitchImpactResponse{}
	if h.deps.Registry == nil {
		return connect.NewResponse(resp), nil
	}
	engine, ok := h.deps.Registry.Engine(req.Msg.GetFromEngineId())
	if !ok || engine.Kind != sttengine.KindLocalResource || engine.Resource == "" {
		// Non-local engines own no local resource; nothing to stop.
		resp.SafeToStop = true
		resp.ConsumersKnown = true
		return connect.NewResponse(resp), nil
	}
	resp.Resource = engine.Resource
	resp.StopCommand = "vrooli resource stop " + engine.Resource

	fsys, _, located := sttengine.ScenariosFS()
	if !located {
		// Can't enumerate consumers — conservatively NOT safe to stop.
		resp.ConsumersKnown = false
		resp.SafeToStop = false
		return connect.NewResponse(resp), nil
	}
	consumers, err := sttengine.ScanResourceConsumers(fsys, engine.Resource, currentScenarioID())
	if err != nil {
		resp.ConsumersKnown = false
		resp.SafeToStop = false
		return connect.NewResponse(resp), nil
	}
	resp.ConsumersKnown = true
	for _, c := range consumers {
		resp.Consumers = append(resp.Consumers, &sttv1.ScenarioResourceConsumer{
			Scenario:    c.Scenario,
			DisplayName: c.DisplayName,
			Required:    c.Required,
		})
	}
	resp.SafeToStop = len(consumers) == 0
	return connect.NewResponse(resp), nil
}

// currentScenarioID is the scenario directory name to exclude from the
// consumer scan (audio-tools should not count itself). Derived from
// VROOLI_SCENARIO_DIR's basename, defaulting to "audio-tools".
func currentScenarioID() string {
	if raw, configured := os.LookupEnv("VROOLI_SCENARIO_DIR"); configured {
		if dir := strings.TrimSpace(raw); dir != "" {
			if base := filepath.Base(filepath.Clean(dir)); base != "" && base != "." && base != string(filepath.Separator) {
				return base
			}
		}
	}
	return "audio-tools"
}
