# Product Requirements Document (PRD) - n8n

## 🎯 Infrastructure Definition

### Core Infrastructure Capability
**What permanent infrastructure capability does this resource add to Vrooli?**
n8n provides visual workflow automation infrastructure, enabling no-code/low-code automation pipelines that connect resources, process data, and orchestrate complex multi-step operations across the Vrooli ecosystem.

### System Amplification
**How does this resource make the entire Vrooli system more capable?**
- **Cross-resource orchestration** - Visually connects any resources through webhook/API integrations
- **Event-driven automation** - Triggers workflows based on system events, schedules, or external webhooks
- **Data transformation** - Built-in nodes for processing, filtering, and transforming data between resources
- **Error handling & retries** - Automatic failure recovery and notification capabilities
- **Rapid prototyping** - Visual workflow builder enables quick scenario development without code

### Enabling Value
**What new scenarios become possible when this resource is available?**
1. **Business Process Automation**: Order processing, invoice generation, customer onboarding workflows
2. **Data Pipeline Orchestration**: ETL operations, data synchronization, report generation
3. **AI Agent Coordination**: Multi-agent workflows, model chaining, prompt engineering pipelines
4. **Monitoring & Alerting**: System health checks, anomaly detection, automated incident response
5. **Content Management**: Automated publishing, content moderation, social media management

## 📊 Infrastructure Metrics

### Functional Requirements
- **Must Have (P0)**
  - [x] Core workflow engine with visual editor
  - [x] Webhook trigger support for event-driven automation
  - [x] REST API for programmatic workflow management
  - [x] Integration with Vrooli resource framework
  - [x] Standard CLI interface (resource-n8n)
  - [x] Health monitoring and status reporting
  - [x] Docker containerization and networking
  - [ ] Content management for workflows (add/list/get/remove/execute)
  
- **Should Have (P1)**
  - [x] Credential management for resource integrations
  - [x] Workflow activation/deactivation controls
  - [x] Execution history and logging
  - [x] Performance optimization features
  - [x] Advanced configuration options
  - [ ] Workflow templates and sharing
  
- **Nice to Have (P2)**
  - [ ] Workflow versioning and rollback
  - [ ] A/B testing capabilities
  - [ ] Advanced analytics and metrics

### Performance Criteria
| Metric | Target | Measurement Method |
|--------|--------|-------------------|
| Service Startup Time | < 30s | Container initialization |
| Health Check Response | < 500ms | API/CLI status checks |
| Resource Utilization | < 30% CPU/Memory | Resource monitoring |
| Availability | > 99% uptime | Service monitoring |
| Workflow Execution | < 100ms overhead | Execution timing |

### Quality Gates
- [x] All P0 requirements implemented and tested
- [x] Integration tests pass with all dependent resources
- [x] Performance targets met under expected load
- [x] Security standards met for resource type
- [ ] Documentation complete and accurate

## 🏗️ Technical Architecture

### Resource Dependencies
```yaml
required:
  - resource_name: postgres
    purpose: Stores workflow definitions, credentials, and execution history
    integration_pattern: Direct database connection
    access_method: PostgreSQL client
    
  - resource_name: redis
    purpose: Queue management, caching, and pub/sub for workflows
    integration_pattern: Redis client connection
    access_method: Redis protocol
    
optional:
  - resource_name: vault
    purpose: Secure credential storage
    fallback: Use n8n's built-in encrypted credential storage
    access_method: API
    
  - resource_name: browserless
    purpose: Web scraping and screenshot capabilities
    fallback: Workflows requiring browser automation are disabled
    access_method: API
```

