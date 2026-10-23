# E15 worker handoff

Parent: `11175250-c50d-44e9-9af5-cd6eba79bab2`  
Task/run: `7a4fbfb9-05d9-4a57-9e99-46c2b1db4db6` / `cf481f97-272b-4f1c-b42d-dbfa6ead94cf`  
Epoch: `epochs/E15-history-selected-state-fidelity.md`

## Change

Moved History's clear-filter action out of the main content stack into a dedicated bordered footer band using the existing app surface tokens. Clear-filter behavior and routed test-mode preservation are unchanged. Added an assertion that the action is presented within the footer. Existing bounded 200-row filtering, local query behavior, incomplete-account copy, keyboard-operable outcome filters, and five destination routes remain covered by the focused page tests. No API, auth, lifecycle, localization, generated string, shared component, shared token, or BAS assertion changes were made.

Exact current source hashes:

- `ui/src/features/asks/HistoryPage.tsx`: `5b7b31ed8a84136345174044ab12718bd5d5c5a4018fe883277c959a8c598c26`
- `ui/src/features/asks/HistoryPage.test.tsx`: `e0eb89cd7ec00c9d7b411bf5b1fe4c133297fb5ca577d372470235b344497eb0`
- `ui/src/styles.css`: `beeff1fc0ea65d10f34be7a2b8956cf9d636221f4abf58a58b4451ea1817b1b0` (preexisting E14 shared-shell/safe-area work; not edited by E15)

## Focused validation

- `pnpm exec vitest run src/features/asks/HistoryPage.test.tsx`: PASS, 1 file / 5 tests.
- `pnpm exec eslint src/features/asks/HistoryPage.tsx src/features/asks/HistoryPage.test.tsx`: PASS.
- `pnpm type-check`: PASS; generated strings check and TypeScript check completed.
- `vrooli scenario freshness notification-hub --json`: stale UI build, cause `build input changed`, file `shared_package:@vrooli/proto-types:generate`; API freshness is clear. This is the exact normal-owner blocker to a fresh managed capture. No build, restart, Test Genie rerun, or local-render substitution was used.

## Visual identities and remaining owner boundary

Selected target originals are NH16.1 “No matching loaded requests”, 390×844 CSS px at DPR 2 (780×1688 PNG):

- Light: `docs/internal/delivery-operations/2026-10-12/assets/notification-hub-revised/originals/NH16.1-light-phone-p1-r3.png`, SHA-256 `dd0f265a97772e573d8dd00d243f7a6e3291da9d49fee43c32eee4c14e962313`.
- Dark: `docs/internal/delivery-operations/2026-10-12/assets/notification-hub-revised/originals/NH16.1-dark-phone-p1-r3.png`, SHA-256 `1febf9bf5328bfc55f4e3fd6347f4d5c259473875431fa61a11403724855648e`.

E14 comparison pair from `evidence/E14-worker-46b2766d/D3-final-622-images/`, managed build `sha256:62204327e9ff6923b14a58c6eb2c61f3aa63068c6c852953ea65b6b9caf36dde`:

- Served light SHA-256 `1127733c961c62c833eee300684ff3266fcfb566e9f086d9d58df252a7b46fcb`.
- Served dark SHA-256 `8e75099f4ece938f41bcda6fca9feda04ffc35c3bcbd61d5f46916db117663ea`.

The originals' hashes were rechecked. The footer band is implemented, but fresh matched post-E15 served captures are unavailable because shared Proto freshness is unresolved. Visual acceptance is therefore unverified. The broader dark surface palette is owned by Notification Hub's shared theme/design-token layer (`ui/src/design-tokens.css` and its consuming shared shell), which this bounded slice explicitly excludes; route-local overrides would create a competing theme owner. Parent should route that exact surface mismatch to the shared theme/token owner. Keep the safe-area inset and five persistent tabs from E14. No redesign or benchmark-gate acceptance is claimed.

## Native epoch check

Command: `agent-manager effort epoch-check scenarios/notification-hub/docs/internal/goal/epochs/E15-history-selected-state-fidelity.md --runs cf481f97-272b-4f1c-b42d-dbfa6ead94cf --wake-key 11175250-c50d-44e9-9af5-cd6eba79bab2 --effort effort:notification-hub-redesign --json` — exit 0. Result: 4 work units against estimate 3; no step-back triggers; no open directives; no flatline, overrun, growth, or wall-clock trigger. The nonblocking `legacy-gates` warning notes that this epoch has prose exit criteria but no typed `G<n>` entries; no acceptance claim was made.

Native queue handoff was written with `agent-manager effort handoff set scenarios/notification-hub/docs/internal/goal --file /tmp/notification-hub-e15-handoff.md --json` (changed=true). The queue keeps the parent-owned E15 item active for reconciliation and independent review.
