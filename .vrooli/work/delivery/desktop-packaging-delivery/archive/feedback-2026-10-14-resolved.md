# Resolved desktop-packaging feedback (verbatim)

### DPF-FB-020 — 2026-10-14T04:30Z — operator's Claude session — interactive launch fixed; dispatch E3 now

The Agent Manager → Web Console authenticated-caller fix is live, under the
operator's decision "A, OK to restart".
- Agent Manager holds a scenario-authenticator service principal for the
  `scenario-authenticator:web-console` audience. It was issued at
  2026-10-14T04:06:35Z, and the first exchange was at 04:11:13Z.
- Web Console's List now returns `stable_create_owner = agent-manager`.
- A live reserved Create plus typed input passed.
- Unverified callers are still refused before Create.
- All 37 web-console sessions survived the restart.

Details are in `docs/internal/IDENTITY-AUTHENTICATION-MIGRATION-LEDGER.md`
(2026-10-14 receipt). Dispatch capacity no longer counts attached sessions
(DPF-FB-018).

Action: admit one E3 worker now with the unchanged brief and worker card, and
carry DPF-FB-017 into its assignment. Your dispatch is also the first full
`run create` interactive proof of this fix. If it fails, record the exact error
in `WORKAROUNDS.md` and in the Handoff.

### DPF-FB-019 — 2026-10-14T03:45Z — operator's Claude session — E4 host setup done; interactive launch fix in progress

- The operator ran `sudo vrooli setup --scenarios scenario-to-desktop --include-optional`.
  It completed at 2026-10-14T03:11Z (`setup-3770390-1791947158532930682`).
  Verified afterwards:
  - `pkg-config --modversion webkit2gtk-4.1` reports `2.52.6`;
  - `/usr/bin/WebKitWebDriver` is present.

  Record this result in QUEUE.md and remove the E4 host-setup item from
  `## Needs operator`. E4 still waits for E3 acceptance.
- The E3 worker launch failure (`web-console stable-create identity unsupported`,
  scenario-qa `knw-1791946188667278785`) is being fixed now under the
  operator's decision "A, OK to restart". Agent Manager will get a
  scenario-authenticator service principal for the Web Console audience, and
  Web Console will verify it. Do not retry E3 dispatch until the operator's
  Claude session wakes you after the live interactive-run proof.

### DPF-FB-018 — 2026-10-14T03:10Z — operator's Claude session (operator-authorized) — capacity fixed: attached sessions no longer count; dispatch E3 now

The operator authorized the Claude session to unblock this goal.

- 18 dead attached opencode run records were stopped. Their processes were
  already reaped (DPF-FB-017). Two live opencode sessions were left alone.
- Root cause of the remaining block: Agent Manager's `MaxConcurrentRuns` check
  counted attached interactive sessions (operator terminals, this Claude
  session) as dispatch slots. Fixed in
  `scenarios/agent-manager/api/internal/orchestration/run_creation.go` via the
  new `CountCapacityByStatus`, which excludes `execution_mode=attached`.
  Regression test `TestOrchestrator_SlotEnforcement_IgnoresAttachedSessions`;
  the orchestration, database and repository suites pass. Agent Manager was
  restarted at about 03:05Z with the fix.
- Live state at 03:05Z: 2 dispatched runs (the Nooch and Planner leaders) plus
  attached sessions. Dispatch capacity is 2 of 10.

Action: when you count capacity, count only non-attached RUNNING or STARTING
runs, which is what Agent Manager now enforces. Admit one E3 worker now and
carry DPF-FB-017 (the leak from packaged-app agent sessions) into its
assignment. If `run create` still returns `CapacityExceededError`, record its
exact `Current/Maximum` in `WORKAROUNDS.md` and wake nobody else.

### DPF-FB-017 — 2026-10-13T23:30Z — operator's Claude session (operator-authorized) — packaged-app launches leak agent sessions; capacity cleared

The operator authorized the Claude session to unblock the goal on their behalf.
Findings and actions:

- **Root cause of the `concurrent_runs 30/10` refusal.** The 24 extra runs were
  attached opencode sessions whose owner was `desktop-app:…-demo/runtime` or
  `…-protocol/runtime`, which is the packaged desktop apps launched by E1
  smoke and measured captures between 2026-10-12 13:00Z and 13:55Z. Each launch
  started an opencode agent session that outlived the app. That is a product
  defect: a packaged app's bundled runtime must not start host agent sessions,
  and if it does, they must die with the app (see the F1 D5 process-lifetime
  work and the 2026-09-15 bundled-runtime host-env leak). Admit the fix into E3
  scope or file it to the owning scenario. Every future measurement matrix
  multiplies this leak.
- **Action taken.** `vrooli agent reap --orphaned` stopped 24 orphaned opencode
  sessions plus 5 other confirmed-gone sessions (about 1.4 GiB memory and
  5.2 GiB swap). Agent Manager still lists their run records as RUNNING until
  they are reconciled. The operator's Claude session is doing that with
  `agent-manager run recover <id>` under an operator grant.
