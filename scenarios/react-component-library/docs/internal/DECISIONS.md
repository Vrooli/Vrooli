# Decisions — React Component Library

This document records durable decisions and tradeoffs future agents
should not accidentally relitigate.

## Purpose Of This Document

Use this document when a choice:

- affects multiple files or future agents,
- rejects a plausible alternative,
- changes architecture, deployment, data, security, monetization, or
  testing direction,
- needs a revisit trigger.

Routine implementation notes belong in [`PROGRESS.md`](PROGRESS.md).
Known unresolved issues belong in [`PROBLEMS.md`](PROBLEMS.md).

## Decision Log

| Date | Decision | Context | Consequences | Revisit Trigger |
|---|---|---|---|---|
| 2026-05-12 | Use the generated `react-vite` scenario documentation contract. | Scenario scaffold was generated from the template. | Docs start with stubs and maturity metadata in `docs/manifest.json`. | Revisit when scenario adopts a different template or doc contract. |
| 2026-07-08 | Treat SQLite as an additive projection of Git-tracked component files. | Slot, headers, dependency declarations, preview import maps, and style affinities all derive from `component.json` and version source headers. | Schema changes use one-shot migrations for existing dev DBs; re-index rebuilds projection rows without recreating the database. | Revisit only if the component library stops using Git-tracked files as source of truth. |
| 2026-07-08 | Use `templates/design/*/metadata.json` IDs as the canonical design-style vocabulary. | Component affinity declarations need a stable key that scenario generation and UX validation can share. | `component.json designStyles[]` is reconciled against the registry during index, stale IDs are reported as non-fatal findings, `components styles` exposes the registry, and search/UI/adoption workflows treat affinities as advisory signals. | Revisit when design styles move out of templates or become versioned cross-scenario resources. |
| 2026-07-08 | Keep design-style affinities component-scoped. | Existing components do not have version-specific style intent, while dependency declarations are version-scoped because adopters can pin old code with different package needs. | `component_design_affinities` remains keyed by `(component_id, style_id)` and can carry a rationale; version-specific fit must be introduced deliberately if component versions diverge visually. | Revisit when two versions of the same component intentionally target different design styles. |
| 2026-07-08 | Store dependency declarations by component version and kind. | Adopters can pin older component versions, and peer dependencies have different compatibility semantics than runtime/dev dependencies. | `component_dep_declarations` is keyed by `(component_id, version, dep_name)` and carries `kind`; adoption validation resolves against the requested adopted version. | Revisit if component artifacts gain a package-lock-style dependency manifest. |
| 2026-07-14 | TabBar is withdrawn from the harvest loop; DrawerShell is the sole harvest exemplar. | The predecessor harvest plan (`rcl-harvest-loop-completion-pilot-closeout-origin-parity`) intended to harvest both DrawerShell and TabBar as multi-file units and re-adopt them in web-console. During execution the operator determined TabBar carries too much scenario-specific logic (web-console-coupled routing/state) to become a governed shared primitive without distorting either side. DrawerShell completed the full harvest→canonicalize→adopt loop end-to-end and stands as the reference exemplar. | Do NOT re-harvest TabBar or restart a TabBar migration. The successor plan (`rcl-trust-hardening-adoption-reconciliation-and-detail-view`) does no new component migrations; it only cleans up TabBar's orphaned catalog/index rows and hardens the loop's invariants. No new component is harvested in that plan. | Revisit only if TabBar is deliberately refactored to strip its scenario-specific coupling and a fresh case is made for it as a shared primitive. |
| 2026-07-16 | Preserve VoiceMicButton presentation and pointer semantics as the RCL VoiceInputButton contract. | The first RCL version used a circular generic control, which omitted the established compact rectangular geometry, icon palette, level fill, timeout ring, error popover, and pointer-down/up semantics. | RCL reproduces those presentation and interaction rules using RCL tokens and injected callbacks only; browser capture, TTS, VAD stores, transport, and scenario policy remain outside the asset. | Revisit when a proven reusable voice state cannot be represented by the controlled component-plus-hook contract. |
| 2026-08-04 | Make `ControlBase` the governing primitive for adopted controls, with one coupled size scale and container-supplied density. | Button, IconButton, and VoiceInputButton had independently drifting geometry, spacing, focus, hover, and disabled treatment. A control should resolve height, horizontal padding, icon size, and radius from `xs`–`xl`; `comfortable` and `compact` density adjust internal spacing without changing the public size contract. | New controls compose `ControlBase` and expose the same size/density vocabulary. Component experience claims can validate spacing, state treatment, and size parity at the catalog boundary. | Revisit only if a control genuinely cannot express its interaction geometry through the shared primitive or if a new density mode is proven necessary. |
| 2026-08-04 | Keep token translation scenario-owned and require injective, CSS-variable-backed mappings with a contrast floor. | Consumer palettes differ, and a library-owned translator previously collapsed distinct semantic roles. Each adopter now owns `ui/token-map.json`; adoption fails closed when roles collide, a role is missing, a target is not CSS-backed, or a declared contrast pair is below the floor. | Theme changes remain local to the adopting scenario while shared assets preserve semantic roles. Adoption errors name the colliding roles and target. | Revisit if the token contract gains a shared cross-scenario palette service rather than consumer-owned CSS variables. |

## Superseded Decisions

| Date | Superseded Decision | Replacement | Notes |
|---|---|---|---|
| None yet. | n/a | n/a | Add when a durable decision is replaced. |

## Cross-References

- [`../concepts/ARCHITECTURE.md`](../concepts/ARCHITECTURE.md) — system decisions
- [`PROBLEMS.md`](PROBLEMS.md) — unresolved drift and debt
- [`PROGRESS.md`](PROGRESS.md) — completed work history
