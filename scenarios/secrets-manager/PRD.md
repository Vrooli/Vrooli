# Product Requirements Document (PRD)
> Version 2.0 — Operational Targets aligned with the scenario-generator standard. Detailed implementation guidance now lives in `requirements/`.

## 🎯 Overview
- Vrooli-wide security operations console that inventories every scenario/resource secret, validates what's in Vault, and surfaces drift before production failures.
- Serves internal platform engineers, ecosystem maintainers, and CI agents who need a quick read on whether local resources are safe to start.
- Shipping surfaces include the Go API, React/Vite dashboard, and CLI wrapper so both humans and automations can consume the same signals.
- Core value: eliminate "missing secret" fire drills, expose security regressions before launch, and keep the recursive Vrooli stack trustworthy.

## 🎯 Operational Targets
### 🔴 P0 – Must ship for viability
- [ ] OT-P0-001 | Vault Coverage Map | Discover every declared secret, detect missing/invalid entries, and expose the results via `/vault/secrets/status` + CLI/UI consumers.
- [ ] OT-P0-002 | Cross-Scenario Vulnerability Scanning | Crawl scenario source trees, flag vulnerabilities with severity + file metadata, and persist structured findings.
- [ ] OT-P0-003 | Unified Compliance Scoring | Blend vault coverage + scan posture into an API response that highlights risk score trends and supports health checks.

### 🟠 P1 – Should have post-launch
- [ ] OT-P1-001 | Guided Provisioning & Export | Provide APIs/CLI hooks to provision secrets, re-run validation, and export the values safely into workflows.
- [ ] OT-P1-002 | Operator Dashboard | Deliver the dark-chrome React UI with hero stats, filters, and actionable tables sourced directly from the API/requirements registry.
- [ ] OT-P1-003 | Historical Telemetry | Persist validation + scan history in Postgres so compliance scores include context and trend deltas.
- [ ] OT-P1-004 | Automation-Friendly CLI | Keep the CLI as a thin API proxy so CI and other scenarios can script audits without bespoke logic.
- [ ] OT-P1-005 | Lifecycle & Testing Guardrails | Ensure lifecycle setup, phased tests, and resource seeds keep the scenario reproducible in dev/CI.

### 🟢 P2 – Future / expansion ideas
- [ ] OT-P2-001 | Auto Remediation Suggestions | Offer prescriptive fixes or trigger downstream agents that can resolve common misconfigurations.
- [ ] OT-P2-002 | Trend & Forecasting Analytics | Highlight posture drift over rolling windows and alert on unusual deltas.
- [ ] OT-P2-003 | Policy Enforcement Hooks | Allow other scenarios or orchestrators to block launches until secrets/security targets hit thresholds.

## 🧱 Tech Direction Snapshot
- React + TypeScript + Vite UI, Go API, Postgres persistence, and HashiCorp Vault via the `resource-vault` CLI remain the canonical stack.
- Lifecycle-managed ports, pnpm workspaces, and shared packages (`@vrooli/api-base`, `@vrooli/iframe-bridge`) keep the scenario aligned with the react-vite template.
- CLI should strictly proxy API endpoints; no business logic or duplicated parsing lives outside the Go service.
- Requirements, tests, and docs must cite `[REQ:ID]` for traceability so future agents can update coverage programmatically.

## 🤝 Dependencies & Launch Plan
- Hard dependencies: Vault (secret storage/validation), Postgres (metadata & telemetry), claude-code optional for remediation experiments.
- Setup order: install CLI → build Go API → bootstrap Postgres schema/seed → install UI deps → build UI bundle; life-cycle commands enforce this order.
- Launch readiness requires phased tests (`test/run-tests.sh`), health checks for API/UI, and `requirements/index.json` kept in sync with the PRD.
- Operational risks center on stale resource manifests and long security scans; mitigate via scheduled scans + requirement coverage reporting.

## 🎨 UX & Branding
- Dark chrome / neon accents consistent with security tooling, WCAG AA contrast, and lucide iconography from the shared react-vite template.
- Dashboard flows prioritize quick-status tiles, detailed tables for vault coverage and vulnerabilities, and inline badges for severity.
- Animations stay subtle (no flashing) to support long-running operator sessions; all colors have semantic text fallbacks for accessibility.
