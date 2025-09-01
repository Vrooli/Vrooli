# Product Requirements Document (PRD)

## 🎯 Capability Definition

### Core Capability
**What permanent capability does this scenario add to Vrooli?**
Adds intelligent personal relationship management that proactively nurtures human connections through automated birthday reminders, personalized gift suggestions, relationship insights, and contact enrichment. This creates a foundational social graph capability that other scenarios can leverage for personalization and relationship-aware interactions.

### Intelligence Amplification
**How does this capability make future agents smarter?**
This capability provides a reusable social context layer that enables agents to understand user relationships, preferences, and social patterns. Future agents can query relationship data to personalize interactions, understand social dynamics, and make contextually appropriate suggestions based on the user's social network.

### Recursive Value
**What new scenarios become possible after this exists?**
- **event-planner**: Leverages relationship data to automatically invite the right people to events
- **network-optimizer**: Analyzes relationship patterns to suggest professional networking opportunities
- **gift-exchange-coordinator**: Organizes group gift exchanges and Secret Santa events
- **social-calendar-sync**: Integrates relationship milestones with calendar and scheduling systems
- **relationship-health-monitor**: Tracks interaction frequency and suggests when to reconnect

## 📊 Success Metrics

### Functional Requirements
- **Must Have (P0)**
  - [x] Store and manage contact information with custom fields
  - [x] Track important dates (birthdays, anniversaries) with recurring reminders
  - [x] Generate personalized gift suggestions using Ollama
  - [x] Provide API and CLI access for all core functions
  - [x] Initialize with proper database schema and n8n workflows
  
- **Should Have (P1)**
  - [x] Enrich contact data with interests and preferences
  - [x] Generate relationship insights and patterns
  - [ ] Support multiple reminder channels (email, webhook)
  - [ ] Track gift history to avoid repetition
  
- **Nice to Have (P2)**
  - [ ] Social media integration for automatic updates
  - [ ] Photo memories association with contacts
  - [ ] Relationship timeline visualization

### Performance Criteria
| Metric | Target | Measurement Method |
|--------|--------|-------------------|
| Response Time | < 500ms for 95% of requests | API monitoring |
| Throughput | 100 operations/second | Load testing |
| Gift Suggestion Accuracy | > 80% relevance score | User feedback |
| Resource Usage | < 512MB memory, < 10% CPU | System monitoring |

### Quality Gates
- [x] All P0 requirements implemented and tested
- [x] Integration with postgres, n8n, ollama resources
- [ ] Performance targets met under load
- [x] Basic documentation (README, API structure)
- [x] CLI interface available for other scenarios

## 🏗️ Technical Architecture

### Resource Dependencies
```yaml
required:
  - resource_name: postgres
    purpose: Primary data storage for contacts, events, and relationships
    integration_pattern: CLI commands for queries and updates
    access_method: vrooli resource postgres query
    
  - resource_name: n8n
    purpose: Workflow automation for reminders and enrichment
    integration_pattern: Workflow injection and webhook triggers
    access_method: Shared workflows and direct workflow execution
    
  - resource_name: ollama
    purpose: AI-powered gift suggestions and relationship insights
    integration_pattern: Shared workflow for reliable execution
    access_method: initialization/n8n/ollama.json
    
optional:
  - resource_name: redis
    purpose: Caching for frequently accessed relationship data
    fallback: Direct database queries with slightly higher latency
    access_method: vrooli resource redis get/set
```

### Resource Integration Standards
```yaml
integration_priorities:
  1_shared_workflows:
    - workflow: ollama.json
      location: initialization/n8n/
      purpose: Reliable LLM inference for gift suggestions
  
  2_resource_cli:
    - command: resource-postgres query
      purpose: Database operations
    - command: resource-redis get/set
      purpose: Caching layer
  
  3_direct_api:
    - justification: N8n webhook responses only
      endpoint: /webhook/[workflow-id]
```

