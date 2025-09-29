# Home Automation - Product Requirements Document (PRD)

## 🎯 Capability Definition

### Core Capability
**What permanent capability does this scenario add to Vrooli?**
This scenario creates a **self-evolving intelligent home automation system** that can control devices through Home Assistant, manage user permissions via scenario-authenticator, coordinate scheduling through calendar integration, and **autonomously generate new automations** using resource-claude-code. Unlike traditional static automation systems, this creates a home that literally learns and improves itself over time.

### Intelligence Amplification
**How does this capability make future agents smarter?**
- **Temporal Intelligence**: Calendar integration enables time-aware decision making across ALL scenarios
- **Permission-based Intelligence**: Profile system becomes a reusable pattern for multi-user, permission-aware scenarios  
- **Self-Code Generation**: Demonstrates how scenarios can use Claude Code to write their own logic, creating a template for self-improving systems
- **Physical World Interface**: Establishes patterns for scenarios to interact with real-world devices, unlocking IoT integration across the ecosystem

### Recursive Value
**What new scenarios become possible after this exists?**
1. **Commercial Building Automation**: Enterprise-grade systems that manage office buildings, warehouses, retail spaces
2. **Smart Healthcare Monitoring**: Medical device integration with AI-driven health analytics and emergency response
3. **Agricultural Automation**: Farm management with climate control, irrigation, and crop monitoring
4. **Energy Trading Platform**: Dynamic energy buying/selling based on usage patterns and grid conditions
5. **Security Command Center**: Integrated surveillance, access control, and threat response automation

## 📊 Success Metrics

### Functional Requirements
- **Must Have (P0)**
  - [x] Basic device control through Home Assistant CLI integration (API endpoints working, devices controllable)
  - [x] User authentication and profile-based permissions via scenario-authenticator (permission validation active)
  - [x] Calendar-driven automation scheduling and context awareness (fallback mode handles unavailable calendar)
  - [x] Natural language automation creation using resource-claude-code (generates YAML automations with safety validation)
  - [x] Real-time device status monitoring and health checks (health endpoint reporting all dependencies)
  
- **Should Have (P1)**
  - [ ] Scene management with intelligent suggestions based on patterns
  - [ ] Energy usage optimization with cost analysis
  - [ ] Mobile-responsive UI with real-time device controls
  - [ ] Automated conflict resolution between competing automations
  - [ ] Integration with shared n8n workflows for common patterns
  
- **Nice to Have (P2)**
  - [ ] Machine learning-based usage pattern recognition
  - [ ] Voice control integration through existing audio resources
  - [ ] Advanced scheduling with weather and calendar integration
  - [ ] Multi-home management for property managers

### Performance Criteria
| Metric | Target | Measurement Method |
|--------|--------|-------------------|
| Device Command Response | < 500ms for 95% of commands | Home Assistant API timing |
| UI Load Time | < 2s initial load | Web performance monitoring |
| Automation Generation | < 30s from description to deployed workflow | End-to-end timing |
| System Availability | > 99.5% uptime | Health check monitoring |
| Calendar Sync Latency | < 5s for schedule updates | Event-driven monitoring |

### Quality Gates
- [x] All P0 requirements implemented and tested (5 of 5 P0s fully working)
- [x] Integration tests pass with Home Assistant, authenticator, calendar, and Claude Code (all tests passing)
- [ ] Performance targets met under realistic device loads (not tested at scale)
- [x] Documentation complete (README, API docs, CLI help) (PRD and README comprehensive)
- [x] Scenario can be invoked by other agents via API/CLI (API on dynamic port, CLI working)
- [x] Permissions system prevents unauthorized device access (validation active on all endpoints)
- [x] Generated automations are syntactically correct and safe (full validation with safety checks)

## 🏗️ Technical Architecture

