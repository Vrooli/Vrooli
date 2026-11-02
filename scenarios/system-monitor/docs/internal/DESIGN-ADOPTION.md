# System Monitor design adoption contract

Status: Full Vega redesign target. Implementation and product acceptance are unverified.

## Current target and authority

This contract defines the full Vega redesign: a complete existing-product experience using the selected atlas as its basis, with independently verified improvements. The effective target and delivery constraints below govern every affected surface.

The selected direction is **Signal Workbench composition with Optical Observatory A styling**. Preserve the verified Vega logo, five persistent history charts, the navbar Bridge/machine selector, complete mobile workflows, both appearances, and the current genuine code/JSON editor treatment. Use graph-navbar-parity-v2 for graph chrome, compact-task v6 for corrected mobile tasks, and editor v7.5/v7.5.1 for the genuine editor treatment. The generated mark in the atmospheric art study is not the product logo.

The redesign preserves operational requirements, truthful metric semantics, accessibility protections, working integrations, user data, and owner-controlled effects. The PRD, REQ-P0-007 and experience inventory use this current target.

Canonical inputs, relative to the repository root:

- Product contract: `scenarios/system-monitor/PRD.md` and its imported requirements.
- Full behavior/design dossier: `docs/internal/delivery-operations/2026-10-12/sources/vega-ux-dossier.md` (operator journey, route/action contracts, D1-D10, and acceptance scenarios).
- Version context: `docs/internal/delivery-operations/2026-10-12/sources/vega-design-atlas.md`.
- Exact images, source archives, logo/font/fixture custody and per-requirement coverage: `scenarios/system-monitor/docs/mockups/vega-redesign/assets-manifest.json` and `.vrooli/work/delivery/vega-redesign-delivery/sources/commissioning-source-map.json`.
- Owner review companion: [full-resolution PDF](https://drive.google.com/file/d/1nwgKBHGV67YnysSQy0r6Ar2Fekhdpc18/view). It is a browsable view of originals; it does not select every historical image or replace the dossier and source bytes.
- Current metric semantics: `scenarios/system-monitor/docs/internal/METRIC_CONTRACT.md`; platform availability: `scenarios/system-monitor/docs/reference/platform-capabilities.md`.

## Effective design and improvement policy

Apply `experiential-ui-design` section 2a. The effective target consists of selected originals, applicable owner amendments, retained capabilities, and independently verified agent refinements. Use the same task, state, theme and viewport to judge changes. The atlas is the starting implementation target; it does not require copying an awkward artifact or deleting an operation absent from a screenshot.

Agents may improve hierarchy, spacing, density, navigation, suitable icon buttons, editors, component fit, recovery wording and motion within the selected concept. Use suitable governed RCL components after inspecting their actual contracts; customize when the shared component does not meet the task. Keep familiar secondary tools compact and preserve clear labels for consequential or unfamiliar actions. Phone and desktop are co-equal compositions, not a reduced phone feature set.

Describe the effective design and verified refinements here as current requirements. Keep proposals, source/author/date, rationale, tradeoffs, before/after evidence and disposition in delivery records under stable author-scoped IDs. Routine finishing choices may be grouped. An agent proposal becomes part of the effective target only after its behavior and benefit are verified through the existing independent acceptance route. No replacement atlas or repeated concept approval is required for ordinary in-scope improvements. Changed concepts, removed capabilities or unresolved consequential authority choices require the owner's specific decision.

| ID | Scope | Current requirement |
|---|---|---|
| VEGA-DESIGN-20261025-01 | All existing surfaces | Deliver the full redesign using the atlas as inspiration, with independently verified improvements. |
| VEGA-DESIGN-20261025-02 | Delivery activation | Report readiness before enabling the delivery team; activation requires the owner's instruction. |
| VEGA-DESIGN-20261025-03 | Shared shell, graphs, mobile tasks, editors | Use the selected original version for each matching screen and state; genuine editors and compact tasks govern their applicable surfaces. |

## Full redesign boundary

Deliver all ten existing routes: `/`, `/metrics/cpu`, `/metrics/memory`, `/metrics/disk`, `/metrics/network`, `/metrics/gpu`, `/forensics`, `/logs`, `/capacity`, and `/scripts`. Include the Dashboard's infrastructure, investigations/reports and settings workspaces, process ownership, shared navigation/context/view controls, dialogs, editors, first use, and recovery. These shared workspaces are not invented new top-level routes.

The delivery destination has thirteen acceptance surfaces matching the dossier's V01-V13 families: Overview; CPU/ownership; Memory/paging; Disk; Network; GPU; Forensics; Capacity/policy; Logs; Scripts; Investigations/reports; Settings/shared controls; Devices/infrastructure. Cross-cutting shell, code/JSON, motion, appearance, accessibility, unknown-result recovery and source/build identity must pass on every affected surface; they are not counted as substitute accepted surfaces.

Scope includes the UI, API/domain fixes and governed shared component/contract work necessary to deliver these existing journeys truthfully. Retain the operational obligations in the PRD even where they are unfinished. Do not claim full product readiness from redesign acceptance. New prediction/correlation features, custom collectors, distributed tracing, a custom dashboard builder, a separate native mobile app, new transport protocols, commercialization and external publication are not commissioned by this redesign. Existing webhook delivery, profiles and storage behavior remain protected; their nonvisual obligations are mapped rather than discarded.

## Dossier decisions: preparation dispositions

These dispositions define the delivery boundaries. Verify current source and served behavior for every affected journey and preserve applicable operator decisions.

| Dossier item | Binding preparation disposition | Required delivery proof / escalation |
|---|---|---|
| D1 remote subject | Preserve verified existing remote capability. Never silently fall back to local data under a remote label. Unqualified actions remain explicit local-only/unavailable with an intentional return path; do not hide or remove local capability. New remote parity is not implied. | Operate subject switching and late results on affected routes. Raise a concrete owner choice only if an unavoidable tradeoff changes working capability or authority. |
| D2 pause/follow | Label what is frozen per surface; never represent presentation pause as stopping collection/evaluation. Preserve useful historical selection and deliberate return to latest. | Operate polling, off-top readers, history selection and return; do not imply unified backend freeze. |
| D3 investigation | Preserve actual auto-fix/profile authority, launch review, lifecycle and read-only reconciliation of an unknown result. No blind retry or implied time-range parameter. | Qualified disposable fixtures for admission/recovery; actual effect authority is checked separately at operation time. |
| D4 settings/policy | Keep draft/effective/readback distinct, real units/bounds, whole-object Settings versus per-key Capacity semantics, and honest partial/unknown outcomes. Required transport repairs belong to their native owners within this outcome. | Operate save/reset, failure, delayed response, readback, newer drafts and subject change. No fictitious atomicity or confirmed defaults after load failure. |
| D5 light/identity | Use verified original Vega logo bytes unchanged; design a calibrated light appearance within the same Optical hierarchy. | Inspect both appearances, compact identity and contrast. No generated replacement logo. |
| D6 simulation | Preserve any currently useful demo/debug access with persistent simulation labeling; do not present canned output as host evidence. Relocation must preserve discoverability. | Prove the retained access path and distinct provenance. Removal of useful behavior needs a specific decision. |
| D7 scripts | Preserve inspection and qualified execution. Builtins remain read-only when production rejects updates; preserve any verified existing custom-authoring capability. No invented clone/overlay persistence. | Exact content, viewer/editor authority, return state, complete stdout/stderr and unknown-run recovery. |
| D8 inspection | Retain resource-detail routes at ordinary phone/desktop widths and selected five-chart overview; use an adjacent inspector only where it demonstrably fits. | Matching viewport/state comparisons and return continuity; no warning hero/slideshow substitution. |
| D9 adoption | The current target, PRD and REQ-P0-007 describe the selected Vega experience. | Receiver verifies source precedence, requirement coverage and actual originals before enablement. |
| D10 view entry | Viewing a route must not silently change the effective monitoring/collection state. Preserve an explicit activation control. | Inspect existing source first, then operate entry/return with collection on/off. Necessary behavior repair is in scope; retain current source/build evidence. |

## Acceptance

The orchestrator independently accepts each complete surface, not just its worker's report. Use focused regressions and relevant Test Genie phases under `docs/TESTING.md`, then the applicable integration gates. Do not require an unrelated certification run for each iteration. Requirement statuses and PRD checkboxes are earned from evidence, not changed during preparation.

For each affected surface record exact source/served build identity, governing originals and versions, applied amendment/refinement IDs, matching desktop/phone and dark/light captures, operated journeys and retained failure/recovery cases. Classify an independently verified improvement as an expected difference; missing behavior, unverified differences and actual defects remain open. A screenshot, comparison ledger, file count or passing suite alone is not completion.

Verify keyboard/focus, labels, readable dense data, contrast, 44px targets, responsive bounds, safe areas, 200% zoom and reduced motion where applicable. Use lawful disposable/routed fixtures for effects; document any unavailable device, assistive-technology or live-remote proof and its return condition instead of inventing a pass. Synthetic fixtures stay labeled. Never interpolate missing observations as measurements or relabel unsupported/failed/stale data as healthy zeroes.
