# Product Requirements Document (PRD)

## 🎯 Capability Definition

### Core Capability
**What permanent capability does this scenario add to Vrooli?**
Real-time issue capture and agent spawning for rapid iteration. Provides a persistent overlay that captures context (screenshot, DOM state, errors) and spawns focused agents to fix issues immediately, creating a zero-friction feedback loop for continuous improvement.

### Intelligence Amplification
**How does this capability make future agents smarter?**
Every issue reported includes rich context (visual, technical, environmental) that agents can learn from. Patterns in issues are tracked, enabling proactive fixes. The system builds a knowledge base of common problems and their solutions, making each subsequent fix faster and more accurate.

### Recursive Value
**What new scenarios become possible after this exists?**
- **auto-tester**: Automatically generates tests based on captured issues and their fixes
- **ux-optimizer**: Analyzes interaction patterns from screenshots to suggest UI improvements  
- **agent-performance-analyzer**: Tracks which agents successfully fix which types of issues
- **proactive-issue-detector**: Scans scenarios preemptively for patterns matching previous issues
- **developer-experience-tracker**: Measures iteration speed and identifies bottlenecks in the development flow

## 📊 Success Metrics

### Functional Requirements
- **Must Have (P0)**
  - [ ] Electron app with global hotkey activation (Cmd+Shift+Space / Ctrl+Shift+Space)
  - [ ] Screenshot capture of current screen with annotation capability
  - [ ] Prompt input field for describing issues/improvements
  - [ ] Agent spawning with full context injection (screenshot, URL, scenario name)
  - [ ] Task creation in backlog with link to agent session
  - [ ] Persistent overlay that doesn't interfere with underlying applications
  
- **Should Have (P1)**
  - [ ] Context enrichment (DOM state, console errors, network requests)
  - [ ] Agent type selection (claude-code, agent-s2, custom)
  - [ ] Issue categorization and tagging system
  - [ ] History view of recent issues and their resolution status
  - [ ] Quick action buttons for common issue types
  
- **Nice to Have (P2)**
  - [ ] Voice input for issue description
  - [ ] Pattern recognition for duplicate issues
  - [ ] Integration with CI/CD for automatic validation of fixes
  - [ ] Multi-monitor support with smart overlay positioning

### Performance Criteria
| Metric | Target | Measurement Method |
|--------|--------|-------------------|
| Activation Time | < 100ms from hotkey press | Electron performance monitoring |
| Screenshot Capture | < 500ms for full screen | Internal timing logs |
| Agent Spawn Time | < 2s to launch with context | API response tracking |
| Memory Usage | < 100MB when idle | System monitor |
| Overlay Responsiveness | 60fps animations | Chrome DevTools profiling |

### Quality Gates
- [ ] All P0 requirements implemented and tested
- [ ] Electron app packages successfully for all platforms
- [ ] Global hotkey works across all major applications
- [ ] Screenshot captures include all monitors correctly
- [ ] Agent receives complete context package

## 🏗️ Technical Architecture

### Resource Dependencies
```yaml
required:
  - resource_name: claude-code
    purpose: Primary agent for code fixes
    integration_pattern: CLI invocation with context injection
    access_method: vrooli agent spawn claude-code --context [json]
    
  - resource_name: postgres
    purpose: Store issue history and patterns
    integration_pattern: Direct API connection
    access_method: Resource CLI for schema management
    
optional:
  - resource_name: qdrant
    purpose: Vector search for similar issues
    fallback: Linear search through postgres
    access_method: resource-qdrant CLI
    
  - resource_name: browserless
    purpose: Enhanced DOM capture for web scenarios
    fallback: Basic screenshot only
    access_method: resource-browserless CLI
```

### Resource Integration Standards
```yaml
integration_priorities:
  1_shared_workflows:
    - workflow: agent-spawner.json
      location: initialization/automation/n8n/
      purpose: Orchestrates agent creation with context
  
  2_resource_cli:
    - command: vrooli agent spawn [type]
      purpose: Launch agents with context
    - command: resource-postgres query
      purpose: Store and retrieve issue history
  
  3_direct_api:
    - justification: Electron needs direct IPC for performance
      endpoint: Internal Electron IPC channels
```

### Data Models
```yaml
primary_entities:
  - name: Issue
    storage: postgres
    schema: |
      {
        id: UUID
        timestamp: DateTime
        screenshot_path: String
        scenario_name: String
        url: String
        description: Text
        context_data: JSONB
        agent_session_id: UUID
        status: Enum(captured, assigned, fixing, resolved, failed)
        resolution_notes: Text
        tags: Array<String>
      }
    relationships: Links to agent sessions, related issues

  - name: Pattern
    storage: qdrant/postgres
    schema: |
      {
        id: UUID
        pattern_type: String
        frequency: Integer
        affected_scenarios: Array<String>
        suggested_fix: Text
        embedding: Vector
      }
```

