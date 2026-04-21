# Programmatic Conversion Playbook

Converting a prose-heavy skill into a thin wrapper over a scenario CLI is meta-optimization's highest-leverage lever. This doc captures what we've learned about doing it well.

**Posture:** This is a living notebook. Entries get added as conversions happen. Patterns stabilize into rules; rules get promoted into a skill or scenario feature; entries get retired. The `debt-curator` runs that promotion loop.

**Revisit markers:** Review the "patterns" section after every 5 conversions. Review the "anti-patterns" section after every 3 rejected proposals.

---

## When to attempt conversion

A prose skill is a conversion candidate if all three are true:

1. **There's a scenario CLI that covers the behavior.** Or could, with a small addition. If the scenario is partial, file a `capability-gap` instead of attempting a partial conversion.
2. **The behavior is deterministic.** Same input → same output. If the skill is guiding judgment ("be thoughtful about X"), leave it as prose.
3. **The skill is meaningfully prose-heavy.** If the main prose section is under ~300 tokens, the conversion isn't worth the validation cost.

If any is false, route the skill through normal `skill-improvement` instead.

---

## The conversion procedure

*Revisit this section after 5 conversions. Refine based on what actually worked.*

1. **Baseline the skill.** Count tokens in the main prose section. Record usage count from `prompt-manager graph popular --type skill`. Note drift age.
2. **Identify the scenario CLI target.** Find the specific `vrooli <scenario> ...` command that covers the behavior.
3. **Draft the wrapper.** The new skill should be short — typically under 300 tokens. It should: state the CLI command, describe inputs/outputs, list edge cases the CLI handles, and defer to the scenario's own help/docs for anything deeper.
4. **Validate with an agent.** Before proposing, run the wrapper through an agent session (or a few) that would have used the prose version. Check that the agent does the same thing via the CLI that it would have done via the prose.
5. **Measure the delta.** New token count vs. baseline. Usage continuity over next N heartbeats.
6. **File `skill-conversion-candidate` decision** — include the baseline, the draft wrapper, and the validation result.

---

## Patterns (distilled)

*Promotion target: once a pattern here appears in ≥3 entries in the log below, debt-curator proposes a skill or playbook-skill that encodes it.*

_(empty — will fill in as conversions accumulate)_

---

## Anti-patterns (distilled)

*Promotion target: once an anti-pattern here appears in ≥2 rejected proposals, debt-curator proposes tightening the criteria in `skill-optimizer`'s HEARTBEAT.md.*

_(empty — will fill in as rejected proposals accumulate)_

---

## Conversion log

Append a row here for every attempted conversion. Format:

```
### [YYYY-MM-DD] <skill-id> → <scenario-cli>

- **Baseline tokens:** N
- **Post-conversion tokens:** M
- **Delta:** -X%
- **Usage at conversion time:** N references/heartbeat
- **Usage 4 heartbeats post-conversion:** N references/heartbeat (continuity check)
- **Outcome:** accepted | rejected | rolled back
- **Lessons:** one or two sentences on what this conversion taught us
```

_(empty — first entry lands when the first conversion ships)_

---

## Open questions

Things we haven't figured out yet. Each entry should eventually become a pattern, an anti-pattern, or a `capability-gap`.

- How do we handle skills that are partly deterministic and partly judgment? (Split into two? Keep as prose with a scenario-call sub-section?)
- What's the right validation-agent count for a conversion? (One might be enough; three is expensive.)
- How do we detect a conversion that silently degraded downstream agents? (Watch for retry-rate increase on any agent that referenced the converted skill?)
