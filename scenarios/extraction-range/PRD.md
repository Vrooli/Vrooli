# Product Requirements Document (PRD)

> **Template Version**: 2.0
> **Canonical Reference**: `scenarios/business-health/docs/reference/canonical-prd-template.md`
> **Validation**: Enforced by `business-health` (`validate scenario extraction-range`)
> **Policy**: Generated once and treated as read-only (checkboxes may auto-update)

## 🎯 Overview

- **Purpose**: Provide Vrooli with an owned website data-extraction proving ground: graded, reusable practice pages with held answers and per-tool scoring.
- **Primary users/verticals**: The freelance website-extraction toolkit, Vrooli agents building extraction workflows, and scenario authors who need repeatable extraction fixtures.
- **Deployment surfaces**: Go API and CLI for level discovery, scoring and scoreboards; owned HTML practice pages for crawlers; React UI for operator review.
- **Value promise**: Compare extraction tools on the same known-answer tasks, find failures by page family and improve delivery odds before client work. No client target or data is used.

## 🎯 Operational Targets

> Checkboxes auto-update from requirements sync; do not hand-edit them.

### 🔴 P0 – Must ship for viability

- [x] OT-P0-001 | Graded coverage across ten families | Serve at least 40 data-defined levels, with at least four each for tables, pagination/deep links, search, script-loaded/endless scroll, sessions, downloads, messy markup, changing layouts, rate/errors/timeouts, and owned-ground bot checks.
- [x] OT-P0-002 | Known-answer scoring and tool scoreboard | Hold an expected answer for each level; score submitted records deterministically, prevent duplicate inflation, and keep the best per-tool/per-level score with precision, recall, F1 and pass status.
- [x] OT-P0-003 | Reachable owned practice pages | Serve each challenge's requested target behavior through lifecycle-managed local pages and publish a stable task/API contract without exposing held answers in level metadata.

### 🟠 P1 – Should have post-launch

- [ ] OT-P1-001 | Operator review surface | Provide a polished, accessible browser UI with family filters, challenge instructions, score submission and per-tool scoreboard, including loading, empty and error states.
- [ ] OT-P1-002 | Persistent score history | Persist best score and submission history across scenario restarts, with explicit tool and family filters.
- [ ] OT-P1-003 | Broader challenge corpus | Grow the data-defined catalog beyond its initial 40 levels with varied record volumes and held-out variants while keeping answer keys reviewable.

### 🟢 P2 – Future / expansion

- [ ] OT-P2-001 | Benchmark comparison exports | Export reproducible score summaries for toolkit iterations and compare families and historical runs.
- [ ] OT-P2-002 | Composable scenario fixtures | Allow other Vrooli scenarios to add owned challenge data through a documented schema and validate it without code changes.

## 🧱 Tech Direction Snapshot

- Preferred stacks / frameworks: Go API and CLI, protobuf Connect contracts, React and TypeScript operator UI.
- Data + storage expectations: Versioned JSON level definitions with held answers; SQLite-backed durable submissions and best-score aggregates.
- Integration strategy: Stable HTTP routes for extraction tools, Connect for Vrooli clients, and lifecycle-owned local pages. No resource or scenario dependency is required for the core slice.
- Non-goals / guardrails: No crawling of real websites, client data, prospects, or unauthorized targets; no real-site bot circumvention; no credentials, purchases, or external services.

## 🤝 Dependencies & Launch Plan

- Required resources: None.
- Scenario dependencies: None; the scenario is a self-contained owned practice website.
- Operational risks: Keep synthetic access codes and challenge tokens confined to owned local fixtures; preserve deterministic answer/data alignment; verify the routed SQLite scoreboard survives a lifecycle restart.
- Launch sequencing: Complete scenario contract and data schema, implement the grader and owned pages, expose API and CLI, finish the operator UI, qualify all families and tool integration, then close the required scenario gates.

## 🎨 UX & Branding

- Look & feel: Dense but readable practice console, clear family and difficulty markers, calm contrast, responsive layout, and no decorative motion that slows repeated comparisons.
- Accessibility: WCAG AA contrast, keyboard-operable filters and forms, semantic tables, visible focus, labels and announced loading/error states.
- Voice & messaging: Direct, neutral, evidence-focused; distinguish expected from submitted rows and show the exact score.
- Branding hooks: Use the Extraction Range name and distinct family color accents only where status meaning remains accessible without color.