- **E3 dispatch.** Re-check capacity, then dispatch the E3 worker. Per
  SUP-FOLLOWUP-20261013, do not stop unrelated runs or raise limits.
- **E4 host setup.** `sudo vrooli setup` failed on two defects that are now
  fixed in source and installed:
  - `agent_session_containment`: `systemd-analyze --user verify` had no runtime
    dir under root. Fixed in `packages/platform-go/validate.go`, with a
    regression test.
  - `webkit2gtk-driver`: the manifest had no `packages.apt-get` mapping, so it
    was reported Unsupported. Fixed in
    `internal/tools/webkit2gtk-driver/tool.json`, with a conformance test.

  The operator's earlier run also used `scenarios=none`, which skips
  scenario-to-desktop's optional WebKitGTK tools. The command that installs
  them is `sudo vrooli setup --scenarios scenario-to-desktop --include-optional`.

### DPF-FB-016 — 2026-10-12T00:55Z — operator's Claude session (check-in) — skill change: park and handoff now go through `effort` commands

`large-effort-orchestration` SKILL.md and `references/worker.md` changed on
2026-10-11 around 00:26Z. The relevant new text, verbatim:

> 4. **Park.** `agent-manager effort park <goal home> --run <your run ID> --timeout 1h`
> (it applies the park policy below; do not call `run park` directly).

> **Handoff.** `QUEUE.md` holds one `## Handoff` section (what changed, what was
> verified, what remains, the exact next action). Replace it only with
> `agent-manager effort handoff set <goal home> --file <path>` at each check-in
> that changed something; name every open FEEDBACK entry in it.

`agent-manager effort park --help`: "Parks the orchestrator … for up to 1h while
a direct child run is live, or up to 72h while ## Needs operator holds an item;
otherwise refuses (exit 4) with the admissible-slice rule."

What to do:
- Park only through `agent-manager effort park`.
- Write the Handoff only through `effort handoff set`, and name the open
  entries DPF-FB-009, -012, -013, -014, -015 and -016.
- After acting on DPF-FB-015, the proposal-recovery item leaves
  `## Needs operator`. Under the new policy that means you can no longer park
  for 72h: admit E1 and spawn its worker.
- DPF-FB-015 still governs this goal. The skill text at acceptance (§4.4,
  "run `git-control-tower proposal create` … cite the returned `gctp-…` ID")
  is best-effort here and never a gate.

Context: since the 72h timer fired at 2026-10-11T13:23Z, this orchestrator run
has woken every 15 minutes (about 45 times) and re-parked without re-reading
the goal home. It has not yet acted on DPF-FB-015.

### DPF-FB-015 — 2026-10-08T17:05Z — operator — commits are never a gate; accept F1, resume E1

Operator, verbatim, answering the F1 proposal-recovery "Needs operator" item:

> I don't want to commit anything yet, but I approve of everything else. I
> feel like if there are things that require a commit to continue then that's
> not right. None of our processes should rely on that since that is something
> in this project where I can commit at my own pace I'm the one that decides
> that it shouldn't ever be like a gate for anything.

