# Current-source selected-case lease check (2026-10-21)

Parent: Agent Manager run `5d07e651-eaa6-48ec-89b4-5dc359059193` (task `790cb264-1a63-47eb-89e6-3b9ee3e16c23`), verified RUNNING. This is evidence for the existing E1 per-case isolation subcheck; it does not accept E1/Q1.

## Current-source stale-revision case

- Workflow Health selected run `75bb202a-a62c-49c5-83ea-50dfa5c861de`, execution `401fc9f8-cc9a-4e3d-9b7e-a4ce1b471c99`, parent matches the orchestrator, case `bas/cases/calendar/stale-revision-conflict.json`, idempotency key `planner-i18-d2-stale-current-20261021-0225`.
- Target: `personal-planner@shadow`, refreshed current build `sha256:0d23db3bd2d9fef78a62f93e63d1b21417d91757052663eaec323086aa79a084`.
- Case identity SHA-256: `645f5048c614766cef48aef0493d1cad9853939bcf3984e52fcb4f2b47c4eb92`.
- Workflow Health run state: SUCCEEDED; nested selected execution: FAILED, 1 executed / 0 passed / 1 failed. The user-visible create response was “The save outcome could not be confirmed. Check its idempotency key before retrying.” The case expected a durable ID immediately and then cascaded through missing event/editor state; stale-conflict winner preservation was not reached. Retain this as a failed current-source journey; the owning product/test boundary remains to be corrected.
- LeaseStats: installed `true`, lease ID equals selected run ID, 20 test-pool requests, 0 primary-during-test-mode requests, 0 primary-root writes, 0 test-root writes; no install/heartbeat/clear error on this run. Artifact checksum `sha256:e2a10edc7268d487baaaf6ce62e6174f86b594261f25a3146cdcae41ba161e0b`; timeline and latest JSON are under the Workflow Health run artifacts for this ID.

## Duplicate-submit case admission

- Workflow Health selected run `070c9a3c-fbb8-4265-a30b-bc97d091f942`, parent matches the orchestrator, case `bas/cases/calendar/duplicate-submit-once.json`, idempotency key `planner-i18-d2-duplicate-current-20261021-0232`.
- `validate wait` returned terminal Workflow Health state SUCCEEDED with nested execution REFUSED before any case ran (`executed=0`, `test_pool_requests=0`). The requested lease install was refused because another selected run was live at that instant; the provider's attempted cleanup of the uninstalled requested lease returned a lease mismatch. The refusal was active-lease protection, not evidence of a stuck terminal lease.
- Actual conflicting run `5e52fa19-f470-40d8-9b18-ba911adc3ae5`, idempotency key `planner-i18-d2-stale-current-20261021-0230`, started `2026-10-21T02:27:06.821Z`, completed `02:28:21.420Z`, exact parent. `workflow-health validate get` now shows terminal Workflow Health SUCCEEDED, nested execution FAILED (1 executed / 0 passed / 1 failed), same uncertain-create result and downstream missing-editor failures as the first attempt. LeaseStats: installed `true`, own lease ID, 20 test-pool requests, zero primary requests or writes and zero test-root writes; no install/heartbeat/clear error. Artifact checksum `sha256:a9f8bf672706df27d42eecbe6722360ef19295202c04b12163dd3c7a99761b09`; execution `08c01ac2-ab26-4fce-9e07-37c33ca89b60`.
- The duplicate-submit refusal timestamp preceded completion of `5e52fa19`; it did not demonstrate a stale terminal lease. Current test-pool occupancy after that run is unknown. No pool clear/reset or retry occurred.

## Disposition

The existing selected-case route exposes exact per-execution counters when a lease installs, so the historical aggregate-only limitation is narrowed. Two current-source stale-revision attempts each failed at uncertain create before the conflict/readback assertions; they both retained exact counters. The duplicate-submit run was refused while the second stale-revision run was live and is not evidence of a persistent lease-owner fault. D2 current-source coverage and authenticated cross-surface actor parity remain incomplete. Reconcile each started run by its exact returned ID and idempotency key before considering another selected case. Do not duplicate the stale-revision case, clear/reset a pool, or infer current occupancy from the old `5e70...` run.

The shadow lifecycle log route was also checked once and returned `open .../personal-planner@shadow: no such file or directory`; no app-side log evidence was available. Source review shows the UI already disables draft changes while create status is uncertain and offers idempotency-key status lookup. The failed ordinary user journey is the defect to resolve; whether the correction belongs in product behavior or the existing test contract remains open.
