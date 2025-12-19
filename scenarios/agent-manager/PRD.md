# Product Requirements Document (PRD)

## 🎯 Overview
Purpose: Central orchestration and governance layer for running AI agents against codebases in a controlled, reviewable, and extensible way. Creates reusable patterns for safe agent execution, policy enforcement, event tracking, and approval workflows.

Target users:
- Developers requiring safe multi-agent codebase operations
- Vrooli ecosystem scenarios (agent-inbox, app-issue-tracker, ecosystem-manager)
- CI/CD pipeline operators

Deployment surfaces: API, CLI, Web UI

Value proposition: Provides standardized, policy-driven agent execution management with built-in safety controls, event tracking, and approval workflows that other scenarios can leverage without reimplementing orchestration logic.

## 🎯 Operational Targets

### 🔴 P0 – Must ship for viability
- [ ] OT-P0-001 | AgentProfile Management | Create and manage agent profiles with runner configurations and permissions
- [ ] OT-P0-002 | Task Management | Implement CRUD operations for tasks with status tracking
- [ ] OT-P0-003 | Run Creation | Enable run creation with sandbox allocation and mode selection
- [ ] OT-P0-004 | Run Status Tracking | Track and update run status with timestamps
- [ ] OT-P0-005 | Sandbox Integration | Integrate with workspace-sandbox for isolated execution
- [ ] OT-P0-006 | Runner Adapter Interface | Define and implement runner adapter interface
- [ ] OT-P0-007 | Runner Execution | Execute agents via adapters with profile configurations
- [ ] OT-P0-008 | Event Streaming | Implement append-only event stream for run activities
- [ ] OT-P0-009 | Diff Collection | Generate and store diffs from sandbox changes
- [ ] OT-P0-010 | Basic Approval Flow | Support approve/reject workflows for runs
- [ ] OT-P0-011 | Health Check API | Provide health monitoring endpoint

### 🟠 P1 – Should have post-launch
- [ ] OT-P1-001 | Scope Locks | Implement path-scoped exclusive locks
- [ ] OT-P1-002 | Resource Limits | Add concurrent run limits and capacity tracking
- [ ] OT-P1-003 | Partial Approval | Enable selective file/hunk approval
- [ ] OT-P1-004 | Policy Engine | Build configurable policy system
- [ ] OT-P1-005 | Default Policies | Store default policies in agent profiles
- [ ] OT-P1-006 | Runner Capabilities | Query and track runner feature support
- [ ] OT-P1-007 | Real-time Events | Add WebSocket streaming for run events
- [ ] OT-P1-008 | Run Artifacts | Store and manage run-related artifacts
- [ ] OT-P1-009 | CLI Interface | Develop comprehensive command-line interface

### 🟢 P2 – Future / expansion
- [ ] OT-P2-001 | Multi-phase Runs | Support sequential agent prompts
- [ ] OT-P2-002 | Cost Tracking | Track token usage and API costs
- [ ] OT-P2-003 | Runner Metrics | Collect detailed runner performance metrics
- [ ] OT-P2-004 | Policy Templates | Create pre-built policy configurations
- [ ] OT-P2-005 | Webhook Notifications | Add external system notifications
- [ ] OT-P2-006 | Batch Operations | Enable multi-task batch management
- [ ] OT-P2-007 | UI Dashboard | Build web interface for management
- [ ] OT-P2-008 | Run Analytics | Implement historical run data analysis

## 🧱 Tech Direction Snapshot
Preferred stacks:
- Backend: Go with WebSocket support
- Frontend: React/Vite with TailwindCSS
- CLI: Go with Cobra/Viper

Preferred storage:
- PostgreSQL for structured data
- Append-only event logs for audit trails

Integration strategy:
- Interface-based runner adapters
- workspace-sandbox for isolation
- WebSocket for real-time events

Non-goals:
- Building new runners
- Providing hardened security boundary
- Replacing git branching workflows

## 🤝 Dependencies & Launch Plan
Required resources:
- PostgreSQL database
- workspace-sandbox scenario
- Runner resources (claude-code, codex, opencode)

Scenario dependencies:
- workspace-sandbox for isolation and diff management

Operational risks:
- Sandbox API availability
- Runner process stability
- Event log growth

Launch sequencing:
1. Alpha: P0 features with claude-code
2. Beta: P1 features with all runners
3. GA: After UI and stability validation

## 🎨 UX & Branding
User experience:
- CLI-first interface with JSON output
- WebSocket streaming for real-time updates
- REST API for all operations
- Clear status messaging

Visual design:
- CLI with clear status indicators
- Dashboard following Vrooli patterns
- Emphasis on safety and control

Accessibility:
- WCAG 2.1 AA compliance for UI
- CLI color schemes for color blindness
- Clear status messages for automation