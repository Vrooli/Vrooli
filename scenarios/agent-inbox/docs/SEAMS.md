# Agent Inbox - Architectural Seams

This document describes the key architectural seams in the Agent Inbox scenario. A seam is a place in the code where behavior can be substituted without changing the code that uses it—critical for testing, flexibility, and maintainability.

## Domain Model

The Agent Inbox is an AI chat management interface with three core functions:

1. **Chat**: Manage AI conversations via OpenRouter (multiple models)
2. **Dispatch**: Delegate coding tasks to specialized agents (via agent-manager)
3. **Organize**: Email-like inbox with labels, read/unread, archive, stars

### Core Entities

| Entity | Description | Location |
|--------|-------------|----------|
| **Chat** | A conversation containing messages | `api/domain/types.go` |
| **Message** | A single turn in conversation (user/assistant/system/tool) | `api/domain/types.go` |
| **Label** | Colored category for organizing chats | `api/domain/types.go` |
| **ToolCall** | AI-requested invocation of external capability | `api/domain/types.go` |
| **ToolCallRecord** | Execution record with status, result, timing | `api/domain/types.go` |

---

## API Architecture

The API follows a "Screaming Architecture" where the folder structure clearly communicates the domain purpose:

```
api/
├── main.go              # Server wiring only (~140 lines)
├── domain/
│   └── types.go         # Core domain types, constants, validation
├── persistence/
│   └── repository.go    # All database operations
├── handlers/            # HTTP handlers organized by domain
│   ├── handlers.go      # Shared types, response helpers, route registration
│   ├── health.go        # Health check endpoint
│   ├── chat.go          # Chat CRUD operations
│   ├── message.go       # Message and chat state toggles
│   ├── label.go         # Label management
│   └── ai.go            # AI completion, models, tools, streaming
├── integrations/        # External service clients
│   ├── openrouter.go    # OpenRouter API client
│   ├── ollama.go        # Ollama client for auto-naming
│   ├── agent_manager.go # Agent-manager HTTP client
│   ├── tool_executor.go # Tool call routing and execution
│   └── tools.go         # Tool definitions
└── middleware/
    └── middleware.go    # CORS, logging
```

### Architecture Principles Applied

1. **Handlers organized by domain**: Instead of one massive main.go, handlers are split by business domain (chat, message, label, ai)
2. **Thin main.go**: Only server wiring and entry point (~140 lines vs original 968 lines)
3. **Clear seams**: Each package has a focused responsibility that can be substituted for testing
4. **Dependency injection**: Handlers receive dependencies (repo, ollama client) at construction time

---

## Key Seams

### Seam: Handlers

**Location**: `api/handlers/`

**Purpose**: HTTP request handling separated by domain responsibility.

**Files**:
- `handlers.go` - Shared Handlers struct, route registration, response helpers
- `health.go` - Health check endpoint
- `chat.go` - Chat CRUD (list, create, get, update, delete)
- `message.go` - Add message, toggle read/archive/star
- `label.go` - Label CRUD and chat assignment
- `ai.go` - AI completion (streaming/non-streaming), models, tools, auto-naming

**Substitution Points**:
- For **tests**: Create Handlers with mock Repository and OllamaClient
- For **different frameworks**: Implement new handlers with same interfaces

**Key Interface**:
```go
type Handlers struct {
    Repo         *persistence.Repository
    OllamaClient *integrations.OllamaClient
}

func New(repo, ollama) *Handlers
func (h *Handlers) RegisterRoutes(r *mux.Router)
```

---

### Seam: Repository (Database)

**Location**: `api/persistence/repository.go`

**Purpose**: Centralizes all database operations behind a single interface.

**Substitution Points**:
- For **unit tests**: Create a mock `Repository` that returns canned data
- For **integration tests**: Use a test database with `InitSchema()`
- For **different databases**: Implement a new repository with the same interface

**Key Operations**:
```go
// Chat operations
ListChats(ctx, archived, starred) ([]Chat, error)
GetChat(ctx, chatID) (*Chat, error)
CreateChat(ctx, name, model, viewMode) (*Chat, error)
UpdateChat(ctx, chatID, name, model) (*Chat, error)
DeleteChat(ctx, chatID) (bool, error)

// Message operations
GetMessages(ctx, chatID) ([]Message, error)
CreateMessage(ctx, chatID, role, content, ...) (*Message, error)
SaveAssistantMessage(ctx, chatID, model, content, tokenCount) (*Message, error)

// Label operations
ListLabels(ctx) ([]Label, error)
CreateLabel(ctx, name, color) (*Label, error)
AssignLabel(ctx, chatID, labelID) error

// Tool call operations
SaveToolCallRecord(ctx, messageID, record) error
ListToolCallsForChat(ctx, chatID) ([]ToolCallRecord, error)
```

---