### API Contract
```yaml
endpoints:
  - method: POST
    path: /api/v1/assistant/capture
    purpose: Capture issue with screenshot and context
    input_schema: |
      {
        screenshot: Base64String
        description: String
        scenario: String
        url: String
        context: {
          dom_state: Object
          console_errors: Array
          network_requests: Array
        }
      }
    output_schema: |
      {
        issue_id: UUID
        agent_session_id: UUID
        task_id: UUID
        status: String
      }
    sla:
      response_time: 500ms
      availability: 99.9%

  - method: GET
    path: /api/v1/assistant/history
    purpose: Retrieve issue history
    output_schema: |
      {
        issues: Array<Issue>
        patterns: Array<Pattern>
      }
```

### Event Interface
```yaml
published_events:
  - name: assistant.issue.captured
    payload: Issue object with full context
    subscribers: agent-spawner, pattern-analyzer, task-manager
    
  - name: assistant.agent.spawned
    payload: Agent session details
    subscribers: monitoring systems, cost tracker

consumed_events:
  - name: agent.session.completed
    action: Update issue status and extract learnings
    
  - name: task.status.changed
    action: Update UI to reflect fix progress
```

## 🖥️ CLI Interface Contract

### Command Structure
```yaml
cli_binary: vrooli-assistant
install_script: cli/install.sh

required_commands:
  - name: status
    description: Show assistant daemon status
    flags: [--json, --verbose]
    
  - name: help
    description: Display command help
    flags: [--all, --command <name>]
    
  - name: version
    description: Show version information
    flags: [--json]

custom_commands:
  - name: capture
    description: Manually trigger issue capture
    api_endpoint: /api/v1/assistant/capture
    arguments:
      - name: description
        type: string
        required: true
        description: Issue description
    flags:
      - name: --screenshot
        description: Include screenshot
      - name: --context
        description: Include full context
    output: Issue ID and agent session ID

  - name: history
    description: View recent issues
    api_endpoint: /api/v1/assistant/history
    flags:
      - name: --limit
        description: Number of issues to show
      - name: --status
        description: Filter by status
    output: Formatted issue list

  - name: daemon
    description: Control the overlay daemon
    arguments:
      - name: action
        type: string
        required: true
        description: start|stop|restart
    output: Daemon status
```

## 🔄 Integration Requirements

### Upstream Dependencies
**What capabilities must exist before this can function?**
- **Agent Infrastructure**: Claude-code and agent-s2 must be available for spawning
- **Task Management**: Backlog system must accept programmatic task creation
- **Resource Access**: Postgres and optional resources must be accessible

### Downstream Enablement
**What future capabilities does this unlock?**
- **Automated Testing**: Test generation based on captured issues
- **Pattern Analysis**: ML-based issue prediction and prevention
- **Developer Metrics**: Iteration speed and productivity tracking
- **Proactive Fixes**: Automatic issue detection before user reports

### Cross-Scenario Interactions
```yaml
provides_to:
  - scenario: all scenarios
    capability: Real-time issue reporting and fixing
    interface: Electron overlay + API
    
  - scenario: agent-dashboard
    capability: Issue tracking data for agent performance
    interface: Events and API
    
  - scenario: system-monitor
    capability: Development iteration metrics
    interface: API

consumes_from:
  - scenario: task-planner
    capability: Task creation and tracking
    fallback: Direct file writes to backlog
    
  - scenario: agent-dashboard
    capability: Agent session monitoring
    fallback: Basic session tracking only
```

## 🎨 Style and Branding Requirements

### UI/UX Style Guidelines
```yaml
style_profile:
  category: technical
  inspiration: Linear/Raycast command palette meets Loom screen recorder
  
  visual_style:
    color_scheme: dark with accent colors
    typography: modern monospace for code, sans-serif for UI
    layout: minimal floating overlay
    animations: subtle slide and fade
  
  personality:
    tone: efficient and helpful
    mood: focused and unobtrusive
    target_feeling: empowerment and speed

design_principles:
  - Invisible until needed (hotkey activation)
  - Minimal UI that doesn't obstruct work
  - Fast capture with smart defaults
  - Clear visual feedback for all actions
  - Keyboard-first navigation
```

### Target Audience Alignment
- **Primary Users**: Developers using Vrooli scenarios
- **User Expectations**: Fast, unobtrusive, developer-friendly
- **Accessibility**: Keyboard navigation, screen reader support
- **Responsive Design**: Overlay adapts to screen size and position

## 💰 Value Proposition

### Business Value
- **Primary Value**: 10x faster iteration speed on scenario improvements
- **Revenue Potential**: Reduces development cost by 30-50%
- **Cost Savings**: Eliminates context switching, reduces time to fix from hours to minutes
- **Market Differentiator**: Only platform with built-in meta-improvement capability

### Technical Value
- **Reusability Score**: 100% - enhances every single scenario
- **Complexity Reduction**: Makes reporting and fixing issues a single hotkey press
- **Innovation Enablement**: Creates foundation for self-improving system

