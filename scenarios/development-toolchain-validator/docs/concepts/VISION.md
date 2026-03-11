# Vision & Purpose

## The Problem

The Vrooli ecosystem uses **steer skills** to guide AI agents during scenario development. There are 45+ steer skills, each focusing on one architectural dimension: API design, storage patterns, CLI structure, testing architecture, documentation, UX, security, and more.

These skills are managed by **prompt-manager** and applied during **ecosystem-manager** development loops. When an agent runs a loop iteration with a steer skill active, the skill's guidance focuses the agent on one specific dimension of scenario quality.

### The Cross-Steer Coherence Gap

While individual steer skills can be improved in isolation (via prompt-manager's meta optimization team), **there is no mechanism to detect cross-steer conflicts**. Consider:

- `api-steer` says "organize endpoints by bounded context/domain"
- `storage-steer` says "use repository pattern with these conventions" — but might imply a different module decomposition
- `cli-steer` says "thin wrapper over the API" — but its command grouping might not align with how `api-steer` organized the endpoints
- `interoperability-steer` defines contract rules that might conflict with how `api-steer` handles error shapes

Each skill converges the scenario toward its own ideal state, but those ideals might pull the scenario in **incompatible directions**. An agent running under `api-steer` makes structural decisions that a later `cli-steer` pass fights against, and the next `api-steer` pass fights back — **oscillation instead of convergence**.

### The Tooling Accuracy Gap

The ecosystem-manager's scenario-improver loop uses several tools in its Quick Validation Loop:

1. `vrooli scenario status` — lifecycle health
2. `scenario-completeness-scoring score` — quality scoring (0-100)
3. `scenario-auditor audit` — standards violations
4. `vrooli scenario test` (test-genie) — 11-phase test suite
5. `vrooli scenario ui-smoke` — UI validation

If any of these tools produce **false positives** (reporting issues that don't exist) or **false negatives** (missing real issues), the development loop makes wrong decisions. But there's no systematic way to detect these tooling bugs — they're discovered incidentally during agent iterations.

### The Skill Maturity Gap

Some steer skills are well-structured with clear convergence patterns, decision trees, and audit checklists. Others are more prose-heavy and vague. There's no way to measure this difference or systematically drive skills toward greater structure.

## The Solution

### Reference Scenarios as Ground Truth

A **reference scenario** is a full, known-good implementation that lives in `scenarios/` and uses all standard Vrooli tooling. It demonstrates what a fully-developed scenario looks like when all applicable steer skills are properly followed.

Reference scenarios serve as **two-directional validators**:

- **Inward (skill validation)**: If steer skills conflict when mapped to a reference, the skills are wrong — not the reference.
- **Outward (tool validation)**: If scenario-auditor reports violations on a known-good reference, the auditor rule is wrong. If test-genie fails a phase, the test logic is wrong. If completeness scoring gives a low score, the model is miscalibrated.

### Skill Connections with Declarative Expectations

Instead of *executing* steer skills (which would be expensive and could modify references), we store **declarative mappings** of what each skill expects:

- **Structural expectations**: Required/optional folders, files matching patterns, content snippets at specific locations
- **CLI tool assertions**: Read-only validation commands with expected JSON output (path + operator + value)

These expectations are the bridge between prose guidance (the skill's SKILL.md) and programmatic validation (this scenario's engine).

### Configuration as Maturity Metric

A skill with no structural config means we cannot programmatically describe what it does to a scenario. **The ability to define config IS the maturity metric.** Skills that are too vague or prose-heavy for programmatic validation are the lowest maturity — directly informing the meta optimization team about what to improve.

## The Promotion-Retirement Vision

This scenario is part of a larger system goal: **migrating AI-powered (expensive, slow) quality checks to programmatic (fast, deterministic) quality checks**.

```
Stage 1: Steer skills are markdown guidance
         Agent reads, interprets, does open-ended assessment
         → Expensive (LLM tokens), slow (reasoning loops)

Stage 2: Skill expectations configured as structural checks + CLI assertions
         Validation is programmatic and fast
         → This scenario enables this stage

Stage 3: Meta optimization team uses DTV CLI to identify poorly-configured skills
         Improves skills to be more structured
         Builds/enhances CLI tools that encode prose into checks
         → Autonomous improvement loop

Stage 4: Steer skill becomes a single CLI call
         Programmatically assesses and potentially fixes a scenario
         → Deterministic, fast, cheap
```

Each stage represents a reduction in the **search space** that agents must navigate. The easier it is to determine the current state of a scenario, the faster an agent can make progress.

## Ecosystem Integration

```
prompt-manager (skill source)
      │
      │ API: read skills, versions, content
      ▼
development-toolchain-validator (this scenario)
      │
      │ CLI: validate references against skill configs
      │ CLI: run tooling baselines (auditor, test-genie, completeness)
      │
      ├──► reference-react-vite (first reference)
      ├──► reference-cli-only (future)
      ├──► reference-landing-page (future)
      │
      ▼
prompt-manager meta optimization team
      │
      │ Uses DTV CLI output to prioritize skill improvements
      │ and tooling fixes
      ▼
ecosystem-manager
      │
      │ Uses improved skills and validated tools
      │ in scenario development loops
      ▼
All scenarios benefit from higher-quality steers and tools
```

### Dependency Direction

- **This scenario depends on**: prompt-manager API (read skills), scenario CLIs (run validations)
- **prompt-manager does NOT depend on this scenario**: The meta optimization team uses DTV's CLI as a consumer, configured in agent instructions — not in code.
- **ecosystem-manager does NOT depend on this scenario**: It benefits indirectly from improved skills and tools.

## What Success Looks Like

1. **Every steer skill** applicable to the react-vite template has a connection to `reference-react-vite` with at least structural expectations defined.
2. **All structural validations pass** — the reference satisfies every skill's expectations simultaneously (proving no cross-steer conflicts).
3. **All tooling baselines pass** — scenario-auditor, test-genie, and completeness scoring produce correct results on the reference.
4. **Skill maturity is visible** — a dashboard shows which skills have robust configs and which are too vague.
5. **The meta optimization team** routinely uses DTV output to drive skill improvements, steadily increasing the ratio of programmatic to AI-powered validation.
