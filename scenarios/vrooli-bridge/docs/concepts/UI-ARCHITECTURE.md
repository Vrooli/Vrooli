# UI Architecture

## Purpose Of This Document

Describe the canonical layout of the `ui/` source tree for scenarios generated
from the `react-vite` template, and the **slot taxonomy** that lets external
tools (notably `react-component-library`'s adoption resolver) place components
without asking the user for a path.

## Source Layout

```
ui/src/
├── api/            # api-client slot — Connect-RPC wrappers
├── app/            # app-bootstrap — Providers composition and route table
├── components/     # shared-component slot — cross-cutting components
│   └── ui/         # ui-primitive slot — headless primitives (kebab-case files)
├── consts/         # consts slot — strings + selectors registries
├── features/       # feature slot — per-feature folders (one subfolder per feature)
│   └── <feature>/  # feature-component slot — components inside a feature
├── hooks/          # hook slot — reusable React hooks
├── i18n/           # i18n bootstrap
│   └── locales/    # i18n-strings slot — one JSON per locale
├── layout/         # layout-shell + layout-nav slots — AppShell, Sidebar, TopBar, BottomNav
├── lib/            # lib-util slot — framework-agnostic utilities
├── pages/          # page slot — routed pages mounted under <Outlet />
├── test-utils/     # test-util slot — render helpers, factories, a11y
└── theme/          # theme-token slot — ThemeProvider + tokens.css
```

## Slots Are A Contract

Every directory above maps to a named slot in `ui/manifest.json`. The manifest
declares the directory **and** a default path pattern (e.g.
`{dir}/{ComponentName}.tsx`), so external tools can compute the canonical
filesystem path for a new file given just the component's name and slot.

A component library that publishes `"slot": "layout-nav"` and ships
`SidebarShell` knows — without any per-scenario configuration — that the file
should land at `ui/src/layout/SidebarShell.tsx`. Override the slot's `dir` in
a scenario-level overlay if you've reorganized; the resolver will pick up the
new path automatically.

## Adoption Resolver Flow

1. Library declares the component's slot (e.g. `"slot": "layout-nav"`).
2. Resolver looks up the slot in this scenario's UI manifest (this file's JSON
   sibling).
3. Resolver substitutes path-pattern tokens (`{dir}`, `{ComponentName}`,
   `{kebab-name}`, `{camelName}`, `{feature}`, `{locale}`) and returns the path.
4. Scenarios with no manifest fall through to a heuristic (scan for the slot's
   expected dir name) and then a final fallback
   (`ui/src/components/<ComponentName>.tsx`). Both flag warnings on the
   resulting adoption record.

## Extending The Manifest

- **Add a slot.** Add an entry to `ui/manifest.json`. Keep its `dir` inside
  `ui/src/` and pick a pattern that matches your file-naming convention. The
  schema (`scenario-ui-manifest/v1`) does not enum-restrict slot names — open
  set on purpose.
- **Override a slot in a single scenario.** Drop a partial manifest at
  `.vrooli/ui-manifest.json` in the scenario root; the resolver will read it
  as an overlay over the template manifest. (Overlay support tracked in
  scenarios/react-component-library's PRD.)
- **Add a `postApply` action** (auto barrel-export, route-register,
  i18n-merge). Reserved for a future schema bump (`scenario-ui-manifest/v2`).
  Document the intent in the consuming scenario's PRD until then.

## Fleet Dashboard

> This section is the **manual-validation reference** for the Phase-5 fleet
> dashboard (OT-P1-005, requirement `BRG-P1-005`). It documents what the
> operator surfaces are and how they compose, so the behaviour can be checked by
> hand against the running UI as well as by the component tests.

### Surfaces

The dashboard is the bridge's fleet **control plane**. Three feature surfaces
compose it, each owning its own loading / error / empty states:

- **Pairing** — `ui/src/features/fleet/PairNodeForm.tsx`. One-touch onboarding:
  the owner types a node label and mints a **single-use pairing code**
  (`PairingService.IssuePairingCode`). The plaintext code + control-plane public
  key are shown **once** for out-of-band delivery to the node's bootstrap
  installer; the server only stores the hash.
- **Fleet panel** — `ui/src/features/fleet/FleetPanel.tsx`. The owner's trusted
  nodes, each row showing:
  - **Live presence** (online/offline) as a labelled dot.
  - **OS / Arch / Version / Health** as discrete labelled metadata fields. Health
    (node status) is conveyed by **three redundant channels** — a colored dot, a
    distinct icon, *and* a text label — so it never relies on color alone
    (WCAG 1.4.1).
  - **Live per-node job status** from `QueueService.ListQueue` (running / queued
    counts), polled on a short interval so dispatch is visible without a manual
    refresh.
  - An atomic **Revoke** (confirmation-gated) that severs the node server-side.
- **Run history** — `ui/src/features/runs/RunHistory.tsx`. A newest-first feed of
  durable runs (`RunsService.ListRuns`). Each row drills into the run's
  **output** (persisted/live `RunEvent`s from `GetRun`) and its **downloadable
  artifacts** (device-sync-hub refs, linked via `artifactDownloadUrl`). A
  long / in-flight run shows a **determinate progress bar**, an **ETA** (remaining
  wall-clock budget), and a **Cancel** control (`AbortRun`) — never a frozen
  spinner; there is always either progress or a terminal state.

The dashboard page (`ui/src/pages/DashboardPage.tsx`) composes all three plus
the API health card. Run history also has a dedicated route at `/runs`
(`ui/src/pages/RunsPage.tsx`), linked from the sidebar and bottom nav.

### Data Layer

Each surface reads through a typed Connect-Web client in `ui/src/api/`
(`fleet.ts`, `queue.ts`, `runs.ts`, `pairing.ts` — mirroring `nodes.ts`) and a
React Query hook in the feature's `queries.ts`. In-flight runs and the queue
overlay poll; terminal runs stop polling. The queue overlay is **best-effort** —
a queue error never blanks the fleet, because presence is the primary signal.

### Operator Flow

The end-to-end flow an operator can complete without leaving the dashboard:

1. **Pair a node.** Enter a label in the pairing form → generate a code →
   deliver it to the new node's installer. The node dials in and appears in the
   fleet panel with live presence + OS/arch/version/health.
2. **Dispatch a job.** A job dispatched to that node surfaces as **live job
   status** on its fleet row (running / queued) and as a new entry in run
   history.
3. **Watch it complete.** Open the run in run history to follow its output, see
   the progress bar advance toward a terminal status (with an ETA), cancel it if
   needed, and download any artifacts it produced once it finishes.

## Cross-References

- Schema: `.vrooli/schemas/scenario-ui-manifest.schema.json` (`$id:
  scenario-ui-manifest/v1`)
- Manifest: `ui/manifest.json`
- Slot reference: [`ui-manifest.md`](../reference/ui-manifest.md)
- Adoption resolver: `scenarios/react-component-library/api/internal/adoptions/pathresolver.go`
