# Calendar - Product Requirements Document (PRD)

## 🎯 Capability Definition

### Core Capability
**What permanent capability does this scenario add to Vrooli?**
Universal scheduling intelligence that enables any scenario to create, manage, and coordinate time-based events, recurring schedules, and temporal workflows. This becomes Vrooli's temporal coordination brain, transforming scenarios from isolated tools into time-aware, interconnected systems that can orchestrate complex multi-step workflows across time.

### Intelligence Amplification
**How does this capability make future agents smarter?**
Agents can now reason about time, schedule cascading workflows, coordinate resources across scenarios, and create sophisticated automation that respects temporal constraints. This enables agents to build enterprise-grade scheduling applications, automate recurring business processes, and create intelligent time-management systems that improve over time through user interaction patterns.

### Recursive Value
**What new scenarios become possible after this exists?**
- **Enterprise Project Manager**: Multi-team coordination with milestone dependencies and resource allocation
- **Automated Customer Onboarding**: Multi-step sequences triggered by calendar events with personalized timing
- **Resource Booking System**: Conference rooms, equipment, and personnel scheduling with conflict resolution
- **Smart Home Orchestration**: Device automation tied to personal schedules and presence detection
- **Business Intelligence Scheduler**: Automated report generation and data pipeline execution based on business cycles

## 📊 Success Metrics

### Functional Requirements
- **Must Have (P0)**
  - [ ] Multi-user event creation with authentication via scenario-authenticator (PARTIAL: single-user mode working)
  - [x] Recurring event patterns (daily, weekly, monthly, custom) - Tested 2025-09-24
  - [ ] Event reminders via notification-hub integration (NOT CONFIGURED: disabled in current deployment)
  - [ ] Natural language scheduling through chat interface (NOT CONFIGURED: Ollama not integrated)
  - [ ] Event-triggered code execution for scenario automation (NOT IMPLEMENTED)
  - [x] PostgreSQL storage with full CRUD operations - Tested 2025-09-24
  - [x] AI-powered schedule search using Qdrant embeddings - Tested 2025-09-24
  - [x] REST API for programmatic access by other scenarios - Tested 2025-09-24
  - [ ] Professional calendar UI with week/month/agenda views (PARTIAL: UI running but views not tested)
  
- **Should Have (P1)**
  - [ ] Schedule optimization suggestions ("free up my schedule tonight")
  - [ ] Smart conflict detection and resolution
  - [ ] Timezone handling for distributed teams
  - [ ] Event categorization and filtering
  - [ ] Bulk operations and batch scheduling
  - [ ] iCal import/export for external calendar sync
  - [ ] Event templates for common meeting types
  - [ ] Attendance tracking and RSVP functionality
  
- **Nice to Have (P2)**
  - [ ] External calendar synchronization (Google Calendar, Outlook)
  - [ ] Meeting preparation automation (agenda creation, document gathering)
  - [ ] Travel time calculation and buffer insertion
  - [ ] Resource double-booking prevention
  - [ ] Advanced analytics on scheduling patterns and productivity
  - [ ] Voice-activated scheduling through audio scenarios

### Performance Criteria
| Metric | Target | Measurement Method |
|--------|--------|-------------------|
| API Response Time | < 200ms for 95% of requests | API monitoring |
| Event Search Speed | < 500ms for semantic search | Qdrant query timing |
| Chat Response Time | < 2s for natural language queries | End-to-end timing |
| Concurrent Users | 1,000+ simultaneous calendar views | Load testing |
| Database Throughput | 10,000 events/minute creation | Stress testing |

### Quality Gates
- [ ] All P0 requirements implemented and tested (4/9 COMPLETE)
- [ ] Integration tests pass with notification-hub and scenario-authenticator (NOT TESTED - resources not configured)
- [ ] Performance targets met under load (NOT TESTED - basic health checks pass)
- [ ] Natural language processing accuracy > 90% for common scheduling requests (NOT TESTED - Ollama not configured)
- [ ] Professional UI matches design specifications (NOT FULLY TESTED)

