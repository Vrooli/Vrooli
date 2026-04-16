# Product Requirements Document (PRD)

> **Template Version**: 2.0
> **Canonical Reference**: `/scenarios/prd-control-tower/docs/CANONICAL_PRD_TEMPLATE.md`
> **Validation**: Enforced by `prd-control-tower` + `scenario-auditor`
> **Policy**: Generated once and treated as read-only (checkboxes may auto-update)

## 🎯 Overview

- **Purpose**: Validate cross-steer skill interoperability, development tooling correctness, and scenario quality infrastructure by testing all development tools against known-good reference scenarios. This scenario detects conflicts between steer skills, false positives/negatives in tooling (scenario-auditor, test-genie, scenario-completeness-scoring), and skill maturity gaps — enabling the meta optimization team to autonomously improve the entire development ecosystem.
- **Primary users/verticals**:
  - prompt-manager's meta optimization team (primary consumer — uses CLI to detect skill issues and tooling regressions)
  - Ecosystem-manager (validates that its development loop tools produce correct results)
  - Human developers (understanding steer skill coverage, conflicts, and gaps across scenarios)
  - AI agents authoring or improving steer skills (checking their work against reference implementations)
- **Deployment surfaces**: Go API, React UI, CLI
- **Value promise**:
  - Make cross-steer conflicts visible and testable instead of discovered through expensive agent iteration loops
  - Provide ground-truth validation for development tools — if a tool gives bad results on a known-good reference, the tool is wrong
  - Measure steer skill maturity through configurability: skills with no structural config are too vague for programmatic validation
  - Enable the autonomous migration from AI-powered (expensive, slow) quality checks to programmatic (fast, deterministic) quality checks
  - Surface exactly where each steer skill affects a reference scenario, revealing overlaps, gaps, and conflicts between skills

## 🎯 Operational Targets

### 🔴 P0 – Must ship for viability

- [ ] OT-P0-001 | Reference Scenario Registry | Register existing scenarios as references, storing which template they are based on. Support CRUD operations via API and CLI. Persist to PostgreSQL.
- [ ] OT-P0-002 | Skill Connection Management | Connect prompt-manager skills to reference scenarios via the prompt-manager API. Store the skill's version number (from prompt-manager's version history) and content hash at connection time. Support connecting/disconnecting skills via API and CLI.
- [ ] OT-P0-003 | Skill Drift Detection | Compare a connected skill's stored version/hash against its current version in prompt-manager. Flag when skill content has changed since the connection was configured, indicating the configuration may be stale.
- [ ] OT-P0-004 | Structural Expectation Config | For each skill-reference connection, allow defining structural expectations: required/optional folders, files matching glob patterns, and content snippets expected at specific file locations. Store as JSON configuration per connection.
- [ ] OT-P0-005 | CLI Tool Expectation Config | For each skill-reference connection, allow defining CLI tool assertions: a command to run (read-only tools with `--json` output), a JSONPath expression, a comparison operator (eq, neq, gt, gte, lt, lte, exists, contains, matches, between), and an expected value. These represent the validation/search tools referenced in steer skills.
- [ ] OT-P0-006 | Structural Validation Engine | Run all structural expectations against a reference scenario and report pass/fail per expectation, per skill connection. Detect missing folders, missing files, and snippet mismatches.
- [ ] OT-P0-007 | CLI Tool Validation Engine | Execute configured CLI tool commands against a reference scenario, parse JSON output, evaluate assertions, and report pass/fail per assertion, per skill connection.
- [ ] OT-P0-008 | Skill Overlap Detection | For each reference scenario, analyze all connected skill configurations to identify overlapping structural expectations (multiple skills expecting structures in the same files/folders). Report overlaps with the specific skills involved and the conflicting expectations.
- [ ] OT-P0-009 | Validation Report API | GET endpoint that returns a comprehensive validation report for a reference scenario: all skill connections, their structural and CLI tool assertion results, overlaps detected, and unconfigured skills.
- [ ] OT-P0-010 | CLI Interface | CLI commands for all core operations: managing references, connecting skills, configuring expectations, running validations, and viewing reports.

### 🟠 P1 – Should have post-launch

