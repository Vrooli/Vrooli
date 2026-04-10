# Compaction Event Type Implementation Plan

**Status**: Ready for Implementation
**Created**: 2026-02-24
**Scope**: Add first-class `RUN_EVENT_TYPE_COMPACTION` event type across the full stack

## Executive Summary

Implement proper compaction event handling so that when a runner (like claude-code) performs context compaction (manually via `/compact` or automatically), the agent-inbox UI can display it professionally with visual dividers, token savings statistics, and clear context boundaries.

## Architecture Overview

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                              LAYER DIAGRAM                                   │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                             │
│   ┌─────────────────┐                                                       │
│   │ 1. PROTO LAYER  │  packages/proto/schemas/agent-manager/v1/domain/      │
│   │                 │  ├── types.proto (enum)                               │
│   │                 │  └── events.proto (CompactionEventData message)       │
│   └────────┬────────┘                                                       │
│            │ make generate                                                  │
│            ▼                                                                │
│   ┌─────────────────┐                                                       │
│   │ 2. DOMAIN LAYER │  scenarios/agent-manager/api/internal/domain/         │
│   │                 │  └── types.go (CompactionEventData struct)            │
│   └────────┬────────┘                                                       │
│            │ NewCompactionEvent()                                           │
│            ▼                                                                │
│   ┌─────────────────┐                                                       │
│   │ 3. RUNNER LAYER │  scenarios/agent-manager/api/internal/adapters/runner/│
│   │                 │  └── claude_code.go (detect & emit compaction)        │
│   └────────┬────────┘                                                       │
│            │ gRPC / HTTP                                                    │
│            ▼                                                                │
│   ┌─────────────────┐                                                       │
│   │ 4. INTEGRATION  │  scenarios/agent-inbox/api/integrations/              │
│   │    LAYER        │  └── agent_manager.go (TranslateProtoEvent)           │
│   └────────┬────────┘                                                       │
│            │ JSON over HTTP                                                 │
│            ▼                                                                │
│   ┌─────────────────┐                                                       │
│   │ 5. FRONTEND     │  scenarios/agent-inbox/ui/src/                        │
│   │    LAYER        │  ├── lib/api.ts (AgentEvent type)                     │
│   │                 │  └── components/chat/agent/                           │
│   │                 │      ├── AgentEventList.tsx (grouping)                │
│   │                 │      └── AgentCompactionCard.tsx (NEW)                │
│   └─────────────────┘                                                       │
│                                                                             │
└─────────────────────────────────────────────────────────────────────────────┘
```

## Implementation Phases

---

## Phase 1: Proto Schema Definition

**Goal**: Define the CompactionEventData message and add enum value
**Estimated Files**: 2
**Dependencies**: None

### 1.1 Add Enum Value

**File**: `packages/proto/schemas/agent-manager/v1/domain/types.proto`

```protobuf
enum RunEventType {
  RUN_EVENT_TYPE_UNSPECIFIED = 0;
  RUN_EVENT_TYPE_LOG = 1;
  RUN_EVENT_TYPE_MESSAGE = 2;
  RUN_EVENT_TYPE_TOOL_CALL = 3;
  RUN_EVENT_TYPE_TOOL_RESULT = 4;
  RUN_EVENT_TYPE_STATUS = 5;
  RUN_EVENT_TYPE_METRIC = 6;
  RUN_EVENT_TYPE_ARTIFACT = 7;
  RUN_EVENT_TYPE_ERROR = 8;
  RUN_EVENT_TYPE_MESSAGE_DELETED = 9;
  RUN_EVENT_TYPE_COMPACTION = 10;  // NEW
}
```

### 1.2 Add CompactionEventData Message

**File**: `packages/proto/schemas/agent-manager/v1/domain/events.proto`

Add after line ~520 (after existing event data messages):

```protobuf
// CompactionEventData represents a context compaction event where
// conversation history is summarized to fit within context limits.
message CompactionEventData {
  // The summarized conversation content
  string summary = 1;

  // What triggered the compaction
  // Values: "manual" (user typed /compact), "auto" (context limit reached)
  string trigger = 2;

  // User-provided focus instruction (from "/compact focus on X")
  // Empty string if no focus was specified
  string focus = 3;

  // Statistics about the compaction
  int64 messages_compacted = 4;  // Number of messages summarized
  int64 tokens_before = 5;       // Token count before compaction
  int64 tokens_after = 6;        // Token count after compaction

  // Original compaction command if manual (e.g., "/compact focus on auth")
  string original_command = 7;
}
```

Add to RunEvent oneof (around line 98):

```protobuf
message RunEvent {
  // ... existing fields ...

  oneof data {
    // ... existing cases ...
    MessageDeletedEventData message_deleted = 21;
    CompactionEventData compaction = 22;  // NEW - field number 22
  }
}
```

### 1.3 Regenerate Protos

```bash
cd /home/matthalloran8/Vrooli/packages/proto
make generate
```

### 1.4 Validation Checklist
- [ ] Proto files compile without errors
- [ ] Generated Go files in `packages/proto/gen/go/` include CompactionEventData
- [ ] No breaking changes to existing field numbers

---

## Phase 2: Domain Layer (agent-manager)

**Goal**: Add Go domain types and constructor for compaction events
**Estimated Files**: 1-2
**Dependencies**: Phase 1 complete

### 2.1 Add CompactionEventData Struct

**File**: `scenarios/agent-manager/api/internal/domain/types.go`

Add after other EventData structs (~line 500):

```go
// CompactionEventData represents a context compaction/summarization event.
type CompactionEventData struct {
    Summary            string `json:"summary"`
    Trigger            string `json:"trigger"`              // "manual" or "auto"
    Focus              string `json:"focus,omitempty"`      // Optional focus instruction
    MessagesCompacted  int64  `json:"messagesCompacted"`
    TokensBefore       int64  `json:"tokensBefore"`
    TokensAfter        int64  `json:"tokensAfter"`
    OriginalCommand    string `json:"originalCommand,omitempty"`
}