## 🏗️ Technical Architecture

### Resource Dependencies
```yaml
required:
  - resource_name: postgres
    purpose: Event storage, user schedules, recurring patterns, metadata
    integration_pattern: Direct SQL via Go database/sql with connection pooling
    access_method: resource-postgres CLI for setup, direct connection for runtime
    
  - resource_name: qdrant
    purpose: Event embeddings for AI-powered search and scheduling intelligence
    integration_pattern: REST API with vector embeddings
    access_method: Direct HTTP API for real-time search operations
    
  - resource_name: scenario-authenticator
    purpose: Multi-user authentication and user context
    integration_pattern: JWT token validation via API
    access_method: /api/v1/auth/validate endpoint for request authentication
    
  - resource_name: notification-hub
    purpose: Event reminders and schedule change notifications
    integration_pattern: Profile-scoped API with notification templates
    access_method: /api/v1/profiles/{profile_id}/notifications/send API
    
optional:
  - resource_name: ollama
    purpose: Natural language processing for chat interface
    fallback: Basic command parsing without AI enhancement
    access_method: Direct HTTP API integration for NLP processing
```

### Resource Integration Standards
```yaml
integration_priorities:
  1_resource_cli:
    - command: resource-postgres create-database calendar-system
      purpose: Initialize calendar database with proper schema
    - command: resource-qdrant create-collection calendar_events --dimensions 1536
      purpose: Create vector collection for semantic event search
  
  2_direct_api:
    - justification: Real-time event queries need sub-second response times
      endpoint: scenario-authenticator /api/v1/auth/validate
    - justification: Immediate notification delivery for time-sensitive reminders
      endpoint: notification-hub /api/v1/profiles/calendar-prod/notifications/send
    - justification: Natural language processing requires direct model access
      endpoint: ollama /api/chat for scheduling intent parsing

integration_patterns:
  - All reminder processing handled via internal calendar-processor component
  - NLP scheduling requests processed via internal nlp-processor component  
  - Event automation triggers handled via internal event handlers
  - No external workflow orchestration dependencies required
```

### Data Models
```yaml
primary_entities:
  - name: Event
    storage: postgres
    schema: |
      {
        id: UUID PRIMARY KEY,
        user_id: UUID REFERENCES users(id),
        title: VARCHAR(255) NOT NULL,
        description: TEXT,
        start_time: TIMESTAMPTZ NOT NULL,
        end_time: TIMESTAMPTZ NOT NULL,
        timezone: VARCHAR(50) DEFAULT 'UTC',
        location: VARCHAR(500),
        event_type: VARCHAR(50) DEFAULT 'meeting',
        recurrence_rule: JSONB,
        metadata: JSONB DEFAULT '{}',
        automation_config: JSONB,
        created_at: TIMESTAMPTZ DEFAULT NOW(),
        updated_at: TIMESTAMPTZ DEFAULT NOW()
      }
    relationships: Belongs to User, has many EventReminders, has many EventEmbeddings
    
  - name: EventEmbedding
    storage: qdrant
    schema: |
      {
        point_id: UUID (matches event.id),
        vector: [1536 dimensions],
        payload: {
          user_id: UUID,
          title: string,
          description: string,
          event_type: string,
          location: string,
          start_time: timestamp,
          keywords: array[string]
        }
      }
    relationships: One-to-one with Event
    
  - name: EventReminder
    storage: postgres
    schema: |
      {
        id: UUID PRIMARY KEY,
        event_id: UUID REFERENCES events(id),
        reminder_time: TIMESTAMPTZ NOT NULL,
        notification_type: VARCHAR(20) DEFAULT 'email',
        status: VARCHAR(20) DEFAULT 'pending',
        notification_id: VARCHAR(255),
        created_at: TIMESTAMPTZ DEFAULT NOW()
      }
    relationships: Belongs to Event
    
  - name: RecurringPattern
    storage: postgres
    schema: |
      {
        id: UUID PRIMARY KEY,
        parent_event_id: UUID REFERENCES events(id),
        pattern_type: VARCHAR(20) NOT NULL,
        interval_value: INTEGER DEFAULT 1,
        days_of_week: INTEGER[],
        end_date: TIMESTAMPTZ,
        max_occurrences: INTEGER,
        created_at: TIMESTAMPTZ DEFAULT NOW()
      }
    relationships: Belongs to Event (parent)
```