### Integration Standards
```yaml
resource_category: automation

standard_interfaces:
  management:
    - cli: cli.sh (using CLI framework)
    - actions: [help, install, uninstall, start, stop, restart, status, validate, test, content, inject, list-workflows, activate-workflow, deactivate-workflow, delete-workflow, auto-credentials]
    - configuration: config/defaults.sh
    - documentation: README.md + docs/
    
  networking:
    - docker_networks: [vrooli-network]
    - port_registry: Port 5678 defined in scripts/resources/port_registry.sh
    - hostname: vrooli-n8n
    
  monitoring:
    - health_check: GET /health
    - status_reporting: resource-n8n status (uses status-args.sh framework)
    - logging: Docker container logs
    
  data_persistence:
    - volumes: [n8n-data, n8n-config]
    - backup_strategy: PostgreSQL backup includes workflow data
    - migration_support: n8n handles schema migrations automatically

integration_patterns:
  scenarios_using_resource:
    - scenario_name: study-buddy
      usage_pattern: Generates flashcards and manages spaced repetition
      
    - scenario_name: mind-maps
      usage_pattern: Auto-organizes content and semantic search
      
    - scenario_name: image-generation-pipeline
      usage_pattern: Orchestrates image generation workflows
      
  resource_to_resource:
    - postgres → n8n: Workflow and credential storage
    - n8n → ollama: Invokes local AI models in workflows
    - n8n → openrouter: Calls external AI models
    - n8n → browserless: Web automation tasks
```

### Configuration Schema
```yaml
resource_configuration:
  defaults:
    enabled: true
    port: 5678 # Retrieved from port_registry.sh
    networks: [vrooli-network]
    volumes: [n8n-data, n8n-config]
    environment:
      - N8N_HOST=0.0.0.0
      - N8N_PORT=5678
      - N8N_PROTOCOL=http
      - WEBHOOK_URL=http://localhost:5678/
    
  templates:
    development:
      - description: Dev-optimized with verbose logging
      - overrides: 
          - N8N_LOG_LEVEL=debug
          - EXECUTIONS_DATA_SAVE_ON_SUCCESS=all
      
    production:
      - description: Production-optimized with minimal logging
      - overrides:
          - N8N_LOG_LEVEL=warn
          - EXECUTIONS_DATA_SAVE_ON_SUCCESS=none
      
    testing:
      - description: Test-optimized with full execution tracking
      - overrides:
          - N8N_LOG_LEVEL=info
          - EXECUTIONS_DATA_SAVE_ON_SUCCESS=all
      
customization:
  user_configurable:
    - parameter: N8N_BASIC_AUTH_ACTIVE
      description: Enable basic authentication
      default: false
      
    - parameter: N8N_METRICS
      description: Enable metrics endpoint
      default: false
      
  environment_variables:
    - var: DB_TYPE
      purpose: Database type (postgres/sqlite)
      
    - var: DB_POSTGRESDB_HOST
      purpose: PostgreSQL host
```

### API Contract
```yaml
api_endpoints:
  - method: GET
    path: /api/workflows
    purpose: List all workflows
    output_schema: |
      {
        "data": [
          {
            "id": "string",
            "name": "string",
            "active": boolean,
            "createdAt": "string",
            "updatedAt": "string"
          }
        ]
      }
    authentication: API key or session
    rate_limiting: None
    
  - method: POST
    path: /api/workflows/{id}/activate
    purpose: Activate a workflow
    output_schema: |
      {
        "success": boolean,
        "data": {
          "id": "string",
          "active": true
        }
      }
    authentication: API key or session
    rate_limiting: None
    
  - method: POST
    path: /webhook/{id}
    purpose: Trigger workflow via webhook
    input_schema: |
      {
        [any JSON payload]
      }
    output_schema: |
      {
        [workflow execution result]
      }
    authentication: Optional (workflow-specific)
    rate_limiting: Workflow-specific
```

## 🖥️ Management Interface Contract

### Required Management Actions
```yaml
standard_actions:
  - name: install
    description: Install and configure n8n
    flags: [--force, --template <name>]
    
  - name: start  
    description: Start the n8n service
    flags: [--wait]
    
  - name: stop
    description: Stop the n8n service gracefully
    flags: [--force]
    
  - name: status
    description: Show detailed n8n status
    flags: [--json, --verbose]
    
  - name: uninstall
    description: Remove n8n and cleanup
    flags: [--keep-data, --force]
    
  - name: content
    description: Manage workflow content (add/list/get/remove/execute)
    flags: [--file, --name, --format]

resource_specific_actions:
  - name: list-workflows
    description: List all workflows with their status
    flags: [--active-only, --json]
    example: resource-n8n list-workflows --active-only
    
  - name: activate-workflow
    description: Activate a specific workflow
    flags: [--workflow-id, --workflow-name]
    example: resource-n8n activate-workflow --workflow-name "Data Sync"
    
  - name: auto-credentials
    description: Auto-discover and create credentials for available resources
    flags: [--force]
    example: resource-n8n auto-credentials
    
  - name: inject
    description: (Deprecated - use content) Import workflow from file
    flags: [--workflow-file, --credential-file]
    example: resource-n8n inject --workflow-file workflow.json
```

