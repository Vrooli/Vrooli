# Implementation Plan: Expose Approval Gate Controls in Deployment Manager

## 1. Purpose

Surface the existing deployment-manager approval-gating backend (7 API endpoints, full DB schema, orchestrator integration) through operator-facing CLI commands and UI pages. This closes the gap where approval state is only accessible via raw API calls.

## 2. Required Reading

```bash
prompt-manager skill read cli-steer api-steer implementation-plan-authoring seam-discovery-and-enforcement
```

**Research dependency:** `research/desktop-release-control-plane-audit` (completed) — provides the system contract and dependency map. Key findings:
- deployment-manager owns canonical approval state (Finding 1)
- Release gate check is `GET /profiles/{id}/release-gate?commit={hash}` (Finding 1)
- 7 approval API endpoints already exist and are fully implemented (Finding 2)
- CLI and UI have zero approval coverage currently

## 3. Problem Statement

The approval-gating backend is production-ready at the API level:
- `deployment_approvals` and `profile_required_platforms` tables with indexes
- 7 REST endpoints: create, list, get, decide, release-gate check, set/get required platforms
- Orchestrator integration that blocks deployment when gate is not satisfied (HTTP 412)

However, operators cannot use this system without crafting raw HTTP requests. There are:
- **No CLI commands** for any approval operation
- **No UI pages or components** for viewing/managing approvals
- **No TypeScript types** in the UI api client for approval data

## 4. Scope

**In scope:**
- New `approvals` CLI command group with subcommands mirroring all 7 API endpoints
- New UI page(s) for approval management (list, decide, gate status)
- Required platforms configuration in UI (per-profile settings)
- TypeScript types and API client functions for approval endpoints
- Integration into existing ProfileDetail page (gate status indicator)

**Out of scope:**
- Modifying the existing API endpoints or database schema
- Approval-to-deploy automation (separate backlog item: Action 5 from research)
- Webhook/event notifications (separate backlog item: Action 4 from research)
- Rollback API (separate backlog item: Action 3 from research)
- Visual validation integration (existing separate workflow)

## 5. Current Technical Context

### Backend (complete, no changes needed)
- **Schema:** `initialization/postgres/schema.sql:51-71` — `deployment_approvals`, `profile_required_platforms`
- **Migration:** `api/migrations/003_add_deployment_approvals.sql`
- **Types:** `api/deployments/approvals_types.go` — `DeploymentApproval`, `ReleaseGateStatus`, `PlatformGateStatus`, etc.
- **Handler:** `api/deployments/approvals_handler.go` — 7 handler methods
- **Repository:** `api/deployments/approvals_repository.go` — full CRUD + gate check
- **Routes:** `api/server/routes.go:70-79` — all 7 endpoints registered
- **Orchestrator:** `api/deployments/orchestrator.go:165-192` — gate check blocks deploys

### CLI (no approval coverage)
- Uses cli-core pattern with domain-specific command packages
- Existing groups: overview, profiles, swaps, deployments, secrets, signing, validations, config
- Pattern: each group is a package under `cli/` with a `commands.go`

### UI (no approval coverage)
- React + Vite, uses `@vrooli/api-base` for URL resolution
- `ui/src/lib/api.ts` — typed fetch wrapper with `apiFetch<T>()` helper
- Existing pages: Dashboard, Profiles, ProfileDetail, Deployments, Validation, Analyze, BundleTelemetry
- No approval types or API functions defined

## 6. Target End State

After implementation:
1. `deployment-manager approvals list <profile-id>` — shows approval history
2. `deployment-manager approvals get <id>` — shows single approval detail
3. `deployment-manager approvals create <profile-id> --commit <hash> --platform <plat>` — creates pending approval
4. `deployment-manager approvals decide <id> --decision approved --reviewer <name>` — approves/rejects
5. `deployment-manager approvals gate <profile-id> --commit <hash>` — checks release gate
6. `deployment-manager approvals platforms set <profile-id> --platforms win,mac,linux` — configures required platforms
7. `deployment-manager approvals platforms get <profile-id>` — shows required platforms
8. UI ProfileDetail page shows release gate status badge and required platforms config
9. New Approvals UI page for listing, filtering, and deciding approvals
10. All approval TypeScript types and API client functions defined

## 7. Implementation Strategy