### API Contract
```yaml
endpoints:
  - method: POST
    path: /api/v1/events
    purpose: Create new calendar event with optional recurrence
    input_schema: |
      {
        title: string (required),
        description: string (optional),
        start_time: ISO8601 timestamp (required),
        end_time: ISO8601 timestamp (required),
        timezone: string (optional, default UTC),
        location: string (optional),
        event_type: string (optional),
        recurrence: {
          pattern: "daily|weekly|monthly|custom",
          interval: number,
          end_date: ISO8601 timestamp
        },
        reminders: array[{
          minutes_before: number,
          type: "email|sms|push"
        }],
        automation: {
          trigger_url: string,
          payload: object
        }
      }
    output_schema: |
      {
        success: boolean,
        event: { id, title, start_time, end_time },
        recurrence_count: number,
        reminders_scheduled: number
      }
    sla:
      response_time: 200ms
      availability: 99.9%
      
  - method: GET
    path: /api/v1/events
    purpose: List events with filtering and search capabilities
    input_schema: |
      Query parameters: {
        start_date: ISO8601 (optional),
        end_date: ISO8601 (optional),
        event_type: string (optional),
        search: string (optional, semantic search),
        user_id: UUID (optional, admin only)
      }
    output_schema: |
      {
        events: array[Event],
        total_count: number,
        has_more: boolean
      }
      
  - method: POST
    path: /api/v1/schedule/chat
    purpose: Natural language scheduling interface
    input_schema: |
      {
        message: string (required),
        context: object (optional, conversation state)
      }
    output_schema: |
      {
        response: string,
        suggested_actions: array[{
          action: "create_event|modify_event|cancel_event",
          confidence: number,
          parameters: object
        }],
        requires_confirmation: boolean,
        context: object
      }
    sla:
      response_time: 2000ms
      availability: 99.0%
      
  - method: POST
    path: /api/v1/schedule/optimize
    purpose: AI-powered schedule optimization
    input_schema: |
      {
        request: string ("free up tonight", "find 2 hours this week"),
        constraints: {
          preserve_high_priority: boolean,
          min_buffer_minutes: number,
          business_hours_only: boolean
        }
      }
    output_schema: |
      {
        suggestions: array[{
          description: string,
          affected_events: array[Event],
          proposed_changes: array[{
            event_id: UUID,
            action: "move|cancel|shorten",
            new_time: ISO8601
          }],
          confidence_score: number
        }]
      }
```

### Event Interface
```yaml
published_events:
  - name: calendar.event.created
    payload: { event_id, user_id, title, start_time, timestamp }
    subscribers: [notification-hub, analytics-tracker, project-manager]
    
  - name: calendar.event.starting
    payload: { event_id, user_id, automation_config, timestamp }
    subscribers: [automation-engine, attendance-tracker, resource-manager]
    
  - name: calendar.schedule.optimized
    payload: { user_id, optimization_request, changes_made, timestamp }
    subscribers: [analytics-tracker, user-preferences]
    
consumed_events:
  - name: auth.user.logged_in
    action: Sync user timezone and preferences
  - name: notification.delivered
    action: Update reminder status in database
```

## 🖥️ CLI Interface Contract

