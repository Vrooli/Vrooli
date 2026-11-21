# Testing

Project-level testing guidance is intentionally thin. The canonical rule is:

- use `vrooli scenario test <name>` for scenario suites
- use the relevant Go or package-level test commands for the platform code you changed
- prefer current CLI surfaces and maintained fixtures over shell-era ad hoc test flows

For mutating BAS/E2E workflows, follow the two-engine
[routed test-storage contract](agent-system/routed-test-db.md): SQL and file
writes must be leased and test-mode routed before execution.

## Start Here

- [Authoring tests that protect behavior](testing/UNIT-TEST-AUTHORING.md) — boundaries, fixtures, expectations, and evidence limits
- [reference/cli-commands.md](reference/cli-commands.md)
- [reference/health-maturity-assessments.md](reference/health-maturity-assessments.md)
- [scenarios/VALIDATION.md](scenarios/VALIDATION.md)
- [../scenarios/test-genie/docs/QUICKSTART.md](../scenarios/test-genie/docs/QUICKSTART.md)

## Common Commands

```bash
vrooli scenario test <name>
go test ./cmd/vrooli/... ./internal/...
make hygiene
make test-packages
```

Use the smallest validation surface that honestly covers your change.

## Shared-package baseline

Test Genie validates scenarios; shared packages under `packages/` and the
`resources` Go module have their own baseline:

```bash
make test-packages
```

It runs `go test -count=1 ./...` in every governed package that has Go code and
in `resources/`, prints `ok`/`FAIL` per module with the failing test names, and
exits non-zero listing the red modules (full logs are kept in the printed
temporary directory). The baseline is expected to be green. A red module you did
not touch is pre-existing rot: report it to its owner instead of reading it as
your breakage, and do not paper over it.

Run it before and after a change that other modules read as data, because those
breakages surface in a module you did not edit: `cli/manifest.json` and the CLI
manifest schema, proto enums, `.vrooli/schemas/*`, scenario
`.vrooli/test-genie.json` descriptors, `.vrooli/repo-contract.json`, and the
reliability setpoint. For a change inside one package, `go test ./...` in that
package is enough.

The baseline covers every package because the registry is complete:
`go test ./internal/packagegov` fails when a directory under `packages/` lacks a
loadable `.vrooli/package.json`. Package tests must be hermetic. A check that
executes installed host binaries or reads live host configuration is a live
check: pin its inputs in the test, or gate it behind an explicit
`VROOLI_<AREA>_LIVE=1` opt-in, as `VROOLI_CLI_CONFORMANCE_LIVE=1` does for the
derived-verb CLI conformance check in `packages/api-core/scopecatalog`.

A Go test's verdict must not depend on who runs it. Tests run from agent
sessions and lifecycle-managed shells, and those carry ambient state:

- **Agent identity.** Inside an agent session, cli-core ignores
  `API_BASE_URL` and sends local requests through the live credential broker.
  A CLI test pointed at a fake server would reach the running scenario
  instead. `cliapptest.NewAPIServer`/`WithAPIBase` detach the test
  (`cliapptest.IsolateAgentEnvironment`). A package whose tests call local
  servers through cli-core uses
  `func TestMain(m *testing.M) { os.Exit(cliapptest.RunWithoutAgentSession(m)) }`.
  A test that execs a built CLI as the operator clears
  `VROOLI_AGENT_IDENTITY_TOKEN`.
- **Per-install storage.** Contract-routed storage (operator state, receipts,
  scenario databases) resolves from the environment, not from the repository
  root a test passes in. Under `go test`, the shared resolver in
  `packages/api-core/storage` refuses the operator's live runtime home. A
  package whose code resolves storage on its own isolates it with
  `storagetest.RunWithIsolatedStorage` in `TestMain` (`testenv` re-exports it
  for the control plane) or `storagetest.IsolateStorage`. Tests that resolve
  through an injected environment, or that expect default-layout paths under a
  HOME they set, use `storagetest.RunWithPrivateHome` instead.
