# Implementation Plan: Expose Approval Gate Controls in Deployment Manager

## 1. Purpose

Surface the existing deployment-manager approval-gating backend (7 API endpoints, full DB schema, orchestrator integration) through operator-facing CLI commands and UI pages. This closes the gap where approval state is only accessible via raw API calls.

## 2. Required Reading

```bash
prompt-manager skill read cli-steer api-steer implementation-plan-authoring seam-discovery-and-enforcement react-coherence
```

**Research dependency:** `research/desktop-release-control-plane-audit` (completed) — provides the system contract and dependency map. Key findings:
- deployment-manager owns canonical approval state (Finding 1)
- Release gate check is `GET /profiles/{id}/release-gate?commit={hash}` (Finding 1)
- 7 approval API endpoints already exist and are fully implemented (Finding 2)
- CLI and UI have zero approval coverage currently
- deployment-manager never talks to LPBS directly; flow is deployment-manager → scenario-to-desktop → LPBS (Finding 3)

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
- New Approvals UI page for listing, filtering, and deciding approvals (standalone `/approvals` route)
- Gate status badge integrated into ProfileDetail page (defaulting to latest deployment commit)
- Required platforms configuration as a checkbox Card section on ProfileDetail
- TypeScript types and API client functions for all approval endpoints
- Automated tests for CLI subcommands and UI components

**Out of scope:**
- Modifying the existing API endpoints or database schema
- Approval-to-deploy automation (separate backlog item: Action 5 from research)
- Webhook/event notifications (separate backlog item: Action 4 from research)
- Rollback API (separate backlog item: Action 3 from research)
- Visual validation integration (existing separate workflow)
- Real-time polling or WebSocket updates (use React Query defaults + manual refresh)

## 5. Current Technical Context

### Backend (complete, no changes needed)
- **Schema:** `initialization/postgres/schema.sql:51-71` — `deployment_approvals`, `profile_required_platforms`
- **Migration:** `api/migrations/003_add_deployment_approvals.sql`
- **Types:** `api/deployments/approvals_types.go` — `DeploymentApproval`, `ReleaseGateStatus`, `PlatformGateStatus`, etc.
- **Handler:** `api/deployments/approvals_handler.go` — 7 handler methods
- **Repository:** `api/deployments/approvals_repository.go` — full CRUD + gate check
- **Routes:** `api/server/routes.go:70-79` — all 7 endpoints registered (conditional on `ApprovalsHandler != nil`)
- **Orchestrator:** `api/deployments/orchestrator.go:165-192` — gate check blocks deploys (HTTP 412)

### CLI (no approval coverage)
- Uses cli-core pattern with domain-specific command packages
- Existing groups: overview, profiles, swaps, deployments, secrets, signing, validations, config, bundles
- Pattern: each group is a package under `cli/` with `commands.go`, `Commands` struct, `New(api *cliutil.APIClient)` constructor, `Run(args []string) error` dispatcher
- Flag parsing: `flag.NewFlagSet` + `cliutil.ParseInterspersed(fs, args)`
- Format system: global `cmdutil.SetGlobalFormat`/`ResolveFormat` with `--json` consumed in `app.go`; default output is human-readable
- Output: `cmdutil.PrintByFormat(format, body)` dispatches to JSON or plain text

### UI (no approval coverage)
- React + Vite, uses `@vrooli/api-base` for URL resolution
- `ui/src/lib/api.ts` — typed fetch wrapper with `apiFetch<T>()` helper
- Existing pages: Dashboard, Profiles, ProfileDetail, Deployments, Validation, Analyze, BundleTelemetry
- Routing: `ui/src/App.tsx` with `react-router-dom` `<Routes>` / `<Route>`
- Navigation: `ui/src/components/Layout.tsx` `navItems` array (path, label, icon from lucide-react)
- State: React Query (`useQuery`/`useMutation`) for server state; local `useState` for UI state
- Components: shadcn UI (Card, Button, Badge, etc.)
- Tests: Vitest + React Testing Library, `vi.mock('../lib/api')`, `createWrapper()` with QueryClient + BrowserRouter

### API Types (for reference in CLI/UI implementation)

```go
// Request types
CreateApprovalRequest { GitCommitHash, Platform, ValidationID? }
ApprovalDecisionRequest { Decision ("approved"|"rejected"), Reviewer, Notes? }
SetRequiredPlatformsRequest { Platforms []string }

// Response types
DeploymentApproval { ID, ProfileID, GitCommitHash, Platform, Status, ApprovedBy?, ApprovedAt?, Notes?, ValidationID?, CreatedAt, UpdatedAt }
ReleaseGateStatus { ProfileID, GitCommitHash, Ready bool, Platforms []PlatformGateStatus }
PlatformGateStatus { Platform, Required bool, Status ("pending"|"approved"|"rejected"|"stale"|"missing") }
```

