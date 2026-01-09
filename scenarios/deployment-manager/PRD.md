# Product Requirements Document (PRD)

> **Version**: 1.0.0
> **Canonical Reference**: /scenarios/deployment-manager/PRD.md
> **Last Updated**: 2025-11-21
> **Status**: Initialization (Generator Phase)

---

## 🎯 Overview

### Purpose
deployment-manager is the **control tower for the entire deployment lifecycle** in Vrooli. It analyzes scenario dependencies, scores platform fitness, guides users through portable deployments, orchestrates platform-specific packagers, and monitors deployed scenarios across all deployment tiers.

This scenario transforms Vrooli's deployment story from "manually package and ship" to "intelligent, automated, multi-tier deployments with dependency optimization."

### Target Users
- **Primary**: Scenario developers deploying to desktop, mobile, or cloud platforms
- **Secondary**: End users managing deployed scenario instances
- **Tertiary**: Enterprise administrators overseeing compliance and multi-tenant deployments

### Deployment Surfaces
- **Tier 1 (Local/Dev Stack)**: Full Vrooli installation for development (baseline reference)
- **Tier 2 (Desktop)**: Windows, macOS, Linux standalone applications
- **Tier 3 (Mobile)**: iOS and Android native apps
- **Tier 4 (SaaS/Cloud)**: DigitalOcean, AWS, bare metal server deployments
- **Tier 5 (Enterprise)**: Hardware appliances with compliance overlays

### Value Proposition
**Without deployment-manager**: Scenarios are trapped in Tier 1 (local dev). Attempting to package scenarios fails because dependencies aren't portable (e.g., postgres doesn't run on mobile, ollama is too heavy for desktop bundles).

**With deployment-manager**: Any scenario can target any tier. The system analyzes dependencies, scores fitness, suggests swaps (postgres → sqlite for mobile), validates configurations, orchestrates packaging, and monitors deployments — all through an intuitive UI and CLI.

**Business Impact**: Unlocks Vrooli's full vision of scenarios as portable, deployable products that can ship as SaaS, desktop apps, mobile apps, or enterprise appliances.

---

## 🎯 Operational Targets

### 🔴 P0 – Must ship for viability

#### Dependency Analysis & Fitness Scoring
- [x] OT-P0-001 | Dependency Aggregation | Recursively fetch all resource and scenario dependencies (depth 10) within 5 seconds
- [x] OT-P0-002 | Circular Dependency Detection | Display clear error when circular dependencies detected (A → B → C → A)
- [x] OT-P0-003 | Multi-Tier Fitness Scoring | Calculate fitness score (0-100) for all 5 tiers within 2 seconds per dependency
- [x] OT-P0-004 | Fitness Score Breakdown | Break down fitness into 4+ sub-scores: portability, resource requirements, licensing, platform support
- [x] OT-P0-005 | Blocker Identification | Clearly state reason for blockers (fitness = 0) with actionable remediation
- [x] OT-P0-006 | Aggregate Resource Tallies | Display total memory, CPU, GPU, storage, network requirements with units

#### Dependency Swapping
- [x] OT-P0-007 | Swap Suggestions | Suggest at least one swap for each blocker/low-fitness dependency (<50) or show "No known alternatives"
- [x] OT-P0-008 | Swap Impact Analysis | Show fitness delta, trade-offs (pros/cons), and migration effort estimate for each swap
- [x] OT-P0-009 | Non-Destructive Swaps | Applying swaps updates deployment profile only, never modifies source code
- [x] OT-P0-010 | Real-Time Recalculation | Recalculate fitness and requirements within 1 second after swap application
- [x] OT-P0-011 | Cascading Swap Detection | Detect and warn about cascading impacts (e.g., swapping postgres removes dependent scenario X)

#### Deployment Profile Management
- [x] OT-P0-012 | Profile Creation | Create new deployment profile in ≤ 3 clicks from home screen
- [x] OT-P0-013 | Auto-Save with Debounce | Auto-save profile edits with 500ms debounce (or manual save button)
- [x] OT-P0-014 | Profile Export/Import | Support profile export/import in JSON format for version control
- [x] OT-P0-015 | Profile Versioning | Create new version for each edit with timestamp and user attribution
- [x] OT-P0-016 | Version History View | Display version history list with diffs showing changes
- [x] OT-P0-017 | Version Rollback | Restore all profile settings (swaps, secrets, env vars) within 2 seconds when rolling back