- **Temp location.** `t.TempDir()` follows `GOTMPDIR`, which lifecycle shells
  set to `~/.vrooli/tmp/go-work`. A temp dir can therefore sit under a home
  directory that contains `.vrooli/` (or a stray `.git`), and its paths are
  long. Put Unix sockets in a short `os.MkdirTemp("", …)` directory. Do not
  assume a temp dir has no Vrooli ancestor. Build Go fixture modules in temp
  dirs with `-buildvcs=false`.

Run a suspect package once normally and once under
`env -i HOME=$HOME PATH=$PATH`. A difference between the two is a
hermeticity bug.

Repository-wide source audits walk `repocontract.Contract.WalkSource`. It skips
documentation trees, whose retained evidence snapshots code without being
source.

`vrooli package test <package>` is the full Test Genie validation of one package
(unit, structure, quality and the other package-applicable phases). It starts a
server-owned run and returns before the verdict, so its exit status is not the
result. Block once with the printed
`test-genie runs wait --json package:<package> <run-id>`.

## Generated-output freshness

Generated files are an output of a declared owner, not an independent source
tree. A compile failure that names a missing generated type, client, selector,
registry, or descriptor is first a freshness signal; do not hand-edit the
generated file or infer a product defect until the owner output has been
checked.

For a governed generated package, prefer the package boundary:

```bash
vrooli package test <package>
```

Package-target tests preflight the repository-level owner's declared generation
or generated-output-owning build lifecycle before delegating to Test Genie.
This rule is package-kind agnostic: Proto and future context-free schema,
selector, registry, descriptor, or runtime owners are picked up by the same
manifest contract. Scenario-scoped build artifacts remain owned by scenario shared-
package setup, where the requesting scenario is available; they must not gain a
Test-Genie-specific regeneration exception.

Lifecycle setup and the `vrooli scenario test` entrypoint perform the same
repository-generated owner preflight before a consumer is built or a run is
handed to Test Genie. A stale owner therefore fails at the shared lifecycle
boundary, regardless of which scenario, phase, or consumer will use the
generated output. The preflight is freshness-gated and writes diagnostics to
stderr, so current artifacts add no regeneration work and do not corrupt
JSON/JSONL test output.

When running a focused module test directly, use the owner's verifier or
generation command first. For protobuf consumers, the read-only check is:

```bash
cd packages/proto && make verify-committed-gen
```

If it reports drift, run the governed scoped generator (`vrooli package
generate proto` or `go run -mod=mod ./cmd/protogen generate --changed` from
`packages/proto`) and verify again. Never use direct `buf generate` for routine
work: it bypasses dependency closure, output-digest repair, locking, and the
atomic publication boundary. Record the freshness check and artifact identity
with the focused test evidence so stale-build failures cannot be mistaken for
behavioral regressions.

## Ordinary iteration versus certification

For an ordinary fix, reproduce the defect with a focused regression first.
Run affected package checks while editing. For filtered tests, retain the names
that actually ran (`go test -v` or JSON events), not only the package's `ok` line.
Confirm each intended regression was selected and did not skip; an anchored exact
name can silently exclude longer test names. Correct missing coverage once;
do not count `no tests to run`, compilation alone, or another test as that proof.

For long or noisy direct package checks, retain the complete output and exit
status in the task's evidence before printing a bounded summary. Go JSON output
must retain package and test failure events. Read that evidence before rerunning;
a truncated terminal response can hide the failing test name and is not proof of
a clean run. Use the server-owned artifacts for scenario suites instead of
reconstructing their output from a terminal excerpt.

