# Enhanced Plan: Vrooli Onboarding

## Overview

Vrooli Onboarding is a permanent configuration hub (not just a first-run wizard) that guides users — including non-technical ones — through setting up and managing Vrooli resources. It follows the standard scenario pattern: API holds all logic, with both a CLI and web UI providing identical functionality. V1 focuses on coding agents (claude-code, codex, opencode) and AI providers (openrouter, ollama), with deep-links to other scenarios (autoheal, prompt-manager) for configuration outside v1 scope.

## Clarifications Applied

| Question | Answer | Impact |
|----------|--------|--------|
| Interface modality | API-first with both CLI and UI having the same functionality | Standard scenario pattern; no TUI needed. API holds logic, CLI and UI are equal frontends |
| MVP scope | Ongoing configuration hub, not just first-run wizard | Must support return visits, not just one-time setup. Needs persistent state and a dashboard view |
| Target user technical level | Mixed, assume non-technical | All UI/CLI text must avoid jargon. Visual feedback is critical. Error messages must be actionable |
| V1 resource scope | Curated: coding agents (claude-code, codex, opencode) then AI providers (openrouter, ollama) | Tight scope. Other resources deferred. Ordered flow: agents first, then providers |

## Suggestions Integrated

### Accepted

| Suggestion | Integration |
|------------|-------------|
| S1: Reuse BAS GuidedTour framework | Extract or replicate the GuidedTour pattern (step tracking, skip/complete, progress persistence) from browser-automation-studio for the onboarding step flows. Enables resume-after-abandon for non-technical users |
| S2: Real-time resource health visualization | After enabling a resource, show live health status (spinner → green checkmark) using existing health check infra (.vrooli/running-resources.json + HTTP endpoints). Reference tunnel-manager TunnelStatus pattern |
| S3: Secret validation and safe-entry UX | Trim whitespace, check format patterns (e.g. sk-or- prefix for OpenRouter), and do lightweight API pings before persisting to secrets.json. Show clear pass/fail feedback inline |
| S4: Defer prompt-manager/autoheal config to v2 | V1 deep-links to their existing UIs with brief explanations. Does not embed their configuration flows |
| S5: Generate personalized service.json | Use onboarding choices to generate a complete service.json with transitive dependencies auto-enabled and sensible defaults. Reusable config-generation logic for future scenarios |

### Not Accepted

_None — all suggestions were accepted._

## Refined Scope

### Included (Must Have)
- API with all onboarding/configuration logic
- CLI frontend with guided interactive flows
- Web UI frontend with identical functionality
- Guided setup for coding agents: resource-claude-code, resource-codex, resource-opencode
- Guided setup for AI providers: resource-openrouter, resource-ollama
- Secret entry with validation (format checks, whitespace trimming, API ping verification)
- Real-time resource health feedback during and after setup
- Personalized service.json generation based on choices (with transitive dependency resolution)
- Progress persistence so users can abandon and resume
- Dashboard/hub view for returning users to see current configuration state

### Included (Should Have)
- GuidedTour-style step framework (reuse BAS pattern) for UI flows
- Deep-links to vrooli-autoheal and prompt-manager UIs with contextual explanations
- Clear ordering: coding agents first, then AI providers

### Excluded (Out of Scope)
- Embedded autoheal configuration — deep-link only (per S4)
- Embedded prompt-manager configuration — deep-link only (per S4)
- Resources beyond coding agents and AI providers (postgres, redis, qdrant, etc.)
- Desktop/mobile deployment tiers
- Multi-user or team onboarding flows

### Deferred (Future)
- Full configuration hub for all 46+ resources — Target: v2
- Embedded autoheal/prompt-manager config panels — Target: v2
- Onboarding templates per use case (e.g. "data science starter", "web dev starter") — Target: v2+
- Hardware-specific onboarding for Vrooli server appliances — Target: Phase 3

## Implementation Notes

### Technical Approach
- **Standard scenario structure**: api/, cli/, ui/ directories
- **API-first**: All logic (resource enablement, secret validation, service.json generation, health checks) lives in the API. CLI and UI are thin frontends
- **Step framework**: Port the GuidedTour pattern from BAS (step definitions, progress tracking, skip/complete state). API stores progress; UI renders it
- **Config generation**: API endpoint accepts user choices, resolves transitive dependencies, and produces a valid service.json. This becomes a reusable capability
- **Health integration**: Poll .vrooli/running-resources.json and resource HTTP health endpoints. Stream status to UI via SSE or polling

### Integration Points
- **resource-claude-code, resource-codex, resource-opencode**: Enable/configure via service.json manipulation + secrets.json for API keys
- **resource-openrouter**: Secret validation via list-models API ping
- **resource-ollama**: Local service health check (no API key needed, but verify ollama is running)
- **vrooli-autoheal**: Deep-link to its UI for configuration (v1)
- **prompt-manager**: Deep-link to its UI for skill/team configuration (v1)
- **.vrooli/service.json**: Read current state, write generated configurations
- **.vrooli/secrets.json**: Read/write API keys with validation
- **.vrooli/running-resources.json**: Read for health status

### Dependencies
- Existing health check infrastructure (running-resources.json, HTTP health endpoints)
- BAS GuidedTour component as reference (not a hard dependency — can replicate pattern)
- Resource-specific API key format knowledge for validation
- vrooli CLI for resource lifecycle management

### Risks & Mitigations

| Risk | Mitigation |
|------|------------|
| API key validation pings may be slow or rate-limited | Use async validation with timeout; show pending state; allow skip with warning |
| Transitive dependency resolution could produce invalid configs | Start simple (hardcoded dependency maps for v1 resources); validate generated service.json against schema |
| Non-technical users may not understand "resources" concept | Use plain language: "coding tools", "AI services". Add brief explanations at each step |
| Health checks may be flaky during initial resource startup | Use polling with backoff; show "starting up..." state distinct from "failed" |

## Success Criteria
- [ ] Non-technical user can go from fresh Vrooli install to working coding agent in under 10 minutes using either CLI or UI
- [ ] All v1 resources (claude-code, codex, opencode, openrouter, ollama) have guided setup flows
- [ ] Secret validation catches common mistakes (whitespace, wrong key type) before persisting
- [ ] Generated service.json is valid and includes all transitive dependencies
- [ ] Users can return to the hub to see current resource status and modify configuration
- [ ] Progress persists across sessions (abandon and resume works)
- [ ] Health visualization shows real-time resource status after enablement

## Readiness Gate
- [x] All critical questions answered
- [x] Scope clearly defined
- [x] Technical approach validated (standard scenario pattern, existing infra)
- [x] Dependencies available (health check infra, BAS reference, resource APIs)
- [x] Success criteria measurable
- [x] Archive materials incorporated (N/A — no archive materials)

**Ready for processing:** Yes

## Staging Artifacts Produced
- `enhance/prd-context.md` — Synthesized PRD context covering overview, targets by priority, tech direction, and dependencies
- `enhance/requirements-context.md` — Validation approach and technical constraints for requirements generation
- `enhance/doc-outlines.md` — Not produced (no documentation source material)
