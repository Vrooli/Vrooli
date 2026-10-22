# Independent review — authenticated ownership for new native calendar events

## Native provenance

- Reviewer task: `2c6c520f-2ff3-4a23-a2ad-2553b2d498bf` (completed)
- Reviewer run: `c68c8e90-97c8-4671-bc49-47271296d6b8` (complete, exit 0)
- Reviewer profile: `b93d5233-4867-4569-8371-171bea404b7e` (`prompt-manager/heartbeat-judgment`)
- Exact parent run: `5d07e651-eaa6-48ec-89b4-5dc359059193`
- Native approval: `APPROVAL_STATE_APPROVED`, approved by `applyAtRunEnd`
- Selected final candidate: `bcb7aee8-65c9-49e6-84e7-5f0ce1eef82a` (selected, unique terminal main assistant)
- Worker task/run: `cbad6395-7d01-4c2f-8e2b-ad41de6a471f` / `a7df133f-03ca-4217-b451-aa3dac6c8445`, complete, exit 0, exact parent above
- Baseline manifest SHA-256: `01f6871225b82ef6eb5f377dda323c52e98c5ada70aba689c75e4960331b6efa`
- Current manifest SHA-256: `08b52d0247314345d319d065529b0b7e76f50f89d1b3c048cb75f8169b090127`; all five admitted current file identities match this manifest.
- Focused gate: `focused-gate.log`, SHA-256 `8230703ff6b41e4a523709e10ba4cbe1cafe9b661c267324e9caa7eac2a80e5c`, exit 1.
- Parent mechanical check: `parent-epoch-check.log`, SHA-256 `a7a7993262d3a47ce0817227bf854574c8bf093ee038cd9f48c0aea0daf89e79`, exit 0. It reports `ACCEPTANCE not checked: legacy exit gate`; it does not accept this slice or E1/Q1.

## Selected reviewer output (verbatim)

## Verdict: PASS WITH LIMITS

At source level, the five-file change implements the bounded new-event ownership behavior. It reads ownership from `identity.PrincipalFromContext(ctx)` and `Principal.OwnerSubject()`, and the SQLite paths scope list, get, idempotency replay, create, and update to that owner. Creation writes the event, owner relation, and idempotency record in one transaction. The additive `calendar_event_owners` table has no backfill, so legacy rows remain unowned; owner-filtered reads and updates leave them inaccessible.

The change matches the approved Planner contract’s requirement for per-record authorization. The recorded diff is within the admitted paths: **+53 runtime lines and +200 test lines**, exactly the slice’s test-growth limit; cumulative E1 runtime growth is **+1,784 of 2,100**.

The retained tests cover verified synthetic principals, same-owner idempotent retry, cross-owner idempotency-key reuse, owner isolation for list/get/update, and access to an unowned legacy fixture. Source inspection also shows provider-identity collisions with an unowned row or another owner return a conflict without adopting the row.

**Limits:** the retained tests do not directly exercise a provider-identity collision against an event owned by a different principal. The legacy-row regression checks its title remains unchanged, rather than asserting every field. These gaps limit what the test evidence establishes; they do not reveal a source-level defect in the inspected paths. This review did not rerun tests.

## Gate and acceptance

The focused gate **failed**. Per the retained log, `internal/calendar` passed, and the handler test reached its owner assertions before the supported Planner CLI build failed at protected `cli/domains/goals/handlers.go:180` (`*int64 / 100`). That recorded failure is outside this slice’s write set, but the required focused command did not pass. Therefore this verdict is a **source-level PASS WITH LIMITS**, not acceptance of the slice. E1/Q1 acceptance remains open.

Live UI/CLI actor propagation, full actor/workspace parity, and the original E1/Q1 requirements remain open, as directed. No legacy mapping decision or record assignment is inferred.

## Orchestrator disposition

Not accepted as a slice: its required focused gate exited 1. The independent source review is PASS WITH LIMITS; the parent mechanical check exited 0 but explicitly did not check the legacy acceptance gate. Preserve the implementation and all limits. Do not infer full actor/workspace parity or E1/Q1 acceptance.

## Later gate reconciliation (2026-10-21)

The earlier failed focused gate was caused by the protected CLI compile error and then the inherited agent-identity credential in the synthetic subprocess. The separately bounded compile correction and CLI harness correction address those blockers without changing the new-event owner implementation. The exact required focused command subsequently passed both `internal/calendar` and `handlers/calendar`; raw output is retained at `../e1-cli-actor-propagation-20261021/worker-evidence/focused-calendar.stdout` with SHA-256 `866a80ff667d0d3cadb4091ad32e0cadc70c22b2a4a5d297d35b68d1c98882a9`. Parent `epoch-check --acceptance --runs a7df133f-03ca-4217-b451-aa3dac6c8445 --wake-key 5d07e651-eaa6-48ec-89b4-5dc359059193` exited 0, log SHA-256 `410f52817e4f6056955608ae74acdfc059fa3c78cf8ada7ae2c975c4e725e103`; it reports no step-back but warns that E1 has no typed gates and full E1 acceptance remains unchecked. The bounded new-event ownership slice is accepted with limits based on its prior independent PASS WITH LIMITS, the now-passing focused gate, and the parent mechanical check. Existing reviewer limits remain: no direct provider-identity collision test, the legacy fixture assertion is limited, and reviewer did not rerun tests. No full actor/workspace parity, legacy-row assignment, E1/Q1 acceptance or other original gate is implied.
