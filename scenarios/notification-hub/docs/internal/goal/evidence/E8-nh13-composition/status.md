# E8 — NH13.4/.6 selected phone composition

Date: 2026-10-20

## Dispatch and scope

- Parent/orchestrator: `11175250-c50d-44e9-9af5-cd6eba79bab2`.
- Task: `8eac6f38-cd04-4c3e-87fb-15a013324b10` — “Notification Hub E8 NH13.4/.6 selected phone composition”.
- Worker run: `3fc276d0-fd2e-4e4a-8fab-deee86ce6504` (`gpt-6-luna`, medium); effort `effort:notification-hub-redesign`, revision 16 (`sha256:ee51f1db548652434a8bba3c5a301439abf2c316fef04e1652191e46b9332dd0`).
- Pre-write proposal anchor returned: `gcta-0fc50e28dca4`; HEAD `52097258290d1bff0191319242d4948e214af9e3`; scope `scenarios/notification-hub/**`.
- Admission source baseline: `AsksPage.tsx` SHA-256 `631b05009b3a49c615f427f17846488d6dcbdae2cb261a46ffd27cac716a4416`; `AsksPage.test.tsx` SHA-256 `8d0139fcc64e93d911318a0fcc94de55d584f80e91dacba501d03311365faf0d`. Both files already contained E6/E7 dirty work at the anchor; E8 preserves it and does not claim authorship of the preexisting changes.
- Selected source fingerprints: BENCHMARK `45580f33ca453f0331c5ad50dd50e74edec6f54a6e327c457573b42f1a889325`; dossier `d147181b1e7ba3b0f1dcaa06c2f2805b45761e4eda621231d217aa597eb17fdc`; atlas `5893b198f9bb9c9851fd50cc16df1607fb4da5e6a00f27450d61b8d5269ddeb4`; selected image manifest `4365502f3ee6aac508730fbdb44ad3a425a193a3c47282ebfbc82bbcb82953ef`.
- `evidence/commissioning-source-map.md` joins NH13.4 and NH13.6 to their states and source references. Product scope is limited to the existing Asks page/test, the four existing E4 cases, and this evidence directory.

## Product delta

- NH13.4 now uses a compact, flat muted surface. The selected content has no rounded/min-height panel; checked time, latest-200 scope, and the “older requests may exist” qualification remain visible. Its single bordered View history footer remains.
- NH13.6 now uses a flat full-width composition with one not-cleared callout and adjacent Try again / Open settings controls. The former inset card framing and stacked phone controls were superseded. The redundant state-local History footer was removed because the app shell’s History tab remains available; the existing empty-state footer and list History route remain.
- Retry remains one button whose handler calls `asks.refetch()` only. Loading/error/empty states remain distinct, and stale rows remain visible after a failed refresh.
- No text source changed, so the locale JSON and generated strings were not touched. No API, client, service, auth, storage, selector registry, shared package, schema, or dependency was authored for E8.

## Self-validation

- Focused command: `cd scenarios/notification-hub/ui && pnpm test -- src/features/asks/AsksPage.test.tsx` — final run passed, 1 file / 10 tests (2026-10-20 19:34:49 local).
- The expected-behavior tests cover authoritative empty only after a successful completed read, 200-row limit and bounded-scope copy, distinct failed-read state, Retry as another list read, Settings access, stale rows after failed refresh/Retry, local filtering, ordinary History access, focus/scroll restoration, and the selected empty state axe check.
- First Test Genie unit run `20261020-232753-278c57c8` — terminal `FAIL`, one `TEST_EXECUTION_FAILURE` from the API Go workspace command `/home/matthalloran8/.vrooli/bin/go test -json -count=1 -trimpath -covermode=atomic -coverprofile=coverage.out ./...`. The focused UI suite was green. Unit Health also reported 44 warnings (including existing coverage and seam findings). Do not attribute the API command failure to the UI delta without further evidence.
- Final Test Genie combined run `20261020-233604-27e7cfc6` (`unit,workflow`) — terminal `FAIL`, 1 phase passed / 1 failed, 238 seconds. Its `unit` phase passed in 23 seconds (49 observations); Unit Health retained 46 warnings, including two `MISSING_INJECTABLE_SEAM` warnings for API/CLI and 44 coverage warnings. Its `workflow` phase failed in 215 seconds on four unchanged, out-of-scope E4 cases (`delivery-dark`, `delivery-detail`, `history-bounded`, `history-dark`): each could not see `[data-testid="theme-select"]`; the selector resolved to a hidden `<select>` for 65 attempts before a 30-second timeout. No unchanged retry was made. This does not negate the selected four-case E8 proof below, and it is not attributed to the AsksPage delta.
- Workflow Health selected-case run `b696f619-c338-479c-aa0f-7349146ee666` passed all four existing registered cases (4 selected, 4 executed, 4 passed, 0 failed/refused). This is diagnostic selected-case evidence, not a Test Genie workflow-phase receipt.

## Current-build capture evidence

