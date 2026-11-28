# codex-rollout

Reads Codex native rollout records and reports the operational token usage they contain
(input, cached input, output). The figures are observed usage, neither an invoice nor a
statement of subscription quota. Record lines are bounded by `MaxLineBytes`.

It also says how a turn ended and what to do when it ended on an error:

- `ReadLastTurn(path, maxBytes)` reads a rollout backwards to its newest turn boundary
  (`task_started`, `task_complete`, `turn_aborted`), with the error a `task_complete` carries.
- `turn_errors.json` is the one table of turn-ending errors: rules match `codex_error_info`,
  the HTTP status and message text to a class, and each class names its recovery (`resume`
  with backoff, cap, window and staleness, or `escalate`). `DefaultTurnErrorTable().Classify`
  applies it and `Classification.Plan` decides one halt. Add a rule when a new shape is seen
  (its `seen` field records where); the fallback class must escalate. Other languages read the
  JSON file directly. Research Desk's automatic recovery is the first consumer.

Governed by `.vrooli/package.json`; see [package governance](../../docs/package-governance.md).