### Command Structure
```yaml
cli_binary: calendar
install_script: cli/install.sh

required_commands:
  - name: status
    description: Show calendar service status and upcoming events
    flags: [--json, --verbose, --user <user_id>]
    
  - name: help
    description: Display command help and usage
    flags: [--all, --command <name>]
    
  - name: version
    description: Show CLI and API version information
    flags: [--json]

custom_commands:
  - name: event create
    description: Create a new calendar event
    api_endpoint: /api/v1/events
    arguments:
      - name: title
        type: string
        required: true
        description: Event title
      - name: start
        type: string
        required: true
        description: Start time (ISO8601 or natural language)
    flags:
      - name: --end
        description: End time (defaults to 1 hour after start)
      - name: --location
        description: Event location
      - name: --recurring
        description: Recurrence pattern (daily, weekly, monthly)
      - name: --remind
        description: Reminder time in minutes before event
    output: Event ID and confirmation details
    
  - name: event list
    description: List upcoming events
    api_endpoint: /api/v1/events
    flags:
      - name: --days
        description: Number of days to show (default 7)
      - name: --type
        description: Filter by event type
      - name: --search
        description: Semantic search query
    output: Table of events with times and locations
    
  - name: schedule chat
    description: Natural language scheduling interface
    api_endpoint: /api/v1/schedule/chat
    arguments:
      - name: message
        type: string
        required: true
        description: Natural language scheduling request
    output: Assistant response and suggested actions
    
  - name: schedule optimize
    description: Optimize schedule with AI suggestions
    api_endpoint: /api/v1/schedule/optimize
    arguments:
      - name: request
        type: string
        required: true
        description: Optimization request ("free up tonight")
    flags:
      - name: --preserve-priority
        description: Keep high-priority events fixed
      - name: --business-hours
        description: Only suggest changes during business hours
    output: Optimization suggestions with confidence scores
    
  - name: remind test
    description: Test notification delivery for debugging
    arguments:
      - name: event_id
        type: string
        required: true
        description: Event ID to send test reminder for
    output: Notification delivery status
```

## 🔄 Integration Requirements

### Upstream Dependencies
- **scenario-authenticator**: User authentication and multi-user support for personal calendars
- **notification-hub**: Reliable delivery of event reminders and schedule change notifications
- **PostgreSQL**: Persistent storage for events, recurring patterns, and reminder metadata
- **Qdrant**: Vector storage for AI-powered semantic search and scheduling intelligence

### Downstream Enablement
- **Project Management Scenarios**: Can schedule milestones, deadlines, and team meetings
- **Automation Scenarios**: Time-triggered execution of complex workflows
- **Customer Success Scenarios**: Onboarding sequences and follow-up scheduling
- **Resource Management Scenarios**: Equipment booking, room reservations, staff scheduling

### Cross-Scenario Interactions
```yaml
provides_to:
  - scenario: ALL
    capability: Time-based workflow orchestration and event scheduling
    interface: REST API and CLI commands
    
  - scenario: project-manager-agent
    capability: Milestone scheduling with dependency tracking
    interface: /api/v1/events with project metadata
    
  - scenario: customer-onboarding
    capability: Automated multi-step scheduling based on user actions
    interface: Event automation triggers
    
consumes_from:
  - scenario: scenario-authenticator
    capability: User authentication and authorization
    fallback: Single-user mode without authentication
  - scenario: notification-hub
    capability: Multi-channel reminder delivery
    fallback: Email-only reminders via direct SMTP
```

## 🎨 Style and Branding Requirements

### UI/UX Style Guidelines
```yaml
style_profile:
  category: professional
  inspiration: Google Calendar + Calendly - clean, reliable, efficient
  
  visual_style:
    color_scheme: light with dark mode support
    typography: clean, readable sans-serif (Inter or similar)
    layout: responsive grid with sidebar navigation
    animations: subtle transitions, smooth drag-and-drop
  
  personality:
    tone: professional, efficient, helpful
    mood: organized, calm, productive
    target_feeling: Users should feel in control of their time

style_references:
  professional:
    - Google Calendar: "Clean, familiar calendar interface"
    - Calendly: "Streamlined booking and scheduling"
    - Notion Calendar: "Modern, integrated productivity aesthetic"
    - Linear: "Fast, keyboard-friendly interactions"
```

