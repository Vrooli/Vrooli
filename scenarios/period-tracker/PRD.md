# Product Requirements Document (PRD) - Period Tracker

## 🎯 Capability Definition

### Core Capability
**What permanent capability does this scenario add to Vrooli?**
Adds privacy-first menstrual health tracking with local AI-powered insights, multi-tenant support for households, and cross-scenario integration capabilities. This provides a completely private, locally-hosted alternative to corporate period tracking apps that may share sensitive health data.

### Intelligence Amplification
**How does this capability make future agents smarter?**
- Enables health pattern recognition across Vrooli scenarios
- Provides temporal correlation insights for mood, energy, and productivity patterns
- Creates baseline health data that can inform scheduling, wellness, and lifestyle optimization
- Establishes privacy-first health data handling patterns for future medical scenarios

### Recursive Value
**What new scenarios become possible after this exists?**
- **Health Dashboard Hub**: Aggregate period data with other health metrics (sleep, exercise, mood)
- **Wellness Optimization Agent**: AI that suggests lifestyle changes based on cycle patterns
- **Partner/Family Health Coordinator**: Discrete notifications and support systems
- **Medical Report Generator**: Automated summaries for healthcare provider visits
- **Research Data Contributor**: Anonymized, opt-in women's health research platform

## 📊 Success Metrics

### Functional Requirements
- **Must Have (P0)**
  - [x] Secure multi-tenant period cycle tracking with encrypted local storage
  - [x] Basic symptom logging (flow, pain, mood, physical symptoms)
  - [x] Cycle prediction algorithm with pattern recognition
  - [x] Calendar integration for period predictions and blocking
  - [x] Privacy-first architecture with no external data sharing
  - [x] Simple, intuitive UI optimized for daily use
  
- **Should Have (P1)**
  - [ ] AI-powered pattern recognition using local Ollama models
  - [ ] Symptom correlation insights ("Your headaches correlate with Day 2 of cycle")
  - [ ] Medication and supplement reminder system
  - [ ] Export functionality for medical appointments
  - [ ] Partner dashboard with consensual, discrete sharing
  - [ ] Historical trend analysis and reporting
  
- **Nice to Have (P2)**
  - [ ] Photo tracking for skin/symptom changes
  - [ ] Integration with fitness/mood tracking scenarios
  - [ ] Custom reminder scheduling
  - [ ] Advanced fertility window calculations
  - [ ] Research contribution opt-in with anonymization

### Performance Criteria
| Metric | Target | Measurement Method |
|--------|--------|-------------------|
| Response Time | < 200ms for 95% of requests | API monitoring |
| Throughput | 100 operations/second per user | Load testing |
| Data Security | 100% local storage, AES-256 encryption | Security audit |
| Resource Usage | < 512MB memory, < 10% CPU | System monitoring |

### Quality Gates
- [ ] All P0 requirements implemented and tested
- [ ] Integration tests pass with PostgreSQL and Redis
- [ ] Multi-tenant authentication working correctly
- [ ] Performance targets met under concurrent user load
- [ ] Documentation complete (README, API docs, CLI help)
- [ ] Scenario can be invoked by other agents via API/CLI
- [ ] Calendar scenario integration functional

## 🏗️ Technical Architecture

### Resource Dependencies
```yaml
required:
  - resource_name: postgres
    purpose: Encrypted storage of period data, symptoms, predictions
    integration_pattern: Direct SQL via Go database/sql
    access_method: Connection string via environment variables
    
  - resource_name: scenario-authenticator
    purpose: Multi-tenant user authentication and authorization
    integration_pattern: API calls for user session management
    access_method: HTTP API calls to auth endpoints

optional:
  - resource_name: redis
    purpose: Session management and temporary data caching
    fallback: In-memory sessions with reduced multi-instance support
    access_method: Redis client connections
    
  - resource_name: ollama
    purpose: Local AI pattern recognition and insight generation
    fallback: Basic statistical analysis without natural language insights
    access_method: HTTP API to local Ollama instance
```

### Resource Integration Standards
```yaml
# Priority order for resource access (NO n8n workflows as requested):
integration_priorities:
  1_resource_cli:        # FIRST: Use resource CLI commands
    - command: resource-postgres query
      purpose: Database operations
    - command: resource-redis set/get
      purpose: Cache operations
    - command: resource-ollama generate
      purpose: AI insights
  
  2_direct_api:          # SECOND: Direct API calls
    - justification: Real-time performance needed for health tracking
      endpoint: PostgreSQL connection string
    - justification: Session management requires direct Redis access
      endpoint: Redis connection string

# No shared workflows - direct resource access for performance
shared_workflow_criteria:
  - NOT APPLICABLE - Moving away from n8n for performance
```