### Management Standards
```yaml
implementation_requirements:
  - cli_location: cli.sh (uses CLI framework)
  - configuration: config/defaults.sh
  - dependencies: lib/ directory with modular functions
  - error_handling: Exit codes (0=success, 1=error, 2=config error)
  - logging: Structured output with levels (INFO, WARN, ERROR)
  - idempotency: Safe to run commands multiple times
  
status_reporting:
  - health_status: healthy|degraded|unhealthy|unknown
  - service_info: version, uptime, resource usage
  - integration_status: PostgreSQL and Redis connectivity
  - configuration: current settings and overrides
  - workflow_count: Active and total workflows
  
output_formats:
  - default: Human-readable with color coding
  - json: Structured data with --json flag
  - verbose: Detailed diagnostics with --verbose
```

## 🔧 Operational Requirements

### Deployment Standards
```yaml
containerization:
  base_image: n8nio/n8n:latest
  dockerfile_location: Not needed (official image)
  build_requirements: None
  
networking:
  required_networks:
    - vrooli-network: Primary inter-resource communication
    
  port_allocation:
    - internal: 5678
    - external: 5678 (retrieved from port_registry.sh)
    - protocol: tcp
    - purpose: Web UI and API access
    - registry_integration:
        definition: Port defined in scripts/resources/port_registry.sh
        retrieval: Use port_registry functions to get assigned port
        no_hardcoding: Never hardcode ports in resource code
    
data_management:
  persistence:
    - volume: n8n-data
      mount: /home/node/.n8n
      purpose: Workflow files, credentials, and settings
      
  backup_strategy:
    - method: PostgreSQL backup includes workflow data
    - frequency: Daily with database backups
    - retention: 30 days
    
  migration_support:
    - version_compatibility: Automatic schema migration
    - upgrade_path: n8n handles database migrations
    - rollback_support: Database snapshots before upgrade
```

### Performance Standards
```yaml
resource_requirements:
  minimum:
    cpu: 0.5 vCPU
    memory: 512MB RAM
    disk: 1GB
    
  recommended:
    cpu: 2 vCPU
    memory: 2GB RAM
    disk: 10GB
    
  scaling:
    horizontal: Multiple instances with Redis queue
    vertical: Can utilize additional CPU/memory
    limits: Database connection pool limits
    
monitoring_requirements:
  health_checks:
    - endpoint: /health
    - interval: 30s
    - timeout: 5s
    - failure_threshold: 3
    
  metrics:
    - metric: workflow_executions
      collection: API metrics endpoint
      alerting: Failures > 10% in 5 minutes
      
    - metric: execution_time
      collection: Execution history
      alerting: P95 > 30 seconds
```

### Security Standards
```yaml
security_requirements:
  authentication:
    - method: Session-based or API key
    - credential_storage: Encrypted in database
    - session_management: Redis session store
    
  authorization:
    - access_control: Role-based (owner, editor, viewer)
    - role_based: Workflow-level permissions
    - resource_isolation: User-specific workspaces
    
  data_protection:
    - encryption_at_rest: Credential encryption
    - encryption_in_transit: HTTPS recommended
    - key_management: Environment variable keys
    
  network_security:
    - port_exposure: 5678 (configurable)
    - firewall_requirements: Allow webhook traffic
    - ssl_tls: Reverse proxy recommended
    
compliance:
  standards: GDPR data handling
  auditing: Execution history logs
  data_retention: Configurable retention policies
```

## 🧪 Testing Strategy

