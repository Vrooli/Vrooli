# Tool Discovery Protocol v1.0

The Tool Discovery Protocol enables Vrooli scenarios to expose their capabilities to other scenarios in a standardized, discoverable way. This allows scenarios like `agent-inbox` to dynamically discover and use tools from scenarios like `agent-manager`, `research-agent`, or any other tool-providing scenario.

## Overview

```
┌─────────────────────────────────────────────────────────────┐
│                      Consumer Scenario                      │
│                      (e.g., agent-inbox)                    │
│  ┌─────────────────────────────────────────────────────────┐│
│  │              Tool Registry Client                       ││
│  │  - Discovers running scenarios                          ││
│  │  - Fetches tool manifests from each                     ││
│  │  - Merges with user preferences                         ││
│  └─────────────────────────────────────────────────────────┘│
└─────────────────────────────────────────────────────────────┘
         │                    │                    │
         ▼                    ▼                    ▼
┌─────────────────┐  ┌─────────────────┐  ┌─────────────────┐
│  agent-manager  │  │ research-agent  │  │  future-scenario│
│  GET /api/v1/   │  │  GET /api/v1/   │  │  GET /api/v1/   │
│      tools      │  │      tools      │  │      tools      │
└─────────────────┘  └─────────────────┘  └─────────────────┘
```

## Protocol Specification

### Endpoint

```
GET /api/v1/tools
```

Returns the complete tool manifest for the scenario.

### Response Format

```json
{
  "protocol_version": "1.0",
  "scenario": {
    "name": "agent-manager",
    "version": "1.0.0",
    "description": "Manages coding agents for software engineering tasks",
    "base_url": "http://localhost:15000"
  },
  "tools": [
    {
      "name": "spawn_coding_agent",
      "description": "Spawn a coding agent to execute a software engineering task",
      "category": "agent_lifecycle",
      "parameters": {
        "type": "object",
        "properties": {
          "task": {
            "type": "string",
            "description": "Clear description of the coding task"
          },
          "runner_type": {
            "type": "string",
            "enum": ["claude-code", "codex", "opencode"],
            "default": "claude-code"
          }
        },
        "required": ["task"]
      },
      "metadata": {
        "enabled_by_default": true,
        "requires_approval": false,
        "timeout_seconds": 30,
        "rate_limit_per_minute": 10,
        "cost_estimate": "high",
        "long_running": true,
        "idempotent": false,
        "tags": ["coding", "agent", "async"],
        "examples": [
          {
            "description": "Fix a bug in authentication",
            "input": {
              "task": "Fix the login bug where users are logged out after 5 minutes"
            }
          }
        ]
      }
    }
  ],
  "categories": [
    {
      "id": "agent_lifecycle",
      "name": "Agent Lifecycle",
      "description": "Tools for spawning, stopping, and managing agent runs",
      "icon": "play"
    }
  ],
  "generated_at": "2024-01-15T10:30:00Z"
}
```

### Individual Tool Endpoint

```
GET /api/v1/tools/{name}
```

Returns a single tool definition by name. Returns 404 if not found.

## Schema Definitions

### ToolManifest

| Field | Type | Description |
|-------|------|-------------|
| `protocol_version` | string | Protocol version (e.g., "1.0") |
| `scenario` | ScenarioInfo | Metadata about the scenario |
| `tools` | ToolDefinition[] | Array of available tools |
| `categories` | ToolCategory[] | Optional tool groupings |
| `generated_at` | string (ISO 8601) | Manifest generation timestamp |

### ScenarioInfo

| Field | Type | Description |
|-------|------|-------------|
| `name` | string | Scenario identifier (e.g., "agent-manager") |
| `version` | string | Scenario version (semver) |
| `description` | string | Human-readable description |
| `base_url` | string? | API base URL (optional, for remote scenarios) |

### ToolDefinition

| Field | Type | Description |
|-------|------|-------------|
| `name` | string | Unique tool identifier (alphanumeric + underscores) |
| `description` | string | What the tool does (shown to LLM) |
| `category` | string? | Category ID for grouping |
| `parameters` | ToolParameters | Input schema (JSON Schema format) |
| `metadata` | ToolMetadata | Additional orchestration info |

### ToolParameters

Compatible with OpenAI's function-calling format:

| Field | Type | Description |
|-------|------|-------------|
| `type` | string | Always "object" |
| `properties` | object | Parameter definitions |
| `required` | string[]? | Required parameter names |
| `additionalProperties` | boolean? | Allow extra properties |

