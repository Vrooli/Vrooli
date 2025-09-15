# Product Requirements Document (PRD)

## 🎯 Capability Definition

### Core Capability
**What permanent capability does this scenario add to Vrooli?**
Real-time monitoring and basic control of all AI agents running within Vrooli resource containers. Provides a centralized cyberpunk-themed dashboard for developers to observe agent health, view logs, and perform basic agent lifecycle operations without touching the underlying resources or scenarios.

### Intelligence Amplification
**How does this capability make future agents smarter?**
This is purely a monitoring tool - it doesn't add intelligence. Instead, it provides operational visibility that enables:
- **Faster debugging**: Quickly identify which agents are having issues
- **Resource awareness**: See what agents are currently active across the system
- **Log access**: Immediate access to agent logs without CLI hunting
- **System health**: At-a-glance view of agent ecosystem status

### Recursive Value
**What new scenarios become possible after this exists?**
- **Agent health alerting scenarios**: Build on monitoring data to create notification systems
- **Resource optimization scenarios**: Use agent status information to optimize resource allocation
- **Development workflow scenarios**: Integrate agent monitoring into development dashboards
- **System administration scenarios**: Extend monitoring to include resource and scenario health

## 📊 Success Metrics

### Functional Requirements
- **Must Have (P0)**
  - [ ] Discover all agents across all installed Vrooli resources
  - [ ] Display real-time agent status (active/inactive/error states)
  - [ ] Show agent metadata (PID, uptime, command, capabilities)
  - [ ] Basic agent control (stop agents via resource CLI)
  - [ ] Real-time log viewer with follow capability
  - [ ] Auto-refresh every 30 seconds with poll rate limiting
  - [ ] Cyberpunk-themed UI with existing scan animations
  - [ ] Radar view with moving agent dots and color-coded health status
  
- **Should Have (P1)**
  - [ ] Agent cards showing key metrics (memory, CPU if available)
  - [ ] Hover tooltips on radar showing agent details
  - [ ] Agent search/filtering by name, type, or status
  - [ ] Log export/download functionality
  - [ ] Keyboard shortcuts for common operations
  - [ ] Responsive design for different screen sizes
  
- **Nice to Have (P2)**
  - [ ] Agent performance history graphs (session-only)
  - [ ] Multiple log viewers open simultaneously
  - [ ] Agent capability matching/discovery
  - [ ] Custom radar themes and animation speeds
  - [ ] Agent grouping by resource type

### Performance Criteria
| Metric | Target | Measurement Method |
|--------|--------|-------------------|
| Agent Discovery | < 2 seconds for full scan | Time CLI operations |
| UI Responsiveness | < 100ms for status updates | Browser dev tools |
| Memory Usage | < 100MB browser memory | Performance monitoring |
| Polling Reliability | 99%+ successful polls | Error rate tracking |

### Quality Gates
- [ ] All P0 requirements implemented and tested
- [ ] Works with at least 5 different resource types
- [ ] Handles missing/broken resource CLIs gracefully
- [ ] UI remains responsive with 20+ active agents
- [ ] No memory leaks during extended monitoring sessions

## 🏗️ Technical Architecture

### Resource Dependencies
```yaml
required:
  - None (purely monitors existing resources)
    
optional:
  - Any Vrooli resource with agent support:
      - ollama, claude-code, cline, autogen-studio, autogpt
      - crewai, gemini, langchain, litellm, openrouter
      - whisper, comfyui, pandas-ai, parlant, huginn, opencode
```

### Resource Integration Standards
```yaml
integration_priorities:
  1_resource_cli:
    - command: resource-{name} agents list --json
      purpose: Discover all agents within a resource
      output_format: |
        {
          "agents": {
            "agent-id": {
              "id": "string",
              "pid": number,
              "status": "running|stopped|crashed",
              "start_time": "ISO8601",
              "last_seen": "ISO8601", 
              "command": "string"
            }
          }
        }
    
    - command: resource-{name} agents stop <agent-id>
      purpose: Stop a specific agent
      
    - command: resource-{name} agents logs <agent-id> [--follow]
      purpose: View agent logs with optional follow mode
  
  2_direct_execution:
    - justification: No direct resource access needed - uses CLI abstraction layer
```

