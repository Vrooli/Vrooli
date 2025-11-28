# Requirements Registry

Requirements are organized by functional area, mapping to PRD operational targets.

## Directory Structure

```
requirements/
  01-inbox/           # Inbox list view and management
    list-view/        # Chat list, read/unread, archive (P0-001, P0-007, P1-003, P1-004, P1-008)
    labels/           # Label CRUD and assignment (P0-008)
  02-chat/            # Chat conversation functionality
    bubble-view/      # ChatGPT-style interface (P0-002, P0-003, P0-004, P1-006)
    persistence/      # PostgreSQL storage (P0-009, P1-005)
  03-naming/          # Chat naming
    auto-naming.json  # Ollama auto-naming and manual rename (P0-005, P0-006)
  04-terminal/        # Terminal view for coding agents
    core.json         # PTY sessions, agent selection (P0-010)
  05-enhancements/    # Post-launch improvements
    search.json       # Full-text search (P1-001)
    keyboard.json     # Keyboard navigation (P1-002)
```

## Lifecycle
1. Operational targets in PRD map to folders here.
2. `requirements/index.json` imports each module; tests auto-sync their status when they run.
3. Coverage summaries live in `coverage/phase-results/` after each test phase.

## Tagging Tests
Add `[REQ:ID]` to test names or descriptions:
```go
// [REQ:BUBBLE-003] Test streaming responses
func TestStreamingResponse(t *testing.T) { ... }
```

## Test Commands
```bash
vrooli scenario test agent-inbox              # Run all tests
vrooli scenario test agent-inbox --phase unit # Run unit tests only
```

See `docs/testing/guides/requirement-tracking-quick-start.md` for schema details.
