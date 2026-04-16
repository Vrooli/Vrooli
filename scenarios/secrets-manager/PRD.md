# Product Requirements Document (PRD)
> Version 2.0 — Operational Targets aligned with the scenario-generator standard. Detailed implementation guidance now lives in `requirements/`.

## 🎯 Overview
- Vrooli-wide security operations console that inventories every scenario/resource secret, validates what's in Vault, and surfaces drift before production failures.
- Serves internal platform engineers, ecosystem maintainers, and CI agents who need a quick read on whether local resources are safe to start.
- Shipping surfaces include the Go API, React/Vite dashboard, and CLI wrapper so both humans and automations can consume the same signals.
- Core value: eliminate "missing secret" fire drills, expose security regressions before launch, and keep the recursive Vrooli stack trustworthy.

## 🎯 Operational Targets
### 🔴 P0 – Must ship for viability
- [ ] OT-P0-001 | Tier-Aware Secret Intelligence | Maintain a complete inventory of every resource/scenario secret, normalize metadata (owner, tier fitness, rotation notes), and expose per-resource drilldowns with current Vault status.
- [ ] OT-P0-002 | Threat & Vulnerability Detection | Continuously scan scenarios/resources for hardcoded secrets and insecure patterns, rank findings by severity, and link each issue to contextual remediation guidance.
- [ ] OT-P0-003 | Deployment Readiness Engine | Produce tier-specific secret strategies (strip/generate/prompt/delegate), emit bundle-ready manifests for deployment-manager + scenario-to-*, and verify no infrastructure secrets leak outside Tier 1.
- [ ] OT-P0-004 | Guided Operator Journeys | Ship an orientation hub with hero stats and journey cards plus multi-step flows that walk operators from detection → action (configure secrets, fix vulns, prep deployments) without guesswork.

### 🟠 P1 – Should have post-launch
- [ ] OT-P1-001 | Guided Provisioning & Export | Provide APIs/CLI hooks to provision secrets, re-run validation, and export the values safely into workflows.
- [ ] OT-P1-002 | Operator Dashboard | Deliver the dark-chrome React UI with hero stats, filters, actionable tables, and guided flows sourced directly from the API/requirements registry.
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

## 🎯 Capability Definition
Secrets Manager provides centralized secret intelligence across the Vrooli platform. It discovers and inventories all secrets required by scenarios and resources, validates them against Vault storage, performs security scanning for vulnerabilities, and generates deployment-ready manifests. This capability transforms secret management from manual and error-prone to automated and auditable.

## 📊 Success Metrics
- Secret coverage: % of required secrets present in Vault
- Vulnerability detection: Count and severity distribution of security findings
- Deployment readiness: % of scenarios with complete tier-appropriate secret strategies
- API response time: Health/compliance endpoints respond < 500ms
- CLI usability: Commands complete successfully without manual API URL configuration

## 🏗️ Technical Architecture
- **API Layer**: Go service exposing REST endpoints for vault status, vulnerabilities, compliance, and deployment manifests
- **Data Layer**: Postgres for secret requirements, validation history, and scan results
- **Integration Layer**: `resource-vault` CLI for Vault operations with graceful fallback
- **UI Layer**: React + Vite SPA consuming API via standard fetch patterns
- **CLI Layer**: Thin command wrappers that proxy API endpoints without business logic duplication

## 🖥️ CLI Interface Contract
```bash
secrets-manager status              # Health + quick stats
secrets-manager vault list          # All secrets with validation status
secrets-manager vault validate      # Re-run validation checks
secrets-manager security scan       # Security scan across scenarios/resources
secrets-manager security compliance # Aggregate compliance report
secrets-manager deployment plan     # Tier-aware manifest export
```
All commands accept `--api-base`, `--auto-start`, and `--json` for automation.

## 🔄 Integration Requirements
- **Vault Integration**: Depends on `resource-vault` CLI for secret operations
- **Postgres Integration**: Requires schema and seed data during setup
- **Deployment Manager**: Exposes `/deployment/secrets` endpoint for manifest requests
- **Scenario-to-* Tools**: Provides tier-specific secret bundles via API
- **CI/CD**: JSON output mode enables automated compliance checks

## 🎨 Style and Branding Requirements
- Color scheme: Dark charcoal background (#1a1a1a) with cyan accents (#00bcd4) for actions
- Typography: Monospace for code/secrets, sans-serif (Inter) for prose
- Icons: Lucide icon set for consistency with other Vrooli scenarios
- Status indicators: Red/yellow/green with text labels (not color-only)
- Contrast: All text meets WCAG AA minimum 4.5:1 ratio

## 💰 Value Proposition
- **Platform Engineers**: Eliminate "missing secret" deployment failures through continuous validation
- **Security Teams**: Automated vulnerability scanning replaces manual code review
- **DevOps/CI**: JSON-first CLI enables gating deployments on compliance thresholds
- **Business**: Reduces incident response costs and accelerates scenario delivery timelines
- **ROI**: Estimated 10-15 hours saved per quarter per engineer from reduced secret-related debugging

## 🧬 Evolution Path
- **v1.0**: Core vault validation + security scanning + basic dashboard
- **v1.1**: Historical trending + compliance deltas over time
- **v2.0**: Auto-remediation suggestions via claude-code integration
- **v2.1**: Policy gating hooks for deployment-manager + scenario-to-* tools
- **v3.0**: Secret rotation automation + proactive expiration alerts

## 🔄 Scenario Lifecycle Integration
- **Setup Phase**: Builds API binary, installs UI dependencies, applies Postgres schema/seed
- **Develop Phase**: Starts API + UI servers on lifecycle-managed ports
- **Test Phase**: Executes phased tests (structure → unit → integration) with requirement tagging
- **Stop Phase**: Gracefully terminates API/UI processes via lifecycle manager

## 🚨 Risk Mitigation
- **Stale Manifests**: Scheduled scans + validation on scenario/resource file changes
- **Scan Performance**: Timeout limits + file count caps prevent runaway scans
- **Vault Unavailability**: Graceful fallback to local manifest parsing when Vault CLI fails
- **Schema Drift**: Migration tracking + idempotent seed scripts ensure reproducibility
- **Sensitive Data Exposure**: Health endpoints never return secret values, only metadata

## ✅ Validation Criteria
- Health endpoints return compliant responses per lifecycle schema
- All P0 operational targets have passing requirement tests
- Security scan completes across all scenarios without crashes
- CLI commands work without manual API URL configuration
- UI dashboard loads and displays real-time vault status

## 📝 Implementation Notes
- Prefer `resource-vault` CLI over direct Vault API calls for consistency
- Use structured logging (not log.Println) for production observability
- Tag all tests with `[REQ:ID]` for automated requirement tracking
- Keep CLI commands thin; move business logic to API layer
- Production UI uses built bundles, not dev server

## 🔗 References
- Vault CLI: `/resources/vault/README.md`
- Lifecycle System: `/docs/scenarios/LIFECYCLE.md`
- Requirement Tracking: `/docs/testing/guides/requirement-tracking.md`
- Template: `/templates/scenarios/react-vite/`
