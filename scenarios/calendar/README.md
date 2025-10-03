# Calendar - Universal Scheduling Intelligence

## 🎯 Purpose

**Calendar is Vrooli's temporal coordination brain** - a scheduling intelligence system that transforms any scenario from isolated tools into time-aware, interconnected workflows. Every scenario that needs to schedule events, coordinate timing, or orchestrate multi-step processes can leverage this foundational capability.

## 💡 Usefulness

### For Scenarios
- **Project Manager**: Schedule milestones with automatic deadline tracking
- **Customer Onboarding**: Multi-step sequences triggered by calendar events
- **Team Collaboration**: Meeting coordination with conflict detection
- **Business Analytics**: Report generation tied to business cycles
- **Resource Management**: Equipment and room booking with availability

### For Users
- **Professional Calendar**: Clean, reliable interface for business scheduling
- **AI Assistant**: Natural language scheduling ("book lunch with Sarah next Tuesday")
- **Smart Optimization**: AI suggestions to optimize time management
- **Multi-channel Reminders**: Email, SMS, and push notifications
- **Cross-scenario Integration**: Events trigger automated workflows

## 🏗️ Architecture

### Core Components
- **Multi-user Events**: Personal and shared calendars with role-based access
- **Recurring Patterns**: Sophisticated repeat scheduling with exceptions
- **AI Search**: Semantic event search powered by Qdrant embeddings
- **Natural Language Interface**: Chat-based scheduling with Ollama/Claude
- **Automation Engine**: Event-triggered scenario execution
- **Professional UI**: Clean, responsive calendar interface

### Smart Features
- **Schedule Optimization**: AI-powered suggestions to improve time management
- **Conflict Detection**: Automatic identification and resolution of scheduling conflicts
- **Timezone Intelligence**: Global team coordination with timezone awareness
- **Template System**: Reusable meeting types and scheduling patterns
- **Travel Time Intelligence**: Automatic calculation of travel time between events with departure time suggestions
- **Resource Booking**: Prevent double-booking of meeting rooms, equipment, and other resources
- **External Calendar Sync**: Bidirectional synchronization with Google Calendar and Outlook
- **Advanced Analytics**: Comprehensive scheduling patterns analysis with productivity insights

## 🔧 Dependencies

### Required Resources
- **scenario-authenticator**: Multi-user support and authentication
- **notification-hub**: Event reminders and schedule change notifications
- **postgres**: Event storage, recurring patterns, user preferences
- **qdrant**: Vector embeddings for AI-powered semantic search

### Optional Enhancements
- **ollama**: Natural language processing for chat interface (native Go integration)

## 🎨 UX Style

### Design Philosophy
**Professional, efficient, trustworthy** - Inspired by Google Calendar's familiarity with Calendly's streamlined booking experience.

### Visual Style
- **Clean Interface**: Light theme with dark mode support
- **Responsive Layout**: Desktop-first with mobile optimization
- **Smooth Interactions**: Drag-and-drop scheduling, keyboard shortcuts
- **Accessible Design**: Full WCAG 2.1 AA compliance for enterprise use

### Target Experience
Users should feel **in complete control of their time** with an interface that's both powerful and intuitive.

## 🚀 Getting Started

### Quick Start
```bash
# Setup calendar system
vrooli scenario run calendar

# Create your first event
calendar event create "Team Meeting" "2024-01-15T10:00:00Z" --remind 15

# List upcoming events
calendar event list --days 7

# Start natural language interface
calendar schedule chat "find time for a 2-hour project review this week"

# Connect external calendar
calendar sync connect google
calendar sync connect outlook

# Check sync status
calendar sync status

# Manual sync
calendar sync run

# Disconnect external calendar
calendar sync disconnect google
```

