# Product Requirements Document (PRD)

## 🎯 Capability Definition

### Core Capability
**What permanent capability does this scenario add to Vrooli?**
Time-tools provides a comprehensive temporal operations and scheduling platform that enables all Vrooli scenarios to perform timezone conversions, duration calculations, date arithmetic, scheduling optimization, and time-based analysis without implementing custom temporal logic. It supports conflict detection, optimal scheduling, timezone intelligence, and historical time analysis, making Vrooli a temporally-aware platform for all time-sensitive business operations.

### Intelligence Amplification
**How does this capability make future agents smarter?**
Time-tools amplifies agent intelligence by:
- Providing automatic conflict detection that prevents scheduling overlaps and resource conflicts
- Enabling optimal scheduling algorithms that find the best meeting times across timezones
- Supporting timezone intelligence that handles complex international scheduling seamlessly  
- Offering deadline tracking with predictive alerting for time-sensitive tasks
- Creating time analysis capabilities that reveal usage patterns and optimization opportunities
- Providing temporal reasoning that understands business hours, holidays, and working schedules

### Recursive Value
**What new scenarios become possible after this exists?**
1. **calendar-integration-hub**: Multi-platform calendar synchronization and intelligent scheduling
2. **meeting-coordinator**: AI-powered meeting scheduling with global timezone optimization
3. **deadline-management-system**: Project timeline tracking with predictive alerts
4. **time-tracking-analytics**: Work pattern analysis and productivity optimization
5. **scheduling-optimizer**: Resource allocation and appointment optimization
6. **global-business-coordinator**: International business operations with timezone awareness
7. **workflow-scheduler**: Time-based automation and workflow orchestration

## 📊 Success Metrics

### Functional Requirements
- **Must Have (P0)**
  - [ ] Timezone conversion between any global timezones with DST handling
  - [ ] Duration calculations with support for business time and working hours
  - [ ] Date arithmetic (add/subtract days, months, years) with calendar awareness
  - [ ] Custom date/time formatting with locale support
  - [ ] Parsing of various date/time formats with intelligent interpretation
  - [ ] Business time calculations excluding weekends and holidays
  - [ ] RESTful API with comprehensive temporal operation endpoints
  - [ ] CLI interface with full feature parity and human-readable output
  
- **Should Have (P1)**
  - [ ] Intelligent conflict detection for scheduling and resource allocation
  - [ ] Optimal scheduling algorithms for multi-party meetings across timezones
  - [ ] Automatic timezone detection with confidence scoring
  - [ ] Recurring event management with complex patterns (weekly, monthly, custom)
  - [ ] Deadline tracking with configurable alert systems and escalation
  - [ ] Time pattern analysis for usage insights and optimization recommendations
  - [ ] Holiday and special date management with regional calendar support
  - [ ] Time-based access control and availability windows
  
- **Nice to Have (P2)**
  - [ ] Calendar integration with Google, Outlook, and other major platforms
  - [ ] Multi-participant scheduling with preference optimization
  - [ ] Intelligent time blocking with automatic calendar organization
  - [ ] Historical time analysis with trend detection and forecasting
  - [ ] Lunar and solar calculations for astronomical applications
  - [ ] Custom calendar systems (fiscal years, academic calendars, non-Gregorian)
  - [ ] Advanced recurring patterns with AI-powered scheduling suggestions
  - [ ] Real-time collaborative scheduling with live updates

### Performance Criteria
| Metric | Target | Measurement Method |
|--------|--------|-------------------|
| Timezone Conversion | < 10ms response time | API latency monitoring |
| Conflict Detection | < 50ms for 1000 events | Scheduling algorithm benchmarks |
| Date Parsing | 99.5% accuracy on common formats | Format recognition testing |
| Recurring Events | < 100ms for complex patterns | Pattern generation performance |
| Calendar Sync | < 2 seconds for 1000 events | Integration testing |

### Quality Gates
- [ ] All P0 requirements implemented with comprehensive timezone testing
- [ ] Integration tests pass with PostgreSQL, Redis, and calendar APIs
- [ ] Performance targets met with large datasets and complex schedules
- [ ] Documentation complete (API docs, CLI help, timezone references)
- [ ] Scenario can be invoked by other agents via API/CLI/SDK
- [ ] At least 5 time-dependent scenarios successfully integrated

## 🏗️ Technical Architecture