### ParameterSchema

JSON Schema for individual parameters:

| Field | Type | Description |
|-------|------|-------------|
| `type` | string | JSON type (string, number, integer, boolean, array, object) |
| `description` | string? | Parameter purpose |
| `enum` | string[]? | Allowed values |
| `default` | any? | Default value |
| `items` | ParameterSchema? | Array element schema |
| `properties` | object? | Nested object properties |
| `minimum`/`maximum` | number? | Numeric constraints |
| `minLength`/`maxLength` | int? | String length constraints |
| `pattern` | string? | Regex validation |
| `format` | string? | Semantic format (uri, email, uuid) |

### ToolMetadata

| Field | Type | Description |
|-------|------|-------------|
| `enabled_by_default` | boolean | Active by default |
| `requires_approval` | boolean | Needs human approval before execution |
| `timeout_seconds` | int? | Default timeout |
| `rate_limit_per_minute` | int? | Call frequency limit |
| `cost_estimate` | string? | Relative cost (low, medium, high, variable) |
| `long_running` | boolean? | May take extended time |
| `idempotent` | boolean? | Safe to retry |
| `tags` | string[]? | Additional labels |
| `examples` | ToolExample[]? | Usage examples |

### ToolCategory

| Field | Type | Description |
|-------|------|-------------|
| `id` | string | Category identifier |
| `name` | string | Human-readable name |
| `description` | string? | Category purpose |
| `icon` | string? | Icon identifier for UI |

## Implementation Guide

### For Tool Providers (e.g., agent-manager)

1. Create domain types in `internal/domain/tools.go`:

```go
// Use the provided domain types from agent-manager as reference
```

2. Create a tool registry in `internal/toolregistry/`:

```go
// Registry manages tool providers
type Registry struct {
    providers map[string]ToolProvider
}

// ToolProvider interface for contributing tools
type ToolProvider interface {
    Name() string
    Tools(ctx context.Context) []domain.ToolDefinition
    Categories(ctx context.Context) []domain.ToolCategory
}
```

3. Implement the handler in `internal/handlers/tools.go`:

```go
func (h *ToolsHandler) GetTools(w http.ResponseWriter, r *http.Request) {
    manifest := h.registry.GetManifest(r.Context())
    w.Header().Set("Content-Type", "application/json")
    w.Header().Set("Cache-Control", "public, max-age=60")
    json.NewEncoder(w).Encode(manifest)
}
```

4. Wire everything in `main.go`:

```go
toolReg := toolregistry.NewRegistry(toolregistry.RegistryConfig{
    ScenarioName:        "my-scenario",
    ScenarioVersion:     "1.0.0",
    ScenarioDescription: "My scenario description",
})
toolReg.RegisterProvider(myToolProvider)
```

### For Tool Consumers (e.g., agent-inbox)

1. Discover running scenarios via the lifecycle system
2. Fetch tool manifests from each scenario's `/api/v1/tools` endpoint
3. Cache manifests with TTL (e.g., 60 seconds)
4. Merge tool definitions with user preferences
5. Pass enabled tools to the LLM when making completion requests

## Caching

The `/api/v1/tools` endpoint returns `Cache-Control: public, max-age=60` to enable client-side caching. Consumers should:

- Cache manifest responses for at least 60 seconds
- Use the `generated_at` field for freshness checks
- Refresh cache when scenarios restart

## Versioning

The protocol uses semantic versioning:

- **Major version** (1.x): Breaking changes to response schema
- **Minor version** (1.1): New optional fields, new categories
- **Patch version** (1.0.1): Bug fixes, documentation

Consumers should check `protocol_version` and handle unknown versions gracefully.

## Error Handling

| Status | Meaning |
|--------|---------|
| 200 | Success |
| 404 | Tool not found (for `/tools/{name}`) |
| 503 | Scenario unavailable |

## Reference Implementation

See `scenarios/agent-manager/api/internal/toolregistry/` for the reference implementation:

- `registry.go` - Core registry with provider management
- `agent_tools.go` - Agent-manager tool definitions
- `*_test.go` - Comprehensive test coverage

## Future Considerations

1. **Tool Execution API**: Standardized endpoint for executing tools
2. **WebSocket Updates**: Real-time tool availability notifications
3. **Tool Versioning**: Per-tool version tracking for compatibility
4. **Authentication**: Secure tool discovery for multi-tenant deployments