### API Integration
```javascript
// Create event from another scenario
// Note: API_PORT is dynamically assigned by Vrooli (range: 15000-19999)
const API_PORT = process.env.API_PORT || '<check-vrooli-status>';
const response = await fetch(`http://localhost:${API_PORT}/api/v1/events`, {
  method: 'POST',
  headers: {
    'Authorization': `Bearer ${userToken}`,
    'Content-Type': 'application/json'
  },
  body: JSON.stringify({
    title: 'Project Milestone',
    start_time: '2024-01-20T14:00:00Z',
    end_time: '2024-01-20T15:00:00Z',
    automation: {
      trigger_url: `http://localhost:${PROJECT_API_PORT}/api/v1/milestone/complete`,
      payload: { project_id: 'proj_123' }
    }
  })
});
```

## 🔄 Integration Patterns

### Event Automation
Events can trigger other scenarios when they start:
```yaml
automation_config:
  trigger_url: "http://scenario-api/endpoint"
  payload: { context: "meeting_started" }
```

### Travel Time API Usage
```javascript
// Calculate travel time between locations
const travelTime = await fetch('/api/v1/travel/calculate', {
  method: 'POST',
  headers: { 
    'Content-Type': 'application/json',
    'Authorization': 'Bearer token'
  },
  body: JSON.stringify({
    origin: "Office Building A",
    destination: "Conference Room B",
    mode: "driving"  // Options: driving, transit, walking, bicycling
  })
});
// Returns: { duration_seconds: 450, distance_meters: 5000, traffic_condition: "moderate" }

// Get departure time suggestion for an event
const departure = await fetch(`/api/v1/events/${eventId}/departure-time?origin=Home&mode=transit`, {
  headers: { 'Authorization': 'Bearer token' }
});
// Returns: { departure_time: "2026-02-01T13:30:00Z", event_start: "2026-02-01T14:00:00Z" }
```

### Resource Booking API Usage
```javascript
// Create a bookable resource
const resource = await fetch('/api/v1/resources', {
  method: 'POST',
  headers: { 
    'Content-Type': 'application/json',
    'Authorization': 'Bearer token'
  },
  body: JSON.stringify({
    name: "Conference Room A",
    resource_type: "room",
    capacity: 20,
    location: "Building 1, Floor 2"
  })
});

// Check resource availability
const available = await fetch(`/api/v1/resources/${resourceId}/availability?start_time=2025-01-01T10:00:00Z&end_time=2025-01-01T11:00:00Z`, {
  headers: { 'Authorization': 'Bearer token' }
});
// Returns: { available: true/false, conflicts: [...] }

// Book resource for an event
const booking = await fetch(`/api/v1/events/${eventId}/resources`, {
  method: 'POST',
  headers: { 
    'Content-Type': 'application/json',
    'Authorization': 'Bearer token'
  },
  body: JSON.stringify({
    resource_id: resourceId,
    booking_status: "confirmed"
  })
});
```

### Cross-scenario APIs
- `POST /api/v1/events` - Create time-based workflows
- `GET /api/v1/events?search=query` - Semantic event discovery
- `POST /api/v1/schedule/optimize` - AI-powered time optimization
- `POST /api/v1/travel/calculate` - Calculate travel time between locations
- `GET /api/v1/events/{id}/departure-time` - Get departure time for an event
- `POST /api/v1/resources` - Create bookable resources
- `GET /api/v1/resources/{id}/availability` - Check resource availability
- `POST /api/v1/events/{id}/resources` - Book resources for events

## 🎯 Success Metrics

- **Adoption**: 80%+ of business scenarios use calendar for time coordination
- **Performance**: Sub-200ms API responses, sub-500ms semantic search
- **Intelligence**: 90%+ accuracy in natural language scheduling requests
- **Reliability**: 99.9% uptime for mission-critical business scheduling

---

**Value Proposition**: Every scheduling capability built once, reused forever. Calendar intelligence that compounds across all scenarios, making the entire Vrooli ecosystem time-aware and temporally coordinated.