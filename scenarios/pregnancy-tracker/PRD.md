# Product Requirements Document (PRD) - Pregnancy Tracker

## 🎯 Capability Definition

### Core Capability
**What permanent capability does this scenario add to Vrooli?**
Provides comprehensive, privacy-first pregnancy tracking with medical-grade data management, evidence-based information retrieval, and multi-outcome support. This creates a permanent reproductive health monitoring capability that respects user sovereignty while enabling AI-assisted insights without external data transmission.

### Intelligence Amplification
**How does this capability make future agents smarter?**
- Establishes patterns for sensitive medical data handling that other health scenarios can adopt
- Creates reusable timeline visualization and milestone tracking components
- Develops risk assessment models that combine user data with medical knowledge bases
- Builds partner/family data sharing protocols that maintain privacy boundaries
- Provides templates for handling uncertain/variable outcomes in long-term tracking

### Recursive Value
**What new scenarios become possible after this exists?**
1. **postpartum-recovery-tracker** - Continues tracking after birth for recovery monitoring
2. **baby-development-tracker** - Seamless transition to infant health tracking
3. **fertility-optimizer** - Combines period and pregnancy data for conception planning
4. **pregnancy-nutrition-planner** - AI-powered meal planning based on trimester needs
5. **medical-report-generator** - Automated health summaries for healthcare providers

## 📊 Success Metrics

### Functional Requirements
- **Must Have (P0)**
  - [ ] Gestational age tracking with due date calculation from LMP or conception date
  - [ ] Multi-tenant support via scenario-authenticator integration
  - [ ] Daily symptom logging (physical, emotional, measurements)
  - [ ] Appointment tracker with customizable reminders
  - [ ] Data export in JSON and PDF formats for medical visits
  - [ ] Week-by-week fetal development information with scientific citations
  - [ ] Support for multiple pregnancy outcomes (not just live birth)
  - [ ] Encrypted local storage with zero external transmission
  - [ ] Emergency info card with quick-access medical details
  - [ ] Search functionality across all content and logs
  
- **Should Have (P1)**
  - [ ] Integration with period-tracker for conception data import
  - [ ] Integration with calendar scenario for appointment blocking
  - [ ] Kick counter with pattern analysis
  - [ ] Contraction timer with export capability
  - [ ] Weight gain tracker with healthy range indicators
  - [ ] Medication/supplement tracking with reminders
  - [ ] Photo journal for bump progression
  - [ ] Partner portal with limited read-only access
  - [ ] Local Ollama integration for Q&A (with medical disclaimers)
  - [ ] Customizable measurement units (metric/imperial)
  
- **Nice to Have (P2)**
  - [ ] 3D fetal development visualizations
  - [ ] Birth plan builder with printable output
  - [ ] Postpartum mode transition
  - [ ] Budget tracker integration for baby expenses
  - [ ] Multiple pregnancy support (twins/multiples)
  - [ ] Voice notes via Whisper integration
  - [ ] Wearable device data import (heart rate, sleep)
  - [ ] Community features (anonymous forums)

### Performance Criteria
| Metric | Target | Measurement Method |
|--------|--------|-------------------|
| Response Time | < 200ms for 95% of requests | API monitoring |
| Search Speed | < 500ms for content search | UI testing |
| Data Encryption | < 20ms overhead per operation | Benchmark testing |
| Concurrent Users | 10+ simultaneous users | Load testing |
| Resource Usage | < 768MB memory, < 15% CPU | System monitoring |

### Quality Gates
- [ ] All P0 requirements implemented and tested
- [ ] Multi-tenant isolation verified with test data
- [ ] Search functionality returns accurate results
- [ ] Export formats validated with sample medical providers
- [ ] Privacy audit confirms no external data transmission
- [ ] UI responsive on mobile devices
- [ ] Integration tests pass with period-tracker and calendar scenarios

## 🏗️ Technical Architecture

