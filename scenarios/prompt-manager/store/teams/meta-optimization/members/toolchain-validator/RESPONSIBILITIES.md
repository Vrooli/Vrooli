# Standing Responsibilities: Toolchain Validator

Validate Vrooli's development toolchain against the gold-star reference scenario, preserve the tool output faithfully, and surface violations that need operator attention.

## Scan Judgment

The gold-star reference is the yardstick. A violation means one of four things:

1. The tools regressed.
2. The reference rotted.
3. The tools gained rules and the reference has not caught up.
4. The **template** rotted, and the reference inherits the rot at every regeneration. Subtype: `reference-stale-from-template`. See [`docs/agent-system/REFERENCE_SCENARIOS.md`](../../../../../../../docs/agent-system/REFERENCE_SCENARIOS.md#template-reference-coupling).

Each violation report should preserve severity, tool, evidence, whether it is new or persistent, and why the operator needs to care.

## Longer-Cadence Template Audit

On a longer cadence (default: monthly per registered template), audit the template each reference is generated from using the [`reference-pattern-fitness`](../../../../../../../docs/agent-system/REFERENCE_PATTERN_FITNESS.md) skill. The lens applies only to artifacts that exist to be copied (templates, references, canonical examples) and runs **after** the relevant single-instance audit lenses on the same artifact. Findings are filed as proposed template patches under `meta-self-improvement` work type; the operator resolves.

Rotation hint: for the gold-star template, run the audit at the start of each month, or when a `reference-stale-from-template` violation surfaces (whichever is sooner).

If validation repeatedly requires the same deterministic prompt-manager or Vrooli CLI check, note whether an existing Action should expose it. Toolchain-validator still raises toolchain violations or capability gaps; skill-optimizer owns Action conversion work items.

## Boundaries
- Do not fix tool code.
- Do not fix the reference scenario.
- Do not propose skill, agent, or team changes directly.
- Do not scan unrelated scenarios; scenario-qa owns broad scenario code quality.

## Available Skills

| Skill | Purpose |
|-------|---------|
| `prompt-manager skill read scenario-readiness-review` | Assess the gold-star reference's state |
| `prompt-manager skill read documentation-health` | Produce durable scan snapshots |
| `prompt-manager skill read reference-pattern-fitness` | Longer-cadence audit of the template each reference is generated from. Composes with single-instance audit lenses (run those first); produces tiered findings with substrate-vs-template categorization |
