# Product Requirements Document (PRD)

## 🎯 Capability Definition

### Core Capability
**What permanent capability does this scenario add to Vrooli?**
Autonomous business development engine that converts job board opportunities into revenue-generating scenarios. It harvests market demand, evaluates feasibility against existing capabilities, generates new scenarios for viable opportunities, and produces ready-to-submit proposals - creating a continuous loop of capability expansion driven by real market needs.

### Intelligence Amplification
**How does this capability make future agents smarter?**
- **Market Intelligence**: Agents learn what capabilities are commercially valuable by analyzing real job demand patterns
- **Feasibility Assessment**: Agents improve at evaluating technical complexity and resource requirements
- **Proposal Generation**: Agents learn effective proposal patterns that win contracts
- **Capability Mapping**: System builds a semantic understanding of which scenarios solve which business problems
- **Revenue Optimization**: Agents learn to prioritize high-value, achievable opportunities

### Recursive Value
**What new scenarios become possible after this exists?**
1. **market-trend-analyzer**: Analyzes job patterns to predict emerging technology needs
2. **pricing-optimizer**: Uses win/loss data to optimize proposal pricing
3. **skill-gap-identifier**: Identifies missing capabilities preventing us from bidding on lucrative contracts
4. **customer-success-tracker**: Monitors delivered projects and generates case studies
5. **competitive-intelligence**: Tracks competitor capabilities based on jobs they're likely winning

## 📊 Success Metrics

### Functional Requirements
- **Must Have (P0)**
  - [ ] Huginn agent scrapes Upwork jobs every 30 minutes
  - [ ] Screenshot upload with OCR text extraction for manual job entry
  - [ ] Research phase using knowledge-observatory and research-assistant via CLI
  - [ ] Job evaluation with clear outcomes (NO_ACTION, NOT_RECOMMENDED, ALREADY_DONE, RECOMMENDED)
  - [ ] File-based task management with state folders (pending/researching/evaluated/approved/building/completed)
  - [ ] Manual approval gate before scenario generation
  - [ ] Integration with ecosystem-manager for approved jobs
  - [ ] Proposal document generation with delivery instructions
  - [ ] Trello-like UI showing jobs in columns by state
  
- **Should Have (P1)**  
  - [ ] Batch processing of multiple jobs simultaneously
  - [ ] Historical tracking of win/loss rates per job type
  - [ ] Template library for common proposal sections
  - [ ] Cost estimation based on resource requirements
  - [ ] Automatic scenario validation after generation
  
- **Nice to Have (P2)**
  - [ ] Integration with other freelance platforms (Fiverr, Freelancer.com)
  - [ ] AI-powered pricing recommendations
  - [ ] Client communication templates
  - [ ] Portfolio generation from completed scenarios

### Performance Criteria
| Metric | Target | Measurement Method |
|--------|--------|-------------------|
| Job Processing Time | < 5 minutes per job | API monitoring |
| Research Accuracy | > 85% correct feasibility assessment | Manual validation |
| Proposal Generation | < 2 minutes | Time from approval to document |
| UI Response Time | < 200ms for state transitions | Frontend monitoring |

### Quality Gates
- [ ] All P0 requirements implemented and tested
- [ ] Can process sample Upwork job end-to-end
- [ ] Research reports contain actionable recommendations
- [ ] Generated proposals include all required sections
- [ ] UI displays accurate job states in real-time

## 🏗️ Technical Architecture