### Data Models
```yaml
primary_entities:
  - name: Contact
    storage: postgres
    schema: |
      {
        id: UUID
        name: string
        nickname: string
        email: string
        phone: string
        interests: json[]
        notes: text
        created_at: timestamp
        updated_at: timestamp
      }
    
  - name: Event
    storage: postgres
    schema: |
      {
        id: UUID
        contact_id: UUID
        event_type: string
        event_date: date
        recurring: boolean
        reminder_days_before: integer
      }
```

### API Contract
```yaml
endpoints:
  - method: POST
    path: /api/v1/contacts
    purpose: Create new contact with relationship data
    
  - method: GET
    path: /api/v1/birthdays/upcoming
    purpose: Get upcoming birthdays for reminder processing
    
  - method: POST
    path: /api/v1/gifts/suggest
    purpose: Generate personalized gift suggestions
    
  - method: POST
    path: /api/v1/insights/generate
    purpose: Create relationship insights from interaction data
```

## 🖥️ CLI Interface Contract

### Command Structure
```yaml
cli_binary: relationship-manager
install_script: cli/install.sh

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

custom_commands:
  - name: add-contact
    description: Add a new contact to the relationship manager
    api_endpoint: /api/v1/contacts
    arguments:
      - name: name
        type: string
        required: true
        description: Contact's full name
    flags:
      - name: --interests
        description: Comma-separated list of interests
        
  - name: suggest-gift
    description: Get gift suggestions for a contact
    api_endpoint: /api/v1/gifts/suggest
    arguments:
      - name: contact-id
        type: string
        required: true
        description: Contact ID or name
    flags:
      - name: --budget
        description: Budget range for gifts
      - name: --occasion
        description: Gift occasion (birthday, holiday, etc)
```

## 🔄 Integration Requirements

### Upstream Dependencies
- **ollama**: Required for AI-powered suggestions and insights
- **postgres**: Database must be initialized before scenario can function
- **n8n**: Workflow engine must be running for automation features

### Downstream Enablement
- **Social Graph API**: Provides queryable relationship data for other scenarios
- **Gift History Tracking**: Enables smarter suggestions over time
- **Relationship Patterns**: Analyzable data for social insights

### Cross-Scenario Interactions
```yaml
provides_to:
  - scenario: event-planner
    capability: Guest list suggestions based on relationships
    interface: API/CLI
    
  - scenario: morning-vision-walk
    capability: Social reminders and relationship priorities
    interface: API
    
consumes_from:
  - scenario: stream-of-consciousness-analyzer
    capability: Extract relationship mentions from notes
    fallback: Manual contact entry only
```

## 🎨 Style and Branding Requirements

### UI/UX Style Guidelines
```yaml
style_profile:
  category: friendly
  inspiration: Modern personal CRM with warm, approachable design
  
  visual_style:
    color_scheme: light with pastel accents
    typography: modern, clean, readable
    layout: card-based dashboard
    animations: subtle transitions
  
  personality:
    tone: friendly and thoughtful
    mood: caring and organized
    metaphor: "Your thoughtful friend who never forgets"
```

## 📈 Growth Path

### Phase 1 (Current)
- Basic contact management
- Birthday reminders
- Gift suggestions

### Phase 2
- Relationship health scoring
- Interaction frequency tracking
- Multi-channel reminders

### Phase 3
- Social media integration
- Group relationship dynamics
- Network effect analysis

## 🔒 Privacy & Security

### Data Handling
- All relationship data stored locally
- No external API calls except to local Ollama
- Encryption at rest for sensitive notes
- User-controlled data retention policies

### Access Control
- API authentication required
- CLI inherits user permissions
- Workflow access controlled by n8n

## ✅ Validation Criteria

### Conversion Success
- `vrooli scenario run personal-relationship-manager` completes without errors
- All n8n workflows properly injected
- Database schema successfully created

### Runtime Success
- `vrooli scenario run personal-relationship-manager` launches all components
- API responds to health checks at configured port
- UI accessible and displays contact list
- CLI commands execute successfully

### Integration Success
- Gift suggestions return valid JSON with relevance scores
- Birthday reminders trigger on schedule
- Contact enrichment processes successfully