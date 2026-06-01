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

### 2026-06-01 — RESOLVED: partial-index correctness gap (coverage-gated AI)

**Symptom (was):** `aiSearch` fell back to TEXT only when the index was
*completely* empty. During the minutes-long backfill a *partially* populated
index returned a ranked subset and did **not** fall back, so AI mode could
silently miss records TEXT would have found — worse than either extreme.

**Resolution:** `Service` now gates `MODE_AI` on a cached **readiness** signal:
serve AI only when `index.CountPoints() / store.DistinctPackageCount() ≥
threshold` (default 0.95, `SECURITY_HEALTH_INDEX_READY_THRESHOLD`); otherwise
TEXT. Readiness is recomputed after each full `syncIndex`, at startup
(`EnsureIndex`), and lazily when the cache is older than 60s (TTL), so `Search`
adds **zero** per-query network calls on the hot path. `Status` now exposes
`indexed_vectors`/`expected_vectors`/`index_ready` (proto fields 8–10); CLI
`deps status` and the UI status strip render `building… N%` → `ready`. The old
`len(scored)==0` fallback is kept as defense-in-depth.

**Refs:** `internal/dependencies/service.go` (`indexReadyNow`/`refreshReadiness`),
`internal/dependencies/types.go` (`Status`), proto
`security-health/v1/dependencies` `StatusResponse`.

### 2026-06-01 — Tradeoff: scenario name is no longer a semantic-search signal

**Symptom:** The vector index is keyed by **package identity**
(`ecosystem|name|version`), and `packageEmbeddingText` deliberately drops the
owning scenario + source file. Semantic queries phrased around a *scenario name*
("deps in the billing app") therefore no longer match via embeddings.

**Why this is acceptable:** Scenario is a **structured-filter / TEXT** concern,
not a good semantic signal — a scenario id is an opaque token, not natural
language. Both other retrieval paths still handle it: structured search filters
by scenario, and TEXT search matches the scenario substring. Result granularity
is unchanged — a matched package fans out to **every** exposed scenario (the
blast radius a security console exists to surface). This decoupling of
*embedding* granularity from *result* granularity is what removes the 9.6×
embedding duplication (42,382 records → 4,390 distinct packages).

**Real fix (only if a real need surfaces):** add a second, scenario-keyed
collection, or fold scenario tokens back into the package text at the cost of
re-duplicating vectors. Not planned.

**Owner:** unassigned.

**Refs:** `internal/dependencies/service.go` (`packageEmbeddingText`/`aiSearch`),
`internal/dependencies/store.go` (`PackageItems`/`RecordsByPackages`).

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

### 2026-06-01 — RESOLVED: AI/Qdrant semantic search shipped

**Resolution:** `MODE_AI` now ranks by vector similarity. New package
`internal/dependencies/aisearch` (`embedder.go` + `vectorstore.go` cloned from
cli-health's proven substrate; `index.go` is the dependency-specific overlay
with `Sync`/`Query`/`Available` over neutral `Item`/`KeyScore` types).
`Service.Search` embeds the query, ANN-searches the `security-health-deps`
Qdrant collection, hydrates hits back into records from the SQLite corpus,
applies the structured filters in-memory, and preserves the vector ranking; on
any embedder/qdrant failure it falls back to deterministic TEXT (`mode_used`
reports which served). The corpus is embedded after each authoritative
`store.Apply` (skip-unchanged via `payload_hash`); SQLite stays the source of
truth. **Proven live** against real Ollama + Qdrant
(`SECURITY_HEALTH_LIVE_AISEARCH=1 go test ./internal/dependencies/aisearch/
-run TestLive`): a semantic query with zero literal name overlap ranked the
vulnerable `golang.org/x/net` record first (score 0.758). Unit-proven:
sync/skip/delete, query mapping+ranking, filter honoring, TEXT fallback,
scoped-skip. Kept for history.

**Note (one-time backfill cost):** the fleet corpus is large (a full reconcile
reported ~43,871 records). The first reconcile after a *fresh* Qdrant
collection embeds every record sequentially through Ollama — a multi-minute
one-time backfill that runs in the background and never blocks search (search
degrades to TEXT until points exist). It is resumable (restart re-scrolls
Qdrant and skips already-embedded points) and persists across restarts, so
later reconciles embed only changed records. The pre-`Apply` discovery +
osv-scanner annotation (once per scenario, ~60 invocations) is itself the
larger fixed cost, unchanged by this work. If backfill latency ever matters
operationally, add bounded-parallel embedding (cli-health uses `errgroup` at
`ReconcileParallelism`) — the `Sync` loop is the single seam.

**Refs:** `internal/dependencies/aisearch/`, `internal/dependencies/service.go`
(`aiSearch`/`syncIndex`).

### 2026-06-01 — Scoped reindex defers index refresh to the next full reconcile

**Symptom:** `reindex run <scenario>` updates that scenario's rows in the SQLite
corpus but does **not** update the Qdrant vector index for it; the semantic
index only catches up on the next full (fleet-wide) reconcile.

**Root cause:** `aisearch.Indexer.Sync` treats the entire collection as the
universe (it deletes every point whose key is absent from the synced set, by
design, so the index mirrors the corpus). Feeding it a single scenario's records
would delete every *other* scenario's vectors. `Service.syncIndex` therefore
no-ops on a scoped reindex (`scenario != ""`) and only does the full sweep on
`RunReconcileOnce`/full reindex.

**Workaround:** None needed for correctness — SQLite (the source of truth) is
updated immediately; only the AI ranking lags by ≤1 reconcile cycle (≤5 min).
TEXT/structured search reflects the scoped reindex instantly.

**Real fix (optional):** Make `Sync` scope-aware — carry `scenario` in the point
payload, project it via `ScrollIDs`, and restrict the stale-delete set to the
synced scenario.

**Owner:** unassigned.

**Refs:** `internal/dependencies/service.go` (`syncIndex`),
`internal/dependencies/aisearch/index.go` (`Sync`).

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
