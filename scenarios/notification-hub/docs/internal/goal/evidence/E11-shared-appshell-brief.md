# E11 worker assignment — shared AppShell menu plus tabs

Parent/run: `11175250-c50d-44e9-9af5-cd6eba79bab2`. Read `scenarios/prompt-manager/store/skills/packs/core/large-effort-orchestration/references/worker.md` and this complete brief. Authoritative ledger: `scenarios/notification-hub/docs/internal/goal/epochs/E11-shared-appshell-menu-tabs.md`. Resume only its assigned files and cited sources; do not read `FEEDBACK.md`.

## Assignment

Implement one coherent enabling behavior: extend the existing RCL AppShell through the governed draft workflow so a phone can show the compact header destination menu and keep the existing five bottom tabs at once; adopt that published contract in Notification Hub and remove the duplicate Hub-owned mobile `<nav>` and menu state when the library owns them. Preserve the selected NH16.1 phone hierarchy, all five destinations, router links, current state, accessibility and unique tests. Do not suppress `standard_shell_ownership`, drop tabs, or broaden to other screens.

## Authority, sources and exact starting state

This is inside the already commissioned five-destination redesign. The owner approved a measured shared-component extension in seq38 feedback `HUB-E5-REAL-SHELL-GAP-AND-INFO-LIMITS-20261021`; seq39 digest correction `HUB-E5-DARK-DIGEST-CORRECTION-20261021` is already reconciled. Read the relevant BENCHMARK, selected dossier/atlas NH16.1, RCL `docs/guides/asset-update-flow.md`, and existing AppShell source/tests before writes. Exact selected originals: NH16.1 light `docs/internal/delivery-operations/2026-10-12/assets/notification-hub-revised/originals/NH16.1-light-phone-p1-r3.png` SHA-256 `dd0f265a97772e573d8dd00d243f7a6e3291da9d49fee43c32eee4c14e962313`; dark `NH16.1-dark-phone-p1-r3.png` SHA-256 `1febf9bf5328bfc55f4e3fd6347f4d5c259473875431fa61a11403724855648e`. The selected NH16.1 phone composition includes compact destination menu plus lower tabs. AppShell 2.1.1 defines mutually exclusive `mobileNav="tabs"|"drawer"`; Hub currently supplies a second `<nav>` in its `header` prop.

Admission anchors (dirty scope fingerprints, not authorship): RCL `gcta-9ffee1da12e6`; Hub `gcta-52e0dd80b663`; shared HEAD `34515f0f501d511f65aa06c8e323620bf6d05891`. Current Hub hashes: `ui/src/layout/AppShell.tsx` `923d893384838c64ddadd1b6308b306678a35e805b8defd914234b62f6d29f90`; `ui/src/styles.css` `413b7e6bfb5e7133aefdf6adc3a6651597264bec6ccfd1667345df5fff746b45`; `ui/src/layout/AppShell.test.tsx` `7c6575e97e6dc6d99c959b99871a2246445a9fe1321457ea4fce6b526e8c266e`.

Current whole-goal state: all ten benchmark gates open. E1 seq38 returned terminal COMPLETE unaccepted: focused AskPage tests/checks pass, but its source is not served; the managed shadow owner refused because rebuild would replace files served by live Hub, and fixture-subject authorization lacks a verified binding. No restart, lease clear, alternate route/identity, answer or data write. Do not resume E1 or repeat its lifecycle attempt. E5 UI Health `20261021-011625-5224f968` remains FAIL at the local nav owner `AppShell.tsx:52`; native-screen/safe-area are INFO limits. Workflow Health `a53cf6f6-417e-42ae-8b18-9576ee79c639` passed its registered History cases; do not rerun unchanged cases. Keep these separate from your checks.

## RCL owner route and product scope

Use `react-component-library components draft-begin react-component-library:AppShell --json`; capture exact returned component/version/draft source path and digest. Edit a temp copy and submit only through `components content-set <returned-id> <temp-file> --expected-sha256 <digest>`. Never write a released version or generated artifact directly. Follow the asset update guide: applicable catalog generation, exact draft `type-check:catalog` and `lint:catalog`, exact draft story/test, `components draft-publish ... --json`, strict package artifact build and required library checks. Confirm final component registry/version and package artifact identity. No dependency install or pin change.

Hub paths are limited to `scenarios/notification-hub/ui/src/layout/AppShell.tsx`, `AppShell.test.tsx`, and only necessary shell-local strings/selectors/style files. Use the published version; remove Hub's own menu `<nav>` and associated menu state/effects only when the shared owner replaces them. Do not touch unrelated dirty paths. If either admission anchor/current source owner conflicts with these exact files, stop before writing and report it.

## Required proof and stop rules

- Add meaningful RCL story/test for `tabs`, `drawer`, and the new composition; keep existing expectations intact.
- Add focused Hub tests for accessible menu open/close, route/current state, and all five persistent tabs at phone width, plus desktop behavior where directly affected.
- Run applicable exact draft catalog type/lint/test commands, focused Hub tests and TypeScript/build checks. Record actual results and preserve pre-existing failures.
- Record whether current served-build adoption is available. Managed lifecycle is the only runtime route. Stop on a refusal; do not restart live Hub, clear/reuse a lease, change identity or route, or repeat unchanged UI Health. A non-served source build is not served proof.
- Compare relevant captures with the exact selected NH16.1 light/dark originals only if a supported current build is available. Preserve matching state/theme/viewport and identify adaptations. Never claim full visual fidelity from static tests.
- No independent review or gate acceptance by worker. Return exact changed files/hashes, deleted symbols/callers, validation receipts, draft/publish identity, adopted source/artifact identity, remaining served/UI Health gates, attribution limits, waits/interventions, and all remaining E1/E5/full-goal gaps.

## Exit

Complete this bounded enabling slice when the governed shared composition is implemented, its RCL and Hub focused checks are run, and the handoff clearly distinguishes source, published artifact and served build. If a required owner command fails, preserve the exact result and return its evidence; do not mark that gate passed. The parent owns independent review and all acceptance.