#### Secret Management Integration
- [x] OT-P0-018 | Secret Identification | Identify all required secrets within 3 seconds when profile created or swaps applied
- [x] OT-P0-019 | Secret Categorization | Categorize secrets as required, optional, dev-only, user-supplied, vault-managed
- [x] OT-P0-020 | Desktop/Mobile Secret Templates | Generate .env.template with human-friendly descriptions for desktop/mobile tiers
- [x] OT-P0-021 | SaaS/Enterprise Secret References | Generate Vault/AWS Secrets Manager references in correct format for cloud tiers
- [x] OT-P0-022 | Secret Validation Testing | Provide "Test API key" buttons that call endpoints to validate secrets before deployment

#### Pre-Deployment Validation
- [x] OT-P0-023 | Comprehensive Validation Checks | Run 6+ checks: fitness threshold, secret completeness, licensing, resource limits, platform requirements, dependency compatibility
- [x] OT-P0-024 | Fast Validation Execution | Complete all validation checks within 10 seconds for scenarios with ≤50 dependencies
- [x] OT-P0-025 | Validation Report UI | Display pass/fail/warning status with color coding (green/red/yellow) for each check
- [x] OT-P0-026 | Actionable Remediation Steps | Provide at least one actionable step for each failed check (e.g., "Upload code signing certificate here")
- [x] OT-P0-027 | SaaS Cost Estimation | Estimate monthly cost (compute, storage, bandwidth) within ±20% accuracy for SaaS/cloud tiers

#### Deployment Orchestration
- [x] OT-P0-028 | One-Click Deployment Trigger | Trigger deployment with single "Deploy" button after validation passes
- [x] OT-P0-029 | Profile Locking During Deployment | Lock profile to prevent edits during active deployment
- [x] OT-P0-030 | Real-Time Log Streaming | Stream deployment logs to UI with <2s latency via WebSocket or SSE
- [x] OT-P0-031 | Log Filtering and Search | Enable log filtering by level (info/warning/error) and full-text search
- [x] OT-P0-032 | Deployment Failure Handling | Capture error logs on failure and display "Retry" button with error context
- [x] OT-P0-033 | scenario-to-* Integration | Call platform-specific packagers (scenario-to-desktop, scenario-to-ios, etc.) with deployment profile
- [x] OT-P0-034 | scenario-to-* Auto-Discovery | Auto-discover installed scenario-to-* scenarios on startup, warn if missing required packagers

#### Dependency Graph Visualization
- [x] OT-P0-035 | Interactive Dependency Graph | Render dependency tree as interactive graph (React Flow or d3-force) with color-coded fitness scores
- [x] OT-P0-036 | Table View Alternative | Provide sortable table view with columns: name, type, fitness score, issues, suggested swaps
- [x] OT-P0-037 | Graph Rendering Performance | Render dependency graph within 3 seconds for scenarios with ≤100 dependencies

### 🟠 P1 – Should have post-launch

#### Post-Deployment Monitoring
- [ ] OT-P1-001 | Health Status Tracking | Check health (up/degraded/down) every 60 seconds for deployed scenarios
- [ ] OT-P1-002 | Graceful Timeout Handling | Mark health as "unknown" (not "down") after 10s timeout to avoid false alarms
- [ ] OT-P1-003 | SaaS Metrics Collection | Collect uptime, response time, error rate every 5 minutes with 30-day retention
- [ ] OT-P1-004 | Desktop/Mobile Telemetry | Aggregate daily telemetry (active installs, crash reports) for desktop/mobile tiers
- [ ] OT-P1-005 | Configurable Alerting | Allow users to configure alerts (email, Slack, webhook) with custom thresholds (e.g., "Error rate >5%")
- [ ] OT-P1-006 | Fast Alert Delivery | Fire alerts within 2 minutes of threshold breach
- [ ] OT-P1-007 | Monitoring Dashboard | Display at-a-glance status for all deployments (health grid, metric sparklines)
- [ ] OT-P1-008 | Deployment Detail Page | Load metrics charts (<5s) for last 24 hours with zoom/pan for longer ranges

