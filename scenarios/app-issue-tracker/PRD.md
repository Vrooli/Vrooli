# Product Requirements Document: App Issue Tracker

## Executive Summary

**What**: File-based issue tracking system with AI-powered investigation and automated fix generation
**Why**: Eliminate database overhead while enabling intelligent issue resolution
**Who**: Development teams, AI agents, and small organizations avoiding enterprise tools
**Value**: $15K-30K per deployment through reduced resolution time and automated fixes
**Priority**: Critical infrastructure for Vrooli ecosystem debugging

## 🎯 Capability Definition

- Holistic issue lifecycle management spanning creation, assignment, investigation, resolution, and archival
- Hybrid human + AI workflow where unified-resolver automates diagnosis and remediation proposals
- File-native storage pattern (YAML on disk) so every issue is version-controlled, portable, and auditable
- Multi-surface access: REST API, responsive React UI, and CLI automation hooks for agent orchestrations

## 💰 Value Proposition

- $15K–30K per deployment via accelerated debugging and reduced SaaS licensing
- 2× faster MTTR by coupling automated investigations with actionable remediation plans
- Zero-database architecture lowers operating costs and simplifies customer onboarding
- Reusable templates and exports turn postmortems into monetizable knowledge products

## Progress Tracking

**Overall Completion**: 97%
**Last Updated**: 2025-10-01

### History
- 2025-09-30: 0% → 45% (Fixed directory structure, verified API health, tested issue creation)
- 2025-09-30: 45% → 85% (Verified fix generation via investigation workflow, fixed CLI references, all P0 requirements working)
- 2025-09-30: 85% → 87% (Verified Web UI is functional and displaying issues/dashboard)
- 2025-10-01: 87% → 90% (Fixed Go compilation error in git_integration.go, verified all P0 requirements, ran auditor scan - 1 security issue, 495 standards violations)
- 2025-10-01: 90% → 93% (Wired git integration to API endpoint, verified export fully working in JSON/CSV/Markdown, all tests passing)
- 2025-10-01: 93% → 97% (Implemented configurable CORS security, real-time performance analytics, comprehensive integration tests, complete security documentation)

## 2. Target Users
- Software developers tracking bugs and features
- Team leads managing project issues
- AI agents automating investigation workflows
- Small teams avoiding heavy tools like Jira

## 3. Goals and Objectives
### Business Goals
- Reduce issue tracking overhead by 80%
- Enable AI-assisted resolution for 50% of issues
- Provide semantic search across all historical issues

### Technical Goals
- 100% file-based storage with YAML schema
- REST API for programmatic access
- Web UI for visual management
- CLI for terminal operations

## 4. User Stories

### Core Stories
- As a developer, I want to create a new issue from a template so that I can standardize reporting
- As a team member, I want to assign issues to myself or others so that responsibilities are clear
- As a lead, I want to search issues by keyword or semantic similarity so that I can find related problems
- As an AI agent, I want to investigate an issue automatically so that I can generate fix suggestions

### Advanced Stories
- As a maintainer, I want to close issues with resolution notes so that knowledge is preserved
- As an analyst, I want to generate reports on issue trends so that I can identify patterns
- As a collaborator, I want to integrate with GitHub so that issues sync with PRs

## Requirements Checklist