## 🧬 Evolution Path

### Version 1.0 (Current)
- Core Electron overlay with hotkey
- Basic screenshot and prompt capture
- Agent spawning with context
- Task creation in backlog

### Version 2.0 (Planned)
- Pattern recognition and duplicate detection
- Voice input support
- Multi-agent orchestration for complex issues
- Integration with CI/CD pipelines

### Long-term Vision
- Fully autonomous issue detection and resolution
- Predictive issue prevention based on code changes
- Cross-scenario learning and optimization
- Natural language programming through issue descriptions

## 🔄 Scenario Lifecycle Integration

### Direct Scenario Deployment
```yaml
direct_execution:
  supported: true
  structure_compliance:
    - Electron app with auto-start capability
    - System tray integration for persistence
    - Cross-platform packaging (Mac, Windows, Linux)
    
  deployment_targets:
    - local: Electron desktop application
    - cloud: Not applicable (desktop only)
    
  revenue_model:
    - type: Platform enhancement (internal tool)
    - value: Productivity multiplier for all scenarios
```

### Capability Discovery
```yaml
discovery:
  registry_entry:
    name: vrooli-assistant
    category: meta-tools
    capabilities: [issue-capture, agent-spawning, context-preservation]
    interfaces:
      - api: http://localhost:3250/api/v1/assistant
      - cli: vrooli-assistant
      - electron: Global hotkey overlay
      
  metadata:
    description: Real-time issue capture and agent spawning overlay
    keywords: [meta, improvement, iteration, feedback, electron, overlay]
    dependencies: [electron, agent-infrastructure]
    enhances: [all scenarios]
```

## 🚨 Risk Mitigation

### Technical Risks
| Risk | Probability | Impact | Mitigation |
|------|------------|--------|------------|
| Hotkey conflicts | Medium | Low | Configurable hotkeys, conflict detection |
| Electron performance | Low | Medium | Lazy loading, efficient IPC |
| Screenshot failures | Low | High | Multiple capture methods, fallbacks |
| Agent spawn failures | Medium | High | Retry logic, queue system |

### Operational Risks
- **Security**: Screenshots may capture sensitive data - add blur/redaction tools
- **Resource Usage**: Persistent Electron app - implement sleep mode
- **Compatibility**: Cross-platform differences - extensive testing per OS

## ✅ Validation Criteria

### Declarative Test Specification
```yaml
version: 1.0
scenario: vrooli-assistant

structure:
  required_files:
    - .vrooli/service.json
    - PRD.md
    - ui/electron/main.js
    - ui/electron/preload.js
    - ui/electron/renderer.js
    - api/main.go
    - cli/vrooli-assistant
    - cli/install.sh
    - initialization/storage/postgres/schema.sql
    - scenario-test.yaml
    
  required_dirs:
    - api
    - cli
    - ui/electron
    - initialization/storage
    - data

resources:
  required: [postgres]
  optional: [qdrant, browserless, claude-code, agent-s2]
  health_timeout: 30

tests:
  - name: "Electron app launches"
    type: exec
    command: npm run electron:test
    expect:
      exit_code: 0
      
  - name: "Global hotkey registers"
    type: exec
    command: ./cli/vrooli-assistant test-hotkey
    expect:
      exit_code: 0
      output_contains: ["Hotkey registered successfully"]
      
  - name: "Screenshot capture works"
    type: exec
    command: ./cli/vrooli-assistant capture --test
    expect:
      exit_code: 0
      output_contains: ["Screenshot captured"]
      
  - name: "API endpoint responds"
    type: http
    service: api
    endpoint: /api/v1/assistant/status
    method: GET
    expect:
      status: 200
      
  - name: "Database schema initialized"
    type: sql
    service: postgres
    query: "SELECT COUNT(*) FROM information_schema.tables WHERE table_name = 'issues'"
    expect:
      rows:
        - count: 1
```

## 📝 Implementation Notes

### Design Decisions
**Electron vs Browser Extension**: Chose Electron for system-wide availability and richer integrations
- Alternative considered: Browser extension
- Decision driver: Need to capture issues outside browser contexts
- Trade-offs: Larger footprint for system-wide capability

**Global Hotkey**: Single configurable hotkey for minimal friction
- Alternative considered: Multiple hotkeys for different actions  
- Decision driver: Speed and simplicity
- Trade-offs: Less granular control for more universal access

### Known Limitations
- **Platform Differences**: Hotkey behavior varies per OS
  - Workaround: Platform-specific key combinations
  - Future fix: Universal input handler in v2.0

### Security Considerations
- **Data Protection**: Screenshots stored encrypted, auto-deleted after 30 days
- **Access Control**: Only local user can trigger capture
- **Audit Trail**: All captures logged with timestamp and user

---

**Last Updated**: 2025-01-03  
**Status**: In Development  
**Owner**: Human + AI Agent Partnership  
**Review Cycle**: Weekly validation against implementation