### Seam: OpenRouter Client

**Location**: `api/integrations/openrouter.go`

**Purpose**: Isolates OpenRouter API communication for chat completions.

**Substitution Points**:
- For **tests**: Mock the HTTP client or inject a test client
- For **different AI providers**: Create alternative implementation with same interface

**Key Operations**:
```go
NewOpenRouterClient() (*OpenRouterClient, error)
CreateCompletion(ctx, request) (*http.Response, error)
ParseNonStreamingResponse(body) (*OpenRouterResponse, error)
AvailableModels() []ModelInfo
```

**Configuration**:
- `OPENROUTER_API_KEY` - Required for production

---

### Seam: Ollama Client

**Location**: `api/integrations/ollama.go`

**Purpose**: Local AI for privacy-preserving auto-naming.

**Substitution Points**:
- For **tests**: Mock the client or inject a test Ollama instance
- For **offline use**: Graceful fallback to "New Conversation"

**Key Operations**:
```go
NewOllamaClient() *OllamaClient
GenerateChatName(ctx, conversationSummary) (string, error)
IsAvailable(ctx) bool
```

**Configuration**:
- `OLLAMA_BASE_URL` - Custom Ollama URL
- `OLLAMA_NAMING_MODEL` - Model for naming (default: llama3.1:8b)

---

### Seam: Agent Manager Client

**Location**: `api/integrations/agent_manager.go`

**Purpose**: HTTP client for dispatching coding tasks to Claude Code, Codex, or OpenCode agents.

**Substitution Points**:
- For **tests**: Mock the HTTP calls to agent-manager
- For **different runners**: Agent-manager abstracts runner selection

**Key Operations**:
```go
NewAgentManagerClient() (*AgentManagerClient, error)
SpawnCodingAgent(ctx, task, runnerType, workspace, timeout) (map, error)
CheckAgentStatus(ctx, runID) (map, error)
StopAgent(ctx, runID) (map, error)
ListActiveAgents(ctx) (interface{}, error)
GetAgentDiff(ctx, runID) (map, error)
ApproveAgentChanges(ctx, runID) (map, error)
```

**Configuration**:
- `AGENT_MANAGER_API_URL` - Direct URL override
- Falls back to `vrooli scenario port agent-manager API_PORT`

---

### Seam: Tool Executor

**Location**: `api/integrations/tool_executor.go`

**Purpose**: Routes tool calls from AI to appropriate integrations. Separated from agent_manager.go for clear responsibility boundaries.

**Substitution Points**:
- For **tests**: Inject mock AgentManagerClient
- For **new tools**: Add cases to `ExecuteTool` switch
- For **different scenarios**: Add new integration clients

**Key Interface**:
```go
type ToolExecutor struct {
    agentManager *AgentManagerClient
}

func NewToolExecutor() *ToolExecutor
func (e *ToolExecutor) ExecuteTool(ctx, chatID, toolCallID, toolName, args) (*ToolCallRecord, error)
```

**Current Tools**:
| Tool | Description |
|------|-------------|
| `spawn_coding_agent` | Create new coding agent run |
| `check_agent_status` | Get status of running agent |
| `stop_agent` | Stop a running agent |
| `list_active_agents` | List all running agents |
| `get_agent_diff` | Get code changes from agent |
| `approve_agent_changes` | Apply agent's changes |

---

## UI Architecture

```
ui/src/
├── App.tsx                    # Main app, layout orchestration
├── hooks/
│   ├── useChats.ts           # Main chat orchestration (delegates to focused hooks)
│   ├── useCompletion.ts      # AI streaming and tool call state
│   ├── useLabels.ts          # Label CRUD operations
│   └── useKeyboardShortcuts.ts
├── lib/
│   └── api.ts                # API client (clean seam for all HTTP)
├── components/
│   ├── chat/                 # ChatView, ChatList, MessageList, etc.
│   ├── labels/               # LabelManager
│   ├── layout/               # Sidebar
│   ├── settings/             # Settings, KeyboardShortcuts
│   └── ui/                   # Generic UI components
```

### Seam: API Client

**Location**: `ui/src/lib/api.ts`

**Purpose**: All HTTP communication with the backend API.

**Substitution Points**:
- For **unit tests**: Mock individual functions
- For **E2E tests**: Use MSW (Mock Service Worker)
- For **offline mode**: Implement a local storage adapter

**Key Functions**:
```typescript
// Chat operations
fetchChats(options?) -> Promise<Chat[]>
fetchChat(id) -> Promise<ChatWithMessages>
createChat(data?) -> Promise<Chat>
updateChat(id, data) -> Promise<Chat>
deleteChat(id) -> Promise<void>
addMessage(chatId, data) -> Promise<Message>

// AI operations
completeChat(chatId, options?) -> Promise<Message | void>
autoNameChat(chatId) -> Promise<Chat>
fetchModels() -> Promise<Model[]>
fetchTools() -> Promise<ToolDefinition[]>

// Label operations
fetchLabels() -> Promise<Label[]>
createLabel(data) -> Promise<Label>
assignLabel(chatId, labelId) -> Promise<void>
```

