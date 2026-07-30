# Testing

Project-level testing guidance is intentionally thin. The canonical rule is:

- use `vrooli scenario test <name>` for scenario suites
- use the relevant Go or package-level test commands for the platform code you changed
- prefer current CLI surfaces and maintained fixtures over shell-era ad hoc test flows

For mutating BAS/E2E workflows, follow the two-engine
[routed test-storage contract](agent-system/routed-test-db.md): SQL and file
writes must be leased and test-mode routed before execution.

## Start Here

- [reference/cli-commands.md](reference/cli-commands.md)
- [reference/health-maturity-assessments.md](reference/health-maturity-assessments.md)
- [scenarios/VALIDATION.md](scenarios/VALIDATION.md)
- [../scenarios/test-genie/docs/QUICKSTART.md](../scenarios/test-genie/docs/QUICKSTART.md)

## Common Commands

```bash
vrooli scenario test <name>
go test ./cmd/vrooli/... ./internal/...
make hygiene
make validate-package-governance
```

Use the smallest validation surface that honestly covers your change.

## Tiered confidence for agent behavior

Agent behavior requires more than a replay fixture, but a live agent run is
not the default unit-test loop. Use these rungs in order:

1. **Tier 1 — deterministic/unit tests.** Exercise codecs, policy, persistence,
   and HTTP contracts using fakes. These are fast and mandatory for changed Go
   behavior.
2. **Tier 2 — replay and recorded corpus.** Re-run representative provider
   transcripts and wire fixtures to detect parser/regression drift without
   incurring model cost.
3. **Tier 3 — bounded live smoke.** Run one deliberately tiny real-agent
   exercise only when the behavior crosses a live boundary that fixtures cannot
   prove (for example, sandbox apply plus provenance). The command must have a
   deadline, print each assertion, leave no intentional scratch artifact, and
   fail nonzero if an assertion is false.

For Agent Manager's tracking/provenance boundary, `make smoke` is the Tier 3
entrypoint. It makes one sandboxed tracking run, verifies terminal completion,
`changed_files > 0`, and an applied Workspace Sandbox provenance row, then
removes its uniquely named scratch file. It selects an unrestricted code
profile, preferring `code.cheap` when it has the required write capability; use
`agent-manager scenario-smoke --profile-id <id>` when that choice must be
explicit. This is a real model invocation and therefore incurs cost;
do not use it as a retry loop. A deliberately wrong endpoint (for example
`--workspace-sandbox-url http://127.0.0.1:1`) is an intentional negative check
and exits nonzero at the provenance assertion.

For provider-specific health maturity, use the provider's default human CLI output. Reserve `--json` for Test Genie and other automation.

## Waiting on runs (for agents)

The run is owned by the test-genie server, so it survives your command being cancelled. Just run it — the run id + a re-attach command are printed up front, and a known-long run auto-backgrounds so your shell returns immediately. `vrooli scenario test <name>` is a direct entry point for `test-genie --auto-start execute <name>`.

**Do NOT poll with repeated "still waiting" checks. To wait, block ONCE with the quiet wait verb:**

- `test-genie runs wait --json <scenario> <run-id>` (also `vrooli scenario test wait <scenario> <run-id>`). It blocks server-side and returns exactly once with the verdict + the run's real exit code (`0` passed, `1` failed/aborted, `124` if you pass `--timeout` and it elapses first). It does NOT stream — one call, one return. This is the verb the start banner and re-attach commands print; copy it verbatim.
- **A quiet wait announces its attachment on stderr immediately.** Its JSON stdout deliberately stays empty until the terminal snapshot. If a coding tool/session returns after that attachment receipt but before terminal JSON, the session detached; it does **not** mean the waiter exited or the run is stuck. Read `test-genie runs status --json <scenario> <run-id>` once for durable state, then follow its typed action rather than treating the attachment loss as a blocker.
- `test-genie runs status <scenario> <run-id>` is a snapshot, not a polling loop. While pending, human output prints the same one-wait command exactly once and JSON adds a typed `nextAction` (`kind=wait`, command, timeout, `doNotPoll=true`). Terminal status omits it. The `vrooli scenario test status` proxy has the same behavior.
- **A terminal wait is a durable run read, not a reduced status summary.** For a terminal run, `runs wait --json` and `runs show --json` must project the same run id, verdict, phase set, per-phase statuses, and durations from the canonical persisted run snapshot. This invariant must still hold after the live run retires from memory and after Test Genie restarts. Missing or corrupt historical fields are explicit degraded/UNKNOWN evidence, never zero-filled success-shaped data.
- **Run evidence is catalogued and path-free.** New runs atomically persist a typed artifact catalog over their existing run-owned files. Consumers list/filter evidence by open kind strings (not phase names) and fetch bytes with a run-scoped opaque artifact ID; provider relative/absolute paths never cross the API boundary. Historical runs without a catalog use explicit degraded legacy discovery. Missing bytes, unsafe symlinks, cross-run IDs, and incompatible catalogs are errors, never absent evidence interpreted as PASS.
- **If you must bound the wait, use `--timeout=<seconds>`.** On timeout it returns `124`, the JSON snapshot still carries `recommended_next_check_seconds`, and stderr prints the exact re-invoke line. **Re-call only after that many seconds — never poll faster, never re-run immediately.**
- `test-genie runs follow <scenario> <run-id>` is the **human** live-watch verb (a continuous, heartbeating stream). Do not use it to "wait" as an agent — a backgrounded stream re-wakes you on every heartbeat. Use `runs wait --json`.
- Cancel ≠ abort — to actually stop a run use `vrooli scenario test abort <scenario> <run-id>`.

