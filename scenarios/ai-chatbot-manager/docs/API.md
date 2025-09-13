# AI Chatbot Manager API Documentation

## Overview

The AI Chatbot Manager API provides endpoints for creating, managing, and interacting with AI-powered chatbots. All API endpoints follow RESTful conventions and use JSON for request and response bodies.

## Base URL

```
http://localhost:{API_PORT}/api/v1
```

The API port is dynamically allocated by the Vrooli lifecycle system. Use `vrooli scenario port ai-chatbot-manager API_PORT` to find the current port.

## Authentication

Currently, the API does not require authentication for development purposes. Production deployments should implement appropriate authentication mechanisms.

## Endpoints

### Health Check

#### `GET /health`

Check the health status of the API server.

**Response:**
```json
{
  "status": "healthy",
  "version": "1.0.0",
  "database": "connected",
  "timestamp": "2024-01-09T12:00:00Z"
}
```

---

### Chatbot Management

#### `POST /api/v1/chatbots`

Create a new chatbot configuration.

**Request Body:**
```json
{
  "name": "Customer Support Bot",
  "description": "24/7 customer support assistant",
  "personality": "You are a helpful and friendly customer support agent.",
  "knowledge_base": "Company policies, product information, FAQ answers...",
  "model_config": {
    "model": "llama3.2",
    "temperature": 0.7,
    "max_tokens": 500
  },
  "widget_config": {
    "primaryColor": "#007bff",
    "position": "bottom-right",
    "title": "Support Chat",
    "welcomeMessage": "Hello! How can I help you today?"
  }
}
```

**Response:**
```json
{
  "chatbot": {
    "id": "uuid-string",
    "name": "Customer Support Bot",
    "description": "24/7 customer support assistant",
    "personality": "You are a helpful and friendly customer support agent.",
    "knowledge_base": "Company policies, product information, FAQ answers...",
    "model_config": {...},
    "widget_config": {...},
    "is_active": true,
    "created_at": "2024-01-09T12:00:00Z",
    "updated_at": "2024-01-09T12:00:00Z"
  },
  "widget_embed_code": "<!-- HTML embed code -->"
}
```

#### `GET /api/v1/chatbots`

List all chatbots.

**Query Parameters:**
- `active_only` (boolean): Filter to show only active chatbots

**Response:**
```json
[
  {
    "id": "uuid-string",
    "name": "Customer Support Bot",
    "description": "24/7 customer support assistant",
    "is_active": true,
    "created_at": "2024-01-09T12:00:00Z",
    "updated_at": "2024-01-09T12:00:00Z"
  }
]
```

#### `GET /api/v1/chatbots/{id}`

Get a specific chatbot by ID.

**Response:**
```json
{
  "id": "uuid-string",
  "name": "Customer Support Bot",
  "description": "24/7 customer support assistant",
  "personality": "You are a helpful and friendly customer support agent.",
  "knowledge_base": "Company policies, product information, FAQ answers...",
  "model_config": {...},
  "widget_config": {...},
  "is_active": true,
  "created_at": "2024-01-09T12:00:00Z",
  "updated_at": "2024-01-09T12:00:00Z"
}
```

#### `PUT /api/v1/chatbots/{id}` or `PATCH /api/v1/chatbots/{id}`

Update a chatbot configuration.

**Request Body (all fields optional):**
```json
{
  "name": "Updated Bot Name",
  "description": "Updated description",
  "personality": "Updated personality",
  "knowledge_base": "Updated knowledge",
  "model_config": {...},
  "widget_config": {...},
  "is_active": false
}
```

**Response:**
```json
{
  "message": "Chatbot updated successfully",
  "chatbot": {...}
}
```

#### `DELETE /api/v1/chatbots/{id}`

Delete (soft delete) a chatbot.

**Response:**
```json
{
  "message": "Chatbot deleted successfully"
}
```

---

### Chat Interaction

#### `POST /api/v1/chat/{chatbot_id}`

Send a message to a chatbot and receive a response.

**Request Body:**
```json
{
  "message": "Hello, I need help with my order",
  "session_id": "unique-session-id",
  "context": {
    "user_id": "optional-user-id",
    "metadata": {...}
  }
}
```

**Response:**
```json
{
  "response": "I'd be happy to help you with your order. Could you please provide your order number?",
  "confidence": 0.95,
  "should_escalate": false,
  "lead_qualification": {
    "is_qualified": true,
    "score": 0.8
  },
  "session_id": "unique-session-id",
  "conversation_id": "uuid-string"
}
```

---

### WebSocket Connection

#### `WS /api/v1/ws/{chatbot_id}`

Establish a WebSocket connection for real-time chat.

**Connection URL:**
```
ws://localhost:{API_PORT}/api/v1/ws/{chatbot_id}
```

**Message Format (Client to Server):**
```json
{
  "message": "User message text",
  "session_id": "unique-session-id"
}
```

