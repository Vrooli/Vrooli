# Agent Inbox

AI chat inbox serving as the conversational hub for Vrooli's agent ecosystem.

## Overview

Agent Inbox is a unified chat management interface that acts as a "dispatcher + timeline" for AI conversations. It handles normal chat via OpenRouter while dispatching specialized tasks to other Vrooli scenarios as tools. Think of it like an intelligent foreman that can:

- **Chat**: Normal conversation, planning, brainstorming via OpenRouter (Claude, GPT-4, etc.)
- **Dispatch**: Spawn coding agents, track issues, run investigations via scenario tools
- **Organize**: Email-like inbox with labels, read/unread status, archive

## Key Architecture

Agent Inbox follows an **"inbox as dispatcher"** pattern:

- **Inbox calls OpenRouter** for assistant messages (normal chat, reasoning, planning)
- **Inbox uses tools** to delegate work to specialized scenarios:
  - `agent-manager`: Spawn Claude Code/Codex/OpenCode for coding tasks
  - Future: `app-issue-tracker`, investigation scenarios, etc.
- **Tool results** stream back and display in the chat timeline

This separation keeps chat cheap/fast while coding agents handle expensive/stateful work.

## Features

### Inbox Management
- Email-like list view with chat previews
- Labels with custom colors for organization
- Read/unread status tracking
- Archive, star, and delete chats
- Tool call history tracking per chat

### Chat Interface
- Streaming AI responses via SSE
- Tool calling with live progress display
- Switch models mid-conversation
- Auto-generate chat names using local Ollama
- Keyboard shortcuts (j/k navigation, Cmd+N new chat)

### Tool Calling
- **spawn_coding_agent**: Run Claude Code, Codex, or OpenCode on a task
- **check_agent_status**: Monitor running agent progress
- **stop_agent**: Cancel a running agent
- **list_active_agents**: See all active agent runs
- **get_agent_diff**: View code changes from an agent run
- **approve_agent_changes**: Apply agent changes to main workspace

## Quick Start

```bash
# Start the scenario
cd scenarios/agent-inbox && make start

# Or via CLI
vrooli scenario start agent-inbox

# Access the UI at the displayed port
# Typically: http://localhost:[UI_PORT]
```

## Prerequisites

- PostgreSQL (for chat storage)
- Ollama (for auto-naming chats)
- OpenRouter API key (for chat completions)

Set your OpenRouter API key:
```bash
export OPENROUTER_API_KEY="your-key-here"
```

## CLI Usage

```bash
agent-inbox status          # Check API health
agent-inbox list            # List all chats
agent-inbox new             # Create new chat
agent-inbox open <id>       # Open a chat
agent-inbox labels          # List labels
agent-inbox models          # List available models
```

## API Endpoints

| Method | Path | Description |
|--------|------|-------------|
| GET | /health | Service health check |
| GET | /api/v1/chats | List all chats |
| POST | /api/v1/chats | Create new chat |
| GET | /api/v1/chats/:id | Get chat with messages |
| PATCH | /api/v1/chats/:id | Update chat (rename, labels) |
| DELETE | /api/v1/chats/:id | Delete chat |
| POST | /api/v1/chats/:id/messages | Send message |
| POST | /api/v1/chats/:id/complete | Get AI completion (supports streaming) |
| GET | /api/v1/chats/:id/tool-calls | List tool calls for a chat |
| POST | /api/v1/chats/:id/auto-name | Auto-generate name |
| GET | /api/v1/labels | List labels |
| POST | /api/v1/labels | Create label |
| GET | /api/v1/models | List OpenRouter models |
| GET | /api/v1/tools | List available tools |

## Environment Variables

| Variable | Required | Description |
|----------|----------|-------------|
| `API_PORT` | Yes | Port for Go API server |
| `UI_PORT` | Yes | Port for React UI |
| `OPENROUTER_API_KEY` | Yes | OpenRouter API key |
| `DATABASE_URL` | Yes | PostgreSQL connection string |

## Development

```bash
# Install dependencies
cd ui && pnpm install

# Build
make build

# Run tests
make test

# Start dev servers
make dev
```

## Documentation

- [PRD.md](PRD.md) - Product requirements and operational targets
- [docs/PROGRESS.md](docs/PROGRESS.md) - Development progress log
- [docs/PROBLEMS.md](docs/PROBLEMS.md) - Known issues and blockers
- [docs/RESEARCH.md](docs/RESEARCH.md) - Research notes and references
- [requirements/README.md](requirements/README.md) - Requirements registry

## Architecture

```
┌─────────────────┐    ┌──────────────────┐    ┌─────────────────┐
│   React UI      │    │    Go API        │    │   PostgreSQL    │
│   (Inbox)       │◄──►│   REST/SSE       │◄──►│   (Storage)     │
└─────────────────┘    └──────────────────┘    └─────────────────┘
                               │
              ┌────────────────┼────────────────┐
              ▼                ▼                ▼
     ┌────────────────┐ ┌────────────┐ ┌──────────────────┐
     │   OpenRouter   │ │   Ollama   │ │  agent-manager   │
     │   (Chat + Tools)│ │  (Naming)  │ │  (Coding Agents) │
     └────────────────┘ └────────────┘ └──────────────────┘
                                               │
                              ┌────────────────┼────────────────┐
                              ▼                ▼                ▼
                     ┌────────────────┐ ┌──────────┐ ┌──────────────┐
                     │  Claude Code   │ │  Codex   │ │   OpenCode   │
                     └────────────────┘ └──────────┘ └──────────────┘
```

### Flow

1. **User sends message** → Stored in PostgreSQL, forwarded to OpenRouter
2. **OpenRouter responds** → Either text content or tool calls (or both)
3. **If tool_calls** → API executes each tool against target scenario (e.g., agent-manager)
4. **Tool results** → Stored as "tool" role messages, streamed to UI
5. **Follow-up** → OpenRouter gets tool results for next response turn