### P0 Requirements (Must Have - Core Functionality)
- [x] **Health Check Endpoint**: API responds with service status (VERIFIED: 2025-10-01 - API on port 19751, returns JSON with version and storage info)
- [x] **Issue Creation**: Create issues via API with metadata (VERIFIED: 2025-10-01 - CLI and API both working, issues persist to YAML)
- [x] **Issue Listing**: List and filter issues via API (VERIFIED: 2025-10-01 - 20+ issues in system, CLI formatting works)
- [x] **File-Based Storage**: YAML files in folder structure (VERIFIED: 2025-10-01 - using data/issues/* structure)
- [x] **Investigation Trigger**: Start AI investigation via API (VERIFIED: 2025-10-01 - unified-resolver agent starts, background execution)
- [x] **Fix Generation**: Generate and apply fixes automatically (VERIFIED: 2025-10-01 - integrated with investigation workflow using auto_resolve flag)
- [x] **CLI Operations**: Complete CLI for all issue operations (VERIFIED: 2025-10-01 - create, list, investigate commands all working, API-backed)

### P1 Requirements (Should Have - Enhanced Features)
- [ ] **Semantic Search**: Vector-based similar issue detection (Blocked: Qdrant not available, feature disabled)
- [x] **Web UI**: Interactive dashboard for issue management (VERIFIED: 2025-10-01 - Running on port 36221, React/Vite dev server)
- [x] **Git Integration**: Create PRs with fixes (VERIFIED: 2025-10-01 - API endpoint wired at POST /api/v1/issues/{id}/create-pr, requires GITHUB_TOKEN/OWNER/REPO env vars)
- [x] **Export Functions**: Generate reports in CSV/Markdown (VERIFIED: 2025-10-01 - All formats working: JSON, CSV, Markdown via /api/v1/export endpoint)
- [x] **Security Configuration**: Production-ready CORS, authentication, and rate limiting (VERIFIED: 2025-10-01 - Configurable via environment variables, comprehensive docs/SECURITY_SETUP.md)
- [x] **Performance Analytics**: Real-time resolution metrics calculation (VERIFIED: 2025-10-01 - avg_resolution_hours calculated from actual issue timestamps)
- [x] **Integration Testing**: Comprehensive API validation test suite (VERIFIED: 2025-10-01 - test-integration.sh covers health, stats, CORS, export, git endpoints)

### P2 Requirements (Nice to Have - Advanced Features)
- [ ] **Advanced Performance Dashboards**: Historical trend analysis and forecasting
- [ ] **Multi-Agent Support**: Different AI agents for different issue types
- [ ] **Webhook Integration**: Notify external systems on issue state changes

## 🧬 Evolution Path
- Phase 1: stabilize file-based backend and API (complete)
- Phase 2: harden semantic search + Redis caching for large installations
- Phase 3: introduce multi-agent orchestration with specialization per issue taxonomy
- Phase 4: marketplace packaging so external teams can deploy via Vrooli scenario catalog

## 6. Non-Functional Requirements

### Performance
- Handle 1000+ issues with <500ms search times
- File operations optimized for frequent updates
- Caching layer for repeated queries (Redis optional)

### Security
- File permissions management
- Input sanitization for descriptions and titles
- Audit log for issue changes
- No sensitive data in YAML files

### Reliability
- 99.9% uptime for API/UI
- Graceful degradation without vector search
- Automated backups of issue files
- Recovery from corrupted YAML files

### Usability
- Intuitive CLI with help and examples
- Clean, responsive web UI
- Keyboard shortcuts in UI
- Export issues to CSV/Markdown

### Compatibility
- Linux/macOS/Windows (via WSL)
- Go 1.21+, Node.js 18+
- Git integration for version control
- Docker support for containerized deployment

## 🏗️ Technical Architecture

### Components
- **API Server**: Go-based REST API handling file I/O and business logic
- **Web UI**: Node.js server with static file serving and optional SSR
- **CLI Tool**: Bash script wrapping API calls and direct file operations
- **AI Investigator**: Script integrating with Codex API
- **Search Engine**: Optional Qdrant for semantic search, fallback to keyword

### Data Flow
1. User creates issue via CLI/UI/API
2. System validates and saves YAML to appropriate status directory
3. Search queries scan files or query vector store
4. Investigation triggers AI analysis on issue content
5. Results stored as comments or separate investigation files

### Deployment
- Local development: `make start`
- Production: Docker containers or direct binary execution
- Resources: Optional Redis/Qdrant for enhanced features

## 📝 Implementation Notes
- Go modules pinned via `go.mod`; use `gofmt`/`gofumpt` for formatting and `golangci-lint` for static analysis
- UI relies on Vite dev server; production build uses `npm run build` with environment-injected API base URL
- Scenario lifecycle caches transcripts under `tmp/codex`; cleanup runs post-execution to avoid disk bloat
- Automated tests executed through `make test` leverage scenario harness and isolated temp directories

## 🖥️ CLI Interface Contract

- `app-issue-tracker create --title <text> --priority <level>` → scaffolds YAML issue with metadata prompts
- `app-issue-tracker list [--status open|active|completed|failed]` → renders tabular view sourced from file store
- `app-issue-tracker investigate --issue <id> [--agent unified-resolver] [--auto-resolve]` → dispatches investigation job
- `app-issue-tracker export --format json|csv|markdown` → streams formatted report via `/api/v1/export`
- `./issues/manage.sh <action>` → maintenance utilities (archive, search, reindex) used by scenario lifecycle scripts

All commands honor `API_PORT`, `ISSUES_DIR`, and agent environment variables, failing fast when configuration is missing to protect automation correctness.

## 🎨 Style and Branding Requirements
- UI follows Vrooli dark theme primitives with accent colors driven by priority state (critical=red, high=orange, etc.)
- Typography: Inter for headings, Roboto Mono for code/IDs, minimum 14px body text for accessibility
- Components expose `data-testid` and ARIA attributes to support automated QA and screen readers
- Snackbar and toast notifications adhere to 4-second auto-dismiss with manual close button for compliance

## 📊 Success Metrics
- 90% user satisfaction with issue creation speed
- 70% issues resolved with AI assistance
- <5% error rate in file operations
- 100% test coverage for core functionality

## ✅ Validation Criteria
- API `GET /health` returns `200` with `storage` field populated
- `app-issue-tracker create` followed by `list` shows new issue in YAML store
- Automated investigation writes transcript + last message artifacts and transitions issue to `completed`
- `scenario-auditor scan app-issue-tracker --wait` reports zero high severity lifecycle violations after fixes

## 🔄 Integration Requirements
- **GitHub**: optional PR automation requires `GITHUB_TOKEN`, `GITHUB_OWNER`, `GITHUB_REPO`; failures must surface descriptive HTTP 5xx responses
- **Qdrant**: semantic search hooks enabled when `QDRANT_URL` resolves; scenario must degrade gracefully to keyword mode if unavailable
- **Redis**: caching layer toggled via lifecycle resources; absence must not break base workflows
- **Scenario Auditor**: lifecycle scripts ensure API + UI expose `/health` endpoints so cross-scenario monitors can probe status
- **Agent Infrastructure**: depends on `resource-claude-code` CLI; agent settings reloadable via `/api/v1/agent/settings`

## 🔄 Scenario Lifecycle Integration
- `make start` → orchestrates API + UI via lifecycle, exporting URLs to console for operator visibility
- `make test` → routes through shared scenario test harness ensuring API, CLI, and file structure remain healthy
- `make stop` → gracefully terminates Go API and Vite UI processes, releasing ports back to orchestrator
- Lifecycle health checks feed ecosystem-manager dashboards through `/health` endpoints and scenario metadata

## 🚨 Risk Mitigation
- **Risk**: YAML parsing errors – **Mitigation**: Schema validation + error recovery
- **Risk**: Performance with large issue volumes – **Mitigation**: Indexing + caching
- **Risk**: AI investigation inaccuracies – **Mitigation**: Human review workflow + feedback loop
- **Risk**: File conflicts in teams – **Mitigation**: Git integration + locking mechanism

## 10. Timeline
- MVP (CLI + basic API): 2 weeks
- UI Integration: 1 week
- AI Features: 2 weeks
- Testing & Polish: 1 week
- Total: 6 weeks

## 🔗 References
- docs/SECURITY_SETUP.md – hardening checklist and env var matrix
- docs/context.md – platform overview and recursive intelligence vision
- scenarios/app-issue-tracker/api/README.md – API endpoints and testing instructions
- scenarios/app-issue-tracker/ui/README.md – UI development workflow and tooling