### Data Models
```yaml
# Core data structures that define the capability
primary_entities:
  - name: User
    storage: postgres
    schema: |
      {
        id: UUID
        username: string
        email: string (encrypted)
        created_at: timestamp
        updated_at: timestamp
        preferences: jsonb
        timezone: string
      }
    relationships: One-to-many with cycles, symptoms
    
  - name: Cycle
    storage: postgres
    schema: |
      {
        id: UUID
        user_id: UUID (foreign key)
        start_date: date
        end_date: date (nullable)
        length: integer (computed)
        flow_intensity: enum
        notes: text (encrypted)
        predicted: boolean
        created_at: timestamp
      }
    relationships: One-to-many with daily_symptoms
    
  - name: DailySymptom
    storage: postgres
    schema: |
      {
        id: UUID
        user_id: UUID
        cycle_id: UUID (nullable)
        date: date
        symptoms: jsonb
        mood_rating: integer (1-10)
        pain_level: integer (0-10)
        flow_level: enum
        notes: text (encrypted)
        created_at: timestamp
      }
    relationships: Belongs to user and cycle
    
  - name: Prediction
    storage: postgres
    schema: |
      {
        id: UUID
        user_id: UUID
        predicted_start: date
        confidence: float
        algorithm_version: string
        created_at: timestamp
      }
    relationships: Belongs to user
```

### API Contract
```yaml
# Defines how other scenarios/agents can use this capability
endpoints:
  - method: POST
    path: /api/v1/cycles
    purpose: Log new cycle start for calendar integration
    input_schema: |
      {
        user_id: UUID,
        start_date: "YYYY-MM-DD",
        flow_intensity: "light|medium|heavy"
      }
    output_schema: |
      {
        cycle_id: UUID,
        predictions: {
          next_period: "YYYY-MM-DD",
          fertile_window: ["YYYY-MM-DD", "YYYY-MM-DD"]
        }
      }
    sla:
      response_time: 100ms
      availability: 99.9%
      
  - method: GET
    path: /api/v1/predictions/{user_id}
    purpose: Get cycle predictions for calendar blocking
    input_schema: |
      {
        user_id: UUID,
        start_date: "YYYY-MM-DD" (optional),
        end_date: "YYYY-MM-DD" (optional)
      }
    output_schema: |
      {
        predictions: [
          {
            predicted_start: "YYYY-MM-DD",
            predicted_end: "YYYY-MM-DD",
            confidence: 0.85
          }
        ]
      }
    sla:
      response_time: 50ms
      availability: 99.9%

  - method: POST
    path: /api/v1/symptoms
    purpose: Log daily symptoms for pattern recognition
    input_schema: |
      {
        user_id: UUID,
        date: "YYYY-MM-DD",
        symptoms: ["cramps", "headache", "fatigue"],
        mood_rating: 7,
        pain_level: 3
      }
    output_schema: |
      {
        symptom_id: UUID,
        patterns_detected: ["headaches_day_2", "mood_drops_pms"]
      }
    sla:
      response_time: 150ms
      availability: 99.9%
```

### Event Interface
```yaml
# Events this capability publishes for others to consume
published_events:
  - name: period.cycle.started
    payload: {user_id: UUID, start_date: date, predicted_length: integer}
    subscribers: [calendar, wellness-tracker, partner-dashboard]
    
  - name: period.prediction.updated
    payload: {user_id: UUID, next_period: date, confidence: float}
    subscribers: [calendar, notification-hub]
    
  - name: period.pattern.detected
    payload: {user_id: UUID, pattern_type: string, correlation: float}
    subscribers: [wellness-optimizer, health-dashboard]
    
consumed_events:
  - name: calendar.event.created
    action: Check for conflicts with predicted periods
  - name: mood.rating.logged
    action: Correlate mood patterns with cycle phases
```

## 🖥️ CLI Interface Contract

