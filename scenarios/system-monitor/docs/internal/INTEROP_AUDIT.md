# Interoperability Audit: system-monitor

**Date**: 2026-02-17
**Scenario**: system-monitor
**Dependencies**: agent-manager (required)

## Communication Paths

| Path | Protocol | Status |
|------|----------|--------|
| API → agent-manager | HTTP + protojson | Good |
| UI → API | HTTP + JSON | Hardened (this audit) |

## Findings

### Resolved

#### F1: Hardcoded `localhost:port` in investigation handler response (HIGH)
- **File**: `api/internal/handlers/investigations.go`
- `resolveAPIBaseURL()` built `http://localhost:<port>` for the `TriggerInvestigation` response.
- **Fix**: Updated to derive URL from `X-Forwarded-Host`/`X-Forwarded-Proto` headers, falling back to `r.Host`, then to configured port.

#### F4: Investigation status values stringly-typed across UI and API (MEDIUM)
- **UI File**: `ui/src/features/investigations/hooks/useInvestigationAgents.ts`
- **Types File**: `ui/src/types/api.ts`
- Terminal statuses were inline with extra values (`"canceled"`) not sent by the API.
- **Fix**: Created `INVESTIGATION_TERMINAL_STATUSES` in `types/api.ts` with aligned values. Removed inline constant.

### Documented (no code change)

#### F2: No proto schemas for system-monitor's own domain (MEDIUM)
- API uses Go structs with `json:"snake_case"` tags; UI has 33 hand-written TypeScript interfaces mirroring them.
- Proto schemas exist only for consumed agent-manager types.
- **Future work**: Define `.proto` schemas for system-monitor's domain types and generate Go + TypeScript bindings.

#### F3: UI agent payload parsing overly defensive (MEDIUM)
- `mapAgentPayload()` tries multiple fallback field names (`start_time` / `startTime` / `started_at`).
- API sends consistent `snake_case`; camelCase fallbacks are unnecessary but safe.
- **Future work**: Remove camelCase fallbacks once proto schemas enforce the contract (see F2).

#### F5: Resource URLs use hardcoded localhost defaults (LOW)
- `api/internal/config/config.go:141-146` — resource URLs (Postgres, Redis, QuestDB) default to `localhost:<port>`.
- These are **resources** (not scenarios), so discovery doesn't apply. Defaults are correct for local dev; production overrides via env vars.

#### F6: Health endpoint self-reference uses hardcoded localhost (LOW)
- `api/internal/config/config.go:298` — `loadHealthEndpoints()` builds `http://localhost:<API_PORT>/health`.
- Self-check within the same process; localhost is appropriate.

## What's Good (no changes needed)
- **Agent-manager client** (`api/internal/agentmanager/client.go`): Proto-generated types, `protojson` marshal/unmarshal with `DiscardUnknown: true`, per-request `discovery.ResolveScenarioURLDefault()`.
- **Terminal status handling**: Uses proto enum constants (`RUN_STATUS_COMPLETE`, etc.) — no stringly-typed comparisons.
- **Dependency declaration**: `agent-manager` properly declared as required in `.vrooli/service.json`.
- **No hardcoded ports** for inter-scenario calls.

## Completion Gates
- [x] No hardcoded `localhost` in non-config, non-test API responses
- [x] Terminal statuses centralized and aligned with API contract
- [ ] Proto schemas for system-monitor domain types (future)
- [ ] Remove defensive camelCase fallbacks in UI payload parsing (future, blocked by proto schemas)