- [ ] OT-P1-001 | Scenario-Auditor Baseline | Run `scenario-auditor standards scan <reference> --wait --json` against each reference and assert zero violations (or a configured allowlist). Report any unexpected violations as tooling issues.
- [ ] OT-P1-002 | Test-Genie Baseline | Run `test-genie execute <reference> --preset comprehensive --json` against each reference and assert all phases pass. Report any unexpected failures as tooling issues.
- [ ] OT-P1-003 | Completeness Scoring Baseline | Run `scenario-completeness-scoring score <reference> --json` against each reference and assert score >= 96 (Production Ready). Report unexpected low scores as scoring model issues.
- [ ] OT-P1-004 | Conflict Detection | Beyond overlap detection, identify semantic conflicts: structural expectations from different skills that are mutually exclusive (e.g., skill A requires folder structure X, skill B requires incompatible structure Y in the same location).
- [ ] OT-P1-005 | Skill Maturity Score | Calculate a maturity score per connected skill based on: has structural config (weighted), has CLI tool assertions (weighted), all assertions pass (weighted), no conflicts with other skills (weighted). Surface unconfigured skills as the lowest maturity.
- [ ] OT-P1-006 | Coverage Map | For a reference scenario, produce a file/folder-level coverage map showing which steer skills have expectations covering each area, and which areas have no skill coverage.
- [ ] OT-P1-007 | Dashboard UI | Visual overview showing all references, their connected skills, validation status (pass/fail/unconfigured), overlaps, and conflicts in an interactive layout.
- [ ] OT-P1-008 | Skill Detail View | Drill-down UI showing a single skill's connection to a reference: its structural expectations, CLI assertions, where in the reference it applies (file tree highlighting), and related/conflicting skills.
- [ ] OT-P1-009 | Reference Health Summary | Aggregate health endpoint combining: validation results, tooling baselines (auditor/test-genie/completeness), skill maturity scores, and coverage map into a single health score per reference.
- [ ] OT-P1-010 | Heartbeat Mode | Run all validations on a configurable schedule and persist results history. Detect regressions when a previously-passing validation starts failing.

### 🟢 P2 – Future / expansion

- [ ] OT-P2-001 | Auto-Config Suggestions | Analyze a steer skill's SKILL.md content and suggest structural expectations and CLI tool assertions that could be configured, based on patterns in the skill text (folder references, CLI commands mentioned, convergence patterns).
- [ ] OT-P2-002 | Multi-Template Support | Support reference scenarios based on different templates (e.g., CLI-only, landing-page-react-vite) with template-aware validation.
- [ ] OT-P2-003 | Skill Diff Impact Analysis | When a skill's content changes (drift detected), analyze what structural/CLI expectations might be affected and surface them for review.
- [ ] OT-P2-004 | Cross-Reference Consistency | When a skill is connected to multiple references (of different templates), verify that its config is consistent across all references where applicable.
- [ ] OT-P2-005 | Tooling Regression History | Track tooling baseline results over time and detect trends (e.g., scenario-auditor introducing new false positives across releases).
- [ ] OT-P2-006 | Export/Import Configs | Export skill-reference configurations as portable JSON for sharing or backup.
- [ ] OT-P2-007 | Webhook Notifications | Notify on validation failures, drift detection, or new conflicts via configurable webhooks.
- [ ] OT-P2-008 | Skill Authoring Integration | When creating a new steer skill via prompt-manager, automatically suggest connecting it to relevant references and guide config creation.

## 🧱 Tech Direction Snapshot

- **Preferred stacks / frameworks**:
  - API: Go with standard library HTTP server
  - UI: React + TypeScript + Vite (via react-vite template)
  - CLI: Go with cli-core integration
  - Storage: PostgreSQL for persistent data (connections, configs, validation results)
- **Data + storage expectations**:
  - Reference registry, skill connections, structural expectations, CLI assertions, and validation results all stored in PostgreSQL
  - Skill content and versions fetched from prompt-manager API (not duplicated locally)
  - Validation result history retained for trend detection
- **Integration strategy**:
  - prompt-manager API: Read skill content, versions, metadata. This scenario is a consumer only — prompt-manager has no dependency on this scenario.
  - scenario-auditor CLI: Run audits against reference scenarios and parse JSON output
  - test-genie CLI: Run test suites against reference scenarios and parse JSON output
  - scenario-completeness-scoring CLI: Score reference scenarios and parse JSON output
  - All integrations via CLI with `--json` output — no direct API-to-API coupling for tooling validation
- **Non-goals / guardrails**:
  - Does NOT modify reference scenarios (read-only validation)
  - Does NOT execute steer skills (they are text files for agents; executing them would be expensive and could modify references)
  - Does NOT replace prompt-manager's Graph or health scoring — it is a complementary, independent system
  - Does NOT manage skills — skill CRUD remains in prompt-manager
  - Does NOT run ecosystem-manager loops — it validates the tools those loops use

## 🤝 Dependencies & Launch Plan

- **Required resources**:
  - PostgreSQL (primary data store for connections, configs, results)
- **Scenario dependencies**:
  - prompt-manager (API consumer: read skills, versions, metadata)
  - scenario-auditor (CLI consumer: run audits against references)
  - test-genie (CLI consumer: run test suites against references)
  - scenario-completeness-scoring (CLI consumer: score references)
  - reference-react-vite (first reference scenario to validate against)
- **Operational risks**:
  - Dependency on external scenario CLIs being available and returning stable JSON formats
  - Steer skill content is free-form markdown; extracting structural meaning requires careful config authoring (initially manual, later AI-assisted via OT-P2-001)
  - Reference scenarios must be actively maintained as steer skills evolve
- **Launch sequencing**:
  1. P0: Core registry, connections, config, and validation engine with CLI
  2. P1: Tooling baselines (auditor, test-genie, completeness) + UI dashboard
  3. P2: Auto-suggestions, multi-template, cross-reference analysis

## 🎨 UX & Branding

