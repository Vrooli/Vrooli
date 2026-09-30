# Failure recovery

Submission resolves every name statically before a kernel runs. An unresolved
name is refused with its line and, when one is genuinely close, a nearest match:

```python
explanation = test_geni.runs.list()   # refused: nearest match "test_genie"
```

Names the program itself binds — parameters, lambda arguments, comprehension and
loop variables, `with` and `except` targets, function and class names — are
never flagged. Use `programs submit --explain` to see diagnostics without
executing.

A failed program stores a **cause** in `failure_shape`, drawn from a closed
vocabulary rather than a Python exception class. `failure_cause` is the typed
enum mirror of the same value:

`unresolved_name`, `unknown_field`, `ambiguous_response`, `unreachable_scenario`,
`refused_no_grant`, `refused_not_run_eligible`, `inference_spend_exceeded`,
`delegated_run_spend_exceeded`, `deadline_exceeded`, `kernel_syntax`,
`kernel_runtime`, `bridge_transport`, `unclassified`.

Fix the named cause and submit a new program:

- `unknown_field` — the error lists the candidate proto fields.
- `ambiguous_response` — pass `rows="<field>"`.
- `unreachable_scenario` — start the owner through the lifecycle.
- `refused_no_grant` — request the exact session grant. Never retry a refused
  destructive call without it.

`program-runtime programs mine` aggregates recurring causes across the corpus,
which is how repeated friction becomes visible to the readiness board.
