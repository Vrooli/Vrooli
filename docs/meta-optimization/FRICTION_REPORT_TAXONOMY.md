# Friction Report Triage Taxonomy

Cross-team-readable canon for how the `meta-optimization/friction-curator` member classifies and routes entries written to `friction-inbox/*`. Human-readable view of [`friction-report-taxonomy.json`](friction-report-taxonomy.json).

**Owner team:** meta-optimization. **Status:** canon. Operator-curated via `meta-self-improvement` decisions (owned by debt-curator) when scope, severity, or routing rules need to evolve.

Cited by:
- `scenarios/prompt-manager/store/teams/meta-optimization/members/friction-curator/topics.json` — `intake[].taxonomy = "friction-report"`.
- `scenarios/prompt-manager/store/skills/packs/core/report-friction/SKILL.md` — required reading; the skill writes entries that conform to the `friction-report` schema below.

Universal-source intake: `friction-inbox/*` declares `source_team: "*"` — any team's members may write. The producer-side anchor is the `report-friction` skill (declared in the friction-curator's `external_producers` so the validator's `wildcard_source_misuse` rule stays quiet).

This is the second universal observation flow in the system; the first is `bug-inbox/*` on scenario-qa. See [`docs/scenario-qa/BUG_REPORT_TAXONOMY.md`](../scenario-qa/BUG_REPORT_TAXONOMY.md) for the sister pattern.

## Topic shape

```
friction-inbox/<scope>/<short-slug>
```

The producer (the agent invoking `report-friction`) picks the scope from the fixed list below; the curator validates the choice and may reclassify if evidence supports it. There is **no separate classifier skill** — friction-inbox uses deterministic-prefix routing, mirroring bug-inbox's pattern. Routing is 1-to-1 from scope to destination scoped-topic.

## What is NOT friction (use the right channel instead)

Friction is *system-level capture-leak* — the gap between what the system promised and what it delivered when an agent tried to use it. These adjacent signals look like friction but route differently:

| Signal | Use this instead | Why |
|---|---|---|
| Broken scenario or code behavior — code defects, regressions, prompt confusion, unexpected errors | [`report-bug`](../../scenarios/prompt-manager/store/skills/packs/core/report-bug/SKILL.md) — writes to `bug-inbox/*` on scenario-qa | Bugs are defects against documented behavior; friction is gaps in promised capability. Different drainer, different fix layer. |
| Disagreement with an existing decision, plan, or operating contract | Raise a decision in the appropriate context (e.g., `decision-rejection-proposed` for stale decisions, `framework-update` for contract-level disputes) | Disagreement is structural input, not observation. |
| Capability the system should have but doesn't | The owning member raises a `capability-gap` decision (toolchain-validator, run-introspector, or team-agent-optimizer) | Capability gaps are commitments to build new infrastructure, not observations about existing infrastructure. |
| A fix the reporter could just apply right now | Apply the fix directly | Don't file friction reports for things you can fix in five minutes. The whole point of friction reporting is signal that the *system* should change. |
| Post-hoc deep analysis of a long conversation | [`conversation-friction-analysis`](../../scenarios/prompt-manager/store/skills/packs/core/conversation-friction-analysis/SKILL.md) | That skill is for analytic decomposition with timeline + attribution + scoring; `report-friction` is for in-flight observation. |

If your observation doesn't fit anywhere clean, prefer `report-friction` over silence — the curator can reclassify or hand off to debt-curator. Better to overshoot the inbox than to lose the signal.

## Scopes

| Scope                          | Definition                                                                  | Routes to                                               |
|--------------------------------|-----------------------------------------------------------------------------|---------------------------------------------------------|
| `toolchain`                    | CLI/command/tool gap, confusing flag, missing capability, misleading output | `friction-report/toolchain/<date>/<slug>` (toolchain-validator)|
| `run-execution`                | Run-loop, heartbeat, or coordination friction; stalls and loops             | `friction-report/run-execution/<date>/<slug>` (run-introspector)|
| `prompt-team-agent-storage`    | Storage-map ambiguity, write-target confusion, role-boundary issues         | `friction-report/prompt-team-agent-storage/<date>/<slug>` (team-agent-optimizer)|
| `recurring-workaround`         | A workaround applied multiple times — a pattern costing recurring attention | `friction-report/recurring-workaround/<date>/<slug>` (debt-curator)|
| `unknown`                      | Scope unclear; curator reclassifies during triage                           | reclassified → one of the four above; if not reclassifiable, handoff to debt-curator |

Scope choice is made by the producer at file time. If the producer is uncertain, `unknown` is the honest choice; the curator owns reclassification.

## Severity

The reporter assigns severity:

- `blocking` — the reporter is currently stopped; no workaround applied. Routes through curator with priority; may also escalate to a `capability-gap` decision via the destination scoped-topic owner.
- `recurring` — the friction has been observed multiple times across heartbeats or runs. Requires evidence of recurrence (count or prior-entry pointer).
- `one-off` — observed once, with workaround applied. **The curator drops `one-off` entries with a triage note** — file in handoff next time, not the inbox. This keeps the inbox actionable.

The curator may overrule severity based on observed scope or recurrence (e.g., a "recurring" report whose evidence shows only one occurrence becomes `one-off` and is dropped).

## Evidence rules

