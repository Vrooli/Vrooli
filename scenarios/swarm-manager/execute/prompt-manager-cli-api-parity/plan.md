# Plan: Prompt-Manager CLI Must Be Thin Wrapper Over API

## Purpose

Restore the morning-vision-walk skill (and any skill that drives prompt-manager via CLI) by ensuring the prompt-manager CLI is a complete thin wrapper over its HTTP API. Specifically: ship the missing `team decision-accept` subcommand, audit and close all other CLI/API gaps, and add a CI guard that prevents future drift.

## Required Reading

```bash
prompt-manager skill read cli-steer api-steer utils-unification seam-discovery-and-enforcement
```

Domain references (read before coding):
- `scenarios/prompt-manager/docs/reference/heartbeat-api.md` — decisions/tasks/knowledge endpoints
- `scenarios/prompt-manager/docs/reference/heartbeat-cli.md` — current heartbeat-state CLI surface
- `scenarios/prompt-manager/docs/reference/api-endpoints.md` — full v1 route docs
- `scenarios/prompt-manager/docs/reference/cli-commands.md` — current CLI surface

## Problem Statement

On 2026-04-23 the morning-vision-walk skill called `prompt-manager team decision-accept …` and failed: the CLI does not expose that subcommand even though the API supports `PATCH /teams/{id}/decisions/{decisionId}` (`scenarios/prompt-manager/api/heartbeat/handlers.go:2115`). Root cause is not just the one missing command — the CLI/API parity contract is not enforced anywhere, so any future endpoint can ship without a CLI counterpart and break the next CLI-driven skill.

## Scope

**In scope**
- Add `team decision-accept` subcommand wrapping PATCH on `/teams/{id}/decisions/{decisionId}`.
- Audit every prompt-manager v1 API endpoint against the CLI command tree; add missing subcommands.
- Land a CI test (Go test, not external linter) that fails when an API route has no CLI mapping.
- Update `docs/reference/cli-commands.md` and `docs/reference/heartbeat-cli.md` to reflect the new commands.

**Out of scope**
- Refactoring the CLI dispatcher into codegen / OpenAPI tooling (separate initiative if proposed).
- Changing API behavior (no new endpoints, no payload changes).
- Changes to the morning-vision-walk skill itself (consumer; will work once CLI lands).
- Web UI / other consumers of the API.

## Current Technical Context

CLI dispatcher (file-per-domain, central switch):
- `scenarios/prompt-manager/cli/app.go:82` — domain registration.
- `scenarios/prompt-manager/cli/teams/teams.go:539` — `route()` switch for `team` subcommands.
- `teams.go:618` `decision-add`, `teams.go:620` `decision-list` — sibling commands; `decision-accept` is absent.

API routes:
- `scenarios/prompt-manager/api/main.go:299–585` — full v1 route table (≈166 endpoints).
- `scenarios/prompt-manager/api/heartbeat/handlers.go:2115` — PATCH decision handler.
- `scenarios/prompt-manager/api/heartbeat/models.go:207–220` — PATCH payload (selected, notes, freeform, status, …).

Approval-mode enforcement:
- Header `X-Caller-ID` (`handlers.go:2226`); empty or `ui-user` is treated as human (`handlers.go:2227`).
- Team field `DecisionMode` ∈ {`approval`, `yolo`} (`handlers.go:2221`); approval mode blocks agents from setting `accepted`/`rejected` (`handlers.go:2249`).

Tests:
- `scenarios/prompt-manager/cli/teams/teams_test.go` — flag parsing only.
- `scenarios/prompt-manager/api/heartbeat/handlers_decision_test.go` — handler unit tests.
- No CLI/API parity test exists.

## Target End State

1. `prompt-manager team decision-accept --team <id> --decision <id> --selected <key> [--notes <text>] [--as-agent <id>]` issues the PATCH and prints the updated decision.
2. Every other `/api/v1/...` endpoint has at least one CLI subcommand reachable from `prompt-manager <domain> <verb>`. Endpoints intentionally without CLI parity are listed in an explicit allowlist.
3. `go test ./scenarios/prompt-manager/...` includes a parity test that fails when a new route is added without a CLI mapping (or allowlist entry).
4. CLI reference docs reflect the new surface.

## Implementation Strategy

### Phase 1 — `team decision-accept`