### Phase 1: CLI — `approvals` command group
- Create `cli/approvals/` package following the validations pattern
- Register as new CommandGroup in `app.go`
- Wire `App.approvals` field with `approvals.New(core.APIClient)`
- Implement subcommand routing in `commands.go`
- Each subcommand: parse flags → call API → format output (human default, `--json` for machine)

**Commands:**
| Subcommand | API Endpoint | Key Flags |
|------------|-------------|-----------|
| `list <profile-id>` | `GET /profiles/{id}/approvals` | `--commit` (optional filter) |
| `get <id>` | `GET /approvals/{id}` | — |
| `create <profile-id>` | `POST /profiles/{id}/approvals` | `--commit`, `--platform`, `--validation-id` |
| `decide <id>` | `POST /approvals/{id}/decide` | `--decision`, `--reviewer`, `--notes` |
| `gate <profile-id>` | `GET /profiles/{id}/release-gate` | `--commit` |
| `platforms set <profile-id>` | `PUT /profiles/{id}/required-platforms` | `--platforms` (CSV) |
| `platforms get <profile-id>` | `GET /profiles/{id}/required-platforms` | — |

### Phase 2: UI — TypeScript types + API client
- Add approval types to `ui/src/lib/api.ts`
- Add API client functions for all 7 endpoints

### Phase 3: UI — Approval components and pages
- Add release gate status badge to ProfileDetail page
- Add required platforms configuration section to ProfileDetail
- Create Approvals page with:
  - Approval list table (filterable by profile, commit, status)
  - Approval detail view with decide action
  - Release gate status summary

### Phase 4: UI routing + navigation
- Add Approvals page route to router
- Add navigation link in Layout sidebar

## 8. Contract Decisions

- **CLI output:** Human-friendly default output following the data-retrieval contract (Summary → Results → Retrieval hints) for list/get, mutation contract for create/decide, and operational contract for gate check
- **CLI subcommand routing:** Single `approvals` top-level command with subcommand dispatch (like `validations`)
- **UI page location:** Standalone `/approvals` page + gate status integrated into existing ProfileDetail
- **No new API endpoints needed** — everything uses the existing 7 endpoints

## 9. Testing Plan

- **CLI unit tests:** Test argument parsing and help output for each subcommand
- **CLI integration tests:** Test against running API (if test infrastructure supports it)
- **UI component tests:** Test approval components render correctly with mock data
- **Manual verification:** 
  - Create a profile → set required platforms → create pending approval → decide → check gate
  - Verify gate blocks deployment when not all platforms approved

## 10. Rollout / Validation Checklist

- [ ] CLI builds without errors: `cd cli && go build ./...`
- [ ] CLI tests pass: `cd cli && go test ./...`
- [ ] UI builds without errors: `cd ui && npm run build`
- [ ] UI tests pass: `cd ui && npm test`
- [ ] All 7 approval operations work via CLI
- [ ] ProfileDetail shows gate status
- [ ] Approvals page lists and filters approvals
- [ ] Approval decide action works from UI

## 11. Risks + Mitigations

| Risk | Likelihood | Impact | Mitigation |
|------|-----------|--------|------------|
| API endpoint behavior differs from handler code reading | Low | Medium | Verify with integration test before building CLI |
| UI build time (5-10 min) slows iteration | Medium | Low | Test components in isolation first |
| Existing CLI tests break from new command group | Low | Low | New group is additive; run full test suite |

## 12. Non-goals / Prohibited Patterns

- Do NOT modify existing API endpoints or database schema
- Do NOT implement business logic in CLI (thin wrapper only)
- Do NOT add approval-to-deploy automation (separate backlog item)
- Do NOT bypass cli-core patterns (use ScenarioApp, APIClient, ParseInterspersed)
- Do NOT add bash scripts for CLI functionality

## 13. Definition of Done

1. All 7 approval API endpoints have corresponding CLI commands
2. All CLI commands produce human-friendly default output and support `--json`
3. UI has TypeScript types for all approval models
4. UI has API client functions for all approval endpoints
5. ProfileDetail page displays release gate status and required platforms config
6. Standalone Approvals page allows listing, filtering, and deciding approvals
7. `go build ./...` and `go test ./...` pass in `cli/`
8. `npm run build` and `npm test` pass in `ui/`
