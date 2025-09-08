# Product Requirements Document (PRD)

## 🎯 Capability Definition

### Core Capability
**What permanent capability does this scenario add to Vrooli?**
Vrooli-orchestrator adds **contextual intelligence adaptation** - the ability to automatically configure Vrooli's entire environment (resources, scenarios, and UI) for specific user contexts, use cases, and organizational needs. Instead of a one-size-fits-all startup, Vrooli becomes infinitely customizable through startup profiles that define exactly what runs and how.

### Intelligence Amplification
**How does this capability make future agents smarter?**
- **Dynamic Environment Optimization**: Agents can programmatically switch Vrooli configurations based on current tasks (e.g., load development tools for coding, creative tools for content creation)
- **Context-Aware Resource Management**: Smart resource allocation based on workload patterns and user preferences
- **Cross-Scenario Orchestration**: Other scenarios can preemptively load dependencies before showing users content
- **Adaptive Learning**: System learns usage patterns and optimizes profile recommendations over time

### Recursive Value
**What new scenarios become possible after this exists?**
- **scenario-surfer**: Can intelligently preload relevant scenarios before displaying them
- **morning-vision-walk**: Can activate productivity profiles automatically for work sessions
- **deployment-manager**: Can create customer-specific minimal profiles for app delivery
- **usage-analytics**: Can track which profiles are most effective for different user types
- **smart-workspace**: Can automatically switch profiles based on calendar events or project contexts

## 📊 Success Metrics

### Functional Requirements
- **Must Have (P0)**
  - [ ] Create, edit, and delete startup profiles via CLI and UI
  - [ ] Activate profiles that start/stop specific resources and scenarios
  - [ ] Profile persistence in local configuration files
  - [ ] CLI interface: `vrooli-orchestrator activate <profile-name>`
  - [ ] Basic profile metadata (name, description, target_audience)
  - [ ] Integration with main `vrooli` CLI for resource/scenario lifecycle
  
- **Should Have (P1)**
  - [ ] Profile templates for common use cases (developer, business, household)
  - [ ] Auto-open browser tabs for configured dashboards
  - [ ] Idle shutdown timers for resource management
  - [ ] Environment variable overrides per profile
  - [ ] Profile validation and dependency checking
  - [ ] Web UI dashboard for visual profile management
  
- **Nice to Have (P2)**
  - [ ] Usage analytics and profile optimization recommendations
  - [ ] Profile sharing and versioning
  - [ ] Conditional activation based on time/context
  - [ ] Integration with calendar for automatic profile switching
  - [ ] Resource usage monitoring and optimization

### Performance Criteria
| Metric | Target | Measurement Method |
|--------|--------|-------------------|
| Profile Activation Time | < 30s for typical profile | CLI timing measurement |
| Profile Switch Time | < 15s between profiles | System monitoring |
| Configuration Load Time | < 2s for profile list/display | API monitoring |
| Resource Cleanup Time | < 10s for profile deactivation | Process monitoring |

### Quality Gates
- [ ] All P0 requirements implemented and tested
- [ ] Can manage profiles for 5+ different user contexts
- [ ] Graceful handling of resource conflicts and failures
- [ ] Complete CLI-API parity for all operations
- [ ] Profile activation/deactivation is idempotent

## 🏗️ Technical Architecture

### Resource Dependencies
```yaml
required:
  - resource_name: postgres
    purpose: Store profile configurations, activation history, and analytics
    integration_pattern: Shared workflow + CLI
    access_method: resource-postgres CLI and n8n workflow
    
optional:
  - resource_name: browserless
    purpose: Auto-open browser tabs for dashboard URLs
    fallback: Manual URL display to user
    access_method: resource-browserless CLI
```

### Resource Integration Standards
```yaml
integration_priorities:
  1_shared_workflows:
    - workflow: config-manager.json
      location: initialization/n8n/
      purpose: CRUD operations for profile configurations
  
  2_resource_cli:
    - command: resource-postgres query/update
      purpose: Profile persistence and retrieval
    - command: vrooli resource start/stop
      purpose: Resource lifecycle management
    - command: vrooli scenario run/stop
      purpose: Scenario lifecycle management
  
  3_direct_api:
    - justification: Need real-time status of resources/scenarios
      endpoint: Various vrooli internal APIs
```

