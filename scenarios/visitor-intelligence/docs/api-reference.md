# Visitor Intelligence API Reference

Complete API reference for the Visitor Intelligence tracking and analytics system.

## Base URL
```
http://localhost:${API_PORT}/api/v1
```

## Authentication
Currently, the API operates without authentication for simplicity. In production deployments, consider adding API key authentication.

## Rate Limiting
- Tracking endpoint: 1000 requests/minute per IP
- Analytics endpoints: 100 requests/minute per IP
- Profile endpoints: 50 requests/minute per IP

## Error Responses
All endpoints return consistent error structures:

```json
{
    "success": false,
    "error": {
        "code": "VALIDATION_ERROR",
        "message": "Missing required field: fingerprint",
        "details": {
            "field": "fingerprint",
            "requirement": "non-empty string"
        }
    }
}
```

## Endpoints

### POST /visitor/track
Track a visitor event in real-time.

**Request Body:**
```json
{
    "fingerprint": "abc123def456",
    "session_id": "sess_789xyz",
    "scenario": "my-scenario",
    "event_type": "pageview",
    "page_url": "https://example.com/page",
    "timestamp": "2023-12-01T12:00:00Z",
    "properties": {
        "referrer": "https://google.com",
        "utm_source": "google",
        "utm_medium": "cpc",
        "custom_property": "value"
    }
}
```

**Required Fields:**
- `fingerprint` (string): Browser fingerprint hash
- `event_type` (string): Type of event (pageview, click, etc.)
- `scenario` (string): Scenario name generating the event

**Optional Fields:**
- `session_id` (string): Session identifier (auto-generated if not provided)
- `page_url` (string): Current page URL
- `timestamp` (string): ISO 8601 timestamp (defaults to server time)
- `properties` (object): Additional event properties

**Response:**
```json
{
    "success": true,
    "visitor_id": "visitor_abc123",
    "session_id": "session_def456"
}
```

**Status Codes:**
- `201 Created`: Event tracked successfully
- `400 Bad Request`: Invalid request data
- `500 Internal Server Error`: Server error

**Example:**
```bash
curl -X POST http://localhost:8080/api/v1/visitor/track \
  -H "Content-Type: application/json" \
  -d '{
    "fingerprint": "test123",
    "event_type": "pageview",
    "scenario": "my-app",
    "page_url": "http://localhost:3000/dashboard",
    "properties": {
      "page_title": "Dashboard",
      "user_agent": "Mozilla/5.0..."
    }
  }'
```

### GET /visitor/{id}
Retrieve detailed visitor profile and history.

**Path Parameters:**
- `id` (string, required): Visitor ID

**Query Parameters:**
- `include_events` (boolean): Include visitor events (default: false)
- `include_sessions` (boolean): Include session data (default: false)
- `limit` (integer): Limit number of events/sessions (default: 100)

**Response:**
```json
{
    "id": "visitor_abc123",
    "fingerprint": "abc123def456",
    "first_seen": "2023-11-15T10:30:00Z",
    "last_seen": "2023-12-01T15:45:00Z",
    "session_count": 8,
    "identified": true,
    "email": "user@example.com",
    "name": "John Doe",
    "user_agent": "Mozilla/5.0 (Windows NT 10.0; Win64; x64)...",
    "ip_address": "192.168.1.100",
    "timezone": "America/New_York",
    "language": "en-US",
    "screen_resolution": "1920x1080",
    "device_type": "desktop",
    "total_page_views": 47,
    "total_session_duration": 2847,
    "tags": ["high-value", "engaged", "returning"],
    "created_at": "2023-11-15T10:30:00Z",
    "updated_at": "2023-12-01T15:45:00Z",
    "events": [
        {
            "id": "event_xyz789",
            "event_type": "pageview",
            "page_url": "https://example.com/product/123",
            "timestamp": "2023-12-01T15:45:00Z",
            "properties": {
                "product_id": "123",
                "category": "electronics"
            }
        }
    ],
    "sessions": [
        {
            "id": "session_def456",
            "scenario": "my-store",
            "start_time": "2023-12-01T15:00:00Z",
            "end_time": "2023-12-01T15:45:00Z",
            "duration": 2700,
            "page_views": 5,
            "referrer": "https://google.com",
            "utm_source": "google",
            "utm_medium": "cpc"
        }
    ]
}
```

**Status Codes:**
- `200 OK`: Visitor found
- `404 Not Found`: Visitor not found
- `500 Internal Server Error`: Server error

**Example:**
```bash
curl "http://localhost:8080/api/v1/visitor/visitor_abc123?include_events=true&limit=10"
```

