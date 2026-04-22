# Widget Standard — Meta-Orchestrator Summary

## Source

Brainstorming session (2026-04-22). The user proposed `widgets` as a counterpart to tools on the UI side: relevant scenario UI sections rendered inline in an agent-inbox chat, automatically surfaced when the conversation implies them. Example: user says "the audio isn't recognizing my voice" → inbox renders the audio-settings widget from the relevant scenario inline so the user can adjust it without leaving the chat. Made possible because every scenario UI is already required to adopt iframe-bridge.

Sibling initiatives: `tool-authoring-standard`, `cli-conversational-surface`, `agent-inbox-unified-retrieval`.

## Shared Decisions (apply across all four sibling initiatives)

1. **Proto-first.** A new `widget.proto` lives alongside `tool.proto`.
2. **Manifest-free.** No `widgets: []` in `service.json`; the UI source-of-truth file and iframe-bridge runtime are authoritative.
3. **Fewer packages.** Extend whatever the shared UI / iframe-bridge package is rather than creating a new one.
4. **Auditor comparison, not manifest declaration.**
5. **Static embedding extraction** is the default. Runtime extraction via iframe-bridge is a future supplement if static proves insufficient.

## Scope of This Initiative

The UI-side authoring standard + iframe-bridge extension that lets agent-inbox request and render scoped scenario UI sections.

### Granularity — coarse, section-level

v1 rule of thumb: a widget is the smallest thing a user would plausibly want surfaced inline in chat. Concretely:

- Settings sections ("execution settings", "audio settings")
- CRUD forms for entity types (create/update/delete for main entities)
- Entity detail pages (the whole page when the page's purpose is showing one entity — e.g., a backlog item detail)
- Dashboard sections (one chart, one panel)
- Main sections of a page

Splitting finer (single-control granularity) is explicitly deferred. Going coarse-to-fine later is easy; the reverse is hard.

### Authoring pattern — single source of truth + data attribute

- `ui/src/widgets.ts` declares every widget: `id`, `description`, `category`, `tags`. This is the source of truth.
- React code imports from it and applies the ID as a data attribute:
  ```tsx
  <section data-widget={WIDGETS.EXECUTION_SETTINGS.id}>...</section>
  ```
- Iframe-bridge at runtime uses the `data-widget` attribute to scope the render when asked for widget X.
- **Drift prevention** (the thing the user specifically wanted to get right): scenario-auditor parses `widgets.ts`, greps source for `data-widget=` usages, and checks iframe-bridge's runtime widget manifest. All three must agree; any mismatch fails.
- Reuses/extends the BAS e2e selector pattern where feasible. Research will study BAS's selector file layout and decide whether widgets can merge with it, extend it, or should stay parallel.

### Embedding extraction — static, build-time

At scan time, parse the React source AST; for each `data-widget="X"` element, collect text content underneath (JSX text children, hardcoded labels, form field names, section headings). Combined with the declared `description` + `tags`, that text is the embedding corpus for retrieval.

Chosen over runtime extraction because static is deterministic, doesn't require the scenario to be running, and audits cleanly. Runtime extraction is left open as a future supplement only if static misses meaningful signal (e.g., widgets whose labels are computed from i18n at runtime).

### Iframe-bridge extensions — in scope

- **Widget-manifest handshake**: on iframe attach, the scenario UI advertises its widget manifest (IDs, descriptions, categories) to the parent (agent-inbox).
- **Scoped-render directive**: parent can request `render widget X`. The scenario UI responds by rendering only the subtree marked with `data-widget="X"` (and whatever CSS/layout it needs to display coherently). Options for isolation include CSS scoping, React portals, or a dedicated `/widget/:id` route — research picks the cleanest.

### What we're NOT building

- The retrieval/auto-surfacing logic in agent-inbox — that's `agent-inbox-unified-retrieval`.
- Per-control or fine-grained widgets.
- Dynamic runtime extraction (deferred).
- A widget framework that tries to abstract over different UI frameworks. React is the only UI framework in use today.

## Anticipated Items

- `research/widget-authoring-pattern-design` — formalize the `widgets.ts` shape, data-attribute convention, static-extraction algorithm, iframe-bridge handshake + scoped-render mechanism, BAS selector reconciliation decision.
- `execute/widget-proto-schema` — add `widget.proto` to `packages/proto/schemas/agent-inbox/v1/domain/`.
- `execute/iframe-bridge-widget-handshake` — extend iframe-bridge with manifest handshake + scoped-render directive.
- `execute/widget-authoring-ui-package` — shared UI helpers for declaring widgets + applying data attributes type-safely.
- `execute/widget-static-extraction-indexer` — AST walker that produces the embedding corpus from `data-widget` boundaries.
- `execute/widget-authoring-auditor-rule` — cross-check source / `widgets.ts` / runtime manifest.
- `execute/widget-authoring-prompt-manager-skill` — `widget-authoring` skill.
- `execute/widget-authoring-template-scaffolding` — react-vite template ships with an example widget.

## Cross-Initiative Dependencies

- **Consumed by** `agent-inbox-unified-retrieval` — widget descriptors are one of three surface types in the unified index; inbox side also needs the client for the scoped-render protocol.
- **Parallel to** `tool-authoring-standard` and `cli-conversational-surface`.
- **Touches** `iframe-bridge` (cross-cutting UI shared package) — the widget handshake is an iframe-bridge feature, not an agent-inbox feature. This is worth calling out during rollout: iframe-bridge changes propagate to every scenario UI on upgrade.

## Open Questions Deferred to Workshop / Research

- **Scoped-render mechanism**: CSS scoping, React portal, dedicated `/widget/:id` route, or something else. Research picks.
- **BAS selector reconciliation**: merge, extend, or keep parallel with BAS's e2e selectors. Research decides after reading BAS's current pattern.
- **Composition**: can a page contain another widget's subtree? If yes, does rendering widget A also expose widget B inside? v1 probably says no — each widget renders stand-alone.
- **Multi-instance widgets** (e.g., a CRUD form rendered for *this* entity ID) — how does the scoped-render directive parameterize? Probably: widget ID + arbitrary props. Research shapes this.
- **Auth/state when rendered inline**: the scenario UI inside the iframe already carries its own auth context via iframe-bridge; no new work expected, but validate early.