### Data Models
```yaml
primary_entities:
  - name: Profile
    storage: postgres
    schema: |
      {
        id: UUID,
        name: string,
        display_name: string,
        description: string,
        metadata: {
          target_audience: string,
          resource_footprint: "low|medium|high",
          use_case: string,
          created_by: string,
          created_at: timestamp
        },
        resources: [string], // resource names to start
        scenarios: [string], // scenario names to run
        auto_browser: [string], // URLs to open
        environment_vars: object,
        idle_shutdown_minutes: integer,
        dependencies: [string], // other profiles this depends on
        status: "active|inactive|error"
      }
    relationships: Profile can depend on other profiles for composition

  - name: ProfileActivation
    storage: postgres
    schema: |
      {
        id: UUID,
        profile_id: UUID,
        activated_at: timestamp,
        deactivated_at: timestamp,
        user_context: string,
        success: boolean,
        error_details: string
      }
    relationships: Many activations per profile for analytics
```

### API Contract
```yaml
endpoints:
  - method: GET
    path: /api/v1/profiles
    purpose: List all available profiles
    output_schema: |
      {
        profiles: [Profile]
      }
    sla:
      response_time: 200ms
      availability: 99.9%
      
  - method: POST
    path: /api/v1/profiles
    purpose: Create new startup profile
    input_schema: |
      {
        name: string,
        display_name: string,
        description: string,
        resources: [string],
        scenarios: [string],
        auto_browser: [string],
        metadata: object
      }
    sla:
      response_time: 500ms
      availability: 99.9%
      
  - method: POST
    path: /api/v1/profiles/{id}/activate
    purpose: Activate a specific profile (start resources/scenarios)
    output_schema: |
      {
        profile_id: string,
        activation_id: string,
        status: "activating|active|error",
        started_resources: [string],
        started_scenarios: [string],
        opened_urls: [string]
      }
    sla:
      response_time: 30000ms  # 30s for full activation
      availability: 99.5%
```

### Event Interface
```yaml
published_events:
  - name: orchestrator.profile.activated
    payload: { profile_id, activation_id, resources, scenarios }
    subscribers: [usage-analytics, morning-vision-walk]
    
  - name: orchestrator.profile.deactivated
    payload: { profile_id, deactivation_reason }
    subscribers: [cleanup-manager, usage-analytics]
    
consumed_events:
  - name: scenario.*.started
    action: Update profile activation status
  - name: resource.*.health_changed
    action: Update profile health monitoring
```

## 🖥️ CLI Interface Contract

### Command Structure
```yaml
cli_binary: vrooli-orchestrator
install_script: cli/install.sh

required_commands:
  - name: status
    description: Show current active profile and system status
    flags: [--json, --verbose]
    
  - name: help
    description: Display command help and usage
    flags: [--all, --command <name>]
    
  - name: version
    description: Show CLI and API version information
    flags: [--json]

custom_commands:
  - name: list-profiles
    description: Show all available startup profiles
    api_endpoint: /api/v1/profiles
    flags:
      - name: --json
        description: Output in JSON format
      - name: --category
        description: Filter by target audience
    output: Table or JSON list of profiles
    
  - name: create-profile
    description: Create new startup profile
    api_endpoint: /api/v1/profiles
    arguments:
      - name: name
        type: string
        required: true
        description: Profile name (kebab-case)
    flags:
      - name: --resources
        description: Comma-separated list of resources
      - name: --scenarios  
        description: Comma-separated list of scenarios
      - name: --auto-browser
        description: Comma-separated list of URLs
    output: Created profile details
    
  - name: activate
    description: Activate a startup profile
    api_endpoint: /api/v1/profiles/{name}/activate
    arguments:
      - name: profile_name
        type: string
        required: true
        description: Name of profile to activate
    flags:
      - name: --force
        description: Force activation even if conflicts exist
    output: Activation status and started services
    
  - name: deactivate
    description: Deactivate current profile and stop services
    api_endpoint: /api/v1/profiles/current/deactivate
    output: Deactivation status
    
  - name: edit-profile
    description: Edit existing profile configuration
    api_endpoint: /api/v1/profiles/{name}
    arguments:
      - name: profile_name
        type: string
        required: true
        description: Profile to edit
    output: Interactive editor or JSON update confirmation
```

