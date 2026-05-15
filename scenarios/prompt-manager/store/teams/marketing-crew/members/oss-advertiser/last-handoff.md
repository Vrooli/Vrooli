### Source-system health
- Could not verify health for `git-control-tower`, `agent-manager`, `swarm-manager`, or `app-issue-tracker`.
- `vrooli scenario status <scenario>` fails for all four with: `read runtime registry schema version: unable to open database file (14)`.
- Workspace checkout is empty again; no repo files available for direct mining.

### Artifact requests reviewed
- No `artifact-request/oss/*` entries found.
- No `publish-log/*` entries found; latest coverage snapshots still say publish-log is empty.
- Latest accessible coverage remains 2026-04-28: `business` and `oss-platform` missing; `oss-platform` awaits first artifact.

### Story arcs considered
- No new shipped-work arcs mined because both checkout and live health path are blocked.
- Last viable queued arc remains 2026-04-28 sandboxing auto-approval p1-p6 for post #2, but only after post #1 resolves and evidence path is healthy.

### Drafts produced
- 0 drafts.
- No `campaign-drafts.jsonl` append: no draft, and file unavailable in empty checkout.

### Coverage or capability gaps
- Coverage-gap not raised: pending post #1 proposal `dec-1777318386116434321` still addresses missing `oss-platform` coverage.
- Capability-gap raised: `dec-1778787137208717804` for restoring reliable OSS advertiser evidence-path access.
- Attempted `report-bug` cross-team write to `scenario-qa`, but it failed with `team_mismatch`; recorded that in the ad-run entry.

### Supersessions
- None.
- `dec-1777318386116434321` remains relevant and pending; no newer draft or publish-log exists.

### Knowledge entry written
- `knw-1778787163539586440` on `oss-ad-run/2026-05-14`.

Next run should first check whether `dec-1777318386116434321` resolved, then whether `dec-1778787137208717804` resolved or the environment is healthy. If both post #1 is published and evidence access is restored, draft post #2 from sandboxing auto-approval with prior-post linkage.