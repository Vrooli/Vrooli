# Product Requirements Document (PRD)

## 🎯 Overview
Purpose: Create fast, storage-efficient, copy-on-write workspace sandboxes for safely running agents and tools against project folders without risking canonical repo modification.

Target users:
- AI Agents running code modification tasks
- Developers running experimental changes
- CI/CD pipelines needing isolated test environments
- Vrooli ecosystem scenarios requiring safe execution contexts

Deployment surfaces:
- Local Vrooli server (Tier 1)
- CLI tool for direct usage
- API for programmatic integration
- Web UI for diff review and approval

Value proposition:
- Enable scalable agent parallelism with isolated workspace views
- Replace long-lived worktrees/duplicate checkouts with ephemeral, cheap sandboxes
- Disk cost proportional only to files actually changed (not full repo)
- Clean accept/reject semantics for agent-generated changes

## 🎯 Operational Targets

Operational targets are measurable outcomes; checkboxes may auto-update based on validation.

### 🔴 P0 – Must ship for viability
- [x] OT-P0-001 | Sandbox Create/Mount Operations | Create + mount a sandbox in ~1-2 seconds (best effort) with stable ID, scopePath, and status tracking
- [x] OT-P0-002 | Overlayfs Copy-on-Write Driver | Linux driver using overlayfs (lowerdir=canonical read-only, upperdir/workdir per sandbox, merged mount point)
- [ ] OT-P0-003 | Bubblewrap Process Isolation | bwrap to run agent/tool processes in constrained filesystem view with canonical repo mounted read-only
- [x] OT-P0-004 | Path-Scoped Mutual Exclusion | Forbid creating sandbox for path P if any active sandbox has scope S where S is ancestor of P or P is ancestor of S
- [x] OT-P0-005 | Diff Generation | Generate unified diff and file list of all changes in sandbox's upper layer with stable ordering
- [x] OT-P0-006 | Patch Application | Apply approved diffs to canonical repo via controlled, trusted process
- [x] OT-P0-007 | Sandbox Lifecycle Management | Create, list, inspect, stop, delete sandboxes with idempotent cleanup and no leaked mounts
- [ ] OT-P0-008 | Process/Session Tracking | Track PIDs/process groups spawned under sandbox, clean up on stop/delete
- [x] OT-P0-009 | Health Check API | Basic health endpoint for lifecycle monitoring

### 🟠 P1 – Should have post-launch
- [ ] OT-P1-001 | Hunk-Level Approval | UI and API support for selecting individual hunks/sections to approve, not just full files
- [ ] OT-P1-002 | Partial Approval Workflow | Apply only selected diffs while preserving remaining sandbox state for follow-up runs
- [ ] OT-P1-003 | GC/Prune Operations | Garbage collection by age, max-total-size, inactive TTL policies
- [ ] OT-P1-004 | Audit Trail Metadata | Immutable metadata per sandbox: id, scope, creator/owner, timestamps, driver, status, size, changed files
- [ ] OT-P1-005 | Pre-commit Validation Hooks | Configurable hooks to run tests/lint before final commit of approved changes
- [ ] OT-P1-006 | Commit Attribution Policy | Configurable commit message templates and author attribution for applied patches
- [ ] OT-P1-007 | Safe-Git Wrapper | PATH-injected wrapper that explains correct workflow for blocked git commands during agent runs
- [ ] OT-P1-008 | Metrics/Observability | Active sandbox count, total disk usage, per-sandbox size distribution, create latency, cleanup success rates
- [ ] OT-P1-009 | CLI Interface | Full-featured CLI for create, list, inspect, diff, approve, reject, delete, gc operations

### 🟢 P2 – Future / expansion
- [ ] OT-P2-001 | Diff Viewer UI | Web UI with file-level toggle approval, hunk selection, sandbox metadata display, one-click approve/reject
- [ ] OT-P2-002 | Conflict Detection | Detect when canonical repo has changed since sandbox creation, surface patch application failures clearly
- [ ] OT-P2-003 | Retry/Rebase Workflow | Support re-running sandbox or regenerating diff when conflicts occur
- [ ] OT-P2-004 | Cross-Platform Driver Interface | Pluggable SandboxDriver interface with clear fallback strategy for non-Linux environments
- [ ] OT-P2-005 | Size Accounting | Real-time tracking of sandbox disk usage with configurable limits per-sandbox and total
- [ ] OT-P2-006 | Policy Guardrails | Configurable max lifetime/TTL, max size, require explicit human approval, allow sandbox reuse
- [ ] OT-P2-007 | Structured Event Logging | Emit structured logs for create/mount/run/diff/approve/reject/delete/gc with failure details
- [ ] OT-P2-008 | Agent-Manager Integration | Integration with agent-manager for trusted patch application

## 🧱 Tech Direction Snapshot
Preferred stacks:
- API: Go with standard library HTTP or Fiber
- CLI: Go with Cobra/Viper
- UI: React + Vite + TypeScript with diff viewer component

Preferred storage:
- PostgreSQL for sandbox metadata
- Filesystem for overlay layers

Integration strategy:
- RESTful API first, CLI wrapper, agent-manager integration
- Expose operations via API and CLI: create, list, inspect, start/stop sessions, diff, approve/reject, delete, gc

Non-goals:
- Security hardening against adversarial processes
- Replacement for proper git branching for collaborative work
- Very long-lived sandbox persistence

## 🤝 Dependencies & Launch Plan
Required resources:
- PostgreSQL for metadata storage
- Linux with overlayfs support
- bubblewrap package installed

Scenario dependencies:
- None required (standalone capability)
- Can integrate with: ecosystem-manager, agent-inbox, test-genie

Operational risks:
- Mount leaks if cleanup fails
- Build artifacts/caches growing sandbox size
- Background processes persisting after stop

Launch sequencing:
- Alpha: P0 features completion
- Beta: P1 features and CLI
- GA: After UI completion and stability proven

## 🎨 UX & Branding
User experience:
- Developer-focused interface with clear sandbox status indicators
- Diff viewer following GitHub/GitLab conventions
- Clear visual distinction between approved/rejected/pending hunks
- Status indicators for sandbox health and size

Visual design:
- Clean, technical aesthetic
- High contrast mode support
- Responsive layout for various screen sizes

Accessibility:
- WCAG 2.1 AA compliance
- Keyboard navigation for diff review
- Screen reader support for critical actions