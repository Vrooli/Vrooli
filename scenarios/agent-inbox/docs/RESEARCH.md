# Research Notes

## Uniqueness Check

Searched existing scenarios for chat/inbox functionality:

- **ai-chatbot-manager**: Creates embeddable chatbots for website visitors (B2B SaaS). Different purpose - that's for THEIR customers, this is for YOUR personal AI chats.
- **prompt-manager**: Manages prompt templates, not conversations
- **vrooli-assistant**: General assistant, not multi-chat inbox management

**Conclusion**: agent-inbox is unique - no existing scenario provides a personal AI chat inbox with multi-model support and terminal view.

## Related Scenarios

| Scenario | Relationship |
|----------|-------------|
| ai-chatbot-manager | Similar tech stack (Go API, React, Ollama) but different use case |
| prompt-manager | Could integrate saved prompts as chat starters |
| audio-tools | Could add voice input/output in P2 |

## External References

### OpenRouter
- API Docs: https://openrouter.ai/docs
- Supports streaming via Server-Sent Events
- Model list endpoint: GET /api/v1/models
- Chat completions: POST /api/v1/chat/completions

### Terminal UI Inspiration
- xterm.js for browser-based terminal rendering
- node-pty for PTY session management
- Consider ttyd or gotty patterns for Go-based terminal server

### Email Client UX Patterns
- Gmail: Labels, archive, keyboard shortcuts (j/k/e/r)
- Superhuman: Split view, command palette (Cmd+K)
- Spark: Smart inbox, pinned emails

### Ollama Auto-Naming
- Use small fast model (e.g., llama3.2:1b) for quick naming
- Prompt pattern: "Summarize this conversation in 3-5 words: {first N messages}"
- Trigger after 2-3 message exchanges or on user request

## Technical Decisions

### Why OpenRouter vs Direct Model APIs?
- Single API key for multiple models
- Unified interface for switching models
- Cost tracking built-in
- Fallback/routing handled by OpenRouter

### Why Ollama for Naming?
- Free, local, fast
- No API costs for metadata operations
- Privacy - conversation content stays local
- Works offline

### Why PostgreSQL?
- Full-text search built-in
- JSONB for flexible message metadata
- Transactional integrity for chat operations
- Already used across Vrooli scenarios
