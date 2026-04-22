# Community Contribution Loop — Isolated Verification

## Strategic Rationale

Lets the inbound triage team safely run incoming PR patches and scenario proposals on an isolated install before approving them. Builds on the existing `vrooli-bridge` scenario — no new infrastructure, just integration. Only becomes relevant once code-running artifacts (PRs, scenarios) are in scope, so it's the last initiative in the sequence.

## Cross-Item Decisions

- **vrooli-bridge is the verification runtime.** Don't build a second isolated-install mechanism — reuse what we have.
- **Isolated install is reset between verification runs.** No cross-contamination between candidate contributions.
- **Verified-fail is a strong rejection signal.** Feeds into the contribution-inbound-triage team's working-notebook with high weight, making the learning loop tighter.
- **Verification is not auto-approval.** A passing verification is a necessary-but-not-sufficient signal; triage team still makes the final call.

## Deferred to Research

- Exact test suite to run against candidates (project tests? scenario-specific tests? targeted regression?) — research/vrooli-bridge-verification-integration.
- Result schema and how it feeds back to triage disposition — research/vrooli-bridge-verification-integration.

## Sequencing Notes

Depends on contribution-outbound-extensions. Not needed for V1 bug-reports (no code to run). Research + execute pair, straightforward.

## Vision Context

The infrastructure piece that lets the Community Contribution Loop handle untrusted code safely — a prerequisite for the loop to ever include PRs or scenarios, not just issue-style bug reports.
