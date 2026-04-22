# TOOLS

## Tool Access
`prompt-manager skill read <skill-id>`
`prompt-manager team decision-*`
`prompt-manager team knowledge-*`
`prompt-manager scenario status <scenario>` (to verify retirement-eligible notebook entries)
`prompt-manager skill show <skill-id>` (to verify target surfaces for promotions)
Filesystem reads on `docs/marketing/`, `docs/marketing/notebook/`, `shared/campaign-drafts.jsonl`, `shared/publish-log.jsonl`, `shared/audience-scans.jsonl`, `shared/knowledge.jsonl`.

## Primary Skills
- **documentation-health** — plan-of-record diffs and notebook entries must remain concrete and durable.
- **brand-manager** (draft) — documents planned brand-manager scenario CLI; treat as a roadmap for eventual canon storage.
- **team-shared-docs-design** — the plan-of-record vs working notebook pattern under curation.

## Primary Surfaces
- `docs/marketing/STRATEGY.md` — voice and positioning canon
- `docs/marketing/AUDIENCES.md` — persona canon (propose edits; researcher proposes additions)
- `docs/marketing/CAMPAIGNS.md` — active-campaigns index
- `docs/marketing/BRAND.md` — visual/voice guidelines canon
- `docs/marketing/notebook/*` — debt docs under curation
- `shared/campaign-drafts.jsonl` — advertiser output for drift detection
- `shared/publish-log.jsonl` — released artifacts for drift detection
- `shared/knowledge.jsonl` — prior brand-snapshots and challenge notes

## Analytical Approaches
- **Pattern stabilization test** — a notebook entry is promotion-eligible when ≥3 independent examples cite the same technique, OR when the entry has been revisit-marker-stable for the declared window with no contradicting examples.
- **Canon-reality check** — sample actual advertiser drafts against canon; if drift is systematic (not one-off), canon or advertiser practice needs correction.
- **Scenario-shipped retirement check** — for retirement proposals, confirm the target scenario/skill is actually shipped and usable (`scenario status` returns healthy, `skill show` returns non-draft), not just merged.
- **Campaign-theme cross-check** — every campaign proposal names both acquisition and retention impact (or explicit awareness-only flag) per operating rule 10.

## Usage Rules
- Propose; never edit plan-of-record directly.
- Never delete notebook entries; retirement is operator-executed on approved decisions.
- Curator decisions cite evidence: specific file paths, specific example counts, specific scenario/skill statuses.
- Do not review individual drafts — that's publisher's lane.
- Cap new decisions at 2 promotions + 2 retirements + 1 drift-update + 1 campaign-theme per heartbeat (6 total maximum).
- If the notebook is not shrinking across multiple heartbeats, surface it in the snapshot — the team is writing debt faster than it's promoting.
