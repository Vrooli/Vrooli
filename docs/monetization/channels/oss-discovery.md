# Channel: OSS Discovery

- **Status:** `active` (uninstrumented)
- **Audience:** humans (developers, technical evaluators, OSS contributors)
- **Owner:** ambiguous — no dedicated owner today. README/repo polish lives partially with marketing-crew (positioning), partially with director-swarm (project-identity canon), partially with scenario teams (per-scenario READMEs). The lack of a single owner is a known gap.
- **Feeds:** [`subscription`](../revenue-lines/subscription.md) — primarily through the "could self-host but prefer not to" conversion path described in [STRATEGY.md §1](../STRATEGY.md). Strong OSS discovery is what produces that audience in the first place.
- **Coupling:** Spans all tiers, with strongest pull on Tier 2 (self-hosted) since OSS-discovery audiences are most likely to self-host.

## Hypothesis

GitHub and adjacent OSS-discovery surfaces (trending, awesome-lists, README-as-marketing, contributor inbound, dependency-graph traffic) are a real and structural channel for Vrooli because the project is open source. This is happening *whether or not it's measured*. Acknowledging it as a channel is the first step toward making the right minimum investments without overinvesting.

The audience this channel reaches — developers who evaluate via reading code and docs — is also the audience most likely to convert to subscriptions later (per principle 1: most paying customers are people who *could* self-host but prefer not to). The funnel from "starred the repo" to "paid subscriber" is real but currently invisible.

## Why this is `active` (uninstrumented)

The channel produces signal today: GitHub stars accumulate, contributor PRs arrive, READMEs get traffic. None of it is measured against subscription conversion or any other downstream metric. "Active but uninstrumented" is the honest status — capturing the reality without overstating the channel's instrumented presence.

## Operational discipline

What "doing this channel well" looks like, even before instrumentation:

- **README polish.** The top-level README and per-scenario READMEs are public-facing. Treat them as marketing surfaces, not as engineer-internal docs. Cover what Vrooli is, why it exists, how to try it, and a clear path forward.
- **GitHub social proof.** Badges (build status, license, version), screenshots / demos in READMEs, a hero image or short demo GIF when meaningful. Cheap; high impact on first-impression conversion.
- **Contributing-friendly issue triage.** Open issues are a public surface. Stale unanswered issues damage credibility; quick acknowledgment + transparent triage builds trust.
- **Awesome-list submissions.** When a Vrooli scenario fits an awesome-list category cleanly, submit it. One-time effort with long tail discovery.
- **GitHub Discussions / community participation.** When relevant. Don't force; do participate when the activity is real.

## Anti-patterns

- **Starbuying / fake stars.** Permanent credibility damage; named explicitly in the channel cross-cutting "what's NOT a channel" list. Do not propose this even as a "test."
- **README-as-essay.** A README that buries the demo / install / first-action behind paragraphs of narrative is a conversion killer. Lead with the concrete; defer the philosophy to [`narrative/`](../../narrative/) docs and [`VISION.md`](../../../VISION.md).
- **Stale "coming soon" sections.** Promising future work and not delivering damages credibility worse than not promising it.
- **Inflated star counts via cross-promotion campaigns** that don't reflect genuine interest. The audience that finds Vrooli through OSS discovery is sophisticated about social-proof signals.
- **Dark-pattern badges or fake metrics.** Same prohibition as everywhere else.

## Telemetry (when instrumented)

What this channel needs once a measurement plan exists. None of this is wired up today.

- Stars × time (rate, not absolute count)
- Forks × time
- README → landing-page click-through rate (requires UTM tagging or referrer analytics)
- Contributor inbound (issues, PRs, discussions per period)
- Awesome-list referral traffic
- GitHub Trending appearances (and which keywords)
- Star-to-subscriber conversion (the eventual primary KPI; requires landing-page UTM tagging from README links + subscription-attribution)

The instrumentation gap is real. Filling it is a future-work item; documenting the channel without instrumentation is still useful because it captures what good looks like even before measurement.

## Cross-channel relationships

- **Web SEO** — strong reinforcement. README content and blog content overlap; cross-linking is healthy in both directions.
- **Community content** — strong reinforcement. HN, Reddit, dev forums route inbound traffic to GitHub; GitHub's own trending and starring mechanics surface scenarios for community discussion.
- **Skill registries** — partial overlap. A published skill is also a GitHub repo; the same scenario can produce signal on both channels. Disambiguate when measuring; don't double-count.
- **App stores** — partial conflict. App-store audiences typically don't care about open source; messaging in app-store listings shouldn't lean on OSS-discovery framings that resonate with developers but confuse general consumers.
- **In-product expansion** — orthogonal. Different funnel.

## Phase posture

- **Current:** `active` (uninstrumented). Discipline applies; measurement does not yet exist.
- **Instrumented:** when UTM tagging in READMEs + referrer analytics on landing pages exists, status updates to plain `active` with full instrumentation. No date set; this depends on landing-page-business-suite maturity and whoever takes ownership of the channel.
- **Sunset:** unlikely while Vrooli remains open source.

## Notes

- The ambiguity of ownership is a real risk. README polish, GitHub Discussions hygiene, awesome-list submissions, and similar "one-time-but-recurring" tasks tend to fall through the cracks when no one owns them. A future `channel-strategy-update` decision should propose an owner — likely inside marketing-crew or as a structural responsibility of director-swarm — before this channel scales further.
- This channel's existence is partly *why* the open-source positioning is strategic, not a revenue leak (per [STRATEGY.md §1](../STRATEGY.md)). Closing the source would close this channel; the audience it produces is the audience subscriptions later capture.
