# Product Requirements Document (PRD)

## 🎯 Capability Definition

### Core Capability
**What permanent capability does this scenario add to Vrooli?**
Date Night Planner adds intelligent relationship date planning capability that learns from couples' preferences, leverages local information, and orchestrates multi-step date experiences. It transforms the decision fatigue of date planning into personalized, memorable experiences while building a knowledge base of successful romantic activities.

### Intelligence Amplification
**How does this capability make future agents smarter?**
- **Preference Learning Engine**: Builds relationship intelligence patterns that other scenarios can use for personalized recommendations
- **Multi-Scenario Orchestration**: Demonstrates how to coordinate multiple specialized scenarios (authenticator, contact-book, local-info-scout, research-assistant) to create compound capabilities
- **Relationship Context**: Adds emotional intelligence and relationship awareness to Vrooli's decision-making capabilities
- **Temporal Planning**: Introduces sophisticated multi-step, time-bound activity coordination that applies to any sequential planning scenario

### Recursive Value
**What new scenarios become possible after this exists?**
- **Anniversary Planner**: Year-long celebration planning with milestone tracking
- **Event Coordinator**: Corporate and social event planning using the relationship intelligence patterns  
- **Travel Planner**: Multi-day itinerary planning leveraging the temporal orchestration engine
- **Family Activity Planner**: Child-friendly activity coordination using preference learning
- **Group Social Planner**: Friend group activity coordination with multiple preference sets

## 📊 Success Metrics

### Functional Requirements
- **Must Have (P0)**
  - [ ] Generate personalized date suggestions based on partner preferences from contact-book
  - [ ] Integrate with local-info-scout for real-time venue availability and recommendations
  - [ ] Multi-tenant authentication via scenario-authenticator for couple privacy
  - [ ] Preference learning that improves suggestions over time
  - [ ] Date memory storage with photos, notes, and ratings
  
- **Should Have (P1)**
  - [ ] Weather-aware planning with backup indoor alternatives
  - [ ] Budget-conscious suggestions with cost estimates
  - [ ] Calendar integration for optimal timing
  - [ ] Surprise mode vs collaborative planning options
  - [ ] Social media integration for date inspiration
  
- **Nice to Have (P2)**
  - [ ] AR/VR date experiences for long-distance couples
  - [ ] Gift suggestion integration based on date outcomes
  - [ ] Anniversary and milestone reminder system
  - [ ] Date night recipe suggestions for at-home dates
  - [ ] Integration with ride-sharing and reservation platforms

### Performance Criteria
| Metric | Target | Measurement Method |
|--------|--------|-------------------|
| Response Time | < 2000ms for date suggestions | API monitoring |
| Suggestion Relevance | > 85% user satisfaction rating | Post-date feedback |
| Integration Latency | < 500ms for external scenario calls | Service monitoring |
| Memory Usage | < 512MB during peak usage | System monitoring |

### Quality Gates
- [ ] All P0 requirements implemented and tested
- [ ] Integration tests pass with all required scenarios (authenticator, contact-book, local-info-scout, research-assistant)
- [ ] Performance targets met under load
- [ ] Documentation complete (README, API docs, CLI help)
- [ ] Scenario can be invoked by other agents via API/CLI

## 🏗️ Technical Architecture

### Resource Dependencies
```yaml
required:
  - resource_name: postgres
    purpose: Store date plans, preferences, memories, and user relationships
    integration_pattern: Direct database connection via Go API
    access_method: Connection string from environment
    
  - resource_name: n8n
    purpose: Orchestrate complex date planning workflows and external integrations
    integration_pattern: Shared workflows + scenario-specific workflows
    access_method: Webhook triggers and manual execution
    
  - resource_name: ollama
    purpose: Generate creative date ideas and personalized suggestions
    integration_pattern: Shared ollama.json workflow
    access_method: initialization/n8n/ollama.json via webhook
    
optional:
  - resource_name: redis
    purpose: Cache frequently accessed preferences and venue data
    fallback: Direct database queries with acceptable performance impact
    access_method: Redis client in Go API
    
  - resource_name: qdrant
    purpose: Semantic search of date activities and venue matching
    fallback: PostgreSQL full-text search with reduced relevance accuracy
    access_method: Vector embeddings via REST API
```

