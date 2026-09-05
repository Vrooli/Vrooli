# Marketing — Plan of Record

This folder is the **plan of record** for Vrooli's external voice: subscription marketing, open-source and community marketing, brand canon, audience frames, content formats, publishing discipline, and the learning loop that improves those systems over time.

It is maintained by the `marketing-crew` team and consumed by its members every heartbeat. The team's live operating rules are at [`scenarios/prompt-manager/store/teams/marketing-crew/shared/TEAM.md`](../../scenarios/prompt-manager/store/teams/marketing-crew/shared/TEAM.md); this folder is the strategic-canon side.

The local contract is [`manifest.json`](manifest.json), which instantiates the shared plan-of-record shape from [`docs/agent-system/team-plan-of-record.manifest.json`](../agent-system/team-plan-of-record.manifest.json).

## Start here for agents

Use this README first, then choose the module that matches the work:

| Question | Start with |
|---|---|
| How does the marketing team operate end to end? | [`operating/OPERATING_MODEL.md`](operating/OPERATING_MODEL.md) |
| What should Vrooli sound like? | [`strategy/STRATEGY.md`](strategy/STRATEGY.md) |
| Who is the audience? | [`strategy/AUDIENCES.md`](strategy/AUDIENCES.md) |
| Which campaign is active? | [`strategy/CAMPAIGNS.md`](strategy/CAMPAIGNS.md) |
| What are the publishing rules for a platform? | [`strategy/CHANNELS.md`](strategy/CHANNELS.md) |
| Which brand or visual rule applies? | [`strategy/BRAND.md`](strategy/BRAND.md), [`strategy/ASSETS.md`](strategy/ASSETS.md), or [`strategy/IMAGE_STYLE.md`](strategy/IMAGE_STYLE.md) |
| Which post shape should be used? | [`catalogs/post-types/README.md`](catalogs/post-types/README.md) |
| Which reusable writing technique applies? | [`methods/post-techniques/README.md`](methods/post-techniques/README.md) |
| Which cross-cutting marketing strategy applies? | [`strategy/patterns/README.md`](strategy/patterns/README.md) |
| How should external signals become research evidence? | [`evidence/research/README.md`](evidence/research/README.md) |
| Which generated-media asset or schema applies? | [`catalogs/rich-media/README.md`](catalogs/rich-media/README.md) |
| How is marketing research classified and routed? | [`taxonomies/marketing-research/README.md`](taxonomies/marketing-research/README.md) |

## Folder map

| Folder | Purpose |
|---|---|
| [`operating/`](operating/README.md) | Team operating contract and validation commands. |
| [`taxonomies/marketing-research/`](taxonomies/marketing-research/README.md) | Human-readable taxonomy plus machine-readable `taxonomy.json` for marketing research signals. |
| [`methods/post-techniques/`](methods/post-techniques/README.md) | One file per reusable cross-cutting voice or structure technique. |
| [`catalogs/post-types/`](catalogs/post-types/README.md) | One file per kind of marketing post Vrooli produces, organized by medium. |
| [`catalogs/rich-media/`](catalogs/rich-media/README.md) | Structured-data substrate for generated imagery and video: character, scene, product, asset, and prompt-template registries. |
| [`strategy/`](strategy/README.md) | Voice, audience, campaign, channel, brand, asset, image-style, and strategy-pattern canon. |
| [`evidence/research/`](evidence/research/README.md) | Research intake, collection, method, promotion, and evidence rules. |
| [`governance/`](governance/editing.md) | Editing authority, adoption validation, and changelog. |

## Editing rules

- **Agents never write to plan-of-record canon directly.** All canon edits come through operator-approved decisions.
- **Working observations go to typed team knowledge topics.** Use the most specific topic prefix available (`research-inbox/*`, `marketing-craft-observation/*`, `friction-inbox/*`, `bug-inbox/*`, or `capability-work/*`) and promote only through an owned decision.
- **Use the most specific module.** Add post shapes under `catalogs/post-types/`, reusable writing techniques under `methods/post-techniques/`, directional marketing truth under `strategy/`, classification/routing rules under `taxonomies/`, and supporting proof under `evidence/`.
- **Operator executes accepted edits.** Commit messages cite the decision id.

Decision-context detail lives in [`governance/editing.md`](governance/editing.md).

## Cross-references

- [`docs/agent-system/team-plan-of-record.manifest.json`](../agent-system/team-plan-of-record.manifest.json) — shared machine-readable PoR contract and extension rules.
- [`docs/agent-system/TEAM_DOCS_PATTERNS.md`](../agent-system/TEAM_DOCS_PATTERNS.md) — typed knowledge flow and plan-of-record write boundary.
- [`docs/narrative/`](../narrative/README.md) — project-identity canon. Marketing owns the linguistic how; narrative owns the what.
- [`docs/monetization/`](../monetization/README.md) — positioning, pricing, catalog, and bundle ground truth. Marketing reads; monetization owns.
- [`VISION.md`](../../VISION.md) — operator-authored manifesto and long-term north star.

## Future PoR work

- Add typed marketing-craft observation schemas for recurring production lessons that are not research evidence, bugs, friction, or capability gaps.
- Add PoR manifest validation once prompt-manager consumes `manifest.json`.
- Split large strategy documents only when one-entity-per-file structure would make future edits safer.
- Move generated-media schemas toward dedicated scenario support when the future `rich-media-studio` or `brand-manager` scenario owns that capability.
