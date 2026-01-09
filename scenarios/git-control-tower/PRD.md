# Product Requirements Document (PRD)

> **Template Version**: 2.0.0  
> **Last Updated**: 2025-12-16  
> **Status**: Initialized (scaffold)  
> **Canonical Reference**: PRD Control Tower

## 🎯 Overview
- **Purpose**: Turn git from a human-only CLI into a structured control plane (API + CLI + UI) so agents and automation can perform version-control workflows safely and programmatically.
- **Primary users/verticals**: AI agents managing code changes, automated deployment/maintenance systems, multi-agent dev workflows.
- **Deployment surfaces**: REST API, CLI tool, Web UI dashboard.
- **Value promise**: Agents operate on repository state via typed JSON and explicit actions (no brittle CLI text parsing), with an auditable trail for mutating operations.

## 🎯 Operational Targets

### 🔴 P0 – Must ship for viability
- [ ] OT-P0-001 | Health check endpoint | Validates database connectivity, git binary availability, repository access with structured JSON response
- [ ] OT-P0-002 | Repository status API | Returns current branch, tracking status, staged/unstaged/untracked files, conflicts with scope detection
- [ ] OT-P0-003 | File diff endpoint | Returns git diff for any file path relative to HEAD
- [ ] OT-P0-004 | Stage/unstage operations | Stage/unstage specific files or by scope (scenario/resource) via POST endpoints
- [ ] OT-P0-005 | Commit composition API | Create commits with conventional commit message validation, returns commit hash
- [ ] OT-P0-006 | Push/pull status | Check if push needed with safety checks for behind-remote scenarios
- [ ] OT-P0-007 | PostgreSQL audit logging | Logs all mutating operations (stage, unstage, commit) to database with graceful degradation
- [ ] OT-P0-008 | CLI command parity | CLI wrapper commands matching all API functionality (status, diff, stage, unstage, commit, health)

### 🟠 P1 – Should have post-launch
- [ ] OT-P1-001 | Branch operations | List, create, and switch branches with uncommitted change detection
- [ ] OT-P1-002 | AI-assisted commit messages | Generate commit message suggestions using Ollama/OpenRouter with rule-based fallback
- [ ] OT-P1-003 | Conflict detection & reporting | Detect active merge conflicts and potential conflicts (local changes + behind remote)
- [ ] OT-P1-004 | Change preview | Preview commit impact with LOC analysis, affected scenarios/resources, deployment risk assessment

### 🟢 P2 – Future / expansion
- [ ] OT-P2-001 | Web UI dashboard | Visual diff viewer, interactive staging, branch management, service health monitoring
- [ ] OT-P2-002 | Worktree management | List, create, and manage git worktrees via API
- [ ] OT-P2-003 | Stash operations | Save, apply, and drop stashes programmatically

## 🧱 Tech Direction Snapshot
- Preferred stacks / frameworks: Go for API, React + Vite + TypeScript for UI, CLI wrapper for agent ergonomics.
- Data + storage expectations: PostgreSQL for audit logs/metadata; optional Redis for caching.
- Integration strategy: Operate on the local Vrooli repository working tree (single-repo focus) and expose safe, structured endpoints for agent consumption.
- Non-goals / guardrails: Not a git server; no multi-repo support initially; no remote push/pull by default (status-only unless explicitly expanded); avoid requiring agents to parse raw CLI output.

## 🤝 Dependencies & Launch Plan
- Required resources: PostgreSQL (audit logs) and a git binary available in `PATH`.
- Optional resources: Ollama + OpenRouter (AI commit message suggestions), Redis (diff/status caching).
- Operational risks: Large repos require pagination/caching; missing database must degrade gracefully; missing git binary should fail fast with clear health errors.
- Launch sequencing: Build API + UI, start services via lifecycle, validate `/health`, then iterate operational targets (P0 → P1 → P2) with requirements-linked tests.

## 🎨 UX & Branding
- Look & feel: Dark, terminal-inspired theme with a GitHub PR diff viewer vibe (split-pane file list + diff viewer).
- Accessibility: WCAG AA for contrast and keyboard navigation; readable monospace for diffs.
- Voice & messaging: Technical and direct; explicit warnings for risky operations; no fluff.
- Branding hooks: Git semantics (branch, commit, diff) as primary iconography.

## 📎 Appendix
- `docs/RESEARCH.md` (uniqueness + related scenarios/resources)