### Resource Dependencies
```yaml
required:
  - resource_name: home-assistant
    purpose: Device control and state management for IoT ecosystem
    integration_pattern: CLI commands via resource-home-assistant
    access_method: resource-home-assistant [device|scene|automation] [action]
    
  - resource_name: scenario-authenticator
    purpose: Multi-user authentication and permission management
    integration_pattern: API validation + JWT tokens
    access_method: http://localhost:3252/api/v1/auth/validate
    
  - resource_name: calendar
    purpose: Time-based automation and scheduling intelligence
    integration_pattern: API integration + event subscriptions
    access_method: http://localhost:3300/api/v1/events
    
  - resource_name: resource-claude-code
    purpose: Dynamic automation generation from natural language
    integration_pattern: CLI commands for code generation
    access_method: resource-claude-code generate automation --description="..."
    
  - resource_name: postgres
    purpose: Automation rules, user preferences, device history storage
    integration_pattern: Direct SQL via Go drivers
    access_method: Database instance: home_automation
    
optional:
  - resource_name: redis
    purpose: Real-time device state caching and event pub/sub
    integration_pattern: Direct Redis client
    fallback: In-memory caching with periodic persistence
    access_method: Redis client on localhost:6379
    
  - resource_name: n8n
    purpose: Complex workflow orchestration and external integrations
    integration_pattern: Shared workflows in initialization/n8n/
    fallback: Direct API calls without workflow orchestration
    access_method: Shared workflows via webhook triggers
```

### Resource Integration Standards
```yaml
integration_priorities:
  1_shared_workflows:
    - workflow: home-assistant-device-control.json
      location: initialization/n8n/
      purpose: Standardized device control with error handling and retries
    - workflow: calendar-automation-scheduler.json  
      location: initialization/n8n/
      purpose: Schedule-aware automation triggers and context switching
  
  2_resource_cli:
    - command: resource-home-assistant device list
      purpose: Device discovery and capability enumeration
    - command: scenario-authenticator user validate
      purpose: Permission validation for device access
    - command: calendar event list
      purpose: Context-aware scheduling queries
  
  3_direct_api:
    - justification: Real-time device state requires websocket/streaming
      endpoint: Home Assistant WebSocket API for live updates
```

### Data Models
```yaml
primary_entities:
  - name: HomeProfile
    storage: postgres
    schema: |
      {
        id: UUID,
        user_id: UUID,  // Links to scenario-authenticator
        name: string,
        permissions: {
          devices: string[],    // Allowed device IDs
          scenes: string[],     // Allowed scene IDs
          automation_create: boolean,
          admin_access: boolean
        },
        preferences: {
          default_scene: string,
          energy_optimization: boolean,
          notification_preferences: object
        }
      }
    relationships: Links to scenario-authenticator users
    
  - name: AutomationRule
    storage: postgres  
    schema: |
      {
        id: UUID,
        name: string,
        description: string,
        created_by: UUID,
        trigger_type: string,  // "schedule", "device", "calendar", "manual"
        trigger_config: object,
        actions: [
          {
            device_id: string,
            action: string,
            parameters: object
          }
        ],
        conditions: object,
        active: boolean,
        generated_by_ai: boolean,
        source_code: string  // For AI-generated automations
      }
    relationships: Belongs to HomeProfile, references Home Assistant devices
    
  - name: DeviceState
    storage: redis
    schema: |
      {
        device_id: string,
        state: object,
        last_updated: timestamp,
        attributes: object,
        available: boolean
      }
    relationships: Cached from Home Assistant real-time updates
```

### API Contract
```yaml
endpoints:
  - method: GET
    path: /api/v1/devices
    purpose: List available devices with current state and user permissions
    input_schema: |
      {
        profile_id?: UUID,
        filter?: "lights" | "sensors" | "switches" | "all"
      }
    output_schema: |
      {
        devices: [
          {
            id: string,
            name: string,
            type: string,
            state: object,
            capabilities: string[],
            user_can_control: boolean
          }
        ]
      }
    sla:
      response_time: 200ms
      availability: 99.5%
      
  - method: POST
    path: /api/v1/devices/{device_id}/control
    purpose: Control device with permission validation
    input_schema: |
      {
        action: string,
        parameters?: object,
        profile_id: UUID
      }
    output_schema: |
      {
        success: boolean,
        new_state: object,
        message?: string
      }
    sla:
      response_time: 500ms
      availability: 99.9%
      
  - method: POST
    path: /api/v1/automations/generate
    purpose: Generate new automation using Claude Code from natural language
    input_schema: |
      {
        description: string,
        profile_id: UUID,
        schedule_context?: string,
        target_devices?: string[]
      }
    output_schema: |
      {
        automation_id: UUID,
        generated_code: string,
        explanation: string,
        estimated_energy_impact: string,
        conflicts?: string[]
      }
    sla:
      response_time: 30000ms
      availability: 95%
```

