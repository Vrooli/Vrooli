# Glossary

## Core Terms

### Reference Scenario
A full, known-good scenario implementation that lives in `scenarios/` and uses all standard Vrooli tooling. It demonstrates what a fully-developed scenario looks like when all applicable steer skills are properly followed. References are NOT deployed as products — they exist solely for validation. Example: `reference-react-vite`.

### Steer Skill
A prompt-manager skill (mode: "Steer") that provides focused architectural guidance for one dimension of scenario development. Examples: `api-steer`, `storage-steer`, `cli-steer`. Steer skills are markdown text files consumed by AI agents during ecosystem-manager development loops.

### Skill Connection
A mapping between a steer skill and a reference scenario, stored with the skill's version number and content hash at connection time. A connection without expectations is "unconfigured" — it represents a skill that needs structured validation defined.

### Structural Expectation
A declarative check that a reference scenario's filesystem matches a pattern. Types:
- **Folder**: A directory path that should exist (e.g., `api/handlers/projects/`)
- **File**: A glob pattern matching expected files (e.g., `api/handlers/*_test.go`)
- **Snippet**: Expected content at a specific location in a file

### CLI Tool Assertion
A declarative check that a read-only CLI command produces expected JSON output. Consists of a command, a JSONPath expression, a comparison operator, and an expected value.

### Skill Drift
When a connected skill's content has changed in prompt-manager since the connection was established. Detected by comparing the stored version/content hash against the current version. Drift means the connection's expectations may be stale.

### Overlap
When structural expectations from two or more different skill connections target the same files or folders in a reference scenario. Overlaps are not necessarily conflicts — they may indicate that multiple skills legitimately care about the same area.

### Conflict
When structural expectations from different skill connections are mutually exclusive — they cannot both be satisfied simultaneously. Example: Skill A requires folder `api/routes/` while Skill B requires folder `api/handlers/` for the same purpose.

### Skill Maturity Score
A calculated score reflecting how well a skill's behavior can be described programmatically. Skills with structural expectations + CLI assertions that all pass = high maturity. Skills with no config = lowest maturity.

### Coverage Map
A file/folder-level visualization showing which areas of a reference scenario are covered by skill expectations and which areas have no coverage from any connected skill.

## Ecosystem Terms

### prompt-manager
The scenario that manages all skills (steer, search, tools, practice, meta). DTV reads skills from prompt-manager's API. The meta optimization team within prompt-manager uses DTV's CLI to check their work.

### ecosystem-manager
The scenario that orchestrates scenario development through task queues, auto-steer profiles, and agent loops. It uses the scenario-improver.md prompt which invokes development tools in its Quick Validation Loop.

### scenario-improver.md
The prompt template used by ecosystem-manager for scenario improvement iterations. It instructs agents to run `vrooli scenario status`, `scenario-completeness-scoring score`, `scenario-auditor audit`, `vrooli scenario test`, and `vrooli scenario ui-smoke` as validation steps.

### Auto Steer Profile
An ecosystem-manager configuration that defines multi-phase development cycles. Each phase references specific skill IDs from prompt-manager, with metric-driven stop conditions. DTV validates that the skills referenced in profiles are coherent.

### Meta Optimization Team
A prompt-manager team of AI agents that autonomously improve skills, agents, and teams. DTV provides the feedback loop: the team uses DTV's CLI to detect skill issues, then improves skills and tools based on findings.

### Promotion-Retirement Lifecycle
The skill-principles pattern for evolving skills from prose guidance to CLI tool contracts:
1. **Interim prose guardrail**: Skill has detailed markdown guidance
2. **Promote to CLI contract**: Tool implements deterministic checks
3. **Retire superseded prose**: Remove skill instructions now covered by tools

DTV accelerates this lifecycle by making it visible which skills are still purely prose (no config = not yet promotable) vs. which have programmatic expectations defined.

## Assertion Operators

| Operator | Meaning | Value Type |
|----------|---------|------------|
| `eq` | Equals | any |
| `neq` | Not equals | any |
| `gt` | Greater than | number |
| `gte` | Greater than or equal | number |
| `lt` | Less than | number |
| `lte` | Less than or equal | number |
| `exists` | Path exists in output | none |
| `contains` | String contains substring | string |
| `matches` | Regex match | string (pattern) |
| `between` | Value in inclusive range | [min, max] array |