// EventType implements EventPayload
func (d *CompactionEventData) EventType() RunEventType {
    return EventTypeCompaction
}

// isEventPayload implements EventPayload marker
func (d *CompactionEventData) isEventPayload() {}
```

### 2.2 Add EventType Constant

**File**: `scenarios/agent-manager/api/internal/domain/types.go`

Add to constants section:

```go
const (
    // ... existing constants ...
    EventTypeCompaction RunEventType = "compaction"
)
```

### 2.3 Add Constructor Function

**File**: `scenarios/agent-manager/api/internal/domain/types.go`

Add after other New*Event functions:

```go
// NewCompactionEvent creates a new compaction event.
// trigger should be "manual" or "auto".
// focus is optional (empty string if not specified).
func NewCompactionEvent(
    runID uuid.UUID,
    summary string,
    trigger string,
    focus string,
    messagesCompacted int64,
    tokensBefore int64,
    tokensAfter int64,
    originalCommand string,
) *RunEvent {
    return &RunEvent{
        ID:        uuid.New(),
        RunID:     runID,
        EventType: EventTypeCompaction,
        Timestamp: time.Now(),
        Data: &CompactionEventData{
            Summary:           summary,
            Trigger:           trigger,
            Focus:             focus,
            MessagesCompacted: messagesCompacted,
            TokensBefore:      tokensBefore,
            TokensAfter:       tokensAfter,
            OriginalCommand:   originalCommand,
        },
    }
}
```

### 2.4 Add Unit Tests

**File**: `scenarios/agent-manager/api/internal/domain/types_test.go` (create if needed)

```go
func TestNewCompactionEvent(t *testing.T) {
    runID := uuid.New()
    event := NewCompactionEvent(
        runID,
        "Summary of authentication work...",
        "manual",
        "auth",
        47,
        89432,
        3201,
        "/compact focus on auth",
    )

    assert.NotEqual(t, uuid.Nil, event.ID)
    assert.Equal(t, runID, event.RunID)
    assert.Equal(t, EventTypeCompaction, event.EventType)
    assert.NotZero(t, event.Timestamp)

    data, ok := event.Data.(*CompactionEventData)
    require.True(t, ok)
    assert.Equal(t, "Summary of authentication work...", data.Summary)
    assert.Equal(t, "manual", data.Trigger)
    assert.Equal(t, "auth", data.Focus)
    assert.Equal(t, int64(47), data.MessagesCompacted)
    assert.Equal(t, int64(89432), data.TokensBefore)
    assert.Equal(t, int64(3201), data.TokensAfter)
    assert.Equal(t, "/compact focus on auth", data.OriginalCommand)
}

func TestCompactionEventData_EventType(t *testing.T) {
    data := &CompactionEventData{}
    assert.Equal(t, EventTypeCompaction, data.EventType())
}
```

### 2.5 Validation Checklist
- [ ] Domain types compile
- [ ] Constructor creates valid events
- [ ] Unit tests pass: `go test ./internal/domain/...`

---

## Phase 3: Runner Adapter (claude-code)

**Goal**: Detect compaction in claude-code output and emit CompactionEvent
**Estimated Files**: 2
**Dependencies**: Phase 2 complete

### 3.1 Research: Claude Code Compaction Output Format

**Key insight**: Claude Code doesn't emit a special "compaction" event type in its JSON stream.
Instead, compaction manifests as:
1. A user message containing `/compact [instructions]`
2. An assistant message containing `<summary>...</summary>` tags

**Detection Strategy**:
- Track when we see a message starting with `/compact`
- When the next assistant message contains `<summary>`, emit CompactionEvent
- Extract token counts from usage metrics if available

### 3.2 Add Compaction Detection State

**File**: `scenarios/agent-manager/api/internal/adapters/runner/claude_code.go`

Add struct for tracking compaction state:

```go
// compactionState tracks pending compaction detection
type compactionState struct {
    pendingCompact  bool   // True if we just saw a /compact command
    originalCommand string // The full /compact command text
    focus           string // Extracted focus instruction
}