**One run per scenario at a time.** The test-genie server allows at most one in-progress run per scenario (different scenarios run concurrently). An identical re-request coalesces onto the running run (no second suite); a *different* request for a busy scenario is rejected with the in-flight run id + `runs wait --json`/`runs abort` guidance — wait or abort, don't retry-spam.

**Waiting on several runs at once?** Use one `test-genie runs wait-all --run <scenario>:<run-id> --run …` call (repeatable; add `--json`). It blocks until every named run is terminal and returns one aggregate exit code (`0` all passed, `1` any failed, `124` any still in-flight at `--timeout`, `2` any not-comparable) — so two parallel suites/diffs resolve in a single call instead of two backgrounded streams.

**Scenario starts have the same contract — don't poll them either.** `vrooli scenario start|restart` write a durable start-operation record; to wait on one, block once with `vrooli scenario wait <scenario> --json [--timeout N]` (exit `0` healthy, `1` failed, `2` degraded, `124` timeout with the start unaffected). Full protocol: `docs/reference/cli-commands.md` ("Scenario start wait contract").

**Baseline diff is durable too — don't poll it.** `git-control-tower baseline diff --scenario S --name N` returns immediately with a run id + re-attach command. It reuses a prior current result only when that scenario's declared source fingerprint and validation configuration still match; otherwise it runs the tests again. Ordinary dirty or unrelated workspace files do not invalidate the baseline. Resolve the verdict with `git-control-tower baseline diff status --scenario S --name N --run <run-id>` (exit `0` clean, `1` regression, `2` not-comparable, `3` not-ready), or add `--wait` to block server-side and print it inline.

**Baseline snapshot is re-attachable.** `git-control-tower baseline snapshot --scenario S --name N` returns as soon as the server owns the run. Resolve the manifest write with `git-control-tower baseline snapshot status --scenario S --name N --run <run-id>` (exit `0` ready, `2` missing/failed, `3` pending), or add `--wait` to block server-side. If a later `show`/`diff` cannot find the manifest, use `snapshot status` first; it distinguishes pending/failed snapshot intents from a wrong baseline name and prints similar-name hints when available.

**Baseline collections use the same standing contract.** `baseline collection
show` and `baseline collection diff status` are pure reads that return the
typed `OperationStanding` with `wait`, `inspect`, or `recover`; they never
decide completion from client timing. `baseline collection show --wait` and
`baseline collection diff wait` are the only blocking attachments. A bounded
wait exits `124` after emitting its detached standing and exact reattach
command; cancellation never rewrites durable state or becomes success.

**Freeze each pending member's captured source scope.** A queued collection
child is normal server-owned work, not permission to continue implementing in
that scenario. Editing files in a pending member's declared source scope makes
strict provenance reject the immutable capture: Test Genie recorded the
before-state fingerprint at start, and the later evidence would otherwise
describe different source. Wait for the collection's terminal result first, or
work only outside every pending member's source scope.

Start a collection comparison with `git-control-tower baseline collection diff
--name <collection> --operation-id <stable-operation-id> [--member <scenario>
...]`. The operation id is an idempotency key: retain and reuse it to inspect or
reattach the same operation; do not invent a new id after an interrupted start.

**Plan Manager delegates; it does not wait.** A Plan Manager execution first
renders a Git Control Tower baseline-collection capture command. Run that exact
producer command, use the wait/recovery command printed by Git Control Tower,
then run `plan-manager exec baseline-sync <execution-id>` once. Phase and final
validation follow the same shape: `plan-manager validate start …` creates a
ticket and prints exact producer argv; wait through Git Control Tower/Test Genie,
then run `plan-manager validate sync <operation-id>`. Never replace those waits
with Plan Manager `wait`, shell polling, or a guessed timeout. A phase can use a
captured-member subset, but final Definition of Done omits selectors and covers
the entire captured collection.

All long validation operations follow the same ownership rule: persist intent and an operation id before dispatch, separate queue/execution/transport budgets, and let one blocking wait reattach to that id. A client timeout, disconnect, or unexpected EOF may end the attachment, but it never decides or rewrites the server-owned outcome.

**Running inside an agent-manager run? The wait suspends you automatically — nothing to manage.** When `test-genie runs wait` or `git-control-tower baseline diff` is invoked from inside an agent-manager-managed agent (detected via the injected `VROOLI_AGENT_IDENTITY_TOKEN`), the command **parks** the run instead of blocking: your process exits (zero tokens burned), agent-manager performs the blocking wait on your behalf, and resumes your conversation with the result injected as the next turn. You will see a `PARKED — …` message and should simply stop — do not keep working or re-issue the wait; you will be woken with the result (or a typed timeout). **Outside an agent-manager run (a human shell, CI, or another runtime) behaviour is exactly as described above — the command blocks normally.** This is the only reliable wait for AM agents, which have no native blocking primitive; see `scenarios/agent-manager/docs/internal/TEMPORAL-FLOWS.md` (park/wake).
