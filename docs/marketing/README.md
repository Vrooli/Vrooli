# Marketing — Plan of Record

This folder is the **plan-of-record** for Vrooli's external voice: subscription marketing, open-source / community marketing, brand canon, and audience frames. It's maintained by the `marketing-crew` team and consumed by its members every heartbeat.

There is a companion working notebook under [`notebook/`](notebook/README.md). The two are deliberately different:

- **Plan-of-record (this folder).** Canon. Operator-curated via approved decisions. Agents propose diffs; they never edit directly. Entries grow and stabilize over time.
- **Working notebook ([`notebook/`](notebook/)).** Debt. Any member may append freely. `brand-manager` proposes promotions and retirements. Entries shrink over time as they're crystallized into permanent structure.

See [`scenarios/prompt-manager/store/skills/packs/core/team-shared-docs-design/SKILL.md`](../../scenarios/prompt-manager/store/skills/packs/core/team-shared-docs-design/SKILL.md) for the pattern definition. See [`scenarios/prompt-manager/store/teams/marketing-crew/shared/TEAM.md`](../../scenarios/prompt-manager/store/teams/marketing-crew/shared/TEAM.md) for the team's live operating rules.

## Files in this folder

| File | Purpose |
|------|---------|
| [`STRATEGY.md`](STRATEGY.md) | Voice canon: positioning principles, dual-audience framing (subscription + OSS), voice samples, anti-patterns, dev-log narrative principles. |
| [`AUDIENCES.md`](AUDIENCES.md) | Personas. Subscription buyer, OSS contributor. Researcher proposes updates via `audience-update` decisions. |
| [`CAMPAIGNS.md`](CAMPAIGNS.md) | Index of active campaigns. Brand-manager proposes launches via `campaign-launch-proposal` decisions. |
| [`CHANNELS.md`](CHANNELS.md) | Per-platform publishing rules. Publisher proposes drift corrections via `channel-update` decisions. |
| [`BRAND.md`](BRAND.md) | Visual identity navigation hub. Points at ASSETS, IMAGE_STYLE, STRATEGY for the actual content. |
| [`ASSETS.md`](ASSETS.md) | Canonical brand asset registry (logos, favicons, OG image, font, usage rules). Eventually subsumed by the `brand-manager` scenario when it ships. |
| [`IMAGE_STYLE.md`](IMAGE_STYLE.md) | AI image generation style guide. Palette (dark blue / deep purple / neon green), aesthetic (abstract, futuristic, neon), prompt directives. Eventually subsumed by the `brand-manager` scenario. |

## Write rules

- **Agents never write to these files directly.** All edits come through operator-approved decisions.
- **Edit context:** `brand-guideline-update` covers STRATEGY / BRAND / ASSETS / IMAGE_STYLE *and* the cross-team narrative canon at `docs/narrative/*`. Use `audience-update` for AUDIENCES; `campaign-launch-proposal` for CAMPAIGNS; `channel-update` for CHANNELS.
- **Operator executes edits** on decision acceptance. Commit messages cite the decision id.

## Cross-references

- `docs/narrative/` — project-identity canon (pitch, story, FAQ, press kit, pitch-deck outline). Cross-team consumers (advertisers, monetization, director, LPBS) pull narrative content from there. Voice canon (this folder's STRATEGY.md) is the linguistic *how*; narrative is the *what*.
- `docs/monetization/` — STRATEGY, CATALOG, PRICING, TIERS, per-bundle files, scenario-sku-map.json. Positioning ground truth. The team reads; never writes.
- `VISION.md` (root) — operator-authored manifesto. Long-term north-star; pulled into deepest narrative layers.
- `docs/concepts/ARCHITECTURE.md` — canonical technical reference for "how Vrooli actually works."
