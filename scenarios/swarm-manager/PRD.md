# Product Requirements Document (PRD)

> **Template Version**: 2.0
> **Canonical Reference**: `/scenarios/prd-control-tower/docs/CANONICAL_PRD_TEMPLATE.md`
> **Validation**: Enforced by `prd-control-tower` + `scenario-auditor`
> **Policy**: Generated once and treated as read-only (checkboxes may auto-update)

## 🎯 Overview
- **Purpose**: The autonomous orchestration engine that coordinates work across all Vrooli scenarios, enabling true self-improvement without human intervention by intelligently prioritizing, dispatching, and learning from task execution outcomes
- **Primary users/verticals**: System administrators, DevOps teams, autonomous AI agents, Vrooli ecosystem developers
- **Deployment surfaces**: CLI (task management and monitoring), API (task orchestration and status), UI (Trello-like dashboard with real-time monitoring), n8n workflows (automated task discovery and execution)
- **Value promise**: Transforms Vrooli from a collection of tools into a self-improving intelligence system that never stops optimizing itself, achieving >50 improvements per day with >80% success rate and zero human intervention for 7+ days

## 🎯 Operational Targets

### 🔴 P0 – Must ship for viability
- [ ] OT-P0-001 | File-based task lifecycle | Create, stage, execute, complete, and fail tasks using file-based storage with proper state transitions
- [ ] OT-P0-002 | Intelligent priority calculation | Calculate task priority using (Impact × Urgency × Success_Probability) / (Resource_Cost × Cooldown_Factor) formula
- [ ] OT-P0-003 | CLI-based scenario integration | Dispatch work to scenarios via CLI commands (ecosystem-manager, resource-experimenter, app-debugger, etc.)
- [ ] OT-P0-004 | Task monitoring dashboard | Trello-like dark theme UI showing tasks across active/backlog/staged/completed/failed columns
- [ ] OT-P0-005 | Real-time agent monitoring | Display what each agent is working on with live status updates
- [ ] OT-P0-006 | Manual task creation | Allow humans to create priority tasks by dropping YAML files in tasks/backlog/manual/

### 🟠 P1 – Should have post-launch
- [ ] OT-P1-001 | Learning and adaptation | Track success/failure rates and adjust priority weights based on actual vs predicted impact
- [ ] OT-P1-002 | Pattern recognition | Store successful task sequences and failure modes in Qdrant for future reference
- [ ] OT-P1-003 | Cooldown management | Prevent thrashing on repeatedly failing tasks with exponential backoff
- [ ] OT-P1-004 | Dependency resolution | Execute tasks in correct order based on dependency chains
- [ ] OT-P1-005 | PROBLEMS.md scanning | Automatically discover and create tasks from PROBLEMS.md files across resources and scenarios
- [ ] OT-P1-006 | YOLO mode | Auto-approve AI-generated tasks when enabled in configuration

### 🟢 P2 – Future / expansion
- [ ] OT-P2-001 | Distributed tracing | Full execution tracing across multi-step task workflows
- [ ] OT-P2-002 | Mobile monitoring app | iOS/Android app for task monitoring and approval
- [ ] OT-P2-003 | Advanced analytics | Success metrics, throughput charts, resource usage graphs, and learning rate visualization
- [ ] OT-P2-004 | Task templates | Reusable task templates for common improvement patterns
- [ ] OT-P2-005 | Multi-cluster coordination | Coordinate work across multiple Vrooli installations

## 🧱 Tech Direction Snapshot
- Preferred stacks / frameworks: Go API (task orchestration and priority engine), React UI (Trello-inspired dashboard with dark theme), n8n workflows (automated scanning and task generation)
- Data + storage expectations: PostgreSQL (task metadata and execution history), Redis (task locks and real-time updates), Qdrant (learning patterns and task embeddings for pattern recognition)
- Integration strategy: CLI-first integration calling scenario CLIs directly (e.g., `ecosystem-manager add scenario`, `app-debugger analyze`), n8n workflows for orchestration, file-based task system for transparency and manual intervention
- Non-goals / guardrails: No API-based scenario integration (use CLIs instead), no replacement of existing scenario UIs (complement not replace), no task execution within swarm-manager itself (delegate to appropriate scenarios)

## 🤝 Dependencies & Launch Plan
- Required resources: postgres (task storage), redis (locks and pub/sub), qdrant (pattern storage), n8n (orchestration), claude-code (task analysis and execution)
- Scenario dependencies: ecosystem-manager (unified resource/scenario management), resource-experimenter (testing resource integrations), app-debugger (error analysis), app-issue-tracker (issue management), system-monitor (health monitoring), task-planner (planning assistance)
- Operational risks: Claude Code availability and API limits could throttle execution; task priority miscalculation could cause important work to be delayed; infinite task generation loops if backlog generator is too aggressive
- Launch sequencing: Phase 1 - Deploy alongside auto/ for comparison (2 weeks), Phase 2 - Promote swarm-manager to primary with auto/ as fallback (1 month), Phase 3 - Disable auto/ loops and achieve full autonomy (ongoing validation)

## 🎨 UX & Branding
- Look & feel: Modern dark professional aesthetic with dark blue-black background (#1a1a2e), Trello-inspired kanban board with drag-and-drop, real-time updates, neon accents for status indicators
- Accessibility: High contrast color coding for priority and severity, keyboard navigation for all task operations, screen reader support for task status, color-blind friendly status indicators
- Voice & messaging: Technical, authoritative, focused on autonomous intelligence - "The brain of Vrooli that never stops improving"
- Branding hooks: Task priority badges with color coding (🔥 Critical 1000+, 🚨 User Requests 500-999, 🔧 System Health 200-499, 🌱 Capability Growth 100-199, 📚 Knowledge Building 1-99)

## 📎 Appendix

Success metrics, task file format, and reference documentation are maintained in the scenario's README.md and supporting documentation.
