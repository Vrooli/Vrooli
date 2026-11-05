# Product Requirements Document (PRD)

## 🎯 Overview
Purpose: Architecture Cartographer maps the actual import-and-ownership graph of a Vrooli scenario, compares it against a manifest-defined target architecture, and guides agents through per-domain migrations that converge the real architecture toward the intended one. It is the L5 "programmatic drift checks" maturity tool called out by the screaming-architecture audit, and it is the durable substrate that replaces token-heavy manual audits.

Target users: Vrooli migration agents, scenario maintainers, screaming-architecture auditors, and template authors. Indirect beneficiaries are every downstream scenario that needs to be realigned to a newer template.

Deployment surfaces: CLI (primary surface for agents), API (Connect-RPC for programmatic consumers and the UI), UI (graph visualization, conflict workbench, history dashboards), plus integration with knowledge-observatory-style health workflows.

Value proposition: Cut manual screaming-architecture audit and migration cost by orders of magnitude. Replace prose-heavy reviews with reproducible graphs, classified conflicts, ranked fix suggestions with evidence, and mechanical execution of file moves and import rewrites — while keeping every design decision in human (or LLM-agent) hands.

## 🎯 Operational Targets

Operational targets are measurable outcomes; checkboxes may auto-update based on validation.

### 🔴 P0 – Must ship for viability
- [ ] OT-P0-001 | Ground-Truth Code Graph | Produce a deterministic, reproducible graph of files, packages, symbols, and import edges per target scenario by delegating to go-code-graph and typescript-code-graph; cartographer never parses source code itself.
- [ ] OT-P0-002 | API Manifest Contract | Define and validate a manifest schema declaring domains, allowed_dependencies, shared substrate locations, domain glossaries, signal weights, confidence thresholds, and transitional declarations as the trust anchor for intended architecture.
- [ ] OT-P0-003 | Drift Findings Engine | Deterministically compare the actual graph against the manifest target and emit a typed Conflict envelope (id, type, severity, locations, description, suggested_fixes, evidence) via a pluggable detector registry that accepts new conflict types without changing the envelope.
- [ ] OT-P0-004 | Pluggable Signal Ladder for Auto-Placement | Implement deterministic-only scoring signals (path tokens, import-cluster community detection, importer-voting majority, test-file coupling, symbol-glossary matching, git co-edit) as pure functions with weights and thresholds in the manifest; every verdict carries Reason+Evidence for explainability; no embeddings in v1.
- [ ] OT-P0-005 | Cycle Detection and Classification | Detect SCCs across the import graph and classify cycles by pattern (type-only, junk-drawer, cross-domain, within-domain) to surface appropriate fix suggestions (extract-shared-types, invert-dependency, manual).
- [ ] OT-P0-006 | Conflict-Driven CLI Workbench | Deliver a CLI that lets agents review unresolved conflicts, read source on demand, assign chunks to domains, mark resolutions, and run validate to re-verify; the CLI is a decision-tracker, verifier, and mechanical executor — not a business-logic author.
- [ ] OT-P0-007 | Per-Domain Apply | Emit ordered migration plans as per-domain file-move and import-rewrite operations applied one domain at a time with atomic commits; whole-scenario big-bang apply is discouraged and guarded by analytics.
- [ ] OT-P0-008 | Build-Green Baseline Guardrail | Capture build (and optionally test) status at migration start; refuse to land conflict resolutions or per-domain applies when the build has broken since baseline; allow --force only with a required --note that is logged in analytics.
- [ ] OT-P0-009 | Analytics from Day One | Persist conflict detection events, resolution methods/outcomes, auto-placement verdicts and overrides, and build status deltas; suppress historical success rates until N≥5 observations; override tracking is the primary signal for ladder calibration and recipe candidate identification.
- [ ] OT-P0-010 | Connect-RPC + CLI + UI Parity | Expose every cartographer capability through proto+Connect-RPC with CLI and UI as translation layers; no REST/JSON workarounds permitted.

### 🟠 P1 – Should have post-launch
- [ ] OT-P1-001 | UI Graph Visualization | Render actual vs. expected architecture as a mermaid/DOT graph with conflict overlay, drill-down, and per-domain filtering; keyboard-navigable with accessible text alternatives required.
- [ ] OT-P1-002 | Refactor Recipes (First Batch) | Ship extract-shared-types, invert-dependency, and split-file recipes as mechanical executors for known patterns, each validated by the build-green guardrail and released only after history shows ≥10 recurring instances.
- [ ] OT-P1-003 | Transitional State Lifecycle | Support explicit manifest declarations for re-export shims, adapters, and duplicated types pending merge; auto-detect undeclared-transitional patterns as warning conflicts; enforce two-tier strictness (--mid-migration tolerates, --strict errors when expires_when predicates fire).
- [ ] OT-P1-004 | Snapshot Persistence | Store graph snapshots across runs in a local SQLite store so agents can diff before-domain vs. after-domain state and measure drift over time; no external resource dependency.
- [ ] OT-P1-005 | knowledge-observatory + ui-health Integration | Surface cartographer docs through the existing doc-health validation pattern; expose cartographer findings to ui-health where useful without duplicating its React parsing logic.

