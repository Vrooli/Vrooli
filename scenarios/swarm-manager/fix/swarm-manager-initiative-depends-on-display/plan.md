# Plan: Display initiative `depends_on` in CLI human output and UI

## Purpose
Fix display gap: the initiative data model and API already carry `depends_on: ["<initiative-name>"]`, but the human-readable surfaces (CLI `initiatives get` / `initiatives list`, and UI initiative detail / list) do not show it. Operators currently cannot see cross-initiative dependencies without reading raw JSON. This is a display-only change across two surfaces.

## Required Reading
```
prompt-manager skill read scientific-debugging cli-steer api-steer seam-discovery-and-enforcement react-coherence ux
```

## Greenfield Constraint
This is greenfield work. Do not add compatibility shims, legacy wrappers, dead code, unused re-exports, `// removed` comments, or renamed `_unused` variables.

## Problem Statement
- `swarm-manager initiatives get --name <n>` (human/text mode) prints title, status, priority, description, member items — but not `depends_on`.
- `swarm-manager initiatives list` (human/text mode) prints one row per initiative — but not `depends_on`.
- UI initiative detail view does not surface upstream dependencies.
- UI initiative list view does not surface them either.
- JSON output from the same commands already includes `depends_on`, confirming the data is available end-to-end.

This was surfaced 2026-04-24, initially misdiagnosed as a data-model gap and corrected to display-only after code inspection.

## Scope
**In scope**
- CLI human formatter for `initiatives get`: add a `Depends on: a, b, c` line when non-empty.
- CLI human formatter for `initiatives list`: add a `Depends on` column/line per row when non-empty.
- UI initiative **detail** view: render upstream dependencies under a "Depends on" label.
- UI initiative **list** view: render upstream dependencies inline in compact form.
- Tests for both CLI formatters.

**Out of scope**
- Data model, API, or validation changes (already done and tested).
- JSON output (already correct).
- Showing **downstream** (what depends on *this* initiative) — upstream only per d3.
- Filtering/sorting initiatives by dependency.
- Truncation/wrapping rules for long dep lists — comma-join all, per d4.
- Handling broken refs at display time beyond what the API already returns.

## Acceptance Allow / Deny
- `acceptance_allow`: `scenarios/swarm-manager/**` (already set on spec).
- `acceptance_deny`: not needed — change is self-contained to swarm-manager scenario.

## Current Technical Context
This workshop round is running in a workspace that contains agent-manager only; the swarm-manager scenario source lives in a separate workspace and must be checked out before execution. Specific files to locate (executing agent, first step):

- CLI human formatters for `initiatives get` and `initiatives list` (Go). Search: `func` definitions that print `Title:` / `Status:` / `Priority:` for an initiative; likely under `scenarios/swarm-manager/cli/...` or similar. The same area renders `Members:` for `get`.
- UI initiative detail view (TSX). Search the swarm-manager UI for the initiative detail page that shows title/status/priority/description.
- UI initiative list view (TSX). Search for the initiatives list/index component.
- Existing CLI formatter tests — locate one and follow the same pattern (likely golden-string assertions or table-driven cases).

`depends_on` arrives as `[]string` of bare initiative names from the API. Empty array is the common case and must be handled per d1 (omit entirely).

## Target End State
- `swarm-manager initiatives get --name X` (human) prints `Depends on: a, b, c` when non-empty; line is omitted entirely when empty.
- `swarm-manager initiatives list` (human) shows the same information per row when non-empty; omitted when empty. Full comma-joined list, no truncation.
- UI detail view shows a "Depends on" labeled section listing deps; section omitted when empty.
- UI list view shows deps inline per row in compact form; omitted when empty.
- Tests cover: empty (line absent), single dep, multiple deps, ordering preserved.

## Implementation Strategy

### Phase 1 — CLI: `initiatives get` human formatter
1. Locate the formatter function and the section that renders top-level scalar fields.
2. When `depends_on` is non-empty, insert a `Depends on: a, b, c` line after `Priority:` (before description/members). Comma-separated, preserving the order returned by the API.
3. When `depends_on` is empty, emit nothing (no placeholder).
4. Add/extend tests: empty (no line), single dep, multiple deps.

