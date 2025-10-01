# Calendar API Documentation

## Overview

The Calendar API is a comprehensive RESTful service that provides intelligent scheduling, event management, and AI-powered natural language processing for calendar operations. This API is designed to integrate seamlessly with the Vrooli ecosystem and supports advanced features like semantic event search, automated reminder processing, and cross-scenario coordination.

## Base URL

```
http://localhost:{API_PORT}/api/v1
```

> **Note**: The `API_PORT` is dynamically assigned within the range 15000-19999 as configured in the service.json file.

## Authentication

The Calendar API integrates with the `scenario-authenticator` service. All API endpoints require authentication via Bearer token in the Authorization header:

```http
Authorization: Bearer <your-jwt-token>
```

Authentication is validated against the configured auth service URL at `/api/v1/auth/validate`.

## API Versioning

The API follows semantic versioning with the current version being `v1`. All endpoints are prefixed with `/api/v1` to ensure backward compatibility and smooth migration paths.

## Health Check

### GET /health

Check the health status of the Calendar API service.

**Request:**
```http
GET /health HTTP/1.1
```

**Response:**
```json
{
  "status": "healthy",
  "timestamp": "2024-01-15T10:30:00Z",
  "version": "1.0.0",
  "database": "connected",
  "external_services": {
    "qdrant": "connected",
    "notification_hub": "connected",
    "auth_service": "connected"
  }
}
```

---

## Events Management

### POST /api/v1/events

Create a new calendar event with optional recurrence patterns and reminders.

**Request:**
```http
POST /api/v1/events HTTP/1.1
Content-Type: application/json
Authorization: Bearer <token>
```

**Request Body:**
```json
{
  "title": "Team Strategy Meeting",
  "description": "Quarterly planning and goal setting session",
  "start_time": "2024-01-15T14:00:00Z",
  "end_time": "2024-01-15T16:00:00Z",
  "timezone": "America/New_York",
  "location": "Conference Room A",
  "event_type": "meeting",
  "metadata": {
    "priority": "high",
    "attendees": ["john@example.com", "sarah@example.com"],
    "project_id": "proj-456",
    "tags": ["strategy", "planning", "quarterly"]
  },
  "automation_config": {
    "auto_summary": true,
    "meeting_recording": true,
    "post_meeting_actions": {
      "send_summary": true,
      "create_action_items": true,
      "update_crm": true
    }
  },
  "recurrence": {
    "pattern": "weekly",
    "interval": 1,
    "days_of_week": [2],
    "max_occurrences": 12
  },
  "reminders": [
    {
      "minutes_before": 1440,
      "type": "email"
    },
    {
      "minutes_before": 30,
      "type": "push"
    }
  ]
}
```

**Response (201 Created):**
```json
{
  "id": "e47ac10b-58cc-4372-a567-0e02b2c3d479",
  "user_id": "f47ac10b-58cc-4372-a567-0e02b2c3d479",
  "title": "Team Strategy Meeting",
  "description": "Quarterly planning and goal setting session",
  "start_time": "2024-01-15T14:00:00Z",
  "end_time": "2024-01-15T16:00:00Z",
  "timezone": "America/New_York",
  "location": "Conference Room A",
  "event_type": "meeting",
  "status": "active",
  "metadata": {
    "priority": "high",
    "attendees": ["john@example.com", "sarah@example.com"],
    "project_id": "proj-456",
    "tags": ["strategy", "planning", "quarterly"]
  },
  "automation_config": {
    "auto_summary": true,
    "meeting_recording": true,
    "post_meeting_actions": {
      "send_summary": true,
      "create_action_items": true,
      "update_crm": true
    }
  },
  "created_at": "2024-01-10T08:30:00Z",
  "updated_at": "2024-01-10T08:30:00Z"
}
```