- **Look & feel**: Clean developer-tool aesthetic. Data-dense dashboards with clear status indicators (pass/fail/unconfigured/drift). File tree visualizations for coverage mapping.
- **Accessibility**: WCAG 2.1 AA compliance. Color-blind safe status indicators (icons + color, not color alone).
- **Voice & messaging**: Technical and precise. "Validation passed", "Drift detected", "Conflict: skills X and Y both expect incompatible structures at path Z".
- **Branding hooks**: Development toolchain / quality infrastructure theme. Gear/shield iconography.

## 📎 Appendix

### Background: Why This Scenario Exists

The Vrooli ecosystem uses **steer skills** (managed by prompt-manager) to guide AI agents during scenario development loops (managed by ecosystem-manager). Each steer skill focuses on one architectural dimension (API design, storage patterns, CLI structure, testing architecture, etc.) and defines convergence patterns that agents follow to improve scenarios.

**The core problem**: While individual steer skills can be improved in isolation (via prompt-manager's meta optimization team), there is no mechanism to detect **cross-steer conflicts** — cases where two skills' guidance leads to incompatible architectural decisions in the same scenario. An agent running under api-steer might make structural decisions that a later cli-steer pass fights against, creating oscillation instead of convergence.

**The solution**: Reference scenarios serve as **known-good ground truth** implementations. By mapping steer skills to references with explicit structural and tooling expectations, we can:
1. Detect conflicts before they cause expensive agent iteration loops
2. Validate that development tools (scenario-auditor, test-genie, scenario-completeness-scoring) produce correct results on known-good code
3. Measure steer skill maturity through the ability to define programmatic expectations

### Key Design Decisions

1. **Skills are connected, not executed**: Steer skills are markdown text files consumed by AI agents. Executing them would be expensive (LLM tokens) and could modify reference scenarios. Instead, we store a declarative mapping of what each skill expects structurally.

2. **Configuration is the maturity metric**: A skill with no structural config means we cannot programmatically describe what it does to a scenario. This is a signal that the skill is too prose-heavy or unstructured for efficient validation — directly informing the meta optimization team about what to improve.

3. **Version-pinned connections**: Each skill connection stores the skill's version number (from prompt-manager's version history) and content hash at connection time. When the skill content changes, we flag potential drift — the configuration may no longer accurately represent what the skill does.

4. **JSONPath assertions for CLI tools**: Steer skills reference read-only CLI tools for search-space optimization (detecting files with insufficient test coverage, high cyclomatic complexity, missing configs, etc.). We configure assertions against these tools' `--json` output using a standard path + operator + value pattern, making validation fast and deterministic.

5. **Prompt-manager is not a dependency of this scenario**: This scenario consumes prompt-manager's API. The reverse dependency (prompt-manager's meta team using this CLI) is configured in prompt-manager's agents, not in this scenario's code.

6. **Reference scenarios are full scenarios**: They live in `scenarios/` and use all standard Vrooli tooling (Makefiles, service.json, lifecycle commands). This is necessary because skeletal references cannot exercise the real toolchain — you cannot validate that test-genie works correctly against a skeleton.

### The Promotion-Retirement Vision

This scenario is part of a larger system goal: **migrating AI-powered (expensive, slow) quality checks to programmatic (fast, deterministic) quality checks**. The progression:

1. **Today**: Steer skills are markdown guidance — agents read them, interpret them, do open-ended assessment work. Expensive and slow.
2. **With this scenario**: Skill expectations are configured as structural checks and CLI assertions. Validation is programmatic and fast.
3. **Future**: The meta optimization team uses this scenario's CLI to identify poorly-configured skills, improves them to be more structured, and builds/enhances CLI tools that encode what used to be prose guidance into programmatic checks.
4. **Ideal end state**: A steer skill becomes a single CLI call that programmatically assesses and potentially fixes a scenario — deterministically, fast, and cheaply.

### Ecosystem Integration Flow

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

### Assertion Operator Reference

CLI tool assertions use the following operators to validate JSON output from development tools:

| Operator | Meaning | Example |
|----------|---------|---------|
| `eq` | Equals | `{"path": "$.success", "op": "eq", "value": true}` |
| `neq` | Not equals | `{"path": "$.status", "op": "neq", "value": "failed"}` |
| `gt` | Greater than | `{"path": "$.score", "op": "gt", "value": 80}` |
| `gte` | Greater than or equal | `{"path": "$.score", "op": "gte", "value": 96}` |
| `lt` | Less than | `{"path": "$.violations", "op": "lt", "value": 5}` |
| `lte` | Less than or equal | `{"path": "$.penalty", "op": "lte", "value": 2}` |
| `exists` | Path exists in output | `{"path": "$.breakdown.quality", "op": "exists"}` |
| `contains` | String contains | `{"path": "$.status", "op": "contains", "value": "pass"}` |
| `matches` | Regex match | `{"path": "$.version", "op": "matches", "value": "^\\d+\\.\\d+"}` |
| `between` | Value in range [min, max] | `{"path": "$.rate", "op": "between", "value": [0, 1]}` |
