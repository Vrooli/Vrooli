# Community Contribution Loop — Outbound Extensions (PRs & Scenarios)

## Strategic Rationale

Extends the proven V1 bug-reports pipeline to handle the higher-value but higher-risk artifact types: full PR-ready code fixes and user-built scenario proposals. Intentionally sequenced after V1 and inbound triage so we don't ship code-running artifacts before the review/verification infrastructure exists.

## Cross-Item Decisions

- **Reuse everything from V1.** Classifier, scrubber, submitter core, attribution, settings — all inherited. Only the artifact generation and channel logic is new.
- **PR artifacts need isolated verification.** Unlike bug reports, PRs contain code that someone has to run to evaluate. Links to contribution-verification-isolated initiative.
- **Scenario proposals may need a separate channel.** Likely not raw PRs to the main repo — possibly a scenario-proposals repo or a labeled discussion. Design decision in research/scenario-proposal-artifact-design.
- **Eligibility for scenario proposals is a separate question from eligibility for bug reports.** "Is this scenario general-purpose or personal?" is a new classifier problem layered on top of git-diff-vs-upstream.

## Deferred to Research

- PR artifact format (git format-patch vs branch-and-push, commit attribution, PR body template) — research/pr-fix-artifact-design.
- Scenario proposal packaging and channel — research/scenario-proposal-artifact-design.
- Whether PR submissions need inbound verification before landing (go/no-go for contribution-verification-isolated initiative).

## Sequencing Notes

Depends on both V1 outbound (reuses pipeline) AND inbound triage (needs a review path). Each artifact type has its own research → execute pair; the two pairs are independent of each other.

## Vision Context

Where the Community Contribution Loop graduates from "bug reporting service" to a real contribution pipeline — users can upstream actual code and scenarios they've built.