### Resource Integration Standards
```yaml
# Priority order for resource access (MUST follow this hierarchy):
integration_priorities:
  1_shared_workflows:     # FIRST: Use existing shared n8n workflows
    - workflow: ollama.json
      location: initialization/n8n/
      purpose: AI-powered date suggestion generation and preference analysis
  
  2_resource_cli:        # SECOND: Use resource CLI commands
    - command: resource-postgres [action]
      purpose: Database health checks and maintenance
    - command: resource-n8n list-workflows
      purpose: Workflow discovery and health validation
  
  3_direct_api:          # LAST: Direct API only when necessary
    - justification: High-frequency preference lookups require direct DB access for performance
      endpoint: PostgreSQL connection for real-time preference queries

# Shared workflow guidelines:
shared_workflow_criteria:
  - preference-analyzer.json: Analyzes couple compatibility and suggests date types
  - venue-matcher.json: Matches date preferences to local venues using semantic search
  - date-orchestrator.json: Coordinates multi-step date experiences with timing
```

### Data Models
```yaml
# Core data structures that define the capability
primary_entities:
  - name: Couple
    storage: postgres
    schema: |
      {
        id: UUID,
        auth_user_ids: [UUID], // Links to scenario-authenticator users
        relationship_start: DATE,
        shared_preferences: JSONB,
        privacy_settings: JSONB,
        created_at: TIMESTAMP,
        updated_at: TIMESTAMP
      }
    relationships: Links to both partners' individual preferences via contact-book
    
  - name: DatePlan
    storage: postgres  
    schema: |
      {
        id: UUID,
        couple_id: UUID,
        title: STRING,
        description: TEXT,
        planned_date: TIMESTAMP,
        activities: JSONB[], // Array of activity objects
        estimated_cost: DECIMAL,
        estimated_duration: INTERVAL,
        weather_backup: JSONB,
        status: ENUM['planned', 'completed', 'cancelled'],
        rating: INTEGER, // 1-5 stars post-date
        memory_notes: TEXT,
        photos: STRING[], // URLs to stored photos
        created_at: TIMESTAMP
      }
    relationships: Belongs to Couple, references venues from local-info-scout
    
  - name: Preference
    storage: postgres
    schema: |
      {
        id: UUID,
        couple_id: UUID,
        category: STRING, // 'cuisine', 'activity_type', 'ambiance', etc.
        preference_key: STRING,
        preference_value: STRING,
        confidence_score: DECIMAL, // 0.0-1.0, increases with successful dates
        learned_from: STRING, // 'explicit', 'feedback', 'behavior'
        created_at: TIMESTAMP,
        updated_at: TIMESTAMP
      }
    relationships: Belongs to Couple, used by preference learning algorithm
```

### API Contract
```yaml
# Defines how other scenarios/agents can use this capability
endpoints:
  - method: POST
    path: /api/v1/dates/suggest
    purpose: Generate personalized date suggestions for a couple
    input_schema: |
      {
        couple_id: UUID,
        date_type?: 'romantic' | 'adventure' | 'cultural' | 'casual',
        budget_max?: NUMBER,
        preferred_date?: ISO_DATE,
        weather_preference?: 'indoor' | 'outdoor' | 'flexible'
      }
    output_schema: |
      {
        suggestions: [{
          title: STRING,
          description: STRING,
          activities: [ACTIVITY_OBJECT],
          estimated_cost: NUMBER,
          estimated_duration: STRING,
          confidence_score: NUMBER,
          weather_backup?: ACTIVITY_OBJECT
        }],
        personalization_factors: [STRING]
      }
    sla:
      response_time: 2000ms
      availability: 99.5%
      
  - method: POST
    path: /api/v1/dates/plan
    purpose: Create and save a complete date plan
    input_schema: |
      {
        couple_id: UUID,
        selected_suggestion: SUGGESTION_OBJECT,
        planned_date: ISO_TIMESTAMP,
        customizations?: JSONB
      }
    output_schema: |
      {
        date_plan: DATE_PLAN_OBJECT,
        calendar_invites?: [CALENDAR_OBJECT],
        reservations?: [RESERVATION_OBJECT]
      }
    sla:
      response_time: 3000ms
      availability: 99.0%
```

