# Supervision loops → programs (standing capability)

Supervisors and delivery-team agents run the same read loops over and over (health
checks, validation checks, "why is this scenario unhealthy", investigation). A bounded
program can combine repeated governed reads when its contract fits the review. The
supervisor still selects subjects, evaluates evidence, and uses available owner reads
when a program or dependency is unavailable.

It exists because the program corpus is healthy in supply but under-consumed (a 2026-10
portfolio-audit scored `exercised` ≈ 38%). Reuse is worthwhile when it improves the
outcome and its overhead is justified. Inspect the envelope `status`, per-source signals,
errors and evidence before deciding; successful program execution is not a successful
supervision decision. **`unavailable` is unknown, never zero.**

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
| Selected research threads, runs and changed event counts | `program-runtime.research-supervision-read` | Caller supplies exact identities. Optional per-run cursors select one page of counts without payloads; `has_more` means more pages remain. Missing receipts, stale imports and missing sources are not proof of health. |
| Research workspace navigation signals | `program-runtime.research-workspace-review` | Pass an authorized absolute research root. Bounded missing-file and size signals are prompts to inspect, not judgments of research quality. |

## The supervision cycle

```bash
program-runtime library run program-runtime.supervision-sweep    # running/health
program-runtime library run program-runtime.validation-sweep     # latest validation verdict
# for anything in either signals.attention:
program-runtime library run program-runtime.scenario-triage --input scenario=<slug>
```

Weekly, keep the layer honest: `portfolio-audit` + `usage-triage`.

For research, `prompt-manager skill read research-coordination` owns the adaptive
review cadence and response to uncertainty. For example, after resolving a run ID:

```bash
program-runtime library run program-runtime.research-supervision-read \
  --input 'run_ids=["<run-id>"]' --input 'event_cursors={"<run-id>":null}'
program-runtime library run program-runtime.research-workspace-review \
  --input research_root=/absolute/authorized/research/root
```

For the next bounded event page, pass the returned `next_cursor` for that same run.
Inspect `has_more`, coverage and errors before accepting any observation. Event counts
do not establish live import freshness, event significance, or absence of unobserved
receipts. No program here wakes a supervisor or enforces a research cutoff.

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
program + 1 read in one example) — never an invented token number or a general claim of
faster or cheaper supervision. Compare full worker and supervisor usage and outcomes.
