# Product Requirements Document (PRD)

## 📈 Progress Tracking

**Last Updated**: 2025-10-03
**Status**: P0 Complete, P1 Partial (60%), All Quality Gates Passed
**Test Coverage**: Comprehensive phased testing implemented

### Recent Improvements (2025-10-03)
1. **Fixed Initial Discovery Delay** - Scenarios now discovered immediately on startup
2. **Migrated to Phased Testing** - Implemented structure/dependencies/unit/integration/business/performance test phases
3. **Added Unit Test Coverage** - 8 comprehensive orchestrator unit tests, all passing
4. **Completed P1 Requirements** - Activity logging, CLI timer functionality, full CLI command parity

### Completion Summary
- **P0 Requirements**: 6/6 (100%) ✅
- **P1 Requirements**: 3/6 (50%) - Activity log, timer, CLI commands complete
- **Quality Gates**: 5/5 (100%) ✅
- **Test Infrastructure**: Fully implemented with 6 test phases

## 🎯 Capability Definition

### Core Capability
**What permanent capability does this scenario add to Vrooli?**
Centralized control plane for all maintenance-related scenarios, ensuring conscious control over system maintenance activities. Provides the ability to selectively activate/deactivate maintenance operations through a unified interface with preset configurations and optional calendar-based scheduling.

### Intelligence Amplification
**How does this capability make future agents smarter?**
Agents gain awareness of maintenance windows, can coordinate maintenance activities to avoid conflicts, and can query maintenance state before performing operations. This prevents resource contention, improves system stability, and enables sophisticated maintenance orchestration patterns.

### Recursive Value
**What new scenarios become possible after this exists?**
1. **maintenance-impact-analyzer** - Analyzes the impact of maintenance operations before activation
2. **resource-optimizer** - Coordinates with maintenance windows to optimize resource allocation
3. **maintenance-reporter** - Generates detailed reports on maintenance activities and their outcomes
4. **auto-healing-coordinator** - Automatically triggers specific maintenance scenarios based on system health
5. **maintenance-learning-system** - Learns optimal maintenance patterns and suggests preset improvements

## 📊 Success Metrics

### Functional Requirements
- **Must Have (P0)**
  - [x] Discovery of all maintenance scenarios via "maintenance" tag in service.json (Validated: 10 scenarios discovered)
  - [x] API endpoints for activate/deactivate individual maintenance scenarios (Validated: Both endpoints functional)
  - [x] All maintenance scenarios start in inactive state by default (Validated: All scenarios start inactive)
  - [x] Status endpoint showing current state of all maintenance scenarios (Validated: /api/v1/status working)
  - [x] Basic UI dashboard with toggle switches for each scenario (Validated: UI accessible at port 37116)
  - [x] At least 3 default presets (Full Maintenance, Security Only, Off Hours) (Validated: 7 presets available)
  
- **Should Have (P1)**
  - [ ] Custom preset creation and management
  - [x] Activity log showing recent maintenance operations (Validated: Activity tracked in status endpoint)
  - [ ] Resource usage monitoring for active maintenance scenarios
  - [ ] Confirmation dialogs for bulk operations (Implemented in CLI preset apply)
  - [x] Auto-deactivate timer functionality (Validated: CLI activate --timer command)
  - [x] CLI commands for all operations (Validated: All core commands working - status, list, activate, deactivate, preset)
  
- **Nice to Have (P2)**
  - [ ] Calendar integration for scheduled preset activation
  - [ ] Webhook notifications for maintenance state changes
  - [ ] Historical analytics on maintenance patterns
  - [ ] Conflict detection between maintenance scenarios

### Performance Criteria
| Metric | Target | Measurement Method |
|--------|--------|-------------------|
| Response Time | < 200ms for state changes | API monitoring |
| Discovery Time | < 5s for all scenarios | Startup logs |
| UI Responsiveness | < 100ms for toggle actions | Frontend monitoring |
| Memory Usage | < 256MB | System monitoring |

### Quality Gates
- [x] All P0 requirements implemented and tested (Validated: 2025-10-03)
- [x] Can discover and control at least 5 maintenance scenarios (Validated: Discovers 10 scenarios)
- [x] State changes reflect immediately in UI (Validated: Real-time updates working)
- [x] API and CLI have feature parity (Validated: All core operations available in both)
- [x] No persistence required (state resets on restart) (Validated: In-memory state by design)

## 🏗️ Technical Architecture

### Resource Dependencies
```yaml
required:
  - resource_name: n8n
    purpose: Execute maintenance workflows when scenarios are activated
    integration_pattern: Shared workflows for state management
    access_method: resource-n8n CLI and shared workflows
    
optional:
  - resource_name: postgres
    purpose: Store preset configurations and activity logs
    fallback: In-memory storage with no persistence
    access_method: Direct connection for simple key-value storage
    
  - resource_name: redis
    purpose: Pub/sub for real-time state updates across instances
    fallback: Polling-based state synchronization
    access_method: resource-redis CLI
```