### Resource Dependencies
```yaml
required:
  - resource_name: postgres
    purpose: Encrypted storage of all pregnancy data with multi-tenant isolation
    integration_pattern: Direct SQL with application-layer encryption
    access_method: CLI command via resource-postgres
    
  - resource_name: scenario-authenticator
    purpose: Multi-tenant authentication and user isolation
    integration_pattern: API integration for auth tokens
    access_method: Direct API calls for session management
    
optional:
  - resource_name: ollama
    purpose: Local AI for answering pregnancy questions
    fallback: Static FAQ content if unavailable
    access_method: CLI command via resource-ollama
    
  - resource_name: redis
    purpose: Session caching and temporary data
    fallback: In-memory cache if unavailable
    access_method: CLI command via resource-redis
    
  - resource_name: minio
    purpose: Photo storage for bump progression journal
    fallback: Base64 in postgres if unavailable
    access_method: CLI command via resource-minio
```

### Resource Integration Standards
```yaml
integration_priorities:
  1_shared_workflows:     # None for now (moving away from n8n per requirements)
    - note: Direct resource access preferred for performance
  
  2_resource_cli:        # Primary integration method
    - command: resource-postgres query
      purpose: All database operations
    - command: resource-ollama generate
      purpose: AI-powered Q&A responses
    - command: resource-redis get/set
      purpose: Session management
  
  3_direct_api:          # Only for scenario-authenticator
    - justification: Auth requires direct API for token validation
      endpoint: /api/v1/auth/validate

shared_workflow_criteria:
  - No n8n workflows (performance requirement)
  - Direct CLI/API access for all resources
  - Batch operations where possible for efficiency
```

### Data Models
```yaml
primary_entities:
  - name: Pregnancy
    storage: postgres (encrypted)
    schema: |
      {
        id: UUID
        user_id: STRING (from scenario-authenticator)
        lmp_date: DATE
        conception_date: DATE (optional)
        due_date: DATE (calculated)
        current_week: INTEGER
        current_day: INTEGER
        pregnancy_type: ENUM (singleton, twins, multiples)
        outcome: ENUM (ongoing, live_birth, loss, terminated, unknown)
        created_at: TIMESTAMP
        updated_at: TIMESTAMP
      }

  - name: DailyLog
    storage: postgres (encrypted)
    schema: |
      {
        id: UUID
        pregnancy_id: UUID
        date: DATE
        weight: DECIMAL
        blood_pressure: STRING
        symptoms: JSON[]
        mood: INTEGER (1-10)
        energy: INTEGER (1-10)
        notes: TEXT (encrypted)
        photos: STRING[] (minio URLs or base64)
        created_at: TIMESTAMP
      }

  - name: Appointment
    storage: postgres
    schema: |
      {
        id: UUID
        pregnancy_id: UUID
        date: DATETIME
        type: ENUM (ob, ultrasound, lab, specialist, other)
        provider: STRING
        location: STRING
        notes: TEXT
        reminder_sent: BOOLEAN
        results: JSON
      }

  - name: KickCount
    storage: postgres
    schema: |
      {
        id: UUID
        pregnancy_id: UUID
        timestamp: TIMESTAMP
        count: INTEGER
        duration_minutes: INTEGER
        notes: TEXT
      }

  - name: EmergencyInfo
    storage: postgres (encrypted)
    schema: |
      {
        id: UUID
        user_id: STRING
        blood_type: STRING
        allergies: STRING[]
        medications: STRING[]
        conditions: STRING[]
        ob_contact: JSON
        emergency_contacts: JSON[]
        hospital_preference: STRING
      }
```

### API Endpoints
```yaml
health:
  - GET /health
  - GET /api/v1/status

pregnancy:
  - POST /api/v1/pregnancy/start
  - GET /api/v1/pregnancy/current
  - PUT /api/v1/pregnancy/{id}
  - GET /api/v1/pregnancy/week-info/{week}

tracking:
  - POST /api/v1/logs/daily
  - GET /api/v1/logs/range
  - POST /api/v1/kicks/count
  - GET /api/v1/kicks/patterns
  - POST /api/v1/contractions/timer
  - GET /api/v1/contractions/history

appointments:
  - POST /api/v1/appointments
  - GET /api/v1/appointments/upcoming
  - PUT /api/v1/appointments/{id}

search:
  - GET /api/v1/search?q={query}
  - GET /api/v1/content/week/{week}

export:
  - GET /api/v1/export/json
  - GET /api/v1/export/pdf
  - GET /api/v1/export/emergency-card

partner:
  - POST /api/v1/partner/invite
  - GET /api/v1/partner/view (limited data)
```

