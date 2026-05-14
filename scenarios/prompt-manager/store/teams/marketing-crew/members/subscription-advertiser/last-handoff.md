### Coverage triage summary
- Missing deployed subscription SKUs: 0.
- Stale deployed subscription SKUs: 0.
- Imminent subscription SKUs with committed launch window: 0.
- `business` remains pre-launch under rule 13: no accessible evidence of shipped deployable component or committed launch window.
- `oss-platform` remains out-of-lane for subscription advertiser.

### Artifact requests reviewed
- No accessible `artifact-request/subscription/*` entries.
- No accessible publish-log entries; latest publisher snapshots still say `publish-log.jsonl` is empty.
- Research evidence remains future positioning fuel, especially BYOK partial-boundary and subscription-as-convenience framing, but does not override launch/deployed-SKU gates.

### Drafts produced
- None. Reason: zero deployed subscription SKUs, zero active subscription campaigns, zero committed launch windows, and no open subscription artifact request.

### Coverage-gap decisions raised
- None. No fresher coverage gap than the existing documented rule-13 gate.

### Capability-gap decisions raised
- None. No drafting attempted, so no marketing-specific tooling blocker surfaced.

### Supersessions
- None. Owned-context pending decisions for subscription advertiser: 0. Team queue has 1 pending OSS publish proposal, below ceiling.

### Knowledge entry written
- `knw-1778697100202142252` on `subscription-ad-run/2026-05-13`.

### Friction / next check
- This sandbox had no repo checkout files, so direct reads of docs and JSONL state were unavailable. I used `prompt-manager` decision/knowledge storage instead. Also, `prompt-manager team knowledge-add --by=...` is obsolete; attribution is automatic now. Treat both as one-off unless repeated next run.