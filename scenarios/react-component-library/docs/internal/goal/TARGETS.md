# Rehabilitation contract — selected core

Status: **active contract.** This document is the canonical source for the
selected-core rehabilitation obligations referenced by `DESIGN.md`, `PRD.md`
and `requirements/25-rehabilitation`. It records obligations, not claims: a
requirement is satisfied only when current executable or human evidence is
attached to it.

The five-destination workbench target that supersedes the historical eleven-page
map is defined in [WORKBENCH-MATURATION.md](../../concepts/WORKBENCH-MATURATION.md)
and delivered by [WORKBENCH-PROGRAM.md](../WORKBENCH-PROGRAM.md).

## 1. Scope

The rehabilitation certifies a **selected core**, not the full desired catalog.
The core is the small set of components, patterns and hooks that a real scenario
must be able to adopt, compose, upgrade and verify. Assets outside the selected
core remain catalogued but are excluded from this certification.

In scope:

- Reuse-first discovery: a person can find an existing asset, page type,
  template or workspace before creating another.
- Selected-core certification: exact version, dependency closure, public
  contract and meaningful states proved in a real built consumer.
- A complete guided studio: workspace → page → display → state authoring,
  preview, capture and evidence.
- Template reuse: curated workspaces are immutable and fork on use.
- Shared consumption: consumers adopt exact releases through supported paths.
- Safe upgrades: drift, provenance and compatibility are visible and tested.
- Cleanup: duplicate stores, dead surfaces and unbounded debt are removed.

Out of scope for this contract: full-catalog launch certification, multi-user
presence/comments, native mobile applications, freeform in-context drag beyond
the supported placement model, and external asset imports.

## 2. Five proving journeys

Every release candidate must pass these journeys end to end. A journey passes
only with a recorded artifact: a test, a capture, or a signed human review.

### J1 — Find before you build

A maintainer searches an outcome such as “landing page” or “onboarding flow”,
sees the page type, curated workspaces and composed assets in one ranked result
set, and opens the correct object without visiting a second search surface.

Evidence: golden-query test, ranked result capture, no-match and index-empty
states.

### J2 — Compose a page and prove its states

A maintainer creates or forks a workspace, adds a page, adds desktop/tablet/phone
displays and named states, and captures a state with its args and scenario data.

Evidence: workspace aggregate test, state-address test
(`workspace/page/display/state`), capture artifact, phone composition.

### J3 — Reuse a curated template without forking the original

A maintainer opens a curated workspace, forks it, edits the fork, and the curated
original remains byte-identical with provenance `curated`.

Evidence: fork test, immutability test, provenance test.

### J4 — Adopt and upgrade in a real consumer

A consumer adopts a selected-core asset at an exact version, builds, and can see
the release, dependency closure and drift state. Upgrading to the next compatible
release is a supported operation with a rollback path.

Evidence: real build transcript, adoption/refresh test, upgrade and rollback
drill.

### J5 — Review and recover from truth

A reviewer opens Review, sees server-owned findings with evidence and durable
dispositions, opens Stats, and distinguishes measured zero from unavailable in
Quality, Activity and Services.

Evidence: findings projection test, disposition write, Stats section tests,
capture of each unavailable/empty/partial state.

## 3. Selected-core certification

A selected-core asset is certified only when all of the following hold at one
exact version:

1. The manifest identifies `libraryId`, exact version, entry, dependencies and
   deprecations.
2. The dependency closure resolves and respects composition rank.
3. The public contract (props, variants, slots, events, tokens) is generated or
   checked, not hand-maintained.
4. Meaningful states (default, loading, empty, error, disabled, permission,
   narrow viewport) are declared and exercised.
5. A real built consumer imports the exact release and renders the states.
6. Accessibility: axe zero, keyboard reachable, visible focus, no horizontal
   overflow at 390 and 320.
7. Provenance: the release has a publish or backfill record; a bypass release is
   removed rather than certified.
8. Evidence is attached at the current revision; historical receipts do not
   satisfy a new revision.

The certified set and its evidence live in the scenario's gate evidence store and
are surfaced by Review and Stats. The catalog is not “complete” because it is
large; it is healthy when the selected core is proved and the rest is honestly
labelled.

## 4. Guided studio obligations

- One composition owner: the RCL database owns workspace, page, display, state,
  theme, assets and revisions. Scenario `experience/pages/*.json` owns intent and
  claims only.
- A workspace contains one or more pages. A page contains one or more displays.
  A display contains one or more named states.
- A state is addressable as `workspace/page/display/state` and can carry args,
  scenario data, capture status and evidence.
- A curated template is a workspace snapshot with `provenance=curated`. It forks
  on use; editing a fork never mutates the curated original.
- No silent local fallback. Authoring is either durable through the API or
  explicitly read-only/offline with a visible unsaved state and a promotable
  draft queue.
- The studio shows loading, empty, partial, stale, error and permission states.

## 5. Shared consumption and safe upgrades

- Consumers adopt exact releases or documented major aliases; an alias is never
  presented as proof of a historical build.
- Drift is detected from source and reported with a recovery path.
- A selected-core upgrade has a compatibility note and a rollback path.
- Duplicate component copies are not an accepted consumption strategy; the
  adoption ledger is the single record of who uses what.
- Tokens have one owner; consumer overrides are explicit and bounded.

## 6. Deletion and budget gates

The rehabilitation is not complete while duplicate owners remain. The following
must be deleted, not hidden. Legacy design and template owners were retired on
2026-11-20: the migration imports rows and uploads, writes receipts, validates
binding uniqueness, and drops old tables transactionally. Their API modules,
repositories, and code-defined curated shelf are gone; the neutral blank-page
starter is not a curated template.

- `internal/templates`, `/api/templates`, `studio_templates`, `internal/designs`,
  `/api/designs`, `studio_designs`, and `studio_design_assets` — **closed**;
  retain only the one-release local-cache bookmark reader for unsynced drafts.
- Sketch composition storage after the intent/composition split.
- Catalog-only search once one multi-source index serves Start and ⌘K.
- Coverage, Capabilities and Activity as separate routes once Stats owns their
  views.
- UI-side findings and activity reconstruction once the server owns them.
- Duplicate verdict renderers, device metric tables and preview chrome.
- Dead CSS, dead exports and unreferenced legacy page components.
- Inline production DDL.
- Write-path formatting in `make check`.

Budgets (reviewed, not silent):

- No page component over 400 lines; no hook over 350; no CSS file over 1,500
  unless an exception is recorded in `docs/studio/DEBT.md`.
- No service over 16 RPCs without an ADR.
- No new localStorage source of truth.
- Every selected-core requirement has evidence at the current revision.

## 7. Evidence admission

A claim of completion must include:

- the exact command or capture that proves it;
- the revision or timestamp it applies to;
- the resulting counts, not just “passed”.

`docs/internal/PROGRESS.md` records dated evidence. `docs/internal/PROBLEMS.md`
records open defects. `docs/studio/DEBT.md` records known debt. None of them may
be treated as current merely because it exists.

## 8. Program mapping

| Contract area | Program phase | Primary evidence |
|---|---|---|
| Truth and guardrails | 0 | gate baseline, resolved references |
| Contracts and schema | 1 | proto parity, migration dry-run |
| Workspace core and studio | 2 | J2, J3 aggregate and fork tests |
| Unified search | 3 | J1 golden queries, p95, degradation |
| Start, Library, Stats | 4 | relocation ledger, five routes, phone |
| Review and Stats | 5 | J5 findings, dispositions, sections |
| Cleanup and certification | 6 | deletion ledger, net-negative report |