#### Update & Rollback Management
- [ ] OT-P1-009 | Update Change Detection | Detect scenario changes (git diff, dependency updates) when viewing deployed profile
- [ ] OT-P1-010 | Update Change Summary | Show files modified, dependencies added/removed, breaking changes (if detectable)
- [ ] OT-P1-011 | Desktop/Mobile Update Distribution | Push updates to auto-update server or app stores within 30 minutes (excludes store review time)
- [ ] OT-P1-012 | SaaS Zero-Downtime Updates | Use blue-green or rolling deployment for stateless SaaS apps
- [ ] OT-P1-013 | Fast Rollback Execution | Restore previous version within 5 minutes for SaaS deployments
- [ ] OT-P1-014 | Automatic Rollback on Failure | Trigger rollback within 2 minutes if health checks fail post-update (when enabled)

#### Multi-Tier Deployment Orchestration
- [ ] OT-P1-015 | Multi-Tier Profile Support | Allow single profile to target multiple tiers simultaneously (e.g., desktop + iOS + SaaS)
- [ ] OT-P1-016 | Tier-Specific Settings UI | Show tier-specific settings (code signing, bundle IDs) only when that tier is selected
- [ ] OT-P1-017 | Parallel Tier Orchestration | Call scenario-to-* packagers concurrently for multi-tier deployments
- [ ] OT-P1-018 | Per-Tier Status Tracking | Independently track status (queued/deploying/deployed/failed) for each tier
- [ ] OT-P1-019 | Partial Failure Retry | Allow retry of failed tiers without redeploying successful ones after partial failures

#### Agent Integration (Migration Strategies)
- [ ] OT-P1-020 | Agent Migration Proposals | Agent receives profile + source code and responds with migration proposals within 5 minutes
- [ ] OT-P1-021 | Detailed Migration Estimates | Each proposal includes line count estimate, testing checklist, deployment impact
- [ ] OT-P1-022 | Auto-Generated Issue Creation | Agent-generated app-issue-tracker issues pre-filled with all context (no manual edits required)
- [ ] OT-P1-023 | Bulk Proposal Approval | Allow approve/reject proposals in bulk (e.g., "Create 5 issues" with one click)

#### Custom Swap Requests
- [ ] OT-P1-024 | Custom Swap Request Form | Provide form for users to request custom swaps (dependency name, desired alternative, justification)
- [ ] OT-P1-025 | Fast Issue Creation | Create app-issue-tracker issue with all form data within 3 seconds when custom swap submitted

#### Performance & UX
- [ ] OT-P1-026 | Fast Swap Application | Apply swap and recalculate fitness in <1 second
- [ ] OT-P1-027 | Keyboard Accessibility | Support full keyboard navigation (tab, Enter/Space activation) for all interactive elements
- [ ] OT-P1-028 | Screen Reader Fallback | Provide screen-reader-friendly table view with ARIA labels as dependency graph fallback
- [ ] OT-P1-029 | User-Friendly Error Messages | Display user-friendly errors (no stack traces in UI) with at least one suggested action
- [ ] OT-P1-030 | Collapsible Technical Details | Hide technical details (logs, stack traces) behind "Details" button

#### Integration Enhancements
- [ ] OT-P1-031 | Fast Dependency Data Fetches | Fetch dependency data from scenario-dependency-analyzer API in <3 seconds
- [ ] OT-P1-032 | Stale Data Detection | Offer "Re-scan dependencies" button when dependency data is >7 days old
- [ ] OT-P1-033 | Bidirectional Secret Sync | Sync secret mappings bidirectionally between deployment-manager and secrets-manager
- [ ] OT-P1-034 | Secret Rotation Notifications | Receive webhook from secrets-manager on rotation, mark affected deployments as "needs re-deployment"
- [ ] OT-P1-035 | Deployment Profile Metadata in Issues | Include deployment profile ID as metadata when creating app-issue-tracker issues for linking
- [ ] OT-P1-036 | Issue Closure Webhooks | Receive webhook when linked app-issue-tracker issue closes, update deployment status accordingly

### 🟢 P2 – Future / expansion