## 6. Target End State

After implementation:
1. `deployment-manager approvals list <profile-id>` — shows approval history
2. `deployment-manager approvals get <id>` — shows single approval detail
3. `deployment-manager approvals create <profile-id> --commit <hash> --platform <plat>` — creates pending approval
4. `deployment-manager approvals decide <id> --decision approved --reviewer <name>` — approves/rejects
5. `deployment-manager approvals gate <profile-id> --commit <hash>` — checks release gate (operational output: Status → Triage → Next Steps)
6. `deployment-manager approvals platforms set <profile-id> --platforms win,mac,linux` — configures required platforms
7. `deployment-manager approvals platforms get <profile-id>` — shows required platforms
8. UI ProfileDetail page shows release gate status badge (defaults to latest deployment commit, with commit selector) and required platforms config (checkbox Card section)
9. New `/approvals` UI page for listing, filtering, and deciding approvals (inline quick-action buttons + detail view with validation link)
10. All approval TypeScript types and API client functions defined
11. Navigation sidebar includes Approvals link

## 7. Implementation Strategy

### Phase 1: CLI — `approvals` command group
- Create `cli/approvals/` package following the validations pattern
- Register as new CommandGroup in `app.go`
- Wire `App.approvals` field with `approvals.New(core.APIClient)`
- Implement subcommand routing in `commands.go` with nested `platforms` sub-dispatch

**Commands:**
| Subcommand | API Endpoint | Key Flags | Output Contract |
|------------|-------------|-----------|-----------------|
| `list <profile-id>` | `GET /profiles/{id}/approvals` | `--commit` (optional filter) | Data retrieval |
| `get <id>` | `GET /approvals/{id}` | — | Data retrieval |
| `create <profile-id>` | `POST /profiles/{id}/approvals` | `--commit`, `--platform`, `--validation-id` | Mutation |
| `decide <id>` | `POST /approvals/{id}/decide` | `--decision`, `--reviewer`, `--notes` | Mutation |
| `gate <profile-id>` | `GET /profiles/{id}/release-gate` | `--commit` | Operational (Status → Triage → Next Steps) |
| `platforms set <profile-id>` | `PUT /profiles/{id}/required-platforms` | `--platforms` (CSV) | Mutation |
| `platforms get <profile-id>` | `GET /profiles/{id}/required-platforms` | — | Data retrieval |

### Phase 2: CLI tests
- Unit tests for all 7 subcommands using `httptest.NewServer` mocking
- Test flag parsing, output format (human default + `--json`), error handling
- Follow table-driven pattern from `cli/deployments/commands_test.go`

### Phase 3: UI — TypeScript types + API client
- Add approval types to `ui/src/lib/api.ts` matching Go types
- Add API client functions for all 7 endpoints:
  - `listApprovals(profileId, commit?)`, `getApproval(id)`, `createApproval(profileId, req)`, `decideApproval(id, req)`
  - `checkReleaseGate(profileId, commit)`, `setRequiredPlatforms(profileId, platforms)`, `getRequiredPlatforms(profileId)`

### Phase 4: UI — ProfileDetail integration
- Gate status badge Card section: queries release gate for latest deployment commit, shows Ready/Blocked with per-platform status
- Commit selector dropdown to check other commits
- Required platforms checkbox Card section (collapsible): three checkboxes (Windows, macOS, Linux) with Save button

### Phase 5: UI — Standalone Approvals page
- Route: `/approvals`
- Sidebar nav item with appropriate lucide-react icon
- Approval list table with filtering (by profile, commit, status)
- Inline approve/reject quick-action buttons in table rows
- Detail view on row click: notes field, reviewer attribution, validation link (if validation_id set)
- React Query for data fetching (defaults: refetch on focus/mount) + manual Refresh button

### Phase 6: UI tests
- Component tests for Approvals page (list rendering, filter interaction, approve/reject actions)
- Component tests for ProfileDetail gate badge and required platforms config
- Follow Vitest + React Testing Library pattern from `ui/src/pages/Profiles.test.tsx`

## 8. Contract Decisions (Settled)

