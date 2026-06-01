# Problems — Security Health

Persistent register of known issues, tech debt, and deferred work
specific to **this** scenario. Future agents read this file to avoid
re-discovering the same constraint.

This file ships empty in newly generated scenarios. Append entries as
they appear.

## What belongs here

- **Known bugs** that are real but not yet worth fixing
- **Tech debt** — workarounds that need a real fix later
- **Deferred work** — features descoped from a phase, with the reason
- **Architecture drift** — code/docs/tests that no longer line up with
  the intended capability map or boundary model
- **Constraints discovered the hard way** that aren't visible from
  the code (e.g., "this resource needs warm-up before the first call;
  see commit X")

## What does NOT belong here

- **Generic template issues** — those go in
  [`../guides/troubleshooting.md`](../guides/troubleshooting.md)
- **Open feature requests** — track those in PRD operational targets
- **Code comments** — if the constraint is local to one file, a
  comment there is more discoverable
- **Test failures** — fix them, don't document them

## Entry template

Use this shape so entries are scannable. Append newest at the bottom.

```markdown
### YYYY-MM-DD — short title

**Symptom:** What goes wrong, observable from outside the system.

**Root cause:** What actually causes it (or "unknown" if not yet diagnosed).

**Workaround:** What to do today to keep moving.

**Real fix:** What needs to happen for this entry to be deleted.

**Owner:** Who should drive the fix (or "unassigned").

**Refs:** Code paths, related issues, prior commits.
```

## Entries

### 2026-06-01 — RESOLVED: Phase F UI built + `notes` example removed (Gate 7)

**Resolution:** The Posture / Dependencies / Secrets pages and the embeddable
`PostureBadge` (`@vrooliWidget`, slot INLINE) shipped under `ui/src/features/`
(`posture`, `dependencies`) with api clients `api/validation.ts` +
`api/dependencies.ts`. The `notes`/attachment reference domain was removed from
API, CLI, UI, and proto; `vrooli scenario orient` now passes
`example-domain-removed`. 140 UI tests + `tsc` + `vite build` green. Kept for
history; no further action.

### 2026-06-01 — Pre-existing react-vite scaffolding lint debt (AppShell + ThemeProvider)

**Symptom:** `pnpm lint` reports 3 errors in template files this scenario never
touched: `layout/AppShell.tsx:27` uses a literal `aria-label="Main content"`
instead of the i18n registry (`no-restricted-syntax`), and
`theme/ThemeProvider.tsx:28,58` have `typeof window === "undefined"` SSR guards
the typed-lint rule flags as always-falsy (`no-unnecessary-condition`).

**Root cause:** security-health was generated from an older react-vite snapshot
than the current pristine template (ui-health's AppShell uses `t(...)` for the
main-content label and its ThemeProvider lacks the SSR guards). This is part of
the platform-wide STANDARDS scaffolding campaign, not this scenario's work.

**Workaround:** All Phase-F code is lint-clean; only these two untouched template
files trip the audit. Treated as pre-existing debt.

**Real fix:** Route the `<main>` label through a `layout.mainContentLabel` string
key, and drop the dead SSR guards in `resolveChoice`/the media-query effect
(this is a Vite SPA — `window` is always defined), mirroring the current
react-vite template.

**Owner:** unassigned (template-wide campaign).

**Refs:** `ui/src/layout/AppShell.tsx`, `ui/src/theme/ThemeProvider.tsx`; sibling
`scenarios/ui-health/ui` for the fixed shape.

### 2026-06-01 — Residual `notes` prose in doc surfaces (heading contract coupling)

**Symptom:** The notes domain is gone from code/proto/CLI/UI, but some doc prose
still mentions it: `docs/manifest.json` `requiredHeadings` for the API and CLI
doc pages include "Notes (CRUD reference)" / "Scenario commands — `notes`", and
the corresponding `api`/`cli` README sections still describe the removed
endpoints.

**Root cause:** The docs-completeness contract (manifest required-headings) and
the doc bodies were authored against the template example. They are internally
consistent (headings present ⇒ validation passes), so removing one without the
other would *break* doc validation.

**Workaround:** Left as-is — it validates green; the content is stale, not
failing.

**Real fix:** Update `docs/manifest.json` required-headings and the matching
`docs/` (api/cli reference) bodies together, replacing the `notes` worked-example
sections with the real `validation`/`dependencies`/`reindex` surfaces.

**Owner:** unassigned.

**Refs:** `docs/manifest.json` (api/cli requiredHeadings), `docs/` API/CLI
reference pages.

### 2026-06-01 — AI/Qdrant semantic search deferred (dependency index is TEXT-only)

**Symptom:** `DependencyService.Search` serves TEXT + structured-filter queries
over the SQLite corpus. A `MODE_AI` request degrades to TEXT (`mode_used=TEXT`),
and `Status.qdrant`/`ollama` report availability but nothing is embedded.

**Root cause:** Proper semantic search needs a pre-embedded vector index
(Qdrant); embedding the whole corpus on every query (the only no-Qdrant option)
does not scale. The faithful cli-health vectorstore clone (~600 lines) was
deferred in favor of shipping the always-available TEXT + structured core, which
already answers the headline query ("which scenarios are exposed to CVE-X?")
deterministically via `--vulnerable-only`/`--name-glob`.

**Workaround:** TEXT + structured filters cover the high-value queries today.
The `qdrant`+`ollama` deps stay declared `required:false`/`try_start` (their
documented `degraded_behavior` is exactly this TEXT fallback).

**Real fix:** Clone `scenarios/cli-health/api/internal/aisearch/{embedder,
vectorstore,reconciler}.go`, retarget points to `DependencyRecord` in the
`security-health-deps` Qdrant collection, embed on reconcile, and have
`Service.Search` ANN-rank in MODE_AI. The `Service.aiProbe` seam + `ModeUsed`
plumbing are already in place for it.

**Owner:** unassigned.

**Refs:** `internal/dependencies/service.go` (aiProbe seam), source plan Phase E.

### 2026-06-01 — Template ships a critical vitest dev-dependency CVE (fleet-wide)

**Symptom:** Every react-vite scenario's `ui/pnpm-lock.yaml` pins `vitest <4.1.0`
(GHSA-5xrq-8626-4rwp, critical). security-health's own dev-dependency audit
flags it.

**Root cause:** The react-vite template's pinned vitest predates the fix.

**Workaround:** It is a **dev-only** dependency (test runner, not in the shipped
artifact), so security-health's pnpm-audit scanner downgrades it to WARNING via
the prod/dev split — it does not gate R1. security-health validates itself clean
(errors=0).

**Real fix:** Bump the react-vite template (and existing scenarios) to
vitest ≥ 4.1.0. Cross-cutting — file via `report-bug` against the template, not
fixed here.

**Owner:** unassigned (template-wide).

**Refs:** `internal/validation/scan_pnpm_audit.go` (prod/dev split).

## Architecture Drift

Use this section for deferred findings from `screaming-architecture-audit`.
Do not create a standalone architecture-audit report unless the work is
a migration handoff with a planned retirement path back into
`ARCHITECTURE.md`, `SEAMS.md`, or this file.

| Area | Drift | Maturity Impact | Real Fix |
|---|---|---|---|
| _None yet._ |  |  |  |

## Cross-references

- [`PROGRESS.md`](PROGRESS.md) — lifecycle log (forward-looking)
- [`SEAMS.md`](SEAMS.md) — boundary registry (load-bearing for tests)
- [`TESTING.md`](TESTING.md) — test patterns
- [`../guides/troubleshooting.md`](../guides/troubleshooting.md) — generic-template issues
