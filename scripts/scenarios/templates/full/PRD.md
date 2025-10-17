# Product Requirements Document (PRD)

## 🎯 Capability Definition

### Core Capability
**What permanent capability does this scenario add to Vrooli?**
[Define the fundamental capability this scenario provides that will persist forever in the system. Be specific about what problem-solving ability this adds.]

### Intelligence Amplification
**How does this capability make future agents smarter?**
[Describe how this capability compounds with existing capabilities to enable more complex problem-solving.]

### Recursive Value
**What new scenarios become possible after this exists?**
[List 3-5 concrete examples of future scenarios that can build on this capability.]

## 📊 Success Metrics

### Functional Requirements
- **Must Have (P0)**
  - [ ] [Core requirement that defines minimum viable capability]
  - [ ] [Essential integration with shared resources]
  - [ ] [Critical data persistence requirement]
  
- **Should Have (P1)**
  - [ ] [Enhancement that significantly improves capability]
  - [ ] [Additional resource integration]
  - [ ] [Performance optimization]
  
- **Nice to Have (P2)**
  - [ ] [Future enhancement]
  - [ ] [Advanced feature]

### Performance Criteria
| Metric | Target | Measurement Method |
|--------|--------|-------------------|
| Response Time | < [X]ms for 95% of requests | API monitoring |
| Throughput | [Y] operations/second | Load testing |
| Accuracy | > [Z]% for [specific task] | Validation suite |
| Resource Usage | < [N]GB memory, < [M]% CPU | System monitoring |

### Quality Gates
- [ ] All P0 requirements implemented and tested
- [ ] Integration tests pass with all required resources
- [ ] Performance targets met under load
- [ ] Documentation complete (README, API docs, CLI help)
- [ ] Scenario can be invoked by other agents via API/CLI

## 🏗️ Technical Architecture

### Resource Dependencies
```yaml
required:
  - resource_name: [e.g., postgres]
    purpose: [Why this resource is essential]
    integration_pattern: [How it's used - CLI, API, or shared workflow]
    access_method: [CLI command | API endpoint | Shared workflow name]
    
optional:
  - resource_name: [e.g., redis]
    purpose: [Enhancement this enables]
    fallback: [What happens if unavailable]
    access_method: [How this resource is accessed]
```

### Resource Integration Standards
```yaml
# Priority order for resource access (MUST follow this hierarchy):
integration_priorities:
  1_shared_workflows:     # FIRST: Use existing shared n8n workflows
    - workflow: [e.g., ollama.json]
      location: initialization/automation/n8n/
      purpose: [What capability this provides]
  
  2_resource_cli:        # SECOND: Use resource CLI commands
    - command: resource-[name] [action]
      purpose: [What this accomplishes]
  
  3_direct_api:          # LAST: Direct API only when necessary
    - justification: [Why CLI/workflow won't work]
      endpoint: [API endpoint used]

# Shared workflow guidelines:
shared_workflow_criteria:
  - Must be truly reusable across multiple scenarios
  - Place in initialization/automation/n8n/ if generic
  - Document reusability in workflow description
  - List all scenarios that will use this workflow
```

### Data Models
```yaml
# Core data structures that define the capability
primary_entities:
  - name: [Entity]
    storage: [postgres/qdrant/minio]
    schema: |
      {
        id: UUID
        # ... key fields
      }
    relationships: [How it connects to other entities]
```

### API Contract
```yaml
# Defines how other scenarios/agents can use this capability
endpoints:
  - method: POST
    path: /api/v1/[capability]/[action]
    purpose: [What this enables other systems to do]
    input_schema: |
      {
        # Required fields for invoking capability
      }
    output_schema: |
      {
        # What calling systems can expect back
      }
    sla:
      response_time: [X]ms
      availability: [Y]%
```

### Event Interface
```yaml
# Events this capability publishes for others to consume
published_events:
  - name: [capability].[action].completed
    payload: [Data structure]
    subscribers: [Who might care]
    
consumed_events:
  - name: [other_capability].[event]
    action: [What this scenario does in response]
```