### Data Models
```yaml
session_only_data:
  - name: Agent
    storage: browser_memory
    schema: |
      {
        id: string,
        name: string,
        type: string,              // resource type (ollama, claude-code, etc)
        status: "active|inactive|error",
        pid: number,
        start_time: timestamp,
        last_seen: timestamp,
        uptime: string,            // human readable
        command: string,
        capabilities: string[],
        metrics: {
          memory_mb?: number,
          cpu_percent?: number,
          custom_fields: any
        },
        radar_position: {          // for radar visualization
          x: number,
          y: number,
          target_x: number,
          target_y: number
        }
      }
  
  - name: ResourceScanResult  
    storage: browser_memory
    schema: |
      {
        resource_name: string,
        scan_timestamp: timestamp,
        success: boolean,
        error?: string,
        agents_found: number,
        scan_duration_ms: number
      }
```

### API Contract
```yaml
endpoints:
  - method: GET
    path: /health
    purpose: Service health check
    output_schema: |
      {
        status: "healthy|degraded|down",
        service: "agent-dashboard-api",
        version: string,
        timestamp: string
      }
      
  - method: GET
    path: /api/v1/agents
    purpose: Get all discovered agents across resources
    output_schema: |
      {
        agents: Agent[],
        last_scan: timestamp,
        scan_in_progress: boolean,
        errors: ResourceScanResult[]
      }
      
  - method: POST
    path: /api/v1/agents/{id}/stop
    purpose: Stop a specific agent
    output_schema: |
      {
        success: boolean,
        message: string,
        agent_id: string
      }
      
  - method: GET
    path: /api/v1/agents/{id}/logs
    purpose: Get agent logs (supports SSE for follow mode)
    query_params:
      - follow: boolean (enables SSE streaming)
      - lines: number (number of recent lines)
    output_schema: |
      {
        logs: string[],
        agent_id: string,
        timestamp: string
      }
      
  - method: POST
    path: /api/v1/scan
    purpose: Trigger immediate agent discovery scan
    output_schema: |
      {
        scan_started: boolean,
        estimated_duration_ms: number
      }
```

## 🖥️ CLI Interface Contract

### Command Structure
```yaml
cli_binary: agent-dashboard
install_script: cli/install.sh

required_commands:
  - name: status
    description: Show dashboard status and agent summary
    flags: [--json, --verbose]
    
  - name: help
    description: Display command help and usage
    flags: [--all, --command <name>]
    
  - name: version
    description: Show CLI and API version information
    flags: [--json]

custom_commands:
  - name: list
    description: List all discovered agents
    api_endpoint: /api/v1/agents
    flags:
      - name: --resource
        description: Filter by resource type
      - name: --status
        description: Filter by agent status
      - name: --json
        description: Output raw JSON
    output: Formatted table of agents with status
    
  - name: stop
    description: Stop a specific agent
    api_endpoint: /api/v1/agents/{id}/stop
    arguments:
      - name: agent_id
        type: string
        required: true
        description: ID of agent to stop
    output: Success/failure message
    
  - name: logs
    description: View agent logs
    api_endpoint: /api/v1/agents/{id}/logs
    arguments:
      - name: agent_id
        type: string  
        required: true
        description: ID of agent to view logs for
    flags:
      - name: --follow
        description: Follow logs in real-time
      - name: --lines
        description: Number of recent lines to show
    output: Agent log output
    
  - name: scan
    description: Trigger immediate agent discovery
    api_endpoint: /api/v1/scan
    output: Scan initiation confirmation
```

## 🎨 Style and Branding Requirements

### UI/UX Style Guidelines
```yaml
style_profile:
  category: cyberpunk_dev_tool
  inspiration: Matrix command centers meets GitHub dark mode
  
  visual_style:
    color_scheme: dark cyberpunk (current cyan/magenta/yellow palette)
    typography: Orbitron headers, Exo 2 body, Share Tech Mono code
    layout: grid-based dashboard with radar overlay
    animations: scanning lines, data streams, radar sweep
  
  personality:
    tone: playful but functional
    mood: high-tech, futuristic, engaging
    target_feeling: "Command center for my AI empire"

key_ui_components:
  agent_cards:
    - Matrix-style agent information panels
    - Color-coded status indicators (green=active, red=error, gray=inactive)
    - Hover effects with glitch animations
    - Action buttons for logs/stop operations
    
  radar_view:
    - Circular radar with rotating scanner line
    - Agents represented as moving colored dots
    - Smooth position transitions (ease-in-out animations)
    - Hover tooltips showing agent details
    - Agent health determines dot color and pulse effect
    
  log_viewer:
    - Terminal-style modal dialogs
    - Real-time log streaming with auto-scroll
    - Syntax highlighting for error/warning levels
    - Cyberpunk window chrome with scan line effects

style_references:
  technical:
    - tron_legacy: "Blue/cyan color schemes and geometric interfaces"
    - matrix_reloaded: "Code rain and terminal aesthetics"
    - github_dark: "Professional readability with dark themes"
    
  inspiration: "LCARS meets modern dev tools - functional but fun"
```