**Event Types:**
- `meeting`: Business meetings, calls, conferences
- `appointment`: Personal appointments (doctor, dentist, etc.)
- `task`: Task blocks or work sessions
- `reminder`: Simple reminders or notifications
- `block`: Calendar blocking (focus time, travel, etc.)
- `personal`: Personal events (lunch, social, etc.)
- `work`: Work-related activities
- `travel`: Travel time and arrangements

### GET /api/v1/events

Retrieve events with filtering, pagination, and search capabilities.

**Request:**
```http
GET /api/v1/events?start=2024-01-15&end=2024-01-31&status=active&type=meeting&limit=20&offset=0&search=strategy HTTP/1.1
Authorization: Bearer <token>
```

**Query Parameters:**
- `start` (optional): Filter events starting from this date (ISO 8601)
- `end` (optional): Filter events ending before this date (ISO 8601)
- `status` (optional): Filter by status (`active`, `cancelled`, `completed`)
- `type` (optional): Filter by event type
- `search` (optional): Search events by title, description, or location
- `limit` (optional): Number of results per page (default: 50, max: 100)
- `offset` (optional): Number of results to skip (default: 0)
- `include_metadata` (optional): Include metadata in response (default: false)

**Response (200 OK):**
```json
{
  "events": [
    {
      "id": "e47ac10b-58cc-4372-a567-0e02b2c3d479",
      "title": "Team Strategy Meeting",
      "description": "Quarterly planning and goal setting session",
      "start_time": "2024-01-15T14:00:00Z",
      "end_time": "2024-01-15T16:00:00Z",
      "timezone": "America/New_York",
      "location": "Conference Room A",
      "event_type": "meeting",
      "status": "active",
      "created_at": "2024-01-10T08:30:00Z",
      "updated_at": "2024-01-10T08:30:00Z"
    }
  ],
  "pagination": {
    "total": 1,
    "limit": 20,
    "offset": 0,
    "has_more": false
  }
}
```

### GET /api/v1/events/{id}

Retrieve a specific event by ID.

**Request:**
```http
GET /api/v1/events/e47ac10b-58cc-4372-a567-0e02b2c3d479 HTTP/1.1
Authorization: Bearer <token>
```

**Response (200 OK):**
```json
{
  "id": "e47ac10b-58cc-4372-a567-0e02b2c3d479",
  "user_id": "f47ac10b-58cc-4372-a567-0e02b2c3d479",
  "title": "Team Strategy Meeting",
  "description": "Quarterly planning and goal setting session",
  "start_time": "2024-01-15T14:00:00Z",
  "end_time": "2024-01-15T16:00:00Z",
  "timezone": "America/New_York",
  "location": "Conference Room A",
  "event_type": "meeting",
  "status": "active",
  "metadata": {
    "priority": "high",
    "attendees": ["john@example.com", "sarah@example.com"],
    "project_id": "proj-456"
  },
  "automation_config": {
    "auto_summary": true,
    "meeting_recording": true
  },
  "created_at": "2024-01-10T08:30:00Z",
  "updated_at": "2024-01-10T08:30:00Z"
}
```

### PUT /api/v1/events/{id}

Update an existing event.

**Request:**
```http
PUT /api/v1/events/e47ac10b-58cc-4372-a567-0e02b2c3d479 HTTP/1.1
Content-Type: application/json
Authorization: Bearer <token>
```

**Request Body:**
```json
{
  "title": "Updated Team Strategy Meeting",
  "description": "Q1 planning and goal setting session - updated",
  "start_time": "2024-01-15T15:00:00Z",
  "end_time": "2024-01-15T17:00:00Z",
  "location": "Conference Room B",
  "metadata": {
    "priority": "critical",
    "attendees": ["john@example.com", "sarah@example.com", "mike@example.com"]
  }
}
```