## 🖥️ CLI Interface Contract

### Command Structure
```yaml
# Primary CLI executable name and pattern
cli_binary: [scenario-name]  # Must match scenario name
install_script: cli/install.sh  # Required for PATH integration

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

# Scenario-specific commands (must mirror API endpoints):
custom_commands:
  - name: [action]
    description: [What this command does]
    api_endpoint: /api/v1/[capability]/[action]
    arguments:
      - name: [arg_name]
        type: [string|int|bool]
        required: [true|false]
        description: [What this argument controls]
    flags:
      - name: --[flag]
        description: [What this flag does]
    output: [Expected output format]
```

### CLI-API Parity Requirements
- **Coverage**: Every API endpoint MUST have a corresponding CLI command
- **Naming**: CLI commands should use kebab-case versions of API endpoints
- **Arguments**: CLI arguments must map directly to API parameters
- **Output**: Support both human-readable and JSON output (--json flag)
- **Authentication**: Inherit from API configuration or environment variables

### Implementation Standards
```yaml
# CLI must be a thin wrapper pattern:
implementation_requirements:
  - architecture: Thin wrapper over lib/ functions
  - language: [Go preferred for consistency with APIs]
  - dependencies: Minimal - reuse API client libraries
  - error_handling: Consistent exit codes (0=success, 1=error)
  - configuration: 
      - Read from ~/.vrooli/[scenario]/config.yaml
      - Environment variables override config
      - Command flags override everything
  
# Installation requirements:
installation:
  - install_script: Must create symlink in ~/.vrooli/bin/
  - path_update: Must add ~/.vrooli/bin to PATH if not present
  - permissions: Executable permissions (755) required
  - documentation: Generated via --help must be comprehensive
```

## 🔄 Integration Requirements