## 🔄 Integration Requirements

### Upstream Dependencies
**What capabilities must exist before this can function?**
- **Vrooli Core CLI**: Must be able to call `vrooli resource start/stop` and `vrooli scenario run/stop`
- **Postgres Resource**: For persistent profile storage and activation history
- **Basic Resource Management**: Resources must expose health/status endpoints

### Downstream Enablement
**What future capabilities does this unlock?**
- **Intelligent Scenario Surfing**: scenario-surfer can preload before showing content
- **Context-Aware Morning Briefings**: morning-vision-walk can switch to productivity mode
- **Customer-Specific Deployments**: deployment-manager can create minimal customer profiles
- **Automated Workflow Optimization**: System can learn and optimize profile effectiveness

### Cross-Scenario Interactions
```yaml
provides_to:
  - scenario: scenario-surfer
    capability: Preload scenarios before display
    interface: CLI command `vrooli-orchestrator activate gaming-profile`
    
  - scenario: morning-vision-walk  
    capability: Context-aware environment switching
    interface: API call to activate productivity profile
    
  - scenario: deployment-manager
    capability: Customer-specific minimal configurations
    interface: Profile templates for deployment
    
consumes_from:
  - scenario: usage-analytics
    capability: Profile effectiveness data
    fallback: Basic time-based analytics only
```

## 🎨 Style and Branding Requirements

### UI/UX Style Guidelines
```yaml
style_profile:
  category: professional
  inspiration: Mission control dashboard + modern dev tools
  
  visual_style:
    color_scheme: dark
    typography: modern
    layout: dashboard
    animations: subtle
  
  personality:
    tone: technical
    mood: focused
    target_feeling: Control and confidence over complex systems
```

### Target Audience Alignment
- **Primary Users**: System administrators, power users, different household/business types
- **User Expectations**: Clean, efficient interface that doesn't get in the way
- **Accessibility**: WCAG AA compliance, keyboard navigation essential
- **Responsive Design**: Desktop-first (command center feel), mobile for monitoring

### Brand Consistency Rules
- **Professional Design**: This is infrastructure - must feel reliable and trustworthy
- **Technical Aesthetic**: Similar to system-monitor and app-debugger 
- **Clear Information Hierarchy**: Status, actions, and configuration clearly separated

## 💰 Value Proposition

### Business Value
- **Primary Value**: Makes Vrooli adaptable to any customer context without configuration complexity
- **Revenue Potential**: $15K - $40K per deployment (enables custom customer experiences)
- **Cost Savings**: Eliminates need for manual environment setup and reduces resource waste
- **Market Differentiator**: First AI platform that automatically adapts to user context

### Technical Value
- **Reusability Score**: 9/10 - Every scenario can benefit from contextual environment control
- **Complexity Reduction**: Transforms complex startup procedures into one-command activation
- **Innovation Enablement**: Unlocks context-aware computing patterns for all scenarios

## 🧬 Evolution Path

### Version 1.0 (Current)
- Profile CRUD operations
- Basic resource/scenario lifecycle management
- CLI and web UI interfaces
- Manual profile activation

### Version 2.0 (Planned)
- Automatic profile recommendations based on usage patterns
- Calendar integration for time-based activation
- Profile composition (profiles that extend other profiles)
- Resource usage optimization and idle management

### Long-term Vision
- AI-driven profile optimization based on productivity metrics
- Predictive profile switching based on user behavior
- Integration with external systems (calendar, project management, etc.)
- Self-healing profiles that adapt to resource availability

## 🔄 Scenario Lifecycle Integration

### Direct Scenario Deployment
```yaml
direct_execution:
  supported: true
  structure_compliance:
    - .vrooli/service.json with orchestration metadata
    - Profile management API and UI
    - CLI for profile operations
    - Health monitoring for activated profiles
    
  deployment_targets:
    - local: Full orchestration capabilities
    - kubernetes: Distributed profile management
    - cloud: Multi-tenant profile isolation
    
  revenue_model:
    - type: subscription
    - pricing_tiers: Based on number of profiles and automation features
    - trial_period: 30 days
```

