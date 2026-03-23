# PRD Context Brief: Vrooli Onboarding

## Overview & Value Proposition

Vrooli Onboarding is the central configuration hub for the Vrooli platform. It transforms the currently manual, expert-requiring setup process into a guided experience accessible to non-technical users. Users interact through either a CLI or web UI (both with identical functionality), with all logic residing in the API.

**Target Users:**
- Non-technical users setting up Vrooli for the first time
- Developers new to the Vrooli ecosystem
- Returning users managing their resource configuration

**Value:** Reduces time-to-first-working-agent from hours of manual config reading to under 10 minutes of guided setup. Becomes a permanent hub users return to for all configuration needs.

## P0 — Core Capabilities (Operational Targets)

1. **Guided Coding Agent Setup**: Step-by-step flows for enabling and configuring resource-claude-code, resource-codex, and resource-opencode. Includes secret entry, validation, and health verification.

2. **Guided AI Provider Setup**: Step-by-step flows for resource-openrouter (API key with format validation and ping test) and resource-ollama (local service health check).

3. **Secret Validation Pipeline**: Before persisting any API key to secrets.json: trim whitespace, check format patterns (e.g. sk-or- prefix for OpenRouter keys), perform lightweight API ping where possible. Show clear inline pass/fail feedback.

4. **Personalized service.json Generation**: Based on user choices, generate a complete service.json with transitive dependencies auto-resolved and sensible defaults applied. Validate against schema before writing.

5. **Configuration Dashboard**: Hub view showing current resource status (enabled/disabled, healthy/unhealthy), allowing users to return and modify their configuration at any time.

6. **Progress Persistence**: Store onboarding progress so users can abandon mid-flow and resume later without losing work. API-managed state.

## P1 — Important Enhancements

7. **Real-Time Health Visualization**: After resource enablement, show live health status using existing infrastructure (running-resources.json + HTTP health endpoints). Spinner → green checkmark flow. Reference tunnel-manager TunnelStatus pattern.

8. **GuidedTour Step Framework**: Port the proven GuidedTour pattern from browser-automation-studio (step tracking, skip/complete actions, progress persistence via useGuidedTour hook) for UI step flows.

9. **Deep-Links to Related Scenarios**: Contextual links to vrooli-autoheal UI and prompt-manager UI with brief plain-language explanations of what each does and when to use them.

## P2 — Nice-to-Have Polish

10. **Ordered Flow Guidance**: Suggest optimal setup order (coding agents before AI providers) while allowing users to jump to any section.

11. **Plain-Language Resource Descriptions**: Replace technical terms ("resources") with user-friendly labels ("coding tools", "AI services") throughout all flows.

## Tech Direction Snapshot

- **Architecture**: Standard Vrooli scenario pattern — api/ (Go), cli/ (Go), ui/ (React/TypeScript)
- **API-first**: All business logic in the API. CLI and UI are thin presentation layers calling the same endpoints
- **Config management**: Read/write .vrooli/service.json and .vrooli/secrets.json
- **Health monitoring**: Poll .vrooli/running-resources.json and resource HTTP health endpoints
- **Step framework**: Server-side step definitions with progress state; UI renders via GuidedTour-style components
- **Config generation**: Endpoint accepts choices → resolves transitive deps → validates → writes service.json

## Dependencies & Launch Plan

**Dependencies:**
- Existing health check infrastructure (running-resources.json, HTTP health endpoints)
- Resource-specific API key format knowledge (OpenRouter sk-or- prefix, etc.)
- BAS GuidedTour component as implementation reference
- vrooli CLI for resource lifecycle management

**Launch sequence:**
1. API with config generation + secret validation + health polling
2. CLI frontend with interactive guided flows
3. UI frontend with GuidedTour-based step flows
4. Integration testing with all v1 resources
