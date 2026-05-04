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

Inbox entries live in the marketing-crew team knowledge log under a `research-inbox/<signal-type>/<slug>` topic. There is no separate JSONL file. Concurrency safety, retention, and querying come from the existing `prompt-manager team knowledge-*` CLI.

Work may enter through:

- the inbox view: `prompt-manager team knowledge-list marketing-crew --topic-prefix=research-inbox/`
- researcher handoff and prior knowledge
- operator-provided source references during a vision walk
- proactive baseline scans when inbox signal is empty or stale
- future bookmark-intelligence-hub CLI exports
- cross-team requests from director-swarm, monetization, or meta-optimization

### Inbox convention

The inbox uses hierarchical topics under `research-inbox/`:

```
research-inbox/<signal-type>/<short-slug>
```

`signal-type` is one of: `audience`, `hook`, `channel`, `competitor`, `workflow`, `skill`, `format`, `funnel`, `benchmark`, `unknown`. The `team knowledge-list --topic-prefix=` filter (added alongside this convention) makes the hierarchy queryable: list the whole inbox with `--topic-prefix=research-inbox/`, or just one signal type with `--topic-prefix=research-inbox/audience/`.

Adding an inbox entry:

```bash
prompt-manager team knowledge-add marketing-crew \
  --by=<source-id> \
  --topic="research-inbox/<signal-type>/<short-slug>" \
  --content="<raw operator note + honesty/confidence flags + optional suggested-method>" \
  --source="<original-url-if-known>"
```

- `--by` identifies the source (e.g. `vision-walk`, `researcher`, `operator`).
- `--source` carries the original URL when available.

Each entry must preserve: source URL when available, raw operator note, confidence / honesty flags, and a proposed next method (or a reason no follow-up is warranted). Signal-type lives in the topic, not the content.

### Routing inbox entries

After the router classifies an entry, it must either retag or delete it:

- **Retag** when the entry was promoted to a permanent observation:

  ```bash
  prompt-manager team knowledge-update marketing-crew <id> --topic="<destination-topic>"
  ```

  e.g. `research-inbox/audience/foo` → `audience-scan/foo`. Destination topics use the canonical prefix for the surface: `audience-scan/<slug>`, `competitor-record/<slug>`, `hook-record/<slug>`, `monetization-benchmark-adjacent-record/<slug>`, etc. If the routed action creates a new entry on a different surface (decision, notebook debt, capability-gap), delete the inbox row instead of retagging.

- **Delete** when the entry was weak, duplicate, or otherwise dropped:

  ```bash
  prompt-manager team knowledge-delete marketing-crew <id>
  ```

The inbox view (`--topic-prefix=research-inbox/`) is therefore always the unrouted set. Permanent canon never uses any topic under `research-inbox/`; it lives under whichever destination prefix matches the routed surface.

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
| Low-signal or one-off item | Handoff, or `research-inbox/*` knowledge entry. |
| Concrete observation with source | `audience-scan/*` knowledge entry, or `audience-scans.jsonl` for batch scan rows. |
| Unresolved repeated pattern | `docs/marketing/notebook/*`. |
| Audience/persona change with converging evidence | `audience-update` decision. |
| Channel priority or activation change | `channel-strategy-update` decision. |
| New post format with enough evidence | `post-type-proposal` decision. |
| Stable reusable hook | `hook-candidate-promotion` decision. |
| External skill/workflow worth operationalizing | skill/action/scenario proposal routed through the right team. |
| Missing source access, CLI, action, or scenario blocks research | `capability-gap` decision. |
| Monetization-relevant but not marketing-owned fact | `monetization-benchmark-adjacent-record/<topic>` knowledge entry. |

## Evidence Rules

- Never fabricate engagement, revenue, conversion, audience-size, or pricing facts.
- One source is an observation, not canon.
- Three converging scans can justify a decision when the sources are meaningfully independent.
- Single-snapshot data must be labeled `light-interpretation`.
- Bookmark-intelligence-hub classifications are inputs, not proof. The researcher still evaluates relevance and evidence quality.
- Researcher never drafts marketing content or directly edits `docs/marketing/` canon; it proposes changes for operator review.

## Skill Surface

Inbox draining is now a structural concern, not a single skill:

- `docs/marketing/SIGNAL_TAXONOMY.md` (paired with `signal-taxonomy.json`) — owns the signal vocabulary, dispatch table, evidence rules, and destination schemas.
- `marketing-signal-classifier` — pure-judgment skill that returns `signal_type`, `evidence_strength`, `honesty_flags`, and a `recommended_method` for each raw item. Member-agnostic; no inbox or destination coupling.
- The researcher's heartbeat receives a generated `# Inbox Flow` section that names the prefix, classifier, destinations, and dispatch.

Planned focused method skills (audience-pain-mining, competitor-positioning-scan, hook-pattern-mining, etc.) cite the taxonomy as their PoR hub and keep procedure in the skill, not in this document.