### Capability Discovery
```yaml
discovery:
  registry_entry:
    name: vrooli-orchestrator
    category: automation
    capabilities: [profile-management, environment-orchestration, context-switching]
    interfaces:
      - api: http://localhost:${API_PORT}/api/v1
      - cli: vrooli-orchestrator
      - events: orchestrator.*
      
  metadata:
    description: Contextual environment orchestration and profile management
    keywords: [orchestration, profiles, automation, context, environment]
    dependencies: [postgres]
    enhances: [scenario-surfer, morning-vision-walk, deployment-manager]
```

## 🚨 Risk Mitigation

### Technical Risks
| Risk | Probability | Impact | Mitigation |
|------|------------|--------|------------|
| Resource conflicts during activation | Medium | High | Pre-activation validation and conflict resolution |
| Profile corruption/inconsistency | Low | Medium | Schema validation and backup/restore |
| Activation timeout/failure | Medium | Medium | Graceful fallbacks and partial activation support |

### Operational Risks
- **Profile Drift**: Regular validation ensures profiles match their intended configurations
- **Resource Exhaustion**: Built-in resource monitoring prevents overallocation
- **User Confusion**: Clear profile naming and description standards prevent misuse

## ✅ Validation Criteria

### Declarative Test Specification
```yaml
version: 1.0
scenario: vrooli-orchestrator

structure:
  required_files:
    - .vrooli/service.json
    - PRD.md
    - api/main.go
    - api/go.mod
    - cli/vrooli-orchestrator
    - cli/install.sh
    - initialization/n8n/config-manager.json
    - initialization/postgres/schema.sql
    - scenario-test.yaml
    - ui/index.html
    
  required_dirs:
    - api
    - cli
    - initialization
    - initialization/n8n
    - initialization/postgres
    - ui
    - data

resources:
  required: [postgres]
  optional: [browserless]
  health_timeout: 60

tests:
  - name: "Postgres is accessible"
    type: http
    service: postgres
    endpoint: /health
    method: GET
    expect:
      status: 200
      
  - name: "API profiles endpoint responds"
    type: http
    service: api
    endpoint: /api/v1/profiles
    method: GET
    expect:
      status: 200
      body:
        profiles: []
        
  - name: "CLI list-profiles command works"
    type: exec
    command: ./cli/vrooli-orchestrator list-profiles --json
    expect:
      exit_code: 0
      output_contains: ["profiles"]
      
  - name: "Profile creation works"
    type: http
    service: api
    endpoint: /api/v1/profiles
    method: POST
    body:
      name: "test-profile"
      display_name: "Test Profile"
      resources: ["postgres"]
      scenarios: []
    expect:
      status: 201
      body:
        id: "string"
        name: "test-profile"
```

## 📝 Implementation Notes

### Design Decisions
**Profile Storage**: Postgres chosen over file-based storage for ACID properties and analytics capabilities
- Alternative considered: JSON files in ~/.vrooli/profiles/
- Decision driver: Need for concurrent access and historical tracking
- Trade-offs: Slight complexity increase for better reliability and analytics

**CLI-First Design**: CLI commands mirror API endpoints exactly  
- Alternative considered: Web UI as primary interface
- Decision driver: Power users prefer CLI for automation and scripting
- Trade-offs: Web UI becomes secondary but still fully functional

### Known Limitations
- **Single Active Profile**: v1.0 only supports one active profile at a time
  - Workaround: Manual resource/scenario management for complex setups
  - Future fix: v2.0 will support profile composition and layering

### Security Considerations
- **Profile Access Control**: Profiles stored per-user, no cross-user access
- **Resource Authorization**: Inherits existing vrooli resource permissions
- **Audit Trail**: All profile activations logged for security monitoring

## 🔗 References

### Documentation
- README.md - User-facing overview and quick start
- docs/api.md - Complete API specification  
- docs/cli.md - CLI command reference
- docs/profiles.md - Profile configuration guide

### Related PRDs
- scenario-surfer: Will use orchestrator for preloading scenarios
- morning-vision-walk: Will use orchestrator for context switching
- deployment-manager: Will use orchestrator for customer configurations

### External Resources
- Docker Compose documentation for understanding service orchestration patterns
- Kubernetes documentation for distributed orchestration concepts
- System service management best practices

---

**Last Updated**: 2025-09-08
**Status**: Draft  
**Owner**: AI Agent (Claude)  
**Review Cycle**: Validated against implementation weekly