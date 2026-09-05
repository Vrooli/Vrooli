# Requirements

Requirement modules live here, one folder per group of operational
targets. Every requirement links back to a PRD operational target via
`prd_ref` and carries at least one validation entry pointing at its
proof.

- Statuses are earned, not asserted: auto-sync updates them from
  `[REQ:ID]`-tagged test results on comprehensive suite runs.
- Replace scaffolded manual validation stubs with test-typed entries
  (a `ref` to the test file plus the `[REQ:ID]` tag) as behavior lands.
- Validate with `business-health validate scenario <scenario>`; inspect
  traceability with `business-health matrix show <scenario>`.

## Expensive validation

Long-running audio qualification is intentionally marked `out_of_band` so the
ordinary scenario suite stays bounded and reproducible. The owning CLI is the
only supported runner:

```bash
audio-tools validation run-expensive
audio-tools validation check-freshness
```

`run-expensive` requires at least 60 minutes. It keeps a production TTS
stream exercised while a production bidi STT stream receives continuous
synthetic PCM, then atomically writes
`requirements/evidence/expensive-validation.json` and refreshes each
out-of-band validation's `last_validated_at`. The business-phase freshness
gate fails closed when a timestamp is missing, malformed, or older than
`valid_for_days`.
