# PRD: App Issue Tracker

## Executive Summary

**What**: File-based issue tracking system with AI-powered investigation and automated fix generation
**Why**: Eliminate database overhead while enabling intelligent issue resolution
**Who**: Development teams, AI agents, and small organizations avoiding enterprise tools
**Value**: $15K-30K per deployment through reduced resolution time and automated fixes
**Priority**: Critical infrastructure for Vrooli ecosystem debugging

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

## 7. Technical Architecture

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

## 8. Success Metrics
- 90% user satisfaction with issue creation speed
- 70% issues resolved with AI assistance
- <5% error rate in file operations
- 100% test coverage for core functionality

## 9. Risks and Mitigations
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
