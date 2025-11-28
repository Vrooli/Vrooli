# Agent Inbox

AI chat inbox for managing conversations with multiple AI models and coding agents.

## Overview

Agent Inbox is a unified chat management interface designed like an email client. Organize multiple AI conversations with labels, track read/unread status, and choose between two interaction modes:

- **Bubble Chat View**: ChatGPT-style message bubbles powered by OpenRouter (Claude, GPT-4, etc.)
- **Terminal View**: Console interface for coding agents like Claude Code and Codex

## Features

### Inbox Management
- Email-like list view with chat previews
- Labels with custom colors for organization
- Read/unread status tracking
- Archive and delete chats
- Full-text search across all conversations

### Chat Interface
- Streaming AI responses via WebSocket
- Switch models mid-conversation
- Auto-generate chat names using local Ollama
- Manual rename with double-click
- Keyboard shortcuts (j/k navigation, Cmd+N new chat)

### Terminal Mode
- PTY-based terminal sessions
- Real-time output streaming
- Support for Claude Code, Codex, and other CLI agents
- Sandboxed execution with resource limits

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
| POST | /api/v1/chats/:id/auto-name | Auto-generate name |
| GET | /api/v1/labels | List labels |
| POST | /api/v1/labels | Create label |
| GET | /api/v1/models | List OpenRouter models |

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
│   (Inbox)       │◄──►│   REST/WebSocket │◄──►│   (Storage)     │
└─────────────────┘    └──────────────────┘    └─────────────────┘
                               │
              ┌────────────────┼────────────────┐
              ▼                ▼                ▼
     ┌────────────────┐ ┌────────────┐ ┌──────────────┐
     │   OpenRouter   │ │   Ollama   │ │  PTY/Terminal│
     │   (Chat AI)    │ │  (Naming)  │ │   (Agents)   │
     └────────────────┘ └────────────┘ └──────────────┘
```
