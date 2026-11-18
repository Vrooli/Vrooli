# Runtime friction report

`GET /api/v1/runtime/friction?since=<RFC3339>&until=<RFC3339>` reports measured
runtime for native and imported Agent Manager runs. Omitting both bounds selects
the latest seven days. The interval must be positive, no longer than 30 days,
and cannot end in the future.

The report contains six duration categories: `stopped_by_check`,
`waiting_on_validation`, `agent_manager_failure`, `rework`,
`sandbox_environment`, and `building`. Durations are allocated only from durable
time accounting or invocation/operation durations. Unattributed, idle, human
waiting, and evidence gaps remain unknown. Coverage reports the observed run
count, imported run count, event coverage, and unknown duration.

`blockedChecks` groups failed verification command evidence and structured
guard refusals. It reports recurrence (`occurrences` and distinct `runs`),
distinct `blockedRunIds` for runs with at least one blocked evaluator outcome,
enforcement outcomes, measured enforcement request time and its coverage, plus
whether later run evidence shows a successful verification, stopped as
blocked, used an observed workaround, asked for help, or has an unknown
response. A command path or display label is not an evaluator or guard
identity. `checkId` is populated only from structured owner-authored evidence,
currently the Workspace Sandbox denial event. Codex hook evaluator IDs and
durations may appear in persisted stderr/transcripts, but transcript text is
mutable/importable and is not authoritative; those markers are ignored for
identity and enforcement duration. Harness refusals remain visible with an
empty ID, and cannot qualify a structural decision for an exact evaluator.
Historical runs with no retained denial metadata remain unknown rather than
being reconstructed from command text.

For Workspace Sandbox 403 denials, the durable error event retains the exact
source code, `blocked` outcome, and measured request duration. That duration
measures the enforcement request only; it does not estimate agent rework,
human waiting, or total lost time. Workspace Sandbox codes are not assumed to
map to selectable guard modes. Only IDs declared by G's guard runtime may be
used as evaluator selectors.

An authoritative Codex hook receipt is not implemented. Before Codex refusal
identity can feed recurrence, the policy runner and Agent Manager need a
durable authenticated receipt contract with one-time event identity, exact
unambiguous native-session-to-run binding, and a trustworthy link to the actual
hook invocation. Receipt payloads must exclude raw tool input. Tests must cover
deny-only emission, redaction, authentication, exact/foreign/ambiguous binding,
replay rejection, marker-only rejection, and idempotent import/recovery. No
native refusal or live recurrence is claimed by the current report.

The authenticated consumer command is:

```sh
agent-manager usage runtime-friction --json
agent-manager usage runtime-friction --since 2026-11-09T00:00:00Z --until 2026-11-16T00:00:00Z --json
```

The JSON output preserves the API response. Human output summarizes coverage,
category durations, blocked command responses, and whether an evaluator ID was
recorded.