### GET /analytics/scenario/{scenario}
Get analytics for a specific scenario.

**Path Parameters:**
- `scenario` (string, required): Scenario name

**Query Parameters:**
- `timeframe` (string): Time period - `1d`, `7d`, `30d`, `90d` (default: `7d`)
- `metrics` (string[]): Specific metrics to include (comma-separated)

**Response:**
```json
{
    "scenario": "my-store",
    "timeframe": "7d",
    "period": {
        "start": "2023-11-24T00:00:00Z",
        "end": "2023-12-01T23:59:59Z"
    },
    "unique_visitors": 1247,
    "total_sessions": 1856,
    "avg_session_duration": 185.5,
    "total_page_views": 8923,
    "bounce_rate": 0.34,
    "identified_visitors": 423,
    "top_pages": [
        {
            "url": "/product/123",
            "page_views": 1234,
            "unique_visitors": 567
        },
        {
            "url": "/category/electronics",
            "page_views": 890,
            "unique_visitors": 445
        }
    ],
    "traffic_sources": {
        "direct": 45.2,
        "organic": 32.1,
        "social": 12.8,
        "paid": 9.9
    },
    "device_breakdown": {
        "desktop": 62.3,
        "mobile": 31.4,
        "tablet": 6.3
    }
}
```

**Status Codes:**
- `200 OK`: Analytics retrieved successfully
- `404 Not Found`: Scenario not found
- `500 Internal Server Error`: Server error

**Example:**
```bash
curl "http://localhost:8080/api/v1/analytics/scenario/my-store?timeframe=30d"
```

### GET /analytics/overview
Get system-wide analytics across all scenarios.

**Query Parameters:**
- `timeframe` (string): Time period - `1d`, `7d`, `30d`, `90d` (default: `7d`)

**Response:**
```json
{
    "timeframe": "7d",
    "total_unique_visitors": 5847,
    "total_sessions": 8932,
    "total_page_views": 34521,
    "avg_session_duration": 167.8,
    "overall_bounce_rate": 0.41,
    "identification_rate": 0.42,
    "top_scenarios": [
        {
            "scenario": "my-store",
            "unique_visitors": 1247,
            "percentage": 21.3
        },
        {
            "scenario": "blog",
            "unique_visitors": 987,
            "percentage": 16.9
        }
    ],
    "growth_metrics": {
        "visitors_growth": 15.6,
        "sessions_growth": 12.3,
        "engagement_growth": 8.7
    }
}
```

**Example:**
```bash
curl "http://localhost:8080/api/v1/analytics/overview?timeframe=30d"
```

### GET /analytics/realtime
Get real-time visitor activity.

**Query Parameters:**
- `limit` (integer): Number of recent events to return (default: 50, max: 200)

**Response:**
```json
{
    "active_visitors": 23,
    "events_last_minute": 145,
    "recent_events": [
        {
            "timestamp": "2023-12-01T15:45:30Z",
            "event_type": "pageview",
            "scenario": "my-store",
            "page_url": "/product/456",
            "visitor_country": "US",
            "device_type": "mobile"
        },
        {
            "timestamp": "2023-12-01T15:45:28Z",
            "event_type": "click",
            "scenario": "blog",
            "page_url": "/article/ai-trends",
            "visitor_country": "CA",
            "device_type": "desktop"
        }
    ],
    "active_scenarios": [
        {
            "scenario": "my-store",
            "active_visitors": 12,
            "events_per_minute": 67
        },
        {
            "scenario": "blog",
            "active_visitors": 8,
            "events_per_minute": 34
        }
    ]
}
```

**Example:**
```bash
curl "http://localhost:8080/api/v1/analytics/realtime?limit=100"
```

### POST /visitor/identify
Identify an anonymous visitor with user information.

**Request Body:**
```json
{
    "fingerprint": "abc123def456",
    "email": "user@example.com",
    "name": "John Doe",
    "user_id": "user_12345",
    "properties": {
        "plan": "premium",
        "signup_date": "2023-01-15",
        "company": "Acme Corp"
    }
}
```

**Required Fields:**
- `fingerprint` (string): Browser fingerprint hash
- `email` (string): User email address

**Response:**
```json
{
    "success": true,
    "visitor_id": "visitor_abc123",
    "identified": true,
    "merged_sessions": 3
}
```

**Status Codes:**
- `200 OK`: User identified successfully
- `400 Bad Request`: Invalid request data
- `404 Not Found`: Visitor not found
- `500 Internal Server Error`: Server error

