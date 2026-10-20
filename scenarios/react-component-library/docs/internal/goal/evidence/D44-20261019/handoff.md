# D44 bounded RCL UI repair handoff

## Identity and scope

- Run: `2a48efd8-0385-4aa5-bf7e-ffad0dfb120d` (Agent Manager verified identity).
- Native task: `568a8c73-436e-4faa-8e18-e6d7f7e03799`.
- Parent: `2b16a068-2bc1-4a41-8629-54c4992c93c6`; profile `4c19aef1-36a2-4cc5-8961-42648692883b`; `gpt-6-luna`, medium; workload `rcl/E1`; verified effort revision `rcl-rehabilitation-20260930-1`.
- Product write set: UI `src/**` only. No API, CLI, library release, generated output, dependency, lifecycle, publishing, full RCL/API suite, or E1/G5/G7 acceptance work.

## Initial diagnostic

One `pnpm run test:coverage` from `scenarios/react-component-library/ui`; exit 1; start `2026-10-19T21:28:23Z`, end `21:29:07Z`, wall 44.542s. Full stdout/stderr are `initial-coverage.stdout` and `initial-coverage.stderr`; metadata is `initial-coverage.meta`. Recovered exactly 20 failing test names (the D44 directive expected 21): see `initial-failures.txt`. Results: 20 failed, 557 passed, 0 skipped; 11 files failed, 98 passed, 109 total; functions 82.78%, branches 80.04%. This preserves the requested historical baseline and unchanged 85% thresholds.

## Changes and evidence

- Fixed `ActionLauncher` multi-target adoption: its prior `multiple` prop was sent to the shared single-select, so targets could not be selected independently. It now uses the shared Checkbox asset for filtered multi-target selection. Tests retain the two-workflow behavior and shell deep-link selection.
- Fixed `CreateComponentDialog` initial focus: the modal default focused its close control and stole focus from the slug field during fast typing. The dialog now supplies the slug input as initial focus; regressions assert the complete typed slug survives and both success/error outcomes remain covered.
- Repaired stale UI tests to interact with the shared Select button/listbox and verify its current accessible behavior. Updated route uniqueness and current stylesheet-key assertions, and corrected the clipboard-missing setup.
- Exact original path snapshot and hashes: `baseline-original/manifest.json` (SHA-256 `4829be5ce275dda4aeb1f9cfdbade025a60686612f58b8fc1651dacf3c6a2a9e`); supplementary ApplicationShell test checkpoint: `checkpoint-after-initial-fix/manifest.json` (SHA-256 `e3f2b96086c28f93048fbdb0169dd22360ddb72116c9dd1ca3ca847c58d82915`). Postimage hashes for all captured paths are in `postimage-manifest.json` (SHA-256 `f9c398cedd15a219892abf330ea4e66982376ae9469d3feb8e026c490f48df58`). Exact diff: `D44.diff` (SHA-256 `e5f0e6c35d42ec04da1e6016a959178daf8fad3e0cb8482c5b2ec1aa7e5aba8e`). 13 changed paths: two runtime files and 11 test files; net runtime +17 lines, net test +16 lines.
- Desired-behavior regressions: `ActionLauncher.test.tsx`, `ApplicationShell.test.tsx`, and `CreateComponentDialog.test.tsx` retain multi-target workflow count, deep-linked selected target, and full input/error/success behavior.
- Old-source causal controls ran in `/tmp/D44-old-source-control/scenarios/react-component-library/ui`, using captured original `ActionLauncher.tsx` and `CreateComponentDialog.tsx`, plus the changed regressions. Exit 1 as intended: 2 failed, 1 passed, 8 skipped. The old launcher could not supply target checkboxes; the old dialog did not preserve the complete slug. Complete output is `old-source-controls.stdout`, stderr `old-source-controls.stderr`, metadata `old-source-controls.meta`. An earlier control setup that symlinked the library failed Vite filesystem resolution; the corrected disposable library copy made the actual controls runnable without shared-source writes.

## Focused and final validation

- Changed-input focused run: ten test files, 97 passed, 0 failed, 0 skipped; exit 0; `focused-ui-final.meta` records 7.240s wall time and stdout includes the exact verbose names.
- Corrective launcher/shell/adoption guard run: three files, 27 passed, 0 failed/skipped; exit 0; 4.489s wall; full output in `focused-launcher-shell.stdout`.
- Settled final `pnpm run test:coverage`: exit 1; `2026-10-19T21:37:51Z`–`21:38:28Z`, 37.153s wall. 575 passed, 2 failed, 0 skipped; 1 file failed, 108 passed, 109 total. Functions 92.68%, branches 81.58%, lines/statements 94.59%. The unchanged 85% branch threshold fails. Full stdout/stderr: `final-coverage-settled.stdout` / `final-coverage-settled.stderr`; metadata: `final-coverage-settled.meta`.
- Remaining exact failures: `src/lib/CollectionList.test.tsx > keeps the open detail distinct from keyboard navigation and bulk selection`; `src/lib/CollectionList.test.tsx > does not substitute another current item when the open record is filtered out`. Both behavior assertions are blocked by strict test-setup rejection of `[react-component-library] stylesheet key collision for react-component-library-cardshell-1.0.2`; this is library-owned imported source outside the admitted `ui/src/**` product scope. No test expectation or console guard was weakened. Next owner: admit a separate RCL library repair for the CardShell stylesheet collision, then rerun focused changed inputs and coverage under its own scope.
- One earlier post-fix coverage checkpoint exposed two ActionLauncher regressions (native-checkbox policy and shell selection assertion); it is retained as `final-coverage.*`. Those were corrected, focused-tested, and the settled final coverage above is the authoritative final-state run.
- `pnpm run type-check`: exit 0; `21:38:34Z`–`21:39:25Z`, 51.146s wall. String check, token check, 915 export checks, catalog TypeScript and lint completed. Lint reported 14 warnings (0 errors) in existing library files; complete stdout/stderr and metadata are retained as `type-check.stdout`, `type-check.stderr`, `type-check.meta`.

## Handoff and accounting

- Independent-review packet is ready at this evidence directory; no distinct independent review has yet been performed. Parent must obtain it before disposition.
- `agent-manager effort epoch-check scenarios/react-component-library/docs/internal/goal/epochs/E1.md --runs 2a48efd8-0385-4aa5-bf7e-ffad0dfb120d --wake-key 2b16a068-2bc1-4a41-8629-54c4992c93c6` reported step-back: 63 slice-log work units vs 32 trigger (2.0×), and woke the parent. The D44 line and step-back note are in E1.md; this does not accept the epoch or any gate.
- Latest native token read while the run/task were still `running`: 317,285 weighted non-cache tokens and 18,493,568 cache-read tokens, Luna ×1. This is a live observation, not final accounting; receipts and terminal spend have not been observed, so cost attribution remains provisional.
- Changed: exact bounded UI repairs and evidence. Verified: 97/97 focused, corrective 27/27 focused, old-source controls distinguish both fixes, UI type-check exit 0. Remaining: two CollectionList failures / branch threshold and independent review. Unverified: terminal native accounting, independent review, and every E1/G5/G7 acceptance gate.
