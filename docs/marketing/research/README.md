# Marketing Research

This folder is the plan-of-record hub for how the marketing-crew researcher turns external signals into evidence, decisions, skills, and capability-gap proposals.

Research supports two intake modes:

- **Operator-fed alpha:** links, bookmarks, workflows, skills, posts, and observations surfaced during the morning vision walk or direct operator conversation.
- **Proactive baseline:** low-volume scheduled scans across audiences, competitors, channels, hooks, formats, and benchmark-adjacent market evidence.

When the bookmark-intelligence-hub scenario is available, it is the preferred collection layer for operator bookmarks and social saves. Do not create source-specific intake skills for X, Reddit, newsletters, or similar sources unless bookmark-intelligence-hub cannot reasonably cover that source. The researcher should consume classified bookmark-intelligence-hub CLI output, then apply marketing research methods to the relevant items.

## Pipeline

```text
Intake -> Collection -> Analysis Method -> Promotion / Routing
```

## Intake

Work may enter through:

- `scenarios/prompt-manager/store/teams/marketing-crew/shared/research-inbox.jsonl`
- researcher handoff and prior knowledge
- operator-provided source references during a vision walk
- proactive baseline scans when inbox signal is empty or stale
- future bookmark-intelligence-hub CLI exports
- cross-team requests from director-swarm, monetization, or meta-optimization

Each research item should preserve:

- source and source URL when available
- raw operator note or source summary
- initial signal type
- confidence / honesty flags
- proposed next method or reason no follow-up is warranted

## Collection

Collection gathers source material before analysis.

Current collection modes:

- supplied source references from the operator or research inbox
- manual web research with cited sources
- internal logs, knowledge, campaign drafts, and publish state

Planned collection mode:

- bookmark-intelligence-hub CLI output for archived bookmarks, saved social posts, and source classification

Collection should become a tool, action, or scenario capability when it is credentialed, scheduled, reusable, or deterministic. Until then, method skills may include lightweight collection steps, but they must label weak evidence honestly.

## Analysis Methods

Initial method registry:

| Method | Status | Use |
|---|---|---|
| `alpha-extraction` | planned | Structure raw operator discoveries from vision walk or direct notes. |
| `audience-pain-mining` | planned | Extract pains, buying triggers, vocabulary, urgency, and audience segmentation. |
| `workflow-deconstruction` | planned | Turn external workflows, processes, and agent setups into reusable Vrooli opportunities. |
| `competitor-positioning-scan` | planned | Compare competitor claims, omissions, packaging, and positioning wedges. |
| `channel-format-scan` | planned | Identify channel-native formats worth testing or documenting. |
| `hook-pattern-mining` | planned | Extract repeatable hooks for `docs/marketing/strategies/hook-library.md`. |
| `offer-and-funnel-scan` | planned | Analyze lead magnets, trials, pricing anchors, demos, and funnel mechanics. |
| `skill-opportunity-scan` | planned | Evaluate external skills or workflows as Vrooli skills, actions, scenarios, or rejected patterns. |
| `benchmark-adjacent-scan` | planned | Capture comparable market/pricing facts without treating them as direct Vrooli telemetry. |

Do not collapse these into one permanent mega-skill. A router may classify and choose methods, but repeatable methods should become focused skills when they recur.

## Promotion Matrix

| Signal | Destination |
|---|---|
| Low-signal or one-off item | Handoff or research inbox note. |
| Concrete observation with source | `knowledge.jsonl` or `audience-scans.jsonl`. |
| Unresolved repeated pattern | `docs/marketing/notebook/*`. |
| Audience/persona change with converging evidence | `audience-update` decision. |
| Channel priority or activation change | `channel-strategy-update` decision. |
| New post format with enough evidence | `post-type-proposal` decision. |
| Stable reusable hook | `hook-candidate-promotion` decision. |
| External skill/workflow worth operationalizing | skill/action/scenario proposal routed through the right team. |
| Missing source access, CLI, action, or scenario blocks research | `capability-gap` decision. |
| Monetization-relevant but not marketing-owned fact | `monetization-benchmark-adjacent/<topic>` knowledge entry. |

## Evidence Rules

- Never fabricate engagement, revenue, conversion, audience-size, or pricing facts.
- One source is an observation, not canon.
- Three converging scans can justify a decision when the sources are meaningfully independent.
- Single-snapshot data must be labeled `light-interpretation`.
- Bookmark-intelligence-hub classifications are inputs, not proof. The researcher still evaluates relevance and evidence quality.
- Researcher never drafts marketing content or directly edits `docs/marketing/` canon; it proposes changes for operator review.

## Skill Surface

Current executable skill:

- `marketing-research-router` - classify intake and choose observation, method, decision, skill proposal, or capability-gap routing.

Planned focused skills should cite this file as their plan-of-record hub and keep procedure in the skill, not in this document.