### Resource Dependencies
```yaml
required:
  - resource_name: postgres
    purpose: Store calendar events, schedules, timezone data, and temporal analytics
    integration_pattern: Temporal data warehouse with timezone-aware storage
    access_method: resource-postgres CLI commands with timestamp precision
    
  - resource_name: redis
    purpose: Cache timezone calculations, recurring events, and schedule lookups
    integration_pattern: High-speed temporal cache with expiration handling
    access_method: resource-redis CLI commands with time-based keys
    
  - resource_name: minio
    purpose: Store calendar files, exported schedules, and historical data backups
    integration_pattern: Temporal data storage with versioned calendar exports
    access_method: resource-minio CLI commands with calendar file handling
    
optional:
  - resource_name: google-calendar
    purpose: Integration with Google Calendar API for scheduling synchronization
    fallback: Manual calendar import/export via ICS files
    access_method: resource-google-calendar CLI commands
    
  - resource_name: outlook-calendar
    purpose: Integration with Microsoft Outlook/Exchange calendar systems
    fallback: Manual calendar import/export via ICS files
    access_method: resource-outlook CLI commands
    
  - resource_name: notification-service
    purpose: Send deadline alerts, meeting reminders, and schedule notifications
    fallback: Basic email notifications via SMTP
    access_method: resource-notification CLI commands
```

### Resource Integration Standards
```yaml
integration_priorities:
  1_shared_workflows:
    - workflow: meeting-scheduler.json
      location: initialization/n8n/
      purpose: Standardized multi-party meeting scheduling workflows
    - workflow: deadline-tracker.json
      location: initialization/n8n/
      purpose: Automated deadline monitoring and alerting
  
  2_resource_cli:
    - command: resource-postgres execute
      purpose: Store and query temporal data with timezone awareness
    - command: resource-redis cache
      purpose: Cache time calculations with intelligent expiration
    - command: resource-minio upload/download
      purpose: Handle calendar files and schedule exports
  
  3_direct_api:
    - justification: Calendar APIs require OAuth and real-time sync
      endpoint: Google Calendar API for live calendar integration
    - justification: Time services need direct NTP access
      endpoint: NTP servers for accurate time synchronization

shared_workflow_criteria:
  - Scheduling templates for common meeting patterns
  - Deadline management workflows with escalation rules
  - Time zone conversion workflows for global operations
  - All workflows support both one-time and recurring events
```

### Data Models
```yaml
primary_entities:
  - name: ScheduledEvent
    storage: postgres
    schema: |
      {
        id: UUID
        title: string
        description: text
        start_datetime: timestamptz
        end_datetime: timestamptz
        timezone: string
        all_day: boolean
        location: string
        attendees: jsonb
        recurrence_rule: text
        status: enum(tentative, confirmed, cancelled)
        priority: enum(low, normal, high, urgent)
        created_by: string
        created_at: timestamptz
        updated_at: timestamptz
        external_id: string
        calendar_source: string
        metadata: jsonb
      }
    relationships: Has many Reminders and ConflictDetections
    
  - name: TimezoneDef
    storage: postgres
    schema: |
      {
        id: UUID
        timezone_name: string
        utc_offset: interval
        dst_active: boolean
        dst_start: timestamp
        dst_end: timestamp
        country: string
        region: string
        city: string
        aliases: text[]
        last_updated: timestamptz
        historical_changes: jsonb
      }
    relationships: Referenced by ScheduledEvents and TimeCalculations
    
  - name: RecurrencePattern
    storage: postgres
    schema: |
      {
        id: UUID
        event_id: UUID
        pattern_type: enum(daily, weekly, monthly, yearly, custom)
        interval_value: integer
        days_of_week: integer[]
        day_of_month: integer
        week_of_month: integer
        month_of_year: integer
        end_condition: enum(never, count, until_date)
        end_count: integer
        end_date: timestamptz
        exceptions: timestamptz[]
        generated_until: timestamptz
      }
    relationships: Belongs to ScheduledEvent, generates EventOccurrences
    
  - name: TimeAnalytics
    storage: postgres
    schema: |
      {
        id: UUID
        analysis_type: enum(usage_pattern, conflict_analysis, productivity_score)
        time_period: enum(daily, weekly, monthly, quarterly)
        start_date: date
        end_date: date
        metrics: jsonb
        insights: jsonb
        recommendations: jsonb
        confidence_score: decimal(3,2)
        generated_at: timestamptz
        entity_type: enum(user, resource, location, project)
        entity_id: string
      }
    relationships: Can reference ScheduledEvents and other temporal entities
```