// parseCompactCommand extracts focus from "/compact focus on auth" -> "auth"
func parseCompactCommand(content string) (isCompact bool, focus string) {
    content = strings.TrimSpace(content)
    if !strings.HasPrefix(content, "/compact") {
        return false, ""
    }

    // Extract focus instruction if present
    // Patterns: "/compact focus on X", "/compact X", "/compact"
    remainder := strings.TrimPrefix(content, "/compact")
    remainder = strings.TrimSpace(remainder)

    if strings.HasPrefix(remainder, "focus on ") {
        focus = strings.TrimPrefix(remainder, "focus on ")
    } else if remainder != "" {
        focus = remainder
    }

    return true, strings.TrimSpace(focus)
}

// isCompactionSummary checks if content looks like a compaction summary
func isCompactionSummary(content string) bool {
    return strings.Contains(content, "<summary>") ||
           strings.HasPrefix(strings.TrimSpace(content), "Summary of")
}

// extractSummaryContent extracts content from <summary>...</summary> tags
func extractSummaryContent(content string) string {
    start := strings.Index(content, "<summary>")
    end := strings.Index(content, "</summary>")

    if start != -1 && end != -1 && end > start {
        return strings.TrimSpace(content[start+9 : end])
    }

    // No tags, return as-is (some runners don't use tags)
    return content
}
```

### 3.3 Modify parseStreamEvents to Detect Compaction

**File**: `scenarios/agent-manager/api/internal/adapters/runner/claude_code.go`

Update the Execute function to track compaction state:

```go
func (r *ClaudeCodeRunner) Execute(ctx context.Context, req ExecuteRequest) (*ExecuteResult, error) {
    // ... existing setup code ...

    var compaction compactionState
    var tokensBefore int64 // Track from usage events

    // ... in the line processing loop ...

    for scanner.Scan() {
        line := scanner.Text()
        events, err := r.parseStreamEvents(runID, line, &compaction)
        if err != nil {
            // ... error handling ...
        }

        for _, event := range events {
            // Track token usage for compaction stats
            if metric, ok := event.Data.(*domain.MetricEventData); ok {
                if metric.Name == "input_tokens" {
                    tokensBefore = int64(metric.Value)
                }
            }

            if err := req.EventSink.Emit(event); err != nil {
                // ... error handling ...
            }
        }
    }
}

// Updated signature to include compaction state
func (r *ClaudeCodeRunner) parseStreamEvents(
    runID uuid.UUID,
    line string,
    compaction *compactionState,
) ([]*domain.RunEvent, error) {
    // ... existing JSON parsing ...

    switch streamEvent.Type {
    case "message":
        msg := streamEvent.Message

        // Check for /compact command
        if msg.Role == "user" {
            if isCompact, focus := parseCompactCommand(msg.Content); isCompact {
                compaction.pendingCompact = true
                compaction.originalCommand = msg.Content
                compaction.focus = focus

                // Don't emit the /compact as a regular message
                // It will be included in the compaction event
                return nil, nil
            }
        }

        // Check for compaction summary response
        if compaction.pendingCompact && msg.Role == "assistant" {
            if isCompactionSummary(msg.Content) {
                compaction.pendingCompact = false

                summary := extractSummaryContent(msg.Content)

                return []*domain.RunEvent{
                    domain.NewCompactionEvent(
                        runID,
                        summary,
                        "manual",           // trigger
                        compaction.focus,
                        0,                  // messagesCompacted (not available)
                        0,                  // tokensBefore (could track)
                        0,                  // tokensAfter (could track)
                        compaction.originalCommand,
                    ),
                }, nil
            }

            // Not a summary, reset state and emit as normal message
            compaction.pendingCompact = false
        }

        // Normal message handling
        return []*domain.RunEvent{
            domain.NewMessageEvent(runID, msg.Role, msg.Content),
        }, nil

    // ... rest of existing cases ...
    }
}
```

### 3.4 Add Unit Tests for Compaction Detection

**File**: `scenarios/agent-manager/api/internal/adapters/runner/claude_code_test.go`

```go
func TestParseCompactCommand(t *testing.T) {
    tests := []struct {
        input     string
        isCompact bool
        focus     string
    }{
        {"/compact", true, ""},
        {"/compact focus on auth", true, "auth"},
        {"/compact focus on API changes", true, "API changes"},
        {"/compact authentication flow", true, "authentication flow"},
        {"  /compact  ", true, ""},
        {"regular message", false, ""},
        {"/compacting", false, ""},  // Not a match
    }

    for _, tt := range tests {
        t.Run(tt.input, func(t *testing.T) {
            isCompact, focus := parseCompactCommand(tt.input)
            assert.Equal(t, tt.isCompact, isCompact)
            assert.Equal(t, tt.focus, focus)
        })
    }
}

func TestIsCompactionSummary(t *testing.T) {
    tests := []struct {
        content  string
        expected bool
    }{
        {"<summary>We worked on auth...</summary>", true},
        {"Summary of the conversation so far...", true},
        {"Here is what we did...", false},
        {"The <summary> tag is mentioned but not a summary", false},
    }

    for _, tt := range tests {
        t.Run(tt.content[:min(30, len(tt.content))], func(t *testing.T) {
            assert.Equal(t, tt.expected, isCompactionSummary(tt.content))
        })
    }
}

