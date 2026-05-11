# Community Contribution Loop — Inbound Triage Team

## Strategic Rationale

Makes the inbound side of the Community Contribution Loop self-improving, not just self-operating. A new prompt-manager team owns incoming submissions, and the real innovation is the **rejection → typed evidence → plan-of-record → customer update** learning loop: rejected submissions teach the system what to stop submitting, and that learning ships out as part of the next project update so customer-side classifiers/submitters get smarter automatically.

## Cross-Item Decisions

- **New team, not folded into QA.** The plan-of-record and typed evidence topics belong to this specific concern; mixing it into scenario-qa's bug intake muddies both.
- **Plan-of-record is the key integration point.** It lives in the repo, ships with project updates, and is read at runtime by the customer-side classifier/submitter. Versioning matters so customers can tell which ruleset they're running.
- **Typed evidence first, plan-of-record second.** Rejections update typed evidence immediately. A promotion agent periodically distills stable entries into plan-of-record updates.
- **Learning output can include code, not just docs.** Plan-of-record updates may trigger classifier/scrubber updates, not just guideline text.

## Deferred to Research

- Exact team roster (how many agents, responsibilities) — lives in research/contribution-triage-team-design.
- Cadence and promotion criteria for typed evidence → plan-of-record — lives in research/contribution-triage-team-design.

## Sequencing Notes

Depends on contribution-outbound-v1-bug-reports because the triage team needs real submissions to learn from. Research item for team design comes first, then the actual team materialization, then the plan-of-record + typed evidence infrastructure.

## Vision Context

Closes the Community Contribution Loop on the maintainer side. Without a triage team + learning loop, submissions pile up and guidelines stagnate; with it, the loop self-improves.