### Command Structure
```yaml
# Primary CLI executable name and pattern
cli_binary: period-tracker
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

# Scenario-specific commands:
custom_commands:
  - name: log-cycle
    description: Log the start of a new period cycle
    api_endpoint: /api/v1/cycles
    arguments:
      - name: start_date
        type: string
        required: false (defaults to today)
        description: Start date in YYYY-MM-DD format
    flags:
      - name: --flow
        description: Flow intensity (light|medium|heavy)
      - name: --notes
        description: Additional notes for this cycle
    output: "Cycle logged. Next period predicted for YYYY-MM-DD"
    
  - name: log-symptoms
    description: Log daily symptoms and mood
    api_endpoint: /api/v1/symptoms
    arguments:
      - name: date
        type: string
        required: false (defaults to today)
        description: Date in YYYY-MM-DD format
    flags:
      - name: --symptoms
        description: Comma-separated symptom list
      - name: --mood
        description: Mood rating 1-10
      - name: --pain
        description: Pain level 0-10
    output: "Symptoms logged for YYYY-MM-DD"
    
  - name: predictions
    description: Show upcoming cycle predictions
    api_endpoint: /api/v1/predictions
    arguments:
      - name: user_id
        type: string
        required: true
        description: User ID to get predictions for
    flags:
      - name: --days
        description: Number of days to predict ahead (default 90)
      - name: --json
        description: Output in JSON format
    output: "Next period: YYYY-MM-DD (85% confidence)"
    
  - name: patterns
    description: Show detected health patterns
    api_endpoint: /api/v1/patterns
    arguments:
      - name: user_id
        type: string
        required: true
        description: User ID to analyze patterns for
    flags:
      - name: --timeframe
        description: Analysis timeframe (3m|6m|1y)
    output: "Pattern detected: Headaches correlate with Day 2 (r=0.73)"
```

## 🔄 Integration Requirements

### Upstream Dependencies
**What capabilities must exist before this can function?**
- **PostgreSQL Resource**: Essential for encrypted health data storage with ACID compliance
- **Scenario Authenticator**: Required for multi-tenant user isolation and privacy
- **Redis Resource (Optional)**: Enhances session management and reduces database load

### Downstream Enablement
**What future capabilities does this unlock?**
- **Health Pattern Recognition**: Enables AI-driven wellness optimization across scenarios
- **Medical Data Management**: Establishes patterns for HIPAA-compliant health data handling
- **Privacy-First Architecture**: Template for sensitive personal data scenarios

### Cross-Scenario Interactions
```yaml
# How this scenario enhances other scenarios
provides_to:
  - scenario: calendar
    capability: Period predictions for automatic calendar blocking
    interface: API
    
  - scenario: wellness-optimizer
    capability: Hormonal cycle data for lifestyle recommendations
    interface: API/Events
    
  - scenario: partner-dashboard
    capability: Discrete period tracking information sharing
    interface: API
    
consumes_from:
  - scenario: calendar
    capability: Schedule optimization around predicted periods
    fallback: Manual calendar entry by user
    
  - scenario: mood-tracker
    capability: Correlation with cycle phases
    fallback: Internal mood logging only
```

## 🎨 Style and Branding Requirements

### UI/UX Style Guidelines
```yaml
# Define the visual and experiential personality of this scenario
style_profile:
  category: personal-health-focused
  inspiration: "Combination of Clue app's data visualization with lo-fi, warm aesthetics"
  
  # Visual characteristics:
  visual_style:
    color_scheme: warm pastels with privacy-focused dark mode option
    typography: modern, readable, medical-friendly font stack
    layout: clean, spacious design optimized for daily quick entry
    animations: subtle, calming transitions that don't distract
  
  # Personality traits:
  personality:
    tone: supportive, informative, non-judgmental
    mood: calm, private, empowering
    target_feeling: "Users should feel safe, informed, and in control of their data"

# Privacy-first design requirements:
privacy_design:
  - No external tracking or analytics
  - Clear data ownership messaging
  - Granular privacy controls visible in UI
  - Secure by default, convenience as opt-in
  - Multi-tenant awareness in all UI flows
```

### Target Audience Alignment
- **Primary Users**: Privacy-conscious women aged 16-50 who want local control of health data
- **User Expectations**: Medical-grade privacy with consumer app usability
- **Accessibility**: WCAG 2.1 AA compliance, menstrual health stigma awareness
- **Responsive Design**: Mobile-first (phone in private settings), desktop secondary

### Brand Consistency Rules
- **Scenario Identity**: Professional health tool with warm, supportive personality
- **Vrooli Integration**: Feels integrated but respects the sensitive nature of the data
- **Professional Design**: Health data requires trustworthy, clean interface design

## 💰 Value Proposition

### Business Value
- **Primary Value**: Privacy-compliant menstrual health tracking in an increasingly regulated data environment
- **Revenue Potential**: $15K - $35K per household deployment (premium health privacy feature)
- **Cost Savings**: Eliminates privacy compliance risks vs. cloud-hosted alternatives
- **Market Differentiator**: Only fully local, AI-powered period tracking with multi-tenant support

### Technical Value
- **Reusability Score**: High - establishes patterns for health data, privacy, and multi-tenancy
- **Complexity Reduction**: Makes sensitive health tracking simple and private
- **Innovation Enablement**: Enables complete local health ecosystem development

## 🧬 Evolution Path

