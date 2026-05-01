# Action Conversion Playbook

Converting repeated deterministic prose into a Vrooli-controlled CLI plus an Action contract is meta-optimization's highest-leverage lever. This doc captures what we learn while doing that work.

**Posture:** This is a living notebook. Entries get added as conversions happen. Patterns stabilize into rules; rules get promoted into skills, Action contracts, CLI backlog, or scenario features; entries get retired. The `debt-curator` runs that promotion loop.

**Revisit markers:** Review the patterns section after every 5 conversions. Review the anti-patterns section after every 3 rejected proposals.

## When to Attempt Conversion

A prose skill section or notebook entry is an Action conversion candidate if all three are true:

1. **A Vrooli-controlled CLI command covers the behavior.** This may be a project CLI, resource CLI, prompt-manager CLI, or scenario CLI. If no controlled command exists, file `cli-backlog` or `capability-gap` instead of creating a partial Action.
2. **The behavior is deterministic.** Same input should produce the same operation and a clear success/failure state. If the work is judgment, synthesis, or taste, leave it in a Skill or Plan of Record.
3. **Discoverable execution would reduce future cost.** The current prose causes meaningful token load, repeated manual command lookup, or repeated run friction.

If any is false, route through normal skill improvement, notebook retirement, or backlog instead.

## Conversion Procedure

1. **Baseline the prose.** Count the relevant token/prose section, usage count, and current manual steps.
2. **Identify the CLI owner.** Name the exact Vrooli-controlled command. If the command does not exist or needs branching logic, route that work to the owning CLI before creating an Action.
3. **Inspect existing Actions.** Run `prompt-manager discover "<operation>" --type all` and `prompt-manager action show <id>` for any candidate. Improve an existing Action before proposing a new one.
4. **Draft or update the Action.** The Action wraps exactly one CLI command, declares inputs/outputs, permissions, examples, validation, and `runEligible`.
5. **Collapse or retire prose.** Keep judgment and safety boundaries in Skills. Replace deterministic command prose with an Action reference once the Action validates.
6. **Validate.** Use `prompt-manager action validate <id>` and, when appropriate, `prompt-manager action run <id> --dry-run`.
7. **Measure the delta.** Compare prose/token cost, repeated manual operation count, discovery hits, and Action run history after adoption.
8. **File the decision.** Use `action-candidate`, `action-improvement`, or `action-deprecation` with baseline, expected delta, validation evidence, and measurement plan.

## Promotion Classifier

```text
If it says what is true -> Plan of Record.
If it says how to decide -> Skill.
If it says what to run -> Action.
If it says how it works -> CLI implementation.
If it says what is missing -> Backlog/capability-gap.
If it is unverified or one-off -> Notebook.
```

## Patterns

*Promotion target: once a pattern here appears in at least 3 entries in the log below, debt-curator proposes a skill, Action-authoring rule, or CLI backlog rule that encodes it.*

_(empty - will fill in as conversions accumulate)_

## Anti-Patterns

*Promotion target: once an anti-pattern here appears in at least 2 rejected proposals, debt-curator proposes tightening `skill-optimizer` or `meta-contrarian` criteria.*

- Creating an Action before a controlled CLI exists.
- Encoding branching, shell conditionals, or multi-command workflows in the Action contract.
- Proposing Action conversion without a baseline and post-adoption measurement.
- Treating every CLI-adjacent skill as convertible when the remaining value is judgment.

## Conversion Log

Append a row here for every attempted Action conversion. Format:

```markdown
### [YYYY-MM-DD] <source> -> <action-id>

- **Source:** skill | notebook | run lesson | direct seed
- **CLI target:** `<command>`
- **Baseline:** token count / manual steps / run friction
- **Action status:** draft | active | archived
- **Validation:** command and result
- **Post-conversion measurement:** discovery/run/prose delta after N heartbeats
- **Outcome:** accepted | rejected | rolled back | still measuring
- **Lessons:** one or two sentences on what this conversion taught us
```

### 2026-05-01 seed review -> action:scenario.status.show

- **Source:** direct seed
- **CLI target:** `vrooli scenario status {{scenario}}`
- **Baseline:** scenario status lookup existed as command knowledge, not an Action-discoverable operation.
- **Action status:** active
- **Validation:** `prompt-manager action validate scenario.status.show --json` and dry-run pass.
- **Post-conversion measurement:** pending; graph still reports low inbound references.
- **Outcome:** still measuring
- **Lessons:** Active Actions still need prompt/skill/doc references and discovery tests, or agents continue to miss them.

### 2026-05-01 seed review -> action:team.decisions.list

- **Source:** direct seed
- **CLI target:** `prompt-manager team decision-list {{team}} --json`
- **Baseline:** agents manually recall decision-list syntax or rely on prose.
- **Action status:** active
- **Validation:** `prompt-manager action validate team.decisions.list --json` and dry-run pass after declaring `apiRead`.
- **Post-conversion measurement:** pending.
- **Outcome:** still measuring
- **Lessons:** Prompt-manager read commands need explicit `apiRead` permissions before they can become runnable Actions.

## Open Questions

- Should draft Action validation report contract validity separately from run eligibility?
- What Action usage threshold proves a prose section can be safely collapsed?
- How should mixed discovery balance methodology Skills and executable Actions without crowding either out?
