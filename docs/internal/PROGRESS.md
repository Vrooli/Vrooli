# Documentation Progress

## Rewrite Status

Completed across the current rewrite passes:

- rewrote the docs hub at `path:docs/README.md`
- added `path:docs/manifest.json`
- added canonical `path:docs/QUICKSTART.md`
- added `path:docs/concepts/ARCHITECTURE.md`
- added `path:docs/concepts/GLOSSARY.md`
- added the strategy layer and moved project framing docs under `path:docs/strategy/`
- rewrote the top-level contributor, testing, business, and production docs
- rebuilt `path:docs/deployment/` around clear current-vs-reference-vs-planning boundaries
- tightened `path:docs/reference/` against the live Makefile and CLI help surfaces
- tightened `path:docs/scenarios/` into a smaller cross-scenario canon layer
- tightened `path:docs/resources/` into a smaller cross-resource canon layer
- fixed moved-path seams and link rot across the major docs sections

## Structural Reorganization Status

Completed:

- added `path:docs/guides/`, `path:docs/operations/`, and `path:docs/strategy/`
- moved active DevOps guidance into guides, operations, reference, and deployment reference locations
- relocated scenario-specific project docs into the owning scenario docs where appropriate
- updated `path:docs/README.md` and `path:docs/manifest.json` to match the new tree
- removed the temporary archive layer after updating or deleting its callers
- removed the now-empty top-level `path:docs/devops/` subtree

## Current State

The docs tree now has a coherent canonical structure and the major section rewrites are in place.

What remains is mostly:

- specialized leaf cleanup
- continued content tightening where pages are intentionally conservative
- future maintenance to keep the canonical layer aligned with evolving CLI and platform behavior

## System Monitor Production-Readiness Evidence — 2026-08-20

The current system-monitor readiness pass has implemented and evidence-checked the
three-state metric contract, native cross-platform collectors and investigation
policies, authoritative disk-pressure autoheal behavior, lifecycle stop ownership,
platform support documentation, cost/headroom profiles, CORS and CLI surface
cleanup, and the governed responsive UI migration.

Current evidence:

- `pnpm run type-check`, `pnpm run build`, and the UI coverage suite pass. Coverage
  is 91.76% statements, 85.09% branches, 91.01% functions, and 93.58% lines,
  clearing the configured 85% thresholds.
- `pnpm run lint` passes with zero errors and 616 warnings. UI governance counts
  49 remaining inline style objects and zero TypeScript raw hex literals.
- `ui-health validate scenario system-monitor --json` returned
  `VALIDATION_STATUS_PASSED` against a lifecycle-started runtime; direct BAS
  evidence also covers settled desktop/tablet/mobile captures and a 68-second
  hold.
- The targeted governed contracts phase passes after declaring the built-in
  `space` command. `GOWORK=off go mod tidy -diff` is clean after correcting the
  CLI module's direct/indirect metadata.
- The latest full server-owned run is `20260820-083236-aabbc0fe`; its terminal
  result is a clean 22/22 phase pass. Agent Conformance reached L4 after the
  Agent Manager lifecycle readiness budget was corrected for synchronous
  bootstrap recovery.

## React Component Library Version-Model Closure — 2026-08-31

- Removed stale Arabic and Japanese locale keys after the English registry
  cleanup and regenerated the derived string registry.
- Evidence: `pnpm test -- src/i18n/locales/locales.test.ts` passes 5/5;
  `pnpm run strings:check` passes; `pnpm run catalog:check` passes with zero
  errors; and `pnpm run build` passes. Catalog lint retains 14 non-blocking
  warnings in legacy library assets.
- A fresh `make corpus-report` confirms the floor and all retention invariants;
  a fresh `make matrix` reports 192 attributable findings and zero
  finding-free failures, preserving the plan's attribution guarantee.
- Repaired component-test identity normalization so stable catalog ids resolve
  to canonical component ids before durable reports are saved. Focused Go
  tests pass, and a lifecycle-restarted live API accepted
  `controls.slider`, returning HTTP 200 with a passed 15-result report for
  `react-component-library:Slider@1.1.2`.
- The affected Test Genie phase now reaches the RCL provider and passes after
  the governed local-replace repair for template-manager. The dependency
  analyzer reports no remaining local-replace candidates, template-manager is
  healthy, and `component-tests` completed 1/1 with L1 standing in 2 seconds.

Open blockers are explicit rather than suppressed:

- Raspberry Pi hardware is unavailable, so Linux arm64 remains build-verified
  only and is not promoted to supported host-collection status.
- Agent Manager remains an explicitly declared optional dependency with a
  profile declaration source. System Monitor reports typed unavailability when
  it is absent; with the lifecycle provider healthy, the governed conformance
  phase now passes.
- UI health has no blocking contract failure, but retains advisory PWA/branding,
  governed-component, and primitive-architecture debt reported by its provider.
