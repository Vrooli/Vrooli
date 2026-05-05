# Marketing — Working Notebook

This folder is the **working notebook** of the `marketing-crew` team. Unlike [`path:docs/marketing/`](../) (the plan-of-record — canon the team is bound to), these files are a living record of workarounds, techniques, and one-off observations the team accumulates as it runs.

## Posture: debt, not gospel

Every entry in these docs is prose describing *something that should eventually be permanent structure*. A workaround in `VIDEO_WORKAROUNDS.md` is a note saying "we don't have `video-studio` scenario yet, so here's how we produce videos by hand." A pattern in `DEV_LOG_CRAFT.md` is an observation about how dev logs land that should eventually feed back into the `x-dev-log` skill.

Entries are **technical debt**. They exist because the permanent solution — a skill, a scenario feature, a plan-of-record addition — doesn't exist yet. The team's `brand-manager` member periodically scans these docs and proposes promoting mature entries into permanent structure.

A notebook entry has exactly **three promotion targets** (per [`docs/agent-system/TEAM_DOCS_PATTERNS.md`](../../agent-system/TEAM_DOCS_PATTERNS.md)) — and the curator picks the right one per entry, not all entries go to the same target:

1. **A skill** in `path:scenarios/prompt-manager/store/skills/packs/core/<skill>/` — when the pattern is *executable agent guidance* (e.g., "x-dev-log should mine X when Y is unavailable" → edit to `x-dev-log` SKILL.md). Most `*_CRAFT.md` entries promote here.
2. **A plan-of-record file** under `path:docs/marketing/` (including `post-types/` and `post-techniques/` sub-folders) — when the pattern is *strategic canon* the operator reads to make decisions (audience refinements, new technique rules, new failure mode at type level).
3. **A scenario / config change** — when the pattern reflects something the harness or a scenario should do automatically (e.g., a `social-media-scheduler` feature replacing a manual workaround in `POSTING_WORKAROUNDS.md`).

Promotion to *any* target deletes the notebook entry as part of the same `notebook-promotion` decision. An entry never lives in both notebook and promoted form simultaneously. Retirement (no promotion, just delete) is the fourth outcome when the pattern proves transient.

The goal is **shrinking**, not growing, documentation over time. Good weeks end with fewer entries than they started with.

## Who writes what

- **All marketing-crew members** may freely *append* entries to these docs as they discover workarounds or patterns. No ceremony, no approval.
- **Nobody** rewrites or deletes entries directly. Retirements and promotions go through `notebook-retirement` / `notebook-promotion` decisions (owned by `brand-manager`), so the operator sees what's being retired, promoted, and why.
- **Operator** curates when accepting retirement/promotion decisions; may reorganize for clarity at any time.

This split avoids the failure mode where members edit each other's notes and the docs become an unreliable consensus slurry.

## Current files

| File | Purpose | Character |
|------|---------|-----------|
| `VIDEO_WORKAROUNDS.md` | How we produce videos without `video-studio` scenario shipped | Memory — grows as workarounds accumulate; shrinks on promotion |
| `POSTING_WORKAROUNDS.md` | Manual cross-posting recipes while `social-media-scheduler` integration is partial | Memory |
| `AUDIENCE_OBSERVATIONS.md` | Raw audience/competitor/trend observations pre-structure | Memory — may feed researcher's structured scans + `AUDIENCES.md` |
| `CAMPAIGN_LESSONS.md` | Post-campaign reflections awaiting distillation into permanent patterns | Memory |
| `DEV_LOG_CRAFT.md` | Patterns observed using the `x-dev-log` skill that should feed back into improving it | Memory — promotion target is `x-dev-log` skill edits |

## What is NOT in these docs

- **Team operating rules** — those live in `scenarios/prompt-manager/store/teams/marketing-crew/shared/TEAM.md` because they define the team's structure. They're part of the code, not the notebook.
- **Brand / voice canon** — those live in `docs/marketing/STRATEGY.md`, `AUDIENCES.md`, etc. Plan-of-record is the home for anything other teams / scenarios read as authoritative.
- **Per-heartbeat state** — campaign-drafts, publish-log, audience-scans, knowledge snapshots live under `path:scenarios/prompt-manager/store/teams/marketing-crew/shared/`. That's the *hot buffer*. When a pattern in the hot buffer stabilizes, it may get distilled into these docs; when a doc entry matures, the `brand-manager` proposes it for promotion into permanent structure.

## The three-tier mental model

```
Hot buffer (per-heartbeat)      Living notebook (distilled)     Permanent structure
─────────────────────────  →    ──────────────────────────  →   ──────────────────────
shared/campaign-drafts.jsonl    docs/marketing/notebook/         Skills, scenarios, plan-of-record
shared/publish-log.jsonl        VIDEO_WORKAROUNDS.md             prompt-manager skills/
shared/audience-scans.jsonl     POSTING_WORKAROUNDS.md           scenarios/video-studio (eventual)
shared/knowledge.jsonl          AUDIENCE_OBSERVATIONS.md         scenarios/social-media-scheduler
...                             CAMPAIGN_LESSONS.md              docs/marketing/*.md (plan-of-record)
                                DEV_LOG_CRAFT.md
```

Observations flow left-to-right over time. Each arrow is someone distilling or promoting. When the rightmost column gains a capability (e.g., `video-studio` ships), the corresponding middle-column entry becomes retirement-eligible.

## Revisit markers

Early-stage docs should say "revisit this section after N heartbeats" or "revisit when scenario X ships." When the marker fires, the `brand-manager` evaluates whether the section has stabilized enough to promote, needs revision, or should be retired. Operating rule 15 makes revisit markers mandatory.

## Cross-references

- `docs/agent-system/TEAM_DOCS_PATTERNS.md` — the two-pattern definition (canon).
- `docs/agent-system/INTAKE_PIPELINE.md` — the inbox-router-drain pattern that has largely replaced markdown notebooks across the agent system.
- `scenarios/prompt-manager/store/teams/marketing-crew/shared/TEAM.md` — live operating rules.