### API Contract
```yaml
endpoints:
  - method: POST
    path: /api/v1/time/convert
    purpose: Convert between timezones with DST handling
    input_schema: |
      {
        datetime: string,
        from_timezone: string,
        to_timezone: string,
        format: "iso8601|unix|custom",
        custom_format: string
      }
    output_schema: |
      {
        original: {
          datetime: string,
          timezone: string,
          utc_offset: string
        },
        converted: {
          datetime: string,
          timezone: string,
          utc_offset: string
        },
        dst_info: {
          original_dst: boolean,
          converted_dst: boolean
        }
      }
    sla:
      response_time: 50ms
      availability: 99.9%
      
  - method: POST
    path: /api/v1/time/duration
    purpose: Calculate time durations with business logic
    input_schema: |
      {
        start_datetime: string,
        end_datetime: string,
        timezone: string,
        calculation_type: "elapsed|business_time|working_hours",
        business_config: {
          working_days: array,
          working_hours: {start: string, end: string},
          holidays: array
        }
      }
    output_schema: |
      {
        duration: {
          total_seconds: number,
          days: integer,
          hours: integer,
          minutes: integer,
          seconds: integer,
          business_days: integer,
          working_hours: number
        },
        readable: string
      }
      
  - method: POST
    path: /api/v1/time/schedule/optimal
    purpose: Find optimal meeting times across participants
    input_schema: |
      {
        participants: [{
          timezone: string,
          availability: [{
            day: string,
            start_time: string,
            end_time: string
          }],
          preferences: {
            preferred_times: array,
            blackout_times: array
          }
        }],
        meeting_duration_minutes: integer,
        date_range: {
          start_date: string,
          end_date: string
        },
        constraints: {
          business_hours_only: boolean,
          same_day_for_all: boolean
        }
      }
    output_schema: |
      {
        optimal_times: [{
          start_datetime: string,
          end_datetime: string,
          score: number,
          participant_times: [{
            participant_id: string,
            local_time: string,
            timezone: string
          }]
        }],
        analysis: {
          total_options: integer,
          constraints_applied: array,
          optimization_factors: object
        }
      }
      
  - method: POST
    path: /api/v1/time/conflicts/detect
    purpose: Detect scheduling conflicts and overlaps
    input_schema: |
      {
        events: [{
          id: string,
          start_datetime: string,
          end_datetime: string,
          timezone: string,
          resource_ids: array,
          attendee_ids: array
        }],
        conflict_rules: {
          allow_overlaps: boolean,
          buffer_minutes: integer,
          check_resources: boolean,
          check_attendees: boolean
        }
      }
    output_schema: |
      {
        conflicts: [{
          conflict_id: UUID,
          type: "time_overlap|resource_conflict|attendee_conflict",
          severity: "warning|error|critical",
          affected_events: array,
          affected_entities: array,
          suggestions: array
        }],
        summary: {
          total_conflicts: integer,
          by_severity: object,
          resolution_required: boolean
        }
      }
      
  - method: POST
    path: /api/v1/time/recurring/generate
    purpose: Generate recurring event instances
    input_schema: |
      {
        base_event: {
          title: string,
          start_datetime: string,
          end_datetime: string,
          timezone: string
        },
        recurrence: {
          frequency: "daily|weekly|monthly|yearly",
          interval: integer,
          by_weekday: array,
          by_month_day: array,
          count: integer,
          until: string,
          exceptions: array
        }
      }
    output_schema: |
      {
        pattern_id: UUID,
        generated_events: [{
          occurrence_id: UUID,
          start_datetime: string,
          end_datetime: string,
          sequence_number: integer
        }],
        next_generation_date: string,
        total_future_events: integer
      }
      
  - method: POST
    path: /api/v1/time/analytics/usage
    purpose: Analyze time usage patterns and productivity
    input_schema: |
      {
        entity_type: "user|resource|project",
        entity_id: string,
        time_period: {
          start_date: string,
          end_date: string
        },
        analysis_types: ["utilization", "patterns", "productivity", "conflicts"],
        granularity: "hourly|daily|weekly|monthly"
      }
    output_schema: |
      {
        utilization: {
          total_scheduled_hours: number,
          available_hours: number,
          utilization_rate: number,
          peak_hours: array,
          low_utilization_periods: array
        },
        patterns: {
          common_meeting_times: array,
          scheduling_trends: object,
          seasonal_variations: object
        },
        recommendations: array
      }
```

