# Product Requirements Document (PRD) - Secrets Manager

## 🎯 Capability Definition

### Core Capability
**What permanent capability does this scenario add to Vrooli?**
Provides automated discovery, validation, and provisioning of secrets required by Vrooli resources. This eliminates the manual process of hunting through resource documentation to identify required environment variables, API keys, and credentials needed for proper resource functionality.

### Intelligence Amplification
**How does this capability make future agents smarter?**
- Agents can automatically validate dependencies before attempting to use resources
- Prevents runtime failures due to missing credentials
- Creates a centralized knowledge base of all resource secret requirements
- Enables predictable, reproducible deployments of scenarios that depend on external services

### Recursive Value
**What new scenarios become possible after this exists?**
1. **Automated Deployment Validator** - Scenarios that check environment completeness before startup
2. **Resource Dependency Mapper** - Visual mapping of which scenarios need which credentials
3. **Security Compliance Scanner** - Auditing which secrets are stored securely vs. in environment
4. **CI/CD Environment Validator** - Ensuring all required secrets are available in deployment pipelines
5. **Resource Onboarding Assistant** - Guided setup for new developers joining Vrooli

## 📊 Success Metrics

### Functional Requirements
- **Must Have (P0)**
  - [ ] Scan all `/resources/` configurations to identify required secrets (env vars, credentials, API keys)
  - [ ] Validate current environment against discovered requirements
  - [ ] Generate prioritized report of missing secrets with resource-specific guidance
  - [ ] Dark chrome security dashboard UI showing secret health status
  - [ ] CLI interface for programmatic secret validation
  
- **Should Have (P1)**
  - [ ] Interactive provisioning wizard for missing secrets
  - [ ] Integration with Vault for secure secret storage and retrieval
  - [ ] Export missing secrets checklist for manual setup
  - [ ] Resource-specific setup documentation and links
  
- **Nice to Have (P2)**
  - [ ] Automated secret rotation reminders
  - [ ] CI/CD environment export functionality
  - [ ] Secret usage analytics (which resources use which secrets)

### Performance Criteria
| Metric | Target | Measurement Method |
|--------|--------|-------------------|
| Resource Scan Time | < 5s for all resources | CLI timing |
| Dashboard Load Time | < 2s initial load | Browser measurement |
| Validation Accuracy | > 95% correct identification | Manual verification |
| Memory Usage | < 500MB peak usage | System monitoring |

### Quality Gates
- [ ] All P0 requirements implemented and tested
- [ ] Integration tests pass with vault, postgres resources
- [ ] Dark chrome UI matches cyberpunk security aesthetic
- [ ] CLI commands provide both human and JSON output
- [ ] Complete resource secret discovery for 90%+ of existing resources

## 🏗️ Technical Architecture

### Resource Dependencies
```yaml
required:
  - resource_name: vault
    purpose: Secure secret storage and retrieval
    integration_pattern: CLI and shared workflow
    access_method: resource-vault CLI commands
    
  - resource_name: postgres
    purpose: Metadata tracking and secret requirement persistence
    integration_pattern: Direct connection
    access_method: Standard postgres connection
    
  - resource_name: n8n
    purpose: Resource scanning and validation workflows
    integration_pattern: Shared workflows
    access_method: shared n8n workflows
    
optional:
  - resource_name: redis
    purpose: Caching scan results between runs
    fallback: In-memory caching with reduced performance
    access_method: resource-redis CLI commands
```

### Resource Integration Standards
```yaml
integration_priorities:
  1_shared_workflows:
    - workflow: resource-scanner.json
      location: initialization/automation/n8n/
      purpose: Automated scanning of resource configurations
    - workflow: secret-validator.json
      location: initialization/automation/n8n/
      purpose: Validation of secrets against live resources
  
  2_resource_cli:
    - command: resource-vault get <path>
      purpose: Retrieve stored secrets
    - command: resource-vault put <path> <data>
      purpose: Store new secrets securely
  
  3_direct_api:
    - justification: Real-time resource health checks require direct API access
      endpoint: Various resource health endpoints
```

