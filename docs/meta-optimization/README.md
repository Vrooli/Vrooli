# Meta Optimization — Living Docs

This folder is the **working notebook** of the meta-optimization team. Unlike `docs/monetization/` (which is doctrine — stable principles the team is bound to), these files are a living record of workarounds, techniques, patterns, and one-off observations the team accumulates as it runs.

## Posture: debt, not gospel

Every entry in these docs is prose describing *something that should eventually be permanent structure*. A workaround in `CONVERSION_PLAYBOOK.md` is a note saying "we don't have a skill for this yet, so here's how we do it by hand." A threshold in `DEPRECATION_POLICY.md` is a rule we apply until we have tooling that enforces it.

Entries are **technical debt**. They exist because the permanent solution — a skill, a scenario feature, a team-config change — doesn't exist yet. The team's `debt-curator` member periodically scans these docs and proposes promoting mature entries into permanent structure.

The goal is **shrinking**, not growing, documentation over time. Good weeks end with fewer entries than they started with.

## Who writes what

- **All meta-optimization members** may freely *append* entries to these docs as they learn things. No ceremony, no approval.
- **Nobody** rewrites or deletes entries directly. Retirements go through `meta-self-improvement` decisions (owned by `debt-curator`), so the operator sees what's being retired and why.
- **Operator** curates when accepting retirement decisions; may reorganize for clarity at any time.

This split avoids the failure mode where members edit each other's notes and the docs become an unreliable consensus slurry.

## Current files

| File | Purpose | Character |
|------|---------|-----------|
| `CONVERSION_PLAYBOOK.md` | Patterns and examples for converting deterministic prose into Vrooli-controlled CLIs and Actions | Memory — grows as conversions happen |
| `DEPRECATION_POLICY.md` | Staleness windows, roadmap-check procedure, archive path for retiring skills/agents/teams | Doctrine + memory — thresholds are rules; edge cases are memory |
| `REFERENCE_SCENARIOS.md` | Which scenario is the gold-star validation target, plus secondary references | State — registry of current references |

## What is NOT in these docs

- **Team operating rules** — those live in `scenarios/prompt-manager/store/teams/meta-optimization/shared/TEAM.md` because they define the team's structure. They're part of the code, not the notebook.
- **Per-heartbeat state** — audits, run lessons, toolchain scans, decision queues, and knowledge snapshots all live under `store/teams/meta-optimization/shared/`. That's the *hot buffer*. When a pattern in the hot buffer stabilizes, it may get distilled into these docs; when a doc entry matures, the `debt-curator` proposes it for promotion into permanent structure.

## The three-tier mental model

```
Hot buffer (per-heartbeat)      Living notebook (distilled)     Permanent structure
─────────────────────────  →    ──────────────────────────  →   ──────────────────────
store/teams/.../shared/         docs/meta-optimization/          Skills, scenarios, team config
RUN_LESSONS.md                  CONVERSION_PLAYBOOK.md           prompt-manager skills/
TOOLCHAIN_SCAN.md               DEPRECATION_POLICY.md            scenarios/*/cli/
SKILL_AUDIT.md                  REFERENCE_SCENARIOS.md           team.json / roles.json
...                             (and future files)
```

Observations flow left-to-right over time. Each arrow is someone distilling or promoting. When the rightmost column gains a capability, the corresponding middle-column entry becomes retirement-eligible.

## Revisit markers

Early-stage docs should say "revisit this section after N heartbeats / after N examples." When the marker fires, the `debt-curator` evaluates whether the section has stabilized enough to promote, needs revision, or should be retired.