### Event Interface
```yaml
published_events:
  - name: time.event.scheduled
    payload: {event_id: UUID, start_time: string, end_time: string, attendees: array}
    subscribers: [calendar-sync, notification-service, resource-manager]
    
  - name: time.conflict.detected
    payload: {conflict_id: UUID, type: string, severity: string, affected_events: array}
    subscribers: [alert-manager, schedule-optimizer, conflict-resolver]
    
  - name: time.deadline.approaching
    payload: {deadline_id: UUID, entity_id: string, time_remaining: object, urgency: string}
    subscribers: [notification-service, task-manager, escalation-handler]
    
  - name: time.pattern.identified
    payload: {pattern_type: string, entity_id: string, insights: object, confidence: number}
    subscribers: [analytics-dashboard, optimization-engine, insight-generator]
    
consumed_events:
  - name: project.deadline_set
    action: Create deadline tracking and alert schedules
    
  - name: user.calendar_updated
    action: Sync calendar changes and detect new conflicts
```

## 🖥️ CLI Interface Contract

### Command Structure
```yaml
cli_binary: time-tools
install_script: cli/install.sh

required_commands:
  - name: status
    description: Show temporal system status and timezone data health
    flags: [--json, --verbose, --timezone-check]
    
  - name: help
    description: Display command help and usage
    flags: [--all, --command <name>]
    
  - name: version
    description: Show CLI and API version information
    flags: [--json]

custom_commands:
  - name: convert
    description: Convert between timezones
    api_endpoint: /api/v1/time/convert
    arguments:
      - name: datetime
        type: string
        required: true
        description: Date and time to convert
      - name: from_tz
        type: string
        required: true
        description: Source timezone
      - name: to_tz
        type: string
        required: true
        description: Target timezone
    flags:
      - name: --format
        description: Output format (iso8601, unix, custom)
      - name: --custom-format
        description: Custom datetime format string
      - name: --dst-info
        description: Include daylight saving time information
    
  - name: duration
    description: Calculate time durations
    api_endpoint: /api/v1/time/duration
    arguments:
      - name: start
        type: string
        required: true
        description: Start date and time
      - name: end
        type: string
        required: true
        description: End date and time
    flags:
      - name: --timezone
        description: Timezone for calculation
      - name: --business-time
        description: Calculate business hours only
      - name: --working-days
        description: Working days (comma-separated)
      - name: --holidays
        description: Holiday dates to exclude
      
  - name: add
    description: Add time to a date
    arguments:
      - name: datetime
        type: string
        required: true
        description: Base date and time
      - name: amount
        type: string
        required: true
        description: Amount to add (e.g., '2 days', '3 hours')
    flags:
      - name: --timezone
        description: Timezone for calculation
      - name: --format
        description: Output format
      
  - name: subtract
    description: Subtract time from a date
    arguments:
      - name: datetime
        type: string
        required: true
        description: Base date and time
      - name: amount
        type: string
        required: true
        description: Amount to subtract (e.g., '1 week', '30 minutes')
    flags:
      - name: --timezone
        description: Timezone for calculation
      - name: --format
        description: Output format
        
  - name: schedule
    description: Scheduling operations
    subcommands:
      - name: find
        description: Find optimal meeting times
      - name: conflicts
        description: Detect scheduling conflicts
      - name: recurring
        description: Generate recurring events
        
  - name: parse
    description: Parse various date/time formats
    arguments:
      - name: input
        type: string
        required: true
        description: Date/time string to parse
    flags:
      - name: --timezone
        description: Assumed timezone if not specified
      - name: --format-hint
        description: Format hint for parsing
      - name: --strict
        description: Use strict parsing mode
        
  - name: format
    description: Format dates and times
    arguments:
      - name: datetime
        type: string
        required: true
        description: Date/time to format
    flags:
      - name: --format
        description: Output format string
      - name: --locale
        description: Locale for formatting
      - name: --timezone
        description: Display timezone
        
  - name: now
    description: Get current time in various formats
    flags:
      - name: --timezone
        description: Display timezone
      - name: --format
        description: Output format
      - name: --utc
        description: Show UTC time
        
  - name: zones
    description: Timezone operations
    subcommands:
      - name: list
        description: List available timezones
      - name: info
        description: Show timezone information
      - name: search
        description: Search timezones by location
```