### Data Models
```yaml
primary_entities:
  - name: ResourceSecret
    storage: postgres
    schema: |
      {
        id: UUID,
        resource_name: string,
        secret_key: string,
        secret_type: enum(env_var, api_key, credential, token),
        required: boolean,
        description: string,
        validation_pattern: string,
        documentation_url: string,
        last_validated: timestamp
      }
    relationships: Links to resource configurations and validation results
    
  - name: SecretValidation
    storage: postgres
    schema: |
      {
        id: UUID,
        resource_secret_id: UUID,
        validation_status: enum(missing, invalid, valid),
        validation_method: enum(env, vault, file),
        validation_timestamp: timestamp,
        error_message: string
      }
    relationships: Links to ResourceSecret records
```

### API Contract
```yaml
endpoints:
  - method: GET
    path: /api/v1/secrets/scan
    purpose: Trigger full resource secret discovery scan
    input_schema: |
      {
        resources?: string[]  // Optional filter to specific resources
      }
    output_schema: |
      {
        scan_id: string,
        discovered_secrets: ResourceSecret[],
        scan_duration_ms: number
      }
    sla:
      response_time: 5000ms
      availability: 99%
      
  - method: GET
    path: /api/v1/secrets/validate
    purpose: Validate current environment against requirements
    input_schema: |
      {
        resource?: string  // Optional specific resource
      }
    output_schema: |
      {
        validation_id: string,
        total_secrets: number,
        valid_secrets: number,
        missing_secrets: SecretValidation[],
        invalid_secrets: SecretValidation[]
      }
    sla:
      response_time: 3000ms
      availability: 99%
      
  - method: POST
    path: /api/v1/secrets/provision
    purpose: Interactive secret provisioning workflow
    input_schema: |
      {
        secret_key: string,
        secret_value: string,
        storage_method: enum(vault, env)
      }
    output_schema: |
      {
        success: boolean,
        storage_location: string,
        validation_result: SecretValidation
      }
    sla:
      response_time: 2000ms
      availability: 99%
```

### Event Interface
```yaml
published_events:
  - name: secrets.scan.completed
    payload: { scan_id: string, discovered_count: number, resources: string[] }
    subscribers: [dashboard, cli, logging]
    
  - name: secrets.validation.completed
    payload: { validation_id: string, missing_count: number, invalid_count: number }
    subscribers: [dashboard, notifications]
    
consumed_events:
  - name: resource.status.changed
    action: Re-validate secrets for the affected resource
```

## 🖥️ CLI Interface Contract

### Command Structure
```yaml
cli_binary: secrets-manager
install_script: cli/install.sh

required_commands:
  - name: status
    description: Show secret validation status across all resources
    flags: [--json, --verbose, --resource <name>]
    
  - name: help
    description: Display command help and usage
    flags: [--all, --command <name>]
    
  - name: version
    description: Show CLI and API version information
    flags: [--json]

custom_commands:
  - name: scan
    description: Discover secrets required by resources
    api_endpoint: /api/v1/secrets/scan
    arguments:
      - name: resource
        type: string
        required: false
        description: Specific resource to scan (optional)
    flags:
      - name: --output
        description: Output format (table, json, yaml)
    output: Table of discovered secrets or JSON structure
    
  - name: validate
    description: Validate current environment secrets
    api_endpoint: /api/v1/secrets/validate
    arguments: []
    flags:
      - name: --resource
        description: Validate specific resource only
      - name: --missing-only
        description: Show only missing secrets
    output: Validation report with missing/invalid secrets
    
  - name: provision
    description: Interactive secret provisioning wizard
    api_endpoint: /api/v1/secrets/provision
    arguments: []
    flags:
      - name: --secret
        description: Specific secret key to provision
      - name: --vault
        description: Store in vault instead of environment
    output: Success confirmation and storage location
```

## 🔄 Integration Requirements

### Upstream Dependencies
**What capabilities must exist before this can function?**
- **Vault Resource**: Required for secure secret storage and retrieval
- **Resource CLI Framework**: Needs access to resource CLI commands for validation