What "everything else" refers to (from the operator's Claude session):

1. **Accept F1 now.** The `ACCEPTED` line lists the exact changed files and a
   suggested commit message. Add a `WORKAROUNDS.md` entry: "GCT proposal
   unavailable for F1: anchors `gcta-62dc30c5bad8`/`gcta-f0ee057b8a3d`
   captured after the work; strict builder refuses unchanged-since-anchor
   paths."
2. **Resume E1 immediately**, alone, per DPF-FB-014, with system-monitor as a
   reference scenario.
3. **Anchor E1 at its start** so that a proposal can be created later if one is
   wanted.

Standing rule for this goal, from this entry onward:
- **A commit, or the creation or approval of a commit proposal, is never a
  gate.** It never holds acceptance, admission, a resume, or any other step.
- The operator commits at their own pace and has not committed yet. Expect the
  tree to stay uncommitted across epochs.
- Proposal tooling (`anchor`, `create`) is best-effort bookkeeping. If it
  fails or is unavailable, log it once in `WORKAROUNDS.md` and continue. Do not
  park, and do not raise a "Needs operator" item for it.
- Never ask the operator to commit in order to unblock work. Remove the F1
  proposal-recovery item from `## Needs operator`.

### DPF-FB-014 — 2026-10-08T13:10Z — operator — F1 budget decision, E1-first, swap secrets-manager for system-monitor

Operator, verbatim, answering the F1 "Needs operator" item after a summary of
the options:

> That makes sense and let's go with your recommendations. I understand that
> like infrastructure fixes, you know rather than progressing the actual
> experiment we want to run is unfortunate but if they're good improvements,
> that's fine. I would say we probably should change what scenario we use in
> the experiment and cite those issues. You know that you've been running into
> maybe use system monitor instead

The recommendations the operator accepted (from the operator's Claude session):

1. **F1: option 2.**
   - Amend F1's `Kind` to `feature`, with `Expected size: 47`, the measured
     net runtime delta. Do not pad it.
   - Record the amendment reason in the brief: F1 adds retained capability
     (per-resource cache, strategy host-tool declarations, process
     containment). Its deletion list is met (D8 audit) and G1–G10 are met.
   - Accept per §4.4 once epoch-check confirms the gates under the amended
     kind. Then create the Git Control Tower proposal (FB-009).
   - Add a Lessons line: label foundation or root-cause repair epochs that add
     capability as `feature` when admitted.
2. **After F1, run E1 alone** until it produces the reference baselines
   (≥10 cold + ≥10 warm comparable samples each) and the hello-desktop
   ~6.2 s waterfall attribution. Keep E2, E3 and E4 parked until E1 is
   accepted. This goal has produced zero measurements after ~24 h. Measuring is
   the deliverable.
3. **Replace `secrets-manager` with `system-monitor` as a reference scenario**
   in TARGETS.md, and cite why:
   - secrets-manager needed a bundled Postgres artifact, which started F1 C1
     (DPF-FB-005).
   - It needed a secret-bearing export path and credential issuance, covered
     by F1 C3, DPF-FB-008 and DPF-FB-010.
   - Its desktop journey gate still fails on an unchanged pointer surface (F1
     G8 log).

   Facts about system-monitor (checked 2026-10-08):
   - `api: go_module`, `ui: pnpm_vite`, react-vite template.
   - Every dependency is optional and degrades gracefully: agent-manager,
     vrooli-autoheal and ollama (`required: false`). No database or secrets
     need bundling.
   - It is running live.
   - It has no prior desktop records, so E1 must prove its first current-source
     build.
   - Caveat for analysis: system-monitor polls host metrics itself, so
     `idle_cpu_pct` includes real app work. Note this in
     `packaging-baseline.md`. Comparisons between strategies stay valid.

   The secrets-manager work already done in F1 stays (operator: "if they're
   good improvements, that's fine"). Only its role as an experiment subject
   ends.
4. If a reference scenario blocks E1 on its own infrastructure again,
   substitute it from the TARGETS.md fallback list and record why. Do not open
   another foundations epoch for one subject without operator approval.

### DPF-FB-009 — 2026-10-07T22:55Z — operator's Claude session (check-in) — skill revision 28: commit proposals

`large-effort-orchestration` is now at revision 28. Changes, verbatim:

> Run `git-control-tower proposal anchor --effort effort:<slug> --epoch E<n>
> --scope '<scenario path>/**' --json`, which fingerprints files that are
> already dirty.

This runs at admission, step 2.

> Then run `git-control-tower proposal create --effort effort:<slug> --epoch E<n>
> --subject '<scenario>: <outcome> (E<n>)' --body-file <body> --path <file>…
> --run <worker run> --run <your run> --json`. The body states the outcome, the
> deletion-list result, the gates and the metric delta. Record
> `ACCEPTED <time> <summary>` as the final log line and cite the returned
> `gctp-…` ID; the operator approves the proposal in Git Control Tower.

This runs at acceptance, §4.4.

What to do:
- Run `proposal anchor` now for F1, and for E1, E2 and E3 when each resumes,
  with `--effort effort:desktop-packaging-performance`.
- Pass a `--scope` for every admitted path root of the epoch, not only
  `scenarios/scenario-to-desktop`. F1 also touches `cmd/vrooli-dist`,
  `packages/resource-deployment`, `resources/kokoro`,
  `scenarios/scenario-dependency-analyzer`, `scenarios/deployment-manager`,
  `scenarios/code-facts` and `scenarios/performance-health`, plus the C3
  identity owners.
- At each acceptance, create the proposal and cite its `gctp-` ID instead of
  a suggested commit message.

## Resolved

### SUP-CAPACITY-QUALIFICATION-20261014 — 2026-10-14T03:00:40.135291+00:00 — supervisor — OPEN

Current native RUNNING inventory contains 11 rows: 9 are EXECUTION_MODE_ATTACHED, 2 are dispatched codec-pipe original owners. Shared AM current source admission uses CountCapacityByStatus for Running/Starting and excludes attached sessions; unchanged attached-capacity regression and final native unit 20261014-024937-9072cd54 PASS. Serving build 0172797354f20ccb7d3fe67f30f1fa1cd05fc5450fe4531dee1a0846e3c934a6 was compiled after that concurrent capacity repair and shared identity correction. Raw RUNNING count is not dispatched occupancy. Read fresh RUNNING/STARTING execution-mode inventory under the native counter semantics, then resume one E3 worker through the SAME native task/profile/parent if eligible; native admission is authoritative and all lower provider/resource gates remain. Do not raise the 10-run limit or stop unrelated sessions. Source/serving/consumer identity recovery is verified; preserve E1/E2, E3 gates, DPF-FB-017 and E4 host prerequisites. Evidence: docs/internal/delivery-operations/2026-10-12/sources/execution-evidence/2026-10-13/current-dispatch-capacity-projection.json.
