# Product Requirements Document (PRD)

## 🎯 Capability Definition

### Core Capability
**What permanent capability does this scenario add to Vrooli?**
Browser Automation Studio provides a visual workflow builder and execution environment for browser automation tasks. It enables agents and users to create, debug, and schedule browser-based workflows with real-time visual feedback, making UI testing, web scraping, and browser task automation accessible and self-healing through AI integration.

### Intelligence Amplification
**How does this capability make future agents smarter?**
This capability transforms browser automation from code-based scripts to visual, AI-assisted workflows. Agents can now validate UIs, collect web data, and automate browser tasks without writing Puppeteer/Playwright code. The AI debugging loop means workflows self-heal when websites change, and successful patterns get shared across all scenarios.

### Recursive Value
**What new scenarios become possible after this exists?**
- **Automated QA Suite**: Test all Vrooli scenarios' UIs automatically with visual regression testing
- **Competitive Intelligence Platform**: Monitor competitor websites and extract structured data
- **Customer Onboarding Automator**: Guide users through complex web forms with screenshot validation
- **Web Content Aggregator**: Collect and synthesize information from multiple sources
- **Accessibility Auditor**: Automatically test scenarios for WCAG compliance with visual proof

## 📊 Success Metrics

### Functional Requirements
- **Must Have (P0)**
  - [ ] Visual workflow builder using React Flow with drag-and-drop nodes
  - [ ] Real-time screenshot display during workflow execution
  - [ ] Integration with resource-browserless CLI for browser control
  - [ ] Save/load workflows in organized folder structure
  - [ ] Execute workflows via API and CLI
  - [ ] AI workflow generation from natural language descriptions
  
- **Should Have (P1)**
  - [ ] Live log streaming alongside screenshots
  - [ ] Claude Code agent integration for debugging failed workflows
  - [ ] Calendar scheduling for recurring workflows
  - [ ] Export workflows as n8n compatible JSON
  - [ ] Workflow templates library
  - [ ] WebSocket-based real-time updates
  
- **Nice to Have (P2)**
  - [ ] Workflow versioning and rollback
  - [ ] Visual diff comparison between executions
  - [ ] Performance metrics dashboard
  - [ ] Collaborative workflow editing
  - [ ] Mobile-responsive workflow viewer

### Performance Criteria
| Metric | Target | Measurement Method |
|--------|--------|-------------------|
| Response Time | < 200ms for UI interactions | Frontend monitoring |
| Workflow Start Time | < 2s from trigger to first action | API monitoring |
| Screenshot Latency | < 500ms from capture to display | WebSocket timing |
| Concurrent Workflows | 10+ simultaneous executions | Load testing |
| Resource Usage | < 2GB memory for UI, < 500MB per workflow | System monitoring |

### Quality Gates
- [ ] All P0 requirements implemented and tested
- [ ] Integration tests pass with browserless resource
- [ ] Performance targets met under load
- [ ] Documentation complete (README, API docs, CLI help)
- [ ] Scenario can be invoked by other agents via API/CLI

## 🏗️ Technical Architecture

### Resource Dependencies
```yaml
required:
  - resource_name: browserless
    purpose: Core browser automation engine
    integration_pattern: CLI commands via resource-browserless
    access_method: resource-browserless [command]
    
  - resource_name: postgres
    purpose: Workflow definitions and execution history storage
    integration_pattern: Direct database connection
    access_method: Database client library
    
  - resource_name: minio
    purpose: Screenshot and artifact storage
    integration_pattern: S3-compatible API
    access_method: S3 client library
    
optional:
  - resource_name: redis
    purpose: WebSocket pub/sub and caching
    fallback: In-memory fallback for single-instance
    access_method: Redis client library
    
  - resource_name: ollama
    purpose: AI workflow generation and debugging
    fallback: Disable AI features if unavailable
    access_method: Shared workflow - initialization/n8n/ollama.json
```