func TestExtractSummaryContent(t *testing.T) {
    tests := []struct {
        input    string
        expected string
    }{
        {
            "<summary>Auth bug was fixed by updating token validation</summary>",
            "Auth bug was fixed by updating token validation",
        },
        {
            "Some preamble\n<summary>The actual summary</summary>\nSome epilogue",
            "The actual summary",
        },
        {
            "No tags here, just plain text",
            "No tags here, just plain text",
        },
    }

    for _, tt := range tests {
        t.Run(tt.expected[:min(20, len(tt.expected))], func(t *testing.T) {
            assert.Equal(t, tt.expected, extractSummaryContent(tt.input))
        })
    }
}

func TestParseStreamEvents_CompactionFlow(t *testing.T) {
    runner := &ClaudeCodeRunner{}
    runID := uuid.New()
    compaction := &compactionState{}

    // Step 1: User sends /compact command
    events1, err := runner.parseStreamEvents(
        runID,
        `{"type":"message","message":{"role":"user","content":"/compact focus on auth"}}`,
        compaction,
    )
    require.NoError(t, err)
    assert.Empty(t, events1)  // /compact is absorbed, not emitted
    assert.True(t, compaction.pendingCompact)
    assert.Equal(t, "auth", compaction.focus)

    // Step 2: Assistant responds with summary
    events2, err := runner.parseStreamEvents(
        runID,
        `{"type":"message","message":{"role":"assistant","content":"<summary>We fixed the auth bug...</summary>"}}`,
        compaction,
    )
    require.NoError(t, err)
    require.Len(t, events2, 1)

    event := events2[0]
    assert.Equal(t, domain.EventTypeCompaction, event.EventType)

    data, ok := event.Data.(*domain.CompactionEventData)
    require.True(t, ok)
    assert.Equal(t, "We fixed the auth bug...", data.Summary)
    assert.Equal(t, "manual", data.Trigger)
    assert.Equal(t, "auth", data.Focus)
    assert.Equal(t, "/compact focus on auth", data.OriginalCommand)

    // State should be reset
    assert.False(t, compaction.pendingCompact)
}
```

### 3.5 Validation Checklist
- [ ] Compaction detection functions work correctly
- [ ] Unit tests pass: `go test ./internal/adapters/runner/...`
- [ ] No regression in existing event parsing

---

## Phase 4: Integration Layer (agent-inbox API)

**Goal**: Translate proto CompactionEventData to frontend-consumable format
**Estimated Files**: 2
**Dependencies**: Phase 1, Phase 3 complete

### 4.1 Add Enum Mapping

**File**: `scenarios/agent-inbox/api/integrations/agent_manager.go`

Update `protoEventTypeToString`:

```go
func protoEventTypeToString(et domainpb.RunEventType) string {
    switch et {
    // ... existing cases ...
    case domainpb.RunEventType_RUN_EVENT_TYPE_MESSAGE_DELETED:
        return "message_deleted"
    case domainpb.RunEventType_RUN_EVENT_TYPE_COMPACTION:
        return "compaction"
    default:
        return et.String()
    }
}
```

### 4.2 Update TranslatedEvent Struct

**File**: `scenarios/agent-inbox/api/integrations/agent_manager.go`

Add compaction-specific fields:

```go
type TranslatedEvent struct {
    // ... existing fields ...

    // Compaction fields
    CompactionTrigger          string `json:"compaction_trigger,omitempty"`
    CompactionFocus            string `json:"compaction_focus,omitempty"`
    CompactionMessagesCompacted int64  `json:"compaction_messages_compacted,omitempty"`
    CompactionTokensBefore     int64  `json:"compaction_tokens_before,omitempty"`
    CompactionTokensAfter      int64  `json:"compaction_tokens_after,omitempty"`
    CompactionOriginalCommand  string `json:"compaction_original_command,omitempty"`
}
```

### 4.3 Add Translation Case

**File**: `scenarios/agent-inbox/api/integrations/agent_manager.go`

Update `TranslateProtoEvent`:

```go
func TranslateProtoEvent(ev *domainpb.RunEvent) *TranslatedEvent {
    event := &TranslatedEvent{
        // ... existing initialization ...
    }

    switch d := ev.Data.(type) {
    // ... existing cases ...

    case *domainpb.RunEvent_Compaction:
        compaction := d.Compaction
        event.Role = "system"
        event.Content = compaction.GetSummary()
        event.CompactionTrigger = compaction.GetTrigger()
        event.CompactionFocus = compaction.GetFocus()
        event.CompactionMessagesCompacted = compaction.GetMessagesCompacted()
        event.CompactionTokensBefore = compaction.GetTokensBefore()
        event.CompactionTokensAfter = compaction.GetTokensAfter()
        event.CompactionOriginalCommand = compaction.GetOriginalCommand()

        // Also store raw data for debugging
        if rawBytes, err := protoMarshalOpts.Marshal(compaction); err == nil {
            event.RawData = string(rawBytes)
        }

    default:
        // ... existing default handling ...
    }

    return event
}
```

### 4.4 Add Integration Tests

**File**: `scenarios/agent-inbox/api/integrations/agent_manager_test.go`

```go
func TestTranslateProtoEvent_Compaction(t *testing.T) {
    ts := timestamppb.New(time.Date(2025, 1, 15, 10, 30, 0, 0, time.UTC))
    ev := &domainpb.RunEvent{
        Id:        "evt-compaction-1",
        Sequence:  42,
        EventType: domainpb.RunEventType_RUN_EVENT_TYPE_COMPACTION,
        Timestamp: ts,
        Data: &domainpb.RunEvent_Compaction{
            Compaction: &domainpb.CompactionEventData{
                Summary:           "We fixed the authentication bug by...",
                Trigger:           "manual",
                Focus:             "auth",
                MessagesCompacted: 47,
                TokensBefore:      89432,
                TokensAfter:       3201,
                OriginalCommand:   "/compact focus on auth",
            },
        },
    }

    result := TranslateProtoEvent(ev)

    assert.Equal(t, "evt-compaction-1", result.ID)
    assert.Equal(t, "compaction", result.Type)
    assert.Equal(t, "system", result.Role)
    assert.Equal(t, "We fixed the authentication bug by...", result.Content)
    assert.Equal(t, "manual", result.CompactionTrigger)
    assert.Equal(t, "auth", result.CompactionFocus)
    assert.Equal(t, int64(47), result.CompactionMessagesCompacted)
    assert.Equal(t, int64(89432), result.CompactionTokensBefore)
    assert.Equal(t, int64(3201), result.CompactionTokensAfter)
    assert.Equal(t, "/compact focus on auth", result.CompactionOriginalCommand)
}

