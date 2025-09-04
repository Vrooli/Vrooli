# Product Requirements Document (PRD)

## 🎯 Capability Definition

### Core Capability
**What permanent capability does this scenario add to Vrooli?**
Elo-swipe adds a universal ranking engine that captures human preferences through simple binary comparisons. It transforms any unordered list into a precisely ranked priority order using chess-style Elo ratings, making subjective prioritization objective and reusable across all Vrooli scenarios.

### Intelligence Amplification
**How does this capability make future agents smarter?**
Every swipe creates training data about human preferences, teaching agents what matters most. Over time, agents can predict priorities without asking, understand trade-offs between competing goals, and make nuanced decisions aligned with learned preferences. The ranking data becomes a preference model that improves all decision-making.

### Recursive Value
**What new scenarios become possible after this exists?**
1. **auto-prioritizer** - AI that pre-ranks new items based on learned preferences
2. **decision-assistant** - Complex multi-criteria decisions broken into simple comparisons
3. **team-consensus** - Aggregate rankings from multiple users for group decisions
4. **preference-predictor** - ML model that learns to rank like you
5. **priority-drift-detector** - Alerts when preferences change over time

## 📊 Success Metrics

### Functional Requirements
- **Must Have (P0)**
  - [ ] Tinder-like swipe UI for binary comparisons
  - [ ] Elo rating calculation and storage
  - [ ] API for submitting lists and retrieving rankings
  - [ ] PostgreSQL persistence for items, ratings, and comparison history
  - [ ] CLI for managing lists and viewing rankings
  - [ ] Multi-list support with isolated ranking pools
  
- **Should Have (P1)**
  - [ ] Smart pairing algorithm to minimize comparisons needed
  - [ ] Confidence scores based on comparison count
  - [ ] Export rankings to JSON/CSV
  - [ ] Undo/skip functionality during swiping
  - [ ] Progress tracking (X of Y potential comparisons)
  
- **Nice to Have (P2)**
  - [ ] AI-assisted pre-filtering using ollama.json workflow
  - [ ] Ranking visualization (graph/chart view)
  - [ ] Import lists from external sources
  - [ ] Collaborative ranking (multiple users)
  - [ ] Historical preference analytics

### Performance Criteria
| Metric | Target | Measurement Method |
|--------|--------|-------------------|
| Response Time | < 100ms for swipe action | API monitoring |
| Throughput | 1000 comparisons/minute | Load testing |
| Convergence | Stable ranking in ~n*log(n) comparisons | Algorithm analysis |
| Resource Usage | < 512MB memory, < 10% CPU | System monitoring |

### Quality Gates
- [x] All P0 requirements implemented and tested
- [ ] Integration tests pass with PostgreSQL
- [ ] Performance targets met under load
- [ ] Documentation complete (README, API docs, CLI help)
- [ ] Scenario can be invoked by other agents via API/CLI

## 🏗️ Technical Architecture

### Resource Dependencies
```yaml
required:
  - resource_name: postgres
    purpose: Persistent storage for lists, items, ratings, and comparison history
    integration_pattern: Direct SQL via Go database/sql
    access_method: PostgreSQL connection via environment variables
    
optional:
  - resource_name: redis
    purpose: Caching for frequently accessed rankings
    fallback: Direct database queries (slower but functional)
    access_method: Redis client if REDIS_URL provided
    
  - resource_name: ollama
    purpose: AI-enhanced comparison suggestions and pre-filtering
    fallback: Pure algorithmic pairing
    access_method: Shared workflow ollama.json
```

### Resource Integration Standards
```yaml
integration_priorities:
  1_shared_workflows:
    - workflow: ollama.json
      location: initialization/automation/n8n/
      purpose: AI-powered comparison suggestions and similarity analysis
  
  2_resource_cli:
    - command: resource-postgres exec
      purpose: Database migrations and queries
  
  3_direct_api:
    - justification: High-performance database operations require direct connection
      endpoint: PostgreSQL connection pool
```

### Data Models
```yaml
primary_entities:
  - name: List
    storage: postgres
    schema: |
      {
        id: UUID
        name: string
        description: string
        created_at: timestamp
        updated_at: timestamp
        owner_id: string (optional)
        metadata: JSONB
      }
    relationships: Has many Items
    
  - name: Item
    storage: postgres
    schema: |
      {
        id: UUID
        list_id: UUID
        content: JSONB
        elo_rating: float (default 1500)
        comparison_count: integer
        wins: integer
        losses: integer
        created_at: timestamp
      }
    relationships: Belongs to List, has many Comparisons
    
  - name: Comparison
    storage: postgres
    schema: |
      {
        id: UUID
        list_id: UUID
        winner_id: UUID
        loser_id: UUID
        timestamp: timestamp
        user_id: string (optional)
        confidence: float
      }
    relationships: References two Items
```