### Downstream Enablement
**What future capabilities does this unlock?**
- **Environment Validation**: Other scenarios can verify prerequisites before startup
- **Automated Onboarding**: New developers can quickly identify and provision required secrets
- **Security Auditing**: Compliance scenarios can verify secret storage practices

### Cross-Scenario Interactions
```yaml
provides_to:
  - scenario: any
    capability: Environment validation and secret discovery
    interface: CLI/API
    
consumes_from:
  - scenario: vault
    capability: Secure secret storage
    fallback: Environment variables with security warnings
```

## 🎨 Style and Branding Requirements

### UI/UX Style Guidelines
```yaml
style_profile:
  category: technical
  inspiration: HashiCorp Vault UI meets cyberpunk security terminal
  
  visual_style:
    color_scheme: dark
    primary_colors: 
      - background: "#0a0a0a" (deep black)
      - surface: "#1a1a1a" (dark gray)
      - accent: "#00ff41" (matrix green)
      - warning: "#ff9500" (amber)
      - error: "#ff4444" (red)
      - chrome: "#c0c0c0" (silver/chrome accents)
    typography: 
      - primary: "JetBrains Mono" (monospace for technical data)
      - ui: "Inter" (clean sans-serif for interface)
    layout: dashboard
    animations: subtle (glow effects, smooth transitions)
  
  personality:
    tone: technical
    mood: focused
    target_feeling: "confident control over security infrastructure"

style_references:
  technical:
    - system-monitor: "Matrix-style green terminal aesthetic"
    - vault-ui: "Professional security dashboard with dark theme"
    - cyberpunk: "Chrome accents, neon highlights, industrial feel"
```

### Target Audience Alignment
- **Primary Users**: Developers and DevOps engineers managing Vrooli deployments
- **User Expectations**: Professional, secure, information-dense interface
- **Accessibility**: WCAG AA compliance, high contrast for security environments
- **Responsive Design**: Desktop primary, tablet secondary (CLI for mobile use)

### Brand Consistency Rules
- **Scenario Identity**: Cyberpunk security vault aesthetic
- **Vrooli Integration**: Maintains Vrooli's technical sophistication
- **Professional Focus**: Function over form, but with distinctive dark chrome personality

## 💰 Value Proposition

### Business Value
- **Primary Value**: Eliminates manual secret configuration debugging
- **Time Savings**: Reduces deployment setup from hours to minutes
- **Error Prevention**: Prevents runtime failures due to missing credentials
- **Developer Experience**: Streamlines onboarding and development workflow

### Technical Value
- **Reusability Score**: High - every scenario using external resources benefits
- **Complexity Reduction**: Transforms secret management from reactive to proactive
- **Innovation Enablement**: Enables confident deployment of complex multi-resource scenarios

## 🧬 Evolution Path

### Version 1.0 (Current)
- Resource secret discovery and validation
- Dark chrome dashboard UI
- Basic CLI interface with JSON output

### Version 2.0 (Planned)
- Automated secret rotation scheduling
- Integration with external secret management systems
- Advanced security compliance reporting

### Long-term Vision
- Becomes the central nervous system for all Vrooli credential management
- Enables zero-touch deployment pipelines
- Provides security audit trails for enterprise compliance

## 🔄 Scenario Lifecycle Integration

### Scenario-to-App Conversion
```yaml
app_conversion:
  supported: true
  app_structure_compliance:
    - service.json with complete metadata
    - All required initialization files
    - Deployment scripts (startup.sh, monitor.sh)
    - Health check endpoints
    
  deployment_targets:
    - local: Docker Compose based
    - kubernetes: Helm chart generation
    
  revenue_model:
    - type: internal-tool
    - pricing_tiers: N/A (development utility)
    - trial_period: N/A
```

## 🚨 Risk Mitigation

### Technical Risks
| Risk | Probability | Impact | Mitigation |
|------|------------|--------|------------|
| Vault unavailability | Low | High | Graceful degradation to env vars |
| Resource API changes | Medium | Medium | Version-aware scanning |
| Secret exposure | Low | Critical | Masked display, secure storage |