1. Add `cmdDecisionAccept` to `scenarios/prompt-manager/cli/teams/teams.go`, registered in `route()`.
2. Flags: `--team`, `--decision`, `--selected`, `--notes` (optional), `--freeform` (optional), `--status` (optional, default "accepted"), `--as-agent <id>` (optional → sets `X-Caller-ID`; absent = human caller).
3. Use existing HTTP helper that other `team` commands use; PATCH `/teams/{id}/decisions/{decisionId}` with payload built from flags.
4. Unit test parallels `handlers_decision_test.go`: human-caller path and `--as-agent` path.

### Phase 2 — API/CLI parity audit

1. Enumerate v1 routes from `api/main.go` (single file, single registration pattern → AST-walk or string-scan of `v1.HandleFunc(`).
2. Enumerate CLI subcommand routes from each domain's `route()` switch (string-scan of `case "..."`).
3. Diff into a parity table; classify each missing route:
   - Add CLI subcommand (default action), or
   - Add to allowlist with rationale (e.g., health, auth bootstrap, internal-only).
4. Implement the missing CLI subcommands in their domain's existing dispatcher file.

### Phase 3 — CI guard

1. New file: `scenarios/prompt-manager/api/parity_test.go` (sits in `api/` so it can import the route table or re-parse `main.go`).
2. Test loads:
   - Route set from API (parsed from `main.go`).
   - CLI command set from each domain's `Commands()` registration.
   - Allowlist from a new `scenarios/prompt-manager/api/parity_allowlist.go` (named map with rationale strings).
3. Test fails with a deterministic, copy-pasteable diff: `missing CLI for: PATCH /teams/{id}/decisions/{decisionId} — add a subcommand or allowlist with reason.`
4. Wire into existing `go test ./...` — no separate CI job needed.

### Phase 4 — Docs

1. Update `docs/reference/cli-commands.md` and `docs/reference/heartbeat-cli.md` with new subcommands.
2. Add a short "CLI must mirror API" note in `docs/reference/api-endpoints.md` linking to the parity test.

## Contract Decisions

- **Command name:** `team decision-accept` (matches `decision-add`, `decision-list`).
- **Caller identity:** absent `--as-agent` → no `X-Caller-ID` header → API treats as human; with `--as-agent foo` → `X-Caller-ID: foo`.
- **Status default:** PATCH defaults to `selected` field write only; `--status` is opt-in to avoid accidental state transitions.
- **Parity unit:** "API endpoint" = (method, path-pattern). N:1 (one CLI command covering several GETs) is allowed when explicitly mapped.

## Testing Plan

- Unit: `team decision-accept` happy path, missing required flag, `--as-agent` header propagation.
- Integration: round-trip with running API — create decision, accept it, verify status.
- Parity: `parity_test.go` runs in `go test`, asserts every route is mapped.
- Manual: re-run morning-vision-walk against a local prompt-manager; the failing step from 2026-04-23 should succeed.

## Rollout / Validation Checklist

- [ ] `go build ./scenarios/prompt-manager/...` clean.
- [ ] `go test ./scenarios/prompt-manager/...` passes (including new parity test).
- [ ] `prompt-manager team decision-accept --help` documents all flags.
- [ ] CLI/API parity table empty (or only allowlisted entries with rationales).
- [ ] Morning-vision-walk dry-run completes through the decision-accept step.

## Risks + Mitigations

| Risk | Mitigation |
|---|---|
| AST-parsing `main.go` is brittle | Prefer importing a route-registry helper if one already exists; fall back to `go/parser` over a regex |
| Audit reveals dozens of missing CLI commands → scope creep | Allowlist with rationale is acceptable for non-critical endpoints; track follow-up items in backlog |
| Approval-mode regressions if `--as-agent` semantics misread | Mirror existing handler tests; explicit assertion that no header → no `X-Caller-ID` is sent |
| Parity test flakes when route ordering changes | Compare as sets, not slices; sort output |

## Non-goals / Prohibited Patterns

- No OpenAPI spec introduction in this item.
- No CLI codegen.
- No change to API payloads or status semantics.
- No silent CLI behavior changes for existing subcommands.

## Definition of Done

- `team decision-accept` exists, tested, documented.
- Parity test exists, passes, and demonstrably fails on a synthetically-added unmapped route.
- CLI/API gap list at zero or fully allowlisted.
- Morning-vision-walk skill verified working end-to-end.
