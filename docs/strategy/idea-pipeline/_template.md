<!--
  Idea-pipeline entry template.

  Copy this file to a new folder named with the idea's slug (kebab-case),
  rename to README.md, and fill in the Tier 1 fields. Tier 2 sections may
  be left empty until the entry reaches `evaluating` status — empty is an
  honest signal that this hasn't been evaluated yet.

  When a Tier 2 section grows past ~500 words, split it into its own file
  (`monetization-framing.md`, `marketing-framing.md`, etc.) inside the
  idea's folder. Until then, keep everything in this single file.

  Read the parent README.md before authoring an entry. The operator-only
  write rule applies — agents propose entries via decisions, operator writes.
-->

---
name: <slug-of-the-idea>
summary: <one-sentence what this is>
source: <vision-walk-N-chore-phase | vision-walk-N-bigpicture-phase | social-media-alpha:<url> | opportunity-scout:<opp-id> | marketing-research | bih-extraction:<id> | operator-notes | team-member-suggestion:<team>:<member> | other>
sourced_at: <YYYY-MM-DD>
status: raw
promoted_to: null
retired_reason: null
retired_at: null
---

# <Title — readable form of the slug>

<One-paragraph description: what this idea is, why it might matter, what's known so far. Keep it tight; deepen via Tier 2 sections.>

## Monetization framing

*Tier 2 — fill when status reaches `evaluating`. Leave the section heading and this italic placeholder if not yet evaluated.*

<!--
When fillable, cover:
- How would this generate revenue, OR if not directly monetizable, what does it enable?
- Which existing SKU(s) in CATALOG.md does it contribute to (if any)?
- Which revenue line(s) does it support (subscription, lead-gen, app-development, consulting, consumer-products, affiliate, flipping)?
- Bundle role: headliner | depth | infrastructure | standalone-app
- Is this a dig-the-gold candidate (operate as a service before selling as a product)?
- If SKU-shaped, what's the revisit trigger?
-->

## Marketing framing

*Tier 2 — fill when status reaches `evaluating`.*

<!--
When fillable, cover:
- Who's the audience (reference docs/marketing/strategy/AUDIENCES.md)?
- What's the conversion goal (click-through / trial / sign-up / OSS-adoption)?
- Which post types apply (dev-log / scenario-spotlight / oss-framework / use-case-tutorial)?
- What assets would be needed (screen recordings via BAS, screenshots, blog-length writeup)?
- Any contrarian failure modes specific to this idea worth flagging?
-->

## Capability multipliers

*Tier 2 — fill when status reaches `evaluating`.*

<!--
When fillable, cover:
- What does this unlock for other scenarios (downstream consumers)?
- What does it require (upstream prerequisites)?
- Is this a substrate (other scenarios depend on it) or a leaf (depends on substrates)?
- What's the recursive-learning-loop story — does this enable agents to do something they currently can't?
-->

## Goal alignment

*Tier 2 — fill when status reaches `evaluating`.*

<!--
When fillable, cover:
- How does this serve project goals? Reference docs/strategy/context.md, roadmap.md, business-solutions.md.
- Which long-term-capability-flag (if any) does this help retire?
- Which phase of the deployment vision (Phase 1 / 2 / 3 per CLAUDE.md) does this advance?
- Honest counter-argument: why might pursuing this be the wrong move?
-->

## Notes / open questions

*Append-only as evaluation deepens. Each entry: `- YYYY-MM-DD: <note>`.*

## Cross-references

<!--
Link relevant existing surfaces. Examples:
- `docs/monetization/catalogs/CATALOG.md#<sku>` — productization target
- `docs/monetization/catalogs/revenue-lines/<line>.md` — relevant revenue line
- `docs/marketing/strategy/AUDIENCES.md#<persona>` — primary audience
- `path:scenarios/<existing-scenario>/` — capability-multiplier upstream/downstream
- `monetization` team knowledge entry under `monetization/opportunity/<slug>` — paired opportunity-scout entry, if any (`prompt-manager team knowledge-list monetization --topic-prefix=monetization/opportunity/<slug>`)
-->
