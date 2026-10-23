# E1 answer/readback/queue-return handoff — seq38

## Result

E1 remains unaccepted. The focused UI behavior is improved and source checks pass, but the changed UI was not served and the operated journey was refused before shadow startup. No answer, fixture request, lease, or database write was attempted. The previous lifecycle refusal about `vrooli-events`/`tunnel-manager` is historical: the owner fix was present in current source and the live dependency was healthy. This attempt encountered a different exact lifecycle refusal, retained below. No alternate route or live restart was attempted.

This seq38 worker is Agent Manager run `9065b0d2-eaa3-4c14-9e41-b4ff04090b4c`, task `42f7ba70-d781-49e5-ae26-17ca11a27e10`, exact parent run `11175250-c50d-44e9-9af5-cd6eba79bab2` / task `5ea39682-bba6-4d35-ac6e-b5b20708d09b`; interactive session `01a15c11-1118-7130-b9fb-3c67421394e9`. At handoff capture the child run is still RUNNING, goal status active, and has no park producer/key. Its terminal result is therefore not yet available; the final Agent Manager run result must be attached after this worker turn terminates. Parent is PARKED on the separate E8 child run `3fc276d0-fd2e-4e4a-8fab-deee86ce6504`; this worker did not replace that wait handle or invent a producer/key.

## Baseline and source inventory

- Source revision at capture: `34515f0f501d511f65aa06c8e323620bf6d05891` (shared checkout; unrelated and existing Notification Hub dirty work remains unattributed).
- Exact bounded prewrite snapshot: `prewrite-original/manifest.json`, SHA-256 `ade8a5827e02877434e1f12b63d4802414088321401aafafdb0c549621fb6829`; 41 exact paths, 38 present and 3 explicitly absent. It includes relevant E1 API/UI sources and tests, the API binary, every `ui/dist` file, the E1 return ledger as read-only evidence context, and explicit absences for this handoff, a possible E1 BAS case, and the obsolete `AskDetail.tsx` candidate. Capture recipe follows the worker card; source-path list staged at `/tmp/e1-seq38-prewrite.paths`.
- Compared against that immutable baseline, only two admitted product paths changed: `ui/src/features/asks/AskPage.tsx` (SHA-256 `bba6f0aba2f43ca5aa9b195e1ee858b084d330461a471de76cc52f903c70b871` → `787dfca0c5cb5172866fa99d3200e7caa59ca15fc9f5e8f713baab9aa583bcb4`, +16/-17 lines) and `AskPage.test.tsx` (`f327adcc161fb7ec254ee843fdd5d9a471c9b995cd2c5839861160340874fff7` → `54b9ff0f7ca7038d872d723fbec3c6421794c17ad124d3d5c0c095dcf4abbe40`, +19/-4 lines). Net runtime -1 line and net test +15 lines, within the 300/300 caps.
- API, existing BAS cases/registry, `ui/dist`, and the captured API binary did not change in this task. No generated output was hand-edited or regenerated. No product case was added or removed; the exact `bas/cases/e1/answer-readback-queue-return.json` path remains absent.

## Behavior change and regression

A successful `Answer` mutation response previously displayed “Answer recorded” immediately and only then invalidated the query. The UI now starts its existing recipient-scoped `GetAsk` readback after the mutation and shows “Answer recorded” only when the returned state, exact answer key, and note match the immutable attempted payload. While that read is pending it exposes the existing “Checking status…” message. A lost response remains one-shot and uses the same readback path; no automatic Answer replay was added.

Focused validation:

- `pnpm --dir scenarios/notification-hub/ui exec vitest run src/features/asks/AskPage.test.tsx src/features/asks/AsksPage.test.tsx` — PASS, 2 files / 27 tests. The regression holds GetAsk pending, verifies no premature recorded claim, resolves the exact answer/note from readback, and confirms one Answer call. Existing queue selection/scroll/focus tests remain protected.
- `pnpm --dir scenarios/notification-hub/ui type-check` — exit 0; string freshness check and TypeScript check pass.
- `git diff --check -- scenarios/notification-hub/ui/src/features/asks/AskPage.tsx scenarios/notification-hub/ui/src/features/asks/AskPage.test.tsx` — clean.
- An initial focused run failed only because the new assertion used “Checking answer status…” instead of the existing localized “Checking status…” copy; the assertion was corrected and the final focused run passed.