### Upstream Dependencies
**What capabilities must exist before this can function?**
- **[Capability/Resource]**: [Why it's needed, what it provides]
- **[Shared Workflow]**: [Which n8n workflows this depends on]

### Downstream Enablement
**What future capabilities does this unlock?**
- **[Future Capability]**: [How this scenario enables it]
- **[Enhancement Pattern]**: [Reusable pattern this establishes]

### Cross-Scenario Interactions
```yaml
# How this scenario enhances other scenarios
provides_to:
  - scenario: [scenario_name]
    capability: [What enhancement this provides]
    interface: [API/CLI/Event]
    
consumes_from:
  - scenario: [scenario_name]
    capability: [What it uses]
    fallback: [Behavior if unavailable]
```

## 🎨 Style and Branding Requirements

### UI/UX Style Guidelines
```yaml
# Define the visual and experiential personality of this scenario
style_profile:
  category: [professional|creative|playful|technical|minimalist]
  inspiration: [Reference existing scenario or external product]
  
  # Visual characteristics:
  visual_style:
    color_scheme: [dark|light|high-contrast|custom]
    typography: [modern|retro|technical|playful]
    layout: [dense|spacious|dashboard|single-page]
    animations: [none|subtle|playful|extensive]
  
  # Personality traits:
  personality:
    tone: [serious|friendly|quirky|technical]
    mood: [energetic|calm|focused|fun]
    target_feeling: [What emotion users should feel]

# Style examples from existing scenarios:
style_references:
  professional: 
    - research-assistant: "Clean, professional, information-dense"
    - product-manager: "Modern SaaS dashboard aesthetic"
  creative:
    - study-buddy: "Cute, lo-fi, cozy study space vibe"
    - notes: "Minimalist, distraction-free writing"
  playful:
    - retro-game-launcher: "80s arcade cabinet aesthetic"
    - picker-wheel: "Carnival game show energy"
  technical:
    - system-monitor: "Matrix-style green terminal aesthetic"
    - agent-dashboard: "NASA mission control vibes"
```

### Target Audience Alignment
- **Primary Users**: [Who will use this most]
- **User Expectations**: [What style they expect based on use case]
- **Accessibility**: [WCAG compliance level, special considerations]
- **Responsive Design**: [Mobile, tablet, desktop priorities]

### Brand Consistency Rules
- **Scenario Identity**: Must feel unique and memorable
- **Vrooli Integration**: Should feel part of the Vrooli ecosystem
- **Professional vs Fun**: [Decision criteria based on business value]
  - If business/enterprise tool → Professional design
  - If consumer/creative tool → Unique personality encouraged
  - If technical tool → Function over form, but still polished

## 💰 Value Proposition

### Business Value
- **Primary Value**: [Core business problem solved]
- **Revenue Potential**: $[X]K - $[Y]K per deployment
- **Cost Savings**: [Time/resource savings quantified]
- **Market Differentiator**: [What makes this unique]

### Technical Value
- **Reusability Score**: [How many other scenarios can leverage this]
- **Complexity Reduction**: [What complex tasks become simple]
- **Innovation Enablement**: [New possibilities this creates]

## 🧬 Evolution Path

### Version 1.0 (Current)
- Core capability implementation
- Basic resource integration
- Essential API/CLI interface

### Version 2.0 (Planned)
- [Enhanced capability based on learnings]
- [Additional resource integrations]
- [Performance optimizations]

### Long-term Vision
- [How this capability evolves with the system]
- [Ultimate potential when combined with future capabilities]

## 🔄 Scenario Lifecycle Integration

### Direct Scenario Deployment
```yaml
# Requirements for direct scenario execution:
direct_execution:
  supported: true
  structure_compliance:
    - service.json with complete metadata
    - All required initialization files
    - Deployment scripts (startup.sh, monitor.sh)
    - Health check endpoints
    
  deployment_targets:
    - local: Docker Compose based
    - kubernetes: Helm chart generation
    - cloud: AWS/GCP/Azure templates
    
  revenue_model:
    - type: [subscription|one-time|usage-based]
    - pricing_tiers: [Define if applicable]
    - trial_period: [Days if applicable]
```

### Capability Discovery
```yaml
# How other scenarios/agents discover and use this capability:
discovery:
  registry_entry:
    name: [scenario-name]
    category: [research|automation|analysis|generation]
    capabilities: [List of specific capabilities]
    interfaces:
      - api: [Base URL pattern]
      - cli: [Command name]
      - events: [Event namespace]
      
  metadata:
    description: [One-line description for discovery]
    keywords: [searchable terms]
    dependencies: [required scenarios]
    enhances: [scenarios this improves]
```

### Version Management
```yaml
# Compatibility and upgrade paths:
versioning:
  current: 1.0.0
  minimum_compatible: 1.0.0  # Oldest version that works with current
  
  breaking_changes:
    - version: [When breaking change occurred]
      description: [What changed]
      migration: [How to migrate]
      
  deprecations:
    - feature: [Deprecated feature]
      removal_version: [When it will be removed]
      alternative: [What to use instead]
```

## 🚨 Risk Mitigation

### Technical Risks
| Risk | Probability | Impact | Mitigation |
|------|------------|--------|------------|
| [Resource unavailability] | Medium | High | Graceful degradation |
| [Performance degradation] | Low | Medium | Caching, optimization |
| [Data inconsistency] | Low | High | Transaction boundaries |

### Operational Risks
- **Drift Prevention**: PRD serves as single source of truth, validated by scenario-test.yaml
- **Version Compatibility**: Semantic versioning with clear breaking change documentation
- **Resource Conflicts**: Resource allocation managed through service.json priorities
- **Style Drift**: UI components must pass style guide validation
- **CLI Consistency**: Automated testing ensures CLI-API parity

## ✅ Validation Criteria

### Declarative Test Specification
```yaml
# REQUIRED: scenario-test.yaml in scenario root
version: 1.0
scenario: [scenario-name]

# Structure validation - files and directories that MUST exist:
structure:
  required_files:
    - .vrooli/service.json
    - PRD.md
    - api/main.go        # Or appropriate API entry point
    - api/go.mod         # Or appropriate dependency file
    - cli/[scenario-name]
    - cli/install.sh
    - initialization/storage/postgres/schema.sql  # If using postgres
    - scenario-test.yaml
    
  required_dirs:
    - api
    - cli
    - initialization
    - initialization/automation  # If using n8n/windmill
    - initialization/storage     # If using databases

# Resource validation:
resources:
  required: [exact list of required resources]
  optional: [list of optional resources]
  health_timeout: 60  # Seconds to wait for resources to be healthy

# Declarative tests:
tests:
  # Resource health checks:
  - name: "[Resource] is accessible"
    type: http
    service: [resource_name]
    endpoint: /health
    method: GET
    expect:
      status: 200
      
  # API endpoint tests:
  - name: "API endpoint [name] responds correctly"
    type: http
    service: api
    endpoint: /api/v1/[endpoint]
    method: POST
    body:
      test: data
    expect:
      status: 201
      body:
        success: true
        
  # CLI command tests:
  - name: "CLI command [name] executes"
    type: exec
    command: ./cli/[scenario-name] status --json
    expect:
      exit_code: 0
      output_contains: ["healthy"]
      
  # Database tests:
  - name: "Database schema is initialized"
    type: sql
    service: postgres
    query: "SELECT COUNT(*) FROM information_schema.tables WHERE table_schema = 'public'"
    expect:
      rows: 
        - count: [expected_table_count]
        
  # Workflow tests (if using n8n):
  - name: "Research workflow is active"
    type: n8n
    workflow: research-orchestrator
    expect:
      active: true
      node_count: [expected_nodes]
```

### Test Execution Gates
```bash
# All tests must pass via:
./test.sh --scenario [scenario_name] --validation complete

# Individual test categories:
./test.sh --structure    # Verify file/directory structure
./test.sh --resources    # Check resource health
./test.sh --integration  # Run integration tests
./test.sh --performance  # Validate performance targets
```

### Performance Validation
- [ ] API response times meet SLA targets
- [ ] Resource usage within defined limits
- [ ] Throughput meets minimum requirements
- [ ] No memory leaks detected over 24-hour test

### Integration Validation
- [ ] Discoverable via resource registry
- [ ] All API endpoints documented and functional
- [ ] All CLI commands executable with --help
- [ ] Shared workflows properly registered
- [ ] Events published/consumed correctly

### Capability Verification
- [ ] Solves the defined problem completely
- [ ] Integrates with upstream dependencies
- [ ] Enables downstream capabilities
- [ ] Maintains data consistency
- [ ] Style matches target audience expectations

## 📝 Implementation Notes

### Design Decisions
**[Decision Point]**: [Chosen approach and rationale]
- Alternative considered: [What else was evaluated]
- Decision driver: [Why this approach won]
- Trade-offs: [What was sacrificed for what benefit]

### Known Limitations
- **[Limitation]**: [Description and impact]
  - Workaround: [How to work within this constraint]
  - Future fix: [When/how this will be addressed]

### Security Considerations
- **Data Protection**: [How sensitive data is handled]
- **Access Control**: [Who can use this capability]
- **Audit Trail**: [What actions are logged]

## 🔗 References

### Documentation
- README.md - User-facing overview
- docs/api.md - API specification
- docs/cli.md - CLI documentation
- docs/architecture.md - Technical deep-dive

### Related PRDs
- [Link to dependent scenario PRDs]
- [Link to enhanced scenario PRDs]

### External Resources
- [Relevant technical documentation]
- [Industry standards referenced]
- [Research papers/articles that informed design]

---

**Last Updated**: [Date]  
**Status**: [Draft | In Review | Approved | Not Tested | Testing | Validated]  
**Owner**: [AI Agent or Human maintainer]  
**Review Cycle**: [How often this PRD is validated against implementation]