func TestProtoEventTypeToString_Compaction(t *testing.T) {
    result := protoEventTypeToString(domainpb.RunEventType_RUN_EVENT_TYPE_COMPACTION)
    assert.Equal(t, "compaction", result)
}
```

### 4.5 Validation Checklist
- [ ] Translation compiles without errors
- [ ] Tests pass: `go test ./integrations/...`
- [ ] JSON output includes compaction fields

---

## Phase 5: Frontend Types (TypeScript)

**Goal**: Add TypeScript types for compaction events
**Estimated Files**: 1
**Dependencies**: Phase 4 complete

### 5.1 Update AgentEvent Interface

**File**: `scenarios/agent-inbox/ui/src/lib/api.ts`

```typescript
export interface AgentEvent {
  id: string;
  /** Known types: message, tool_call, tool_result, status, error, log, metric, artifact, message_deleted, compaction */
  type: string;
  role: "user" | "assistant" | "system" | "tool";
  content: string;
  timestamp: string;
  sequence: number;

  // Tool fields
  tool_name?: string;
  tool_call_id?: string;
  tool_input?: string;
  tool_output?: string;
  tool_success?: boolean;

  // Status fields
  run_status?: AgentRunStatus;
  phase?: string;
  progress?: number;

  // Compaction fields (NEW)
  compaction_trigger?: "manual" | "auto";
  compaction_focus?: string;
  compaction_messages_compacted?: number;
  compaction_tokens_before?: number;
  compaction_tokens_after?: number;
  compaction_original_command?: string;

  // Raw data for generic display
  raw_data?: string;
}
```

### 5.2 Add Type Guard

**File**: `scenarios/agent-inbox/ui/src/lib/api.ts`

```typescript
/** Type guard for compaction events */
export function isCompactionEvent(event: AgentEvent): boolean {
  return event.type === "compaction";
}

/** Calculate token reduction percentage */
export function getCompactionReduction(event: AgentEvent): number | null {
  if (!isCompactionEvent(event)) return null;
  if (!event.compaction_tokens_before || event.compaction_tokens_before === 0) return null;

  const before = event.compaction_tokens_before;
  const after = event.compaction_tokens_after ?? 0;
  return Math.round(((before - after) / before) * 100);
}
```

### 5.3 Validation Checklist
- [ ] TypeScript compiles without errors
- [ ] Type guards work correctly

---

## Phase 6: Frontend Component (AgentCompactionCard)

**Goal**: Create professional compaction card component
**Estimated Files**: 2-3
**Dependencies**: Phase 5 complete

### 6.1 Create AgentCompactionCard Component

**File**: `scenarios/agent-inbox/ui/src/components/chat/agent/AgentCompactionCard.tsx`

```tsx
import { useState } from "react";
import { ChevronDown, ChevronRight, Package, Sparkles } from "lucide-react";
import type { AgentEvent } from "../../../lib/api";
import { getCompactionReduction } from "../../../lib/api";
import type { ViewMode } from "../../settings/Settings";
import { cn } from "../../../lib/utils";

interface AgentCompactionCardProps {
  event: AgentEvent;
  viewMode?: ViewMode;
}