### Target Audience Alignment
- **Primary Users**: Business professionals, project managers, executive assistants
- **User Expectations**: Fast, reliable, familiar calendar interactions
- **Accessibility**: Full keyboard shortcuts, screen reader support (WCAG 2.1 AA)
- **Responsive Design**: Desktop-first with mobile optimization for viewing

## 💰 Value Proposition

### Business Value
- **Primary Value**: Transforms any scenario into a time-aware, schedulable system
- **Revenue Potential**: $15K-50K per enterprise deployment with advanced scheduling features
- **Cost Savings**: 60-120 hours of development per scenario requiring time-based workflows
- **Market Differentiator**: First AI platform with native temporal intelligence across all applications

### Technical Value
- **Reusability Score**: 9/10 - Nearly every business scenario benefits from scheduling
- **Complexity Reduction**: Time management becomes a simple API call
- **Innovation Enablement**: Enables sophisticated multi-step automation workflows

## 🧬 Evolution Path

### Version 1.0 (Current)
- Core event CRUD with PostgreSQL storage
- Basic recurring patterns (daily, weekly, monthly)
- notification-hub integration for reminders
- Professional web UI with calendar views
- REST API for programmatic access

### Version 2.0 (Planned)
- Natural language scheduling with Ollama integration
- AI-powered schedule optimization
- Advanced recurrence patterns and exceptions
- External calendar synchronization
- Event automation and scenario triggering

### Long-term Vision
- Predictive scheduling based on user behavior patterns
- Integration with video conferencing and collaboration tools
- Advanced resource management and conflict resolution
- Voice-activated scheduling through audio scenarios
- Enterprise features with team hierarchies and permissions

## 🚨 Risk Mitigation

### Technical Risks
| Risk | Probability | Impact | Mitigation |
|------|------------|--------|------------|
| Timezone complexity | High | Medium | Use industry-standard timezone libraries, extensive testing |
| Notification delivery failure | Medium | High | Retry mechanisms, multiple delivery channels |
| AI scheduling accuracy | Medium | Medium | Confidence scoring, always require user confirmation |
| Database performance with large calendars | Low | High | Proper indexing, query optimization, pagination |

### Operational Risks
- **Data Consistency**: Event updates must maintain referential integrity with reminders and embeddings
- **Security**: User calendar data is highly sensitive, requires encryption at rest and in transit
- **Scalability**: Calendar queries can become complex, need query optimization and caching
- **Reliability**: Scheduling is mission-critical for business users, requires 99.9% uptime

## ✅ Validation Criteria

### Declarative Test Specification
```yaml
version: 1.0
scenario: calendar

structure:
  required_files:
    - .vrooli/service.json
    - PRD.md
    - README.md
    - api/main.go
    - api/go.mod
    - cli/calendar
    - cli/install.sh
    - initialization/postgres/schema.sql
    - initialization/postgres/seed.sql
    - ui/index.html
    - ui/calendar.js
    - scenario-test.yaml
    
  required_dirs:
    - api
    - cli
    - initialization
    - initialization/postgres
    - ui

resources:
  required: [postgres, qdrant, scenario-authenticator, notification-hub]
  optional: [ollama]
  health_timeout: 90

tests:
  - name: "PostgreSQL calendar database is accessible"
    type: http
    service: postgres
    endpoint: /health
    method: GET
    expect:
      status: 200
      
  - name: "Event creation works with authentication"
    type: http
    service: api
    endpoint: /api/v1/events
    method: POST
    headers:
      Authorization: "Bearer ${test_token}"
    body:
      title: "Test Meeting"
      start_time: "2024-01-15T10:00:00Z"
      end_time: "2024-01-15T11:00:00Z"
    expect:
      status: 201
      body:
        success: true
        
  - name: "Event search returns results"
    type: http
    service: api
    endpoint: /api/v1/events?search=meeting
    method: GET
    headers:
      Authorization: "Bearer ${test_token}"
    expect:
      status: 200
      body:
        events: []
        
  - name: "CLI event creation works"
    type: exec
    command: ./cli/calendar event create "Team Standup" "2024-01-16T09:00:00Z" --end "2024-01-16T09:30:00Z"
    expect:
      exit_code: 0
      output_contains: ["Event created", "ID:"]
      
  - name: "Reminder notification is sent"
    type: integration
    description: "Create event with reminder, verify notification-hub receives request"
    steps:
      - create_event_with_reminder: 5 # 5 minutes before
      - wait_for_reminder_time: true
      - check_notification_delivery: true
```

