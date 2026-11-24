# Regenerate the improve setpoint readings

`program-runtime-improve` section 2 owns the setpoint rows (sensor and band). The dated
readings live in `docs/reference/setpoint-readings.md`, a projection of one
`program-runtime.setpoint-read` envelope. Capture the envelope once, then apply it:

```bash
program-runtime library run program-runtime.setpoint-read --json > /tmp/program-runtime-setpoint.json
python3 scenarios/program-runtime/scripts/regenerate-setpoint-readings.py \
  --envelope /tmp/program-runtime-setpoint.json --date 2026-09-07
```

The generator reads `signals.rows`, preserves each unavailable reason, and writes only the
readings file; it refuses when the skill table and the envelope disagree on a row name in
either direction. Re-running it with the same envelope and date is byte-identical. A fresh
cycle must capture a fresh envelope. A row with `unavailable: true` is written as
`unavailable (<reason>)`; it is never rendered as zero.