- **Severity is the producer's claim**, but the curator may overrule based on actual scope of impact during routing.
- **Honesty flags are required when applicable.** `speculative-cause` (the reporter guessed at why the friction happened); `repeats-existing-friction-topic` (this is the same friction the reporter or another agent has filed before — the curator may merge); `minimal-context` (the report was filed under time pressure with reduced context); `auto-generated` (the description was machine-summarized or produced by a skill, not a human-shaped sentence).
- **`recurring` severity requires evidence of recurrence**: count of past occurrences, or pointer to a prior friction entry being amplified.
- **`blocking` severity requires the reporter is currently stopped** (not just slowed). Blockers route via handoff or capability-gap depending on whether a fix exists.
- **`unknown` scope requires curator reclassification before routing.** If reclassification fails after one heartbeat, the curator hands off to debt-curator with full context — never direct-writes to `friction-report/recurring-workaround/*` from `unknown` (that would corrupt debt-curator's synthesis input).
- **Friction entries are observation, not authority.** The destination scoped-topic owners (toolchain-validator, run-introspector, team-agent-optimizer, debt-curator) decide what action follows. The original `friction-inbox/*` entry is closed on routing; the routing record lives in `friction-triage-record/<YYYY-MM-DD>`.

## Action selection

| Action                  | When                                                                                        |
|-------------------------|---------------------------------------------------------------------------------------------|
| `route`                 | Default. Curator writes the entry to `friction-report/<scope>/<date>/<slug>` on the owner's behalf and closes the inbox entry. Routing record lands in friction-triage. |
| `drop`                  | Severity = `one-off` + no clear recurrence pattern. Curator writes a short triage note explaining the drop and removes the inbox entry. Reporter should use handoff next time. |
| `reclassify`            | Scope = `unknown` but evidence supports a clear reclassification. Curator rewrites the topic to `friction-inbox/<real-scope>/<slug>`, then routes on the next heartbeat. |
| `handoff-debt-curator`  | Scope = `unknown` and reclassification failed after one heartbeat, OR `friction-inbox` overflows past `dailyInboxDrainCap`. Curator hands off to debt-curator for synthesis. |
| `overflow-flag`         | Inbox exceeded `dailyInboxDrainCap` (default 25/day). Curator emits a friction-triage entry tagged `inbox-overflow`, hands off to debt-curator, and pauses further routing this heartbeat. |

## Front-matter schemas

### `friction-report` (entry written by reporter via `report-friction` skill)

```yaml
severity: blocking | recurring | one-off
scope: toolchain | run-execution | prompt-team-agent-storage | recurring-workaround | unknown
reporter: <agent-id>
reporter_team: <team-id>
observed_at: <YYYY-MM-DD>
context:
  scenario: <scenario-id-or-null>
  skill: <skill-id-or-null>
  member: <member-id-or-null>
  command: <CLI-command-or-null>
  doc: <doc-path-or-null>
  task: <task-id-or-null>
expected: <one-line description of the promised behavior>
actual: <one-line description of the observed behavior>
description: |
  <free-form notes; what the reporter observed, why it was friction, hypotheses about cause>
honesty_flags: [<flags from honestyFlags below>]
```

### `friction-triage` (daily snapshot written by friction-curator)

```yaml
date: <YYYY-MM-DD>
curator: friction-curator
inbox_received: <int>
routed: <int>
dropped: <int>
reclassified: <int>
handoff_to_debt_curator: <int>
by_scope:
  toolchain: <int>
  run-execution: <int>
  prompt-team-agent-storage: <int>
  recurring-workaround: <int>
  unknown: <int>
by_reporter_team:
  <team-id>: <int>
overflow: <bool>
```

Body required sections: **Routings** (per-entry list of source → destination), **Drops** (one-off entries dropped, with reason), **Blocked** (entries handed off after failed reclassification).

The snapshot uses `supersedesPrevious: true` semantics: each heartbeat overwrites the day's snapshot with the latest aggregated state. End-of-day, the final snapshot is the durable record for that day.

## Honesty flags

Reporter and curator may set:

- `speculative-cause` — the description guesses at why the friction happened without direct evidence.
- `repeats-existing-friction-topic` — this is the same friction the reporter or another agent has filed before. Curator may merge with the existing scoped-topic entry rather than creating a duplicate.
- `minimal-context` — the report was filed under time pressure with reduced context.
- `auto-generated` — the description text was produced by a summarizer or another skill, not a human-shaped sentence. Curator may rate-limit or batch.

## Caps

- **Per-heartbeat producer cap (honor-system, in `report-friction` skill):** at most 3 entries per heartbeat per agent. Above that, group into a single `recurring-workaround` entry.
- **Daily curator drain cap (`dailyInboxDrainCap`, in friction-curator's taskParameters):** default 25/day. Above that, curator emits an `inbox-overflow` triage entry and hands off to debt-curator.

## Cross-references

- [`README.md`](README.md) — meta-optimization team's friction-related canon overview.
- [`docs/scenario-qa/BUG_REPORT_TAXONOMY.md`](../scenario-qa/BUG_REPORT_TAXONOMY.md) — the sister universal observation flow.
- [`docs/agent-system/INTAKE_PIPELINE.md`](../agent-system/INTAKE_PIPELINE.md) — the inbox-router-drain pattern; friction-inbox uses *deterministic-prefix routing* (no classifier skill).
- [`docs/agent-system/TOPICS.md`](../agent-system/TOPICS.md) — registry of every active topic prefix; meta-optimization entries live there.
- [`docs/agent-system/TOPICS_SCHEMA.md`](../agent-system/TOPICS_SCHEMA.md) — schema reference for `topics.json`; documents `source_team: "*"` (universal-source) semantics.
- [`scenarios/prompt-manager/store/skills/packs/core/report-friction/SKILL.md`](../../scenarios/prompt-manager/store/skills/packs/core/report-friction/SKILL.md) — the universal writer skill any agent invokes to file friction.
- [`scenarios/prompt-manager/store/skills/packs/core/conversation-friction-analysis/SKILL.md`](../../scenarios/prompt-manager/store/skills/packs/core/conversation-friction-analysis/SKILL.md) — the deeper post-hoc analysis skill; complementary, not a replacement for in-flight `report-friction`.