- **CLI output:** Human-friendly default output. Gate check uses operational contract (Status → Triage → Next Steps). List/get use data-retrieval contract. Create/decide use mutation contract.
- **CLI subcommand routing:** Single `approvals` top-level command with nested `platforms` sub-dispatch for set/get.
- **UI page structure:** Standalone `/approvals` page + gate status badge and required-platforms config integrated into ProfileDetail.
- **UI approval decide UX:** Inline quick-action buttons in list table + detail view with notes, reviewer attribution, and validation link.
- **UI data freshness:** React Query defaults (refetch on focus/mount) + manual refresh button. No polling.
- **No new API endpoints needed** — everything uses the existing 7 endpoints.

## 9. Testing Plan

### CLI automated tests (Phase 2)
- **Per-subcommand tests** using `httptest.NewServer`:
  - `list`: verify query param forwarding (`?commit=`), output includes approval summary table
  - `get`: verify path param, output includes all approval fields
  - `create`: verify POST body marshaling, required flags validation
  - `decide`: verify decision and reviewer flags, error on missing required flags
  - `gate`: verify operational output format (status, triage, next steps), both ready and blocked states
  - `platforms set`: verify CSV parsing and PUT body, error on empty platforms
  - `platforms get`: verify output lists platform names
- **Format tests**: verify `--json` produces valid JSON for each subcommand
- **Error tests**: verify HTTP error responses produce user-friendly messages

### UI automated tests (Phase 6)
- **Approvals page:**
  - Renders approval list with mock data
  - Filter by status works
  - Approve/reject inline buttons trigger correct API calls
  - Error state renders error message
  - Empty state renders helpful message
- **ProfileDetail gate badge:**
  - Renders "Ready" badge when all platforms approved
  - Renders "Blocked" badge with platform breakdown when not ready
  - Commit selector triggers new gate check
- **Required platforms config:**
  - Renders checkboxes matching current required platforms
  - Save button calls setRequiredPlatforms with correct array
  - Success feedback on save

## 10. Rollout / Validation Checklist

- [ ] CLI builds without errors: `cd cli && go build ./...`
- [ ] CLI tests pass: `cd cli && go test ./... -timeout 300s`
- [ ] UI builds without errors: `cd ui && npm run build`
- [ ] UI tests pass: `cd ui && npm test`
- [ ] All 7 approval operations work via CLI (human-readable and --json output)
- [ ] Gate check shows operational output with triage and next steps
- [ ] ProfileDetail shows gate status badge defaulting to latest deployment commit
- [ ] ProfileDetail shows required platforms config with checkboxes
- [ ] Approvals page lists and filters approvals
- [ ] Approval decide action works from both inline buttons and detail view
- [ ] Sidebar navigation includes Approvals link

## 11. Risks + Mitigations

| Risk | Likelihood | Impact | Mitigation |
|------|-----------|--------|------------|
| API endpoint behavior differs from handler code reading | Low | Medium | CLI integration tests against mock server verify request/response shapes |
| UI build time (5-10 min) slows iteration | Medium | Low | Test components in isolation first; use `npm test` for fast feedback |
| Existing CLI tests break from new command group | Low | Low | New group is additive; run full test suite before merging |
| ProfileDetail gate badge depends on deployment history for default commit | Low | Medium | Graceful fallback: show "No deployments yet" with commit input field if no deployment history |
| Multiple operators deciding same approval concurrently | Low | Medium | API handles this (last write wins with status update); UI refetches on mutation success |

## 12. Non-goals / Prohibited Patterns

- Do NOT modify existing API endpoints or database schema
- Do NOT implement business logic in CLI (thin wrapper only)
- Do NOT add approval-to-deploy automation (separate backlog item)
- Do NOT bypass cli-core patterns (use ScenarioApp, APIClient, ParseInterspersed)
- Do NOT add bash scripts for CLI functionality
- Do NOT introduce polling/WebSocket for the approvals page (use React Query defaults)
- Do NOT add a new state management pattern — use React Query for server state, useState for local UI state

## 13. Definition of Done

1. All 7 approval API endpoints have corresponding CLI commands with human-friendly default output
2. All CLI commands support `--json` flag for machine-readable output
3. CLI automated tests cover all 7 subcommands (flag parsing, output format, error handling)
4. UI has TypeScript types for all approval models
5. UI has API client functions for all approval endpoints
6. ProfileDetail page displays release gate status badge (with commit selector) and required platforms config (checkboxes)
7. Standalone Approvals page allows listing, filtering, and deciding approvals (inline + detail view)
8. Sidebar navigation includes Approvals link
9. UI automated tests cover Approvals page and ProfileDetail approval integration
10. `go build ./...` and `go test ./...` pass in `cli/`
11. `npm run build` and `npm test` pass in `ui/`
