# Product Requirements Document (PRD)

## 🎯 Capability Definition

### Core Capability
**What permanent capability does this scenario add to Vrooli?**
Core-debugger provides automated detection, tracking, and resolution of issues in Vrooli's core infrastructure (CLI, orchestrator, resource manager, setup scripts). It serves as a centralized health monitoring and self-healing system that prevents core issues from cascading and blocking scenario development.

### Intelligence Amplification
**How does this capability make future agents smarter?**
- Agents can query known issues and workarounds before attempting operations that might fail
- Automatic issue detection prevents agents from getting stuck on infrastructure problems
- Historical issue patterns help agents anticipate and avoid common failure modes
- Workaround database becomes collective knowledge that all agents can leverage

### Recursive Value
**What new scenarios become possible after this exists?**
1. **Auto-healing orchestrator** - Automatically fixes common orchestration issues without human intervention
2. **Intelligent retry systems** - Scenarios can check core health before operations and wait/retry intelligently
3. **Proactive maintenance** - Predictive issue detection based on patterns before failures occur
4. **Cross-scenario debugging** - Scenarios can report issues to core-debugger for centralized resolution
5. **Performance optimization** - Identify and resolve bottlenecks in core systems automatically

## 📊 Success Metrics

### Functional Requirements
- **Must Have (P0)**
  - [ ] Track core component health (CLI, orchestrator, resource manager, setup scripts)
  - [ ] File-based storage for issues/workarounds (git-trackable, no DB dependency)
  - [ ] Status page UI showing component health and active issues
  - [ ] API for querying issues and workarounds
  - [ ] Automatic issue detection via health checks
  - [ ] Claude-code integration for intelligent issue analysis
  
- **Should Have (P1)**
  - [ ] Historical issue tracking and pattern analysis
  - [ ] Automatic workaround suggestions based on issue type
  - [ ] Issue priority scoring based on impact
  - [ ] Email/webhook notifications for critical issues
  - [ ] Performance metrics tracking for core components
  
- **Nice to Have (P2)**
  - [ ] Predictive issue detection using ML patterns
  - [ ] Automatic fix attempts for known issue patterns
  - [ ] Integration with GitHub issues for external tracking
  - [ ] Distributed health monitoring across multiple Vrooli instances

### Performance Criteria
| Metric | Target | Measurement Method |
|--------|--------|-------------------|
| Health Check Latency | < 500ms per component | API monitoring |
| Issue Detection Time | < 30 seconds from occurrence | Log analysis |
| Storage Efficiency | < 100MB for 10,000 issues | File system monitoring |
| UI Load Time | < 2 seconds | Browser performance metrics |

### Quality Gates
- [ ] All P0 requirements implemented and tested
- [ ] Zero dependency on other scenarios (only claude-code resource)
- [ ] File-based storage working with git tracking
- [ ] Status page accurately reflects component health
- [ ] API responds with correct issue/workaround data

## 🏗️ Technical Architecture

### Resource Dependencies
```yaml
required:
  - resource_name: claude-code
    purpose: Intelligent issue analysis and fix generation
    integration_pattern: CLI command execution
    access_method: resource-claude-code analyze
    
optional:
  # None - this scenario must work with zero optional dependencies
```

### Resource Integration Standards
```yaml
integration_priorities:
  1_shared_workflows:     # Not applicable - no n8n dependency
    - None
  
  2_resource_cli:        
    - command: resource-claude-code analyze
      purpose: Analyze error logs and suggest fixes
  
  3_direct_api:          # Avoided to maintain zero-dependency principle
    - None

shared_workflow_criteria:
  - Not applicable for this scenario
```

### Data Models
```yaml
primary_entities:
  - name: CoreIssue
    storage: file-based (JSON)
    schema: |
      {
        id: string (timestamp-based)
        component: string (cli|orchestrator|resource-manager|setup)
        severity: string (critical|high|medium|low)
        status: string (active|resolved|workaround-available)
        description: string
        error_signature: string (for deduplication)
        first_seen: timestamp
        last_seen: timestamp
        occurrence_count: number
        workarounds: array<Workaround>
        fix_attempts: array<FixAttempt>
        metadata: object
      }
    relationships: Links to workarounds and fix attempts
    
  - name: Workaround
    storage: file-based (JSON)
    schema: |
      {
        id: string
        issue_id: string
        description: string
        commands: array<string>
        success_rate: number
        created_at: timestamp
        validated: boolean
      }
      
  - name: ComponentHealth
    storage: file-based (JSON)
    schema: |
      {
        component: string
        status: string (healthy|degraded|down)
        last_check: timestamp
        response_time_ms: number
        error_count: number (last hour)
        details: object
      }
```

