# Go To Market — Data Backup Manager

This document records launch strategy, positioning, channels, and
validation experiments for the scenario.

## Purpose Of This Document

Use this document to answer:

- Who should hear about this scenario?
- Which channels can reach them?
- What claim or offer will be tested?
- What evidence changes the product or monetization plan?

## Audience And Positioning

This is primarily a **platform-internal capability**, not a standalone
consumer SaaS. It de-risks every other scenario by making runtime state
safe to keep out of git and giving the platform a real recovery story.
Its market-facing role is as a **reliability feature of self-hosted and
enterprise Vrooli deployments**, where encrypted, offsite, verified
backups and disaster recovery are a paid expectation.

- Audience: the Vrooli platform itself (every scenario with mutable
  runtime state) first; then operators of self-hosted/enterprise Vrooli
  installs who need DR.
- Positioning: "dependable, engine-backed, verified-restore backup for
  Vrooli runtime state" — table-stakes infrastructure, not a product
  pitch.
- Main claim: backups that are proven to restore (verified-restore
  gate), encrypted by default, and recoverable even if Vrooli is down.
- Proof needed: a passing verified-restore on a real target — first
  proof point is prompt-manager's `store/teams/**` (PRD OT-P1-005).

## Channels

| Channel | Hypothesis | Assets Needed | Validation Signal |
|---|---|---|---|
| Internal platform adoption | Other scenarios will register targets once registration + verified restore exist. | Self-registration API/CLI, verified-restore proof | prompt-manager registers `store/teams/**` and stops committing it. |
| Self-hosted / enterprise deployment tier | DR + encrypted/offsite backup is a deciding feature for self-hosted buyers. | Deployment-hub positioning, install/restore docs | Self-host evaluators cite backup/DR as a reason to adopt the paid tier. |
| Managed-backup / DR tier (future) | Operators will pay to have backup/DR managed for them. | Hosted offsite destinations, retention/compliance story, cost model | Demand for managed retention/offsite from self-host users. |

## Launch Motion

1. Land the companion `kopia` resource and the core
   Source/Destination/Plan/Run/Restore model.
2. Prove verified restore on a real target — register prompt-manager's
   `store/teams/**` and stop committing it to git (PRD OT-P1-005).
3. Validate PRD operational targets and scenario tests.
4. Surface backup/DR as a reliability feature of the self-hosted /
   enterprise deployment tiers (see the deployment hub).
5. Evaluate a managed-backup / DR tier once self-host demand for
   offsite/retention is evident — not before.

## Messaging

| Message | Audience | Evidence | Status |
|---|---|---|---|
| "Stop committing runtime state to git — it's backed up and proven recoverable." | Internal scenario owners | First verified restore on prompt-manager store | intended |
| "Encrypted, offsite, verified backups — restore even if the platform is down." | Self-hosted/enterprise operators | Standalone kopia restore + verify gate | intended |
| "Disaster recovery and compliance-friendly retention, managed for you." | Managed-tier buyers (future) | Managed offsite + retention policies | deferred |

## Validation Experiments

| Experiment | Channel | Threshold | Decision |
|---|---|---|---|
| Migrate prompt-manager store off git | Internal adoption | Verified restore passes; store no longer committed | Greenlight broader internal registration. |
| Cite backup/DR in self-host eval | Deployment tier | Evaluators name DR as an adoption driver | Invest in enterprise DR packaging. |

## Cross-References

- [`MONETIZATION.md`](MONETIZATION.md) — packaging and pricing hypothesis
- [`../../PRD.md`](../../PRD.md) — product outcomes
- [`../operations/OBSERVABILITY.md`](../operations/OBSERVABILITY.md) — validation signals and telemetry
