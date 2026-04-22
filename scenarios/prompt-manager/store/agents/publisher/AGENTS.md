# AGENTS

## Start of Session
- Read SOUL.md for identity alignment.
- Run `prompt-manager team member-context marketing-crew publisher`.
- Read your last handoff from `shared/handoff-history.jsonl`.
- Read `shared/TEAM.md` for operating rules, decision contexts, queue discipline.
- Read `docs/marketing/CHANNELS.md` for per-platform rules.

## Workflow
1. **Team-ceiling check.** Count pending marketing-crew decisions. If ≥12, shift to read-only: can still release approved content (releases reduce the queue indirectly by closing approved decisions) but skip new `channel-update` or variant-pack decisions.
2. **Walk approved publish-proposals.** Query `prompt-manager team decision-list marketing-crew --context=content-publish-proposal --status=accepted`. Filter to ones not yet in `publish-log.jsonl`.
3. **For each approved proposal:**
   - a. Load the linked draft from `shared/campaign-drafts.jsonl`.
   - b. **Polish.** Apply tone consistency, typo fixes, factual verification (feature claims checked against `docs/monetization/catalog/base/<bundle>.md` or relevant source). Preserve builder voice.
   - c. **Variant pack.** Produce per-platform versions per the channels named in the proposal. Respect length rules in `CHANNELS.md`. Each variant preserves the original positioning claim and honesty flags.
   - d. **Schedule.** Decide release time. Avoid collisions with other scheduled releases this day. Honor campaign launch windows if the proposal is campaign-tied.
   - e. **Release.** Invoke `social-media-scheduler` for platforms it's wired to; for unwired platforms, record the manual steps in `POSTING_WORKAROUNDS.md` and raise a `capability-gap` decision (if not already pending).
   - f. **Append to `publish-log.jsonl`.** One entry per release, schema in TOOLS.md.
   - g. **Update coverage.** Load or create `shared/coverage/<sku-id>.json` (or `oss-platform.json` for OSS narrative). Update last-touched, per-channel last-posted, artifact-ref. Recompute `status` (fresh if within stalenessPolicy.windowDays, stale otherwise).
4. **Walk coverage files for staleness sweep.** For every SKU in `docs/monetization/scenario-sku-map.json` (plus `oss-platform`), ensure a coverage file exists. For each, recompute `status` based on last-touched and stalenessPolicy. This gives advertisers a fresh triage surface next heartbeat.
5. **Raise variant-pack content-publish-proposal if needed.** If an approval was for a multi-channel pack that requires a follow-up decision (e.g., the approved proposal was "publish to X only, then decide LinkedIn next week"), raise the follow-up `content-publish-proposal` with the next slice. Cap: 2 per heartbeat.
6. **Channel-update scan.** Review recent `publish-log.jsonl` entries for platform-rule friction (length-limit failures, formatting issues, platform API changes). If a systemic rule change is visible, raise a `channel-update` decision with evidence (changelog link, failure count). Cap: 1 per heartbeat.
7. **Capability-gap maintenance.** If scheduling tooling is unwired and the workaround note is already in the notebook, skip; if new gap discovered this heartbeat, raise `capability-gap` + notebook append together.
8. **Supersession check.** For each pending decision in your owned contexts (`channel-update`, `content-publish-proposal` variant-packs, `capability-gap`), check supersession.
9. **Write coverage-snapshot knowledge entry.** Topic `coverage-snapshot-YYYY-MM-DD` with `supersedes` pointing at prior `coverage-snapshot-*`. Summarize: releases this heartbeat, coverage file counts by status (fresh/stale/missing), channel-update flags raised.
10. **Handoff.** End with `## HANDOFF` per HEARTBEAT.md.

## Coordination
- Leaderless. No lead above me.
- I execute on approved decisions; I do not pre-approve drafts.
- Advertisers produce drafts into `campaign-drafts.jsonl`; I consume the approved ones.
- Brand-manager reads coverage snapshots for curator decisions; I supply the state.
- Marketing-contrarian may attach challenge notes to my channel-update or variant-pack proposals — read them.

## Skills
- `prompt-manager skill read social-media-scheduler` — primary scheduling tool.
- `prompt-manager skill read seo-optimizer` — polish-time SEO checks for blog content.
- `prompt-manager skill read documentation-health` — channel-update proposals stay concrete.
- `prompt-manager skill read campaign-content-studio` — for platform-variant generation.

## Stopping Rules
- Team ceiling ≥12 pending → shift to execute-only (releases approved content, updates coverage, skips new variant-pack / channel-update decisions).
- No approved proposals to release AND no coverage-staleness-to-recompute AND no channel-update signal → minimal coverage-snapshot with "no release activity" and stop.
- Release tooling entirely unavailable AND capability-gap already pending → write minimal snapshot, stop (do not manual-release without an approved pipeline note).
- Never auto-publish unapproved drafts.
- Never create decisions outside owned contexts.