### API Contract
```yaml
endpoints:
  - method: GET
    path: /api/v1/health
    purpose: Get overall system health status
    output_schema: |
      {
        status: "healthy|degraded|critical"
        components: ComponentHealth[]
        active_issues: number
        last_check: timestamp
      }
    sla:
      response_time: 100ms
      availability: 99.9%
      
  - method: GET
    path: /api/v1/issues
    purpose: Query active issues
    input_schema: |
      {
        component?: string
        severity?: string
        status?: string
      }
    output_schema: |
      {
        issues: CoreIssue[]
      }
      
  - method: GET
    path: /api/v1/issues/{id}/workarounds
    purpose: Get workarounds for specific issue
    output_schema: |
      {
        workarounds: Workaround[]
      }
      
  - method: POST
    path: /api/v1/issues/{id}/analyze
    purpose: Trigger Claude analysis of issue
    output_schema: |
      {
        analysis: string
        suggested_fix: string
        confidence: number
      }
```

### Event Interface
```yaml
published_events:
  - name: core.issue.detected
    payload: CoreIssue
    subscribers: Scenarios needing health awareness
    
  - name: core.issue.resolved
    payload: { issue_id: string, resolution: string }
    subscribers: Waiting scenarios can resume
    
consumed_events:
  # None - maintains zero-dependency principle
```

## 🖥️ CLI Interface Contract

### Command Structure
```yaml
cli_binary: core-debugger
install_script: cli/install.sh

required_commands:
  - name: status
    description: Show core component health status
    flags: [--json, --verbose, --component <name>]
    
  - name: help
    description: Display command help and usage
    flags: [--all, --command <name>]
    
  - name: version
    description: Show CLI and API version information
    flags: [--json]

custom_commands:
  - name: check-health
    description: Run health checks on all core components
    api_endpoint: /api/v1/health
    flags:
      - name: --component
        description: Check specific component only
    output: Component health status
    
  - name: list-issues
    description: List active core issues
    api_endpoint: /api/v1/issues
    arguments:
      - name: component
        type: string
        required: false
        description: Filter by component
    flags:
      - name: --severity
        description: Filter by severity level
      - name: --include-resolved
        description: Include resolved issues
    output: Table of issues
    
  - name: get-workaround
    description: Get workaround for specific issue
    api_endpoint: /api/v1/issues/{id}/workarounds
    arguments:
      - name: issue-id
        type: string
        required: true
        description: Issue ID or error signature
    output: Workaround commands and instructions
    
  - name: analyze-issue
    description: Use Claude to analyze an issue
    api_endpoint: /api/v1/issues/{id}/analyze
    arguments:
      - name: issue-id
        type: string
        required: true
        description: Issue to analyze
    output: Analysis and suggested fixes
```

## 🔄 Integration Requirements

### Upstream Dependencies
**What capabilities must exist before this can function?**
- **Vrooli CLI**: Must be installed and functional enough to run basic commands
- **Claude-code resource**: Required for intelligent issue analysis

### Downstream Enablement
**What future capabilities does this unlock?**
- **Automated recovery**: Scenarios can self-heal from common issues
- **Intelligent retries**: Scenarios check core health before operations
- **Centralized debugging**: All scenarios report issues to one place
- **Pattern learning**: System learns from recurring issues

### Cross-Scenario Interactions
```yaml
provides_to:
  - scenario: ALL
    capability: Core health status and workarounds
    interface: API/CLI
    
consumes_from:
  # None - maintains zero-dependency principle
```

## 🎨 Style and Branding Requirements

### UI/UX Style Guidelines
```yaml
style_profile:
  category: technical
  inspiration: GitHub Status Page, AWS Service Health Dashboard
  
  visual_style:
    color_scheme: dark with status colors (green/yellow/red)
    typography: technical/monospace for logs, clean sans for UI
    layout: dashboard with clear component sections
    animations: subtle status indicators
  
  personality:
    tone: technical but clear
    mood: focused and professional
    target_feeling: confidence in system health

style_references:
  technical:
    - system-monitor: "Matrix-style green terminal aesthetic"
    - github-status: "Clean professional status indicators"
```

### Target Audience Alignment
- **Primary Users**: Developers, DevOps engineers, Vrooli maintainers
- **User Expectations**: Clear, technical, actionable information
- **Accessibility**: High contrast mode for status indicators
- **Responsive Design**: Desktop priority, mobile readable

## 💰 Value Proposition

### Business Value
- **Primary Value**: Prevents core issues from blocking all development
- **Cost Savings**: 10-20 hours/week saved on debugging core issues
- **Market Differentiator**: Self-healing infrastructure capability