### API Contract
```yaml
endpoints:
  - method: POST
    path: /api/v1/lists
    purpose: Create new list for ranking
    input_schema: |
      {
        name: string
        description: string
        items: Array<{content: any}>
      }
    output_schema: |
      {
        list_id: UUID
        item_count: integer
        comparison_url: string
      }
    sla:
      response_time: 200ms
      availability: 99.9%
      
  - method: GET
    path: /api/v1/lists/:id/next-comparison
    purpose: Get next optimal pair to compare
    output_schema: |
      {
        item_a: {id: UUID, content: any}
        item_b: {id: UUID, content: any}
        progress: {completed: integer, total: integer}
      }
      
  - method: POST
    path: /api/v1/comparisons
    purpose: Submit comparison result
    input_schema: |
      {
        list_id: UUID
        winner_id: UUID
        loser_id: UUID
      }
      
  - method: GET
    path: /api/v1/lists/:id/rankings
    purpose: Get current rankings
    output_schema: |
      {
        rankings: Array<{
          rank: integer
          item: any
          elo_rating: float
          confidence: float
        }>
      }
```

### Event Interface
```yaml
published_events:
  - name: ranking.comparison.completed
    payload: {list_id, winner_id, loser_id, new_ratings}
    subscribers: Analytics, preference learning systems
    
  - name: ranking.list.converged
    payload: {list_id, final_rankings, comparison_count}
    subscribers: Dependent workflows, notification systems
    
consumed_events:
  - name: workflow.list.created
    action: Automatically create ranking list for workflow outputs
```

## 🖥️ CLI Interface Contract

### Command Structure
```yaml
cli_binary: elo-swipe
install_script: cli/install.sh

required_commands:
  - name: status
    description: Show operational status and database health
    flags: [--json, --verbose]
    
  - name: help
    description: Display command help and usage
    flags: [--all, --command <name>]
    
  - name: version
    description: Show CLI and API version information
    flags: [--json]

custom_commands:
  - name: create-list
    description: Create new list for ranking
    api_endpoint: /api/v1/lists
    arguments:
      - name: name
        type: string
        required: true
        description: List name
      - name: items-file
        type: string
        required: true
        description: JSON file containing items
    flags:
      - name: --description
        description: List description
    output: List ID and comparison URL
    
  - name: swipe
    description: Interactive CLI swiping interface
    arguments:
      - name: list-id
        type: string
        required: true
    flags:
      - name: --auto-save
        description: Save after each comparison
        
  - name: rankings
    description: Display current rankings
    api_endpoint: /api/v1/lists/:id/rankings
    arguments:
      - name: list-id
        type: string
        required: true
    flags:
      - name: --format
        description: Output format (table|json|csv)
      - name: --top
        description: Show only top N items
```

### CLI-API Parity Requirements
- **Coverage**: Every API endpoint has corresponding CLI command
- **Naming**: CLI uses intuitive verb-noun patterns
- **Arguments**: Direct mapping to API parameters
- **Output**: Human-readable by default, JSON with --json
- **Authentication**: Uses API_KEY from environment or config

## 🔄 Integration Requirements

### Upstream Dependencies
- **PostgreSQL**: Core storage for all ranking data
- **Redis (optional)**: Performance caching layer
- **ollama.json workflow**: AI-enhanced pairing suggestions

### Downstream Enablement
- **Issue Prioritization**: Development teams rank bugs/features
- **Backlog Management**: Product managers order user stories
- **Decision Support**: Any scenario needing preference-based ordering
- **ML Training Data**: Preference learning for recommendation systems

### Cross-Scenario Interactions
```yaml
provides_to:
  - scenario: product-manager
    capability: Prioritize feature requests and user feedback
    interface: API/CLI
    
  - scenario: agent-dashboard
    capability: Rank agent tasks by importance
    interface: API
    
  - scenario: roi-fit-analysis
    capability: Rank investment opportunities
    interface: Events
    
consumes_from:
  - scenario: stream-of-consciousness-analyzer
    capability: Items to rank from unstructured input
    fallback: Manual item entry
```

## 🎨 Style and Branding Requirements