### Resource Integration Standards
```yaml
# Priority order for resource access (MUST follow this hierarchy):
integration_priorities:
  1_shared_workflows:     # FIRST: Use existing shared n8n workflows
    - workflow: ollama.json
      location: initialization/n8n/
      purpose: AI-powered workflow generation and debugging
  
  2_resource_cli:        # SECOND: Use resource CLI commands
    - command: resource-browserless screenshot [url]
      purpose: Capture webpage screenshots
    - command: resource-browserless for n8n execute-workflow [id]
      purpose: Execute n8n workflows via browser
  
  3_direct_api:          # LAST: Direct API only when necessary
    - justification: Real-time WebSocket streaming requires direct connection
      endpoint: WebSocket connection for live updates

# Shared workflow guidelines:
shared_workflow_criteria:
  - Browser automation patterns will be packaged as reusable n8n workflows
  - Place in initialization/automation/n8n/ for scenario-specific workflows
  - Common patterns (login, form fill, data extraction) become shared workflows
  - Document all reusable patterns in workflow descriptions
```

### Data Models
```yaml
# Core data structures that define the capability
primary_entities:
  - name: Workflow
    storage: postgres
    schema: |
      {
        id: UUID
        name: string
        folder_path: string
        flow_definition: JSONB (React Flow nodes/edges)
        created_by: string
        created_at: timestamp
        updated_at: timestamp
        version: integer
        tags: string[]
      }
    relationships: Has many Executions, belongs to Folder
    
  - name: Execution
    storage: postgres
    schema: |
      {
        id: UUID
        workflow_id: UUID
        status: enum (pending|running|completed|failed)
        started_at: timestamp
        completed_at: timestamp
        screenshots: string[] (minio URLs)
        logs: JSONB
        error: string
        trigger_type: enum (manual|scheduled|api|event)
      }
    relationships: Belongs to Workflow, has many Screenshots
    
  - name: WorkflowFolder
    storage: postgres
    schema: |
      {
        id: UUID
        path: string (e.g., "/ui-validation/checkout")
        parent_id: UUID (nullable)
        name: string
        description: string
        created_at: timestamp
      }
    relationships: Has many Workflows, has many child Folders
```

### API Contract
```yaml
# Defines how other scenarios/agents can use this capability
endpoints:
  - method: POST
    path: /api/v1/workflows/create
    purpose: Create a new workflow from definition or AI prompt
    input_schema: |
      {
        name: string
        folder_path: string
        flow_definition?: ReactFlowJSON
        ai_prompt?: string
      }
    output_schema: |
      {
        workflow_id: UUID
        status: "created"
      }
    sla:
      response_time: 500ms
      availability: 99.9%
      
  - method: POST
    path: /api/v1/workflows/{id}/execute
    purpose: Execute a workflow and return execution ID
    input_schema: |
      {
        parameters?: Record<string, any>
        wait_for_completion?: boolean
      }
    output_schema: |
      {
        execution_id: UUID
        status: "started"|"completed"|"failed"
        result?: any
      }
    sla:
      response_time: 2000ms
      availability: 99.9%
      
  - method: GET
    path: /api/v1/executions/{id}/screenshots
    purpose: Stream screenshots for an execution
    output_schema: |
      {
        screenshots: Array<{
          timestamp: ISO8601
          url: string
          step_name: string
        }>
      }
      
  - method: POST
    path: /api/v1/workflows/{id}/debug
    purpose: Spawn Claude Code agent to debug failed workflow
    input_schema: |
      {
        execution_id: UUID
        error_context?: string
      }
    output_schema: |
      {
        agent_session_id: UUID
        status: "debugging"
      }
```

### Event Interface
```yaml
# Events this capability publishes for others to consume
published_events:
  - name: workflow.execution.started
    payload: { workflow_id, execution_id, trigger_type }
    subscribers: Monitoring dashboards, dependent workflows
    
  - name: workflow.execution.completed
    payload: { workflow_id, execution_id, result, screenshots }
    subscribers: Downstream automations, notification systems
    
  - name: workflow.execution.failed
    payload: { workflow_id, execution_id, error, last_screenshot }
    subscribers: Debugging agents, alert systems
    
consumed_events:
  - name: calendar.schedule.triggered
    action: Execute scheduled workflow
    
  - name: ui.deployment.completed
    action: Trigger UI validation workflow for newly deployed scenario
```

## 🖥️ CLI Interface Contract

