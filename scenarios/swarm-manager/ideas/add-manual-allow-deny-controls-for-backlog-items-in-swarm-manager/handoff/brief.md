# Idea Execution Handoff

This package captures the finalized swarm-manager idea context for downstream ecosystem-manager execution. It is regenerated from the latest finalized backlog state when idea execution begins so downstream work starts from a stable contract rather than scattered workshop artifacts.

## Execution Contract

- Backlog item: `idea/add-manual-allow-deny-controls-for-backlog-items-in-swarm-manager`
- Title: Add manual allow/deny controls for backlog items in swarm-manager
- Target scenario: `add-manual-allow-deny-controls-for-backlog-items-in-swarm-manager`
- Recommended ecosystem operation: `generator`
- Recommended steer profile: `rapid-mvp`
- Item folder: `scenarios/swarm-manager/ideas/add-manual-allow-deny-controls-for-backlog-items-in-swarm-manager`
- Plan: `scenarios/swarm-manager/ideas/add-manual-allow-deny-controls-for-backlog-items-in-swarm-manager/plan.md`
- Manifest: `scenarios/swarm-manager/ideas/add-manual-allow-deny-controls-for-backlog-items-in-swarm-manager/handoff/manifest.json`
- Source index: `scenarios/swarm-manager/ideas/add-manual-allow-deny-controls-for-backlog-items-in-swarm-manager/handoff/source-index.json`

## Downstream Requirements

- Read `plan.md` and `manifest.json` before creating the ecosystem-manager task.
- Use this `brief.md` file as the ecosystem-manager task notes.
- Preserve the origin metadata so later ecosystem-manager loops can trace back to the swarm-manager source artifacts.

## Product Intent

Currently there is no way to manually set the allow/deny status for backlog items in swarm-manager. This would be useful as a fallback when workshop agents forget to set the status during their processing.

## Locked Decisions

- Round 001 `d3`: Client-side glob validation approach -> option __other__
  Freeform: Glob + api to check that it’s a real path within the project
- Round 001 `d2`: Interaction pattern for editing glob arrays -> Modal/dialog editor
- Round 001 `d4`: Save behavior for glob changes -> Explicit save button
- Round 001 `d1`: Scope clarification: what does 'allow/deny' mean? -> Editable acceptance_allow/acceptance_deny globs
- Round 001 `d5`: Should locked items (queued/in_progress) allow glob editing? -> Lock glob editing on queued/in_progress items
- Round 002 `d2`: API path validation: how to check that globs match real project paths? -> New API endpoint: POST /backlog/validate-globs
- Round 002 `d4`: Error display format for invalid lines -> Error summary below textarea with line numbers
- Round 002 `d1`: Modal layout: combined or separate dialogs for allow vs deny? -> Single modal with two stacked textareas
- Round 002 `d3`: Validation UX: when to show errors? -> Validate on blur + debounced typing (500ms)
- Round 003 `d3`: Empty state UX: what to show when no patterns are set -> Add 'No patterns set — click to add' placeholder on detail page
- Round 003 `d2`: Testing approach: vitest component tests or Playwright E2E? -> Vitest component tests only
- Round 003 `d4`: Textarea placeholder text and helper copy -> Placeholder in textarea + helper text below label
- Round 003 `d1`: Validate-globs API: phasing strategy -> Ship all phases together

## Remaining Open Decisions

- None.

## Execution Boundaries

- acceptance_allow: none recorded
- acceptance_deny: none recorded

## Validation Starting Point

- `vrooli scenario status add-manual-allow-deny-controls-for-backlog-items-in-swarm-manager`
- `scenario-completeness-scoring score add-manual-allow-deny-controls-for-backlog-items-in-swarm-manager`
- `scenario-auditor audit add-manual-allow-deny-controls-for-backlog-items-in-swarm-manager --timeout 240`

## Supporting Sources

- Spec: `scenarios/swarm-manager/ideas/add-manual-allow-deny-controls-for-backlog-items-in-swarm-manager/spec.json`
- Workshop rounds:
  - `scenarios/swarm-manager/ideas/add-manual-allow-deny-controls-for-backlog-items-in-swarm-manager/workshop/round-001.json`
  - `scenarios/swarm-manager/ideas/add-manual-allow-deny-controls-for-backlog-items-in-swarm-manager/workshop/round-002.json`
  - `scenarios/swarm-manager/ideas/add-manual-allow-deny-controls-for-backlog-items-in-swarm-manager/workshop/round-003.json`
  - `scenarios/swarm-manager/ideas/add-manual-allow-deny-controls-for-backlog-items-in-swarm-manager/workshop/round-004.json`