### Version 1.0 (Current)
- Core cycle tracking and symptom logging
- Basic prediction algorithm
- Multi-tenant authentication
- Calendar integration

### Version 2.0 (Planned)
- AI-powered pattern recognition with local Ollama
- Advanced symptom correlation analysis
- Partner/family sharing features
- Export functionality for medical visits

### Long-term Vision
- Complete local health ecosystem hub
- Research data contribution with full anonymization
- Integration with wearable devices for automatic tracking
- AI health coaching based on cycle patterns

## 🔄 Scenario Lifecycle Integration

### Direct Scenario Deployment
```yaml
# Requirements for direct scenario execution:
direct_execution:
  supported: true
  structure_compliance:
    - service.json with PostgreSQL and auth requirements
    - Complete database initialization scripts
    - Health check endpoints for all services
    - Multi-tenant deployment configuration
    
  deployment_targets:
    - local: Docker Compose with encrypted volumes
    - kubernetes: Helm chart with secret management
    - cloud: Private cloud with data residency compliance
    
  revenue_model:
    - type: subscription (privacy-premium tier)
    - pricing_tiers: Individual ($5/mo), Household ($12/mo), Family ($20/mo)
    - trial_period: 30 days full featured trial
```

### Capability Discovery
```yaml
# How other scenarios/agents discover and use this capability:
discovery:
  registry_entry:
    name: period-tracker
    category: health-management
    capabilities: [cycle-tracking, health-patterns, calendar-integration]
    interfaces:
      - api: http://localhost:${API_PORT}/api/v1
      - cli: period-tracker
      - events: period.*
      
  metadata:
    description: "Privacy-first menstrual health tracking with AI insights"
    keywords: [health, menstrual, privacy, tracking, AI, multi-tenant]
    dependencies: [postgres, scenario-authenticator]
    enhances: [calendar, wellness-optimizer, health-dashboard]
```

## 🚨 Risk Mitigation

### Technical Risks
| Risk | Probability | Impact | Mitigation |
|------|------------|--------|------------|
| Data breach/privacy violation | Low | Critical | AES-256 encryption, local-only storage, security audit |
| Authentication bypass | Medium | High | Integration with scenario-authenticator, session validation |
| Prediction algorithm accuracy | Medium | Medium | Machine learning validation, user feedback loops |
| Database corruption | Low | High | Automated backups, transaction boundaries |

### Operational Risks
- **Privacy Compliance**: All data stays local, no external API calls for core functionality
- **Multi-tenant Isolation**: Strict user data separation at database and API levels
- **Health Data Sensitivity**: Clear consent flows, granular data controls
- **Cross-scenario Integration**: Secure API boundaries prevent data leakage

## ✅ Validation Criteria

### Declarative Test Specification
Period Tracker now relies on Vrooli's phased testing architecture (`test/run-tests.sh`). Each phase provides targeted validation:
- **Structure** – confirms critical directories/files exist (.vrooli/service.json, API/UI entry points, run-tests wiring).
- **Dependencies** – dry-runs Go module resolution and npm installs so local builds surface issues early.
- **Unit** – executes centralized Go unit coverage through `testing::unit::run_all_tests` with the same thresholds enforced platform-wide.
- **Integration** – auto-starts the scenario (if needed) and hits the live `/health`, `/api/v1/health/encryption`, and `/api/v1/auth/status` endpoints to guarantee runtime wiring.
- **Business** – guards the presence of core REST routes and seeded privacy safeguards to catch accidental deletions during refactors.
- **Performance** – currently records a skipped placeholder until telemetry hooks come online, keeping the phase visible in dashboards.

```bash
cd scenarios/period-tracker
./test/run-tests.sh            # run all phases
./test/run-tests.sh quick      # structure + dependencies + unit
./test/run-tests.sh integration
```

### Performance Validation
- [ ] API response times < 200ms for 95% of requests
- [ ] Resource usage < 512MB memory, < 10% CPU
- [ ] Concurrent user support (10+ users simultaneously)
- [ ] No memory leaks over 24-hour continuous operation

### Integration Validation
- [ ] Calendar scenario integration functional
- [ ] Multi-tenant authentication working correctly
- [ ] All API endpoints documented and functional
- [ ] All CLI commands executable with --help
- [ ] Health check endpoints responding correctly

### Privacy & Security Validation
- [ ] All sensitive data encrypted at rest
- [ ] User data isolation verified between tenants
- [ ] No external data transmission except opt-in features
- [ ] Session management secure and isolated
- [ ] Data access logging for audit trails

---

**Last Updated**: 2024-01-15
**Status**: Draft
**Owner**: Claude Code AI Agent
**Review Cycle**: Weekly during development, monthly post-launch