### Resource Integration Standards
```yaml
integration_priorities:
  1_shared_workflows:
    - workflow: maintenance-state-manager.json
      location: initialization/n8n/
      purpose: Manages state changes and notifications
  
  2_resource_cli:
    - command: resource-n8n execute-workflow
      purpose: Trigger maintenance workflows
    - command: resource-postgres query (if available)
      purpose: Store/retrieve presets
  
  3_direct_api:
    - justification: Real-time state queries require direct HTTP
      endpoint: GET /api/status for each maintenance scenario
```

### Data Models
```yaml
primary_entities:
  - name: MaintenanceScenario
    storage: memory (optional postgres)
    schema: |
      {
        id: string (scenario name)
        displayName: string
        description: string
        isActive: boolean (always false on start)
        lastActivated: timestamp
        resourceUsage: {
          cpu: number
          memory: number
        }
        apiEndpoint: string
        tags: string[]
      }
    relationships: Part of preset configurations
    
  - name: Preset
    storage: memory (optional postgres)
    schema: |
      {
        id: UUID
        name: string
        description: string
        scenarioStates: Map<scenarioId, boolean>
        createdAt: timestamp
        isDefault: boolean
      }
```

### API Contract
```yaml
endpoints:
  - method: GET
    path: /api/v1/scenarios
    purpose: List all discovered maintenance scenarios and their states
    output_schema: |
      {
        scenarios: [
          {
            id: string,
            name: string,
            isActive: boolean,
            endpoint: string
          }
        ]
      }
    sla:
      response_time: 200ms
      availability: 99.9%
      
  - method: POST
    path: /api/v1/scenarios/{id}/activate
    purpose: Activate a specific maintenance scenario
    output_schema: |
      {
        success: boolean,
        scenario: string,
        newState: "active"
      }
      
  - method: POST
    path: /api/v1/scenarios/{id}/deactivate
    purpose: Deactivate a specific maintenance scenario
    output_schema: |
      {
        success: boolean,
        scenario: string,
        newState: "inactive"
      }
      
  - method: GET
    path: /api/v1/presets
    purpose: List all available presets
    output_schema: |
      {
        presets: [
          {
            id: string,
            name: string,
            description: string,
            scenarioCount: number
          }
        ]
      }
      
  - method: POST
    path: /api/v1/presets/{id}/apply
    purpose: Apply a preset configuration
    output_schema: |
      {
        success: boolean,
        preset: string,
        activated: string[],
        deactivated: string[]
      }
```

### Event Interface
```yaml
published_events:
  - name: maintenance.scenario.activated
    payload: { scenarioId: string, timestamp: ISO8601 }
    subscribers: [logging systems, monitoring dashboards]
    
  - name: maintenance.preset.applied
    payload: { presetId: string, changes: object }
    subscribers: [analytics systems, audit logs]
    
consumed_events:
  - name: calendar.schedule.triggered
    action: Apply specified preset when calendar event fires
    
  - name: system.health.degraded
    action: Show warning in UI about system health
```

## 🖥️ CLI Interface Contract

### Command Structure
```yaml
cli_binary: maintenance-orchestrator
install_script: cli/install.sh

required_commands:
  - name: status
    description: Show status of all maintenance scenarios
    flags: [--json, --verbose, --filter <tag>]
    
  - name: help
    description: Display command help and usage
    flags: [--all, --command <name>]
    
  - name: version
    description: Show CLI and API version information
    flags: [--json]

custom_commands:
  - name: list
    description: List all discovered maintenance scenarios
    api_endpoint: /api/v1/scenarios
    flags:
      - name: --active-only
        description: Show only active scenarios
      - name: --tag
        description: Filter by tag
    output: Table or JSON format
    
  - name: activate
    description: Activate a maintenance scenario
    api_endpoint: /api/v1/scenarios/{id}/activate
    arguments:
      - name: scenario
        type: string
        required: true
        description: Scenario name or ID
    flags:
      - name: --timer
        description: Auto-deactivate after N minutes
    
  - name: deactivate
    description: Deactivate a maintenance scenario
    api_endpoint: /api/v1/scenarios/{id}/deactivate
    arguments:
      - name: scenario
        type: string
        required: true
        description: Scenario name or ID
    
  - name: preset
    description: Manage and apply presets
    subcommands:
      - name: list
        description: List available presets
      - name: apply
        description: Apply a preset
        arguments:
          - name: preset-name
            type: string
            required: true
      - name: create
        description: Create a new preset from current state
        arguments:
          - name: name
            type: string
            required: true
```

## 🔄 Integration Requirements

