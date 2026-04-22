# Marketing — Plan of Record

This folder is the **plan-of-record** for Vrooli's external voice: subscription marketing, open-source / community marketing, brand canon, and audience frames. It's maintained by the `marketing-crew` team and consumed by its members every heartbeat.

There is a companion working notebook under [`notebook/`](notebook/README.md). The two are deliberately different:

- **Plan-of-record (this folder).** Canon. Operator-curated via approved decisions. Agents propose diffs; they never edit directly. Entries grow and stabilize over time.
- **Working notebook ([`notebook/`](notebook/)).** Debt. Any member may append freely. `brand-manager` proposes promotions and retirements. Entries shrink over time as they're crystallized into permanent structure.

See [`scenarios/prompt-manager/store/skills/packs/core/team-shared-docs-design/SKILL.md`](../../scenarios/prompt-manager/store/skills/packs/core/team-shared-docs-design/SKILL.md) for the pattern definition. See [`scenarios/prompt-manager/store/teams/marketing-crew/shared/TEAM.md`](../../scenarios/prompt-manager/store/teams/marketing-crew/shared/TEAM.md) for the team's live operating rules.

## Files in this folder

| File | Purpose |
|------|---------|
| [`STRATEGY.md`](STRATEGY.md) | Voice, positioning principles, dual-audience framing (subscription + OSS), anti-patterns. |
| [`AUDIENCES.md`](AUDIENCES.md) | Personas. Subscription buyer, OSS contributor. Researcher proposes updates via `audience-update` decisions. |
| [`CAMPAIGNS.md`](CAMPAIGNS.md) | Index of active campaigns. Brand-manager proposes launches via `campaign-launch-proposal` decisions. |
| [`CHANNELS.md`](CHANNELS.md) | Per-platform publishing rules. Publisher proposes drift corrections via `channel-update` decisions. |
| [`BRAND.md`](BRAND.md) | Visual and voice guidelines. Thin until the `brand-manager` scenario ships; then points at it. |

## Write rules

- **Agents never write to these files directly.** All edits come through operator-approved decisions.
- **Edit context:** `brand-guideline-update` for STRATEGY / BRAND; `audience-update` for AUDIENCES; `campaign-launch-proposal` for CAMPAIGNS; `channel-update` for CHANNELS.
- **Operator executes edits** on decision acceptance. Commit messages cite the decision id.

## Cross-references to monetization

The team reads `docs/monetization/STRATEGY.md`, `CATALOG.md`, `PRICING.md`, `TIERS.md`, per-bundle files, and `scenario-sku-map.json` as positioning ground truth. It never writes to those files.
