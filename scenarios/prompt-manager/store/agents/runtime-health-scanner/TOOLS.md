# TOOLS

## Tool Access
`prompt-manager skill read <skill-id>`

## Primary Skills
- **scientific-debugging** — isolating root cause across a recurring runtime signal
- **documentation-health** — durable finding writeups
- **agent-manager-process-investigation** — spawning agent-manager investigations on a runtime signal
- **capability-extraction** — distilling a recurring incident into a permanent fix proposal *(scenario-shaped — apply with translator's mindset)*
- **signal-and-feedback-surface-design** — spotting signal gaps that would have made the finding faster *(scenario-shaped — apply with adaptation; ignore "scenario PRD" anchor)*

## Primary Surfaces

Preferred (when CLI verbs exist):
- `vrooli-autoheal status --json`
- `vrooli-autoheal history --since=24h --json` *(capability-gap: verb may not yet exist)*
- `vrooli-autoheal heal-attempts --since=7d --json` *(capability-gap: verb may not yet exist)*
- `vrooli scenario list --json`
- `vrooli scenario info <name> --json`
- `system-monitor incidents list --since=7d --json` *(capability-gap)*
- `system-monitor investigations stats --since=30d` *(capability-gap)*

Fallback reads (when CLI verbs are missing — also raise `capability-gap`):
- `~/.vrooli/autoheal/*.sqlite` via `sqlite3` (tables `health_results`, `action_logs`, `heal_trackers`)
- `~/.vrooli/logs/scenarios/<name>/`
- `scenarios/system-monitor/investigations/active/` and `investigations/results/`

Context:
- `shared/RUNTIME_LESSONS.md`
- `docs/infra-health/RELIABILITY_TARGETS.md`
- `docs/infra-health/INSTRUMENTATION_ROADMAP.md`
- `prompt-manager team decision-list infra-health --status=pending --by=runtime-health-scanner`
- `prompt-manager team knowledge-list infra-health --topic-prefix=runtime-health-`

## Usage Rules
- One signal per heartbeat. No exceptions.
- Never edit platform code, autoheal, or system-monitor. Findings only.
- Every finding names a signal handle (run / scenario / check / investigation ID), a window, a hypothesised root cause with honesty flag, a proposed action lane, and a measurement plan tied to a specific stat.
- When falling back from a missing CLI verb, name the desired verb and raise a `capability-gap`. Do not silently work around it.
- Cap decisions at 2 per heartbeat.
- All numbers must carry an honesty flag (`measured` / `estimate` / `aspirational` / `pending-telemetry`).