### 🟢 P2 – Future / expansion
- [ ] OT-P2-001 | Embedding-Based Suggestion Ranker | When deterministic signals are exhausted, surface ranked candidate domains for orphaned chunks by reusing scenario-dependency-analyzer's Ollama+Qdrant pipeline; suggestions only — never auto-applied.
- [ ] OT-P2-002 | Additional Language Coverage | Extend graph extraction to Python, Rust, and other languages via new code-graph scenarios; cartographer itself remains unchanged.
- [ ] OT-P2-003 | swarm-manager Backlog Integration | File detected conflicts and migration plans as initiative items in swarm-manager for human triage on scenarios the agent cannot complete autonomously.
- [ ] OT-P2-004 | Recipe Identification Pipeline | Surface candidate-recipe reports when ≥10 instances of the same manual edit pattern are observed; recipe authoring remains human-in-the-loop and is never auto-generated.
- [ ] OT-P2-005 | Signal-Weight Calibration | Provide arch-cart calibrate to propose weight adjustments based on override history; all weight changes require explicit human acceptance before taking effect.
- [ ] OT-P2-006 | Architecture Maturity Scoring | Roll up per-scenario cartographer findings into a maturity level (Opaque → Documented → Capability map → Domain-owned → Boundary+seams → Programmatic drift checks) per the screaming-architecture audit doctrine.

## 🧱 Tech Direction Snapshot
Preferred stacks: Go API with Connect-RPC for the cartographer service; React+Vite UI; CLI in Go using the cliapp.ArgSchema pattern from the react-vite template; proto-first schemas registered through api/internal/modules/registry.go. All cross-scenario calls go through proto+Connect-RPC — never REST/JSON. Three-layer scenario architecture: (1) language code-graph scenarios (go-code-graph wrapping golang.org/x/tools/go/packages; typescript-code-graph wrapping ts-morph) — pure parsers with no Vrooli semantics; (2) framework semantics scenarios (react-component-library upgraded to consume typescript-code-graph instead of regex parsing, adding React/widget interpretation); (3) applications (architecture-cartographer consumes the language layer directly; ui-health continues to consume react-component-library). Plug-in seams defined on day one: Detector interface (conflict producers), Resolver interface (mechanical fixers), Signal interface (chunk→domain scoring), and Recipe interface (refactor executors) — all accept an immutable graph snapshot and none mutate the graph during scoring.

Preferred storage: SQLite for local snapshots, conflict history, override tracking, and analytics. No new shared resource dependency unless a future capability genuinely requires one. No Ollama/Qdrant dependency in v1.

Integration strategy: Generic coordinator plus framework-specific analyzers, shelling out to other scenario CLIs and calling their Connect-RPC services. Cartographer never parses source code itself; all parsing belongs in language code-graph scenarios. The Conflict envelope stays stable across versions to ensure backward compatibility for all consumers.

Non-goals: No fully automatic deep refactors in v1. No design-decision automation (interface naming, package merges, and responsibility splits remain human decisions). No embedding-based auto-placement in v1. No whole-scenario big-bang apply path that bypasses per-domain validation. No fabricated suggestions when signal evidence is insufficient — the CLI will honestly state "no good automatic option, manual required." No parser code lives in architecture-cartographer.

## 🤝 Dependencies & Launch Plan
Required resources: None initially. SQLite is bundled. No Ollama/Qdrant dependency in v1.

Scenario dependencies: go-code-graph and typescript-code-graph must exist before or alongside cartographer launch — both are new scenarios. react-component-library is upgraded (not depended on by cartographer) as part of the typescript-code-graph rollout. The knowledge-observatory pattern is referenced for doc-health integration but carries no runtime dependency.

Operational risks: (a) False confidence from incomplete graph extraction — strict equivalence tests against fixture scenarios are required before shipping the code-graph layer. (b) Unsafe migration automation — every code-modifying recipe must leave the build green or roll back atomically; no partial applies permitted. (c) Noisy findings when manifests are underspecified — sensible defaults plus explicit "unknown domain" markers are used rather than silent guesses. (d) Repeat of the swarm-manager big-bang failure — guarded by per-domain apply as the primary path, whole-scenario apply as a discouraged escape hatch, and analytics tracking of how applies actually land in practice.

Launch sequencing: (1) Ship typescript-code-graph and go-code-graph as standalone tools with fixture-validated correctness. (2) Cartographer MVP — read-only graph extraction + manifest comparison + Conflict envelope + cycle detection + CLI workbench (no apply yet). (3) Per-domain apply — file moves and import rewrites for Go via go-code-graph helpers with build-green guardrail enforced. (4) UI graph visualization with conflict overlay and per-domain filtering. (5) TypeScript apply support. (6) First refactor recipe (extract-shared-types) when history justifies ≥10 instances. (7) Snapshot persistence and historical drift trend dashboards.

## 🎨 UX & Branding
User experience: Dense operational workbench targeting migration agents and scenario maintainers — not a marketing surface. Primary interactions are conflict review, domain assignment, resolution marking, validate runs, and per-domain apply. Every conflict surfaces a classified pattern, 2–4 ranked fix options with pre-baked commands and caveats, and an explicit "no good automatic option" message when signals are insufficient. The CLI follows the human-friendly contract throughout. Graphs are paired with conflict lists, evidence tables, and per-domain progress meters for scan-friendly density.

Visual design: Inherits the vrooli-default design kit and AppShell pattern from the react-vite template. Utilitarian, scan-friendly density appropriate for an operational workbench. PWA install surface seeded from the template (`ui/public/site.webmanifest`, `apple-icon-180.png`, `favicon-196.png`, and maskable manifest icons kept valid); generic icons to be replaced when final product branding stabilizes. Color is never the sole information channel for severity — severity levels are always accompanied by labels or icons. Vrooli-default design tokens applied throughout; no custom token overrides without design review.

Accessibility: WCAG AA target. Every visual graph has a keyboard-navigable text alternative. Every conflict has a screen-readable description carrying the same evidence visible in the UI. Color contrast meets AA ratios for all text and interactive elements. Focus order is logical and visible for keyboard-only navigation across the conflict workbench, graph views, and apply workflows.