### Test Categories
```yaml
unit_tests:
  location: Co-located with source files (e.g., lib/docker.sh and lib/docker.bats)
  coverage: Individual function testing
  framework: BATS (Bash Automated Testing System)
  
integration_tests:
  location: test/ directory
  coverage: Workflow execution, API endpoints, credential management
  test_data: Uses shared fixtures from __test/fixtures/data/
  test_scenarios: 
    - Resource lifecycle (install, start, stop, uninstall)
    - Workflow import/export via content management
    - Webhook triggering
    - Credential auto-discovery
    - Error handling and recovery
  
system_tests:
  location: __test/resources/
  coverage: Full resource lifecycle, multi-resource workflows
  automation: Integrated with Vrooli test framework
  
performance_tests:
  load_testing: Concurrent workflow executions
  stress_testing: Maximum workflow complexity
  endurance_testing: Long-running workflows
```

### Test Specifications
```yaml
test_specification:
  resource_name: n8n
  test_categories: [unit, integration, system, performance]
  
  test_structure:
    - BATS files co-located with source files
    - Integration tests in test/ directory
    - Shared fixtures from __test/fixtures/data/
    - Test results included in status output with timestamp
    - Example workflows in examples/ directory
  
  lifecycle_tests:
    - name: "n8n Installation"
      command: resource-n8n install
      expect:
        exit_code: 0
        service_running: true
        health_status: healthy
        
    - name: "n8n Status"  
      command: resource-n8n status --json
      expect:
        exit_code: 0
        json_valid: true
        health_status: healthy
        test_results: includes last test timestamp
        
    - name: "Content Management - Add Workflow"
      command: resource-n8n content add --file workflow.json
      fixture: __test/fixtures/data/n8n/sample-workflow.json
      expect:
        exit_code: 0
        workflow_imported: true
        workflow_active: true
        
  performance_tests:
    - name: "Service Startup Time"
      measurement: startup_latency
      target: < 30 seconds
      
    - name: "Workflow Execution Overhead"
      measurement: execution_overhead
      target: < 100ms per node
      
  failure_tests:
    - name: "Database Connection Loss"
      scenario: PostgreSQL becomes unavailable
      expect: Queued workflows, graceful degradation
```

## 💰 Infrastructure Value

### Technical Value
- **Infrastructure Stability**: Provides reliable workflow orchestration with automatic retries and error handling
- **Integration Simplicity**: Visual workflow builder reduces integration complexity between resources
- **Operational Efficiency**: Automates repetitive tasks and complex multi-step processes
- **Developer Experience**: No-code/low-code approach enables rapid prototyping and iteration

### Resource Economics
- **Setup Cost**: ~5 minutes to configure and deploy
- **Operating Cost**: Low resource consumption (< 512MB RAM idle)
- **Integration Value**: Force multiplier for other resources through orchestration
- **Maintenance Overhead**: Self-maintaining with automatic updates

## 🔄 Resource Lifecycle Integration

### Vrooli Integration Standards
```yaml
resource_discovery:
  registry_entry:
    name: n8n
    category: automation
    capabilities: [workflow-automation, webhook-triggers, api-integration, scheduling]
    interfaces:
      - cli: resource-n8n (installed via install-resource-cli.sh)
      - api: http://localhost:5678/api
      - health: http://localhost:5678/health
      
  metadata:
    description: Visual workflow automation platform
    version: 1.x (auto-updating)
    dependencies: [postgres, redis]
    enables: [process-automation, data-pipelines, event-driven-workflows]

resource_framework_compliance:
  - Standard directory structure (/config, /lib, /docs, /test, etc.)
  - CLI framework integration (cli.sh as thin wrapper over lib/ functions)
  - Port registry integration
  - Docker network integration  
  - Health monitoring integration
  - Configuration management standards
  
deployment_integration:
  supported_targets:
    - local: Docker Compose deployment
    - kubernetes: StatefulSet with persistent storage
    - cloud: Managed n8n cloud option
    
  configuration_management:
    - Environment-based configuration
    - Template-based setup
    - Vault integration for credentials
```