### UI/UX Style Guidelines
```yaml
style_profile:
  category: playful
  inspiration: Tinder meets Trello - fun prioritization
  
  visual_style:
    color_scheme: custom gradient cards
    typography: modern, clean, readable
    layout: single-page swipe interface
    animations: smooth card transitions, swipe physics
  
  personality:
    tone: friendly
    mood: engaging, gamified
    target_feeling: Making hard decisions feel easy and fun

style_references:
  playful:
    - reference: Tinder card swiping mechanics
    - reference: Duolingo's gamification approach
  professional:
    - reference: Linear's keyboard-first interface
```

### Target Audience Alignment
- **Primary Users**: Product managers, developers, team leads
- **User Expectations**: Fast, intuitive, even enjoyable prioritization
- **Accessibility**: Keyboard navigation (arrow keys), WCAG AA
- **Responsive Design**: Mobile-first for actual swiping

### Brand Consistency Rules
- Must feel lightweight and approachable, not overwhelming
- Gamification elements make tedious ranking engaging
- Professional enough for business use, fun enough to want to use

## 💰 Value Proposition

### Business Value
- **Primary Value**: Transform subjective opinions into objective rankings
- **Revenue Potential**: $10K - $30K per enterprise deployment
- **Cost Savings**: 80% reduction in meeting time for prioritization
- **Market Differentiator**: Only ranking system that learns preferences

### Technical Value
- **Reusability Score**: 10/10 - Every scenario with lists can use this
- **Complexity Reduction**: N-way priority decisions → simple binary choices
- **Innovation Enablement**: Preference learning, consensus building

## 🧬 Evolution Path

### Version 1.0 (Current)
- Core Elo ranking system
- Swipe UI and CLI
- PostgreSQL persistence
- Multi-list support

### Version 2.0 (Planned)
- AI-powered pre-filtering
- Team consensus rankings
- Preference learning model
- Real-time collaborative ranking

### Long-term Vision
- Universal preference engine for all Vrooli decisions
- Cross-scenario preference transfer learning
- Predictive ranking without human input

## 🔄 Scenario Lifecycle Integration

### Direct Scenario Deployment
```yaml
direct_execution:
  supported: true
  structure_compliance:
    - service.json with complete metadata
    - PostgreSQL schema initialization
    - UI server configuration
    - Health check endpoints
    
  deployment_targets:
    - local: Docker Compose with PostgreSQL
    - kubernetes: StatefulSet for data persistence
    - cloud: RDS/CloudSQL for database
    
  revenue_model:
    - type: subscription
    - pricing_tiers: 
      - free: 3 lists, 100 items/list
      - pro: Unlimited lists/items
      - enterprise: Team features, API access
    - trial_period: 14 days
```

### Capability Discovery
```yaml
discovery:
  registry_entry:
    name: elo-swipe
    category: analysis
    capabilities: [ranking, prioritization, preference-capture]
    interfaces:
      - api: http://localhost:$API_PORT
      - cli: elo-swipe
      - events: ranking.*
      
  metadata:
    description: Universal ranking engine using Elo ratings
    keywords: [ranking, prioritization, elo, preferences, decision]
    dependencies: [postgres]
    enhances: [product-manager, agent-dashboard, roi-fit-analysis]
```

## ✅ Validation Criteria

### Declarative Test Specification
```yaml
version: 1.0
scenario: elo-swipe

structure:
  required_files:
    - .vrooli/service.json
    - PRD.md
    - api/main.go
    - api/go.mod
    - cli/elo-swipe
    - cli/install.sh
    - ui/index.html
    - ui/server.js
    - initialization/storage/postgres/schema.sql
    - scenario-test.yaml
    
tests:
  - name: "Create and rank a list"
    type: integration
    steps:
      - create list with 5 items
      - perform 10 comparisons
      - verify rankings exist
      - check Elo ratings updated
```

## 📝 Implementation Notes

### Design Decisions
**Elo vs other algorithms**: Chose Elo for proven stability in ranking systems
- Alternative considered: Bradley-Terry, TrueSkill
- Decision driver: Simplicity and proven track record
- Trade-offs: Less sophisticated than TrueSkill but much simpler

### Known Limitations
- **Initial randomness**: First few comparisons may seem arbitrary
  - Workaround: Smart initial pairing based on item similarity
- **Comparison fatigue**: Large lists need many comparisons
  - Future fix: AI pre-filtering to reduce comparison count

### Security Considerations
- **Data Protection**: Lists can be private or shared
- **Access Control**: API key or session-based
- **Audit Trail**: All comparisons logged with timestamps

---

**Last Updated**: 2024-01-04
**Status**: Draft
**Owner**: AI Agent
**Review Cycle**: After each major feature addition