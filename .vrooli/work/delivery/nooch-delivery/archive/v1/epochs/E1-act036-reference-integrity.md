# E1 next slice — ACT-036 reference integrity

Parent run: `16d59037-b9a9-4442-8ae7-0c28921bd54b`; parent task: `27847f20-3191-47d9-b74e-76df6ef918f8`; selected effort/revision: `nooch-selected-delivery-2026-10-03` / `nooch-selected@2026-10-03:b013666f214be361`.

## Original criterion

E1 preserves ACT-036: deny cross-workspace reads, writes, and downloads, including nested foreign references. This is workspace data-integrity behavior under the approved ownerless public model. Do not reinstate authentication or claim confidentiality between anonymous users.

## Assignment

Audit the existing workspace-scoped Connect/API paths for persisted foreign references, concentrating on plan occurrences and selected recipe revisions plus portability/export/download validation. Establish a two-workspace fixture in existing in-memory/API test infrastructure; exercise foreign nested IDs through the real handler boundary, verify the request fails without disclosing foreign record content, and verify neither workspace's persisted rows were mutated. Include direct foreign-ID reads and any reachable write/download cases in the same matrix where current APIs support them. Keep tests limited to ownership relationships between records and workspace IDs; do not assert that one anonymous caller is isolated from another.

Add the narrowest missing regression coverage. Change runtime code only if an exact failing behavior is reproduced. Preserve normal same-workspace success, ownerless access, planning occurrence/revision invariants, portability behavior, and all other E1 gates. No schema, auth, dependency, UI, external provider, or primary-data changes.

## Validation and return

Run only focused API handler/repository tests for the touched owners; record exact test names and raw outcomes. Inspect the diff independently for unintended behavior. Report changed, verified, unverified, exact source/test hashes, and immediate-preimage line deltas where a retained baseline exists. No E1/epoch acceptance; the parent remains responsible for independent review. Do not read `FEEDBACK.md` or change E30.

## Worker

Parent `16d59037-b9a9-4442-8ae7-0c28921bd54b`; worker task `af0e8dee-22ad-4137-80ec-cf1bc98d66cd`; selected profile `4c19aef1-36a2-4cc5-8961-42648692883b`; proposal anchor `gcta-e3d272aff6ec`; selected effort revision remains unchanged. Follow `scenarios/prompt-manager/store/skills/packs/core/large-effort-orchestration/references/worker.md`. Work in the existing shared checkout and do not create another leader or destination.

## Slice log

2026-10-20T09:02Z | Parent reconciled completed worker run `7b5b0cf9-6502-478d-915e-dad0b864af26` / task `af0e8dee-22ad-4137-80ec-cf1bc98d66cd` against the retained immediate preimage and independently reran the focused foreign-reference tests. Worker terminal status COMPLETE/exit 0; parent selected-effort lineage remains verified. Exact final source hashes: planning handler `343dc2576fbfb3ab828d05e3e331d1ea11c757290a3df217135a7c9ccb1752e6`; planning test `161a9e971ba647bde31dfce1386100217b211702934e2934dd12b873b607f2c2`; recipe test `360f03f3e9282edc0c703d023ff0ad3c927b720be3716d07839a40ffcac12f5c`; portability handler `f00cd3a38a4acac9a6151c8cb82d7d2948c4e15297985961dd03403cd9aaaea4`; portability test `709801590af4158497bab85260b17fd6163bf69ac43bed2ded6dde10b9302b30`. Baseline manifest SHA256 `de44304d2193fc2a52b868d36bec791f6fa3544d1ddb195c34d3630d92f234ee`. `go test ./handlers/planning ./handlers/recipe ./handlers/portability -run 'TestPlanningConnectRejectsForeignRecipeOccurrencesWithoutMutation|TestRecipeConnectHandlerRejectsForeignReadsAndWrites|TestWorkspaceExportsRejectForeignPlanReferencesWithoutMutation' -count=1` passed all three packages; worker also recorded the broader focused package run passing. Immediate-preimage delta +136 runtime/+345 tests. Parent epoch-check: no flatline/overrun/growth trigger; legacy-gates warning only, acceptance unevaluated. ACT-036 bounded API work is verified, but broader E1 and independent acceptance remain open.
2026-10-20T08:53:15Z | Added two-workspace ACT-036 handler regressions; preimage tests reproduced unpinned-occurrence GetPlan and workspace-export leaks; validation covers occurrence/candidate IDs, captured revisions and pinned historical same-workspace reads | exit metric=focused planning/recipe/portability handler packages PASS | net runtime lines=+136 | net test lines=+345 | ack=-
