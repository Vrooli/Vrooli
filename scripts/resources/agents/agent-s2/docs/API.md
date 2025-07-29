# Agent S2 API Reference

Agent S2 provides a comprehensive RESTful API for both core automation and AI-powered interactions. The API operates on two layers: **Core Automation** (always available) and **AI Intelligence** (requires API key).

> **💡 Recommended Usage**: Use the `manage.sh` script for most operations. This guide shows direct API usage for advanced integration scenarios.

## Base URL

- **Default**: `http://localhost:4113`
- **Recommended Access**: `./manage.sh --action usage --usage-type [type]`
- **Health Check**: `./manage.sh --action status` (or `GET /health`)
- **Capabilities**: `GET /capabilities`

## Authentication

Agent S2 currently uses API key authentication for AI features only. Core automation features require no authentication.

- **AI Features**: Require `ANTHROPIC_API_KEY` or `OPENAI_API_KEY` environment variable
- **Core Features**: No authentication required

## Core Automation API

### Health Check

**Recommended Method:**
```bash
# Get comprehensive status information
./manage.sh --action status
```

**Direct API (for integrations):**
```http
GET /health
```

Returns service health status and display information.

**Response:**
```json
{
  "status": "healthy",
  "display": "running",
  "ai_available": true,
  "mode": "sandbox",
  "uptime": 3600
}
```

### Get Capabilities

```http
GET /capabilities
```

Returns supported tasks and display configuration.

**Response:**
```json
{
  "ai_available": true,
  "mode": "sandbox",
  "display": {
    "width": 1920,
    "height": 1080
  },
  "supported_actions": ["screenshot", "click", "type", "ai_action"]
}
```

### Take Screenshot

**Recommended Method:**
```bash
# Test screenshot functionality with examples
./manage.sh --action usage --usage-type screenshot
```

**Direct API (for integrations):**
```http
POST /screenshot?format={format}&quality={quality}
```

**Query Parameters:**
- `format`: "png" or "jpeg" (default: "png")
- `quality`: 1-100 for JPEG (default: 95)

**Request Body (optional):**
```json
[x, y, width, height]  // Region array
```

**Direct API Examples (use manage.sh instead when possible):**
```bash
# Full screen PNG
curl -X POST "http://localhost:4113/screenshot?format=png"

# Region screenshot
curl -X POST "http://localhost:4113/screenshot?format=png" \
  -H "Content-Type: application/json" \
  -d '[100, 100, 500, 400]'

# JPEG with quality
curl -X POST "http://localhost:4113/screenshot?format=jpeg&quality=85"
```

### Mouse Operations

**Recommended Method:**
```bash
# Test automation capabilities including mouse operations
./manage.sh --action usage --usage-type automation
```

**Direct API (for integrations):**

#### Click
```http
POST /mouse/click
```

**Request Body:**
```json
{
  "x": 500,
  "y": 300,
  "button": "left|right|middle",
  "clicks": 1
}
```

#### Move
```http
POST /mouse/move
```

**Request Body:**
```json
{
  "x": 600,
  "y": 400,
  "duration": 0.5  // Optional animation duration
}
```

#### Get Position
```http
GET /mouse/position
```

**Response:**
```json
{
  "x": 500,
  "y": 300
}
```

### Keyboard Operations

#### Type Text
```http
POST /keyboard/type
```

**Request Body:**
```json
{
  "text": "Hello World!",
  "interval": 0.1  // Delay between keystrokes
}
```

#### Press Keys
```http
POST /keyboard/press
```

**Request Body:**
```json
{
  "keys": ["ctrl", "c"]  // Key combination
}
```

### Task Management

#### Get Task Status
```http
GET /tasks/{task_id}
```

#### List All Tasks
```http
GET /tasks
```

#### Cancel Task
```http
POST /tasks/cancel/{task_id}
```

## AI Intelligence API

These endpoints require a valid API key (ANTHROPIC_API_KEY or OPENAI_API_KEY).

### Execute AI Action

**Recommended Method:**
```bash
# Test AI planning capabilities with examples
./manage.sh --action usage --usage-type planning
```

**Direct API (for integrations):**
```http
POST /ai/action
```

Execute a natural language task using AI intelligence.

**Request Body:**
```json
{
  "task": "Natural language task description",
  "screenshot": "Optional base64 screenshot data",
  "context": {
    "purpose": "Optional context object",
    "constraints": ["list", "of", "constraints"]
  }
}
```

**Direct API Example (use manage.sh instead when possible):**
```bash
curl -X POST http://localhost:4113/ai/action \
  -H "Content-Type: application/json" \
  -d '{
    "task": "open a text editor and write hello world",
    "context": {"purpose": "demonstration"}
  }'
```