### Resource Dependencies
```yaml
required:
  - resource_name: postgres
    purpose: Store jobs, research reports, proposals, and state history
    integration_pattern: Direct SQL via Go API
    access_method: Database connection pool
    
  - resource_name: ollama
    purpose: LLM for research analysis and proposal generation
    integration_pattern: Direct API calls for speed
    access_method: HTTP API at service.ollama.url
    
  - resource_name: huginn
    purpose: Automated Upwork job scraping
    integration_pattern: Huginn agent with webhook to our API
    access_method: Huginn scenario configuration
    
  - resource_name: browserless
    purpose: Screenshot OCR for manual job entry
    integration_pattern: Direct API for screenshot processing
    access_method: resource-browserless CLI
    
optional:
  - resource_name: qdrant
    purpose: Semantic search for similar past proposals
    fallback: Skip similarity matching
    access_method: HTTP API
    
  - resource_name: redis
    purpose: Cache for research results
    fallback: No caching, direct processing
    access_method: Redis client library
```

### Resource Integration Standards
```yaml
# NO N8N WORKFLOWS - Direct integration only for speed
integration_priorities:
  1_cli_commands:         # FIRST: Use resource CLI for reliability
    - command: resource-ollama query
      purpose: Fast LLM inference
    - command: ecosystem-manager add scenario <name>
      purpose: Generate new scenarios
    - command: resource-browserless screenshot
      purpose: Process job screenshots
  
  2_direct_api:          # SECOND: Direct API for performance-critical paths
    - ollama_api: HTTP POST for batch inference
    - postgres: Connection pool for data operations
    - qdrant: Vector similarity search

  3_scenario_cli:        # THIRD: Leverage other scenario CLIs
    - knowledge-observatory search
    - research-assistant generate-report
```

### Data Models
```yaml
primary_entities:
  - name: Job
    storage: postgres
    schema: |
      {
        id: UUID
        source: string (upwork|screenshot|manual)
        title: string
        description: text
        budget: decimal
        skills_required: jsonb
        state: enum(pending|researching|evaluated|approved|building|completed)
        research_report_id: UUID
        proposal_id: UUID
        created_at: timestamp
        updated_at: timestamp
      }
    
  - name: ResearchReport
    storage: postgres
    schema: |
      {
        id: UUID
        job_id: UUID
        evaluation: enum(NO_ACTION|NOT_RECOMMENDED|ALREADY_DONE|RECOMMENDED)
        feasibility_score: float
        existing_scenarios: jsonb
        required_scenarios: jsonb
        estimated_hours: integer
        technical_analysis: text
        created_at: timestamp
      }
    
  - name: Proposal
    storage: postgres
    schema: |
      {
        id: UUID
        job_id: UUID
        cover_letter: text
        technical_approach: text
        timeline: jsonb
        deliverables: jsonb
        price: decimal
        generated_scenarios: jsonb
        created_at: timestamp
      }
```

### API Contract
```yaml
endpoints:
  - method: POST
    path: /api/v1/jobs/import
    purpose: Import job from screenshot or manual entry
    input_schema: |
      {
        source: "screenshot|manual"
        data: string (base64 image or text)
      }
    output_schema: |
      {
        job_id: UUID
        state: "pending"
      }
    
  - method: POST
    path: /api/v1/jobs/{id}/research
    purpose: Trigger research phase for a job
    output_schema: |
      {
        report_id: UUID
        evaluation: string
        feasibility_score: float
      }
    
  - method: POST
    path: /api/v1/jobs/{id}/approve
    purpose: Approve job for scenario generation
    output_schema: |
      {
        state: "building"
        estimated_completion: timestamp
      }
    
  - method: GET
    path: /api/v1/jobs
    purpose: List jobs with filtering by state
    output_schema: |
      {
        jobs: [Job]
        total: integer
      }
```

### Event Interface
```yaml
published_events:
  - name: job.imported
    payload: {job_id, source, title}
    
  - name: job.researched
    payload: {job_id, evaluation, feasibility_score}
    
  - name: job.approved
    payload: {job_id, scenarios_to_build}
    
  - name: proposal.generated
    payload: {job_id, proposal_id, price}
    
consumed_events:
  - name: huginn.job.scraped
    action: Import and queue for research
```

## 🖥️ CLI Interface Contract