#### Enterprise & Compliance Features
- [ ] OT-P2-001 | Licensing Validation | Validate all dependency licenses against tier-specific compatibility rules (e.g., GPL incompatible with proprietary desktop)
- [ ] OT-P2-002 | License Violation Blocking | Block deployment with clear error when license violations detected
- [ ] OT-P2-003 | License Report Generation | Generate license report (markdown/PDF) in <10 seconds for enterprise tier
- [ ] OT-P2-004 | Comprehensive License Documentation | Include all dependencies with license type, version, source URL in report
- [ ] OT-P2-005 | Immutable Audit Logging | Log all deployment actions (create/deploy/update/rollback) with user ID, timestamp, configuration diff
- [ ] OT-P2-006 | Audit Log Export | Export audit logs (JSON/CSV) within 5 seconds for compliance reviews
- [ ] OT-P2-007 | Approval Workflow Support | Support approval workflows (e.g., "Requires 2 approvals before production deploy") for enterprise tier
- [ ] OT-P2-008 | Email/Slack Approval Requests | Send approval requests via email/Slack with clickable approve/reject links

#### CLI & Automation Features
- [ ] OT-P2-009 | CLI Profile Creation from Templates | Support `deployment-manager create --template <path>` to create profiles from YAML templates
- [ ] OT-P2-010 | Headless CLI Deployment | Support `deployment-manager deploy --profile <id> --no-prompt` for CI/CD pipelines
- [ ] OT-P2-011 | Machine-Readable Status Output | Support `deployment-manager status --profile <id> --format json` for CI/CD scripting
- [ ] OT-P2-012 | CI/CD Exit Code Conventions | Follow conventions (0 = success, non-zero = failure) for CI/CD tooling

#### Advanced Monitoring & Analytics
- [ ] OT-P2-013 | Cost Breakdown by Resource | Break down SaaS/cloud cost estimate by resource (EC2, RDS, S3, etc.)
- [ ] OT-P2-014 | Historical Cost Tracking | Track actual deployment costs over time for SaaS/cloud tiers
- [ ] OT-P2-015 | Advanced Metrics Dashboards | Support custom metric dashboards with user-defined queries
- [ ] OT-P2-016 | Deployment Comparison Views | Compare metrics across multiple deployments (A/B testing, staging vs production)

#### Visual Builder & Advanced UX
- [ ] OT-P2-017 | Visual Dependency Graph Editor | Allow drag-and-drop editing of dependency swaps in graph view
- [ ] OT-P2-018 | Deployment Simulation Mode | Simulate deployment to preview changes without executing (dry-run mode)
- [ ] OT-P2-019 | Deployment Templates Library | Provide pre-built deployment templates for common scenarios (SaaS API, mobile app, desktop tool)
- [ ] OT-P2-020 | Collaborative Deployment Approval | Support multi-user collaboration on deployment profiles with comment threads

#### Multi-Tier Deployment Enhancements
- [ ] OT-P2-021 | Tier Recommendation Engine | AI-powered recommendation of optimal tier(s) based on scenario characteristics
- [ ] OT-P2-022 | Cross-Tier Resource Sharing | Coordinate shared backend for mobile/desktop clients deploying with SaaS tier
- [ ] OT-P2-023 | Deployment Cost Comparison | Compare estimated costs across all viable tiers to inform tier selection

#### Advanced Secret Management
- [ ] OT-P2-024 | Secret Auto-Rotation Integration | Automatically trigger re-deployment when secrets rotate (secrets-manager integration)
- [ ] OT-P2-025 | Secret Compliance Scanning | Scan secrets for compliance with industry standards (NIST, SOC2, HIPAA)
- [ ] OT-P2-026 | Just-In-Time Secret Provisioning | Generate secrets on-demand during deployment rather than pre-provisioning

---

## 🧱 Tech Direction Snapshot

### Preferred Stacks
- **Backend API**: Go (proven orchestration performance, matches existing scenarios)
- **Frontend UI**: React + TypeScript + Vite + TailwindCSS + shadcn + Lucide (Vrooli standard)
- **Data Visualization**: React Flow (dependency graphs) + Recharts (monitoring metrics)
- **CLI**: Bash wrapper around API (consistent with other scenarios)
- **Storage**: PostgreSQL (deployment profiles, audit logs, fitness rules)
- **Caching**: Redis (optional, for fitness score caching)

### Storage Expectations
- **Deployment Profiles**: JSON documents with versioning (PostgreSQL JSONB)
- **Fitness Rules**: Pluggable rules engine (Go structs, PostgreSQL storage for custom rules)
- **Audit Logs**: Append-only table (PostgreSQL with timestamps and immutability constraints)
- **Swap Database**: Structured data (dependency → alternatives with metadata)

