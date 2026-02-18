# Known Issues and Problems

## Open Issues

### P0 Blockers
_None at initialization - scenario scaffold only_

### P1 Issues
_None at initialization_

### P2 Minor Issues
_None at initialization_

## Deferred Ideas

### Not In Scope (Intentionally Excluded)
- **Full metrics collection**: system-monitor handles detailed metrics; autoheal focuses on health/recovery
- **Alerting/paging**: Use dedicated alerting scenarios for notifications
- **Non-Vrooli service management**: Only manages Vrooli resources, scenarios, and OS watchdog

### Future Consideration
- **AI-powered root cause analysis** (OT-P2-005): Use Ollama to analyze failure patterns
- **Custom check plugins** (OT-P2-004): Allow external health check definitions
- **Certificate monitoring** (OT-P2-001): Check SSL cert expiration
- **Display manager health** (OT-P2-002): GDM/lightdm/sddm monitoring

## Architecture Decisions

### ADR-001: CLI-based resource/scenario management
**Decision**: Use `vrooli resource/scenario` CLI commands instead of direct API calls
**Rationale**: CLI already handles authentication, port discovery, and error handling; reduces code duplication
**Trade-offs**: Slightly higher latency than direct calls; dependency on vrooli CLI being installed

### ADR-002: In-memory state with PostgreSQL persistence
**Decision**: Keep current health state in memory, persist history to PostgreSQL
**Rationale**: Fast status queries, survives process restarts, supports UI history views
**Trade-offs**: State lost on restart (acceptable - will rebuild on next tick)

### ADR-003: OS watchdog keeps autoheal running
**Decision**: Install OS-level service (systemd/launchd/Windows) that monitors autoheal itself
**Rationale**: Ensures autoheal survives even its own crashes; brings system back after reboots
**Trade-offs**: Requires elevated privileges for installation on some platforms

## Risk Register

| Risk | Likelihood | Impact | Mitigation |
|------|-----------|--------|------------|
| WSL platform detection edge cases | Medium | Low | Test explicitly on WSL1 and WSL2 |
| Watchdog installation fails without sudo | Medium | Medium | Graceful degradation with warning |
| Circular dependency (autoheal monitors scenarios that monitor autoheal) | Low | Medium | Exclude autoheal from monitored scenario list |
| Too-aggressive restarts cause thrashing | Low | High | Add restart cooldown and max retry limits |
