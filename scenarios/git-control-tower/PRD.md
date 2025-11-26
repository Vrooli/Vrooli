# Product Requirements Document (PRD)

> **Template Version**: 2.0.0
> **Last Updated**: 2025-11-18
> **Status**: Production-Ready
> **Canonical Reference**: PRD Control Tower

## 🎯 Overview

Git Control Tower transforms git from a manual CLI tool into a queryable, controllable REST API that enables safe autonomous version control workflows. It provides structured interfaces for inspecting repository state, managing changes granularly, and orchestrating commit workflows with AI assistance.

**Purpose**: Enables AI agents and autonomous systems to perform git operations programmatically without parsing CLI text output.

**Primary Users**:
- AI agents managing code changes
- Automated deployment systems
- Multi-agent development workflows

**Deployment Surfaces**:
- REST API (port 18700)
- CLI tool (`git-control-tower`)
- Web UI dashboard

## 🎯 Operational Targets

### 🔴 P0 – Must ship for viability

- [x] OT-P0-001 | Health check endpoint | Validates database connectivity, git binary availability, repository access with structured JSON response
- [x] OT-P0-002 | Repository status API | Returns current branch, tracking status, staged/unstaged/untracked files, conflicts with scope detection
- [x] OT-P0-003 | File diff endpoint | Returns git diff for any file path relative to HEAD
- [x] OT-P0-004 | Stage/unstage operations | Stage/unstage specific files or by scope (scenario/resource) via POST endpoints
- [x] OT-P0-005 | Commit composition API | Create commits with conventional commit message validation, returns commit hash
- [ ] OT-P0-006 | Push/pull status | Check if push needed with safety checks for behind-remote scenarios
- [x] OT-P0-007 | PostgreSQL audit logging | Logs all mutating operations (stage, unstage, commit) to database with graceful degradation
- [x] OT-P0-008 | CLI command parity | CLI wrapper commands matching all API functionality (status, diff, stage, unstage, commit, health)

### 🟠 P1 – Should have post-launch

- [x] OT-P1-001 | Branch operations | List, create, and switch branches with uncommitted change detection
- [x] OT-P1-002 | AI-assisted commit messages | Generate commit message suggestions using Ollama/OpenRouter with rule-based fallback
- [x] OT-P1-003 | Conflict detection & reporting | Detect active merge conflicts and potential conflicts (local changes + behind remote)
- [x] OT-P1-004 | Change preview | Preview commit impact with LOC analysis, affected scenarios/resources, deployment risk assessment

### 🟢 P2 – Future / expansion

- [x] OT-P2-001 | Web UI dashboard | Visual diff viewer, interactive staging, branch management, service health monitoring
- [ ] OT-P2-002 | Worktree management | List, create, and manage git worktrees via API
- [ ] OT-P2-003 | Stash operations | Save, apply, and drop stashes programmatically

## 🧱 Tech Direction Snapshot

- **API Stack**: Go with net/http, native git CLI commands via exec.Command
- **Data Storage**: PostgreSQL for audit logs (shared Vrooli database), in-memory for git state
- **AI Integration**: Ollama (primary) and OpenRouter (fallback) for commit message generation
- **CLI**: Bash wrapper calling REST API endpoints with formatted output
- **UI**: Single-page application served from API with syntax-highlighted diff viewer

**Integration Strategy**: Direct git CLI integration for speed, future migration to go-git library for production safety

**Non-goals**:
- Remote git operations (push/pull) - tracking info only
- Multi-repository support - single Vrooli repository focus
- Git server functionality - client operations only

## 🤝 Dependencies & Launch Plan

**Required Resources**:
- PostgreSQL - audit logging and metadata storage
- Git binary (>= 2.0) - repository operations

**Optional Resources**:
- Ollama - AI-assisted commit message generation
- OpenRouter - fallback AI for commit messages when Ollama unavailable

**Launch Sequence**:
1. PostgreSQL schema deployed via initialization/postgres/schema.sql
2. API service started via Vrooli lifecycle system
3. Health check validates all dependencies
4. CLI installed to system path via cli/install.sh
5. Web UI accessible at http://localhost:18700

**Risks**:
- Database unavailable → graceful degradation (operations work, no audit logs)
- Git binary missing → fail fast with clear error message
- Performance degradation on large repos → caching and pagination needed

## 🎨 UX & Branding

**Visual Style**:
- Dark terminal-inspired theme
- Monospace fonts for diffs, sans-serif for UI chrome
- GitHub pull request diff viewer aesthetic
- Split-pane layout (file tree + diff viewer)

**Personality & Tone**:
- Technical and focused
- Clear status indicators
- Precise control messaging
- No fluff, direct feedback

**Accessibility**:
- WCAG AA compliance for color contrast in diffs
- Keyboard navigation support
- Screen reader friendly status updates

**Target Feeling**: "I have precise, programmatic control over my repository"

## 📎 Appendix

### Performance Targets
- Status API: < 500ms for 95% of requests
- Diff generation: < 200ms for files < 10KB
- Commit creation: < 1s including validation
- Database queries: < 50ms for audit logs

### Future Capabilities Enabled
1. Autonomous deployment pipelines - scenarios stage changes and trigger deployments
2. Multi-agent coordination - check for conflicts before making changes
3. Intelligent code review - fetch and analyze diffs programmatically
4. Change analytics - analyze commit patterns to understand codebase evolution

### Related Documentation
- README.md - user-facing quick start
- initialization/postgres/schema.sql - audit log schema
- .vrooli/service.json - lifecycle configuration
