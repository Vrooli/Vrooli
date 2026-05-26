# Monetization — Data Backup Manager

This document records how the scenario could create revenue or support
a monetizable Vrooli capability. Keep it honest: `not-applicable` is
better than inventing a commercial story.

## Purpose Of This Document

Use this document to answer:

- Is this scenario a direct product, internal capability, SKU component,
  add-on, or service accelerator?
- Who would pay for it, and why?
- What packaging or pricing hypothesis exists?
- What validation signal would justify more investment?

## Role In Vrooli

Monetization here is **indirect and tiered**. This is not a standalone
consumer SaaS; it is table-stakes infrastructure that de-risks every
other scenario and underpins paid deployment tiers.

- Direct product: no — not sold on its own.
- Internal capability: yes, primary role — makes runtime state safe to
  keep out of git and gives the platform a real recovery story.
- SKU/bundle candidate: yes, as a reliability/DR feature of the
  self-hosted and enterprise deployment tiers; enables a future
  managed-backup / DR tier.
- Revenue line: indirect today (adoption/retention of paid deployment
  tiers); a managed-backup / DR offering is the direct line later.

## Customer / Buyer

- Primary user: the Vrooli platform and its scenario owners; then
  operators of self-hosted Vrooli installs.
- Buyer: organizations running self-hosted/enterprise Vrooli who expect
  disaster recovery, encrypted/offsite backups, and retention controls.
- Pain: losing runtime state with no recovery story; being forced to
  commit mutable state to git; compliance/retention requirements that
  bare-disk storage cannot meet.
- Existing alternatives: ad-hoc scripts, committing state to git, or
  raw kopia/restic with no platform integration or verified-restore
  gate.

## Packaging

| Packaging Option | Status | Notes |
|---|---|---|
| Standalone app | not-applicable | Not a standalone consumer SaaS; value is in platform integration. |
| Bundle component | intended | Reliability/DR feature of the self-hosted and enterprise deployment tiers. |
| Add-on | candidate | A managed-backup / DR tier (hosted offsite + retention) is the natural paid add-on. |
| Service/consulting assist | candidate | Compliance-friendly retention and DR setup can support done-for-you delivery. |

## Pricing Hypothesis

- Model: indirect (drives adoption/retention of paid deployment tiers)
  now; subscription for a managed-backup / DR tier later (e.g., priced
  on retained storage / offsite destinations / retention class).
- Comparable products: managed backup/DR add-ons in self-hosted
  platforms; backup-as-a-service offerings — captured at monetization
  review.
- Willingness-to-pay evidence: none captured yet; expected to come from
  self-host evaluators citing DR as an adoption driver.
- Cost drivers: `kopia` + `vault` resources (local), plus offsite/hosted
  destination storage and egress for the managed/enterprise path.

## Validation Plan

- Demand signal needed: self-hosted/enterprise evaluators name backup/DR
  as a reason to adopt the paid tier; internal scenarios adopt
  self-registration.
- Channel: see [`GO-TO-MARKET.md`](GO-TO-MARKET.md).
- Success threshold: prompt-manager migrates `store/teams/**` off git
  behind a verified restore; DR cited in at least one self-host eval.
- Revisit trigger: verified restore proven in production and first
  managed/enterprise demand for offsite/retention.

## Current Status

`design-locked, indirect monetization` — the role (internal capability +
reliability feature of paid deployment tiers, with a future
managed-backup / DR tier) is decided. Not yet implemented; no revenue
line active. Ship internally first (prove on prompt-manager), then
surface as a deployment-tier reliability feature.

## Cross-References

- [`../START-HERE.md`](../START-HERE.md) — orientation workflow
- [`../../PRD.md`](../../PRD.md) — product requirements
- [`GO-TO-MARKET.md`](GO-TO-MARKET.md) — channel and launch plan
- [`../operations/OBSERVABILITY.md`](../operations/OBSERVABILITY.md) — telemetry needed for business validation
- [`../../../../docs/monetization/README.md`](../../../../docs/monetization/README.md) — project-level monetization strategy
