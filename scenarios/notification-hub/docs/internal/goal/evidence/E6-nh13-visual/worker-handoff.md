# E6 worker handoff — NH13.4 / NH13.6

Date: 2026-10-20. Worker run: `39226203-f710-4046-9cce-83f22fa35d43`. Parent: `11175250-c50d-44e9-9af5-cd6eba79bab2`.

## Implementation

`AsksPage.tsx` now requests the supported 200-item cap. A settled successful empty read renders the bounded “Needs you” empty composition, checked-view wording, “Latest 200 maximum” scope, and an older-requests caveat. Loading or refreshing an empty cache does not claim a current empty result. A failed initial read renders a connection failure, says the read did not clear requests, and offers Retry (the existing `ListAsks` refetch) and Settings. A failed refresh retains stale rows and the ordinary filter/History controls. The failure path does not call answer, clear, delete, or any primary write API.

English, Japanese, and Arabic copy were added in locale parity. `strings.generated.ts` was refreshed with the scenario-owned `pnpm strings:gen` command. The changes replace the former one-line empty/error state and retain the existing list filtering and History route. No API, service, storage, auth, schema, or dependency files were changed as part of this scoped delta.

## Checks

- Focused asks tests: 31 passed across AsksPage, AskPage, and HistoryPage.
- Locale parity tests: 5 passed.
- Type-check, including `strings:check`: passed.
- ESLint on AsksPage and its tests: 0 errors; two existing unsafe `location.state` warnings remain at AsksPage.tsx:42.
- Test Genie unit run `20261020-222708-c3817c57`: 181/183 tests passed. The two failures are `AppShell.a11y.test.tsx` assertions still waiting for “No decisions are waiting for you”; direct reproduction confirms they fail at that stale text lookup before axe/navigation assertions run. All UI coverage floors passed: lines/statements 90.15%, functions 85.39%, branches 87.53% against the unchanged 85% floors. Details and retained summary are `unit-phase-20261020-222708.json` and `unit-coverage-summary-20261020-2227.json`.
- Managed `vrooli scenario restart notification-hub --json --timeout 300`: succeeded; healthy UI/API at ports 20001/17736. Served build identity: `sha256:b714e695f87451e3d8dad5c3b50a2015ed028c5772b9e431502b329dd6f2dcbb`.
- The run's optional SMTP credential is unavailable; it is unrelated to the UI behavior.

## Workflow and visual evidence status

The supplied terminal Test Genie run `notification-hub/20261020-221520-61329d0e` failed its single `workflow` phase (326 seconds). It reports eight `workflow.execution_failed` findings: step 7, “NH13.4 selected state is visible,” at the existing E4 ask, delivery, and history cases; and two `workflow.selector_unregistered` warnings for literal `[role="alert"]` selectors in the E4 read-failure cases. These workflow assets are outside this epoch's admitted write scope and were not modified. Test Genie's artifact catalog retained only logs/findings/manifest, but BAS's own execution history retained the four selected-state screenshots; copies and per-image hashes are in `captured-testgenie-20261020-2215-pre-final-build/` and `selected-state-capture-manifest.json`.

Those served screenshots visibly contain the new NH13.4 and NH13.6 state content and are 780×1688 px (390×844 CSS px at DPR 2). They were captured under build `sha256:951f47a3011fef3d636bc2f96d24b6d90dc06247607a0747468a7abf443cb2bd`. The final AsksPage source modification occurred after these captures and the managed current build is `sha256:b714e695f87451e3d8dad5c3b50a2015ed028c5772b9e431502b329dd6f2dcbb`. Each dark image is byte-identical to its light counterpart, so dark theme was not applied. These artifacts help diagnose the earlier run but do not pass G2. No current-build matched captures are available. G3 remains unproven: no completed served-browser retry → Settings record or client-call trace exists. G1 remains failed because the unit suite includes the two stale AppShell text assertions and the supplied workflow phase fails existing selected-state selectors, although the focused E6 tests and coverage floors pass. Updating `AppShell.a11y.test.tsx` is outside the admitted E6 files; parent authorization was requested. G4 requires the separate parent review. E6 and E4 are not accepted by this worker.

## Attribution and responsive/accessibility notes

The scoped prewrite manifest and current six-file hashes are in `prewrite-original/manifest.json` and `source-inventory.json`. The workspace has unrelated dirty state, so attribution outside these six scoped paths is unknown. Tests include axe coverage for the empty state and cover Settings navigation, bounded copy, failure distinction, stale-row retention, History access, and read-only retry. Visual responsive behavior on the current build and dark-mode matching remain unverified.

## Work units

Five meaningful units: (1) bounded authoritative empty state and 200 cap; (2) truthful ListAsks failure/retry and stale-row handling; (3) localized copy, generated strings, and expected-behavior regression coverage; (4) recovered and audited the four retained BAS screenshots against build identity and theme, establishing why they cannot certify the current build; (5) isolated the unit-phase failures to stale AppShell copy assertions and verified that coverage floors still pass. This is the actual delivery count, not an estimate stretch.