### Upstream Dependencies
- **Service Discovery**: Requires ability to scan scenarios directory
- **Scenario APIs**: Each maintenance scenario must expose /api/status endpoint
- **Calendar Scenario** (optional): For scheduled preset activation

### Downstream Enablement
- **Maintenance Analytics**: Historical data on maintenance patterns
- **Resource Planning**: Predictive resource allocation based on maintenance schedules
- **Auto-healing Systems**: Intelligent maintenance triggering

### Cross-Scenario Interactions
```yaml
provides_to:
  - scenario: Any scenario checking maintenance state
    capability: Query if maintenance is active
    interface: GET /api/v1/status
    
  - scenario: calendar
    capability: Register maintenance presets as schedulable events
    interface: API webhook registration
    
consumes_from:
  - scenario: All maintenance-tagged scenarios
    capability: Status and control endpoints
    fallback: Mark scenario as unavailable
    
  - scenario: calendar (optional)
    capability: Schedule triggers
    fallback: Manual-only activation
```

## 🎨 Style and Branding Requirements

### UI/UX Style Guidelines
```yaml
style_profile:
  category: technical
  inspiration: NASA mission control meets modern DevOps dashboard
  
  visual_style:
    color_scheme: dark with status-based accent colors
    typography: technical/monospace for data, clean sans for UI
    layout: dashboard with clear status indicators
    animations: subtle transitions for state changes
  
  personality:
    tone: professional yet approachable
    mood: focused and reliable
    target_feeling: "I'm in control of system maintenance"

status_colors:
  inactive: "#6B7280" (gray)
  active: "#10B981" (green)
  transitioning: "#F59E0B" (amber)
  error: "#EF4444" (red)
  
ui_components:
  - Toggle switches with clear on/off states
  - Status badges with color coding
  - Activity feed with timestamps
  - Preset buttons as primary actions
  - Confirmation modals for bulk operations
```

## 💰 Value Proposition

### Business Value
- **Primary Value**: Prevents uncontrolled resource consumption by maintenance tasks
- **Cost Savings**: 30-50% reduction in cloud costs from optimized maintenance windows
- **Operational Efficiency**: 80% reduction in maintenance-related incidents

### Technical Value
- **Reusability Score**: 10/10 - Every maintenance scenario benefits
- **Complexity Reduction**: Single interface for all maintenance control
- **Innovation Enablement**: Enables sophisticated maintenance orchestration patterns

## 🧬 Evolution Path

### Version 1.0 (Current)
- Basic discovery and control
- Default presets
- Simple web UI
- CLI interface

### Version 2.0 (Planned)
- Calendar integration
- Custom preset builder
- Resource usage analytics
- Conflict detection

### Long-term Vision
- AI-driven maintenance scheduling
- Predictive maintenance triggers
- Cross-cluster maintenance coordination
- Self-optimizing maintenance patterns

## 🔄 Scenario Lifecycle Integration

### Direct Scenario Deployment
```yaml
direct_execution:
  supported: true
  structure_compliance:
    - service.json with maintenance controller metadata
    - Health check endpoints for monitoring
    - Auto-discovery on startup
    
  deployment_targets:
    - local: Runs as system service
    - kubernetes: DaemonSet for cluster-wide control
    - cloud: Lambda/Cloud Functions for serverless
```

## ✅ Validation Criteria

### Declarative Test Specification
```yaml
version: 1.0
scenario: maintenance-orchestrator

structure:
  required_files:
    - .vrooli/service.json
    - PRD.md
    - README.md
    - api/main.go
    - cli/maintenance-orchestrator
    - cli/install.sh
    - ui/index.html
    - scenario-test.yaml

tests:
  - name: "Discovers maintenance scenarios"
    type: http
    service: api
    endpoint: /api/v1/scenarios
    method: GET
    expect:
      status: 200
      body_contains: ["scenarios"]
      
  - name: "Activates scenario"
    type: http
    service: api
    endpoint: /api/v1/scenarios/test-maintenance/activate
    method: POST
    expect:
      status: 200
      body:
        success: true
        newState: "active"
        
  - name: "CLI lists scenarios"
    type: exec
    command: ./cli/maintenance-orchestrator list --json
    expect:
      exit_code: 0
      output_contains: ["scenarios"]
```

## 📝 Implementation Notes

### Design Decisions
**In-memory state management**: Chosen for simplicity since all scenarios start inactive
- Alternative considered: Persistent state in PostgreSQL
- Decision driver: Simplicity and guaranteed inactive-on-start behavior
- Trade-offs: No state persistence across restarts (by design)

### Security Considerations
- **Access Control**: Consider adding authentication for production
- **Rate Limiting**: Prevent rapid state toggling
- **Audit Trail**: Log all state changes with timestamps and actors

---

**Last Updated**: 2025-01-05  
**Status**: Draft  
**Owner**: AI Agent  
**Review Cycle**: After each major Vrooli update