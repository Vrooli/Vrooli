# Testing

Project-level testing guidance is intentionally thin. The canonical rule is:

- use `vrooli scenario test <name>` for scenario suites
- use the relevant Go or package-level test commands for the platform code you changed
- prefer current CLI surfaces and maintained fixtures over shell-era ad hoc test flows

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

For provider-specific health maturity, use the provider's default human CLI output. Reserve `--json` for Test Genie and other automation.

## Waiting on runs (for agents)

The run is owned by the test-genie server, so it survives your command being cancelled. Just run it — the run id + a re-attach command are printed up front, and a known-long run auto-backgrounds so your shell returns immediately. `vrooli scenario test <name>` is a direct entry point for `test-genie --auto-start execute <name>`.

**Do NOT poll with repeated "still waiting" checks. To wait, block ONCE with the quiet wait verb:**

- `test-genie runs wait --json <scenario> <run-id>` (also `vrooli scenario test wait <scenario> <run-id>`). It blocks server-side and returns exactly once with the verdict + the run's real exit code (`0` passed, `1` failed/aborted, `124` if you pass `--timeout` and it elapses first). It does NOT stream — one call, one return. This is the verb the start banner and re-attach commands print; copy it verbatim.
- **If you must bound the wait, use `--timeout=<seconds>`.** On timeout it returns `124`, the JSON snapshot still carries `recommended_next_check_seconds`, and stderr prints the exact re-invoke line. **Re-call only after that many seconds — never poll faster, never re-run immediately.**
- `test-genie runs follow <scenario> <run-id>` is the **human** live-watch verb (a continuous, heartbeating stream). Do not use it to "wait" as an agent — a backgrounded stream re-wakes you on every heartbeat. Use `runs wait --json`.
- Cancel ≠ abort — to actually stop a run use `vrooli scenario test abort <scenario> <run-id>`.

**One run per scenario at a time.** The test-genie server allows at most one in-progress run per scenario (different scenarios run concurrently). An identical re-request coalesces onto the running run (no second suite); a *different* request for a busy scenario is rejected with the in-flight run id + `runs wait --json`/`runs abort` guidance — wait or abort, don't retry-spam.

**Waiting on several runs at once?** Use one `test-genie runs wait-all --run <scenario>:<run-id> --run …` call (repeatable; add `--json`). It blocks until every named run is terminal and returns one aggregate exit code (`0` all passed, `1` any failed, `124` any still in-flight at `--timeout`, `2` any not-comparable) — so two parallel suites/diffs resolve in a single call instead of two backgrounded streams.

**Scenario starts have the same contract — don't poll them either.** `vrooli scenario start|restart` write a durable start-operation record; to wait on one, block once with `vrooli scenario wait <scenario> --json [--timeout N]` (exit `0` healthy, `1` failed, `2` degraded, `124` timeout with the start unaffected). Full protocol: `docs/reference/cli-commands.md` ("Scenario start wait contract").

**Baseline diff is durable too — don't poll it.** `git-control-tower baseline diff --scenario S --name N` returns immediately with a run id + re-attach command (it reuses a clean-tree run when one exists, so it usually doesn't even re-run the suite). Resolve the verdict with `git-control-tower baseline diff status --scenario S --name N --run <run-id>` (exit `0` clean, `1` regression, `2` not-comparable, `3` not-ready), or add `--wait` to block server-side and print it inline.

**Baseline snapshot is re-attachable.** `git-control-tower baseline snapshot --scenario S --name N` returns as soon as the server owns the run. Resolve the manifest write with `git-control-tower baseline snapshot status --scenario S --name N --run <run-id>` (exit `0` ready, `2` missing/failed, `3` pending), or add `--wait` to block server-side. If a later `show`/`diff` cannot find the manifest, use `snapshot status` first; it distinguishes pending/failed snapshot intents from a wrong baseline name and prints similar-name hints when available.

**Running inside an agent-manager run? The wait suspends you automatically — nothing to manage.** When `test-genie runs wait` or `git-control-tower baseline diff` is invoked from inside an agent-manager-managed agent (detected via the injected `VROOLI_AGENT_IDENTITY_TOKEN`), the command **parks** the run instead of blocking: your process exits (zero tokens burned), agent-manager performs the blocking wait on your behalf, and resumes your conversation with the result injected as the next turn. You will see a `PARKED — …` message and should simply stop — do not keep working or re-issue the wait; you will be woken with the result (or a typed timeout). **Outside an agent-manager run (a human shell, CI, or another runtime) behaviour is exactly as described above — the command blocks normally.** This is the only reliable wait for AM agents, which have no native blocking primitive; see `scenarios/agent-manager/docs/internal/TEMPORAL-FLOWS.md` (park/wake).
