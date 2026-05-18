# Vision & Purpose

> **Rewritten 2026-05-18** to match the new "execute skill on pristine golden, evaluate sandbox diff against expected-diff manifest" model. The prior framing (declarative structural expectations + CLI assertions) has been retired. See `../../PRD.md` Appendix → *Why the vision changed* for the full rationale.

## The Problem

The Vrooli ecosystem applies a large catalog of **steer skills** to scenarios during agent-driven development. Each skill exists to nudge a scenario along one dimension (API shape, storage, CLI surface, tests, UX, security, …).

Two compounding failure modes accumulate silently:

1. **Skills drift away from the template.** As `templates/scenarios/react-vite` matures, more of what a steer skill *says to do* becomes already true by default in newly generated scenarios. A skill that once meant "add a real change" might now mean "make no change at all" against a pristine golden — but nobody is checking.
2. **Skill behaviour is not regression-tested.** A skill edit can quietly break the agent's behaviour against a scenario that previously worked. There is no end-to-end test that runs each skill against a known-good baseline and compares the result to what was expected.

## The Solution

Execute every steer skill against template-pristine **golden scenarios**, in a sandboxed agent run, and compare the resulting filesystem diff against a per-skill **expected-diff manifest**. The verdict is a function of the diff vs the manifest — not of any declarative structural assertion.

```
golden  ──┐
          ├──► sandboxed agent run with skill active ──► diff ──► compare to manifest ──► verdict
skill   ──┘
```

Four verdicts:

- **pass** — the observed diff matches the manifest's allowed shape.
- **unexpected-mutation** — the run touched paths or contents the manifest does not allow. **Either the template is missing something the skill keeps adding, OR the skill has a bug.** Operator decides which.
- **run-failure** — the agent run failed (errored, timed out, exhausted budget). Distinct from a mutation mismatch.
- **stale** — the manifest is pinned to a template-version or skill-version that has moved since it was recorded; the manifest is treated as unreliable until refreshed.

### Why pristine goldens

A golden is a freshly generated scenario that conforms to the current `templates/scenarios/<template-id>` at a known version. Goldens are committed under `scenarios/reference-<template>/` so they can be inspected, versioned, and re-registered, but the contract is that they were created by the template and not subsequently modified by hand.

This makes the diff a meaningful signal:

- A no-op skill that produces no diff against a pristine golden is **doing its job already through the template**.
- A skill that always produces a large diff against a pristine golden tells us the template should be absorbing that change.
- A skill that produces an *unexpected* diff (vs. its manifest) tells us either the template gap shifted or the skill itself regressed.

### Why expected-diff manifests, not declarative structural assertions

A declarative assertion ("file X exists, content matches Y") asks: *does this scenario look right?* That question is too coarse — it does not catch a skill that *quietly stopped doing what it used to do*. The right question is: *did running this skill produce the file changes we expected it to produce?*

A manifest records exactly which paths a skill may touch and how (path patterns, optional content rules). Anything outside the manifest is a regression signal.

### Staleness

Each manifest is pinned to **both** the template version (because the golden was generated from that template) **and** the skill version it was recorded against:

- Template bump → all manifests against that golden become stale until refreshed.
- Skill bump → only that skill's manifest becomes stale.

Stale manifests are not deleted; they are flagged so the operator can confirm and re-record.

### Exemptions

Some skills are not meant to converge on a no-op against a pristine golden. They are framework primitives — `progress`, `bundle-integration-steer`, `progress-continuity-interruption-resilience` — that always mutate the scenario (advance a progress log, record continuity, etc.) regardless of how mature the template is. These are flagged as **always-mutator** in the catalog and their manifests are expected to keep allowing those mutations indefinitely. The remaining skills are tracked toward eventual no-op against a mature template.

## The Promotion-Retirement Vision

This scenario accelerates the migration of agent guidance from expensive prose to cheap, programmatic, template-encoded defaults:

```
Stage 1: Skill says "add X". Agent reads, interprets, edits files.
         → Expensive (LLM tokens), slow (reasoning loops).

Stage 2: Template starts including X by default in fresh scenarios.
         → DTV detects the skill is now producing a smaller diff.

Stage 3: Template fully covers X. Skill is documented as a no-op against
         the pristine golden — its manifest is empty.
         → DTV verdicts the skill as "absorbed by template".

Stage 4: Skill is retired (or kept only for legacy-fix workflows).
         The template carries the behaviour permanently.
```

DTV's job is to make every step of that progression *visible and verifiable*.

## Ecosystem Integration

```
prompt-manager (skill source)              templates/scenarios/* (golden source)
       │                                          │
       │  read skill manifests + versions         │  read template versions
       ▼                                          ▼
              development-toolchain-validator (this scenario)
                              │
                              │  execute (skill, golden) under sandbox via agent-manager
                              ▼
                       agent-manager (sandboxed run + diff capture)
                              │
                              │  diff + run summary (tokens, cost, duration)
                              ▼
                       compare to expected-diff manifest → verdict
                              │
              ┌───────────────┼────────────────┐
              ▼               ▼                ▼
        validation         CLI (dtv)         UI dashboard
        records DB                           (all-scenarios grid)
```

### Dependency Direction

- **DTV depends on**: prompt-manager (read skill catalog), agent-manager (execute sandboxed run + return diff), `templates/scenarios/*` (regenerate goldens), and the scenario CLIs being validated.
- **Nothing in the broader ecosystem depends on DTV.** It is a verification layer; prompt-manager, agent-manager, and ecosystem-manager all function without it.

## What Success Looks Like

1. Every template (starting with `react-vite`) has a committed golden under `scenarios/reference-<template>/`, registered with DTV and re-generable on demand.
2. Every applicable steer skill has an expected-diff manifest pinned to a specific template version and skill version, with explicit verdicts on every (skill, golden) tuple.
3. The all-scenarios dashboard shows verdicts at a glance: passes, mutations, run failures, stale flags.
4. When a skill or template is bumped, DTV surfaces the staleness immediately, so the operator can re-record manifests with intent rather than discovering drift weeks later.
5. The catalog of "absorbed by template" skills grows over time, measurable as a ratio: the closer it gets to 1.0, the more of agent guidance has been promoted into the template.