### Command Structure
```yaml
# Primary CLI executable name and pattern
cli_binary: browser-automation-studio
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
  - name: workflow create
    description: Create a workflow from file or AI prompt
    api_endpoint: /api/v1/workflows/create
    arguments:
      - name: name
        type: string
        required: true
        description: Workflow name
    flags:
      - name: --folder
        description: Folder path for organization
      - name: --from-file
        description: Import from JSON file
      - name: --ai-prompt
        description: Generate from natural language
    output: Workflow ID and creation status
    
  - name: workflow execute
    description: Execute a workflow by ID or name
    api_endpoint: /api/v1/workflows/{id}/execute
    arguments:
      - name: workflow
        type: string
        required: true
        description: Workflow ID or name
    flags:
      - name: --params
        description: JSON parameters for workflow
      - name: --wait
        description: Wait for completion
      - name: --output-screenshots
        description: Save screenshots to directory
    output: Execution ID and status
    
  - name: workflow list
    description: List all workflows in tree structure
    api_endpoint: /api/v1/workflows
    flags:
      - name: --folder
        description: Filter by folder path
      - name: --json
        description: Output as JSON
    output: Tree view or JSON of workflows
    
  - name: execution watch
    description: Watch live execution with screenshots
    api_endpoint: WebSocket /ws/executions/{id}
    arguments:
      - name: execution-id
        type: string
        required: true
        description: Execution to watch
    output: Real-time screenshots and logs
```

## 🔄 Integration Requirements

### Upstream Dependencies
**What capabilities must exist before this can function?**
- **resource-browserless**: Provides core browser automation engine
- **resource-postgres**: Stores workflow definitions and history
- **resource-minio**: Stores screenshots and artifacts
- **ollama.json workflow**: AI capabilities for workflow generation

### Downstream Enablement
**What future capabilities does this unlock?**
- **Universal UI Testing**: Any scenario can validate its UI automatically
- **Web Data Pipeline**: Structured data extraction becomes trivial
- **Visual Regression Testing**: Compare UI changes over time
- **Automated Documentation**: Generate visual guides from workflows

### Cross-Scenario Interactions
```yaml
provides_to:
  - scenario: study-buddy
    capability: UI validation for quiz interactions
    interface: API - POST /api/v1/workflows/execute
    
  - scenario: product-manager
    capability: Competitor website monitoring
    interface: CLI - browser-automation-studio workflow execute
    
  - scenario: roi-fit-analysis
    capability: Financial data scraping from web sources
    interface: Scheduled workflow execution
    
consumes_from:
  - scenario: calendar-scheduler
    capability: Recurring workflow triggers
    fallback: Manual execution only
    
  - scenario: claude-code-agent
    capability: AI debugging for failed workflows
    fallback: Manual debugging required
```

## 🎨 Style and Branding Requirements

### UI/UX Style Guidelines
```yaml
style_profile:
  category: technical
  inspiration: "GitHub Actions workflow editor meets Chrome DevTools"
  
  visual_style:
    color_scheme: dark
    typography: modern
    layout: dashboard
    animations: subtle
    
    specifics:
      - Dark theme with syntax highlighting for nodes
      - Split-pane layout: workflow builder left, execution viewer right
      - React Flow nodes styled like code blocks
      - Screenshot viewer with filmstrip timeline
      - Console-style log output with ANSI color support
  
  personality:
    tone: technical
    mood: focused
    target_feeling: "Professional developer tool that makes complex tasks visual"

style_references:
  technical:
    - "n8n workflow editor - node-based visual programming"
    - "Playwright Inspector - browser automation debugging"
    - "GitHub Actions - workflow visualization"
    - "Postman - API testing interface"
```

### Target Audience Alignment
- **Primary Users**: Developers, QA engineers, automation specialists
- **User Expectations**: Professional tool with powerful capabilities
- **Accessibility**: WCAG AA compliance, keyboard navigation support
- **Responsive Design**: Desktop-first, tablet support for viewing

## 💰 Value Proposition

### Business Value
- **Primary Value**: Eliminates need for dedicated browser automation developers
- **Revenue Potential**: $15K - $30K per deployment (replaces Selenium Grid setups)
- **Cost Savings**: 80% reduction in UI testing time
- **Market Differentiator**: Visual workflows with AI self-healing

### Technical Value
- **Reusability Score**: 10/10 - Every UI scenario needs validation
- **Complexity Reduction**: Visual workflows replace 100s of lines of code
- **Innovation Enablement**: Makes browser automation accessible to non-developers

## 🧬 Evolution Path

