# Research: agent-manager

## Uniqueness Check

Searched for `agent-manager` in `scenarios/` on 2025-12-18:

- **No duplicate scenario found** - agent-manager is unique
- Found references in:
  - `scenarios/workspace-sandbox/PRD.md` (OT-P2-008 mentions "Agent-Manager Integration")
  - This confirms workspace-sandbox was designed with agent-manager integration in mind

## Related Scenarios

| Scenario | Relationship | Notes |
|----------|--------------|-------|
| workspace-sandbox | Required dependency | Provides sandbox creation, diff generation, patch application. agent-manager delegates all isolation operations to workspace-sandbox. |
| agent-inbox | Consumer | Will use agent-manager for running AI chat agents. Consumes RunEvent stream for chat history. |
| ecosystem-manager | Consumer | Currently runs agents directly; should transition to using agent-manager as central control plane. |
| app-issue-tracker | Consumer (planned) | Intended to use agent-manager for issue resolution agents. |

## Related Resources

| Resource | Relationship | Notes |
|----------|--------------|-------|
| claude-code | Runner | Supported runner type. adapter needed to invoke Claude Code CLI. |
| codex | Runner | Supported runner type. Adapter needed for OpenAI Codex API calls. |
| opencode | Runner | Supported runner type. Adapter needed for open-source code models. |
| postgres | Required | Stores tasks, runs, events, agent profiles, policies. |

## External References

### Agent Orchestration Patterns
- **Langchain Agent Executor**: Similar concept of agent lifecycle management with tool tracking
- **CrewAI**: Multi-agent orchestration with task delegation patterns
- **AutoGPT**: Autonomous agent execution with task decomposition

### Sandbox/Isolation
- **Docker/Podman**: Container-based isolation (heavier than overlayfs)
- **bubblewrap**: Used by workspace-sandbox for process isolation
- **overlayfs**: Copy-on-write filesystem for efficient snapshots

### Event Streaming Patterns
- **Event Sourcing**: Append-only event logs as source of truth
- **CQRS**: Separate read/write models for event processing
- **WebSocket Streaming**: Real-time event delivery pattern

## Architecture Decisions

### AD-001: workspace-sandbox as Sole Isolation Provider
**Decision**: All sandbox operations delegate to workspace-sandbox rather than implementing isolation directly.
**Rationale**:
- workspace-sandbox already handles overlayfs, bubblewrap, diff generation, patch application
- Avoids duplication of complex isolation logic
- Single point for isolation policy enforcement
- Consistent behavior across all agents

### AD-002: Event-Native Design
**Decision**: All agent output captured as structured events in append-only log.
**Rationale**:
- Enables agent-inbox to display conversation history
- Supports debugging and audit requirements
- Allows replay and analysis of agent behavior
- Compatible with event sourcing patterns

### AD-003: Interface-Based Runner Adapters
**Decision**: Runners implemented as adapters behind common interface.
**Rationale**:
- Adding new runners doesn't require core changes
- Runners can declare their capabilities (message support, cost tracking)
- Consistent execution model across all runner types
- Enables runner-specific optimizations without core modifications

## Open Questions

1. **Event Retention Policy**: How long to keep RunEvents? Compress/archive strategy?
2. **Cost Attribution**: How to attribute API costs when multiple runners used?
3. **Sandbox Reuse**: Should completed runs reuse sandboxes for efficiency?
4. **Multi-project Support**: Currently assumes single project root; future multi-project needs?