### Event Interface
```yaml
# Events this capability publishes for others to consume
published_events:
  - name: date_planner.suggestion.generated
    payload: { couple_id: UUID, suggestion_count: NUMBER, preferences_used: [STRING] }
    subscribers: [research-assistant for trend analysis]
    
  - name: date_planner.date.completed
    payload: { couple_id: UUID, date_plan_id: UUID, rating: NUMBER, feedback: STRING }
    subscribers: [contact-book for relationship insight updates]
    
consumed_events:
  - name: contact_book.preference.updated
    action: Refresh couple preferences and re-calculate suggestion confidence scores
  - name: local_info_scout.venue.discovered
    action: Add new venue options to date planning algorithm
```

## 🖥️ CLI Interface Contract

### Command Structure
```yaml
# Primary CLI executable name and pattern
cli_binary: date-night-planner
install_script: cli/install.sh

# Core commands that MUST be implemented:
required_commands:
  - name: status
    description: Show operational status and resource health
    flags: [--json, --verbose]
    
  - name: help
    description: Display command help and usage
    flags: [--all, --command <name>]
    
  - name: version
    description: Show CLI and API version information
    flags: [--json]

# Scenario-specific commands (must mirror API endpoints):
custom_commands:
  - name: suggest
    description: Generate date suggestions for a couple
    api_endpoint: /api/v1/dates/suggest
    arguments:
      - name: couple_id
        type: string
        required: true
        description: UUID of the couple
    flags:
      - name: --budget
        description: Maximum budget for the date
      - name: --type
        description: Type of date (romantic, adventure, cultural, casual)
      - name: --date
        description: Preferred date in YYYY-MM-DD format
    output: JSON array of date suggestions
    
  - name: plan
    description: Create and save a complete date plan
    api_endpoint: /api/v1/dates/plan
    arguments:
      - name: couple_id
        type: string
        required: true
        description: UUID of the couple
      - name: suggestion_id
        type: string
        required: true
        description: ID of selected suggestion
    flags:
      - name: --date
        description: Planned date and time in ISO format
      - name: --customize
        description: JSON string of customizations
    output: Complete date plan with reservations and calendar events
```

## 🔄 Integration Requirements

### Upstream Dependencies
**What capabilities must exist before this can function?**
- **scenario-authenticator**: Multi-tenant authentication for couples, privacy controls for surprise planning
- **contact-book**: Partner preference storage, relationship history, dietary restrictions, and personal details
- **local-info-scout**: Real-time venue availability, local event discovery, weather-aware recommendations
- **research-assistant**: Creative date idea research, trend analysis, and activity inspiration

### Downstream Enablement
**What future capabilities does this unlock?**
- **Event Planning Orchestration**: Pattern for coordinating complex multi-step experiences
- **Relationship Intelligence**: Framework for preference learning and compatibility analysis
- **Temporal Activity Coordination**: Reusable system for time-bound sequential activity planning
- **Multi-Scenario Integration Pattern**: Template for building capabilities that leverage multiple existing scenarios

### Cross-Scenario Interactions
```yaml
# How this scenario enhances other scenarios
provides_to:
  - scenario: personal-relationship-manager
    capability: Date history and relationship milestone tracking
    interface: API + Events
    
  - scenario: calendar
    capability: Romantic occasion reminder and date scheduling
    interface: Calendar integration API
    
  - scenario: budget-tracker
    capability: Date expense categorization and romantic spending insights
    interface: API webhooks
    
consumes_from:
  - scenario: scenario-authenticator
    capability: Couple authentication and privacy controls
    fallback: Single-user mode with reduced privacy features
    
  - scenario: contact-book
    capability: Partner preferences and relationship data
    fallback: Manual preference entry with reduced personalization
    
  - scenario: local-info-scout
    capability: Venue discovery and local event information
    fallback: Static venue database with no real-time updates
    
  - scenario: research-assistant
    capability: Creative date idea research and trend analysis
    fallback: Curated date idea database with no trend awareness
```