**Response:**
```json
{
  "success": true,
  "task_id": "uuid",
  "result": "Task completed successfully",
  "actions_taken": [
    {"action": "screenshot", "timestamp": "2024-01-01T12:00:00Z"},
    {"action": "click", "coordinates": [100, 200], "timestamp": "2024-01-01T12:00:01Z"}
  ]
}
```

### Generate AI Plan

```http
POST /ai/plan
```

Generate a step-by-step plan for achieving a goal.

**Request Body:**
```json
{
  "goal": "High-level goal to achieve",
  "constraints": ["list", "of", "constraints"]
}
```

**Example:**
```bash
curl -X POST http://localhost:4113/ai/plan \
  -H "Content-Type: application/json" \
  -d '{
    "goal": "organize my desktop files by type",
    "constraints": ["do not delete anything", "keep downloads folder visible"]
  }'
```

**Response:**
```json
{
  "plan": [
    {
      "step": 1,
      "action": "take_screenshot",
      "description": "Capture current desktop state"
    },
    {
      "step": 2,
      "action": "analyze_files",
      "description": "Identify file types and locations"
    }
  ],
  "estimated_duration": 120,
  "complexity": "medium"
}
```

### Analyze Screen

```http
POST /ai/analyze
```

Analyze screen content and answer questions about what's visible.

**Request Body:**
```json
{
  "question": "What do you want to know about the screen?",
  "screenshot": "Optional base64 screenshot data"
}
```

**Example:**
```bash
curl -X POST http://localhost:4113/ai/analyze \
  -H "Content-Type: application/json" \
  -d '{
    "question": "What applications are currently running?"
  }'
```

**Response:**
```json
{
  "analysis": "I can see Firefox browser, Terminal, and VS Code running...",
  "elements_detected": [
    {"type": "application", "name": "Firefox", "location": [100, 50]},
    {"type": "application", "name": "Terminal", "location": [500, 100]}
  ]
}
```

## Stealth Mode API

The stealth mode API provides advanced anti-bot detection capabilities including browser fingerprint randomization, session persistence, and various evasion techniques.

### Get Stealth Status

**Recommended Method:**
```bash
# Check stealth mode status and configuration
./manage.sh --action configure-stealth
```

**Direct API:**
```http
GET /stealth/status
```

Returns current stealth mode configuration and status.

**Response:**
```json
{
  "enabled": true,
  "features_enabled": [
    "fingerprint_randomization",
    "webdriver_hiding", 
    "user_agent_rotation",
    "session_persistence"
  ],
  "current_profile": {
    "id": "residential_profile_123",
    "type": "residential",
    "active": true
  },
  "session_storage": {
    "path": "/data/sessions",
    "encryption_enabled": true
  }
}
```

### Configure Stealth Settings

**Recommended Method:**
```bash
# Enable stealth mode
./manage.sh --action configure-stealth --stealth-enabled yes

# Configure specific feature
./manage.sh --action configure-stealth --stealth-feature "fingerprint_randomization=true"
```

**Direct API:**
```http
POST /stealth/configure
```

**Request Body:**
```json
{
  "enabled": true,
  "features": {
    "fingerprint_randomization": true,
    "webdriver_hiding": true,
    "user_agent_rotation": true,
    "session_persistence": true,
    "canvas_noise": true,
    "webgl_randomization": true
  }
}
```

**Response:**
```json
{
  "success": true,
  "config": {
    "enabled": true,
    "fingerprint_randomization": true,
    "webdriver_hiding": true,
    "user_agent_rotation": true,
    "session_persistence": true,
    "canvas_noise": true,
    "webgl_randomization": true
  }
}
```

### Test Stealth Effectiveness

**Recommended Method:**
```bash
# Test stealth mode with default detection site
./manage.sh --action test-stealth

# Test with custom URL
./manage.sh --action test-stealth --stealth-url https://fingerprint.com/demo
```

**Direct API:**
```http
POST /stealth/test
```

**Request Body:**
```json
{
  "url": "https://bot.sannysoft.com/"
}
```

**Response:**
```json
{
  "test_url": "https://bot.sannysoft.com/",
  "timestamp": "2024-01-01T12:00:00Z",
  "fingerprint_changed": true,
  "webdriver_detected": false,
  "stealth_effective": true,
  "details": {
    "user_agent": "Mozilla/5.0 (Windows NT 10.0; Win64; x64)...",
    "canvas_fingerprint": "a7b3c9d2e4f1...",
    "webgl_vendor": "Intel Inc."
  }
}
```

### Profile Management

#### Create Profile

