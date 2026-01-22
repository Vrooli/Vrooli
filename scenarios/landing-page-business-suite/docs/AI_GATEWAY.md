# LPBS AI Gateway

The LPBS AI Gateway provides centralized AI access with credit management for all Vrooli applications. It receives AI requests, atomically checks and charges credits, calls AI providers (OpenRouter), and returns responses.

## Architecture

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                         REQUEST FLOW                                         │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                              │
│  BAS (or other app)                                                          │
│       │                                                                      │
│       ├─► User provides API key? ─► YES ─► BYOK Provider ─► OpenRouter      │
│       │                                    (no credits)       directly       │
│       │                                                                      │
│       └─► NO ─► VrooliProvider ─► LPBS AI Gateway                           │
│                                        │                                     │
│                                        ├─► Check user auth (JWT)            │
│                                        ├─► Check credits (atomic)           │
│                                        ├─► Call OpenRouter                  │
│                                        ├─► Charge credits                   │
│                                        └─► Return response (stream/full)    │
│                                                                              │
└─────────────────────────────────────────────────────────────────────────────┘
```

## API Endpoints

### POST /api/v1/ai/chat

Non-streaming chat completion.

**Authentication:** Required (User JWT in `Authorization: Bearer <token>` header)

**Request:**
```json
{
  "model": "openai/gpt-4o-mini",
  "messages": [
    {"role": "system", "content": "You are a helpful assistant."},
    {"role": "user", "content": "Hello!"}
  ],
  "max_tokens": 1000,
  "metadata": {
    "app_bundle_key": "browser-automation-studio",
    "operation": "ai.analysis"
  }
}
```

**Response:**
```json
{
  "id": "chatcmpl-abc123",
  "model": "openai/gpt-4o-mini",
  "content": "Hello! How can I help you today?",
  "prompt_tokens": 15,
  "completion_tokens": 10,
  "total_tokens": 25,
  "credits_charged": 150000,
  "finish_reason": "stop"
}
```

### POST /api/v1/ai/stream

Streaming chat completion via Server-Sent Events (SSE).

**Authentication:** Required (User JWT)

**Request:** Same as `/api/v1/ai/chat`

**Response:** Server-Sent Events stream

```
data: {"type":"chunk","content":"Hello"}

data: {"type":"chunk","content":"! How can"}

data: {"type":"chunk","content":" I help you?"}

data: {"type":"done","usage":{"prompt_tokens":15,"completion_tokens":10,"total_tokens":25,"credits_charged":150000}}
```

**Event Types:**
- `chunk`: Contains partial response content
- `done`: Final event with usage statistics
- `error`: Error event if something fails mid-stream

### GET /api/v1/ai/models

List available AI models.

**Authentication:** None required

**Response:**
```json
{
  "models": [
    "openai/gpt-4o",
    "openai/gpt-4o-mini",
    "anthropic/claude-3.5-sonnet",
    "anthropic/claude-3-haiku",
    "google/gemini-pro-1.5",
    "google/gemini-flash-1.5"
  ]
}
```

### GET /api/v1/ai/usage

Get AI usage statistics for the current user.

**Authentication:** Required (User JWT)

**Response:**
```json
{
  "user_identity": "user@example.com",
  "tier": "pro",
  "billing_period": "2026-01",
  "reset_date": "2026-02-01T00:00:00Z",
  "ai_credits_used": 5000000,
  "ai_credits_limit": 50000000,
  "ai_credits_remaining": 45000000,
  "display": {
    "used": 50.0,
    "limit": 500.0,
    "remaining": 450.0,
    "unit": "credits"
  }
}
```

### GET /api/v1/ai/health

Health check endpoint.

**Authentication:** None required

**Response:**
```json
{
  "healthy": true
}
```

Or if unhealthy:
```json
{
  "healthy": false,
  "error": "no OpenRouter API key configured"
}
```

## Error Responses

All error responses follow this format:

```json
{
  "error": "Error message",
  "error_type": "error_code"
}
```

**Error Types:**

| Code | HTTP Status | Description |
|------|-------------|-------------|
| `unauthorized` | 401 | Missing or invalid JWT token |
| `insufficient_credits` | 402 | User doesn't have enough credits |
| `validation` | 400 | Invalid request (bad model, missing fields, etc.) |
| `rate_limited` | 429 | Too many requests (60/minute per user) |
| `server_error` | 500/502/503 | Internal error or AI provider error |

## Credit Charging

### Atomic Operation

Credits are checked and charged atomically using database transactions with row-level locking (`SELECT FOR UPDATE`). This prevents TOCTOU (time-of-check-time-of-use) race conditions where concurrent requests could exceed limits.

### Cost Calculation

Costs are calculated based on token usage with model-specific pricing:

| Model | Input (per 1M tokens) | Output (per 1M tokens) |
|-------|----------------------|------------------------|
| openai/gpt-4o | $2.50 | $10.00 |
| openai/gpt-4o-mini | $0.15 | $0.60 |
| anthropic/claude-3.5-sonnet | $3.00 | $15.00 |
| anthropic/claude-3-haiku | $0.25 | $1.25 |
| google/gemini-pro-1.5 | $1.25 | $5.00 |
| google/gemini-flash-1.5 | $0.075 | $0.30 |

### Estimation vs Actual

For non-streaming requests:
1. **Pre-check:** Estimate cost based on input message length (with 1.5x safety margin)
2. **Reserve:** Atomically check limit and charge estimated cost
3. **Execute:** Call OpenRouter
4. **Adjust:** Refund difference if actual < estimated, charge extra if actual > estimated

For streaming requests:
1. **Reserve:** Create credit reservation atomically (prevents TOCTOU races)
2. **Stream:** Send response chunks to client via SSE
3. **Finalize:** Mark reservation as finalized with actual usage

### Streaming Credit Reservation System

Streaming requests use a reservation-based system to prevent race conditions where concurrent requests could exceed credit limits:

```
┌─────────────────┐     ┌─────────────────┐     ┌─────────────────┐
│ 1. Reserve      │────►│ 2. Stream       │────►│ 3. Finalize     │
│    Credits      │     │    Response     │     │    Reservation  │
└─────────────────┘     └─────────────────┘     └─────────────────┘
      │                       │                       │
      ▼                       ▼                       ▼
 Check limit +          Forward chunks         Mark reservation
 create pending         to client via          as finalized +
 reservation            SSE events             record actual usage