## 🎨 Style and Branding Requirements

### UI/UX Style Guidelines
```yaml
# Define the visual and experiential personality of this scenario
style_profile:
  category: playful
  inspiration: "study-buddy's cute, cozy aesthetic meets modern dating app sophistication"
  
  # Visual characteristics:
  visual_style:
    color_scheme: pastel
    primary_colors: ["#FFB6C1", "#E6E6FA", "#F0E68C", "#98FB98"]  # Soft pink, lavender, khaki, mint
    secondary_colors: ["#FFF8DC", "#F5DEB3", "#DDA0DD"]  # Cornsilk, wheat, plum
    accent_colors: ["#FF69B4", "#9370DB"]  # Hot pink, medium purple for CTAs
    typography: modern
    layout: card-based
    animations: playful
  
  # Personality traits:
  personality:
    tone: romantic yet playful
    mood: warm and exciting
    target_feeling: "Butterflies in your stomach excitement about your next date"

# Style examples from existing scenarios:
style_references:
  primary_inspiration: 
    - study-buddy: "Cute, lo-fi, cozy aesthetic with pastel colors and friendly interactions"
  complementary_elements:
    - picker-wheel: "Playful card interactions and engaging animations"
    - calendar: "Clean date/time selection interfaces"
```

### Target Audience Alignment
- **Primary Users**: Couples in committed relationships looking to enhance their romantic life
- **User Expectations**: Instagram-worthy aesthetic that makes planning feel fun rather than overwhelming
- **Accessibility**: WCAG AA compliance, with particular attention to color contrast given pastel palette
- **Responsive Design**: Mobile-first (70% of date planning happens on phones), with desktop for detailed planning

### Brand Consistency Rules
- **Scenario Identity**: Romantic, thoughtful, and effortlessly cute
- **Vrooli Integration**: Maintains Vrooli's intelligent system aesthetic while adding warm personality
- **Professional vs Fun**: Leans heavily toward fun and emotional engagement - this is about love, not business
- **Visual Metaphors**: Hearts, calendar dates, map pins, and card stacks for date options

## 💰 Value Proposition

### Business Value
- **Primary Value**: Eliminates decision fatigue and relationship planning stress for couples
- **Revenue Potential**: $15K - $35K per deployment (relationship/lifestyle SaaS commands premium pricing)
- **Cost Savings**: 2-4 hours saved per month per couple in planning time, plus improved relationship satisfaction
- **Market Differentiator**: First AI-powered date planner with true personalization and multi-step orchestration

### Technical Value
- **Reusability Score**: 8/10 - Temporal planning and preference learning patterns apply to many scenario types
- **Complexity Reduction**: Transforms complex multi-scenario integration into simple API calls
- **Innovation Enablement**: Establishes patterns for relationship intelligence and emotional AI

## 🧬 Evolution Path

### Version 1.0 (Current)
- Core date suggestion and planning capability
- Basic integration with all four upstream scenarios
- Essential preference learning and memory storage
- Pastel UI with card-based date browsing

### Version 2.0 (Planned)
- Advanced AI personality matching for optimal date types
- Integration with reservation and ride-sharing platforms
- Social media inspiration integration
- Advanced weather and seasonal planning

### Long-term Vision
- AR/VR date experiences for long-distance relationships
- Integration with gift suggestion and anniversary planning
- Expansion to friend group and family activity planning
- Become the foundation for all relationship and social planning in Vrooli

## 🔄 Scenario Lifecycle Integration

