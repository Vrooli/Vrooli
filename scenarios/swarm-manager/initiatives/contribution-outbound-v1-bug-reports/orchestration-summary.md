# Community Contribution Loop — Outbound V1 (Bug Reports)

## Strategic Rationale

The core MVP of the Community Contribution Loop. Bug-reports-only because they're the lowest-risk artifact type: no code is executed on your side, scrub surface is bounded (text + logs + diffs), and the signal volume should be highest. Proving the outbound pipeline on bug reports first lets us validate the classifier, scrubber, and submitter before extending to PRs and scenario proposals.

## Cross-Item Decisions

- **Classifier is mechanical, not semantic.** Primary signal: `git diff` against the upstream main commit the user installed from. Files untouched vs. upstream = upstreamable surface; files diverged = fork-local. This collapses a semantic problem ("is this fix project-level or personal?") into a mechanical one (did the user modify this file?).
- **Scrubber fails closed.** If a chunk can't be confidently classified as safe, submission blocks with a diff for user review. User protection > throughput.
- **One pipeline, differentiated by artifact type.** Bug-report is the first artifact; PR-fix and scenario-proposal are later extensions that reuse classifier, scrubber, submitter, attribution, and settings.
- **Agents emit signals, not submissions.** The two natural emission sources (scenario-qa team handoffs, prompt-manager agent handoff docs) produce structured candidate-bug-report signals. The pipeline consumes them; agents stay ignorant of submission mechanics.
- **V1 default channel is issue, not PR.** PR artifacts come in contribution-outbound-extensions. V1 is pure issue filing.

## Deferred to Research

- Exact channel choice within "issue vs discussion" — lives in research/contribution-submitter-architecture.
- Full classifier edge-case table (detached installs, stale upstream refs, partial-overlap fixes) — lives in research/upstream-divergence-classifier.

## Sequencing Notes

Research items (classifier, submitter architecture) come first and can run in parallel. Scrubber depends on submitter architecture to know its integration points. Bug-report submitter depends on both research items plus scrubber plus settings surface. Agent emission is last — the pipeline exists before agents emit into it.

## Vision Context

Core MVP of the **Community Contribution Loop** — when agents on other users' installs fix project-level bugs, those reports flow back upstream with user consent, after scrubbing, and gated by upstreamability.
