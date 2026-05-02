## Tools focus: Marketing Research Router

Classify marketing research intake and route each signal to the right next action: observation, notebook debt, focused research method, decision, skill proposal, or capability-gap. This skill is the router, not the permanent home for every research method.

> **Status:** v1 router. The plan-of-record architecture lives in `docs/marketing/research/README.md`. Focused method skills should be added over time as repeated methods emerge.

---

### 1. When To Use

Use this skill when the marketing-crew researcher needs to process:

- operator-fed alpha from a morning vision walk
- source links, bookmark exports, or social posts
- proactive baseline scan findings
- competitor, audience, channel, hook, workflow, skill, or benchmark-adjacent signals
- a handoff that says research is blocked by missing collection capability

Do not use this skill to draft marketing content. Do not use it as a replacement for a focused method skill once the method is stable.

---

### 2. Required Reading

Read first:

- `docs/marketing/research/README.md`
- inbox view: `prompt-manager team knowledge-list marketing-crew --topic-prefix=research-inbox/ --json` (filter to a single signal type with `--topic-prefix=research-inbox/<signal-type>/`)
- `scenarios/prompt-manager/store/teams/marketing-crew/members/researcher/last-handoff.md`
- recent `audience-scan-*` and `monetization-benchmark-adjacent/*` knowledge entries
- owned pending decisions for `researcher`

Read as needed:

- `docs/marketing/AUDIENCES.md`
- `docs/marketing/CHANNELS.md`
- `docs/marketing/strategies/hook-library.md`
- `docs/marketing/post-types/README.md`
- `docs/marketing/notebook/AUDIENCE_OBSERVATIONS.md`
- `docs/monetization/BENCHMARKS.md`

Planned preferred collection source:

- bookmark-intelligence-hub CLI output, once the scenario exists. Use it for archived bookmarks and social saves rather than creating source-specific intake skills for each platform.

---

### 3. Inputs

Inputs can be supplied by the caller or discovered from the required reading:

| Input | Required | Notes |
|---|---|---|
| `source_items` | No | Links, bookmark export rows, operator notes, or inbox entries. |
| `scan_mode` | No | `inbox-first`, `proactive-baseline`, or `specific-source`. Default: `inbox-first`. |
| `research_goal` | No | Audience, competitor, hook, workflow, skill, channel, funnel, benchmark, or unknown. |
| `time_window` | No | Relevant for proactive scans or bookmark-intelligence-hub exports. |

If source items are present, analyze those first. If no source items are present and the inbox is empty or stale, run a small proactive baseline scan with cited sources.

---

### 4. Routing Process

1. **Normalize intake.** Convert each item into:
   - `source`
   - `source_url` if available
   - `raw_note`
   - `initial_type`
   - `evidence_strength`
   - `honesty_flags`

2. **Classify signal type.**

| Signal type | Typical method |
|---|---|
| audience frustration, buying trigger, user vocabulary | `audience-pain-mining` |
| external process, playbook, agent setup, business workflow | `workflow-deconstruction` |
| external skill or reusable prompt/process | `skill-opportunity-scan` |
| competitor pricing, packaging, positioning, changelog | `competitor-positioning-scan` |
| repeated channel-native format | `channel-format-scan` |
| reusable opening, framing, or copy pattern | `hook-pattern-mining` |
| lead magnet, trial, demo, pricing anchor, funnel | `offer-and-funnel-scan` |
| comparable pricing or market fact | `benchmark-adjacent-scan` |
| new marketing post format | `post-type-discovery` |

3. **Choose the smallest useful action.**

| Condition | Action |
|---|---|
| Weak one-off signal | Mention in handoff, or leave as a knowledge entry under `research-inbox/<signal-type>/<slug>`. |
| Concrete sourced observation | Add a `<topic>/*` knowledge entry (e.g. `audience-scan/*`, `competitor/*`); use `audience-scans.jsonl` only for batch scan rows. |
| Repeated but unresolved pattern | Append notebook debt. |
| Converging evidence meets threshold | Raise the owned decision context. |
| Repeatable method has no skill | Propose a skill through the normal meta-optimization path. |
| Collection requires missing source/tool/scenario | Raise `capability-gap`. |
| Signal belongs to monetization | Write benchmark-adjacent knowledge without editing monetization canon. |

4. **Apply collection discipline.**
   - Prefer supplied source refs and bookmark-intelligence-hub exports when available.
   - Do not create platform-specific intake skills as a default strategy.
   - Use manual web research only as a baseline fallback.
   - Label single-snapshot findings `light-interpretation`.
   - If source access is blocked, raise a capability gap instead of pretending the scan happened.

5. **Resolve the inbox entry.** Every routed item must leave the inbox view (`team knowledge-list marketing-crew --topic-prefix=research-inbox/`) in one of two ways:
   - **Promoted** to permanent canon — retag the entry to its destination topic:
     ```bash
     prompt-manager team knowledge-update marketing-crew <id> --topic="<destination-topic>"
     ```
     e.g. `research-inbox/audience/foo` → `audience-scan/foo`. Destination topics use the canonical prefix for the surface (`audience-scan/<slug>`, `competitor/<slug>`, `hook/<slug>`, `monetization-benchmark-adjacent/<slug>`, etc.). If the routed action creates a *new* entry on a different surface (decision, notebook debt, capability-gap), delete the inbox row instead of retagging.
   - **Dropped** as weak/duplicate/out-of-scope:
     ```bash
     prompt-manager team knowledge-delete marketing-crew <id>
     ```
   Do not leave entries under any `research-inbox/*` topic after routing — that breaks the unrouted-set invariant.

6. **Emit output.** Provide a concise routing summary for the researcher heartbeat.

---

### 5. Output Contract

```markdown
### Research Routing Summary

**Inputs reviewed:** <count and source modes>

**Routed items:**
- `<id/source>` -> `<signal_type>` -> `<action>`; evidence=`<strength>`; flags=`<flags>`

**Observations to write:**
- <knowledge/audience-scan/notebook target and short content>

**Decision candidates:**
- <context, rationale, evidence threshold status>

**Skill or capability gaps:**
- <proposed skill/action/scenario/capability-gap, if any>

**Proactive baseline:**
- <run/not run and why>
```

No known operational edge cases for standard usage.
