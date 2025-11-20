# Product Requirements Document (PRD)

> **Version**: 2.0.0
> **Last Updated**: 2025-11-19
> **Status**: Active
> **Template**: Canonical PRD Template v2.0.0

## 🎯 Overview

**What permanent capability does this scenario add to Vrooli?**

Vrooli-orchestrator adds **contextual intelligence adaptation** - the ability to automatically configure Vrooli's entire environment (resources, scenarios, and UI) for specific user contexts, use cases, and organizational needs. Instead of a one-size-fits-all startup, Vrooli becomes infinitely customizable through startup profiles that define exactly what runs and how.

**Primary users**: System administrators, power users, developers, business users, and different household types who need quick environment switching.

**Deployment surfaces**: CLI (`vrooli-orchestrator`), REST API, Web UI dashboard, and integration with main `vrooli` CLI.

**How this makes agents smarter**: Agents can programmatically switch Vrooli configurations based on current tasks (e.g., load development tools for coding, creative tools for content creation). Other scenarios can preemptively load dependencies before showing content. The system learns usage patterns and optimizes profile recommendations over time.

## 🎯 Operational Targets

### 🔴 P0 – Must ship for viability

- [ ] OT-P0-001 | Create, edit, and delete startup profiles via CLI and UI | Users can define custom profiles with specific resources and scenarios
- [ ] OT-P0-002 | Activate profiles that start/stop specific resources and scenarios | Profile activation orchestrates full environment setup
- [ ] OT-P0-003 | Profile persistence in local configuration files | Profiles survive system restarts and can be version controlled
- [ ] OT-P0-004 | CLI interface: `vrooli-orchestrator activate <profile-name>` | Command-line activation for scripting and automation
- [ ] OT-P0-005 | Basic profile metadata (name, description, target_audience) | Profiles are self-documenting and discoverable
- [ ] OT-P0-006 | Integration with main `vrooli` CLI for resource/scenario lifecycle | Orchestrator uses standard Vrooli lifecycle management

### 🟠 P1 – Should have post-launch

- [ ] OT-P1-001 | Profile templates for common use cases (developer, business, household) | Reduces setup friction with pre-built profiles
- [ ] OT-P1-002 | Auto-open browser tabs for configured dashboards | Profiles can launch web interfaces automatically
- [ ] OT-P1-003 | Idle shutdown timers for resource management | Automatic cleanup saves resources when idle
- [ ] OT-P1-004 | Environment variable overrides per profile | Profiles can customize scenario behavior
- [ ] OT-P1-005 | Profile validation and dependency checking | Pre-activation checks prevent runtime errors
- [ ] OT-P1-006 | Web UI dashboard for visual profile management | Non-technical users can manage profiles visually

### 🟢 P2 – Future / expansion

- [ ] OT-P2-001 | Usage analytics and profile optimization recommendations | System learns which profiles work best for different contexts
- [ ] OT-P2-002 | Profile sharing and versioning | Teams can share and iterate on profile configurations
- [ ] OT-P2-003 | Conditional activation based on time/context | Profiles activate automatically based on calendar or system state
- [ ] OT-P2-004 | Integration with calendar for automatic profile switching | Work/personal mode switching based on schedule
- [ ] OT-P2-005 | Resource usage monitoring and optimization | Profiles adapt to available system resources

## 🧱 Tech Direction Snapshot

**Architecture approach**: Go API backend with Postgres persistence, React web UI, and native CLI binary. Integrates with core Vrooli lifecycle system for resource/scenario management.

**Data storage**: Postgres for profile configurations, activation history, and analytics. Profile data structures allow composition and dependency tracking.

**Integration strategy**:
- Primary: Shared workflows (n8n) for config management CRUD
- Secondary: Resource CLI (`resource-postgres`, `vrooli resource`, `vrooli scenario`) for lifecycle
- Tertiary: Direct API calls only when real-time status needed

**API contract**: RESTful endpoints at `/api/v1/profiles` for CRUD, activation, and status. Target SLA: <200ms for list/get, <30s for activation.

**CLI design**: Native binary `vrooli-orchestrator` with commands mirroring API endpoints. Supports JSON output for automation and human-readable tables for interactive use.