**Message Format (Server to Client):**
```json
{
  "type": "message",
  "payload": {
    "response": "Bot response text",
    "confidence": 0.95,
    "timestamp": "2024-01-09T12:00:00Z"
  }
}
```

**Error Format:**
```json
{
  "type": "error",
  "payload": {
    "message": "Error description",
    "code": "ERROR_CODE"
  }
}
```

---

### Analytics

#### `GET /api/v1/analytics/{chatbot_id}`

Retrieve analytics data for a chatbot.

**Query Parameters:**
- `start_date` (string): Start date for analytics period (ISO 8601)
- `end_date` (string): End date for analytics period (ISO 8601)
- `metrics` (array): Specific metrics to retrieve

**Response:**
```json
{
  "total_conversations": 1234,
  "total_messages": 5678,
  "leads_captured": 89,
  "avg_conversation_length": 12.5,
  "engagement_score": 85.3,
  "conversion_rate": 7.2,
  "top_intents": [
    {
      "intent": "order_status",
      "count": 234,
      "percentage": 18.9
    }
  ],
  "period": {
    "start": "2024-01-01T00:00:00Z",
    "end": "2024-01-09T23:59:59Z"
  }
}
```

---

### Widget Management

#### `GET /api/v1/chatbots/{id}/widget`

Get the embeddable widget code for a chatbot.

**Response:**
```html
<!-- AI Chatbot Manager Widget -->
<script>
  window.CHATBOT_API_URL = 'http://localhost:${API_PORT}';
</script>
<script src="http://localhost:${API_PORT}/api/v1/widget.js"></script>
<script>
  // Widget initialization code
</script>
```

#### `GET /api/v1/widget.js`

Serve the widget JavaScript library.

**Response:**
- Content-Type: `application/javascript`
- Returns the widget JavaScript code

---

## Error Responses

All endpoints may return error responses in the following format:

```json
{
  "error": {
    "code": "ERROR_CODE",
    "message": "Human-readable error message",
    "details": {...}
  }
}
```

### Common HTTP Status Codes

- `200 OK` - Request successful
- `201 Created` - Resource created successfully
- `204 No Content` - Request successful with no content to return
- `400 Bad Request` - Invalid request parameters or body
- `404 Not Found` - Resource not found
- `500 Internal Server Error` - Server error

---

## Rate Limiting

The API implements rate limiting to prevent abuse:
- **Limit:** 100 requests per minute per IP address
- **Headers:** Rate limit information is included in response headers:
  - `X-RateLimit-Limit`: Maximum requests allowed
  - `X-RateLimit-Remaining`: Requests remaining
  - `X-RateLimit-Reset`: Time when limit resets (Unix timestamp)

---

## CORS

The API supports Cross-Origin Resource Sharing (CORS) to allow widget embedding on external websites. The following headers are included:
- `Access-Control-Allow-Origin: *`
- `Access-Control-Allow-Methods: GET, POST, PUT, PATCH, DELETE, OPTIONS`
- `Access-Control-Allow-Headers: Content-Type, Authorization`

---

## Versioning

The API uses URL path versioning. The current version is `v1`. Future versions will be available at `/api/v2`, `/api/v3`, etc., with backward compatibility maintained for deprecated versions.

---

## Examples

### Create a Chatbot with cURL

```bash
curl -X POST http://localhost:${API_PORT}/api/v1/chatbots \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Sales Assistant",
    "personality": "You are a helpful sales assistant.",
    "knowledge_base": "Product catalog and pricing information"
  }'
```

### Send a Chat Message

```bash
curl -X POST http://localhost:${API_PORT}/api/v1/chat/{chatbot_id} \
  -H "Content-Type: application/json" \
  -d '{
    "message": "What products do you offer?",
    "session_id": "test-session-123"
  }'
```

### Connect via WebSocket (JavaScript)

```javascript
const ws = new WebSocket('ws://localhost:${API_PORT}/api/v1/ws/{chatbot_id}');

ws.onopen = () => {
  ws.send(JSON.stringify({
    message: 'Hello!',
    session_id: 'web-session-456'
  }));
};

ws.onmessage = (event) => {
  const data = JSON.parse(event.data);
  console.log('Bot response:', data.payload.response);
};
```

---

## SDK and Client Libraries

### JavaScript/TypeScript

```javascript
// Using the embedded widget
window.AIChatbotWidget({
  chatbotId: 'your-chatbot-id',
  config: {
    primaryColor: '#007bff',
    position: 'bottom-right'
  },
  apiUrl: 'http://localhost:${API_PORT}'
});
```

### CLI Usage

```bash
# List all chatbots
ai-chatbot-manager list

# Create a new chatbot
ai-chatbot-manager create --name "Support Bot" --personality "Helpful assistant"

# Test chat
ai-chatbot-manager chat {chatbot_id}

# View analytics
ai-chatbot-manager analytics {chatbot_id} --days 7
```

---

## Support

For issues, questions, or feature requests, please refer to the main README or contact the development team.