**Response (200 OK):**
```json
{
  "id": "e47ac10b-58cc-4372-a567-0e02b2c3d479",
  "user_id": "f47ac10b-58cc-4372-a567-0e02b2c3d479",
  "title": "Updated Team Strategy Meeting",
  "description": "Q1 planning and goal setting session - updated",
  "start_time": "2024-01-15T15:00:00Z",
  "end_time": "2024-01-15T17:00:00Z",
  "timezone": "America/New_York",
  "location": "Conference Room B",
  "event_type": "meeting",
  "status": "active",
  "metadata": {
    "priority": "critical",
    "attendees": ["john@example.com", "sarah@example.com", "mike@example.com"]
  },
  "created_at": "2024-01-10T08:30:00Z",
  "updated_at": "2024-01-10T09:15:00Z"
}
```

### DELETE /api/v1/events/{id}

Delete (soft delete) an event.

**Request:**
```http
DELETE /api/v1/events/e47ac10b-58cc-4372-a567-0e02b2c3d479 HTTP/1.1
Authorization: Bearer <token>
```

**Response (204 No Content)**

> **Note**: Events are soft-deleted by updating their status to "cancelled" to preserve historical data and maintain referential integrity with reminders and recurring patterns.

---

## AI-Powered Scheduling

### POST /api/v1/schedule/chat

Process natural language scheduling requests using AI/NLP.

**Request:**
```http
POST /api/v1/schedule/chat HTTP/1.1
Content-Type: application/json
Authorization: Bearer <token>
```

**Request Body:**
```json
{
  "message": "Schedule lunch with Sarah next Tuesday at noon for an hour",
  "context": {
    "timezone": "America/New_York",
    "current_date": "2024-01-10T10:00:00Z",
    "preferences": {
      "default_duration": 60,
      "work_hours": {
        "start": "09:00",
        "end": "17:00"
      }
    }
  }
}
```

**Response (200 OK):**
```json
{
  "response": "I understand you'd like to schedule lunch with Sarah for next Tuesday at noon. I've found the optimal time slot and can create this event for you.",
  "suggested_actions": [
    {
      "action": "create_event",
      "confidence": 0.95,
      "parameters": {
        "title": "Lunch with Sarah",
        "start_time": "2024-01-16T12:00:00-05:00",
        "end_time": "2024-01-16T13:00:00-05:00",
        "event_type": "personal",
        "location": "TBD",
        "metadata": {
          "contact": "Sarah",
          "meal_type": "lunch",
          "ai_generated": true
        }
      }
    }
  ],
  "requires_confirmation": true,
  "context": {
    "extracted_entities": {
      "person": "Sarah",
      "time": "next Tuesday at noon",
      "duration": "one hour",
      "event_type": "lunch"
    },
    "schedule_conflicts": [],
    "alternative_suggestions": []
  }
}
```

**Natural Language Examples:**
- "Schedule a team meeting for tomorrow at 2 PM"
- "Book a doctor's appointment next Friday afternoon"
- "Set up a recurring weekly standup every Tuesday at 9 AM"
- "Block my calendar for focus time every morning from 9 to 11"
- "Reschedule the client call to next week"

### POST /api/v1/schedule/optimize

Optimize calendar scheduling using AI algorithms.

**Request:**
```http
POST /api/v1/schedule/optimize HTTP/1.1
Content-Type: application/json
Authorization: Bearer <token>
```

**Request Body:**
```json
{
  "optimization_type": "focus_time",
  "date_range": {
    "start": "2024-01-15T00:00:00Z",
    "end": "2024-01-21T23:59:59Z"
  },
  "constraints": {
    "min_duration_minutes": 120,
    "preferred_times": ["09:00-11:00", "14:00-16:00"],
    "exclude_days": ["saturday", "sunday"],
    "buffer_minutes": 15
  },
  "preferences": {
    "energy_optimization": true,
    "minimize_context_switching": true,
    "respect_work_hours": true
  },
  "context": {
    "work_hours": {
      "start": "09:00",
      "end": "17:00"
    },
    "timezone": "America/New_York"
  }
}
```

