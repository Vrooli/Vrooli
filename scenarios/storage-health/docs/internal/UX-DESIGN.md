# Storage Health — UX Design Spec (Phase 9)

This is the reviewed interaction + layout spec the UI build (Phase 10) implements.
It covers every standalone feature plus the per-scenario validation result view,
with no orphan flows. Built on the react-vite template + `vrooli-default` design
(AppShell with Sidebar/BottomNav, theme + i18n + spatial-nav already wired).

## 1. Audience & job-to-be-done

storage-health has two audiences:

- **The fleet operator** ("is any scenario about to run destructive E2E against
  its real database?"). Their home is the **Dashboard** and the **Fleet
  Inventory** isolation scorecard.
- **The scenario author** ("why did my storage phase fail, and how do I fix
  it?"). Their home is the **Validate** view — findings + remediation + autofix.

The throughline is **safety**: test-isolation readiness is the most prominent
signal on every screen.

## 2. Information architecture (navigation)

Five primary destinations (Sidebar + BottomNav), replacing the template's
Dashboard/Settings pair:

| Key        | Path          | Purpose                                                        |
|------------|---------------|---------------------------------------------------------------|
| `dashboard`| `/`           | Fleet-wide storage posture at a glance; safety first.         |
| `fleet`    | `/fleet`      | Full storage inventory + isolation scorecard; filter by view. |
| `validate` | `/validate`   | Validate one scenario: findings, remediation, autofix.        |
| `advisor`  | `/advisor`    | Migration intelligence + Postgres→SQLite candidates.          |
| `settings` | `/settings`   | Theme + locale (template-provided).                           |

Deep links: `/validate?scenario=<id>` pre-fills and runs; `/fleet?view=isolation`
opens the scorecard filtered to offenders. Dashboard cards link into these.

## 3. Screens

### 3.1 Dashboard (`/`)
A safety-first overview backed by `FleetService.GetInventory` (fast, persisted
snapshot; a "Scan now" action triggers `ScanFleet`).

- **Hero stat band**: `scenario_count`, **`isolation_unready_count`** (the
  headline safety number, rendered in the alert color when > 0), `no_backup_count`,
  `finding_count`. Each stat is a link into the matching `/fleet` view.
- **Isolation scorecard preview**: top N isolation-unready scenarios with their
  reason; "View all" → `/fleet?view=isolation`.
- **Engine distribution**: compact bar list (sqlite/postgres/qdrant/redis/file).
- **Snapshot freshness**: `scanned_at` relative time + "Scan now" button.
- Empty state (no snapshot yet): a single call-to-action card → "Scan the fleet".

### 3.2 Fleet Inventory (`/fleet`)
Backed by `ScanFleet` (with a "use last snapshot" toggle → `GetInventory`).

- **View switcher** (segmented control): `all · isolation · no-backup · engines · stages`.
  Mirrors the CLI `--view` exactly so docs/CLI/UI agree.
- **`all`**: a table — scenario · engines · stage · isolation badge · findings ·
  backup badge. Sortable by isolation-readiness first (offenders on top).
- **`isolation`**: the **isolation scorecard** — only unready scenarios, each with
  its `isolation_reason` and a "How to fix" link to the routed-seams remediation.
  This is the safety artifact; it reads as a checklist of real-data risks.
- **`no-backup`**: data-persisting scenarios with no declared backup target.
- **`engines` / `stages`**: distribution bar lists.
- Row click → `/validate?scenario=<id>`.

### 3.3 Validate (`/validate`)
Backed by `ScenarioValidationService.ValidateScenario`, and
`PreviewFix`/`ApplyFix` for autofix.

- **Scenario picker** (text input + run button; honors `?scenario=`).
- **Result header**: status pill (passed/failed/degraded/error/skipped) +
  error/warning/info counts + detected engines/stage/language chips.
- **Findings list**, severity-sorted (errors first). Each finding card:
  severity badge · code · title · location · **remediation** (always shown for
  errors) · an **Autofix** affordance when `autofix_available`.
- **Autofix flow**: "Preview fixes" → shows candidate diffs (read-only) →
  "Apply" (guarded confirm; write effect) → re-runs validation to show the
  cleared findings. Idempotent — a second apply is a no-op with a clear message.
- Empty/clean state: a reassuring "No storage findings" panel with the engines
  and isolation status still shown (so a green result is informative, not blank).

### 3.4 Migration Advisor (`/advisor`)
Two tabs over `AdvisorService`:

- **Engine fitness** (`AdviseEngines`): ranked Postgres→SQLite candidates as
  cards — scenario · current→recommended · **fitness meter (0–1)** · rationale ·
  blockers (when present). Sorted strongest-first.
- **Migration hygiene** (`AnalyzeMigrations`): scenarios carrying migration debt,
  each with stage-aware notes; a summary line of `with_migrations` / `debt`.
- Empty states per tab ("No migration candidates" / "No migration debt").

## 4. Cross-cutting states

- **Loading**: skeleton rows for tables/cards; the action button shows a spinner
  and disables. A fleet scan can take a couple of minutes for the whole fleet —
  the button copy says "Scanning… this may take a minute" and a subset can be
  scanned first.
- **Error**: an inline alert with the server message + a retry button; never a
  blank screen. RPC errors are surfaced via the shared error-message helper.
- **Empty**: every list has a purpose-built empty state with the next action,
  never a bare "no data".
- **Offline/degraded**: GetInventory returning `scenario_count = 0` renders the
  "no snapshot yet" CTA rather than an error.

## 5. Visual language

- Reuse `vrooli-default` design tokens; no new palette. Two semantic accents:
  **alert** (isolation-unready / error findings) and **ok** (isolation-ready /
  clean). Warnings use the neutral-emphasis token; info is muted.
- Isolation status is a **badge** with an icon + text (never color alone — a11y):
  `🔓 ISOLATION-UNREADY` vs `🔒 isolation-ready`.
- Fitness score renders as a labeled meter (`0.80`), not color alone.

## 6. Accessibility & i18n

- All interactive elements keyboard-reachable; the existing spatial-nav +
  gamepad hooks cover the grid/table navigation.
- Every status conveyed by icon+text, not color alone (axe clean; landmark +
  heading order preserved from the template AppShell).
- All copy flows through the i18n string catalog (`consts/strings` + locale
  JSON), including the new storage vocabulary; no hard-coded user-facing strings.
- Test IDs via the `selectors` registry so tests bind to stable ids, not labels.

## 7. Build order (Phase 10)

1. API clients (`api/fleet.ts`, `api/advisor.ts`, `api/validation.ts`) over the
   shared proto-JSON client.
2. Nav + routes + selectors + strings for the four new destinations.
3. Validate view (highest author value), then Fleet inventory + scorecard, then
   Dashboard (composes the others), then Advisor.
4. Per-screen tests + a11y tests; keep `pnpm test:coverage` ≥ gate.