```

**Key behaviors:**
- Reservations are created atomically with row-level locking (`SELECT FOR UPDATE`)
- Pending reservations count toward effective usage for subsequent requests
- Reservations auto-expire after 10 minutes (background cleanup every 2 minutes)
- If stream fails, reservation is released (no usage recorded)
- If finalization fails, falls back to direct usage recording

**Why small overspend windows are acceptable:**

| Concern | Mitigation |
|---------|------------|
| Actual cost > reserved | Estimation includes 1.5x safety margin |
| User didn't consent | User authorized estimated amount, showing intent |
| Can't know final cost | Impossible to know completion tokens before streaming |
| Alternative is worse | Pre-charging max_tokens would significantly overcharge |

The reservation system prevents the more serious issue: concurrent requests from multiple sessions exceeding credit limits (TOCTOU race condition).

## Rate Limiting

- **60 requests per minute** per user
- **120 requests per minute** per IP address (defense in depth)
- Applies to both `/chat` and `/stream` endpoints
- Returns `429 Too Many Requests` when exceeded

The IP-based limit is more permissive than user-based to allow multiple users on corporate networks while still preventing single-IP abuse.

## Input Validation

| Field | Constraint |
|-------|------------|
| model | Must be in allowed models list |
| messages | At least 1 message required, max 100 |
| message.role | Must be "user", "assistant", or "system" |
| message.content | Max 100KB per message |
| max_tokens | 0-16384 (optional) |

## Integration from BAS

BAS uses the `VrooliProvider` to send requests to LPBS:

```go
provider := ai.NewVrooliProvider(ai.VrooliProviderOptions{
    Logger:    logger,
    APIURL:    "https://vrooli.com",  // or "http://localhost:15000" for local
    Model:     "openai/gpt-4o-mini",
    AuthToken: userJWTToken,  // User's LPBS JWT token
})

if provider.IsAvailable(ctx) {
    response, err := provider.ExecutePrompt(ctx, "Your prompt here")
    if errors.Is(err, ai.ErrInsufficientCredits) {
        // Handle insufficient credits
    }
}
```

## Configuration

### LPBS Side

Requires an OpenRouter API key stored in the `api_keys` table:

```bash
# Store the API key via the admin UI or API
POST /api/v1/api-keys
{
  "provider": "openrouter",
  "key": "sk-or-..."
}
```

### BAS Side

Configure in environment or service.json:

```bash
BAS_AI_VROOLI_API_URL=https://vrooli.com
```

The auth token is obtained when a user authenticates with LPBS.

## Security

1. **Authentication:** All AI endpoints require valid user JWT
2. **Rate Limiting:** Per-user rate limiting prevents abuse
3. **Input Validation:** Strict validation of all request parameters
4. **Credit Isolation:** Users can only spend their own credits
5. **API Key Security:** OpenRouter key is encrypted at rest
