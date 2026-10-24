# Rehabilitation scope and acceptance

Revision: `rcl-rehabilitation-20260930-1`. This is an approved destination, not a claim of implementation. All new requirements remain unimplemented until current evidence meets their gates. The goal home is the sole live delivery ledger; the scenario requirements registry owns obligation status. External effort files and owner boards are pointers/projections.

## Product contract

RCL is a shared UI capability and a design-to-implementation workspace. Reuse takes priority over catalog expansion. Start from an existing page/layout/pattern before a primitive; invent only after catalog and declared designs have been considered. The default consumer path uses versioned shared package imports with governed versions, compatibility and dependency closure. Page-specific composition remains local. An explicit ejection/local ownership path records the source version, reason, upgrade implications and recovery boundary; it must not silently pretend a copied file remains centrally managed.

Separate stable identity, versioned source/contract, stories and registry projections. Each concept has one authoritative owner. Indexes, generated exports, caches and UI projections must not become independently authored truth. Keep the registry, draft/release lifecycle, preview runtime, consumer tooling and studio responsibilities legible; choose boundaries from evidence rather than impose a new framework. BaseStyles/token generation remains the runtime token authority; no parallel studio token palette or scenario-local fork of the core.

Use the existing `templates/design/*/metadata.json` registry and the same IDs used for scenario generation. Initial styles: `vrooli-default`, `switchboard-console`, `vrooli-command-display`, `vrooli-conversion-landing`. The selector identifies the target scenario style, persists the choice and drives preview/composition. Keep native/compatible/discouraged affinity findings distinct from token or dependency incompatibility. Unsupported choices explain the limitation; no silent fallback, competing template list or hardcoded alternate taxonomy. Selected core assets need credible states and composition in all four supported styles; J2/J3 prove two different styles in built consumers.

## Selected core

Initial candidates (subject to consolidation and dependency closure at first epoch):

| Group | Candidates |
| --- | --- |
| Foundations | BaseStyles, Tokens, Stack, Text, Heading |
| Controls/forms | Button, IconButton, Input, Select, Checkbox, FormField |
| Interaction | Tabs, Dialog, Popover, Tooltip |
| Work surfaces | DataTable, MasterDetail, AsyncPanel, EmptyState, CollectionPage |
| Hooks | useAsyncAction, useMediaQuery |

This is a selection hypothesis, not a mandate to introduce missing names or force a poor abstraction. Before repairs, record exact library IDs, versions, dependency closure, existing equivalent assets, supported states and compatibility. Merge/remove redundant candidates when one contract suffices. Add an asset only when a proving journey requires it, with rationale and affected scope recorded in the active brief. The 474-asset desired catalog is outside certification; quarantine or label incomplete assets honestly without deleting consumer dependencies indiscriminately.

Core readiness requires documented typed contracts, public imports, complete needed dependencies, purposeful examples of meaningful states, keyboard/focus behavior, semantic accessible names, readable contrast, responsive composition, error/loading/empty/disabled handling where applicable, supported template fit, and actual consumer builds with generated exports and CSS/Tailwind content inclusion. Story quantity, scaffold markers or a green isolated preview alone do not establish maturity. Existing contracts should be preserved or migrated explicitly; incompatible changes have versioned communication and a safe recovery path.

## Five proving journeys

| ID | Start and outcome | Required proof |
| --- | --- | --- |
| J1 | Ordinary home → discover an asset → understand contracts/states/style → adopt it into a real scenario | No secret URL, seeded bypass or maintainer rescue. Preview and built consumer agree; shared import, exports, dependency closure and CSS work. Keyboard, responsive layout and actionable failure states pass. |
| J2 | Prompt Manager team collection/detail workspace | A bounded usable filter/table/selection/detail/dialog experience in a real build, including loading, empty, failure, permission/disabled explanations and mobile adaptation. No broad Prompt Manager redesign, world-map makeover or agent-system rewrite. |
| J3 | A second existing scenario's settings/form workspace in a different design template | Validation, submit/save, pending state, error/retry, safe recovery and mobile usability work. Select the concrete consumer at baseline for existing workflow value and manageable ownership; record choice before implementation. No new scenario merely to make a test pass. |
| J4 | Ordinary home → intuitive studio entry → choose scenario/page → author a brief → compose → inspect viewports/style → review → governed implementation/adoption | Reopen preserves scenario-owned brief/sketch, style and placements. Reuse suggestions are intelligible; placeholders/unplaced work remain honest. Canvas and controls have keyboard access; each step has a clear next action. Source reconciliation and an executable implementation brief lead to verified real source, not an image-only mockup or an unguided editor. Save/preview/reconcile/adoption failures explain recovery and preserve work. |
| J5 | Discover a defect → governed asset draft → repair and publish → inspect impact → upgrade two consumers → recover | Do not require knowledge of a special component's internals. Use `components draft-begin`; never modify a release directory. Consumers show current/behind/local-modified/incompatible/unknown honestly, preserve local work, require meaningful confirmation for unsafe writes, and demonstrate rollback/recovery. |