### Version Management
```yaml
versioning:
  current: 1.x (rolling release)
  compatibility: Automatic migration between versions
  upgrade_path: Docker image updates handle migrations
  
  breaking_changes: None expected (stable API)
  deprecations: inject command → content management
  migration_guide: Automatic database migration on startup

release_management:
  release_cycle: Weekly updates from upstream
  testing_requirements: Integration tests before update
  rollback_strategy: Volume snapshots before upgrade
```

## 🧬 Evolution Path

### Version 1.x (Current)
- Visual workflow builder with 400+ integrations
- Webhook and schedule triggers
- Basic credential management
- PostgreSQL/Redis backend

### Version 2.x (Planned)
- Workflow versioning and Git integration
- Advanced debugging and testing tools
- Multi-tenant support
- Enhanced AI node capabilities

### Long-term Vision
- AI-powered workflow generation
- Self-optimizing workflows
- Distributed workflow execution
- Native Vrooli resource nodes

## 🚨 Risk Mitigation

### Technical Risks
| Risk | Probability | Impact | Mitigation |
|------|------------|--------|------------|
| Workflow execution failures | Medium | High | Automatic retries, error workflows |
| Database corruption | Low | Critical | Regular backups, transaction logs |
| Memory leaks in long-running workflows | Low | Medium | Workflow timeouts, monitoring |
| Webhook flooding | Medium | Medium | Rate limiting, queue management |

### Operational Risks
- **Configuration Drift**: Version-controlled workflows in content management
- **Dependency Failures**: Graceful degradation when resources unavailable
- **Resource Conflicts**: Port registry prevents conflicts
- **Update Compatibility**: Automatic migration testing

## ✅ Validation Criteria

### Infrastructure Validation
- [x] Resource installs and starts successfully
- [x] All management actions work correctly
- [x] Integration with other resources functions properly
- [x] Performance meets established targets
- [x] Security requirements satisfied
- [ ] Documentation complete and accurate

### Integration Validation  
- [x] Successfully enables dependent scenarios
- [x] Integrates properly with Vrooli resource framework
- [x] Networking and discovery work correctly
- [ ] Configuration management functions properly
- [x] Monitoring and alerting work as expected

### Operational Validation
- [x] Deployment procedures documented and tested
- [x] Backup and recovery procedures verified
- [x] Upgrade and rollback procedures validated
- [ ] Troubleshooting documentation complete
- [x] Performance under load verified

## 📝 Implementation Notes

### Design Decisions
**Content Management over Inject**: Moving from inject pattern to content management
- Alternative considered: Keep inject as primary interface
- Decision driver: Clearer semantics and alignment with other resources
- Trade-offs: Migration period with dual support

**Redis for Queue Management**: Using Redis instead of in-memory queues
- Alternative considered: RabbitMQ or internal queue
- Decision driver: Already available in Vrooli, reduces dependencies
- Trade-offs: Requires Redis but enables horizontal scaling

### Known Limitations
- **Workflow Size**: Large workflows (>100 nodes) may have UI performance issues
  - Workaround: Split into multiple smaller workflows
  - Future fix: Pagination and lazy loading
  
- **Execution History**: Limited retention in free version
  - Workaround: External logging to QuestDB
  - Future fix: Configurable retention policies

### Integration Considerations
- **Database Sharing**: Uses same PostgreSQL as other resources
- **Network Isolation**: Webhooks need external access
- **Credential Security**: Shared credentials need careful management
- **Resource Discovery**: Auto-discovery simplifies integration

## 🔗 References

### Documentation
- README.md - Quick start and overview
- docs/workflow-examples.md - Common workflow patterns
- config/defaults.sh - Configuration options
- lib/inject.sh - Workflow import implementation

### Related Resources
- postgres - Database backend for workflows
- redis - Queue and caching layer
- ollama - Local AI model integration
- browserless - Web automation capabilities

### External Resources
- [n8n Official Documentation](https://docs.n8n.io)
- [n8n Community Workflows](https://n8n.io/workflows)
- [n8n API Reference](https://docs.n8n.io/api)

---

**Last Updated**: 2025-08-21
**Status**: Review
**Owner**: Vrooli Infrastructure Team
**Review Cycle**: Monthly