### Command Structure
```yaml
cli_binary: job-to-scenario
install_script: cli/install.sh

required_commands:
  - name: status
    description: Show pipeline status and job counts by state
    flags: [--json, --verbose]
    
  - name: help
    description: Display command help
    flags: [--all, --command <name>]
    
  - name: version
    description: Show version information
    flags: [--json]

custom_commands:
  - name: import
    description: Import job from source
    arguments:
      - name: source
        type: string
        required: true
        description: "screenshot|manual|url"
    flags:
      - name: --file
        description: Path to screenshot or text file
      - name: --text
        description: Job description text
    
  - name: research
    description: Run research on pending jobs
    flags:
      - name: --job-id
        description: Research specific job
      - name: --batch-size
        description: Number of jobs to process
    
  - name: approve
    description: Approve job for scenario generation
    arguments:
      - name: job-id
        type: string
        required: true
    
  - name: list
    description: List jobs by state
    flags:
      - name: --state
        description: Filter by state
      - name: --limit
        description: Number of results
      - name: --json
        description: JSON output
```

## 🔄 Integration Requirements

### Upstream Dependencies
- **knowledge-observatory**: Provides domain knowledge for feasibility assessment
- **research-assistant**: Generates detailed technical analysis reports
- **ecosystem-manager**: Creates new scenarios from requirements
- **Huginn**: Scrapes job boards automatically

### Downstream Enablement
- **market-trend-analyzer**: Can analyze our job processing history
- **pricing-optimizer**: Uses proposal win/loss data
- **customer-success-tracker**: Tracks delivered scenarios

### Cross-Scenario Interactions
```yaml
provides_to:
  - scenario: pricing-optimizer
    capability: Historical proposal and win/loss data
    interface: API endpoint /api/v1/proposals/history
    
  - scenario: market-trend-analyzer  
    capability: Raw job market demand data
    interface: API endpoint /api/v1/jobs/analytics
    
consumes_from:
  - scenario: knowledge-observatory
    capability: Technical feasibility analysis
    fallback: Basic keyword matching
    
  - scenario: ecosystem-manager
    capability: Scenario creation from requirements
    fallback: Manual scenario creation
```

## 🎨 Style and Branding Requirements

### UI/UX Style Guidelines
```yaml
style_profile:
  category: professional
  inspiration: "Trello meets Upwork - professional project management"
  
  visual_style:
    color_scheme: light with blue/green accents
    typography: modern, clean, highly readable
    layout: kanban board with expandable cards
    animations: subtle state transitions
  
  personality:
    tone: professional yet approachable
    mood: focused and efficient
    target_feeling: "Control and clarity over opportunities"

ui_components:
  - Kanban board with 6 columns (one per state)
  - Job cards showing title, budget, evaluation score
  - Expandable cards for research reports
  - Action buttons for state transitions
  - Upload zone for screenshots
  - Metrics dashboard showing success rates
```

### Target Audience Alignment
- **Primary Users**: Business development teams, freelance agencies
- **User Expectations**: Professional tool that streamlines opportunity evaluation
- **Accessibility**: WCAG 2.1 AA compliance
- **Responsive Design**: Desktop-first, tablet-friendly

## 💰 Value Proposition

### Business Value
- **Primary Value**: Converts job boards into automated revenue pipeline
- **Revenue Potential**: $10K-$50K per successfully delivered scenario
- **Cost Savings**: 10-20 hours/week of manual job evaluation
- **Market Differentiator**: Only platform that auto-generates solutions for opportunities

### Technical Value
- **Reusability Score**: 9/10 - Core components usable by many business scenarios
- **Complexity Reduction**: Makes business development systematic and scalable
- **Innovation Enablement**: Creates feedback loop between market demand and capability

## 🧬 Evolution Path

### Version 1.0 (Current)
- Upwork integration via Huginn
- Manual approval workflow
- Basic research and evaluation
- Proposal generation