Consolidation is limited to replacing the direct success-claim/invalidation path with the already existing authoritative readback reconciliation. `AsksQueue` separately owns queue filter, selected-row, scroll, and return focus; `askDrafts` owns subject/ask drafts; `HistoryPage` owns bounded closed history; the shell and routes own the five destinations. Those boundaries and their unique assertions remain distinct. No safe caller or test deletion was identified; no production caller was removed.

## Current owner state and exact refusal

- Current live Hub is healthy, managed operation `startop-c488cc034d08686ab77214cfe274178a`, build `sha256:244713c24dd987763b769f8b85834bae65050f749eeafc9c29d5e2f587420688`. Current `notification-hub@shadow` has no process or build identity. `vrooli-events` and `tunnel-manager` are healthy live instances. Current lifecycle source `internal/lifecycle/dependencies.go:348-354` clears the parent shadow's `VariantDependencies` and `Variant`; this matches the reviewed correction `f34d5f304b52cdf4c0677447e691b71e203c5b0fd50fa7e204b796ca471c71ae`, installed through normal `make install`. The managed attempt proceeded beyond the old variant-dependency refusal to the distinct artifact-sharing refusal below. This is infrastructure evidence only.
- One normal managed attempt was made with the required auth mode and instance: `VROOLI_AUTH_MODE=personal_local vrooli scenario start notification-hub --instance shadow --timeout 300 --json`. Operation `startop-5a9052dcf2f78f39f945c5f8ed030ff3` failed, CLI exit 1, before build adoption or process startup. Exact owner refusal: `notification-hub@shadow needs a rebuild, but it shares build outputs with running instance(s) notification-hub; rebuilding would replace the files they serve. Restart notification-hub first so the build is fresh, then start notification-hub@shadow`.
- Live Hub remained healthy after refusal. The lifecycle's suggested live restart was not taken; this E1 route stops at the exact refusal. `vrooli scenario deploy-report notification-hub@shadow --json` reports `changed: true`, with the AskPage package's two modified files. The changed source is not in the served build. Do not treat live build `244713…` as adoption of this patch.

The only current test recipient is the disposable test-pool subject `e1-fixture@notification-hub.invalid` (`hub.TestRecipientSubject`, seeded only in the routed pool). In the required `personal_local` mode, `LoopbackOwnerProvider` verifies the actual local OS principal as `osuser:<uid>` (`api/internal/identity/access.go`); test-mode reads resolve to the fixed fixture subject (`api/internal/hub/service.go`); `AnswerAsk` requires `in.Actor == record.Recipient` and returns `ErrNotFound` on mismatch (`api/internal/hub/asks.go`). No legitimate binding from that verified actor to the fixture recipient is present in the admitted sources. That is source-proven authorization insufficiency, not an Answer attempt or an observed Connect refusal. No grant, identity, recipient data, or route was synthesized.

Consequently there is no current E1 shadow build, loopback route, registered E1 answer case, selected Workflow Health/Test Genie run, lease ID, fixture-auth receipt, test-pool request count, primary-during-test count, file-root counters, protected readback, actor attribution, or operated queue-return observation. These counters are unavailable, not zero. No browser capture or answer was made. The old cleared E3 lease is not reused.

## Continuation recheck (2026-10-21 01:36:40 UTC)

- `vrooli scenario status notification-hub --json`: live instance remains healthy, build `sha256:244713c24dd987763b769f8b85834bae65050f749eeafc9c29d5e2f587420688`.
- `vrooli scenario status notification-hub --instance shadow --json`: zero processes, no build identity; operation `startop-5a9052dcf2f78f39f945c5f8ed030ff3` remains failed with the identical shared-build refusal above. This is a read-only status recheck, not a retry.
- `agent-manager run get 9065b0d2-eaa3-4c14-9e41-b4ff04090b4c --json`: child remains RUNNING, observed goal active, no terminal result or park producer/key.
- `agent-manager run get 11175250-c50d-44e9-9af5-cd6eba79bab2 --json`: parent remains PARKED on E8 run `3fc276d0-fd2e-4e4a-8fab-deee86ce6504`. Seq38 did not replace its wait handle.