A fresh agent must complete J1 and a bounded J5 repair using documented discoverable CLI/library entrypoints without bespoke rescue. A human walkthrough must assess J1/J4 discoverability and mobile usability from normal entry. Record observations and evidence separately; automated screenshots cannot establish that the workflow is intuitive. UI/CLI parity means equivalent essential outcomes and safety, not identical widgets.

## Cleanup and maintainability gates

At first admission inventory handwritten runtime lines, generated lines separately, duplicate ownership paths, public entrypoints and obsolete process ledgers. Measure RCL API/UI/CLI and consumer-tooling package separately; Prompt Manager only for touched proving surfaces. Generated corpus size is not a proxy for maintainability. Record the reproducible counting command and baseline in the epoch, then set a meaningful reduction target after inspection; do not copy BAS's numerical target or invent a current RCL baseline.

Each refactor brief names concrete files/symbols/patterns to replace and delete, one surviving owner, callers to migrate, preservation behaviors and the net-negative runtime budget. Acceptance checks every deletion and caller, not just line totals. Feature epochs justify net growth and a bounded size estimate at admission; fixtures/docs/generated outputs are reported separately. Explain net test growth by newly covered public behavior. No duplicate fake infrastructure, sleeps for state or tests of private helpers. Consolidate outdated docs, parallel readiness logic, custom orchestration glue and one-off evidence machinery where discovered; archive historical evidence rather than erase it. Do not remove useful failure recovery merely to reduce code.

Meaningful progress is an accepted user journey or a complete replace-and-delete ownership slice. Receipts, dashboards, catalog counts and new process documents alone do not count. Track accepted outcomes, runtime delta, duplicate owners removed, fresh-agent rescue interventions, operator input required and cost per accepted epoch in the existing epoch log/queue; do not build another tracking product.

## Evidence and target tiers

| Target | Initial tier | Gate |
| --- | --- | --- |
| Linux x64 API/CLI and real RCL/consumer builds | required | Relevant compile/static checks and focused public-behavior regressions; production bundle must resolve imports, exports, dependencies and styles. |
| Chromium desktop and mobile emulation | required | J1–J5 affected journeys from normal navigation, keyboard/focus checks and reviewable captures; mobile is a real layout, not a scaled desktop. |
| All four registered design templates | required | Core state/composition fit and persisted selector identity; two distinct styles in J2/J3 consumers. |
| Real mobile devices and secondary browsers | occasional | Run when available and record coverage; Chromium emulation is not physical-device certification. |
| Unsupported host/browser targets | unavailable until assessed | State what remains unverified and why; never silently certify. |

Use [scenario testing guidance](../TESTING.md) and [repository testing policy](../../../../../docs/TESTING.md). Discover exact focused commands and current BAS stories at admission; no invented executable gates. Server-owned suites use `vrooli scenario test react-component-library --phases <relevant phases>`; wait once with `test-genie runs wait --json <scenario> <run-id>`. Run impacted Prompt Manager/second-consumer phases too. Generated-artifact compile failures go to their owning preflight before consumer diagnosis. Release/consumer proofs must include actual resolved versions and hashes/build identity, style IDs, viewport, result, run reference and known limitations.

Daily qualification after activation: `react-component-library catalog readiness --json` plus the currently established normal-entry J1/J4 smoke in a disposable/shadow engagement. At first epoch record the exact smoke command, fixture isolation and safe start/cleanup in this section. Readiness output is diagnostic: full-corpus failure does not block selected-core closure; missing/stale evidence is unknown, not a behavioral failure. Do not start a shadow now. Use governed lifecycle and baseline commands; preserve live user designs and do not restart an active consumer without checking who uses it and choosing an isolated/fallback validation path.

The orchestrator reruns admission gates independently per `large-effort-orchestration` §4.4. If a shared tool is unavailable, record the fallback and unverified evidence in WORKAROUNDS; do not create a private replacement or silently certify the missing target. Product correctness/recovery failures remain unfinished work even when the evidence tool is unavailable.

## Closure

Close only when the exact selected core/closure and J1–J5 meet current gates, documentation and CLI let a fresh agent succeed, all refactor deletion lists are complete, and no blocking correctness/accessibility/data-safety issue remains in the selected scope. Report excluded catalog assets, compatibility exceptions, unverified platforms and open infrastructure workarounds. Clean up goal-only installs/processes, reconcile obligation statuses with evidence, disable the team and heartbeat, and record the work record. Neither an ended leader run nor the old campaign's completion closes this revision. Product code changes, releases and certification have not been performed by this campaign amendment.
