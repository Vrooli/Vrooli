# Plan: Center graph informational overlays and add "Show anyway" override for edge warning

> **This is greenfield work.** Do not include compatibility shims, legacy wrappers, dead code, unused re-exports, `// removed` comments, or renamed `_unused` variables.

## Required Reading

```bash
prompt-manager skill read swarm-manager-backlog-tools implementation-plan-authoring ux react-coherence
```

## Problem & Goal

The graph view (`surfaces/graph/components/GraphCanvas.tsx`) renders multiple informational/status overlays at inconsistent positions:

| Overlay | Current position | Test ID |
|---|---|---|
| Refreshing indicator | `absolute inset-x-0 top-14 mx-auto w-fit` (top-center, below nav) | `graph-loading` |
| Error banner | `absolute left-1/2 top-14 -translate-x-1/2` (top-center) | `graph-error` |
| High edge count warning | `absolute left-1/2 top-24 -translate-x-1/2` (top-center, lower) | `filter-suggestion` |
| Agent-manager unavailable warning | `absolute bottom-3 left-3` (bottom-left) | `operations-agent-manager-warning` |
| "No nodes match" empty state | `absolute left-1/2 top-1/2 -translate-x-1/2 -translate-y-1/2` (true center) | `graph-empty` |
| `FocusEmptyState` (focus lens empty) | `absolute left-1/2 top-1/2 -translate-x-1/2 -translate-y-1/2` (true center) | (FocusEmptyState) |

The empty-state and `FocusEmptyState` already use the desired centered pattern. The other status overlays are scattered across the top edge or a corner. Goal: bring all informational/status overlays into the same true-center pattern as `FocusEmptyState`.

Additionally, the high edge count warning currently has only a dismiss `×` — the graph already renders regardless. The requested change is to add an explicit **"Show anyway"** affordance that lets users acknowledge and bypass the performance warning. Whether the graph is gated (hidden until acknowledged) or whether "Show anyway" simply replaces/augments the current dismiss is an open decision (see Decision Log).

## Non-Goals

- Changing edge-rendering performance behavior (no new throttling, virtualization, or LOD work).
- Changing the 500-edge `FILTER_SUGGESTION_THRESHOLD` value.
- Recentering true UI controls (mini-map, edge legend, graph controls) — those are *interactive UI*, not informational overlays.
- Touching `FocusEmptyState` itself (it already defines the pattern).

## Scope (acceptance_allow)

- `scenarios/swarm-manager/ui/src/surfaces/graph/**`

Anticipated edited files:

- `scenarios/swarm-manager/ui/src/surfaces/graph/components/GraphCanvas.tsx` — recenter overlays; add "Show anyway" UI + state
- Possibly a small extracted component `components/GraphOverlayCard.tsx` (decision below) for the shared centered shell
- Tests under `scenarios/swarm-manager/ui/src/surfaces/graph/**/__tests__/` or `*.test.tsx`

## Technical Context

- React + Tailwind + React Flow (read-only graph). Overlays sit in the same containing `<div className="h-full w-full" data-testid="graph-canvas">` as the `<ReactFlow>` element, so `absolute` overlays are positioned relative to the graph area.
- `FocusEmptyState` shell pattern: `absolute left-1/2 top-1/2 z-10 -translate-x-1/2 -translate-y-1/2 rounded-2xl border ... bg-slate-950/95 px-6 py-5 shadow-xl text-center`.
- `showFilterSuggestion = overFilterThreshold && !filterSuggestionDismissed` (lines 347-358).
- Auto-reset effect (lines 352-356) clears dismissal when edge count drops below threshold so the banner can re-trigger on a future spike. This pattern needs reconsidering once "Show anyway" is introduced (see decisions).

## Approach (sketch)

1. **Extract a shared `GraphOverlayCard`** (or inline equivalent) using the FocusEmptyState centering shell. Keep the existing color tokens per overlay (red for error, amber for warning, slate for neutral).
2. **Recenter the four scattered overlays** (`graph-loading`, `graph-error`, `filter-suggestion`, `operations-agent-manager-warning`) onto the true-center pattern. Resolve stacking when multiple are visible (decision below).
3. **Add "Show anyway"** as a primary action button on the high-edge warning, alongside (or replacing) the dismiss `×`. Behavior on click depends on the gating decision below.
4. **Persistence**: decide whether "Show anyway" sticks for the session, sticks until edge count drops below threshold, or stays acknowledged forever (decision below).
5. **Tests**: extend existing `GraphCanvas.test.tsx` (or sibling tests) — render with edges over threshold and assert the overlay is centered (positional class assertions) and that "Show anyway" toggles rendering as decided.
6. **Cleanup & verification** (mandatory): fix all lint, type, and test issues in modified files (even pre-existing); restart scenario; verify health.

## Decision Log

(To be populated by workshop rounds. See round-001 for open decisions.)

## Risks & Unknowns

- **Stacking when multiple overlays are simultaneously visible** (e.g., refreshing + high-edge + agent-manager-unavailable). Centering all of them on top of each other will overlap. Need a stacking strategy (queue, vertical stack inside one centered container, priority).
- **Gating render behind "Show anyway" changes default UX** — previously every user saw the graph; if we gate, large-graph users now click an extra button on every load. This is the most behavior-shifting decision.
- **Persistence of acknowledgement** interacts with the auto-reset effect (lines 352-356). Need to clarify intent so reload/refresh behavior is consistent.
- **Center-of-graph vs center-of-viewport**: the canvas div fills the available area, but if the graph header/toolbar sits inside the canvas div, "true center" may be visually off. Verify against `FocusEmptyState`'s actual rendered position before assuming parity.

## Test Plan

<!-- TBD pending decisions -->

## Cleanup & Verification

- Run type checking (`npx tsc --noEmit` from the UI package) and fix ALL errors, even pre-existing ones in modified files.
- Run linter (`eslint`) and fix ALL warnings in modified files.
- Run unit tests covering `GraphCanvas` overlays and the new "Show anyway" behavior.
- `vrooli scenario restart swarm-manager` (or the current restart command for this scenario).
- Verify health: load the swarm-manager UI, force a high-edge graph, confirm the warning is centered and "Show anyway" works as decided.

## Initiative / Cross-Item Notes

This item has no `initiative`. No sibling overlap detected during round-001 search. Tags `swarm-manager, graph, ux, topo-view, edge-warning` suggest this lives next to other topo-view UX work — orchestrator should batch with other graph-overlay tweaks if any are queued.