## 🔐 Security & Privacy

### Data Protection
- AES-256 encryption at rest for all sensitive fields
- No external API calls except for opted-in features
- Audit logging for all data access
- Automatic data purge options after pregnancy completion
- GDPR-compliant data export and deletion

### Multi-tenancy
- Complete user isolation via scenario-authenticator
- No shared data between users except anonymous aggregate stats
- Partner access requires explicit invite with revocable permissions
- Separate encryption keys per user

## 🎨 UI/UX Design

### Design Principles
- **Empathetic**: Acknowledges various pregnancy outcomes
- **Accessible**: WCAG 2.1 AA compliant, large touch targets
- **Informative**: Evidence-based content with citations
- **Flexible**: Adapts to different pregnancy journeys
- **Calm**: Soft colors, no alarming notifications

### Key Screens
1. **Dashboard**: Current week, days until due, today's tip
2. **Daily Log**: Quick symptom entry with smart suggestions
3. **Timeline**: Visual pregnancy journey with milestones
4. **Search**: Full-text search across all content and logs
5. **Emergency Card**: One-tap access to critical medical info
6. **Partner View**: Limited dashboard for invited partners

### Theme
- Light: Soft pastels (lavender, mint, peach)
- Dark: Muted purples and blues with warm accents
- High Contrast: For accessibility needs

## 📚 Content Requirements

### Medical Content
- Week-by-week development guides with scientific citations
- Symptom encyclopedia with "when to call doctor" guidance
- Nutrition guidelines by trimester
- Exercise recommendations with modifications
- Mental health resources

### Search Index
- All medical content must be searchable
- User logs searchable by symptom/keyword
- Appointment notes searchable
- Auto-complete suggestions based on common queries

## 🚀 Implementation Phases

### Phase 1: Core Tracking (Week 1)
- Basic pregnancy setup and week calculation
- Daily symptom logging
- Simple UI with dashboard
- PostgreSQL schema and encryption

### Phase 2: Medical Features (Week 2)
- Week-by-week content with citations
- Appointment tracker
- Emergency info card
- Search functionality

### Phase 3: Integrations (Week 3)
- Period-tracker data import
- Calendar scenario integration
- Partner portal
- Export capabilities

### Phase 4: Advanced Features (Week 4)
- Kick counter and contraction timer
- Photo journal
- Local AI Q&A with Ollama
- Performance optimization

## 📈 Success Criteria

### User Validation
- Successfully tracks pregnancy from conception to outcome
- Exports medical reports accepted by healthcare providers
- Search returns relevant results in < 500ms
- Partner can view limited data without full access
- Supports pregnancy loss without requiring app deletion

### Technical Validation
- Passes all multi-tenant isolation tests
- No data leakage between users
- All P0 features functional
- Mobile responsive design verified
- Integration tests pass with dependent scenarios

## 📝 Notes

### Differentiation from Period Tracker
While period-tracker focuses on cycle patterns, pregnancy-tracker provides:
- Gestational age calculation and milestone tracking
- Medical appointment management
- Fetal development information with citations
- Support for various pregnancy outcomes
- Partner sharing capabilities
- Emergency medical information access

### Research Insights Incorporated
Based on user research, we're addressing key frustrations:
- **Search functionality** (only 24% of apps have this)
- **Evidence-based content** with citations (only 28% provide)
- **Multiple outcome support** (most assume live birth)
- **Safety information** tools (least common but most wanted)
- **Complete pregnancy journey** coverage (60% of apps lack this)

### Future Considerations
- Transition to postpartum-tracker scenario
- Data migration to baby-tracker scenario
- Integration with medical provider APIs (with consent)
- Wearable device support for automated data entry
- Anonymous research data contribution (opt-in)