**Response (200 OK):**
```json
{
  "optimized_schedule": [
    {
      "suggested_time": "2024-01-15T09:00:00-05:00",
      "duration_minutes": 120,
      "confidence": 0.92,
      "reasoning": "High energy period with no conflicts, optimal for deep work"
    },
    {
      "suggested_time": "2024-01-17T14:00:00-05:00",
      "duration_minutes": 150,
      "confidence": 0.88,
      "reasoning": "Extended afternoon block available, good for project completion"
    }
  ],
  "optimization_summary": {
    "total_focus_time": 270,
    "efficiency_score": 0.89,
    "conflicts_resolved": 2,
    "energy_optimization_applied": true
  },
  "suggested_actions": [
    {
      "action": "create_focus_blocks",
      "parameters": [
        {
          "title": "AI Optimized: Deep Work Session",
          "start_time": "2024-01-15T09:00:00-05:00",
          "end_time": "2024-01-15T11:00:00-05:00",
          "event_type": "block",
          "metadata": {
            "ai_optimized": true,
            "optimization_type": "focus_time",
            "confidence": 0.92
          }
        }
      ]
    }
  ]
}
```

**Optimization Types:**
- `focus_time`: Optimize for deep work blocks
- `meeting_batching`: Group similar meetings together
- `energy_matching`: Match tasks to optimal energy levels
- `travel_minimization`: Reduce travel time between locations
- `conflict_resolution`: Resolve scheduling conflicts

---

## Reminder Processing

### POST /api/v1/reminders/process

Process and send pending reminders (typically called by automated systems).

**Request:**
```http
POST /api/v1/reminders/process HTTP/1.1
Content-Type: application/json
Authorization: Bearer <token>
```

**Request Body:**
```json
{
  "batch_size": 100,
  "dry_run": false,
  "notification_types": ["email", "push", "sms"],
  "time_window": {
    "start": "2024-01-15T10:00:00Z",
    "end": "2024-01-15T10:30:00Z"
  }
}
```

**Response (200 OK):**
```json
{
  "processed_count": 15,
  "sent_count": 12,
  "failed_count": 3,
  "processing_summary": {
    "email": {
      "sent": 8,
      "failed": 1
    },
    "push": {
      "sent": 3,
      "failed": 1
    },
    "sms": {
      "sent": 1,
      "failed": 1
    }
  },
  "failed_reminders": [
    {
      "reminder_id": "d47ac10b-58cc-4372-a567-0e02b2c3d480",
      "event_id": "e47ac10b-58cc-4372-a567-0e02b2c3d480",
      "error": "Notification service unavailable",
      "retry_scheduled": "2024-01-15T10:35:00Z"
    }
  ]
}
```

---

## Data Models

### Event Model

```json
{
  "id": "UUID",
  "user_id": "UUID",
  "title": "string (required, max 255 chars)",
  "description": "string (optional)",
  "start_time": "ISO 8601 timestamp (required)",
  "end_time": "ISO 8601 timestamp (required)",
  "timezone": "string (IANA timezone, default: UTC)",
  "location": "string (optional, max 500 chars)",
  "event_type": "enum [meeting, appointment, task, reminder, block, personal, work, travel]",
  "status": "enum [active, cancelled, completed]",
  "metadata": "object (optional, JSON)",
  "automation_config": "object (optional, JSON)",
  "created_at": "ISO 8601 timestamp",
  "updated_at": "ISO 8601 timestamp"
}
```

### Recurrence Pattern

```json
{
  "pattern": "enum [daily, weekly, monthly, yearly, custom]",
  "interval": "integer (required, min: 1)",
  "days_of_week": "array of integers [0-6] (Sunday=0)",
  "days_of_month": "array of integers [1-31]",
  "end_date": "ISO 8601 timestamp (optional)",
  "max_occurrences": "integer (optional)",
  "exceptions": "array of ISO 8601 timestamps (optional)"
}
```

---

## Error Handling

The API uses standard HTTP status codes and returns consistent error responses:

### Error Response Format