For focused browser-case diagnosis, use Workflow Health's existing
[selected-case route](../scenarios/workflow-health/docs/operations/RUNBOOK.md)
(`workflow-health validate start-selected`, then its returned run's `validate wait`).
Retain its isolation evidence and request/write counters. This provider-owned
result is not a Test Genie phase receipt; run the required full workflow phase
before claiming that gate. Use changed inputs and a new request key for another
iteration, rather than repeating a failed full phase for each local correction.

For UI test-helper or framework-policy changes, check the affected workspace with
`unit-health validate scenario <name> --workspace ui` before the executing unit
phase. Without `--execution`, this existing owner checks static framework and
renderer policy without running test commands. It can catch projection drift
before an expensive suite, but does not prove behavior, coverage, or
execution-dependent quality checks.

During ordinary iteration, when test execution is isolated to one workspace,
Unit Health can run only that workspace with
`unit-health validate scenario <name> --workspace <id> --execution`. This is a
scoped Unit Health result, not a passing Test Genie unit phase: the selector
limits planning, analysis, execution, and evidence scope, while required-role
governance still observes the full inventory. `--fast-test-only` omits coverage.
Use this result to avoid rerunning unrelated workspace suites while iterating;
run the required Test Genie unit phase before claiming the scenario gate or
certification is satisfied. Run the relevant scenario phases:

```bash
vrooli scenario test device-control --phases unit,programs
program-runtime library run test-genie.iterate --input 'scenario=device-control,request_id=volume-fix-1,phases=["unit","programs"]'
```

Choose phases from affected interfaces and obligations, not merely changed file
extensions. Shared Go source or import changes can also change the module metadata
needed by consumers using local replacements, even when the shared module's
`go.mod` is unchanged. Include the affected scenarios' `dependencies` phase for
that boundary; it delegates module readiness to Scenario Dependency Analyzer.
SDA's existing `freshness --touched --json` can identify affected Go surfaces.
Review the selected set in a shared dirty checkout. Repair confirmed drift through
the dependency owner's governed commands, not by copying the root `go.sum`.
See [dependency governance](package-governance.md).

Add shared dependency content roots to `iterate` when they affect the
result. Explicit phases remain required even if their cost exceeds the budget.
With no explicit phases, `iterate` requests Test Genie's targeted/quick profile;
the existing owner planner uses historical durations to choose that profile's
checks. The program returns the exact intent and one durable wait command.
Use a new request ID for changed inputs; retain the same ID for a lost admission
response. Admission is not a passing verdict. `plan_only=true` performs no admission.

A comprehensive baseline is not a prerequisite for diagnosis. Reuse prior
observations as diagnostic context and expand only for changed interfaces,
new failures, uncovered risk, or explicit certification requirements. Certification
retains its declared checks and must not use this iteration shortcut.
This section owns scope policy; usage and debugging skills reference it.

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
- **Run evidence is catalogued and path-free.** New runs atomically persist a typed artifact catalog over their existing run-owned files. List/filter evidence by open kind strings (not phase names), then use the Test Genie CLI to download bytes by run-scoped opaque artifact ID; provider relative/absolute paths never cross the API boundary. Historical runs without a catalog use explicit degraded legacy discovery. Missing bytes, unsafe symlinks, cross-run IDs, and incompatible catalogs are errors, never absent evidence interpreted as PASS. The commands are `test-genie runs logs <scenario> <run-id> [--phase <phase>]` (phase command logs), `test-genie runs artifacts <scenario> <run-id>` (catalog), and `test-genie runs download-artifact <scenario> <run-id> <artifact-id>` (bytes to stdout for piping; `--out <path>` publishes a file with a SHA-256 receipt). Every `runs` command takes `<scenario> <run-id>` in that order, and `--scenario <scenario>` is accepted in place of the first argument. Use the catalog's ID as-is; never reconstruct its internal HTTP route.
- Keep detailed run payloads with Test Genie's artifact owner; commit a concise finding, source identity, and artifact/run references instead of copying raw logs, source snapshots, or duplicate patches into scenario documentation. For immutable source comparison, use Git Control Tower's bounded path estimate before capture; snapshots are metadata/digest-only by default and use a bounded retention lease. Capture only a small exact source subset when content is required.
- **Read detailed failures from the unfiltered run artifact catalog.** A producing-phase filter can omit the run-level `findings.report`. That artifact preserves provider `NativeDetail`, including bounded Unit Health command diagnostics. `runs findings` and phase log/result artifacts are summary views; inspect the unfiltered catalog before declaring error output missing or rerunning a failed phase.
- **Artifact metadata is not the payload.** `runs artifact` returns typed metadata only; read the payload with `test-genie runs download-artifact` before classifying diagnostics. A provider `nativeDetail` can be a protobuf `Any` with `type_url` and base64 `value`; decode it with the matching generated type before searching its fields. Record the payload byte count/hash. Metadata-only reads or searches of the encoded value do not establish that command details are absent.
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

**Shared-worktree validation.** Multiple agents edit the same worktree while
tests run. Isolation and a quiet worktree are not prerequisites for ordinary
plan completion. Retain observed assertions and report relevant input changes
as an applicability limitation. A digest does not prove that a mutable run read
one exact source version. Exact-input cache reuse remains conservative; plan
acceptance is a separate decision owned by Plan Manager.

Broad Test Genie phases are advisory for ordinary plans. Inspect a finding for
relevance, credibility, and consequence before deciding to repair it. A newly
observed failure does not establish attribution. Use bounded, non-destructive
investigation; retain unrelated, low-impact, or uncertain findings for triage.
Explicit certification retains its declared required checks.

Baseline results are diagnostic context. Reuse available observations, including
failures, with their actual timestamps and provenance. Capture a new baseline
only when its expected diagnostic value justifies the cost. Missing prior state
remains unknown and does not require capture before ordinary implementation.
Never reset, checkout, restore, or stash shared-worktree contents to reconstruct
prior behavior. Use read-only history inspection such as `git show` or `git diff`.
Runtime leases protect conflicting test-plant mutations, not ordinary commits.

Start a collection comparison with `git-control-tower baseline collection diff
--name <collection> --operation-id <stable-operation-id> [--member <scenario>
...]`. The operation id is an idempotency key: retain and reuse it to inspect or
reattach the same operation; do not invent a new id after an interrupted start.

**Plan Manager consumes validation receipts.** Follow the exact next action from
`plan-manager exec continue <plan> --json`. Validation intents are admitted by
Test Genie, which owns reuse, queued work, execution and the durable receipt.
Attach once with the printed `test-genie validation wait` command; retain the
receipt ID and observer identity across disconnects. Use the printed Plan
Manager synchronization action to consume that receipt. Do not manually shuttle
producer IDs among scenario owners or start another capture to repair a wait.
Explicit certification may still include a Git Control Tower collection as a
receipt child; that does not make comprehensive capture the default edit loop.

Phase scopes can declare `testPhases` and opt into `compareBehavior`; explicit
certification keeps the plan's full evidence policy. Reaching the final phase
does not itself select certification. Receipt compatibility uses
content identity, not commit/branch/dirty attribution. The typed
`behavioralPrior` and selected checks are execution inputs and must match for
reuse. Content identities retain per-file manifests for explaining differences;
they do not isolate a running suite from concurrent edits. Relevant mutations
limit exact-version applicability; preserve the observations and let the
declared completion policy determine whether further checking is necessary.

If a broker attachment to a durable child fails, its receipt stays nonterminal
with a persisted reattachment time. The owner reattaches to the recorded child;
it does not count that observer failure as a failed test attempt. Historical
terminal failures remain historical evidence and are not silently rewritten.

All long validation operations follow the same ownership rule: persist intent and an operation id before dispatch, separate queue/execution/transport budgets, and let one blocking wait reattach to that id. A client timeout, disconnect, or unexpected EOF may end the attachment, but it never decides or rewrites the server-owned outcome.

**Running inside an agent-manager run? The wait suspends you automatically — nothing to manage.** When `test-genie runs wait` or `git-control-tower baseline diff` is invoked from inside an agent-manager-managed agent (detected via the injected `VROOLI_AGENT_IDENTITY_TOKEN`), the command **parks** the run instead of blocking: your process exits (zero tokens burned), agent-manager performs the blocking wait on your behalf, and resumes your conversation with the result injected as the next turn. You will see a `PARKED — …` message and should simply stop — do not keep working or re-issue the wait; you will be woken with the result (or a typed timeout). **Outside an agent-manager run (a human shell, CI, or another runtime) behaviour is exactly as described above — the command blocks normally.** This is the only reliable wait for AM agents, which have no native blocking primitive; see `scenarios/agent-manager/docs/internal/TEMPORAL-FLOWS.md` (park/wake).