export function AgentCompactionCard({ event, viewMode = "bubble" }: AgentCompactionCardProps) {
  const [isExpanded, setIsExpanded] = useState(false);
  const isCompact = viewMode === "compact";

  const reduction = getCompactionReduction(event);
  const isManual = event.compaction_trigger === "manual";

  return (
    <div
      className={cn(
        "relative",
        isCompact ? "my-2" : "my-4"
      )}
    >
      {/* Divider line */}
      <div className="absolute inset-x-0 top-1/2 -translate-y-1/2 h-px bg-gradient-to-r from-transparent via-amber-500/50 to-transparent" />

      {/* Card */}
      <div
        className={cn(
          "relative mx-auto max-w-2xl rounded-lg border",
          "border-amber-500/30 bg-amber-500/5",
          isCompact ? "px-3 py-2" : "px-4 py-3"
        )}
      >
        {/* Header */}
        <button
          onClick={() => setIsExpanded(!isExpanded)}
          className="flex w-full items-center justify-between gap-3 text-left"
        >
          <div className="flex items-center gap-2">
            <Package className="h-4 w-4 text-amber-400" />
            <span className={cn(
              "font-medium text-amber-200",
              isCompact ? "text-xs" : "text-sm"
            )}>
              Conversation Compacted
            </span>
            {isManual && (
              <span className="rounded bg-amber-500/20 px-1.5 py-0.5 text-[10px] font-medium text-amber-300">
                Manual
              </span>
            )}
          </div>

          <div className="flex items-center gap-3">
            {/* Stats badges */}
            {event.compaction_messages_compacted && event.compaction_messages_compacted > 0 && (
              <span className="text-xs text-zinc-400">
                {event.compaction_messages_compacted} messages
              </span>
            )}
            {reduction !== null && (
              <span className="flex items-center gap-1 text-xs text-emerald-400">
                <Sparkles className="h-3 w-3" />
                {reduction}% smaller
              </span>
            )}

            {/* Expand toggle */}
            {isExpanded ? (
              <ChevronDown className="h-4 w-4 text-zinc-400" />
            ) : (
              <ChevronRight className="h-4 w-4 text-zinc-400" />
            )}
          </div>
        </button>

        {/* Original command (if manual) */}
        {isManual && event.compaction_original_command && (
          <div className={cn(
            "mt-2 rounded bg-zinc-800/50 font-mono text-amber-300/80",
            isCompact ? "px-2 py-1 text-xs" : "px-3 py-1.5 text-sm"
          )}>
            {event.compaction_original_command}
          </div>
        )}

        {/* Focus indicator */}
        {event.compaction_focus && (
          <div className={cn(
            "mt-2 text-zinc-400",
            isCompact ? "text-xs" : "text-sm"
          )}>
            Focus: <span className="text-zinc-300">{event.compaction_focus}</span>
          </div>
        )}

        {/* Expandable summary */}
        {isExpanded && (
          <div className={cn(
            "mt-3 border-t border-amber-500/20 pt-3",
            isCompact ? "text-xs" : "text-sm"
          )}>
            <div className="mb-2 text-xs font-medium uppercase tracking-wider text-zinc-500">
              Summary
            </div>
            <div className="whitespace-pre-wrap text-zinc-300">
              {event.content}
            </div>

            {/* Token stats (if available) */}
            {event.compaction_tokens_before && event.compaction_tokens_before > 0 && (
              <div className="mt-3 flex gap-4 text-xs text-zinc-500">
                <span>
                  Before: <span className="text-zinc-400">{event.compaction_tokens_before.toLocaleString()} tokens</span>
                </span>
                <span>
                  After: <span className="text-zinc-400">{(event.compaction_tokens_after ?? 0).toLocaleString()} tokens</span>
                </span>
              </div>
            )}
          </div>
        )}
      </div>
    </div>
  );
}

export default AgentCompactionCard;
```

### 6.2 Update AgentEventList Grouping

**File**: `scenarios/agent-inbox/ui/src/components/chat/agent/AgentEventList.tsx`

Update the groupedEvents useMemo:

```typescript
const groupedEvents = useMemo(() => {
  // ... existing tool result indexing ...

  const grouped: Array<{
    type: "message" | "tool" | "status" | "error" | "compaction" | "raw";  // Add "compaction"
    event: AgentEvent;
    result?: AgentEvent;
  }> = [];

  events.forEach(event => {
    if (event.type === "message") {
      grouped.push({ type: "message", event });
    } else if (event.type === "tool_call") {
      // ... existing tool handling ...
    } else if (event.type === "tool_result") {
      // ... existing tool result handling ...
    } else if (event.type === "compaction") {
      // NEW: Compaction events get their own type
      grouped.push({ type: "compaction", event });
    } else if (event.type === "status") {
      grouped.push({ type: "status", event });
    } else if (event.type === "error") {
      grouped.push({ type: "error", event });
    } else if (event.type === "log" || event.type === "metric") {
      return; // Skip
    } else {
      grouped.push({ type: "raw", event });
    }
  });

  return grouped;
}, [events]);
```

Update the render switch:

```typescript
{groupedEvents.map((item, index) => {
  switch (item.type) {
    // ... existing cases ...

    case "compaction":
      return (
        <AgentCompactionCard
          key={item.event.id || index}
          event={item.event}
          viewMode={viewMode}
        />
      );

    // ... rest of cases ...
  }
})}
```

### 6.3 Add Import

**File**: `scenarios/agent-inbox/ui/src/components/chat/agent/AgentEventList.tsx`

```typescript
import AgentCompactionCard from "./AgentCompactionCard";
```

### 6.4 Validation Checklist
- [ ] Component renders without errors
- [ ] Compact and bubble view modes work
- [ ] Expand/collapse works
- [ ] Stats display correctly

---

## Phase 7: Testing & Integration Tests

**Goal**: Comprehensive test coverage across all layers
**Estimated Files**: 3-4
**Dependencies**: Phases 1-6 complete

### 7.1 Add E2E-style Integration Test

**File**: `scenarios/agent-inbox/ui/src/components/chat/agent/__tests__/AgentCompactionCard.test.tsx`

```tsx
import { render, screen, fireEvent } from "@testing-library/react";
import { AgentCompactionCard } from "../AgentCompactionCard";
import type { AgentEvent } from "../../../../lib/api";

