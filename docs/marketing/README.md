# Marketing — Plan of Record

This folder is the **plan-of-record** for Vrooli's external voice: subscription marketing, open-source / community marketing, brand canon, and audience frames. It's maintained by the `marketing-crew` team and consumed by its members every heartbeat.

There is a companion working notebook under [`notebook/`](notebook/README.md). The two are deliberately different:

- **Plan-of-record (this folder).** Canon. Operator-curated via approved decisions. Agents propose diffs; they never edit directly. Entries grow and stabilize over time.
- **Working notebook ([`notebook/`](notebook/)).** Debt. Any member may append freely. `brand-manager` proposes promotions and retirements. Entries shrink over time as they're crystallized into permanent structure.

See [`docs/agent-system/TEAM_DOCS_PATTERNS.md`](../agent-system/TEAM_DOCS_PATTERNS.md) for the pattern definition. See [`scenarios/prompt-manager/store/teams/marketing-crew/shared/TEAM.md`](../../scenarios/prompt-manager/store/teams/marketing-crew/shared/TEAM.md) for the team's live operating rules.

## Start here for agents

Use this README first, then choose the file or sub-hub that matches the work:

| Question | Start with |
|---|---|
| How does the marketing team operate end to end? | [`OPERATING_MODEL.md`](OPERATING_MODEL.md) |
| What should Vrooli sound like? | [`STRATEGY.md`](STRATEGY.md) |
| Who is the audience? | [`AUDIENCES.md`](AUDIENCES.md) |
| Which campaign is active? | [`CAMPAIGNS.md`](CAMPAIGNS.md) |
| What are the publishing rules for a platform? | [`CHANNELS.md`](CHANNELS.md) |
| Which brand or visual rule applies? | [`BRAND.md`](BRAND.md), [`ASSETS.md`](ASSETS.md), or [`IMAGE_STYLE.md`](IMAGE_STYLE.md) |
| Which post shape should be used? | [`post-types/README.md`](post-types/README.md) |
| Which reusable writing technique applies? | [`post-techniques/README.md`](post-techniques/README.md) |
| Which cross-cutting campaign strategy applies? | [`strategies/README.md`](strategies/README.md) |
| How should external signals become research evidence? | [`research/README.md`](research/README.md) |
| Which generated-media asset or schema applies? | [`rich-media/README.md`](rich-media/README.md) |
| Is this unresolved learning rather than canon? | [`notebook/README.md`](notebook/README.md) |

## Files in this folder

| File | Purpose |
|------|---------|
| [`OPERATING_MODEL.md`](OPERATING_MODEL.md) | Target-state operating model: loops, roles, topic surfaces, decision handoffs, notebook drainage, current implementation gaps, and adoption sequence. |
| [`STRATEGY.md`](STRATEGY.md) | Voice canon: positioning principles, dual-audience framing (subscription + OSS), voice samples, anti-patterns, dev-log narrative principles. |
| [`AUDIENCES.md`](AUDIENCES.md) | Personas. Subscription buyer, OSS contributor. Researcher proposes updates via `audience-update` decisions. |
| [`CAMPAIGNS.md`](CAMPAIGNS.md) | Index of active campaigns. Brand-manager proposes launches via `campaign-launch-proposal` decisions. |
| [`CHANNELS.md`](CHANNELS.md) | Per-platform publishing rules. Publisher proposes drift corrections via `channel-update` decisions. |
| [`BRAND.md`](BRAND.md) | Visual identity navigation hub. Points at ASSETS, IMAGE_STYLE, STRATEGY for the actual content. |
| [`ASSETS.md`](ASSETS.md) | Canonical brand asset registry (logos, favicons, OG image, font, usage rules). Eventually subsumed by the `brand-manager` scenario when it ships. |
| [`IMAGE_STYLE.md`](IMAGE_STYLE.md) | AI image generation style guide. Palette (dark blue / deep purple / neon green), aesthetic (abstract, futuristic, neon), prompt directives. Eventually subsumed by the `brand-manager` scenario. |

## Sub-folders

The first file in each sub-folder is a hub. Start there rather than guessing at individual spokes.

| Folder | Purpose |
|--------|---------|
| [`post-types/README.md`](post-types/README.md) | One file per kind of marketing post Vrooli produces, organized by primary medium under `text/`, `image/`, and `video/`. Each file is the strategic canon at the operator-decision level (purpose, audience, conversion goal, asset requirements, contrarian failure modes). Each pairs with an `x-<type>` skill that is the executable spec. |
| [`post-techniques/README.md`](post-techniques/README.md) | One file per cross-cutting voice/structure technique (essay-shape, hook-vs-body asymmetry, intro-on-first-mention, recommendation-framing, etc.). Techniques are referenced from multiple post-type files rather than duplicated. |
| [`strategies/README.md`](strategies/README.md) | Cross-cutting marketing strategies that span multiple post-types and aren't pure techniques (AI-UGC personas, hook library, funnel patterns). One file per strategy. |
| [`research/README.md`](research/README.md) | Research intake, collection, method, and promotion architecture for operator-fed alpha, proactive scans, future bookmark-intelligence-hub exports, and research method evolution. |
| [`rich-media/README.md`](rich-media/README.md) | Structured-data substrate for AI image and video generation: character / scene / product schemas (JSON), prompt templates (Veo/Seedance-compatible), and ground-truth assets. Drives multi-frame and multi-shot consistency. |
| [`notebook/`](notebook/) | Working notebook for patterns observed in production. Append-anyone; brand-manager curates promotions to plan-of-record (here, in `post-types/`, in `post-techniques/`, in `strategies/`) or to permanent structure (a skill or scenario). |

## Write rules

- **Agents never write to these files directly.** All edits come through operator-approved decisions.
- **Edit context:** `brand-guideline-update` covers STRATEGY / BRAND / ASSETS / IMAGE_STYLE / `path:docs/marketing/strategies/*` / `path:docs/marketing/rich-media/*` / `path:docs/marketing/research/*` *and* the cross-team narrative canon at `path:docs/narrative/*`. Use `audience-update` for AUDIENCES; `campaign-launch-proposal` for CAMPAIGNS; `channel-update` for per-platform-rule edits to CHANNELS; `channel-strategy-update` (researcher) for channel-priority/strategy edits to CHANNELS; `post-type-proposal` (researcher) for new entries under `post-types/`; `hook-candidate-promotion` (researcher) for additions to `strategies/hook-library.md`.
- **Operator executes edits** on decision acceptance. Commit messages cite the decision id.

## Cross-references

- `path:docs/narrative/` — project-identity canon (pitch, story, FAQ, press kit, pitch-deck outline). Cross-team consumers (advertisers, monetization, director, LPBS) pull narrative content from there. Voice canon (this folder's STRATEGY.md) is the linguistic *how*; narrative is the *what*.
- `path:docs/monetization/` — STRATEGY, CATALOG, PRICING, TIERS, per-bundle files, scenario-sku-map.json. Positioning ground truth. The team reads; never writes.
- `VISION.md` (root) — operator-authored manifesto. Long-term north-star; pulled into deepest narrative layers.
- `docs/concepts/ARCHITECTURE.md` — canonical technical reference for "how Vrooli actually works."
