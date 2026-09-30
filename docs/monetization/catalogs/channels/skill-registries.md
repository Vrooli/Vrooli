# Channel: Skill Registries

> Offer Desk is authoritative for this channel's current status, owner,
> activation trigger, and feed relationships. This document keeps the
> hypothesis, prerequisites, and safety judgment rather than a live channel
> snapshot.

- **Audience:** agents (LLM agents running in Claude Code, Codex CLI, Antigravity CLI, Cursor, Windsurf, Cline, JetBrains Junie, Sourcegraph Amp, etc.)
- **Owner:** monetization (strategy) + the scenario team that owns the wrapped capability (operations); no dedicated marketing surface.
- **Activation trigger:** *"Activate when at least one Vrooli capability is installable standalone (without the full runtime) AND a published Claude Skill for it is live in at least one curated registry (anthropics/skills, skills.sh, or equivalent) AND 60+ days of install/referrer telemetry is available."*
- **Feeds:** [`subscription`](../revenue-lines/subscription.md) — drives installs of the underlying scenario; downstream subscription conversions for users who want the convenience layer (managed gateway, hosted infra, etc.).
- **Coupling:** Loose. Couples mostly to Tier 2 (self-hosted) and Tier 3 (hosted) where standalone install of a single Vrooli capability is meaningful. Does not meaningfully couple to Tier 1 (mobile/desktop bundles) or Tier 4 (hardware appliance).
- **Legal surface:** Lighter than services lines but non-zero — registries impose ToS (Anthropic's, Vercel's, etc.), supply-chain attestations create disclosure obligations, and any user-data flow through skills triggers GDPR/CCPA review the same as any other Vrooli surface.

## Hypothesis

A meaningful share of agent-driven workflows in 2026+ will discover external capabilities through **skill registries** rather than through web search or app stores. If Vrooli publishes signed, scanner-clean skills for individual capabilities, agents running in Claude Code, Codex CLI, Antigravity CLI, Cursor, Windsurf, and adjacent runtimes will pull them in à la carte — driving installs of the underlying scenarios and, downstream, subscription conversions for users who want the convenience layer.

This channel is the agentic-era equivalent of SEO + app-store presence: a discovery surface that needs structural investment but that doesn't change the product itself.

Free agent usage is intentional, not leakage. A published skill lets external agents validate that a Vrooli capability has standalone value before the surrounding subscription surface is mature. The near-term return is proof: installs, task fit, failure reports, registry trust signals, and referrer traffic. The later return is conversion into the subscription convenience layer for users who want managed gateway access, hosted infrastructure, or a fuller bundle. Do not force monetization into the skill before the capability has earned usage; do not omit the eventual subscription path once usage is real.

## Activation criteria

Three things have to be true before this channel produces measurable
subscription lift:

1. **At least one Vrooli capability has to be standalone-installable.** Today, scenarios depend on the full Vrooli runtime + shared resources (postgres, redis, ollama). A skill that says "first install the entire Vrooli stack" defeats the point. The architectural prerequisite is per-capability install paths for the headline scenarios — Git Control Tower and Prompt Manager are the natural pilots.
2. **At least one signed, scanned skill has to be live in a curated registry.** Self-publishing to GitHub doesn't count; the discovery flow runs through registries that surface curated/verified skills, and trust scores depend on Cosign signatures + SLSA provenance + scanner clearance. See [`scenario-to-plugin/docs/guides/build-and-publishing.md`](../../../../scenarios/scenario-to-plugin/docs/guides/build-and-publishing.md) for the full pipeline.
3. **Telemetry has to exist.** Without 60+ days of install counts, referrer traffic, and install→subscription correlation, this channel is unfalsifiable. `financial-tracker` needs install-origin attribution before activation.

## Lifecycle interpretation

Offer Desk records whether the prerequisites and measurement window have been
met. A pilot remains a measurement exercise until install-to-subscription
correlation is evidenced; scaling or sunset decisions follow that evidence.

## Operational discipline

When the first skill is published and the channel enters the pilot phase:

1. **Validation hypothesis** — which Vrooli capability is this proving as a standalone unit?
2. **Fixed-duration pilot** — concrete window (60–90 days post-publish) before scaling.
3. **Productization target** — already documented (`subscription`); no new revenue line needed.
4. **Sunset or convert clause** — if the pilot shows no measurable install→subscription correlation by date X, sunset. If it does, scale to second/third capability.

Beyond that:

- **Security baseline is non-negotiable:** every publication must pass the ramp's [security posture](../../../../scenarios/scenario-to-plugin/docs/internal/SECURITY-POSTURE.md).
- **Workspace-sandbox by default:** the [build guide](../../../../scenarios/scenario-to-plugin/docs/guides/build-and-publishing.md) must configure the capability to run under workspace-sandbox where possible.
- **Per-skill telemetry separation.** Each skill's installs, referrer traffic, and downstream subscription conversion tracked separately. Aggregate channel revenue is meaningless without per-skill resolution.
- **Human-facing announcement is optional but expected when useful.** The channel itself is agent-audience, but a signed, curated skill is also a credible OSS/dev-log event. Marketing-crew may tell the builder-in-public story if the post stays factual, avoids paywall framing, and links back to the capability rather than inventing a human marketing campaign around the registry.

## Anti-patterns

Audience-specific failure modes — these matter most on agent channels because the anti-patterns are penalized by the channel mechanics, not just by humans noticing.

- **Sponsored placement is *negative* signal.** Agents discount "Sponsored" / "Promoted" labels in skill registries. Don't pay for placement; earn curation.
- **Broad `allowed-tools` lowers trust.** Skills requesting unrestricted shell, network, or filesystem access fail scanner heuristics and get downranked.
- **`curl ... | bash` to non-pinned URLs in install scripts** fails scanners and signals supply-chain insecurity. Always pin to a commit hash or signed release.
- **Hidden Unicode, prompt-injection patterns, exposed secrets.** Snyk's Feb 2026 audit found 13.4% of skills had at least one critical-level issue from this category. Vrooli skills must be zero-tolerance.
- **Skills that drag in the full Vrooli runtime by stealth.** A skill marketed as "GCT standalone" that secretly pulls postgres, redis, and ollama at install time is a trust violation — the channel is for discrete capabilities, not stealth-bundling.

## Telemetry (when active)

Reported alongside subscription metrics, tagged as channel-attributed:

- Skill install count by registry × time
- Referrer traffic from skill registries to Vrooli landing pages
- Install → free-self-host conversion rate (if measurable)
- Install → subscription conversion rate (the primary success metric)
- Scanner pass rate at publish time (Snyk Agent Scan, Cisco Skill Scanner)
- Registry curation tier (anthropics/skills curated > self-published)
- Time-to-deprecation per skill version (low number suggests skills are out of sync with scenario CLI changes — process failure)

## Cross-channel relationships

- **Web SEO + landing pages** — orthogonal channel, different audience (humans). Both rely on structured data, but discovery flows don't overlap. Reinforce indirectly: a published skill that drives blog/landing-page traffic helps web-seo, but the registries themselves are the discovery surface.
- **App stores (Tier 1)** — orthogonal channel, different audience. App stores serve human users; skill registries serve agent runtimes. No overlap.
- **OSS discovery** — adjacent. A skill in `literal:anthropics/skills` is also a GitHub repo with stars and discoverability of its own; a published skill produces incidental oss-discovery signal. Don't conflate the two when measuring.
- **In-product expansion** — internal to a Vrooli installation; this channel is *external*. Different problem, different surface, no overlap.

## Why this is one channel and not multiple

Each Vrooli capability could in principle have its own channel entry (one for the GCT skill, one for Prompt Manager, etc.). They share the same hypothesis, publish pipeline, scanner discipline, and productization target. Splitting produces management overhead without analytical value. Track per-skill metrics inside this channel; promote to multiple channels only if unit economics or operational disciplines diverge.

## Recommendation-blindness alignment

Both [`consumer-products`](../revenue-lines/consumer-products.md) and [`affiliate-commerce`](../revenue-lines/affiliate-commerce.md) carry the architectural rule that the agent producing a recommendation must not know what Vrooli sells. Skills published through this channel **do not violate that rule** — the published skill teaches an external agent (running in Claude Code, Cursor, etc.) how to use a Vrooli capability; it does not produce recommendations to a Vrooli end-user. The two surfaces are orthogonal.

If a published skill ever participates in producing recommendations *back into a lifestyle-bundle context* (e.g., a skill that calls into a Vrooli scenario which then recommends affiliate products), the recommendation-blindness rule applies in full and the skill must be reviewed against it. Default assumption: skills wrap business-bundle capabilities, where this risk is low. Lifestyle-bundle skills, if any, get extra review.

## Notes

- Operator's call to make this `active`. Capturing it as `candidate` early is cheap; activating it without the prerequisites in place would be a guardrail violation against the candidate-trigger principle ([STRATEGY.md §4](../../strategy/STRATEGY.md)).
- The publication scaffolding lives in the scenario-to-plugin ramp ([doctrine](../../../../scenarios/scenario-to-plugin/docs/concepts/SKILL-PUBLICATION.md), [template](../../../../scenarios/scenario-to-plugin/templates/scenario-skill/), [security posture](../../../../scenarios/scenario-to-plugin/docs/internal/SECURITY-POSTURE.md)) and proceeds independently of activation.