### Event Interface
```yaml
published_events:
  - name: home.device.state_changed
    payload: { device_id: string, old_state: object, new_state: object }
    subscribers: [monitoring systems, energy analytics, notification services]
    
  - name: home.automation.executed
    payload: { automation_id: UUID, trigger_source: string, devices_affected: string[] }
    subscribers: [calendar for scheduling conflicts, energy tracker, audit logger]
    
  - name: home.profile.permission_denied
    payload: { user_id: UUID, attempted_action: string, device_id: string }
    subscribers: [security monitoring, user notification system]
    
consumed_events:
  - name: calendar.event.starting
    action: Switch to appropriate scene/context based on scheduled activity
  - name: calendar.event.ending  
    action: Return to default automation state or next scheduled context
  - name: auth.user.login
    action: Load user's home profile and activate personal automations
```

## 🖥️ CLI Interface Contract

### Command Structure
```yaml
cli_binary: home-automation
install_script: cli/install.sh

required_commands:
  - name: status
    description: Show home automation system status and device health
    flags: [--json, --verbose, --devices, --automations]
    
  - name: help
    description: Display command help and usage
    flags: [--all, --command <name>]
    
  - name: version
    description: Show CLI and API version information
    flags: [--json]

custom_commands:
  - name: devices
    description: List and control connected devices
    api_endpoint: /api/v1/devices
    arguments:
      - name: action
        type: string
        required: false
        description: Device action (list, status, control)
    flags:
      - name: --filter
        description: Filter devices by type (lights, sensors, switches)
      - name: --profile
        description: Use specific user profile for permissions
    output: Device list with states and control capabilities
    
  - name: scenes  
    description: Manage and activate scenes
    api_endpoint: /api/v1/scenes
    arguments:
      - name: scene_name
        type: string
        required: false
        description: Scene to activate or manage
    flags:
      - name: --activate
        description: Activate the specified scene
      - name: --create
        description: Create new scene from current device states
    output: Scene status and activation results
    
  - name: automations
    description: Create, manage, and monitor automations
    api_endpoint: /api/v1/automations
    arguments:
      - name: action
        type: string
        required: true
        description: Automation action (list, create, enable, disable, delete)
    flags:
      - name: --description
        description: Natural language description for AI generation
      - name: --schedule
        description: Schedule expression for time-based triggers
      - name: --profile
        description: User profile for permission context
    output: Automation status and management results
    
  - name: profiles
    description: Manage user profiles and permissions
    api_endpoint: /api/v1/profiles  
    arguments:
      - name: action
        type: string
        required: true
        description: Profile action (list, create, update, delete)
    flags:
      - name: --user-id
        description: User ID from scenario-authenticator
      - name: --permissions
        description: JSON string of permissions to set
    output: Profile information and permission settings
```

## 🔄 Integration Requirements

### Upstream Dependencies
**What capabilities must exist before this can function?**
- **Home Assistant Resource**: Must be functional with CLI interface for device discovery and control
- **Scenario Authenticator**: User management, JWT validation, and permission framework
- **Calendar Scenario**: Event scheduling, time-aware context switching
- **Resource Claude Code**: Natural language to automation code generation capabilities
- **Shared N8N Workflows**: Basic device control and scheduling patterns in initialization/n8n/

### Downstream Enablement
**What future capabilities does this unlock?**
- **Commercial IoT Integration**: Pattern for any scenario to integrate with physical devices
- **Multi-Tenant Automation**: Permission-based access control template for enterprise scenarios
- **Self-Improving Systems**: Template for scenarios that generate their own functionality via AI
- **Real-World Context Awareness**: Calendar + device state creates rich context for other scenarios

### Cross-Scenario Interactions
```yaml
provides_to:
  - scenario: energy-management
    capability: Real-time device power consumption data and control interface
    interface: API
    
  - scenario: security-system
    capability: Automated lighting, lock control, and presence simulation
    interface: API/Events
    
  - scenario: health-monitoring  
    capability: Environmental sensors, air quality, sleep optimization controls
    interface: API/Events
    
  - scenario: workflow-scheduler
    capability: Time-based automation scheduling with device state validation
    interface: API/Events
    
consumes_from:
  - scenario: scenario-authenticator
    capability: User authentication, permission management, profile storage
    fallback: Single-user mode with warning about security implications
    
  - scenario: calendar
    capability: Schedule-aware automation triggers, context switching
    fallback: Manual scheduling only, no intelligent context switching
    
  - scenario: notification-hub
    capability: Multi-channel alerts for device failures, security events
    fallback: Log-only notifications without user alerts
```

## 🎨 Style and Branding Requirements