## 🔄 Integration Requirements

### Upstream Dependencies
**What capabilities must exist before this can function?**
- **PostgreSQL**: Temporal data storage with timezone-aware capabilities
- **Redis**: High-speed caching for time calculations and schedules
- **MinIO**: Storage for calendar files and historical data

### Downstream Enablement
**What future capabilities does this unlock?**
- **calendar-integration-hub**: Multi-platform calendar synchronization
- **meeting-coordinator**: AI-powered global meeting scheduling
- **deadline-management-system**: Project timeline tracking and alerts
- **time-tracking-analytics**: Work pattern analysis and productivity insights
- **scheduling-optimizer**: Resource allocation and appointment optimization
- **workflow-scheduler**: Time-based automation and orchestration

### Cross-Scenario Interactions
```yaml
provides_to:
  - scenario: calendar-integration-hub
    capability: Timezone conversion and scheduling optimization
    interface: API/CLI
    
  - scenario: meeting-coordinator
    capability: Conflict detection and optimal time finding
    interface: API/Events
    
  - scenario: deadline-management-system
    capability: Deadline tracking and temporal calculations
    interface: API/Workflows
    
  - scenario: workflow-scheduler
    capability: Time-based triggers and scheduling logic
    interface: Events/API
    
consumes_from:
  - scenario: notification-service
    capability: Alert delivery for deadlines and reminders
    fallback: Basic logging only
    
  - scenario: data-tools
    capability: Time series analysis and pattern recognition
    fallback: Basic statistical analysis only
```

## 🎨 Style and Branding Requirements

### UI/UX Style Guidelines
```yaml
style_profile:
  category: professional
  inspiration: Calendar applications (Google Calendar, Outlook, Calendly)
  
  visual_style:
    color_scheme: light
    typography: system font with clear time display
    layout: calendar-based
    animations: smooth

personality:
  tone: efficient
  mood: organized
  target_feeling: Reliable and precise
```

### Target Audience Alignment
- **Primary Users**: Business professionals, project managers, schedulers, executives
- **User Expectations**: Accuracy, reliability, intuitive scheduling interfaces
- **Accessibility**: WCAG AA compliance, screen reader calendar support
- **Responsive Design**: Desktop-optimized with mobile scheduling views

## 💰 Value Proposition

### Business Value
- **Primary Value**: Comprehensive time management without complex scheduling software
- **Revenue Potential**: $10K - $40K per enterprise deployment
- **Cost Savings**: 80% reduction in scheduling coordination time
- **Market Differentiator**: AI-powered scheduling with global timezone intelligence

### Technical Value
- **Reusability Score**: 9/10 - Almost all scenarios need time operations
- **Complexity Reduction**: Single API for all temporal operations
- **Innovation Enablement**: Foundation for time-aware business applications

## 🧬 Evolution Path

### Version 1.0 (Current)
- Core temporal operations (conversion, duration, formatting)
- Basic scheduling and conflict detection
- PostgreSQL integration for temporal data storage
- CLI and API interfaces with comprehensive features

### Version 2.0 (Planned)
- Advanced calendar integrations with major platforms
- AI-powered scheduling optimization
- Real-time collaborative scheduling
- Advanced analytics and productivity insights

### Long-term Vision
- Become the "Calendly + Temporal Intelligence of Vrooli"
- Predictive scheduling with machine learning
- Universal calendar protocol for all time-based operations
- Seamless integration with global business workflows

## 🔄 Scenario Lifecycle Integration

### Direct Scenario Deployment
```yaml
direct_execution:
  supported: true
  structure_compliance:
    - PostgreSQL schema with timezone-aware temporal data
    - Redis configuration for time calculation caching
    - MinIO bucket setup for calendar files and exports
    - Timezone database with automatic updates
    
  deployment_targets:
    - local: Docker Compose with timezone data
    - kubernetes: Helm chart with persistent timezone storage
    - cloud: Serverless temporal functions
    
  revenue_model:
    - type: scheduling-based
    - pricing_tiers:
        - individual: Basic scheduling, limited participants
        - team: Advanced scheduling, calendar integration
        - enterprise: Unlimited with analytics and optimization
```

