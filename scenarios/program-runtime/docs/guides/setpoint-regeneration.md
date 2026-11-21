# Regenerate the improve setpoint readings

The canonical `program-runtime-improve` skill keeps its Section 2 readings as a
dated projection of one `program-runtime.setpoint-read` envelope. Capture the
envelope once, then apply it with the repository-owned generator:

```bash
program-runtime library run program-runtime.setpoint-read --json > /tmp/program-runtime-setpoint.json
python3 scenarios/program-runtime/scripts/regenerate-setpoint-readings.py \
  --envelope /tmp/program-runtime-setpoint.json \
  --skill scenarios/program-runtime/skills/program-runtime-improve/SKILL.md \
  --date 2026-09-07
```

The second command reads `signals.rows`, preserves each unavailable reason, and
rewrites only the `Today (generated)` column. Re-running it with the same
envelope and date is byte-identical:

```bash
cp scenarios/program-runtime/skills/program-runtime-improve/SKILL.md /tmp/program-runtime-improve-before.md
python3 scenarios/program-runtime/scripts/regenerate-setpoint-readings.py --envelope /tmp/program-runtime-setpoint.json --skill scenarios/program-runtime/skills/program-runtime-improve/SKILL.md --date 2026-09-07
diff -u /tmp/program-runtime-improve-before.md scenarios/program-runtime/skills/program-runtime-improve/SKILL.md
```

A fresh cycle must capture a fresh envelope. A row with `unavailable: true` is
written as `unavailable (<reason>)`; it is never rendered as zero.