```json
{
  "error": {
    "code": "VALIDATION_ERROR",
    "message": "The provided start_time must be before end_time",
    "details": {
      "field": "start_time",
      "value": "2024-01-15T16:00:00Z",
      "constraint": "must_be_before_end_time"
    },
    "timestamp": "2024-01-15T10:30:00Z",
    "request_id": "req-123e4567-e89b-12d3-a456-426614174000"
  }
}
```

### HTTP Status Codes

- `200 OK`: Successful request
- `201 Created`: Resource created successfully
- `204 No Content`: Successful request with no response body
- `400 Bad Request`: Invalid request data or parameters
- `401 Unauthorized`: Authentication required or invalid token
- `403 Forbidden`: Insufficient permissions
- `404 Not Found`: Resource not found
- `409 Conflict`: Request conflicts with current state (e.g., overlapping events)
- `429 Too Many Requests`: Rate limit exceeded
- `500 Internal Server Error`: Server error
- `503 Service Unavailable`: Service temporarily unavailable

### Common Error Codes

- `VALIDATION_ERROR`: Input validation failed
- `AUTHENTICATION_ERROR`: Authentication failed
- `AUTHORIZATION_ERROR`: Insufficient permissions
- `RESOURCE_NOT_FOUND`: Requested resource doesn't exist
- `CONFLICT_ERROR`: Resource conflict (e.g., overlapping events)
- `RATE_LIMIT_EXCEEDED`: Too many requests
- `EXTERNAL_SERVICE_ERROR`: External service (Qdrant, auth, notifications) unavailable
- `DATABASE_ERROR`: Database connection or query error

---

## Rate Limiting

The API implements rate limiting to ensure fair usage:

- **Standard endpoints**: 1000 requests per hour per user
- **AI endpoints** (`/schedule/chat`, `/schedule/optimize`): 100 requests per hour per user
- **Reminder processing**: 10 requests per hour per user

Rate limit headers are included in responses:

```http
X-RateLimit-Limit: 1000
X-RateLimit-Remaining: 999
X-RateLimit-Reset: 1642248000
X-RateLimit-Window: 3600
```

---

## Integration Examples

### Webhook Integration

Configure automation_config to trigger webhooks:

```json
{
  "automation_config": {
    "webhooks": {
      "event_created": "https://your-app.com/webhooks/calendar/created",
      "event_updated": "https://your-app.com/webhooks/calendar/updated",
      "reminder_sent": "https://your-app.com/webhooks/calendar/reminder"
    }
  }
}
```

### Cross-Scenario Integration

Events can reference other scenarios:

```json
{
  "metadata": {
    "integration": {
      "crm_opportunity_id": "opp-123",
      "project_management": {
        "scenario": "project-tracker",
        "task_id": "task-456"
      },
      "notification_profiles": ["urgent-alerts", "team-updates"]
    }
  }
}
```

---

## SDK and Client Libraries

Official client libraries are available for:

- **JavaScript/TypeScript**: `@vrooli/calendar-client`
- **Python**: `vrooli-calendar-python`
- **Go**: `github.com/vrooli/calendar-go-client`

Example usage (JavaScript):

```javascript
import { CalendarClient } from '@vrooli/calendar-client';

const client = new CalendarClient({
  baseURL: 'http://localhost:15000',
  authToken: 'your-jwt-token'
});

// Create an event
const event = await client.events.create({
  title: 'Team Meeting',
  startTime: '2024-01-15T14:00:00Z',
  endTime: '2024-01-15T15:00:00Z',
  eventType: 'meeting'
});

// Natural language scheduling
const response = await client.schedule.chat({
  message: 'Schedule lunch with John tomorrow at 1 PM'
});
```

---

## Development and Testing

### Environment Variables

Required environment variables:

- `API_PORT`: API server port (from range 15000-19999)
- `POSTGRES_URL`: PostgreSQL connection string
- `QDRANT_URL`: Qdrant vector database URL
- `AUTH_SERVICE_URL`: Authentication service URL
- `NOTIFICATION_SERVICE_URL`: Notification hub URL
- `OLLAMA_URL`: Ollama NLP service URL (optional)
- `JWT_SECRET`: JWT signing secret
- `NOTIFICATION_PROFILE_ID`: Default notification profile
- `NOTIFICATION_API_KEY`: Notification service API key

### Testing Endpoints

Use the health endpoint to verify service connectivity:

```bash
curl -X GET http://localhost:${API_PORT}/health
```

### Local Development

Start the service with development configuration:

```bash
cd /scenarios/calendar
make run
```

This will start all required services and dependencies automatically.

---

## Changelog

### Version 1.0.0 (Current)
- Initial API release
- Core event management
- AI-powered natural language processing
- Schedule optimization
- Multi-channel reminder system
- Cross-scenario integration support

---

For questions, issues, or contributions, please refer to the main Calendar scenario documentation in the [README.md](../README.md) file.
---

## Meeting Preparation Automation

The calendar API provides automated meeting preparation features to help users prepare for meetings efficiently.

### Generate Meeting Agenda

Generate an automatic agenda for a meeting based on its type and duration.

**Endpoint:** `GET /api/v1/events/{event_id}/agenda`

**Request:**
```http
GET /api/v1/events/123e4567-e89b-12d3-a456-426614174000/agenda
Authorization: Bearer <token>
```

**Response:**
```json
{
  "event_id": "123e4567-e89b-12d3-a456-426614174000",
  "title": "Team Planning Meeting",
  "objective": "Define goals, create action plans, and assign responsibilities",
  "duration": "60 minutes",
  "attendees": ["team@example.com", "manager@example.com"],
  "agenda_items": [
    {
      "topic": "Welcome & Agenda Review",
      "duration_minutes": 5,
      "description": "Brief introduction and review of meeting objectives"
    },
    {
      "topic": "Goals & Objectives",
      "duration_minutes": 15,
      "description": "Define what we want to achieve"
    },
    {
      "topic": "Strategy & Approach", 
      "duration_minutes": 20,
      "description": "How we will achieve our goals"
    },
    {
      "topic": "Timeline & Milestones",
      "duration_minutes": 15,
      "description": "Key dates and deliverables"
    },
    {
      "topic": "Summary & Action Items",
      "duration_minutes": 5,
      "description": "Recap decisions and next steps"
    }
  ],
  "pre_work": [
    "Review previous plans and outcomes",
    "Research industry best practices",
    "Prepare initial ideas and proposals"
  ],
  "generated_at": "2025-09-27T10:00:00Z"
}
```

### Update Meeting Agenda

Save or update a customized agenda for a meeting.

**Endpoint:** `PUT /api/v1/events/{event_id}/agenda`

**Request:**
```json
{
  "objective": "Custom objective for the meeting",
  "agenda_items": [
    {
      "topic": "Introduction",
      "duration_minutes": 10,
      "owner": "John Smith",
      "description": "Welcome and introductions"
    }
  ],
  "pre_work": ["Review Q3 reports"],
  "notes": "Remember to discuss budget allocation"
}
```

**Response:**
```json
{
  "success": true,
  "message": "Agenda updated successfully"
}
```

### Supported Meeting Types

The system automatically generates appropriate agendas for various meeting types:

- **Standup/Daily**: Progress updates, blockers, daily plans
- **Review**: Progress overview, feedback, next steps
- **Planning**: Goals, strategy, timeline, responsibilities
- **Retrospective**: Reflections, improvements, action items
- **1-on-1**: Progress discussion, feedback, concerns
- **Kickoff**: Project goals, timeline, roles, success criteria
- **Brainstorming**: Idea generation, solution exploration

The agenda generation considers:
- Meeting duration for appropriate time allocation
- Meeting title keywords for context
- Number of attendees for participation planning
- Event type for specialized formatting
EOF < /dev/null