## Owner epoch-check handoff (2026-10-21 01:38:47 UTC)

Command: `agent-manager effort epoch-check scenarios/notification-hub/docs/internal/goal/epochs/E1-answer-navigation.md --runs 9065b0d2-eaa3-4c14-9e41-b4ff04090b4c --wake-key 11175250-c50d-44e9-9af5-cd6eba79bab2 --json` — exit 3 (`STEP_BACK`). Exact fired trigger: `overrun: 23 log lines against 20 (2.0x estimate 10)`. Flatline and growth did not fire; spend was not evaluated; wall-clock was review-only. The owner reported legacy/untyped gates and did not check acceptance. It reported parent run `11175250-c50d-44e9-9af5-cd6eba79bab2` in `woken`; an immediate read-only `run get` then showed the parent RUNNING at 01:39:00Z. This is the required worker-card handoff signal, not E1 acceptance or a runtime proof. No epoch/goal ledger was edited. The child Agent Manager `run result` endpoint had no final run result before this handoff; the exact terminal result for this bounded worker step is `epoch-check` exit 3 / STEP_BACK above. The orchestrator now owns the next serialized action and must preserve E1's open gates.

## Source/state joins and preserved gates

Current source identity at HEAD is listed above. Required design/source hashes at handoff:

- `BENCHMARK.md`: `45580f33ca453f0331c5ad50dd50e74edec6f54a6e327c457573b42f1a889325`.
- Dossier: `d147181b1e7ba3b0f1dcaa06c2f2805b45761e4eda621231d217aa597eb17fdc`; atlas: `5893b198f9bb9c9851fd50cc16df1607fb4da5e6a00f27450d61b8d5269ddeb4`.
- Selected image manifest: `4365502f3ee6aac508730fbdb44ad3a425a193a3c47282ebfbc82bbcb82953ef`, retaining all 240 references, 102 compositions, and 71 states. Commissioning source map: `ebce0298fe604dcb9e1d1df887af7975a02d8d5f13e682ee6d5db267d6ded28d`.
- E1 mapping: NH01/Needs you and NH18 queue entry map to `AsksPage.tsx`/`AsksPage.test.tsx`; NH14.1–NH14.10 map to `AskPage.tsx`/`AskPage.test.tsx` (NH14.3 now explicitly waits for matching authoritative readback); NH13.1/NH13.3 session/deep-link boundaries map to `SessionGate` and ask tests; queue return focus/scroll is covered in `AsksPage.test.tsx`; five destinations are preserved in `routes.tsx`/`AppShell.tsx` and their tests. API recipient scope, atomic answer, exact answer validation and actor attribution remain protected by `module_read_test.go` and `api/internal/hub/asks_test.go`.
- Applicable E1 obligations remain NH-AC-20, 22–27, 34–37, 40. NH-DEC-26 authority is unchanged. The five destinations (Needs you, Updates, History, Delivery, Settings), A1–A10, later states, all 240/102/71 selected references, and original deadline `2026-10-21T03:48:04Z` remain in scope; no benchmark or redesign gate is accepted here.

Existing changed-producer UI Health run `20261021-011625-5224f968` remains FAIL with `standard_shell_ownership` at `ui/src/layout/AppShell.tsx:52 <nav>` and browser-only native-screen/zero-inset limitations. It was not rerun. This AskPage source change changes the UI inputs; parent should select a fresh relevant UI Health phase after a supported build adoption, without duplicating a validation owner. The source hash and blocked serving identity are recorded above.

## Unrun gates and parent return

1. G1 remains failed on the retained UI Health receipt; a new relevant phase is needed only after source adoption.
2. G2 remains unmet: managed shadow build adoption was refused; fixture authorization is insufficient; no fresh lease, safe route, answer, readback, attribution, or queue return exists. The route must not be substituted or retried unchanged.
3. G3 exact-path comparison is complete for this task. Existing dirty work outside the two source/test deltas retains unknown attribution.
4. G4 source/state/test mapping is recorded here; matching selected-state captures and independent visual/code/architecture review remain outstanding.
5. No Test Genie API/unit/workflow rerun, selected E1 browser case, visual capture, or benchmark gate acceptance is claimed. Parent retains the next owner decision; E1 and all whole-redesign obligations stay open.