All four final screenshots are from the same healthy managed Notification Hub build `sha256:dd7e7df92ffd635f5d25ffbcd63630ec422aeb5d94ac1e2f656b40736118be54`, captured in Workflow Health run `b696f619-c338-479c-aa0f-7349146ee666`. The run used the four existing registered cases and lease `b696f619-c338-479c-aa0f-7349146ee666`. Every case verified CSS viewport 390×844, DPR 2, expected resolved theme, and no horizontal overflow (`scrollWidth=390`). Screenshot PNG dimensions are 780×1688.

| State/theme | Current capture | SHA-256 | Selected original SHA-256 |
| --- | --- | --- | --- |
| NH13.4 light | [asks-empty-light.png](captures/asks-empty-light.png) | `3deea8f9107e209f5297270e99a1dc71e2af33b6197c61a76ef40e4791d658c4` | `15dec5321ed38024785c8f06183a3f756bda225124d0de06e62a201f38850b41` |
| NH13.4 dark | [asks-empty-dark.png](captures/asks-empty-dark.png) | `35c9d68249ea99b72775062436de255124302e36fc957069efbb21925648d932` | `b717520d1c8346be60a23888dad52bed60e4c92324138540038c4fb6ecfafc99` |
| NH13.6 light | [asks-read-failure-light.png](captures/asks-read-failure-light.png) | `6110a79674e056783f8ae88c56066da69a1e98cc83a655d5eeb1882fb86500c4` | `3be41d4a9de476d2031bb6da3114c0ae1badc689ebecd0fe7c97ed21e22b7d45` |
| NH13.6 dark | [asks-read-failure-dark.png](captures/asks-read-failure-dark.png) | `a670a57a15a6ae736d32ceb12bcea50f684f97c98fff2c019f1848b141400238` | `50d3495028ae127ee58f8cbcb6de2977dbd4c0079360cc113f4e3f032e784796` |

Visual comparison: the empty state keeps the selected flat surface, compact evidence rows, bounded-history qualification, and bordered View history route. The failure state keeps the connection explanation, single not-cleared callout, and adjacent recovery actions. Current app chrome includes the existing bottom navigation; the global History tab remains the ordinary History route for failure state. The error composition is adapted to current app chrome rather than pixel-matched to the older selected screen.

Workflow Health isolation result: `installed=true` during the selected execution; `install_error`, `heartbeat_error`, and `clear_error` empty; `test_pool_requests=110`; `primary_during_test_mode_requests=0`; `primary_root_writes_during_test_mode=0`; `test_root_writes=0`; `refused=0`. The owner reported no clear error. The run does not establish a post-clear lease inventory beyond that owner result.

## Limits and independent-review handoff

- Parent review is pending. Review the E8 delta against the pre-write anchor, especially: removal of the error-state History footer while retaining shell History navigation; accessibility of the labelled section, focusable state headings, role=alert, button/link targets and dark/light tokens; Retry ownership and the mock assertions that it performs only a ListAsks refetch; and whether the compact typography and flat surfaces preserve the selected intent in both themes.
- E8 does not claim E7’s separate retry-to-Settings/browser client-call trace. The selected cases prove the states and capture geometry; they do not exercise Retry. The known Workflow Health `validation` artifact profile does not provide a trustworthy API/client-call trace, so no browser-level absence-of-mutation claim is made.
- G1 is evidenced by the final focused AsksPage run and the passing Test Genie unit phase. The combined workflow phase failed in unchanged E4 cases; retain that phase failure and do not repeat it with unchanged inputs. The selected Workflow Health pass is separate diagnostic evidence, not a Test Genie workflow-phase receipt.
- No E6/E4 acceptance, prior benchmark gate closure, or whole-redesign gate closure is claimed. The parent owns independent review and all acceptance decisions.

## Authored file hashes at handoff

| File | SHA-256 |
| --- | --- |
| `ui/src/features/asks/AsksPage.tsx` | `7c30ee14d04574754185305ae213d835d19bfca37bdff7a75186f50d1668564b` |
| `ui/src/features/asks/AsksPage.test.tsx` | `0b334c6074b4c052dd94f9cd9d5db2bba6426518ad65d71f6e53c2e665aa7baf` |
| `bas/cases/e4/asks-empty-dark.json` | `82cfd78498cd2f9a766ee9d6f1e88f119791ab7269d19d691a80fad567bb3270` |
| `bas/cases/e4/asks-empty-light.json` | `633de97f24a4f20e539e5e3e1137b64928b3d6bf6a47c6dda826e12fb181f695` |
| `bas/cases/e4/asks-read-failure-dark.json` | `01b44a22fd6b432c5de3cb8ed79c8a2e97a9fe617ae73cd4e797880887368732` |
| `bas/cases/e4/asks-read-failure-light.json` | `e2c3691482b311093cfeb999dd5bb05c816b69dfa921a60f34c2bd0404f9c18f` |

Actual meaningful work units: 6 — composition and consolidation; behavior test extension; registered-case wait correction; focused test and managed lifecycle validation; four-state current-build capture and visual comparison; evidence and parent-review handoff.