**Recommended Method:**
```bash
# Create residential profile
./manage.sh --action create-profile --profile shopping --type residential
```

**Direct API:**
```http
POST /stealth/profile/create
```

**Request Body:**
```json
{
  "profile_id": "shopping_profile",
  "profile_type": "residential"
}
```

**Response:**
```json
{
  "success": true,
  "profile_id": "shopping_profile",
  "profile": {
    "type": "residential",
    "user_agent": "Mozilla/5.0 (Windows NT 10.0; Win64; x64)...",
    "screen_resolution": "1920x1080",
    "hardware": {
      "cpu_cores": 8,
      "memory_gb": 16
    },
    "timezone": "America/New_York",
    "language": "en-US"
  }
}
```

#### List Profiles

**Recommended Method:**
```bash
# List all saved session profiles
./manage.sh --action list-sessions
```

**Direct API:**
```http
GET /stealth/profile/list
```

**Response:**
```json
{
  "success": true,
  "profiles": [
    {
      "id": "shopping_profile",
      "type": "residential",
      "saved_at": "2024-01-01T12:00:00Z",
      "expires_at": "2024-01-31T12:00:00Z",
      "size": 2048,
      "active": true
    }
  ],
  "count": 1
}
```

#### Activate Profile

```http
PUT /stealth/profile/{profile_id}/activate
```

**Response:**
```json
{
  "success": true,
  "profile_id": "shopping_profile",
  "features_enabled": [
    "fingerprint_randomization",
    "session_persistence"
  ],
  "message": "Profile activated successfully"
}
```

#### Delete Profile

```http
DELETE /stealth/profile/{profile_id}
```

**Response:**
```json
{
  "success": true,
  "message": "Profile shopping_profile deleted"
}
```

### Session Management

#### Save Current Session

**Direct API:**
```http
POST /stealth/session/save
```

**Response:**
```json
{
  "success": true,
  "profile_id": "shopping_profile",
  "session_data_saved": true,
  "items_saved": {
    "cookies": 15,
    "local_storage": 8,
    "session_storage": 3
  }
}
```

#### Reset Session

**Recommended Method:**
```bash
# Reset all session data
./manage.sh --action reset-session-data

# Reset session state only
./manage.sh --action reset-session-state
```

**Direct API:**
```http
DELETE /stealth/session/reset
```

**Response:**
```json
{
  "success": true,
  "message": "Session state reset",
  "items_cleared": {
    "cookies": 15,
    "local_storage": 8,
    "session_storage": 3
  }
}
```

## Error Responses

All endpoints return consistent error responses:

```json
{
  "error": true,
  "message": "Error description",
  "code": "ERROR_CODE",
  "details": {
    "additional": "context"
  }
}
```

### Common Error Codes

- `SERVICE_UNAVAILABLE`: Agent S2 service is not running
- `AI_UNAVAILABLE`: AI features requested but no API key configured
- `INVALID_COORDINATES`: Mouse coordinates out of bounds
- `SCREENSHOT_FAILED`: Display not available or screenshot failed
- `TASK_TIMEOUT`: Operation exceeded timeout limit
- `INVALID_INPUT`: Request body validation failed
- `STEALTH_NOT_INITIALIZED`: Stealth mode not properly initialized
- `PROFILE_NOT_FOUND`: Requested stealth profile does not exist
- `SESSION_SAVE_FAILED`: Unable to save session data
- `ENCRYPTION_ERROR`: Session data encryption/decryption failed

## Rate Limits

- **Core Automation**: No rate limiting
- **AI Endpoints**: 60 requests per minute per client
- **Screenshot**: 10 requests per second

## Response Formats

All responses are JSON unless otherwise specified. Screenshots are returned as base64-encoded image data.

## SDK Integration

**Recommended Method (Testing):**
```bash
# Test all capabilities with built-in examples
./manage.sh --action usage --usage-type all
```

**Python SDK (for programmatic access):**

```python
from agent_s2 import AgentS2Client

client = AgentS2Client(base_url="http://localhost:4113")

# Core automation
client.click(500, 300)
client.type_text("Hello World!")
screenshot = client.screenshot()

# AI features (if available)
if client.get_capabilities().get("ai_available"):
    result = client.ai_action(
        task="Take a screenshot and analyze what's on screen"
    )
```

> **💡 Pro Tip**: Use `./manage.sh --action usage --usage-type [type]` to test functionality before implementing in your applications.

## WebSocket API (Beta)

For real-time interactions:

```javascript
const ws = new WebSocket('ws://localhost:4113/ws');

ws.send(JSON.stringify({
  type: 'screenshot_stream',
  fps: 5
}));
```

See [Python SDK Documentation](../examples/) for complete examples and usage patterns.