### Capability Discovery
```yaml
discovery:
  registry_entry:
    name: time-tools
    category: foundation
    capabilities: [convert, duration, schedule, conflicts, recurring, analytics]
    interfaces:
      - api: http://localhost:${TIME_TOOLS_PORT}/api/v1
      - cli: time-tools
      - events: time.*
      - calendar: icalendar://localhost:${TIME_TOOLS_PORT}/calendar
      
  metadata:
    description: Comprehensive temporal operations and scheduling platform
    keywords: [time, timezone, scheduling, calendar, duration, conflicts]
    dependencies: [postgres, redis, minio]
    enhances: [all time-sensitive scenarios]
```

## 🚨 Risk Mitigation

### Technical Risks
| Risk | Probability | Impact | Mitigation |
|------|------------|--------|------------|
| Timezone data corruption | Low | High | Regular timezone database updates, validation |
| DST calculation errors | Medium | High | Multiple timezone libraries, cross-validation |
| Calendar sync failures | Medium | Medium | Retry mechanisms, offline capability |
| Performance with large schedules | Medium | Medium | Efficient indexing, pagination |

### Operational Risks
- **Data Accuracy**: Continuous timezone database updates and validation
- **Calendar Integration**: Robust error handling for external calendar APIs
- **Performance Scaling**: Efficient algorithms for large-scale scheduling

## ✅ Validation Criteria

### Declarative Test Specification
```yaml
version: 1.0
scenario: time-tools

structure:
  required_files:
    - .vrooli/service.json
    - PRD.md
    - api/main.go
    - cli/time-tools
    - initialization/storage/postgres/schema.sql
    - scenario-test.yaml
    
resources:
  required: [postgres, redis, minio]
  optional: [google-calendar, outlook-calendar, notification-service]
  health_timeout: 60

tests:
  - name: "Timezone conversion accuracy"
    type: http
    service: api
    endpoint: /api/v1/time/convert
    method: POST
    body:
      datetime: "2025-01-15T10:00:00"
      from_timezone: "America/New_York"
      to_timezone: "Asia/Tokyo"
    expect:
      status: 200
      body:
        converted:
          datetime: "2025-01-16T00:00:00"
        
  - name: "Duration calculation works"
    type: http
    service: api
    endpoint: /api/v1/time/duration
    method: POST
    body:
      start_datetime: "2025-01-01T09:00:00"
      end_datetime: "2025-01-01T17:00:00"
      calculation_type: "working_hours"
    expect:
      status: 200
      body:
        duration:
          working_hours: 8
          
  - name: "Conflict detection functionality"
    type: http
    service: api
    endpoint: /api/v1/time/conflicts/detect
    method: POST
    body:
      events:
        - id: "event1"
          start_datetime: "2025-01-15T10:00:00"
          end_datetime: "2025-01-15T11:00:00"
        - id: "event2"
          start_datetime: "2025-01-15T10:30:00"
          end_datetime: "2025-01-15T11:30:00"
    expect:
      status: 200
      body:
        conflicts: [type: array, minItems: 1]
```

## 📝 Implementation Notes

### Design Decisions
**Timezone Handling**: Comprehensive timezone database with automatic updates
- Alternative considered: Basic UTC-only operations
- Decision driver: Global business requirements and DST complexity
- Trade-offs: Storage and complexity for accuracy and usability

**Scheduling Algorithm**: Multi-factor optimization for meeting scheduling
- Alternative considered: Simple availability checking
- Decision driver: Need for intelligent scheduling across timezones
- Trade-offs: Computation complexity for optimal user experience

### Known Limitations
- **Historical Timezone Changes**: Limited to timezone database coverage
  - Workaround: Regular database updates and validation
  - Future fix: Real-time timezone service integration

### Security Considerations
- **Calendar Access**: Secure OAuth handling for calendar integrations
- **Personal Data**: Privacy-conscious handling of scheduling information
- **Audit Trail**: Complete logging of all scheduling operations

## 🔗 References

### Documentation
- README.md - Quick start and common time operations
- docs/api.md - Complete API reference with timezone examples
- docs/cli.md - CLI usage and scheduling workflows
- docs/timezones.md - Timezone database and conversion reference

### Related PRDs
- scenarios/calendar-integration-hub/PRD.md - Calendar synchronization
- scenarios/notification-service/PRD.md - Alert and reminder delivery

---

**Last Updated**: 2025-09-09  
**Status**: Draft  
**Owner**: AI Agent  
**Review Cycle**: Weekly validation against implementation