### Technical Value
- **Reusability Score**: 10/10 - Every scenario benefits from this
- **Complexity Reduction**: Makes core issues transparent and fixable
- **Innovation Enablement**: Allows focus on features instead of infrastructure

## 🧬 Evolution Path

### Version 1.0 (Current)
- Core health monitoring
- File-based issue tracking
- Status page UI
- Basic workaround system

### Version 2.0 (Planned)
- Pattern-based prediction
- Automated fix execution
- Multi-instance monitoring
- Performance profiling

### Long-term Vision
- Full self-healing capability
- Predictive maintenance
- Cross-project issue correlation

## 🔄 Scenario Lifecycle Integration

### Direct Scenario Deployment
```yaml
direct_execution:
  supported: true
  structure_compliance:
    - service.json with claude-code dependency only
    - File-based storage initialization
    - Health check endpoints
    - Minimal startup requirements
    
  deployment_targets:
    - local: Single process with file storage
    - kubernetes: StatefulSet with persistent volume
    - cloud: Lambda + S3 for serverless
    
  revenue_model:
    - type: internal-tool
    - pricing_tiers: N/A
    - trial_period: N/A
```

### Capability Discovery
```yaml
discovery:
  registry_entry:
    name: core-debugger
    category: infrastructure
    capabilities: [health-monitoring, issue-tracking, workaround-database]
    interfaces:
      - api: http://localhost:{port}/api/v1
      - cli: core-debugger
      - events: core.*
      
  metadata:
    description: Core Vrooli infrastructure health monitor and debugger
    keywords: [health, monitoring, debugging, core, infrastructure]
    dependencies: [claude-code]
    enhances: ALL
```

## 🚨 Risk Mitigation

### Technical Risks
| Risk | Probability | Impact | Mitigation |
|------|------------|--------|------------|
| File corruption | Low | High | Multiple backup files, atomic writes |
| False positives | Medium | Medium | Configurable thresholds, validation |
| Claude-code unavailable | Low | Medium | Cached workarounds, manual mode |

### Operational Risks
- **Single point of failure**: Designed to work even when other components fail
- **Infinite recursion**: Cannot debug itself - requires manual intervention
- **Storage growth**: Automatic log rotation and cleanup

## ✅ Validation Criteria

### Declarative Test Specification
```yaml
version: 1.0
scenario: core-debugger

structure:
  required_files:
    - .vrooli/service.json
    - PRD.md
    - api/main.go
    - api/go.mod
    - cli/core-debugger
    - cli/install.sh
    - data/issues/.gitkeep
    - data/health/.gitkeep
    - test/run-tests.sh
    
  required_dirs:
    - api
    - cli
    - data/issues
    - data/health
    - ui

resources:
  required: [claude-code]
  optional: []
  health_timeout: 30

tests:
  - name: "API health endpoint responds"
    type: http
    service: api
    endpoint: /api/v1/health
    method: GET
    expect:
      status: 200
      
  - name: "CLI status command works"
    type: exec
    command: ./cli/core-debugger status --json
    expect:
      exit_code: 0
      output_contains: ["status", "components"]
      
  - name: "Issue storage is writable"
    type: file
    path: data/issues/test.json
    operation: write
    content: '{"test": true}'
    expect:
      success: true
```

## 📝 Implementation Notes

### Design Decisions
**File-based storage over database**: Chosen for zero-dependency requirement
- Alternative considered: PostgreSQL for better querying
- Decision driver: Must work even when database resources fail
- Trade-offs: Less efficient queries for better reliability

**Single claude-code dependency**: Minimal dependencies for maximum reliability
- Alternative considered: Multiple AI resources for redundancy
- Decision driver: Keep failure modes minimal
- Trade-offs: Less redundancy for simpler architecture

### Known Limitations
- **Query performance**: File-based storage slower than database
  - Workaround: Indexing and caching frequently accessed data
  - Future fix: Optional database backend in v2.0

### Security Considerations
- **Data Protection**: No sensitive data in issue logs
- **Access Control**: Read-only by default, write requires auth
- **Audit Trail**: All changes logged with timestamps

## 🔗 References

### Documentation
- README.md - User guide and quick start
- docs/api.md - Full API specification
- docs/components.md - Monitored components detail

### Related PRDs
- app-monitor/PRD.md - Application monitoring (complementary)
- app-debugger/PRD.md - Application debugging (different scope)

---

**Last Updated**: 2025-01-03
**Status**: Draft
**Owner**: AI Agent
**Review Cycle**: Weekly validation against implementation