### Operational Risks
- **Secret Discovery**: False positives/negatives in secret detection
- **Access Control**: Ensure proper permissions for secret access
- **Audit Trail**: All secret operations must be logged securely

## ✅ Validation Criteria

### Declarative Test Specification
```yaml
version: 1.0
scenario: secrets-manager

structure:
  required_files:
    - .vrooli/service.json
    - PRD.md
    - api/main.go
    - api/go.mod
    - cli/secrets-manager
    - cli/install.sh
    - initialization/storage/postgres/schema.sql
    - initialization/automation/n8n/resource-scanner.json
    - initialization/automation/n8n/secret-validator.json
    - ui/index.html
    - ui/script.js
    - ui/server.js
    - ui/styles.css
    - scenario-test.yaml
    
  required_dirs:
    - api
    - cli
    - initialization
    - initialization/automation
    - initialization/storage
    - ui

resources:
  required: [vault, postgres, n8n]
  optional: [redis]
  health_timeout: 60

tests:
  - name: "Vault is accessible"
    type: http
    service: vault
    endpoint: /v1/sys/health
    method: GET
    expect:
      status: 200
      
  - name: "Database connection"
    type: tcp
    service: postgres
    port: 5432
      
  - name: "Secrets scan API endpoint"
    type: http
    service: api
    endpoint: /api/v1/secrets/scan
    method: GET
    expect:
      status: 200
      body:
        type: json
        
  - name: "Secrets validation API endpoint"
    type: http
    service: api
    endpoint: /api/v1/secrets/validate
    method: GET
    expect:
      status: 200
      
  - name: "CLI status command executes"
    type: exec
    command: ./cli/secrets-manager status --json
    expect:
      exit_code: 0
      output_contains: ["secrets"]
      
  - name: "Dark chrome UI loads"
    type: custom
    script: custom-tests.sh
    function: test_dashboard_ui_loads
```

### Performance Validation
- [ ] Resource scan completes within 5 seconds
- [ ] Dashboard loads in under 2 seconds
- [ ] Memory usage stays under 500MB
- [ ] CLI commands respond within 1 second

### Integration Validation
- [ ] All required resources properly integrated
- [ ] Vault integration functional for secret storage/retrieval
- [ ] n8n workflows properly registered and active
- [ ] Dark chrome UI displays secret status correctly

### Capability Verification
- [ ] Successfully discovers secrets from at least 5 different resource types
- [ ] Correctly identifies missing environment variables
- [ ] Interactive provisioning wizard stores secrets in Vault
- [ ] Dashboard provides clear visual indication of secret health status

## 📝 Implementation Notes

### Design Decisions
**Resource Scanning Approach**: File-based configuration parsing instead of runtime introspection
- Alternative considered: Runtime resource querying
- Decision driver: More reliable and doesn't require resources to be running
- Trade-offs: May miss dynamically configured secrets, but safer and more predictable

**UI Framework**: Vanilla JavaScript with custom dark chrome styling
- Alternative considered: React/Vue framework
- Decision driver: Lighter weight, better control over cyberpunk aesthetic
- Trade-offs: More manual styling work, but unique visual identity

### Known Limitations
- **Dynamic Secrets**: Cannot detect secrets that are generated or discovered at runtime
  - Workaround: Focus on configuration-declared secrets initially
  - Future fix: Add runtime secret discovery hooks in version 2.0

### Security Considerations
- **Data Protection**: All secret values masked in UI and logs
- **Access Control**: Inherits Vault's access control mechanisms
- **Audit Trail**: All secret access and provisioning operations logged to postgres

## 🔗 References

### Documentation
- README.md - User-facing overview and quick start
- docs/api.md - Complete API specification
- docs/cli.md - CLI command reference

### Related PRDs
- resources/vault/PRD.md - Vault resource integration
- Future: deployment-validator scenario (depends on this)

### External Resources
- HashiCorp Vault API Documentation
- Resource CLI Framework Standards
- Vrooli Security Best Practices

---

**Last Updated**: 2024-08-29  
**Status**: Draft  
**Owner**: Claude Code AI  
**Review Cycle**: Upon implementation milestone completion