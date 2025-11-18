# Product Requirements Document (PRD)

> **Version**: 2.0.0
> **Last Updated**: 2025-11-18
> **Status**: Active
> **Canonical Reference**: PRD Control Tower

## 🎯 Overview

**Purpose**: File-based issue tracking system with AI-powered investigation and automated fix generation, eliminating database overhead while enabling intelligent issue resolution.

**Primary users**: Development teams, AI agents, small organizations avoiding enterprise tools like Jira

**Deployment surfaces**: REST API, responsive React UI, CLI automation hooks, agent orchestrations

**Value**: $15K-30K per deployment through reduced resolution time (2× faster MTTR), zero-database architecture, and automated remediation proposals

**Key capabilities**:
- Holistic issue lifecycle management (creation, assignment, investigation, resolution, archival)
- Hybrid human + AI workflow with automated diagnosis via unified-resolver agent
- File-native YAML storage pattern for version-controlled, portable, auditable issues
- Multi-surface access across API, UI, and CLI

## 🎯 Operational Targets

### 🔴 P0 – Must ship for viability

- [x] OT-P0-001 | Health Check Endpoint | API responds with service status on /health with version and storage info
- [x] OT-P0-002 | Issue Creation | Create issues via API/CLI with metadata, persist to YAML files
- [x] OT-P0-003 | Issue Listing | List and filter issues via API with status filtering
- [x] OT-P0-004 | File-Based Storage | YAML files in data/issues/* folder structure with validation
- [x] OT-P0-005 | Investigation Trigger | Start AI investigation via API with background execution
- [x] OT-P0-006 | Fix Generation | Generate and apply fixes automatically via auto_resolve flag
- [x] OT-P0-007 | CLI Operations | Complete CLI for create, list, investigate commands with API backing

### 🟠 P1 – Should have post-launch

- [ ] OT-P1-001 | Semantic Search | Vector-based similar issue detection via Qdrant integration
- [x] OT-P1-002 | Web UI | Interactive dashboard for issue management on Vite dev server
- [x] OT-P1-003 | Git Integration | Create PRs with fixes via POST /api/v1/issues/{id}/create-pr
- [x] OT-P1-004 | Export Functions | Generate reports in JSON/CSV/Markdown formats
- [x] OT-P1-005 | Security Configuration | Production-ready CORS, authentication, rate limiting via env vars
- [x] OT-P1-006 | Performance Analytics | Real-time resolution metrics (avg_resolution_hours) from timestamps
- [x] OT-P1-007 | Integration Testing | Comprehensive API validation suite covering health, stats, CORS, export

### 🟢 P2 – Future / expansion

- [ ] OT-P2-001 | Advanced Performance Dashboards | Historical trend analysis and forecasting
- [ ] OT-P2-002 | Multi-Agent Support | Different AI agents specialized for different issue types
- [ ] OT-P2-003 | Webhook Integration | Notify external systems on issue state changes

## 🧱 Tech Direction Snapshot

**Backend**: Go 1.21+ REST API with file I/O and business logic, gofumpt formatting, golangci-lint static analysis

**Frontend**: Node.js 18+ with Vite dev server, React UI, production builds with environment-injected API base URL

**CLI**: Bash script wrapping API calls and direct file operations, honors API_PORT and ISSUES_DIR env vars

**Storage**: 100% file-based YAML schema in data/issues/* directories (active, completed, failed)

**AI Integration**: Codex API integration for automated investigation, transcript caching under tmp/codex

**Search**: Optional Qdrant for semantic search, graceful degradation to keyword mode when unavailable

**Caching**: Optional Redis for large installations, base workflows operate without it

**Non-goals**:
- Enterprise database requirements
- Real-time collaboration features
- Mobile native applications
- Built-in authentication/authorization (delegated to deployment layer)

## 🤝 Dependencies & Launch Plan

**Required Resources**:
- resource-claude-code: CLI for agent infrastructure and settings reloading
- Git: Version control integration for PR automation

**Optional Resources**:
- Qdrant: Semantic search capabilities (graceful degradation if absent)
- Redis: Caching layer for performance with large issue volumes
- GitHub: PR automation requires GITHUB_TOKEN, GITHUB_OWNER, GITHUB_REPO env vars

**Scenario Dependencies**:
- scenario-auditor: Lifecycle validation and health check monitoring
- ecosystem-manager: Dashboard integration via /health endpoints

**Risks**:
- YAML parsing errors (mitigation: schema validation + error recovery)
- Performance with 1000+ issues (mitigation: indexing + optional Redis caching)
- AI investigation inaccuracies (mitigation: human review workflow + feedback loop)
- File conflicts in team environments (mitigation: Git integration + locking mechanism)

**Launch Sequencing**:
1. Stabilize file-based backend and API (complete)
2. Harden semantic search + Redis caching for large installations
3. Introduce multi-agent orchestration with specialization per issue taxonomy
4. Marketplace packaging for external deployment via Vrooli scenario catalog

## 🎨 UX & Branding

**Visual Design**: Vrooli dark theme primitives with priority-based accent colors (critical=red, high=orange, medium=yellow, low=green)

**Typography**: Inter for headings, Roboto Mono for code/IDs, minimum 14px body text for accessibility

**Accessibility**: WCAG 2.1 Level AA compliance, data-testid and ARIA attributes for automated QA and screen readers

**Notifications**: Snackbar and toast with 4-second auto-dismiss and manual close button

**Voice/Personality**: Direct, technical, efficiency-focused - minimize ceremony, maximize clarity

**CLI Experience**: Intuitive help text, tabular output formatting, fast fail with descriptive errors

**UI Experience**: Responsive layout, keyboard shortcuts, clean dashboard, real-time status updates

## 📎 Appendix

### Success Metrics
- 90% user satisfaction with issue creation speed
- 70% issues resolved with AI assistance
- <5% error rate in file operations
- 100% test coverage for core functionality
- <500ms search times for 1000+ issues

### Validation Criteria
- API GET /health returns 200 with storage field populated
- app-issue-tracker create followed by list shows new issue in YAML store
- Automated investigation writes transcript + artifacts and transitions issue to completed
- scenario-auditor scan reports zero high severity lifecycle violations

### CLI Interface Contract
- `app-issue-tracker create --title <text> --priority <level>` → scaffolds YAML issue
- `app-issue-tracker list [--status open|active|completed|failed]` → renders tabular view
- `app-issue-tracker investigate --issue <id> [--agent unified-resolver] [--auto-resolve]` → dispatches investigation
- `app-issue-tracker export --format json|csv|markdown` → streams formatted report
- `./issues/manage.sh <action>` → maintenance utilities (archive, search, reindex)

### Integration Requirements
- GitHub: Optional PR automation, failures surface HTTP 5xx with descriptive messages
- Qdrant: Semantic search when QDRANT_URL resolves, keyword fallback otherwise
- Redis: Toggled via lifecycle resources, absence must not break base workflows
- Scenario Auditor: /health endpoints for cross-scenario monitoring
- Agent Infrastructure: Depends on resource-claude-code CLI, settings reloadable via /api/v1/agent/settings

### Scenario Lifecycle Integration
- `make start` → orchestrates API + UI via lifecycle, exports URLs to console
- `make test` → routes through scenario test harness for API, CLI, file structure validation
- `make stop` → gracefully terminates processes and releases ports
- Lifecycle health checks feed ecosystem-manager dashboards

### References
- docs/SECURITY_SETUP.md – hardening checklist and env var matrix
- docs/context.md – platform overview and recursive intelligence vision
- scenarios/app-issue-tracker/api/README.md – API endpoints and testing instructions
- scenarios/app-issue-tracker/ui/README.md – UI development workflow and tooling
