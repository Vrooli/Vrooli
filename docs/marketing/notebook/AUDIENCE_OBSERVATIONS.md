# Audience Observations

Raw, pre-structured audience / competitor / trend observations that don't yet meet the threshold for a `shared/audience-scans.jsonl` entry OR don't yet have a clear persona key to attach to.

**Promotion target:**
- If an observation stabilizes and attaches to a persona → `audience-update` decision → `AUDIENCES.md` edit.
- If an observation is benchmark-adjacent for monetization → cross-team knowledge entry under `monetization-benchmark-adjacent/<topic>`.
- If an observation is a pattern about how to do research itself → `documentation-health` skill update or new skill proposal.

**Retirement signal:** observation has been promoted into a structured scan, an `AUDIENCES.md` revision, or a market-validator-consumed entry — AND ≥30 heartbeats have passed without the observation being cited again.

**Revisit marker (file-level):** revisit after 14 heartbeats.

## Entries

### 2026-04-27 — Bundle-as-subscription-savings positioning (homelab-built-with-Claude-Code signal)

**Written by:** operator (captured via vision walk #4 chore-audit)
**Observation:** Viral X/social-media post covered someone using Claude Code (Anthropic's coding agent) to set up a personal homelab containing multiple self-hosted services (media server and other typical homelab apps), with the explicit framing of *avoiding multiple SaaS subscriptions*. This signal generates two distinct insights:

1. **Family-bundle positioning angle.** Vrooli's family bundle is structurally a *superset / AI-powered version* of a traditional homelab — it serves the same purpose (run things locally, save on subscriptions, retain control) but is agent-driven rather than requiring the technical expertise to wire up separate self-hosted apps. *"AI-powered homelab that handles your home life"* is a viable positioning lead for the family bundle. The audience that's currently building homelabs by hand is a natural target — they already accept the local-first thesis; Vrooli reduces the technical-overhead cost.

2. **Business-bundle parallel positioning angle.** The same subscription-savings logic applies to the business bundle (developer / solopreneur tools). Buying the Vrooli business bundle *replaces* the stack of SaaS subscriptions a solopreneur would otherwise pay for (analytics, landing-page-builder, marketing automation, etc.). *"One bundle subscription, not N SaaS subscriptions"* is a positioning lead specifically for this audience. This had not been explicitly captured in canonical positioning before — surface during next monetization / brand-canon update cycle.

Both positioning angles are anchored on the same audience truth: knowledge workers and household operators are subscription-fatigued and increasingly suspicious of cloud SaaS lock-in. The Vrooli local-first thesis aligns with that fatigue better than alternative agent platforms (which mostly assume cloud hosting).

**Source:** Bookmark from operator's social media stream (URL not captured — pre-BIH-rework workflow). Operator notes: *"this is one valuable way we can market this project — that you can do everything locally."* Cross-references: existing `STRATEGY.md` dual-audience framing already covers local-sovereignty as values tagline; the bundle-as-subscription-savings angle is the new addition.
**Interpretation flag:** observation + positioning-insight
**Possible persona attachment:** new candidate persona — "homelab-builder" — for family bundle. May also attach to existing OSS-contributor persona for the credibility angle.
**Cross-team relevance:** monetization-benchmark-adjacent (subscription-savings is a pricing / value-anchor angle worth surfacing to monetization for both bundles); director-swarm (research method — see knowledge entry `research-method/homelab-app-scenario-seeding` for the generative method this signal seeded).

**Promotion targets:**
- Family-bundle audience addition to `AUDIENCES.md` (homelab-builder persona) — propose via `audience-update` decision when accumulated evidence supports the persona.
- Business-bundle subscription-savings positioning addition to `STRATEGY.md` and/or `docs/narrative/PITCH.md` — propose via `brand-guideline-update` decision (could surface alongside next bundle launch).

**Revisit marker:** revisit after 14 heartbeats. Watch for additional bookmarks/signals reinforcing the homelab-builder audience or the subscription-fatigue framing — promotion threshold is ≥3 converging signals per operating rules.

---

### 2026-04-24 — Researcher has no structured scanning tooling

**Written by:** researcher
**Observation:** First researcher heartbeat surfaced that no scenario/CLI capability exists for structured competitive-intel or audience-intel scanning. No wire into X/Twitter search, GitHub trending / topic-search, competitor pricing-page monitoring, or industry-blog RSS. `seo-optimizer` skill is keyword-focused and does not cover structured competitor monitoring. Manual reads are possible but slow, hard to triangulate (operating rule: ≥3 converging scans for `audience-update`), and easy to fabricate under time pressure. Workaround this heartbeat: record null-signal scan entry + raise capability-gap rather than invent observations.
**Source:** `shared/audience-scans.jsonl` (empty before this heartbeat); `docs/monetization/BENCHMARKS.md` (all categories empty — no data source to lean on); `TOOLS.md` (no scanning scenario listed).
**Interpretation flag:** observation
**Possible persona attachment:** null (meta-capability, not persona signal)
**Cross-team relevance:** null (not monetization-benchmark-adjacent — this is about researcher tooling itself)
**Revisit marker:** revisit when a competitive-intel scenario lands OR when scanning tooling is wired into this heartbeat (paired with `capability-gap` decision this heartbeat).

### 2026-04-24 — WebSearch provides baseline, not structured, scanning

**Written by:** researcher
**Observation:** Same-day follow-on: WebSearch tooling is available via the Claude runtime and produced citation-grounded baseline scans this heartbeat (4 entries appended, covering Cursor credit-pricing, BYOK-vs-subscription threshold, OSS agent-framework landscape, CrewAI learning-curve anchor — see `shared/audience-scans.jsonl` as-1777062700091991558 through as-1777062700091991561). WebSearch is sufficient for ad-hoc general-web baseline queries but does not replace the structured competitive-intel capability described in the prior entry (no diff detection on pricing pages, no authored-post tracking on X/Twitter, no GitHub topic monitoring with persistence across heartbeats). The prior capability-gap remains valid; its scope narrows from "can't scan at all" to "can't structurally track without manual re-querying."
**Source:** `shared/audience-scans.jsonl` (5 entries after this heartbeat); WebSearch tool availability confirmed in heartbeat 2026-04-24T20:31Z.
**Interpretation flag:** observation
**Possible persona attachment:** null (meta-capability)
**Cross-team relevance:** null
**Revisit marker:** revisit when a competitive-intel scenario lands OR after 10 heartbeats if manual WebSearch continues to be the primary mode.

Append new entries in the shape below:

```markdown
### <YYYY-MM-DD> — <short description>

**Written by:** <member-id, typically researcher>
**Observation:** <concrete, citation-grounded>
**Source:** <URL, post ref, repo link>
**Interpretation flag:** observation | light-interpretation | heavy-interpretation
**Possible persona attachment:** <persona-key-or-null>
**Cross-team relevance:** <null or "monetization-benchmark-adjacent/<topic>">
**Revisit marker:** revisit after N heartbeats
```