### UI/UX Style Guidelines
```yaml
style_profile:
  category: professional
  inspiration: Smart home control apps like Apple HomeKit, Google Home, but with power-user features
  
  visual_style:
    color_scheme: dark  # Better for always-on displays
    typography: modern  # Clean, readable fonts for quick device status
    layout: dashboard   # Grid-based device tiles with status indicators
    animations: subtle  # Smooth transitions for state changes, no distracting effects
  
  personality:
    tone: professional  # Reliable, trustworthy automation system
    mood: focused       # Efficient control without unnecessary complexity
    target_feeling: "Confidence and control over home environment"

style_references:
  professional:
    - system-monitor: "Clean dashboard with real-time status indicators"
    - app-monitor: "Organized service management with health checks"
  
target_audience: "Smart home enthusiasts, property managers, tech-savvy homeowners"
accessibility: "WCAG 2.1 AA compliance for voice control and screen readers"
responsive_design: "Mobile-first for on-the-go control, tablet optimized for wall-mounted displays"

brand_consistency_rules:
  scenario_identity: "Reliable, intelligent home control - Apple HomeKit meets enterprise automation"
  vrooli_integration: "Professional tool within Vrooli ecosystem"
  professional_focus: "This is a high-value business capability - professional design required"
```

## 💰 Value Proposition

### Business Value
- **Primary Value**: Transforms any home into an intelligent, self-improving automation system
- **Revenue Potential**: $25K - $75K per high-end residential deployment, $100K+ for commercial buildings
- **Cost Savings**: 15-30% energy reduction through intelligent automation, reduced manual management overhead
- **Market Differentiator**: Only home automation system that can write its own rules and improve over time

### Technical Value
- **Reusability Score**: 9/10 - Patterns applicable to any IoT scenario, commercial automation, healthcare monitoring
- **Complexity Reduction**: Reduces 100+ lines of custom automation logic to single natural language descriptions
- **Innovation Enablement**: Establishes self-coding system template, physical world integration patterns, permission-based multi-user systems

## 🧬 Evolution Path

### Version 1.0 (Current)
- Core device control via Home Assistant CLI
- Basic user profiles and permissions
- Calendar integration for scheduling
- AI-generated automation creation
- Essential safety and conflict prevention

### Version 2.0 (Planned)
- Machine learning for usage pattern recognition
- Advanced energy optimization with utility integration
- Voice control through existing audio resources
- Multi-property management capabilities
- Weather integration for predictive automation

### Long-term Vision
- Autonomous home maintenance scheduling and vendor coordination
- Integration with smart city infrastructure and grid management
- Predictive health monitoring through environmental sensors
- Commercial building management with tenant-specific automation
- Energy trading and grid-balancing automation

## 🚨 Risk Mitigation

### Technical Risks
| Risk | Probability | Impact | Mitigation |
|------|------------|--------|------------|
| Home Assistant resource unavailability | Medium | High | Graceful degradation to manual controls, cached device states |
| Claude Code generation creates unsafe automations | Low | High | Validation sandbox, user approval for generated code |
| Permission system bypass | Low | High | Multiple validation layers, audit logging, fail-secure defaults |
| Device command latency | Medium | Medium | Command queuing, timeout handling, user feedback |

### Operational Risks
- **Safety-First Design**: All automations include emergency stop mechanisms and manual overrides
- **Permission Validation**: Multi-layer validation ensures users cannot control unauthorized devices  
- **Code Generation Safety**: AI-generated automations require user review and explicit approval
- **Device State Consistency**: Real-time synchronization with Home Assistant prevents state drift

## ✅ Validation Criteria

### Declarative Test Specification
```yaml
version: 1.0
scenario: home-automation

structure:
  required_files:
    - .vrooli/service.json
    - PRD.md
    - README.md
    - api/main.go
    - api/go.mod
    - cli/home-automation
    - cli/install.sh
    - initialization/postgres/schema.sql
    - initialization/n8n/home-assistant-device-control.json
    - scenario-test.yaml
    
  required_dirs:
    - api
    - cli  
    - ui
    - initialization/postgres
    - initialization/n8n

resources:
  required: [postgres, home-assistant, scenario-authenticator, calendar, resource-claude-code]
  optional: [redis, n8n]
  health_timeout: 90

tests:
  - name: "Home Assistant CLI is accessible"
    type: exec
    command: resource-home-assistant status
    expect:
      exit_code: 0
      
  - name: "Authentication service responds"
    type: http
    service: scenario-authenticator
    endpoint: /api/v1/health
    method: GET
    expect:
      status: 200
      
  - name: "Calendar service integration"  
    type: http
    service: calendar
    endpoint: /api/v1/health
    method: GET
    expect:
      status: 200
      
  - name: "API endpoint responds correctly"
    type: http
    service: api
    endpoint: /api/v1/devices
    method: GET
    headers:
      Authorization: "Bearer test-token"
    expect:
      status: 200
      body:
        devices: []
        
  - name: "CLI command executes"
    type: exec
    command: ./cli/home-automation status --json
    expect:
      exit_code: 0
      output_contains: ["healthy", "home-assistant"]
      
  - name: "Database schema initialized"
    type: sql
    service: postgres
    query: "SELECT COUNT(*) FROM information_schema.tables WHERE table_schema = 'public' AND table_name IN ('home_profiles', 'automation_rules')"
    expect:
      rows:
        - count: 2
```

