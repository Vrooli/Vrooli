# Community Contribution Loop — Settings & Onboarding

## Strategic Rationale

This initiative exists to give the Community Contribution Loop a shipping surface. The outbound pipeline cannot ship without a user-facing opt-in (privacy) and a settings surface (attribution, thresholds, private-repo override). Priority 1 and co-equal with outbound V1 because it's a hard prerequisite for the V1 UX.

## Cross-Item Decisions

- **Anonymous by default, opt-in attribution.** Respect privacy first; make community participation a choice. Attribution config supports name/handle/contact so contributors can put themselves on the project if they want.
- **Simple yes/no at onboarding with "customize" hatch.** Don't front-load complexity. Most users should accept or decline in one click.
- **Private-fork auto-detect + off by default.** If the user's repo is private, assume they don't want to contribute back unless they explicitly enable it. They can still turn it on.
- **Per-category auto-approve thresholds.** One global toggle is too blunt — users should be able to auto-approve bug reports while always prompting for PRs.

## Sequencing Notes

Onboarding prompt ships first (execute/onboarding-contribution-opt-in) because it's small and blocks less. Full settings surface (execute/contribution-settings-surface) comes second and is what contribution-submitter-bug-report reads at runtime.

## Vision Context

Part of the larger **Community Contribution Loop** vision: extend Vrooli's self-healing properties beyond the original developer's machine. When agents on other users' installs fix project-level bugs, those fixes should flow back upstream — but only with user consent, scrubbed of PII/secrets, and filtered by an upstreamability classifier.
