## Investigating agent runs

Start with `agent-manager run report <id>`. It supplies status, costs, result
provenance, event/tool failure counts, model fallback, and diff statistics.
Use its `Next:` commands only when a discriminator requires payload evidence.

| Discriminator | Drill-down | Attribution |
| --- | --- | --- |
| Tool failure, unavailable CLI, or non-zero exit | `run tools`, `run events` | Environment/Tooling |
| Invalid or abstained structured result | `run result`, `run messages` | Agent Setup unless the source shows a platform error |
| Requested/actual model mismatch or fallback | `run events` | Environment/Tooling |
| High turns/tokens with no terminal result | `run messages`, `run events` | Agent Setup or Both |
| Diff exists but result is unavailable | `run diff`, `run result` | Both |

Attribute to the lowest layer proven by durable evidence: CLI/tool output,
capability, skill design, docs/discovery, process/policy, then intent/inputs.
Severity is Critical for delivery/safety blockers, Major for repeated retries
or forced guessing, Gap for implied-but-unavailable capability, and Minor for
low-risk clarity work. Prefer fixes that eliminate recurring interpretation.