### Target Audience Alignment
- **Primary User**: Vrooli developers (you) doing day-to-day development
- **User Expectations**: Quick access to agent status, easy log viewing, intuitive controls
- **Accessibility**: Keyboard navigation, high contrast mode available
- **Responsive Design**: Works on ultrawide monitors and laptops

### Brand Consistency Rules
- **Scenario Identity**: Internal dev tool with personality - professional enough to be useful, fun enough to be engaging
- **Vrooli Integration**: Uses standard lifecycle patterns while being visually distinct
- **Fun vs Function**: Strongly fun - this is where we can be creative and whimsical

## 💰 Value Proposition

### Business Value
- **Primary Value**: Developer productivity and debugging efficiency
- **Cost Savings**: Reduces time spent hunting through CLI commands and logs
- **Development Experience**: Makes working with complex multi-agent systems enjoyable
- **System Reliability**: Faster issue detection and resolution

### Technical Value
- **Reusability Score**: Medium - primarily a standalone monitoring tool
- **Complexity Reduction**: Eliminates need to remember agent CLI commands across resources
- **Innovation Enablement**: Provides foundation for future agent ecosystem tooling

## 🧬 Evolution Path

### Version 1.0 (Current Goal)
- Real-time agent discovery and monitoring
- Basic agent control (stop operations)
- Cyberpunk UI with radar visualization
- Log viewer with real-time streaming
- CLI interface for scripting

### Version 2.0 (Future Enhancements)
- Agent performance analytics and trends
- Custom alert/notification system
- Integration with IDE plugins for development workflow
- Multi-server monitoring (remote Vrooli instances)
- Agent deployment and scaling suggestions

### Long-term Vision
- **Ecosystem Integration**: Central hub for all Vrooli operational monitoring
- **AI-Powered Insights**: Pattern recognition for agent behavior and performance
- **Cross-Platform Support**: Native desktop apps and mobile monitoring
- **Community Features**: Share agent configurations and monitoring setups

## 🔄 Scenario Lifecycle Integration

### Direct Scenario Deployment
```yaml
direct_execution:
  supported: true
  structure_compliance:
    - service.json with dynamic port allocation
    - API server with agent discovery and control
    - CLI tool with resource integration
    - UI dashboard with cyberpunk radar visualization
    
  deployment_targets:
    - local: Standard Vrooli scenario deployment
    - development: Primary use case for Vrooli developers
    
  revenue_model:
    - type: internal_tool
    - pricing: free (developer productivity tool)
```

### Capability Discovery
```yaml
discovery:
  registry_entry:
    name: agent-dashboard
    category: monitoring
    capabilities: 
      - real_time_agent_monitoring
      - agent_lifecycle_control
      - log_streaming
      - cyberpunk_visualization
    interfaces:
      - api: http://localhost:${API_PORT}/api/v1
      - cli: agent-dashboard
      - ui: http://localhost:${UI_PORT}
      
  metadata:
    description: "Cyberpunk-themed monitoring dashboard for AI agents within Vrooli resources"
    keywords: [monitoring, agents, dashboard, cyberpunk, logs, development]
    dependencies: []
    enhances: [developer-experience, debugging, system-monitoring]
```

### Version Management
```yaml
versioning:
  current: 1.0.0
  minimum_compatible: 1.0.0
  
  breaking_changes: []
  deprecations: []
```

## 🚨 Risk Mitigation

