# Historical backlog reconciliation

Measured 2026-09-06 from the preserved Plan Manager snapshot
`historical-backlog.json`. Historical status is not treated as implementation
evidence. The maturity plan owns the current disposition below.

| Historical item | Current disposition | Owning deliverable |
|---|---|---|
| `gct-github-api-design` | Superseded as a provider-design task; retain a neutral host seam and fake contract | DEL-GCT-14, DEL-GCT-17 |
| `gct-merge-architecture` | Remaining human-control design obligation | DEL-GCT-18 |
| `gct-trailer-system-design` | Implemented in the typed parser; owner resolution remains | DEL-GCT-06 |
| `agent-sandbox-auditability-contract` | Completed historical input; consume its lifecycle facts without recreating the contract | DEL-GCT-07 |
| `restore-git-control-tower-api-test-harness-compatibility` | Remaining readiness obligation; targeted compile passes, full admission is pending safe fixture conversion | DEL-GCT-01, DEL-GCT-19 |
| `gct-github-pr-api` | Remaining neutral collaboration consumer; no provider SDK in GCT | DEL-GCT-14, DEL-GCT-17 |
| `gct-trailer-support` | Remaining UI/editor integration; parser is now available | DEL-GCT-06, DEL-GCT-16 |
| `gct-pending-ai-provenance-hardening` | Remaining lifecycle and content-evidence work | DEL-GCT-07 |
| `gct-merge-api` | Remaining operator-only control obligation | DEL-GCT-18 |
| `qa-git-control-tower-code-quality-20260408` | Completed historical input; revalidate current changed packages only | DEL-GCT-19 |
| `ecosystem-loop-health-snapshot-contract` | Dropped by historical owner; not a GCT obligation | Explicitly excluded |
| `gct-conflict-resolution-ui` | Remaining workspace obligation | DEL-GCT-17, DEL-GCT-18 |
| `gct-swarm-manager-integration` | Remaining optional owner-backed enrichment; unavailable owner remains explicit | DEL-GCT-06, DEL-GCT-07 |
| `gct-committed-ai-provenance-history` | Remaining committed attribution obligation | DEL-GCT-07, DEL-GCT-08 |
| `build-git-control-tower-git-provenance-search` | Remaining shared-search consumer; do not build a private index | DEL-GCT-08 |
| `gct-review-summary-tests-missing-on-failed-check` | Regression is addressed by durable check preservation; independent failure fixture remains required | DEL-GCT-05 |
| `gct-github-pr-ui` | Remaining host-neutral collaboration UI | DEL-GCT-17 |
| `gct-security-review-tab` | Dropped by historical owner; not in this plan | Explicitly excluded |
| `agent-manager-default-sandboxing-rollout` | Completed historical input; activation evidence remains separate from fixture evidence | DEL-GCT-10, DEL-GCT-14 |
| `gct-merge-time-checks` | Remaining readiness/precondition obligation | DEL-GCT-18 |
| `readiness-git-control-tower` | Remaining scenario readiness obligation | DEL-GCT-19, DEL-GCT-20 |
| `gct-github-release-api` | Remaining neutral release consumer; provider lifecycle stays external | DEL-GCT-14, DEL-GCT-17 |
| `run-level-undo-and-revert-design` | Remaining human-only control design | DEL-GCT-18 |
| `qa-git-control-tower-tests-playbook-schema-20260515` | Dropped historical QA item; no current deliverable | Explicitly excluded |
| `gct-commit-level-gating` | Superseded by evidence-backed human control and advisory review contract | DEL-GCT-02, DEL-GCT-10 |
| `gct-release-authoring-ui` | Remaining draft-only authoring UI | DEL-GCT-17 |
| `run-level-undo-and-revert` | Remaining operator-facing control implementation | DEL-GCT-18 |
| `gct-commit-level-agent` | Dropped by human-only actuation contract; agents may advise only | DEL-GCT-02, DEL-GCT-10 |
| `gct-pr-description-generator` | Remaining grounded draft composer | DEL-GCT-09, DEL-GCT-17 |
| `gct-release-notes-generator` | Remaining grounded release composer | DEL-GCT-09, DEL-GCT-17 |

Related plans were reviewed from `related-plans.json`. Completed grouping,
mobile, optimistic-UI, contract-grouping, validation-truth, and Android/plugin
work are preserved inputs, not silently reimplemented. The active baseline
concurrency plan remains an external dependency and its evidence is not
claimed here. Draft plans remain historical context until their owning work is
implemented and independently evidenced.

## Evidence standing

No historical item is marked complete solely because its snapshot says
`completed`. Current evidence is attached to the maturity phase that consumes
it. Provider authentication, publication, and live host activation remain
unavailable without their owner and human receipts.
