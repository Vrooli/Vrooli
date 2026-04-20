# Meta-Orchestrator Summary

## Source

Follow-up from `execute/deployment-manager-lpbs-desktop-release-orchestration` completion. During post-completion review of the `landing-page-desktop-upload` skill rewrite, we compared capabilities the old multi-stage skill had vs. capabilities the new DM-driven orchestrator has, and identified cloud-deployment freshness drift as a real operational gap.

## Codebase Inspection Already Done

- `scenarios/scenario-to-cloud/api/domain/health.go:58-72` — S2C already exposes a `FreshnessStatus` type on its deployment health DTO with fields `Status`, `VersionStatus`, `FingerprintStatus` (values: `current` | `outdated` | `unknown`). The freshness block is optional (`*FreshnessStatus`) on the health response.
- `scenarios/deployment-manager/api/deployments/cloud_client.go` — DM's current `HTTPCloudHealthClient.CheckLPBSHealth` checks only `resp.StatusCode == 200` and returns `{Healthy: true}` without parsing the body. Freshness is discarded.
- `scenarios/deployment-manager/api/deployments/orchestrator_release.go:56-77` — `deployCheckCloudHealth` fails the step on `!result.Healthy`, otherwise advances. No freshness consideration.

## Decisions Made

- Item scope is: parse freshness from the S2C response, surface it on `CloudHealthResult`, and gate the orchestrator step on `fingerprint_status=outdated`.
- Persist the observation (where exactly — `releases.verification_evidence` vs. a new `cloud_health_evidence` field — is a workshop decision).

## Unresolved Questions Deferred To Workshop

- **Hard-fail vs. auto-converge**: Should `fingerprint_status=outdated` simply fail the release with a clear message, or should DM call `scenario-to-cloud redeploy --if-needed --force-bundle` once before retrying? The old `landing-page-desktop-upload` skill used an explicit one-shot convergence attempt. Hard-fail is simpler and keeps DM decoupled from S2C's mutation API; auto-converge is more operator-friendly but couples DM to a specific S2C command shape.
- **Where to persist evidence**: Reuse `releases.verification_evidence` (JSONB, currently carries verify-endpoint results) vs. new column. Reuse is cheaper; new column keeps concerns clean.
- **Handling `fingerprint_status=unknown`**: Warn-and-proceed, hard-fail, or operator-configurable per profile. Default should probably be warn-and-proceed so unknown doesn't block releases on S2C versions that don't populate the field.

## Dependency Notes

Depends on `execute/deployment-manager-lpbs-desktop-release-orchestration` (completed) for the orchestrator step and cloud client it extends.

## Greenfield Constraint

Additive to existing orchestrator. Do not add backwards-compat shims for a non-freshness-aware cloud-health path; once this ships, freshness parsing is the only path.