### Integration Strategy
- **Upstream Integrations** (deployment-manager consumes):
  - `scenario-dependency-analyzer`: REST API for dependency trees
  - `secrets-manager`: REST API + CLI for secret requirements and templates
  - `app-issue-tracker`: REST API for issue creation and webhooks
- **Downstream Integrations** (deployment-manager orchestrates):
  - `scenario-to-desktop`, `scenario-to-ios`, `scenario-to-android`, `scenario-to-saas`, `scenario-to-enterprise`: REST API + CLI for packaging
- **Event System** (optional P2): Pub/sub for deployment lifecycle events (deploy.started, deploy.succeeded, deploy.failed)

### Non-Goals
- ❌ **Not a packager**: deployment-manager does not build apps directly, only orchestrates scenario-to-* packagers
- ❌ **Not a source code modifier**: Dependency swaps affect deployment profiles, never source code (migrations require developer action via app-issue-tracker)
- ❌ **Not a CI/CD replacement**: Integrates with CI/CD (CLI for automation) but doesn't replace GitHub Actions, Jenkins, etc.
- ❌ **Not a monitoring platform**: Basic health checks and metrics, not a replacement for Datadog, Prometheus, etc.

---

## 🤝 Dependencies & Launch Plan

### Required Resources
1. **postgres**: Deployment profile storage, audit logs, fitness rules database
2. **redis** (optional): Caching layer for fitness scores and dependency tree calculations
3. **claude-code or ollama**: AI-powered migration strategy suggestions (agent integration)

### Required Scenarios
1. **scenario-dependency-analyzer**: Dependency tree data source (critical dependency)
2. **secrets-manager**: Secret classification and template generation (critical dependency)
3. **app-issue-tracker**: Migration task creation and tracking (critical dependency)
4. **At least one scenario-to-*** packager (scenario-to-extension, scenario-to-desktop, etc.): Required to validate orchestration

### Operational Risks
- **Risk**: scenario-to-* packagers don't exist for all tiers yet
  - **Mitigation**: Start with Tier 2 (desktop) and Tier 3 (mobile), expand as packagers become available
- **Risk**: Fitness scoring rules may need continuous tuning
  - **Mitigation**: Pluggable rules engine, allow custom rules per scenario
- **Risk**: Dependency swaps may break scenarios unexpectedly
  - **Mitigation**: Swaps only affect profiles (non-destructive), require explicit user approval, create app-issue-tracker migration tasks

### Launch Sequencing
1. **Phase 1 (P0 Core)**: Dependency analysis, fitness scoring, profile management, validation, basic orchestration (Tier 2 desktop focus)
2. **Phase 2 (P1 Monitoring)**: Post-deployment health tracking, updates/rollbacks, multi-tier orchestration
3. **Phase 3 (P1 Integrations)**: Agent migration strategies, custom swap requests, enhanced secret management
4. **Phase 4 (P2 Enterprise)**: Audit logging, compliance features, licensing validation, approval workflows
5. **Phase 5 (P2 Advanced)**: CLI automation, cost tracking, visual builders, deployment templates

---

## 🎨 UX & Branding

### Desired Look and Feel
- **Aesthetic**: Professional, blueprint-like theme (dark mode default, monospace accents for technical data)
- **Inspiration**: Vercel deployment dashboard meets Kubernetes dashboard meets Lighthouse performance reports
- **Visual Language**:
  - Dependency graphs: Clean, hierarchical layouts with color-coded fitness scores (red = blocker, yellow = warning, green = pass)
  - Configuration forms: Shadcn-style professional forms with validation hints
  - Monitoring dashboards: Recharts time-series with sparklines for at-a-glance health

### Accessibility Bar
- **Standard**: WCAG AA compliance minimum
- **Keyboard Navigation**: Full keyboard support (tab navigation, Enter/Space activation, Esc to cancel)
- **Screen Reader Support**: ARIA labels, table view fallback for dependency graph
- **Color Accessibility**: Avoid color-only indicators (use icons + color, e.g., ✓ green, ⚠ yellow, ✗ red)

### Brand Tone
- **Professional and Technical**: Deployment is serious business, UI should inspire confidence
- **Helpful, Not Condescending**: Errors should guide users to solutions, not blame them
- **Transparent**: Show what's happening (log streaming, progress indicators, cost estimates upfront)
- **Empowering**: Users are in control (explicit approval for swaps, clear rollback options)

---

**This PRD is read-only after generation. Operational target checkboxes will sync automatically based on requirement validation.**
