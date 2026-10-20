# E1 same-ID live UI/CLI/Connect journey — 2026-10-19

## Result

The corrected retained-lease journey completed against the current Planner **live** API/UI pair. The browser created and revised two synthetic calendar records; the supported Planner CLI read those exact durable IDs and revision 2 through its generated CalendarService Connect methods. The independent E1 judgment and product acceptance were not changed. This receipt does not complete E1/Q1.

## Target and source qualification

- Planner runtime: `personal-planner@live`, healthy; managed build identity and API `/health` both reported `sha256:701de714370b0018b4c56dafafc0db303327e66fd97e03688bf8b1092f819049`.
- API/UI ports: `17604` / `20003`; BAS `ScenariosService.GetPort` resolved `http://localhost:20003`, matching the live UI port.
- Test Genie eligibility before each attempt: `routed:true`. The retained seed apply installed the one routed lease on the same live API; the BAS target profile carried `X-Vrooli-Test-Mode: 1`. BAS was invoked without seed flags. The actual same-ID rows appeared through UI and the real CLI before cleanup.
- The registered case source matched the reviewed preimage `fd20cf563e750889aa9c9e61a08834e4fe2fdaed8cba7796af8957e4846c4a70` on the first attempt. After the demonstrated seed collision, only the existing case was corrected: empty-month navigation now uses an unseeded prior month, and reopen actions select the created event by title. Assertions and DST dates were preserved. Corrected case SHA-256: `077f7122ef74365b2504ccac4ff2d200b7edea6854436b8c29ddb12d490e5e08`; `jq empty` passed.
- BAS execution parameters: `browserProfile.extraHeaders = {"X-Vrooli-Test-Mode":"1"}`; no lease ID header was fabricated.

## First attempt and cause

Test Genie retained seed run `a5ca94ab-e326-4f6b-a24c-c1a3d24f70e9`. BAS execution `fa15bb77-a97f-4e91-b157-2f8c41d8e14b` ended `EXECUTION_STATUS_FAILED`: the case expected empty November 2026, but the existing seed had already placed `E1 isolated CLI DST fold` on Nov 1. The unqualified generic timed-event selector then reopened that seed event rather than the new UI event. CLI evidence showed the mismatch: UI-created `964594a6-c28b-4e21-a6ec-67ea601c672d` remained revision 1 while a different seeded row `77559983-652f-4462-ac42-690c588ac4d1` became revision 2. This run is preserved as a failed result, not counted as same-ID proof. Its exact-token cleanup returned `status: cleaned`.

## Corrected journey

Test Genie retained seed run `2fd3388c-1fba-4582-a58d-45e626fe75c4`. Native BAS `execute-adhoc` execution `c4368bec-9f3e-4b0d-87f9-2610ebc6ff33` ended `EXECUTION_STATUS_COMPLETED`. All 38 timeline entries are `STEP_STATUS_COMPLETED`, including the original empty-month assertion, multiday reopen/revision, and DST-fold reopen/revision checks.

| Surface assertion | Durable ID | UI-reported revision | CLI/Connect readback |
|---|---|---:|---|
| Multiday event, title after edit `E1 multiday revised`, 2026-10-01 through exclusive 2026-11-03 | `24381c28-02c8-45aa-93c1-2cfdf42614d8` | 2 | Same ID, revision 2, same title and civil-date bounds |
| New York DST-fold event, title after edit `E1 DST fold revised` | `c988e20f-81c6-424f-a27d-87fe7ae2328a` | 2 | Same ID, revision 2, `2026-11-01T01:30:00-04:00` through `2026-11-01T01:30:00-05:00` |

CLI list showed the separate seeded Nov 1 row (`5636083f-7a66-442e-a111-6fa27373c747`, `E1 isolated CLI DST fold`, revision 1) alongside the UI-created timed event, confirming the corrected selector did not edit the seeded record. Exact CLI get-event reads for both UI IDs succeeded against `http://localhost:17604` with `PERSONAL_PLANNER_TEST_MODE=1` and `--instance live`.

The exact-token cleanup for seed run `2fd3388c-1fba-4582-a58d-45e626fe75c4` returned `status: cleaned`. Test Genie cleanup fails closed on primary DB/file writes, so the successful cleanup is zero-primary-write proof. The Test Genie seed receipt does not expose numeric test-pool request or test-root-write counters; those values remain **unavailable**, not zero.

The BAS execution exposes the target BrowserProfile header and case context, but this receipt did not emit an authenticated actor ID or workspace ID. Do not infer identity parity from the event IDs; the original actor/profile/workspace gate remains subject to its existing independent evidence/review requirement.

## Retained raw evidence

Private owner receipts (directory mode 0700; the seed apply files contain cleanup tokens and are not copied into shared docs):

`/home/matthalloran8/.vrooli/state/vrooli/agent-manager/runs/58257550-4062-40d8-aa7d-ffa4f01d279f/runtime/codex/receipts/planner-live-same-id-20261019/`

This directory retains both preflight command outputs, both complete Test Genie apply/cleanup stdout/stderr/exit receipts, both complete BAS terminal outputs and full timelines, the exact parameters, case preimage, and each CLI command's stdout/stderr. The corrected run timeline and complete source/terminal receipts are retained there.

## Remaining E1/Q1 status

This qualifies the integrated browser-to-CLI/Connect same-ID and revision portion for these two calendar events only. It does not qualify the accepted goal/project/milestone/work/actual/completion/forecast/review journey, per-case LeaseStats numeric counters, backup/restore for product-data writes, or independent reviewer acceptance. The original review remains `NOT ACCEPTED`; F8 remains `NEEDS_REWORK` with missing historical 78-file attribution; no review decision, gate, or acceptance was changed. R27 app-wide lag remains open; this journey makes no responsiveness claim.

BAS actual navigation URL recorded in the corrected execution timeline: `http://localhost:20003/plan`.