## 📝 Implementation Notes

### Design Decisions
**Multi-Resource Integration Pattern**: Chosen layered integration approach using CLI → API → WebSocket for different use cases
- Alternative considered: Direct Home Assistant API integration only
- Decision driver: CLI provides stability and error handling, API enables real-time features
- Trade-offs: Additional complexity for better reliability and feature coverage

**AI-Generated Code Validation**: Implemented sandbox execution with user approval workflow
- Alternative considered: Fully automated code generation and deployment
- Decision driver: Safety requirements for home automation systems
- Trade-offs: User friction for maximum safety in device control

### Known Limitations
- **Home Assistant Dependency**: System requires functional Home Assistant installation with device discovery
  - Workaround: Mock device interface for development and testing
  - Future fix: Native device driver support for common IoT protocols

- **Calendar Integration Timing**: Initial implementation uses polling, may have 5-10s delay for schedule changes
  - Workaround: Manual automation triggers for time-critical scenarios  
  - Future fix: WebSocket integration with calendar service for real-time updates

### Security Considerations
- **Device Access Control**: Multi-layer permission validation with audit trail for all device commands
- **Code Generation Safety**: AI-generated automations run in validation sandbox before user approval
- **Authentication Integration**: JWT tokens with expiration, refresh handling via scenario-authenticator
- **Network Security**: Local-only device communication, no external API exposure without explicit configuration

## 🔗 References

### Documentation
- README.md - User-facing overview and quick start guide
- docs/api.md - Complete API specification with examples
- docs/cli.md - CLI command reference and usage patterns  
- docs/integration.md - Guide for other scenarios to integrate with home automation

### Related PRDs
- scenario-authenticator/PRD.md - Authentication and permission system
- calendar/PRD.md - Time-aware scheduling and context switching
- resource-claude-code documentation - AI code generation capabilities

### External Resources
- Home Assistant Developer Documentation - Device integration patterns
- IoT Security Best Practices - OWASP IoT Security Guidelines
- Smart Home Automation Patterns - Industry standards and protocols

---

**Last Updated**: 2025-09-24  
**Status**: Implementation Phase - P0 Requirements Complete
**Owner**: Claude Code AI Agent  
**Review Cycle**: Pre-implementation architectural review completed

## 🚀 Progress Updates

### 2025-09-28 Improvement Session
**Completion**: 85% → 95% (All P0 requirements fully functional, automation generation enhanced)

**Completed Items:**
- ✅ Enhanced automation generation with YAML code generation
- ✅ Integrated safety validation for all generated automations
- ✅ Fixed missing database tables (safety_rules, active_contexts)
- ✅ Added conflict detection for automation scheduling
- ✅ Implemented energy impact estimation for automations
- ✅ All tests passing comprehensively

**Key Improvements:**
- Automation generation now produces valid Home Assistant YAML
- Safety validator checks for dangerous patterns, device compatibility
- Permission system actively validates user access to devices
- Conflict detection prevents automation overlaps
- Database schema fully initialized with all required tables

**Technical Enhancements:**
- Added helper functions for automation code generation
- Improved validation response with actionable recommendations
- Enhanced error handling with exponential backoff for DB connections
- Comprehensive test coverage for all integrations

### 2025-09-24 Previous Session
**Completion**: 60% → 85% (P0 requirements achieved)

**Completed Items:**
- ✅ Fixed health endpoint routing (added both `/health` and `/api/v1/health`)
- ✅ Implemented automation generation endpoint (`/api/v1/automations/generate`)
- ✅ Fixed database initialization (home_automation database created)
- ✅ Added fallback modes for calendar integration
- ✅ All P0 requirements now functional with appropriate fallbacks

**Remaining Work (P1/P2):**
- Calendar service real-time integration
- Energy usage optimization features
- Machine learning pattern recognition
- Performance testing at scale
- Voice control integration