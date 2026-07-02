# Product Requirements Document (PRD)

> **Version**: 1.0.0
> **Last Updated**: 2025-11-28
> **Status**: Canonical Specification
> **Scenario**: agent-inbox

## 🎯 Overview

**Purpose**: Agent Inbox provides a unified chat management interface for AI conversations, designed like an email client. Users can manage multiple concurrent AI conversations with different models, organize them with labels, track read/unread status, and seamlessly switch between chat views.

The system provides a ChatGPT-style bubble interface for conversational AI (powered by OpenRouter), with tool calling support for dispatching tasks to coding agents via agent-manager integration.

**Primary users**: Developers, power users, and AI enthusiasts who interact with multiple AI models daily
**Deployment surfaces**: API, CLI, Web UI
**Intelligence amplification**: Creates reusable patterns for multi-model chat management, agent-dispatching via tool calls, and conversation organization that other scenarios can leverage.

## 🎯 Operational Targets

### 🔴 P0 – Must ship for viability
- [x] OT-P0-001 | Inbox list view | Display all chats in an email-like list showing name, preview, labels, and read/unread status
- [x] OT-P0-002 | Chat creation | Create new chat sessions with selected model and view mode
- [x] OT-P0-003 | Bubble chat view | ChatGPT-style message bubbles with streaming responses from OpenRouter models
- [x] OT-P0-004 | Model selection | Switch between available OpenRouter models mid-conversation
- [x] OT-P0-005 | Auto-naming with Ollama | Automatically generate descriptive chat names using local Ollama based on conversation content
- [x] OT-P0-006 | Manual renaming | Rename chats manually at any time
- [x] OT-P0-007 | Read/unread tracking | Mark conversations as read/unread with visual indicators
- [x] OT-P0-008 | Label management | Create, edit, delete, and assign colored labels to organize chats
- [x] OT-P0-009 | Chat persistence | Store all conversations in SQLite with full message history
- [x] OT-P0-010 | Tool calling | Support for AI-driven tool calls that dispatch tasks to agent-manager

### 🟠 P1 – Should have post-launch
- [ ] OT-P1-001 | Search functionality | Full-text search across all chat messages
- [x] OT-P1-002 | Keyboard shortcuts | Email-style keyboard navigation (j/k, enter, escape, etc.)
- [x] OT-P1-003 | Archive functionality | Archive old chats without deleting them
- [x] OT-P1-004 | Bulk operations | Select multiple chats for bulk label/archive/delete actions
- [x] OT-P1-005 | Chat export | Export chat history to markdown, JSON, or text formats
- [x] OT-P1-006 | Conversation forking | Create a new chat branching from a specific point in an existing conversation
- [x] OT-P1-007 | Model usage stats | Track token usage and costs per model
- [x] OT-P1-008 | Starred chats | Pin important chats to the top of the inbox

### 🟢 P2 – Future / expansion
- [ ] OT-P2-001 | Multi-agent conversations | Chat sessions with multiple AI models collaborating
- [ ] OT-P2-002 | Prompt templates | Save and reuse system prompts and conversation starters
- [ ] OT-P2-003 | Chat sharing | Generate shareable links for specific conversations
- [ ] OT-P2-004 | Webhook integrations | Trigger external actions based on chat events
- [ ] OT-P2-005 | Voice input/output | Speech-to-text input and text-to-speech responses
- [ ] OT-P2-006 | Plugin system | Allow custom tools and integrations within chat context
- [ ] OT-P2-007 | Conversation analytics | Insights on conversation patterns, frequent topics, model performance

## 🧱 Tech Direction Snapshot

**Preferred stacks**:
- **UI Stack**: React with Vite, TailwindCSS for styling, modern component architecture
- **API Stack**: Go API server with WebSocket support for streaming responses
- **Data Storage**: Embedded SQLite for chats, messages, labels; FTS5 full-text search; zero-config, portable
- **AI Integration**:
  - OpenRouter API for chat completions (primary chat models)
  - Ollama (local) for auto-naming chats (fast, privacy-preserving)
- **Agent Integration**: Tool calling via agent-manager for coding tasks, WebSocket for real-time streaming
- **Non-goals**: Building our own AI models, mobile-first design (desktop-first), real-time collaboration between users

## Dependencies & Launch Plan

**Required resources**:
- SQLite - Embedded database for chats, messages, labels, user preferences (zero-config)
- Ollama - Local LLM for fast, private chat auto-naming

**Required external services**:
- OpenRouter API - Access to Claude, GPT-4, and other chat models

**Launch risks**:
- OpenRouter API rate limits or outages (mitigation: graceful degradation with clear error messages)
- Agent execution security (mitigation: agent-manager handles sandboxing and resource limits)
- Large conversation histories impacting performance (mitigation: pagination and virtual scrolling)

**Launch sequence**: Local development -> Single-user deployment -> Multi-user with auth

## UX & Branding

**Visual palette**: Clean, professional interface inspired by modern email clients (Gmail, Superhuman). Dark mode default with light mode option.
**Accessibility commitments**: WCAG 2.1 AA compliance, keyboard-navigable interface
**Voice/personality**: Efficient, minimal, focused on productivity. No unnecessary animations or distractions.
**Target feeling**: Users should feel organized and in control of their AI conversations, like managing a well-organized inbox
**Key interactions**:
- Single-click to open chat, double-click to rename
- Drag-and-drop labels onto chats
- Right-click context menu for quick actions
- Split-pane layout with resizable sidebar