## 📝 Implementation Notes

### Design Decisions
**PostgreSQL + Qdrant Architecture**: Hybrid storage for structured events + semantic search
- Alternative considered: PostgreSQL-only with full-text search
- Decision driver: AI-powered scheduling requires vector similarity search
- Trade-offs: Additional complexity for advanced semantic capabilities

**JWT Authentication Integration**: Leverage scenario-authenticator for multi-user support
- Alternative considered: Built-in user management
- Decision driver: Reuse existing authentication infrastructure
- Trade-offs: Dependency on external service for better ecosystem integration

**notification-hub Integration**: Use existing notification infrastructure
- Alternative considered: Direct email/SMS integration
- Decision driver: Professional notification capabilities without rebuilding infrastructure
- Trade-offs: External dependency for enterprise-grade delivery features

### Known Limitations
- **External Calendar Sync**: Initial version won't sync with Google/Outlook calendars
  - Workaround: iCal import/export for one-way synchronization
  - Future fix: OAuth integration with major calendar providers in v2.0
  
- **Advanced Recurrence**: Complex recurrence patterns (every 2nd Tuesday, holiday handling)
  - Workaround: Manual event creation for complex patterns
  - Future fix: Full RFC 5545 recurrence rule support

### Security Considerations
- **Data Encryption**: Calendar events encrypted at rest, sensitive meeting details protected
- **Access Control**: User isolation enforced at database and API levels
- **Audit Trail**: All calendar modifications logged with user context and timestamps

## 🔗 References

### Documentation
- README.md - Setup guide and user overview
- docs/api.md - Complete API specification
- docs/cli.md - CLI command reference
- docs/integration.md - How other scenarios can use calendar capabilities

### Related PRDs
- notification-hub/PRD.md - Notification delivery service
- scenario-authenticator/PRD.md - Authentication and user management
- project-manager-agent/PRD.md - Will integrate for milestone scheduling

---

## 📈 Progress History

### 2025-09-24 Improvement Session
**Progress**: 0% → 44% (4/9 P0 requirements functional)

**Completed Improvements**:
- ✅ Fixed compilation errors in main.go
- ✅ Implemented generateRecurringEvents function for recurring patterns
- ✅ Fixed database command in service.json lifecycle
- ✅ Verified PostgreSQL storage and CRUD operations working
- ✅ Confirmed Qdrant integration for semantic search
- ✅ Validated REST API endpoints (health, create, list, search)
- ✅ Basic recurring event creation working

**Remaining Issues**:
- ⚠️ Port environment variables not properly passed (API/UI using random ports)
- ⚠️ Multi-user authentication not configured (running single-user mode)
- ⚠️ Notification-hub integration disabled
- ⚠️ Natural language processing not available (Ollama not configured)
- ⚠️ Event-triggered automation not implemented
- ⚠️ UI calendar views not fully tested

**Next Steps**:
1. Configure scenario-authenticator for multi-user support
2. Integrate notification-hub for reminders
3. Add Ollama for natural language scheduling
4. Implement event automation triggers
5. Test and fix UI calendar views

---

**Last Updated**: 2025-09-24  
**Status**: Partially Functional (44% P0 complete)  
**Owner**: AI Agent  
**Review Cycle**: Weekly during initial development