**Example:**
```bash
curl -X POST http://localhost:8080/api/v1/visitor/identify \
  -H "Content-Type: application/json" \
  -d '{
    "fingerprint": "test123",
    "email": "john@example.com",
    "name": "John Doe",
    "properties": {
      "plan": "enterprise",
      "company": "Tech Corp"
    }
  }'
```

### GET /segments
List all visitor segments.

**Query Parameters:**
- `active_only` (boolean): Only return active segments (default: false)

**Response:**
```json
{
    "segments": [
        {
            "id": "segment_123",
            "name": "High Value Customers",
            "description": "Customers with >$500 lifetime value",
            "visitor_count": 847,
            "criteria": {
                "and": [
                    {"lifetime_value": {"gte": 500}},
                    {"identified": {"eq": true}}
                ]
            },
            "last_calculated": "2023-12-01T14:00:00Z"
        }
    ],
    "total_segments": 8
}
```

### POST /segments
Create a new visitor segment.

**Request Body:**
```json
{
    "name": "Recent High Engagement",
    "description": "Visitors with high engagement in last 7 days",
    "criteria": {
        "and": [
            {"last_seen": {"gte": "7d"}},
            {"total_page_views": {"gte": 10}},
            {"avg_session_duration": {"gte": 300}}
        ]
    }
}
```

**Response:**
```json
{
    "success": true,
    "segment_id": "segment_456",
    "visitor_count": 234
}
```

## Utility Endpoints

### GET /health
System health check.

**Response:**
```json
{
    "status": "healthy",
    "timestamp": "2023-12-01T15:45:00Z",
    "version": "1.0.0",
    "services": {
        "postgres": "healthy",
        "redis": "healthy"
    },
    "metrics": {
        "uptime": 86400,
        "memory_usage": 245.6,
        "requests_per_minute": 1247
    }
}
```

### GET /tracker.js
Visitor tracking JavaScript library.

**Response:** JavaScript file with tracking implementation
**Content-Type:** `text/javascript`
**Cache-Control:** `public, max-age=3600`

## WebSocket Events (Future)
Real-time event streaming for live dashboards:

```javascript
const ws = new WebSocket('ws://localhost:8080/ws/events');

ws.onmessage = (event) => {
    const data = JSON.parse(event.data);
    console.log('Real-time event:', data);
    // Handle real-time visitor events
};
```

## Error Codes

| Code | Description |
|------|-------------|
| `VALIDATION_ERROR` | Request validation failed |
| `VISITOR_NOT_FOUND` | Visitor ID not found |
| `SCENARIO_NOT_FOUND` | Scenario not found |
| `RATE_LIMIT_EXCEEDED` | Too many requests |
| `DATABASE_ERROR` | Database operation failed |
| `CACHE_ERROR` | Redis cache error |
| `INTERNAL_ERROR` | Unexpected server error |

## SDKs and Libraries

### JavaScript/Node.js
```javascript
const VisitorIntelligence = require('visitor-intelligence-sdk');

const client = new VisitorIntelligence({
    apiUrl: 'http://localhost:8080/api/v1',
    scenario: 'my-app'
});

// Track event
await client.track('pageview', {
    page: '/dashboard',
    user_id: '123'
});

// Get analytics
const analytics = await client.getAnalytics('my-app', '7d');
```

### Python
```python
from visitor_intelligence import Client

client = Client(
    api_url='http://localhost:8080/api/v1',
    scenario='my-app'
)

# Track event
client.track('purchase', {
    'amount': 99.99,
    'product': 'premium_plan'
})

# Get visitor profile
visitor = client.get_visitor('visitor_123')
```

### Go
```go
package main

import "github.com/vrooli/visitor-intelligence-go"

func main() {
    client := visitor.NewClient(&visitor.Config{
        APIUrl:   "http://localhost:8080/api/v1",
        Scenario: "my-app",
    })
    
    // Track event
    client.Track("signup", map[string]interface{}{
        "plan": "premium",
        "source": "landing_page",
    })
}
```

## Best Practices

### Event Naming
- Use snake_case: `product_added_to_cart`
- Be descriptive: `user_completed_onboarding`
- Include context: `blog_article_shared_facebook`

### Property Structure
- Use consistent data types
- Include relevant metadata
- Avoid nested objects (use flat structure)
- Include timestamp for time-based analysis

### Performance
- Batch events when possible
- Use appropriate timeframes for analytics
- Cache visitor profiles on client side
- Implement retry logic for failed requests

### Security
- Validate all inputs
- Sanitize properties before storage
- Use HTTPS in production
- Implement rate limiting
- Consider API authentication for sensitive data

---

This API enables powerful visitor intelligence capabilities across all Vrooli scenarios. For integration examples and detailed guides, see the [Integration Guide](integration-guide.md).