---

### Seam: useCompletion Hook

**Location**: `ui/src/hooks/useCompletion.ts`

**Purpose**: Encapsulates AI completion complexity—streaming, tool calls, state.

**Substitution Points**:
- For **tests**: Mock `completeChat` or provide test event sequences
- For **different streaming protocols**: Swap event handling

**State**:
```typescript
interface CompletionState {
  isGenerating: boolean;
  streamingContent: string;
  activeToolCalls: ActiveToolCall[];
}
```

---

### Seam: useLabels Hook

**Location**: `ui/src/hooks/useLabels.ts`

**Purpose**: Isolates label CRUD from main chat logic.

**Substitution Points**:
- For **tests**: Mock React Query or API functions

---

## Testing Strategy

### API (Go)

| Layer | Test Approach |
|-------|---------------|
| **Handlers** | HTTP test with real `Repository` on test DB |
| **Repository** | Direct SQL tests against test DB |
| **Integrations** | Mock HTTP responses, test client logic |
| **Domain** | Unit tests for validation functions |

Example test setup:
```go
func setupTestServer(t *testing.T) *TestServer {
    db := connectTestDB(t)
    repo := persistence.NewRepository(db)
    repo.InitSchema(context.Background())
    h := handlers.New(repo, integrations.NewOllamaClient())
    router := mux.NewRouter()
    h.RegisterRoutes(router)
    return &TestServer{repo: repo, router: router, handlers: h}
}
```

### UI (React/TypeScript)

| Layer | Test Approach |
|-------|---------------|
| **Components** | Vitest + Testing Library |
| **Hooks** | Mock API functions, test state transitions |
| **API Client** | Mock fetch or use MSW |
| **E2E** | Playwright against running scenario |

---

## Adding New Features

### New Tool

1. Add tool definition in `api/integrations/tools.go`
2. Add case in `ToolExecutor.ExecuteTool()` (`api/integrations/tool_executor.go`)
3. Update UI tool call display if needed (`ui/src/components/chat/MessageList.tsx`)

### New Integration

1. Create client in `api/integrations/new_service.go`
2. Add to Handlers struct and wire in `handlers/handlers.go`
3. If UI needs it, add API functions and types in `ui/src/lib/api.ts`

### New Chat Feature

1. Add domain types if needed in `api/domain/types.go`
2. Add repository methods in `api/persistence/repository.go`
3. Add HTTP handlers in appropriate `api/handlers/*.go` file
4. Add API functions in `ui/src/lib/api.ts`
5. Update hooks (`useChats.ts` or create focused hook)
6. Update components

### New API Endpoint

1. Add handler method to appropriate file in `api/handlers/`
2. Register route in `api/handlers/handlers.go` `RegisterRoutes()`
3. Add corresponding API function in `ui/src/lib/api.ts`

---

## Configuration Reference

| Variable | Layer | Purpose |
|----------|-------|---------|
| `API_PORT` | API | Server listen port |
| `DATABASE_URL` | API | PostgreSQL connection |
| `OPENROUTER_API_KEY` | API | OpenRouter authentication |
| `OLLAMA_BASE_URL` | API | Custom Ollama URL |
| `OLLAMA_NAMING_MODEL` | API | Model for auto-naming |
| `AGENT_MANAGER_API_URL` | API | Agent-manager service URL |
| `CORS_ALLOWED_ORIGINS` | API | Allowed CORS origins |
| `UI_PORT` | UI | Vite dev server port |

---

## Architectural Improvements History

### 2025-12-25: Screaming Architecture Refactor

**Changes Made**:
1. **Created `handlers/` package** - Split 968-line main.go into focused handler files organized by domain:
   - `handlers.go` (84 lines) - Shared types, route registration, response helpers
   - `health.go` (26 lines) - Health check
   - `chat.go` (119 lines) - Chat CRUD
   - `message.go` (85 lines) - Messages and state toggles
   - `label.go` (130 lines) - Label operations
   - `ai.go` (282 lines) - AI completion and streaming

2. **Simplified main.go** - Reduced from 968 to ~140 lines, now only handles server wiring

3. **Extracted `tool_executor.go`** - Separated ToolExecutor from agent_manager.go to enforce single responsibility:
   - `agent_manager.go` - HTTP client for agent-manager service
   - `tool_executor.go` - Tool call routing and execution logic

**Benefits**:
- Codebase structure now "screams" its purpose (chat management, AI completion, labels)
- Clear responsibility boundaries make testing easier
- New developers can quickly find where to add features
- Smaller, focused files are easier to review and maintain