### Version 1.0 (Current)
- Visual workflow builder with React Flow
- Real-time screenshot viewing
- Basic AI workflow generation
- Folder organization
- API/CLI access

### Version 2.0 (Planned)
- Workflow marketplace for sharing patterns
- Visual regression testing with diff highlights
- Parallel execution across multiple browsers
- Cloud execution options
- Advanced AI debugging with auto-fix

### Long-term Vision
- Becomes the universal UI validation layer for all Vrooli scenarios
- Workflow patterns evolve into reusable "skills" that agents compose
- Self-optimizing workflows that improve performance over time

## 🔄 Scenario Lifecycle Integration

### Direct Scenario Deployment
```yaml
direct_execution:
  supported: true
  structure_compliance:
    - service.json with browserless, postgres, minio resources
    - React-based UI with Vite build
    - Go API with workflow engine
    - CLI wrapper for all API functions
    
  deployment_targets:
    - local: Docker Compose with resource dependencies
    - kubernetes: StatefulSet for workflow engine
    - cloud: AWS ECS with Aurora Postgres
    
  revenue_model:
    - type: subscription
    - pricing_tiers:
        - free: 100 executions/month
        - pro: 10,000 executions/month ($99)
        - enterprise: Unlimited ($499)
    - trial_period: 14 days
```

### Capability Discovery
```yaml
discovery:
  registry_entry:
    name: browser-automation-studio
    category: automation
    capabilities:
      - visual-workflow-builder
      - browser-automation
      - ui-validation
      - web-scraping
      - screenshot-capture
    interfaces:
      - api: http://localhost:8090/api/v1
      - cli: browser-automation-studio
      - events: browser-automation.*
      
  metadata:
    description: Visual browser automation with AI-powered debugging
    keywords: [browser, automation, testing, scraping, workflow, visual]
    dependencies: [browserless, postgres, minio]
    enhances: [all UI scenarios]
```

## ✅ Validation Criteria

### Declarative Test Specification
```yaml
# REQUIRED: scenario-test.yaml in scenario root
version: 1.0
scenario: browser-automation-studio

structure:
  required_files:
    - .vrooli/service.json
    - PRD.md
    - api/main.go
    - api/go.mod
    - cli/browser-automation-studio
    - cli/install.sh
    - ui/package.json
    - ui/vite.config.ts
    - initialization/storage/postgres/schema.sql
    - scenario-test.yaml
    
resources:
  required: [browserless, postgres, minio]
  optional: [redis, ollama]
  health_timeout: 60

tests:
  - name: "API health check"
    type: http
    service: api
    endpoint: /health
    method: GET
    expect:
      status: 200
      
  - name: "Create workflow via API"
    type: http
    service: api
    endpoint: /api/v1/workflows/create
    method: POST
    body:
      name: "test-workflow"
      folder_path: "/tests"
    expect:
      status: 201
      body:
        workflow_id: "*"
        
  - name: "CLI workflow list command"
    type: exec
    command: ./cli/browser-automation-studio workflow list --json
    expect:
      exit_code: 0
      output_contains: ["workflows"]
```

## 📝 Implementation Notes

### Design Decisions
**React Flow for Visual Builder**: Chosen for maturity, extensive documentation, and customizability
- Alternative considered: Rete.js, Vue Flow
- Decision driver: React Flow has the best ecosystem and community
- Trade-offs: Larger bundle size for better features

**WebSocket for Live Updates**: Real-time screenshot streaming requires persistent connections
- Alternative considered: Server-sent events, polling
- Decision driver: Bidirectional communication needed for control
- Trade-offs: More complex infrastructure for better UX

### Known Limitations
- **Browser Resource Limits**: Browserless can handle ~10 concurrent sessions
  - Workaround: Queue system for execution requests
  - Future fix: Scale browserless horizontally
  
- **Screenshot Storage**: MinIO storage can grow quickly
  - Workaround: Retention policies and compression
  - Future fix: Tiered storage with archival

### Security Considerations
- **Data Protection**: Screenshots may contain sensitive data - encrypted at rest
- **Access Control**: Role-based access to workflows and executions
- **Audit Trail**: All workflow executions logged with user context

---

**Last Updated**: 2025-01-05  
**Status**: Draft  
**Owner**: AI Agent  
**Review Cycle**: After each major feature addition