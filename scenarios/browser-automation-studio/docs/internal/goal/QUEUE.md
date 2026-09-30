# BAS queue

Order from the operator decisions (D19, D20, D23, D25). One line per slice:
**enabling** marks a slice that unlocks others; deletion targets are what must be
gone at close. Line numbers are static estimates; the first epochs re-measure them.
Each epoch file lives in `epochs/`; the orchestrator moves a slice to Done with its
`ACCEPTED` line.

## Next

- E1, E2, E4, E5, E6, and E7 are accepted. E3 completed the first S2 tranche
  and E7 completed the BAS-FB-057 remainder. E6 completed S3 under BAS-FB-058:
  all V1 workflows found by scan were migrated, and the six GCT HTTP cases were
  preserved as non-workflows. E2 and E4 record Gremlins as unverified under
  BAS-FB-056.
1. **S1 — One session broker** (E10; enabling). `api/automation/session` is the only
   caller of driver session routes; delete direct `driver.Client` session calls
   in record-mode handlers, engine, drills, workflow service and the recovery
   reconciler, driver lease fencing (`session-decisions`, `in-flight-guard`,
   cleanup registry) and 3 of 4 page trackers. About −3.5k / +1.2k.
3. **S5 — One recording journal.** Init-script injection only; the Go generator
   is the only merge policy (delete `ActionMergeService`, `mergeActions`); the UI
   is a read-only projection. About −6k / +1.5k.
4. **S7 — One frame transport.** One driver→hub path with a latest-frame slot;
   one UI hook. About −1.5k.
5. **S8 — One AI engine** with server-owned navigation state. Delete
   `claudecode_navigator.go`, the ollama CLI client and the UI navigation
   command hooks (replaces the E01 navigation hooks). About −3.5k.
6. **S9 — One retention planner.** Fold the 5 retention policies (including
    `owner_cleanup.go` from E01) into `services/retention`, out of `package main`.
    About −1.2k.
7. **S10 — One config owner.** The API owns config and sends session options in
    the lease; retire driver `runtime-config.ts`. About −1.3k.
8. **DET — Detectability journey and row** (D23). Local fingerprint page plus a
    few real sign-ins with a persistent profile; stealth defaults in interactive
    mode. After S1.
9. **S6 — One timeline shape.** Proto `TimelineEntry` end to end; delete about
    8 of 10 converters. About −2k.
10. **S12 — Split the workspace controller** (`RecordingSession.tsx`). About −1.5k.
11. **S13 — One replay spec.** About −3k.
12. **DESK — Desktop portability epoch** (D22). When Linux is healthy and a native
    machine is available; macOS through minimouse occasionally.




Every refactor epoch also matures the tests of the modules it touches, under the
same rules (worker card), so test lines fall with runtime lines.

Module budgets (runtime lines, current → target): Session 10.7k→6k, Runtime
adapter 18.4k→14k, Transport 10.9k→7k, Recording 36.2k→24k, Profiles 6.7k→5k,
Workflow 40.8k→26k, Evidence 22.3k→17k, Workspace UI 29.6k→22k, AI 17.9k→11k,
Export 29k→22k, Diagnostics 7k→1k, Config 3.1k→1.8k, other 53k→48k.

## Parked

## Qualification findings

(Daily journey-run failures land here as queue items.)

- **J11 journey coverage pending.** J11 (AI navigation, stubbed model) moved to a
  driver integration test because the shadow driver reads the model gateway only
  from `AI_GATEWAY_URL` at startup. Restore a shadow journey once the stub gateway
  can be injected without restarting the shadow.

## Done

- **E9 — JX: journey expansion before session/recording rewrites.** ACCEPTED 2026-09-30T17:48:18Z (epochs/E9.md); added J03/J04/J13/J15/J17/J18 and reran the full shadow suite 11/11. Runtime stayed at 265,723; test/fixture/support +196. Weighted non-cache tokens currently 299,294 (receipt marked not final).
- **E8 — S4: schema-driven node editor.** ACCEPTED 2026-09-30T17:09:26Z (epochs/E8.md); all 35 per-node components removed, behavior preserved through one ActionNode including screenshot/AI/Navigate previews; final UI and 5/5 Linux journey gates passed. Runtime −5,438 vs 271,161 baseline; weighted non-cache tokens currently 238,576 across E8 workers, with Agent Manager marking receipts not final.
- **E6 — S3: remove the V1 workflow compatibility path.** ACCEPTED 2026-09-30T16:07:29Z (epochs/E6.md); six discovered V1 workflows migrated, BAS scan found zero V1 arrays, and six GCT HTTP cases preserved; compatibility modules absent and targeted symbol search clear. Runtime −329 vs −2,000 directional estimate; final-tree Go, CLI, UI, driver and 5/5 Linux journey gates passed. BAS Test Genie workflow-health remains unverified after a queued-provider timeout.
- **E7 — S2 remainder: remove handwritten action vocabulary owners.** ACCEPTED 2026-09-30T15:59:13Z (epochs/E7.md); all eight paths deleted, runtime −2,101 vs the 273,262 baseline (−399 vs directional −2,500), test/support net −607 (including +55 lines of generated-proto regression coverage); API, CLI, UI, driver and five Linux journeys rerun and passed.
- **E5 — Driver, UI and CLI test consolidation.** ACCEPTED 2026-09-30T13:37:42Z (epochs/E5.md); runtime +0, test −711; driver 130 focused tests, UI 59 focused tests including BAS-RH-J24/BAS-AUTH, and CLI `go test ./...` passed. Stryker is unverified.
- **E4 — Further Go test maturation.** ACCEPTED 2026-09-30T13:09:43Z (epochs/E4.md); one state-based test sleep retired (23→22), remaining sites audited as intentional; full Go and coverage gates passed, runtime net 0.
- **E3 — Action vocabulary generated from proto (first tranche only).** ACCEPTED 2026-09-30T13:02:50Z (epochs/E3.md); runtime −449, test/support +187; proto verifier, affected Go/UI/driver checks, and five Linux preservation journeys passed. Test Genie tidiness passed after in-scope cleanup. BAS-FB-057 says S2 is not complete; the named remainder above is still open.
- **E2 — Go test consolidation.** ACCEPTED 2026-09-30T11:52:12Z (epochs/E2.md); runtime +0 and API test −1,330 (69,837→68,507); `go test ./...`, standalone `go test -cover ./...`, and runtime inventory rerun passed. Gremlins is unverified under BAS-FB-056; further Go test maturation was completed as E4/T1-F.
- **E1 — Dead code and diagnostics ownership.** ACCEPTED 2026-09-30T04:02:56Z (epochs/E1.md); net runtime −6,640 and net test −2,178; all required acceptance gates rerun and passed.
- **E0 — Journey suite and local fixture site.** ACCEPTED 2026-09-29T18:35Z
  (epochs/E0.md) at 6/6; 12 work units, runtime +0. The 2026-09-30 cleanup
  replaced the runner with `pnpm test:journeys` (TESTING.md): 5 journeys (J01,
  J02, J07, J08, J23); J11 is pending above.
- **E01 — cleanup selectors, scheduler cancellation, AI navigation hook split**
  (pre-epoch-model work, landed 2026-09-29 after focused checks; S8 and S9 later
  replace parts of it).