### Direct Scenario Deployment
```yaml
# Requirements for direct scenario execution:
direct_execution:
  supported: true
  structure_compliance:
    - service.json with complete metadata and resource dependencies
    - All required initialization files for workflows and database schema
    - Deployment scripts (startup.sh, monitor.sh) with dependency health checks
    - Health check endpoints for API and integration status
    
  deployment_targets:
    - local: Docker Compose based with all scenario dependencies
    - kubernetes: Helm chart generation with service mesh integration
    - cloud: AWS/GCP/Azure templates with managed database and workflow services
    
  revenue_model:
    - type: subscription
    - pricing_tiers: 
      - basic: $9.99/month (unlimited date suggestions, basic planning)
      - premium: $19.99/month (advanced personalization, reservation integration)
      - couples: $29.99/month (dual-user accounts, surprise planning, anniversary tracking)
    - trial_period: 14 days with full premium access
```

### Capability Discovery
```yaml
# How other scenarios/agents discover and use this capability:
discovery:
  registry_entry:
    name: date-night-planner
    category: personal-life
    capabilities: 
      - romantic_activity_planning
      - couple_preference_learning
      - multi_step_experience_orchestration
      - relationship_intelligence
    interfaces:
      - api: http://localhost:${API_PORT}/api/v1/
      - cli: date-night-planner
      - events: date_planner.*
      
  metadata:
    description: "AI-powered date planning with couple personalization and local integration"
    keywords: [dating, relationships, planning, personalization, local-events, couples]
    dependencies: [scenario-authenticator, contact-book, local-info-scout, research-assistant]
    enhances: [personal-relationship-manager, calendar, budget-tracker]
```

### Version Management
```yaml
# Compatibility and upgrade paths:
versioning:
  current: 1.0.0
  minimum_compatible: 1.0.0
  
  breaking_changes: []  # None planned for v1.0
      
  deprecations: []  # None planned for v1.0
```

## 🚨 Risk Mitigation

### Technical Risks
| Risk | Probability | Impact | Mitigation |
|------|------------|--------|------------|
| Upstream scenario unavailability | Medium | High | Graceful degradation with reduced personalization |
| Local venue data staleness | High | Medium | Caching with TTL and fallback to research-assistant |
| Preference learning accuracy | Medium | Medium | Explicit feedback collection and confidence scoring |
| Multi-step orchestration failure | Low | High | Atomic operations with rollback capabilities |

### Operational Risks
- **Privacy Concerns**: Couple data is highly sensitive - implement end-to-end encryption for surprise planning
- **Integration Complexity**: Four-scenario dependency requires robust health checking and fallback mechanisms  
- **Personalization Expectations**: Users expect high accuracy - must clearly communicate learning curve
- **Seasonal Variations**: Date suggestions must adapt to weather, holidays, and local seasonal patterns

## ✅ Validation Criteria

### Declarative Test Specification
```yaml
# REQUIRED: scenario-test.yaml in scenario root
version: 1.0
scenario: date-night-planner

# Structure validation - files and directories that MUST exist:
structure:
  required_files:
    - .vrooli/service.json
    - PRD.md
    - api/main.go
    - api/go.mod
    - cli/date-night-planner
    - cli/install.sh
    - initialization/storage/postgres/schema.sql
    - initialization/automation/n8n/date-orchestrator.json
    - initialization/automation/n8n/preference-analyzer.json
    - initialization/automation/n8n/venue-matcher.json
    - scenario-test.yaml
    
  required_dirs:
    - api
    - cli
    - initialization/automation/n8n
    - initialization/storage/postgres
    - ui
    - test

# Resource validation:
resources:
  required: [postgres, n8n, ollama]
  optional: [redis, qdrant]
  health_timeout: 90  # Extra time for n8n workflow initialization

# Declarative tests:
tests:
  # Resource health checks:
  - name: "PostgreSQL is accessible"
    type: http
    service: postgres
    endpoint: /health
    method: GET
    expect:
      status: 200
      
  - name: "N8N workflows are active"
    type: exec
    command: resource-n8n list-workflows --active
    expect:
      exit_code: 0
      output_contains: ["date-orchestrator", "preference-analyzer", "venue-matcher"]
      
  # API endpoint tests:
  - name: "Date suggestion API responds correctly"
    type: http
    service: api
    endpoint: /api/v1/dates/suggest
    method: POST
    headers:
      Content-Type: application/json
    body:
      couple_id: "test-couple-123"
      date_type: "romantic"
      budget_max: 100
    expect:
      status: 200
      body:
        suggestions: []
        
  # CLI command tests:
  - name: "CLI suggest command executes"
    type: exec
    command: ./cli/date-night-planner suggest test-couple-123 --budget 50 --json
    expect:
      exit_code: 0
      output_contains: ["suggestions"]
      
  # Database tests:
  - name: "Database schema is initialized"
    type: sql
    service: postgres
    query: "SELECT COUNT(*) FROM information_schema.tables WHERE table_schema = 'public' AND table_name IN ('couples', 'date_plans', 'preferences')"
    expect:
      rows: 
        - count: 3
        
  # Integration tests:
  - name: "Scenario-authenticator integration works"
    type: http
    service: api
    endpoint: /api/v1/couples/authenticate
    method: POST
    body:
      user_tokens: ["test-token-1", "test-token-2"]
    expect:
      status: 201
      body:
        couple_id: "string"
```