### Phase 2 — CLI: `initiatives list` human formatter
1. Locate the row formatter.
2. When `depends_on` is non-empty, render the label `Depends on` with a comma-joined list in the same per-row style as other fields. Full list, no truncation.
3. Omit the field from the row when empty.
4. Add/extend tests covering row output with and without deps.

### Phase 3 — UI: initiative detail view
1. Read `depends_on` from the existing initiative payload (no API change).
2. Render under a "Depends on" label. Each dep should link to that initiative's detail page if such a link already exists in the UI; otherwise plain text is fine.
3. Omit the section entirely when `depends_on` is empty.

### Phase 4 — UI: initiative list view
1. Render compact form per row, consistent with CLI list, under the label "Depends on". Linkify deps if cheap to do so.
2. Omit when empty.

### Final: Cleanup & Verification
- Run type checks (`go build ./...` / `tsc --noEmit`) and fix all errors, including pre-existing in modified files.
- Run linters; fix all warnings in modified files.
- Run unit tests; all green.
- Restart scenario: `vrooli scenario restart swarm-manager`.
- Verify health: `swarm-manager initiatives get --name <some-init-with-deps>` shows the new line; UI loads the detail view.

## Contract Decisions
- **d1 — Empty-state behavior: Omit entirely when empty.** All four surfaces (CLI get, CLI list, UI detail, UI list) render nothing when `depends_on` is empty. No `(none)` placeholder, no empty label.
- **d2 — Label wording: "Depends on".** Used verbatim on CLI get, CLI list, UI detail, and UI list. Matches the field name and the spec's proposed wording.
- **d3 — Upstream only.** This fix displays `depends_on` only. Downstream ("what depends on this initiative") is explicitly out of scope; file a follow-up item if it becomes desired.
- **d4 — Long dep lists: comma-join all, no truncation.** Common case is 0–2 deps; faithful full rendering is preferred over a truncation rule. Applies to both CLI `initiatives list` rows and UI list view.

## Testing Plan
- CLI `initiatives get` formatter unit tests:
  - empty `depends_on` → asserted absence of any `Depends on` line
  - one dep → exact string `Depends on: foo`
  - multiple deps → exact string `Depends on: foo, bar, baz` with input order preserved
- CLI `initiatives list` formatter unit tests: row with no deps (field absent) vs row with deps (field present, comma-joined).
- UI: smoke-test detail view renders the "Depends on" section when deps exist and omits it when empty; list view renders compact form. Visual check (no snapshot churn unless project already uses snapshots here).
- Manual verification with a real initiative that has known deps (e.g., `web-console-readiness` → `continuous-audio-platform` per spec).

## Rollout / Validation Checklist
- [ ] CLI `initiatives get` human output shows `Depends on:` line when non-empty and omits it when empty.
- [ ] CLI `initiatives list` human output shows deps per row when non-empty and omits when empty.
- [ ] UI detail view shows deps section when non-empty; omits when empty.
- [ ] UI list view shows deps inline when non-empty; omits when empty.
- [ ] JSON output unchanged.
- [ ] All tests green; lint/types clean in modified files.

## Risks + Mitigations
- **Wording inconsistency between CLI and UI** — locked to "Depends on" everywhere per d2.
- **Long dep lists overflow the list-view row** — accepted per d4; common case is 0–2 deps. Revisit only if a real initiative proves it painful.
- **Display divergence vs. JSON** — drive both from the same payload field; do not re-fetch or transform.
- **Unrelated lint/type issues in the formatter file** — Greenfield rule + final cleanup step requires fixing them.

## Non-goals / Prohibited Patterns
- No data model changes.
- No API changes.
- No re-derivation of dependencies on the client; render what the API returns.
- No downstream-dependency display in this fix.
- No legacy fallback code paths.
- No truncation / "+N more" branching in list formatters.

## Definition of Done
- All four surfaces (CLI get, CLI list, UI detail, UI list) display `depends_on` when non-empty and omit it cleanly when empty (d1).
- Wording is "Depends on" everywhere (d2).
- Only upstream is shown (d3).
- Full comma-joined list with no truncation (d4).
- Tests cover empty/single/multi cases for CLI formatters.
- Greenfield: no compatibility scaffolding introduced.
- Scenario restarted and healthy.
