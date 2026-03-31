# Implementation Plan: Show Agent-Manager Messages in Execution Details Page

## Required Reading

```bash
prompt-manager skill read implementation-plan-authoring ux react-coherence
```

## 1. Purpose

Add a "Messages" tab to the swarm-manager execution details page that displays the full agent-manager conversation history (messages, tool calls, tool results, logs) for a given execution's run, similar to how prompt-manager displays run events.

## 2. Problem Statement

The execution details page (`ExecutionDetailsPage.tsx`) currently shows status, metadata, post-run checks, and prompt trace — but no visibility into the actual agent conversation that occurred during execution. Users must switch to agent-manager directly to see what happened. The `ExecutionRecord` already carries a `runId` field linking to the agent-manager run, but the UI doesn't use it to fetch or display events.

## 3. Scope

**In scope:**
- New "Messages" tab on the execution details page
- Fetching agent-manager run events via API
- Rendering message, tool_call, tool_result, log, and status event types
- Pagination/lazy loading for large event streams

**Out of scope:**
- Real-time WebSocket streaming of events (future enhancement)
- Editing or replaying conversations
- Changes to agent-manager API itself
- Changes to prompt-manager

## 4. Current Technical Context

### Key Files

| File | Role |
|------|------|
| `scenarios/swarm-manager/ui/src/pages/ExecutionDetailsPage.tsx` | Main detail view — currently shows status card, post-run checks, prompt trace |
| `scenarios/swarm-manager/ui/src/types/domain.ts` | `ExecutionRecord` type with `runId` field |
| `scenarios/swarm-manager/ui/src/services/execution-service.ts` | API service for executions |
| `scenarios/agent-manager/api/internal/handlers/handlers.go` | Agent-manager endpoint: `GET /api/v1/runs/{id}/events` |
| `scenarios/agent-manager/api/internal/domain/types.go` | `RunEvent` type with event types: message, tool_call, tool_result, log, status, metric, artifact, error |

### Agent-Manager Event API

`GET /api/v1/runs/{id}/events` returns events with:
- `after_sequence` (pagination cursor)
- `limit` (page size)
- `event_types` (comma-separated filter)

Each `RunEvent` has: ID, RunID, Sequence, EventType, Timestamp, Data (type-specific payload).

### Existing Pattern: prompt-manager RunEditorPanel

prompt-manager's `RunEditorPanel.tsx` uses three tabs (Info, Events, Investigation) and fetches run details on mount. The Events tab shows a chronological event list. This is the reference pattern.

## 5. Target End State

The execution details page gains a tabbed interface with:
1. **Overview** tab (current content: status, metadata, post-run checks, prompt trace)
2. **Messages** tab (new: chronological agent conversation with message bubbles, tool call/result blocks, and log entries)

The Messages tab:
- Appears only when `execution.runId` is present
- Fetches events from agent-manager API on tab activation
- Paginates with "Load more" or infinite scroll
- Renders each event type with appropriate visual treatment
- Shows loading/empty/error states

## 6. Implementation Strategy

### Phase 1: API Integration Layer
- Add agent-manager event fetching to swarm-manager's service layer
- Define TypeScript types for RunEvent and event payloads
- Handle the cross-scenario API routing (swarm-manager proxy or direct agent-manager URL)

### Phase 2: Message Display Components
- `AgentMessageList` — container component with pagination
- `MessageBubble` — user/assistant/system message rendering
- `ToolCallBlock` — collapsible tool call with input display
- `ToolResultBlock` — tool result with success/error indication
- `LogEntry` — log level + message display
- `StatusEvent` — status transition indicator

### Phase 3: Tab Integration
- Convert ExecutionDetailsPage to use a tabbed layout (Radix Tabs or similar)
- Move existing content into "Overview" tab
- Add "Messages" tab that mounts the AgentMessageList
- Conditionally show Messages tab only when runId exists

### Phase 4: Polish
- Loading skeletons for message list
- Empty state when no events exist
- Error handling for agent-manager unavailability
- Mobile-responsive message layout

## 7. Contract Decisions

<!-- TBD — pending workshop decisions on data fetching approach and event type filtering -->

## 8. Testing Plan

<!-- TBD — pending approach decisions -->

## 9. Rollout/Validation Checklist

- [ ] Messages tab renders for executions with a runId
- [ ] Messages tab is hidden for executions without a runId
- [ ] All event types render without errors
- [ ] Pagination works for runs with many events
- [ ] Mobile layout is usable
- [ ] Error state shows when agent-manager is unavailable

## 10. Risks + Mitigations

| Risk | Mitigation |
|------|------------|
| Agent-manager may be unavailable | Graceful error state, tab still renders with error message |
| Large event streams (1000+ events) | Paginate with sequence-based cursor |
| Cross-scenario API routing | Determine correct URL resolution pattern (proxy vs. direct) |

## 11. Non-goals / Prohibited Patterns

- No real-time WebSocket streaming in this iteration
- No modifications to agent-manager API
- No client-side URL computation for cross-scenario routing (use backend endpoint per feedback)

## 12. Definition of Done

- Execution details page has a "Messages" tab when runId is present
- Tab displays full conversation history from agent-manager
- All event types have appropriate visual rendering
- Pagination handles large event streams
- Mobile-responsive layout
- Loading, empty, and error states are handled