### Performance Validation
- [ ] Date suggestion API responds within 2000ms for 95% of requests
- [ ] Memory usage stays below 512MB during concurrent date planning sessions
- [ ] Integration calls to upstream scenarios complete within 500ms each
- [ ] Database queries for preference lookups complete within 100ms

### Integration Validation
- [ ] Successfully authenticates couples via scenario-authenticator
- [ ] Retrieves partner preferences from contact-book
- [ ] Discovers local venues via local-info-scout
- [ ] Gets creative date ideas from research-assistant
- [ ] All n8n workflows deploy and activate successfully
- [ ] CLI commands provide comprehensive help and error messages

### Capability Verification
- [ ] Generates personalized date suggestions that improve over time
- [ ] Stores and retrieves date memories with photos and ratings
- [ ] Handles surprise planning with appropriate privacy controls
- [ ] Provides weather-aware suggestions with backup options
- [ ] Creates coherent multi-step date experiences

## 📝 Implementation Notes

### Design Decisions
**Multi-Scenario Integration**: Chose API-based integration over direct database sharing to maintain scenario independence and enable graceful degradation when upstream scenarios are unavailable.

**Preference Learning Algorithm**: Implemented confidence scoring system that weights explicit preferences higher than inferred ones, with decay over time to adapt to changing tastes.

**Privacy Architecture**: Surprise planning mode encrypts date details from one partner while maintaining full access for the planning partner.

### Known Limitations
- **Cold Start Problem**: New couples have no preference history - requires 3-5 date feedback cycles for optimal personalization
  - Workaround: Onboarding questionnaire with compatibility analysis
  - Future fix: Import preferences from social media and dating app history
  
- **Local Info Dependency**: Date quality heavily dependent on local-info-scout data completeness
  - Workaround: Fall back to research-assistant for creative alternatives  
  - Future fix: Direct integration with Google Places and Yelp APIs

### Security Considerations
- **Data Protection**: All couple data encrypted at rest, surprise plans use separate encryption keys
- **Access Control**: Couple-level authentication with individual partner permissions for surprise features
- **Audit Trail**: All date suggestions, plans, and modifications logged for preference learning and debugging

## 🔗 References

### Documentation
- README.md - User-facing overview with couple onboarding guide
- docs/api.md - Complete API specification with authentication examples
- docs/cli.md - CLI documentation with date planning workflows
- docs/integration.md - Multi-scenario integration patterns and troubleshooting

### Related PRDs
- scenario-authenticator/PRD.md - Multi-tenant authentication patterns
- contact-book/PRD.md - Relationship data storage and preference management
- local-info-scout/PRD.md - Venue discovery and local event integration
- research-assistant/PRD.md - Creative research and trend analysis capabilities

### External Resources
- Relationship Psychology Research - Gottman Institute findings on relationship satisfaction
- UX Design Patterns - Dating app interaction research and best practices
- Local Business APIs - Google Places, Yelp, OpenTable integration specifications

---

**Last Updated**: 2025-01-21  
**Status**: Draft  
**Owner**: AI Agent (Claude)  
**Review Cycle**: Weekly validation against implementation progress