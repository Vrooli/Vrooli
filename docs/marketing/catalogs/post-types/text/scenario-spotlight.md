# Post Type: Scenario Spotlight

**Status:** v1. New post type introduced at walk #5 (2026-04-28). Will mature as the `x-scenario-spotlight` skill runs in production and typed marketing-craft observations accumulate.

**Paired skill:** [`x-scenario-spotlight`](../../../../../scenarios/prompt-manager/store/skills/packs/core/x-scenario-spotlight/SKILL.md)
**Primary lane/member:** `producer` — subscription lane. Release execution is unowned until account operations gain a scheduler-side home.
**Craft observation topic:** `marketing-craft-observation/scenario-spotlight/<slug>`

## Purpose

A scenario spotlight pitches *one specific Vrooli scenario* as a useful tool / app / product to its target user. It is the conversion surface for "people who would benefit from this scenario."

It is **not**:
- A dev-log entry (those are project-wide progress narrative — see [`STRATEGY.md`'s dev-log narrative principles](../../../strategy/STRATEGY.md), pending extraction to `dev-log.md`).
- An OSS-framework post (those pitch Vrooli as a developer platform, not a single scenario as an end-user app — see `oss-framework.md` when authored).
- A campaign artifact (campaigns may *include* spotlights but are coordinated multi-asset launches owned by `brand-manager` — see [`CAMPAIGNS.md`](../../../strategy/CAMPAIGNS.md)).

## Audience

Primary: the **subscription buyer** persona (see [`AUDIENCES.md`](../../../strategy/AUDIENCES.md) for the full definition). Specifically:
- A person whose existing workflow is hindered by a problem the scenario solves.
- Often non-technical or partially-technical (the spotlight assumes the reader doesn't want to read source code).
- Has either (a) tried existing tools that didn't fit, or (b) hadn't realized the workflow could be automated.

Secondary, when the scenario is dev-tooling itself: developers who would use the scenario as a tool inside their own workflow. The spotlight then leans more technical but the conversion goal is still adoption, not contributor recruitment.

## Conversion goal

The desired action ladders, from softest to hardest:
1. **Click through** to the scenario's landing page or demo URL.
2. **Try it** (start a trial / use the free tier / install).
3. **Sign up / subscribe** at the appropriate tier (see [`docs/monetization/strategy/TIERS.md`](../../../../monetization/strategy/TIERS.md)).

Each spotlight should be explicit about which rung it is targeting. A spotlight aimed at click-through reads differently from one aimed at sign-up. Don't try to do all three in one post; the call-to-action gets blurry.

## Structure

Apply the cross-cutting [essay-shape technique](../../../methods/post-techniques/) — hook → introduction → body → conclusion — with this type-specific shape:

1. **Hook:** the friction or moment of recognition. "If you've ever ___ and ended up ___…" or "Here's what happens when ___…" Concrete. No abstract value-prop language.
2. **Intro:** name the scenario (apply [intro-on-first-mention](../../../methods/post-techniques/) — check `shared/published-scenario-mentions.jsonl` filtered by audience before assuming familiarity). One-sentence what-it-is.
3. **Body:** demonstrate, do not describe. The body is mostly the **asset** (screen recording / screenshots / mid-fidelity demo). Text wraps the asset with what the reader is seeing and why it matters for *their* workflow.
4. **Conclusion:** the call-to-action at the rung this spotlight is targeting. One link. One verb. Inter-post linkage to other relevant material if applicable (see [inter-post-linkage technique](../../../methods/post-techniques/)).

Length is **format-bounded**, not content-bounded:
- X thread: 4-7 tweets typical, structured around the asset.
- LinkedIn post: 200-400 words plus the asset.
- Blog: 600-1200 words; the asset can be a longer screen recording or multiple screenshots.

## Asset requirements

A scenario spotlight is **asset-led**. A spotlight without a working asset is a press release, not a spotlight. Required:

- **A screen recording or sequenced screenshots showing the scenario in actual use.** Not a mockup, not a wireframe. The recording shows the operator (or a credible stand-in) accomplishing the workflow the spotlight claims.
- **Brand consistency** with [`ASSETS.md`](../../../strategy/ASSETS.md) and [`IMAGE_STYLE.md`](../../../strategy/IMAGE_STYLE.md). Logos, OG image, color palette as specified.
- **Sanitization pass** equivalent to [`x-dev-log` guardrails](../../../strategy/CHANNELS.md): no paths, emails, credentials, or internal IDs in the recording.

**Asset production protocol:** the [Browser Automation Studio](../../../../../scenarios/browser-automation-studio/) (BAS) scenario is the canonical substrate for screen recordings (it supports recording with styled overlays and clean export). The `x-scenario-spotlight` skill will document the specific BAS commands; this file is the strategic canon directing skill authors to BAS as the substrate. Known constraint at time of authoring: BAS has a documented `recordVideo` gray-bar issue with a CDP workaround — see BAS scenario docs.

If the scenario being spotlighted is itself a UI scenario, the recording can be made in BAS pointed at the scenario's running UI. If it's a CLI scenario, an asciinema-style terminal recording is acceptable as long as it follows the same sanitization and brand rules.

## Voice

- **Product-led, not personal-builder.** The protagonist of a dev-log is the operator-and-agents *building Vrooli*. The protagonist of a scenario-spotlight is the **reader** using the scenario. Frame the post around what the reader is doing, not what we built.
- **Concrete on the friction.** "Local service businesses with bad websites" beats "small-and-medium-business owners struggling with their digital presence."
- **Avoid feature-list voice.** Demo > description. Show one thing the scenario does well; don't list everything.

## Conversion-rate-friendly variants (style sub-modes)

Different audiences respond to different stylistic framings. The skill supports producing variants; this list names the recognized ones:

- **Direct:** "Here's a scenario that does X. Here's the demo. Here's the link." Default.
- **Recommendation:** uses third-party voice ("someone built this thing that does X" / "here's a tool I came across") *only when there is a genuine third-party basis* — e.g., an external user wrote about the scenario, or the scenario was built by a contributor not the operator. Applying this technique without a real third-party basis is a contrarian failure mode (see below). See the cross-cutting [recommendation-framing technique](../../../methods/post-techniques/) once authored.
- **Problem-led:** opens with the friction in the audience's voice ("If you've ever spent an afternoon manually …"), reveals the scenario as the answer in the body. Higher conversion when the audience hasn't yet identified the friction as a solvable problem.
- **Comparison:** spotlights the scenario against a named alternative the audience already knows. Higher friction (invites contestation) but converts well when the differentiator is concrete.

The `x-scenario-spotlight` skill will support requesting a specific variant or letting the agent pick based on audience and conversion goal.

## Contrarian failure modes

The `marketing-contrarian` member skill ingests this section as type-level review rules. Each failure mode below is a checkable claim the contrarian validates a draft against.

| Failure mode | What it looks like | Why it backfires |
|---|---|---|
| **Capability inflation** | The post claims the scenario does X, but X is partial / WIP / only works in one configuration. | Worst single failure mode for spotlights. End-users who try the scenario after the post and don't get what was claimed feel deceived; refund risk; community-trust damage. Contrarian must verify each claim against scenario PRD / actual current state. |
| **Demo theater** | Screen recording shows an idealized happy path that hides retries / failures / setup steps the user will encounter. | Same as capability inflation but harder to catch because the recording is "real." Contrarian asks: would a fresh user replicate this without operator help? If no, the recording is theater. |
| **Pricing / tier confusion** | The post implies a feature is available at a tier the audience doesn't actually have. | Cross-reference [`docs/monetization/strategy/TIERS.md`](../../../../monetization/strategy/TIERS.md) and the scenario's `scenario-sku-map.json` entry. Contrarian must validate that every demo'd feature is reachable from the call-to-action's tier. |
| **Brand-asset drift** | Recording uses outdated logo / off-palette overlay / pre-canon style. | Cross-reference [`ASSETS.md`](../../../strategy/ASSETS.md) + [`IMAGE_STYLE.md`](../../../strategy/IMAGE_STYLE.md). Contrarian compares the asset against canon. |
| **Internal-vocabulary leakage** | Recording or copy contains internal IDs (`p8`, `round-002`, batch ids), agent names without intro, paths from the operator's machine, file structure not visible to end-users. | Cross-reference [no-internal-numbering-externally technique](../../../methods/post-techniques/). Contrarian must spot internal vocabulary in both copy and recording. |
| **Recommendation-framing without basis** | Post uses third-party "someone built this" voice but the operator authored the scenario; no genuine third party in the loop. | Reads slimy and erodes trust if discovered. Contrarian asks for the third-party basis before approving recommendation-style framing. |
| **Underclaim** | The post is so cautious it fails to communicate why the scenario is interesting. | Less catastrophic than overclaim but wastes the post. Contrarian flags this as a weak signal, not a hard reject. |
| **Conversion-rung blur** | The CTA mixes click-through, trial, and sign-up; the reader doesn't know what to do. | Contrarian asks: what is the *one* action this post is asking the reader to take? If the answer is more than one, reject. |

Each failure mode is `data_source=verifiable` (scenario PRD, monetization docs, brand assets, scenario state) — not judgment-only. The contrarian must cite the specific reference it checked.

Honesty flags the producer must attach to a spotlight draft (mirroring the dev-log pattern):

- `feature_claims=measured | overclaimed | uncertain` — measured = every claim cross-checked against scenario PRD and current state.
- `demo_authenticity=replicable | operator-only | mixed` — replicable = a fresh user from the audience persona could reproduce the demo without operator help.
- `tier_alignment=verified | not-yet-checked` — verified = every demo'd feature is reachable from the CTA's tier.
- `third_party_basis=self-authored | genuine-third-party | mixed` — required when the post uses recommendation framing.

## Cross-cutting techniques this type uses

(All have canonical homes under [`../../../methods/post-techniques/`](../../../methods/post-techniques/). The skill must require-read both this file and the technique files.)

- [Essay-shape per post](../../../methods/post-techniques/essay-shape.md)
- [Hook-vs-body length asymmetry](../../../methods/post-techniques/hook-vs-body-asymmetry.md)
- [Intro-on-first-mention](../../../methods/post-techniques/intro-on-first-mention.md) (with `shared/published-scenario-mentions.jsonl` lookup before assuming familiarity)
- [Inter-post linkage](../../../methods/post-techniques/inter-post-linkage.md) (link to dev-logs about the scenario; link to follow-up spotlights of the same scenario)
- [No internal numbering externally](../../../methods/post-techniques/no-internal-numbering-externally.md)
- Recommendation framing — *applies only when genuine third-party basis exists*; subject to the failure-mode rule above. Canonical home pending — see `../../../methods/post-techniques/README.md` for status.

## Where this fits in the marketing flow

```
                    ┌──────────────────────────────────────┐
                    │ producer drafts the spotlight via    │
                    │ the x-scenario-spotlight skill,      │
                    │ declaring claims                     │
                    └────────────────┬─────────────────────┘
                                     │
                     ┌───────────────▼──────────────────┐
                     │ marketing-contrarian reviews     │
                     │ against type-level failure modes │
                     │ (this file's checkable claims),  │
                     │ and hunts undeclared claims      │
                     └───────────────┬──────────────────┘
                                     │
                                     ▼
                          ┌────────────────────────┐
                          │ operator approves in   │
                          │ content-desk; the gate │
                          │ refuses unverified     │
                          │ claims and v0 types    │
                          └──────────┬─────────────┘
                                     │
                                     ▼
                            operator manually posts;
                            social-media-scheduler
                            (when shipped) handles the
                            URL paste-back / coverage
                            tracking / series chaining
```

`social-media-scheduler` is a planned scenario (initiative-proposal `dec-1777312920606447957`, accepted at walk #5) that owns the publishing plumbing once a spotlight is approved.

## Promotion path for craft observations

Patterns observed during production runs of `x-scenario-spotlight` should be written to `marketing-craft-observation/scenario-spotlight/<slug>`. Promotion targets per [`docs/agent-system/TEAM_DOCS_PATTERNS.md`](../../../../agent-system/TEAM_DOCS_PATTERNS.md):

- **Skill edit** — most observations land as edits to the `x-scenario-spotlight` skill (round structure, mining-strategy adjustments, output-contract refinements).
- **This file** — observations that change strategic canon (audience model, conversion goal phrasing, new variant, new failure mode).
- **A `post-techniques/` file** — observations that turn out to be cross-cutting and apply to multiple post types.

The typed observation never persists indefinitely; brand-manager curates promotions.

## Cross-references

- Paired skill: `x-scenario-spotlight` (in prompt-manager core skills).
- Plan-of-record neighbors: [`../../../strategy/STRATEGY.md`](../../../strategy/STRATEGY.md), [`../../../strategy/AUDIENCES.md`](../../../strategy/AUDIENCES.md), [`../../../strategy/CHANNELS.md`](../../../strategy/CHANNELS.md), [`../../../strategy/ASSETS.md`](../../../strategy/ASSETS.md), [`../../../strategy/IMAGE_STYLE.md`](../../../strategy/IMAGE_STYLE.md).
- Asset substrate: [Browser Automation Studio scenario](../../../../../scenarios/browser-automation-studio/).
- Publishing plumbing: `social-media-scheduler` scenario (planned; initiative-proposal `dec-1777312920606447957`).
- Tier alignment: [`docs/monetization/strategy/TIERS.md`](../../../../monetization/strategy/TIERS.md), [`docs/monetization/catalogs/CATALOG.md`](../../../../monetization/catalogs/CATALOG.md), `docs/monetization/catalogs/scenario-sku-map.json`.

## Changelog

- **2026-04-28** — Initial v1. Authored during walk #5 explicit divergence #2 alongside the post-types / post-techniques structure.