### Version 2.0 (Planned)
- Multi-platform support (Fiverr, Freelancer)
- Automated pricing optimization
- Success prediction ML model
- Client communication automation

### Long-term Vision
- Full autonomous business development
- Predictive capability planning based on market trends
- Automated contract negotiation
- Self-improving proposal strategies

## 🚨 Risk Mitigation

### Technical Risks
| Risk | Probability | Impact | Mitigation |
|------|------------|--------|------------|
| Huginn scraping blocked | Medium | High | Fallback to manual/screenshot import |
| LLM hallucination in research | Medium | High | Human approval gate |
| Scenario generation failure | Low | Medium | Manual intervention path |

### Operational Risks
- **Over-commitment**: Approval gate prevents accepting too many jobs
- **Quality Control**: Research reports reviewed before scenario generation
- **Resource Conflicts**: Job processing queue with rate limiting

## ✅ Validation Criteria

### Declarative Test Specification
```yaml
version: 1.0
scenario: job-to-scenario-pipeline

structure:
  required_files:
    - .vrooli/service.json
    - PRD.md
    - api/main.go
    - api/go.mod
    - cli/job-to-scenario-pipeline
    - cli/install.sh
    - initialization/storage/postgres/schema.sql
    - initialization/automation/huginn/upwork-scraper.json
    - scenario-test.yaml
    
  required_dirs:
    - api
    - cli
    - data/pending
    - data/researching  
    - data/evaluated
    - data/approved
    - data/building
    - data/completed
    - ui

resources:
  required: [postgres, ollama, huginn, browserless]
  optional: [qdrant, redis]
  health_timeout: 60

tests:
  - name: "Import job from text"
    type: http
    service: api
    endpoint: /api/v1/jobs/import
    method: POST
    body:
      source: manual
      data: "Build a React dashboard for analytics"
    expect:
      status: 201
      body:
        state: pending
        
  - name: "Research job feasibility"
    type: http
    service: api
    endpoint: /api/v1/jobs/{job_id}/research
    method: POST
    expect:
      status: 200
      body:
        evaluation: "RECOMMENDED|NOT_RECOMMENDED|ALREADY_DONE|NO_ACTION"
        
  - name: "CLI lists pending jobs"
    type: exec
    command: ./cli/job-to-scenario-pipeline list --state pending --json
    expect:
      exit_code: 0
      output_contains: ["jobs", "total"]
```

## 📝 Implementation Notes

### Design Decisions
**Direct Integration over N8N**: Chose direct API/CLI integration for 10x speed improvement
- Alternative considered: N8N workflows for orchestration
- Decision driver: Performance requirements for real-time processing
- Trade-offs: More code complexity for significant speed gains

**File-based State Management**: Using filesystem for job state transitions
- Alternative considered: Pure database state
- Decision driver: Visibility and debugging ease
- Trade-offs: Filesystem dependency for distributed scalability

### Known Limitations
- **Single Upwork Account**: Currently limited to one scraping account
  - Workaround: Manual import for additional sources
  - Future fix: Multi-account support in v2
  
- **English Only**: Research and proposals in English only
  - Workaround: Manual translation
  - Future fix: Multi-language support planned

### Security Considerations
- **API Keys**: Stored in environment variables, never in code
- **Job Data**: Sanitized before storage to prevent injection
- **Approval Gate**: Human verification before expensive operations

## 🔗 References

### Documentation
- README.md - User guide for pipeline operation
- docs/api.md - Complete API specification
- docs/cli.md - CLI command reference
- docs/architecture.md - Technical architecture details

### Related PRDs
- scenarios/ecosystem-manager/README.md
- scenarios/knowledge-observatory/PRD.md
- scenarios/research-assistant/PRD.md

---

**Last Updated**: 2025-01-06
**Status**: Draft
**Owner**: AI Agent
**Review Cycle**: Weekly validation against job processing metrics
