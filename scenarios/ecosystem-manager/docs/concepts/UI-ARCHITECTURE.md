# UI Architecture

## Purpose Of This Document

Describe the actual layout of the ecosystem-manager `ui/` source tree and
state — honestly — its relationship to the **slot / adoption-resolver UI
manifest** introduced by the react-vite **v2** template.

The short version: ecosystem-manager **predates** the slot manifest system.
It has **not** adopted `ui/manifest.json`, slots, or the adoption resolver.
The sections below describe the tree as it really is, then explain what
adopting the slot taxonomy would entail.

## Source Layout

React + Vite. The dashboard is a Trello/kanban-style board. The UI is served
on the documented dashboard port `30500` (the `UI_PORT` in
[CODE: `.vrooli/service.json`] is `21110`; `30500` is the external/proxy
dashboard address used throughout the operations docs).

```
ui/src/components/
├── controls/        # FloatingControls, NewTaskButton, ProcessMonitor,
│                    #   ProcessorStatusButton, SettingsButton, LogsButton, …
├── kanban/          # KanbanBoard, KanbanColumn, TaskCard(+Header/Body/Footer),
│                    #   TaskBadges, PriorityIndicator, ElapsedTimer, Skeleton…
├── steer/           # SteeringConfigPicker, SteeringConfigDialog,
│   │                #   PhasePicker, PhasePickerDialog, SteerFocusBadge
│   └── panels/      #   ProfilePanel, QueuePanel, ManualPanel, NonePanel
├── executions/      # ExecutionDetailCard, ExecutionFeedbackPanel
├── insights/        # InsightsTab, SystemInsightsTab, *Card, *Viewer (analytics)
├── filters/         # FilterPanel
├── modals/          # CreateTaskModal, TaskDetailsModal, SettingsModal,
│                    #   SystemLogsModal, AutoSteerProfileEditorModal, …
├── markdown/        # MarkdownRenderer + components/hooks/utils
├── shared/          # cross-cutting components
├── ui/              # local primitives
└── ErrorBoundary.tsx
```

The tree is organized by **feature/concern folders chosen by hand**
(`kanban/`, `steer/`, `executions/`, `insights/`, …), not by a manifest-driven
slot taxonomy. The `steer/` tree mirrors the Auto Steer model: a top-level
config picker, a phase picker, and one panel per steering mode
(Profile / Queue / Manual / None).

## Slots Are A Contract

In react-vite **v2**, every UI directory maps to a named slot in
`ui/manifest.json`, and each slot declares a default path pattern (e.g.
`{dir}/{ComponentName}.tsx`). That lets external tooling — notably
`react-component-library`'s adoption resolver — compute the canonical
filesystem path for a new component from just its name and slot, with no
per-scenario configuration.

**Ecosystem-manager does not have a `ui/manifest.json` and declares no
slots.** There is no slot contract here today. The folder names above carry
*convention*, not a machine-readable contract, so external tooling cannot
resolve a path into this tree automatically — it would fall through to the
resolver's heuristic/fallback (and flag warnings) rather than hit a declared
slot.

## Adoption Resolver Flow

For a scenario that *has* adopted the manifest, the resolver:

1. Reads the component's declared slot (e.g. `"slot": "layout-nav"`).
2. Looks that slot up in the scenario's `ui/manifest.json`.
3. Substitutes path-pattern tokens (`{dir}`, `{ComponentName}`, …) to
   produce the target path.
4. Falls back to a directory-name heuristic and finally
   `ui/src/components/<ComponentName>.tsx` when no manifest exists —
   flagging a warning on the adoption record.

**Against ecosystem-manager, only step 4 applies.** With no manifest, every
adoption lands via heuristic/fallback and is flagged. That is the current,
accurate behavior — not a bug, just the consequence of predating the system.

## Extending The Manifest

Ecosystem-manager has no manifest to extend yet. **Adopting** the slot
taxonomy here would mean:

- **Author `ui/manifest.json`** against schema `scenario-ui-manifest/v1`,
  mapping the existing folders to slots — e.g. `components/` →
  `shared-component`, `components/ui/` → `ui-primitive`, `modals/` to a
  modal slot, and the feature folders (`kanban/`, `steer/`, `executions/`,
  `insights/`) to feature slots. Because the schema does not enum-restrict
  slot names, the feature folders can keep their current names.
- **Reconcile the layout** where it diverges from template defaults (this
  tree groups by feature under `components/`; the template default expects a
  top-level `features/` tree). Either align directories or set each slot's
  `dir` to the existing location so the resolver targets the real path.
- **Validate** that resolver-computed paths match the on-disk files, then
  remove the heuristic-fallback warnings that adoptions currently produce.

Until that work is done, treat this scenario as **manifest-unaware**: place
components by the folder conventions above, not by slot.

## Cross-References

- Template slot/adoption concept: [`scenarios/react-component-library`](../../../react-component-library) — `api/internal/adoptions/pathresolver.go`
- Template UI-architecture concept (slot taxonomy): the react-vite v2 `docs/concepts/UI-ARCHITECTURE.md`
- Schema (when adopted): `.vrooli/schemas/scenario-ui-manifest.schema.json` (`$id: scenario-ui-manifest/v1`)
- [`ARCHITECTURE.md`](ARCHITECTURE.md) — system boundaries
- [`DOMAINS.md`](DOMAINS.md) — domain ownership reflected in the `steer/`/`insights/`/`executions/` UI grouping