**Non-goals**: Multi-tenant isolation (single-user local system), distributed orchestration (local-only for v1.0), GUI configuration editor (CLI-first, UI is read-only monitoring for v1.0).

## 🤝 Dependencies & Launch Plan

**Required resources**:
- `postgres`: Profile storage, activation history, and analytics
- Vrooli core CLI: Must support `vrooli resource start/stop` and `vrooli scenario run/stop` commands

**Optional resources**:
- `browserless`: Auto-open browser tabs for dashboard URLs (fallback: print URLs to console)

**Scenario dependencies**: None (foundational capability that others will consume)

**Launch sequencing**:
1. Implement core profile CRUD API and CLI
2. Add profile activation/deactivation logic
3. Integrate with Vrooli lifecycle system
4. Seed default profiles (developer, business, minimal)
5. Add web UI for profile monitoring
6. Document profile creation and usage patterns

**Known risks**:
- Resource conflicts during activation (mitigation: pre-activation validation)
- Profile corruption/inconsistency (mitigation: schema validation and backups)
- Activation timeout/failure (mitigation: graceful fallbacks and partial activation support)

## 🎨 UX & Branding

**Visual style**: Professional mission control dashboard aesthetic. Dark color scheme with modern typography. Layout emphasizes status visibility and quick actions. Subtle animations for state transitions.

**Personality**: Technical and focused. Conveys control and confidence over complex systems. Similar aesthetic to system-monitor and app-debugger (infrastructure reliability).

**Accessibility**: WCAG AA compliance required. Keyboard navigation essential for power users. Clear visual hierarchy separating status, actions, and configuration.

**Responsive design**: Desktop-first (command center feel) with mobile support for status monitoring. CLI remains primary interface for power users.

**Information architecture**:
- Status dashboard: Current active profile, running resources/scenarios
- Profile library: Browse and filter available profiles
- Activation controls: One-click or one-command activation
- Configuration: Create/edit profiles with validation feedback

## 📎 Appendix

### Related Scenarios

This capability enables:
- **app-monitor**: Preload auto-next scenarios before displaying them
- **morning-vision-walk**: Activate productivity profiles automatically for work sessions
- **deployment-manager**: Create customer-specific minimal profiles for app delivery
- **usage-analytics**: Track which profiles are most effective for different user types

### Performance Targets

| Metric | Target | Measurement |
|--------|--------|-------------|
| Profile Activation | <30s | CLI timing |
| Profile Switch | <15s | System monitoring |
| Config Load | <2s | API monitoring |
| Resource Cleanup | <10s | Process monitoring |

### Data Models

**Profile Schema**:
```json
{
  "id": "UUID",
  "name": "string (kebab-case)",
  "display_name": "string",
  "description": "string",
  "metadata": {
    "target_audience": "string",
    "resource_footprint": "low|medium|high",
    "use_case": "string",
    "created_by": "string",
    "created_at": "timestamp"
  },
  "resources": ["string"],
  "scenarios": ["string"],
  "auto_browser": ["string"],
  "environment_vars": "object",
  "idle_shutdown_minutes": "integer",
  "dependencies": ["string"],
  "status": "active|inactive|error"
}
```

**ProfileActivation Schema**:
```json
{
  "id": "UUID",
  "profile_id": "UUID",
  "activated_at": "timestamp",
  "deactivated_at": "timestamp",
  "user_context": "string",
  "success": "boolean",
  "error_details": "string"
}
```

### Test Strategy

Shared test runner (`test/run-tests.sh`) orchestrates phased suite:
- **Structure**: Validates directory layout and critical assets
- **Dependencies**: Checks Go tooling, npm packages, and CLI availability
- **Unit**: Go tests with coverage thresholds (warn ≥80%, fail <50%)
- **Integration**: Auto-managed runtime tests for API and UI endpoints
- **Business**: Validates seeded profiles and CLI workflow with bats
- **Performance**: Tracks throughput benchmarks (pending implementation)

Run via `vrooli scenario test vrooli-orchestrator` or `./test/run-tests.sh`.

### References

- [README.md](README.md) - User-facing overview and quick start
- [TEST_IMPLEMENTATION_SUMMARY.md](TEST_IMPLEMENTATION_SUMMARY.md) - Test coverage details
- Docker Compose documentation for orchestration patterns
- Kubernetes documentation for distributed concepts
