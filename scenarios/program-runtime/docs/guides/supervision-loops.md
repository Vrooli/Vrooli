# Supervision loops → programs (standing capability)

Supervisors and delivery-team agents run the same read loops over and over (health
checks, validation checks, "why is this scenario unhealthy", investigation). Each is a
long governed-read loop — the shape a Program Runtime program exists to replace. This
guide is the standing front door: the loop → the one program call that replaces it.

It exists because the program corpus is healthy in supply but under-consumed (a 2026-10
portfolio-audit scored `exercised` ≈ 38%). Calling these programs instead of hand-running
CLI loops is what raises adoption. Branch only on the envelope `status`
(`ok`/`partial`/`unavailable`/`failed`) and `errors[0].class`. **`unavailable` is unknown,
never zero.**

## The supervision set (program-runtime-owned)

| Loop | Program | Notes |
|---|---|---|
| Running/health across a named scenario set | `program-runtime.supervision-sweep` | one `vrooli/scenario/list` read; `signals.attention` = not running/healthy. Proven 8→1. |
| Latest recorded validation verdict across a set | `program-runtime.validation-sweep` | one `validate` verb read/scenario of the LATEST RECORDED run (never starts one); `signals.attention` = not passing / never validated, `signals.unavailable` = unknown reads. Reads the primary recorded store — empty where runs are routed/isolated (`primary_root_writes=0`). |
| Why is one flagged scenario unhealthy | `program-runtime.scenario-triage` | joins `status` + `freshness` + a bounded log tail into advisory `signals.hypotheses`. Read-only; never restarts. Log classification needs a lifecycle log — dark for registry-runtime scenarios (status + freshness still live). |
| Program corpus health / adoption | `program-runtime.portfolio-audit`, `program-runtime.usage-triage` | keep the layer itself honest. |
| Retained delivery-effort obligations | `agent-manager.supervision-observation-read` | One bounded effort-board page; follow its continuation token and preserve partial, stale, and unknown states. It does not dispatch work or prove completion. |
| Runtime measure summaries / failure shapes / reachable bindings | `program-runtime.fleet-fanout`, `program-runtime.failure-triage`, `program-runtime.registry-sweep` | |
| Agent run efficiency / setpoint / one-subject diagnosis | `agent-manager.efficiency-report`, `agent-manager.setpoint-read`, `agent-manager.investigate` + `investigation-evidence` | agent-manager-owned; reuse, don't re-author. |

## The supervision cycle

```bash
program-runtime library run program-runtime.supervision-sweep    # running/health
program-runtime library run program-runtime.validation-sweep     # latest validation verdict
# for anything in either signals.attention:
program-runtime library run program-runtime.scenario-triage --input scenario=<slug>
```

Weekly, keep the layer honest: `portfolio-audit` + `usage-triage`.

## Deliberately NOT a program (honest boundaries)

- **Owner-decision routing** (`effort decision list|raise|sync`) — recording an owner's
  answer or advancing a gate is owner-reserved; a program would auto-answer. Keep it a
  human-in-the-loop CLI step.
- **Worker spawn** (`task create` + profile resolve) — a write + delegation; that is
  `delegated-run`/`watch-set` territory, not a read program.
- **A complete delivery-team progress decision** — the observation program above is a
  bounded read, not a substitute for examining owner evidence, judging outcomes, or
  making the supervisor's decision. Follow the `large-effort-supervision` skill.

## Efficiency accounting

These programs make **0 inference calls** (read-only). The envelope reports wall/cpu time
and `usage_accounting_complete`, but not token/dollar cost. State the honest, measurable
win — CLI round-trips collapsed (e.g. supervision-sweep: 8 `scenario status` calls → 1
program + 1 read, ~5s) — never an invented token number.