### Technical Risks
| Risk | Probability | Impact | Mitigation |
|------|------------|--------|------------|
| Resource CLI changes breaking discovery | Medium | Medium | Graceful error handling, fallback discovery methods |
| Browser memory leaks with long sessions | Low | Medium | Implement data cleanup, session monitoring |
| Agent state polling creating resource load | Low | Low | Rate limiting, exponential backoff |
| UI performance with many active agents | Medium | Low | Virtual scrolling, optimized rendering |

### Operational Risks
- **Resource Availability**: Handles missing/broken resource CLIs gracefully
- **Agent State Changes**: Polling-based updates may miss rapid state changes (acceptable)
- **Log Volume**: Large log outputs handled with streaming and truncation
- **Cross-Platform**: CLI commands work consistently across development environments

## ✅ Validation Criteria

### Declarative Test Specification
```yaml
version: 1.0
scenario: agent-dashboard

structure:
  required_files:
    - .vrooli/service.json
    - PRD.md
    - api/main.go
    - api/go.mod
    - cli/agent-dashboard
    - cli/install.sh
    - scenario-test.yaml
    - ui/index.html
    - ui/server.js
    - ui/script.js
    
  required_dirs:
    - api
    - cli
    - ui
    - test

resources:
  required: []
  optional: [all agent-supporting resources]
  health_timeout: 30

tests:
  - name: "API health endpoint responds"
    type: http
    service: api
    endpoint: /health
    method: GET
    expect:
      status: 200
      
  - name: "Agent discovery works with mock data"
    type: http
    service: api
    endpoint: /api/v1/agents
    method: GET
    expect:
      status: 200
      body:
        agents: "[array]"
        
  - name: "UI loads successfully"
    type: http
    service: ui
    endpoint: /
    method: GET
    expect:
      status: 200
      
  - name: "CLI help command works"
    type: exec
    command: ./cli/agent-dashboard help
    expect:
      exit_code: 0
      output_contains: ["Agent Dashboard CLI", "Commands:"]
```

### Performance Validation
- [ ] Agent discovery completes within 2 seconds
- [ ] UI remains responsive with 20+ agents
- [ ] Log streaming handles high-volume output
- [ ] Radar animations run smoothly at 60fps

### Integration Validation
- [ ] Works with at least 3 different resource types
- [ ] Handles resource CLI errors gracefully
- [ ] Respects Vrooli lifecycle management
- [ ] CLI integrates properly with shell environments

### User Experience Validation
- [ ] Agent status clearly visible at a glance
- [ ] Log viewer easy to use and follow
- [ ] Radar view entertaining but not distracting
- [ ] Overall interface feels polished and fun

## 📝 Implementation Notes

### Design Decisions
**Polling vs Event-Driven**: Chose polling for simplicity and reliability
- Alternative considered: WebSocket connections to resources
- Decision driver: Resource CLI standardization more mature than event systems
- Trade-offs: Slight latency vs implementation complexity

**Session-Only Data**: No persistent storage for agent history
- Alternative considered: SQLite database for trends
- Decision driver: Monitoring tool, not analytics platform
- Trade-offs: No historical data vs simplified architecture

### Known Limitations
- **Resource CLI Dependency**: Requires standardized CLI across all resources
  - Workaround: Graceful degradation for non-standard resources
  - Future fix: Resource API standardization

- **Real-time Accuracy**: 30-second polling may miss rapid state changes
  - Workaround: Manual refresh capability
  - Future fix: Event-driven updates when available

### Security Considerations
- **Agent Control**: Only supports stop operations, not arbitrary commands
- **Log Access**: No sensitive data filtering (assumes dev environment)
- **Resource Isolation**: Uses CLI abstraction, no direct resource access

## 🔗 References

### Documentation
- README.md - User-facing overview and quick start
- docs/radar.md - Radar visualization specification
- docs/cli.md - CLI command reference
- docs/integration.md - Resource CLI integration guide

### Related Scenarios
- system-monitor: Complementary resource and system monitoring
- maintenance-orchestrator: Higher-level system maintenance automation

### External Resources
- [Vrooli Resource CLI Standards](../../docs/resources.md) - Resource CLI interface specification
- [Cyberpunk Design Patterns](https://cyberpunk.design) - UI design inspiration
- [LCARS Interface Guidelines](https://lcars.org) - Functional interface design

---

**Last Updated**: 2025-09-15  
**Status**: Active Development  
**Owner**: Vrooli Core Developer  
**Review Cycle**: Validate against implementation after each sprint