const mockCompactionEvent: AgentEvent = {
  id: "evt-compact-1",
  type: "compaction",
  role: "system",
  content: "We worked on authentication. Fixed token validation bug in auth.ts. Added rate limiting to login endpoint.",
  timestamp: "2025-01-15T10:30:00Z",
  sequence: 42,
  compaction_trigger: "manual",
  compaction_focus: "auth",
  compaction_messages_compacted: 47,
  compaction_tokens_before: 89432,
  compaction_tokens_after: 3201,
  compaction_original_command: "/compact focus on auth",
};

describe("AgentCompactionCard", () => {
  it("renders compaction header", () => {
    render(<AgentCompactionCard event={mockCompactionEvent} />);

    expect(screen.getByText("Conversation Compacted")).toBeInTheDocument();
    expect(screen.getByText("Manual")).toBeInTheDocument();
  });

  it("shows message count", () => {
    render(<AgentCompactionCard event={mockCompactionEvent} />);

    expect(screen.getByText("47 messages")).toBeInTheDocument();
  });

  it("calculates and shows reduction percentage", () => {
    render(<AgentCompactionCard event={mockCompactionEvent} />);

    // (89432 - 3201) / 89432 * 100 = 96%
    expect(screen.getByText("96% smaller")).toBeInTheDocument();
  });

  it("shows original command for manual compaction", () => {
    render(<AgentCompactionCard event={mockCompactionEvent} />);

    expect(screen.getByText("/compact focus on auth")).toBeInTheDocument();
  });

  it("shows focus indicator", () => {
    render(<AgentCompactionCard event={mockCompactionEvent} />);

    expect(screen.getByText("Focus:")).toBeInTheDocument();
    expect(screen.getByText("auth")).toBeInTheDocument();
  });

  it("expands to show summary on click", () => {
    render(<AgentCompactionCard event={mockCompactionEvent} />);

    // Summary should not be visible initially
    expect(screen.queryByText(/We worked on authentication/)).not.toBeInTheDocument();

    // Click to expand
    fireEvent.click(screen.getByRole("button"));

    // Summary should now be visible
    expect(screen.getByText(/We worked on authentication/)).toBeInTheDocument();
    expect(screen.getByText("89,432 tokens")).toBeInTheDocument();
    expect(screen.getByText("3,201 tokens")).toBeInTheDocument();
  });

  it("handles auto trigger without original command", () => {
    const autoEvent: AgentEvent = {
      ...mockCompactionEvent,
      compaction_trigger: "auto",
      compaction_original_command: undefined,
    };

    render(<AgentCompactionCard event={autoEvent} />);

    expect(screen.queryByText("Manual")).not.toBeInTheDocument();
    expect(screen.queryByText("/compact")).not.toBeInTheDocument();
  });

  it("renders in compact view mode", () => {
    render(<AgentCompactionCard event={mockCompactionEvent} viewMode="compact" />);

    expect(screen.getByText("Conversation Compacted")).toBeInTheDocument();
  });
});
```

### 7.2 Add AgentEventList Compaction Test

**File**: `scenarios/agent-inbox/ui/src/components/chat/agent/__tests__/AgentEventList.test.tsx`

Add test case:

```tsx
it("renders compaction events with AgentCompactionCard", () => {
  const events: AgentEvent[] = [
    {
      id: "evt-1",
      type: "message",
      role: "user",
      content: "Help me fix the auth bug",
      timestamp: "2025-01-15T10:00:00Z",
      sequence: 1,
    },
    {
      id: "evt-2",
      type: "compaction",
      role: "system",
      content: "Summary of auth work...",
      timestamp: "2025-01-15T10:30:00Z",
      sequence: 2,
      compaction_trigger: "manual",
      compaction_focus: "auth",
    },
    {
      id: "evt-3",
      type: "message",
      role: "user",
      content: "Now add rate limiting",
      timestamp: "2025-01-15T10:31:00Z",
      sequence: 3,
    },
  ];

  render(<AgentEventList events={events} />);

  expect(screen.getByText("Help me fix the auth bug")).toBeInTheDocument();
  expect(screen.getByText("Conversation Compacted")).toBeInTheDocument();
  expect(screen.getByText("Now add rate limiting")).toBeInTheDocument();
});
```

### 7.3 Validation Checklist
- [ ] All unit tests pass
- [ ] All integration tests pass
- [ ] Manual testing with real /compact command works

---

## Phase 8: Linting, Types, and Cleanup

**Goal**: Fix all lint errors, type errors, and warnings
**Dependencies**: Phases 1-7 complete

### 8.1 Go Linting and Formatting

```bash
# agent-manager
cd scenarios/agent-manager/api
gofumpt -w .
golangci-lint run --fix
go vet ./...

