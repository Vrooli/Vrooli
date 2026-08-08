# Bug Report Triage Taxonomy

Cross-team-readable canon for how the `literal:scenario-qa/bug-investigator` member classifies and resolves entries written to `bug-inbox/*`. Human-readable view of [`taxonomy.json`](taxonomy.json).

**Owner team:** scenario-qa. **Status:** canon. Operator-curated via `bug-resolution-proposal` decisions on the scenario-qa team.

Cited by:
- `scenarios/prompt-manager/store/teams/scenario-qa/members/bug-investigator/topics.json` — `intake[].taxonomy = "bug-report"`.
- `scenarios/prompt-manager/store/skills/packs/core/report-bug/SKILL.md` — required reading; the skill writes entries that conform to the `bug-report` schema below.

Universal-source intake: `bug-inbox/*` declares `source_team: "*"` — any team's members may write. The producer-side anchor is the `report-bug` skill (declared in the bug-investigator's `external_producers` so the validator's `wildcard_source_misuse` rule stays quiet).

## Topic shape

```
bug-inbox/<signal-type>/<short-slug>
```

The producer (the agent invoking `report-bug`) picks the signal type from the fixed list below; the investigator validates the choice as the first step of investigation. There is **no separate classifier skill** — bug-inbox uses deterministic-prefix routing because investigation includes classification as its first sub-step. (Compare: `marketing-research` uses a separate classifier because the producer is a research scan that doesn't yet investigate. Bugs are different — every drainer also reads the entry to start work.)

## Signal types

| Signal type           | Definition                                                                   |
|-----------------------|------------------------------------------------------------------------------|
| `code-defect`         | Broken code or scenario behavior — implementation doesn't match intent.      |
| `regression`          | Worked previously; now broken. Has a known-good prior state.                 |
| `prompt-confusion`    | Agent misled by ambiguous/contradictory prompt content (skill, doc, etc.).   |
| `data-shape-mismatch` | Observed payload didn't match documented schema.                             |
| `unexpected-error`    | Error not documented anywhere — error code or exception no skill anticipated.|
| `unknown`             | Type unclear; investigator classifies during triage.                         |

Default investigation method for every signal type: `scientific-debugging` (see [`../../methods/investigation/scientific-debugging.md`](../../methods/investigation/scientific-debugging.md)).

## Severity

The reporter assigns severity:

- `blocker` — work is stopped; no workaround exists.
- `major` — work continues but the bug significantly degrades a flow.
- `minor` — annoyance; workaround exists.

The investigator may overrule severity based on actual scope of impact discovered during investigation (e.g., a "minor" report whose root cause is a corrupted invariant becomes a blocker).

## Evidence rules

- **A repro is mandatory.** If absent, the investigator first tries to reproduce. If reproduction fails, the action is `file-work` asking the reporter or operator to supply repro — not `drop`.
- **Honesty flags are required when applicable.** If the reporter didn't try to reproduce, the entry must include `repro-not-attempted`. Other flags: `speculative-cause` (the reporter guessed at a cause), `minimal-context` (the reporter shipped a short report under time pressure), `ai-generated-summary` (the description was machine-summarized).
- **Investigation outcomes cite the technique applied.** The `bug-investigation-report/<slug>` entry's `technique` front-matter field must name a registered investigation technique. This drives technique-graduation decisions on `meta-self-improvement` (e.g., "bisect-debugging recurred 8 times this month — promote to a registered technique").
- **Bug entries are debt, not authority.** Investigation outcomes (`bug-investigation-report/<slug>`) are the durable audit log; the original bug-inbox entry is moved or deleted on close.

## Action selection

| Action                  | When                                                                                                            |
|-------------------------|-----------------------------------------------------------------------------------------------------------------|
| `drop`                  | Cannot reproduce + no clear scope; weak one-off. The investigator writes a short bug-investigation entry explaining the drop and deletes the bug-inbox entry. |
| `observe`               | Confirmed bug; findings recorded in `bug-investigation-report/<slug>` but no fix this heartbeat (deprecated path, queued elsewhere, etc.). |
| `file-backlog`          | Reproducible; investigator files a `swarm-manager` backlog item whose description states the outcome and its done-condition, attaches the investigation evidence as the item's `research/summary.md`, then closes the bug-inbox entry. Do not put the investigation narrative in `description` — see `swarm-manager-backlog-tools` §"Writing Standard". |
| `file-work`         | Cross-cutting; investigator raises `bug-resolution-proposal` for operator review (e.g., a recurring confusion that suggests renaming a CLI verb). |
| `route-to-another-topic`| Misclassified — actually a documentation gap, capability-work, or skill-issue. Retag/rewrite to the appropriate inbox or file the appropriate decision. |
| `capability-work`        | Repro requires a missing tool/scenario/CLI; raise a `capability-work` decision and leave the inbox entry until the gap is closed. |

## Front-matter schemas

### `bug-report` (entry written by reporter via `report-bug` skill)

```yaml
severity: blocker | major | minor
reporter: <agent-id>
reporter_team: <team-id>
observed_at: <YYYY-MM-DD>
context:
  scenario: <scenario-id-or-null>
  skill: <skill-id-or-null>
  member: <member-id-or-null>
  command: <CLI-command-or-null>
repro:
  - <step 1>
  - <step 2>
expected: <one-line description>
actual: <one-line description>
description: |
  <free-form notes; what the reporter observed, hypotheses tried, etc.>
honesty_flags: [<flags from honestyFlags below>]
```

### `bug-investigation` (entry written by investigator on close)

```yaml
bug_id: <source-knw-id of the bug-inbox entry>
investigator: <agent-id>
technique: <technique-skill-id, e.g., scientific-debugging>
outcome: drop | observe | file-backlog | file-work | route-to-another-topic | capability-work
root_cause: <text-or-null>
fix_target: <text-or-null>
closed_at: <YYYY-MM-DD>
```

Body required sections: **Findings**, **Action taken**.

## Honesty flags

Reporter and investigator may set:

- `repro-not-attempted` — the reporter did not try to reproduce before filing.
- `speculative-cause` — the description guesses at a cause without evidence.
- `minimal-context` — the report was filed under time pressure with reduced context.
- `ai-generated-summary` — the description text was produced by a summarizer.

## Cross-references

- [`README.md`](README.md) — scenario-qa team plan-of-record overview.
- [`../../methods/investigation/README.md`](../../methods/investigation/README.md) — registry of techniques the investigator may apply.
- [`../../methods/investigation/scientific-debugging.md`](../../methods/investigation/scientific-debugging.md) — the default technique for every signal type.
- [`docs/agent-system/INTAKE_PIPELINE.md`](../agent-system/INTAKE_PIPELINE.md) — the inbox-router-drain pattern; bug-inbox uses *deterministic-prefix routing* (no classifier skill).
- [`docs/agent-system/TOPICS.md`](../agent-system/TOPICS.md) — scenario-qa registry; bug-inbox listing.
- [`scenarios/prompt-manager/store/skills/packs/core/report-bug/SKILL.md`](../../scenarios/prompt-manager/store/skills/packs/core/report-bug/SKILL.md) — the universal writer skill any agent invokes to file a bug.