# agent-inbox API
cd scenarios/agent-inbox/api
gofumpt -w .
golangci-lint run --fix
go vet ./...
```

### 8.2 TypeScript Linting and Type Checking

```bash
cd scenarios/agent-inbox/ui
npm run lint -- --fix
npm run typecheck
```

### 8.3 Proto Lint

```bash
cd packages/proto
buf lint
```

### 8.4 Run All Tests

```bash
# Proto regeneration
cd packages/proto && make generate

# Go tests
cd scenarios/agent-manager/api && go test ./...
cd scenarios/agent-inbox/api && go test ./...

# TypeScript tests
cd scenarios/agent-inbox/ui && npm test
```

### 8.5 Validation Checklist
- [ ] `gofumpt` reports no changes needed
- [ ] `golangci-lint run` passes with no errors
- [ ] `npm run lint` passes
- [ ] `npm run typecheck` passes
- [ ] All tests pass

---

## Phase 9: Scenario Restart and Health Check

**Goal**: Restart affected scenarios and verify health
**Dependencies**: Phase 8 complete

### 9.1 Stop Running Scenarios

```bash
cd scenarios/agent-manager && make stop
cd scenarios/agent-inbox && make stop
```

### 9.2 Rebuild and Start

```bash
# Rebuild agent-manager
cd scenarios/agent-manager && make build && make start

# Rebuild agent-inbox
cd scenarios/agent-inbox && make build && make start
```

### 9.3 Verify Health

```bash
# Check agent-manager health
curl http://localhost:PORT/health

# Check agent-inbox health
curl http://localhost:PORT/health

# Check logs for errors
cd scenarios/agent-manager && make logs
cd scenarios/agent-inbox && make logs
```

### 9.4 Manual Integration Test

1. Open agent-inbox UI
2. Start an agent chat session
3. Have a conversation with several tool calls
4. Type `/compact focus on what we accomplished`
5. Verify:
   - [ ] Compaction card appears with amber styling
   - [ ] "Conversation Compacted" label visible
   - [ ] "Manual" badge shown
   - [ ] Original command displayed
   - [ ] Focus indicator shows
   - [ ] Click expands to show summary
   - [ ] Subsequent messages appear below the compaction card

### 9.5 Validation Checklist
- [ ] Both scenarios start successfully
- [ ] Health endpoints return 200
- [ ] No errors in logs
- [ ] Manual /compact test works end-to-end

---

## File Change Summary

| Layer | File | Change Type |
|-------|------|-------------|
| Proto | `packages/proto/schemas/agent-manager/v1/domain/types.proto` | Modify |
| Proto | `packages/proto/schemas/agent-manager/v1/domain/events.proto` | Modify |
| Domain | `scenarios/agent-manager/api/internal/domain/types.go` | Modify |
| Domain | `scenarios/agent-manager/api/internal/domain/types_test.go` | Create/Modify |
| Runner | `scenarios/agent-manager/api/internal/adapters/runner/claude_code.go` | Modify |
| Runner | `scenarios/agent-manager/api/internal/adapters/runner/claude_code_test.go` | Modify |
| Integration | `scenarios/agent-inbox/api/integrations/agent_manager.go` | Modify |
| Integration | `scenarios/agent-inbox/api/integrations/agent_manager_test.go` | Modify |
| Frontend | `scenarios/agent-inbox/ui/src/lib/api.ts` | Modify |
| Frontend | `scenarios/agent-inbox/ui/src/components/chat/agent/AgentCompactionCard.tsx` | Create |
| Frontend | `scenarios/agent-inbox/ui/src/components/chat/agent/AgentEventList.tsx` | Modify |
| Frontend | `scenarios/agent-inbox/ui/src/components/chat/agent/__tests__/AgentCompactionCard.test.tsx` | Create |

---

## Risk Mitigation

| Risk | Mitigation |
|------|------------|
| Proto regeneration breaks existing code | Run `make generate` early, fix compile errors before proceeding |
| Runner detection misses compaction | Test with multiple /compact variations; add fallback heuristics |
| Token counts unavailable | Make stats optional; display "N/A" when missing |
| Different runner formats | Abstract detection behind interface; add runner-specific adapters if needed |
| UI breaks in edge cases | Test empty/null fields; add defensive rendering |

---

## Success Criteria

1. **Functional**: `/compact` command in claude-code emits proper compaction event
2. **Visual**: Agent-inbox UI shows professional compaction card with divider
3. **Informative**: Card displays trigger type, focus, message count, token reduction
4. **Expandable**: Summary is collapsible to reduce visual noise
5. **Tested**: All layers have unit tests; integration tests pass
6. **Clean**: No lint errors, type errors, or warnings
7. **Healthy**: Both scenarios restart and pass health checks
