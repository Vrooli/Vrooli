# Product Requirements Document (PRD) - Vault

## 🎯 Infrastructure Definition

### Core Infrastructure Capability
**What permanent infrastructure capability does this resource add to Vrooli?**
Vault provides secure, centralized secrets management and encryption services, enabling zero-trust security for credentials, API keys, tokens, and sensitive configuration across all Vrooli resources.

### System Amplification
**How does this resource make the entire Vrooli system more capable?**
- **Security Foundation**: Provides cryptographic storage and access control for all sensitive data across the platform
- **Dynamic Secrets**: Generates short-lived credentials for resources, reducing exposure risk
- **Unified Access**: Single source of truth for all credentials, eliminating hardcoded secrets
- **Audit Trail**: Complete logging of all secret access for compliance and debugging
- **Developer Experience**: Simplifies secret management with consistent APIs and CLI tools

### Enabling Value
**What new scenarios become possible when this resource is available?**
1. **Multi-Provider AI Scenarios**: Secure storage of API keys for OpenRouter, Gemini, and other AI providers
2. **Financial Integration**: Safe handling of payment processor credentials and transaction keys
3. **Enterprise Workflows**: Compliance-ready audit trails for sensitive business operations
4. **Zero-Trust Automation**: Dynamic credential rotation for database and service connections
5. **Multi-Tenant Applications**: Isolated secret namespaces for different user organizations

## 📊 Infrastructure Metrics

### Functional Requirements
- **Must Have (P0)**
  - [x] Secure key-value secret storage
  - [x] HTTP API for secret operations
  - [x] Docker container deployment
  - [x] Integration with Vrooli resource framework
  - [x] Standard CLI interface (resource-vault)
  - [x] Health monitoring and status reporting
  - [x] Dev mode for local development
  
- **Should Have (P1)**
  - [ ] Content management commands (add/list/get/remove/execute)
  - [ ] Production mode with persistent storage
  - [ ] Dynamic secret generation for databases
  - [ ] Secret rotation capabilities
  - [ ] Namespace isolation for multi-tenancy
  
- **Nice to Have (P2)**
  - [ ] PKI certificate management
  - [ ] SSH key management
  - [ ] Integration with Kubernetes secrets

### Performance Criteria
| Metric | Target | Measurement Method |
|--------|--------|-------------------|
| Service Startup Time | < 5s | Container initialization |
| Health Check Response | < 100ms | API/CLI status checks |
| Resource Utilization | < 5% CPU/Memory | Docker stats monitoring |
| Availability | > 99.9% uptime | Service monitoring |
| Secret Access Latency | < 50ms | API response time |

### Quality Gates
- [x] All P0 requirements implemented and tested
- [x] Integration tests pass with all dependent resources
- [x] Performance targets met under expected load
- [x] Security standards met for secret storage
- [ ] Documentation complete and accurate

## 🏗️ Technical Architecture

### Resource Dependencies
```yaml
required:
  - resource_name: docker
    purpose: Container runtime for Vault service
    integration_pattern: Docker container management
    access_method: Docker API
    
optional:
  - resource_name: postgres
    purpose: Backend storage for production mode
    fallback: File storage backend
    access_method: Direct database connection
    
  - resource_name: redis
    purpose: High-availability coordination
    fallback: Single-node operation
    access_method: Redis protocol
```

### Integration Standards
```yaml
resource_category: storage

standard_interfaces:
  management:
    - cli: cli.sh (using CLI framework)
    - actions: [help, install, uninstall, start, stop, restart, status, validate, test, content, unseal, seal, init]
    - configuration: config/defaults.sh
    - documentation: README.md + docs/
    
  networking:
    - docker_networks: [vrooli-network]
    - port_registry: Port 8200 from scripts/resources/port_registry.sh
    - hostname: vrooli-vault
    
  monitoring:
    - health_check: http://localhost:8200/v1/sys/health
    - status_reporting: resource-vault status (uses status-args.sh framework)
    - logging: Docker container logs
    
  data_persistence:
    - volumes: [/vault/file, /vault/config, /vault/logs]
    - backup_strategy: Vault operator backup command
    - migration_support: Vault operator migrate

integration_patterns:
  scenarios_using_resource:
    - scenario_name: ai-provider-integration
      usage_pattern: Store and retrieve AI provider API keys
      
    - scenario_name: database-credentials
      usage_pattern: Dynamic database credential generation
      
  resource_to_resource:
    - openrouter → vault: Retrieves API keys for model access
    - n8n → vault: Fetches workflow credentials
    - postgres → vault: Stores connection strings
    - vault → audit-log: Sends access logs for compliance
```

### Configuration Schema
```yaml
resource_configuration:
  defaults:
    enabled: true
    port: 8200 # from port_registry.sh
    networks: [vrooli-network]
    volumes: 
      - vault-data:/vault/file
      - vault-config:/vault/config
      - vault-logs:/vault/logs
    environment:
      - VAULT_DEV_ROOT_TOKEN_ID=myroot
      - VAULT_DEV_LISTEN_ADDRESS=0.0.0.0:8200
      - VAULT_ADDR=http://127.0.0.1:8200
    
  templates:
    development:
      - description: Dev mode with in-memory storage
      - overrides:
          VAULT_DEV_ROOT_TOKEN_ID: myroot
          VAULT_LOG_LEVEL: debug
      
    production:
      - description: Production with file storage backend
      - overrides:
          VAULT_STORAGE_FILE: /vault/file
          VAULT_LOG_LEVEL: info
      
    testing:
      - description: Test mode with ephemeral storage
      - overrides:
          VAULT_DEV_ROOT_TOKEN_ID: test-token
          VAULT_LOG_LEVEL: warn
      
customization:
  user_configurable:
    - parameter: token
      description: Root token for dev mode
      default: myroot
      
    - parameter: log_level
      description: Logging verbosity
      default: info
      
  environment_variables:
    - var: VAULT_TOKEN
      purpose: Authentication token for Vault operations
      
    - var: VAULT_ADDR
      purpose: Vault server address
```

### API Contract
```yaml
api_endpoints:
  - method: GET
    path: /v1/sys/health
    purpose: Health check and seal status
    output_schema: |
      {
        "initialized": boolean,
        "sealed": boolean,
        "standby": boolean,
        "version": string
      }
    authentication: none
    rate_limiting: none
    
  - method: GET
    path: /v1/secret/data/{path}
    purpose: Retrieve secret data
    output_schema: |
      {
        "data": {
          "data": object,
          "metadata": object
        }
      }
    authentication: X-Vault-Token header
    rate_limiting: 1000/min per token
    
  - method: POST
    path: /v1/secret/data/{path}
    purpose: Store secret data
    input_schema: |
      {
        "data": object
      }
    authentication: X-Vault-Token header
    rate_limiting: 100/min per token
```

## 🖥️ Management Interface Contract

### Required Management Actions
```yaml
standard_actions:
  - name: install
    description: Install and configure Vault
    flags: [--force, --template <dev|prod|test>]
    
  - name: start
    description: Start the Vault service
    flags: [--wait]
    
  - name: stop
    description: Stop the Vault service gracefully
    flags: [--force]
    
  - name: status
    description: Show detailed Vault status
    flags: [--json, --verbose]
    
  - name: uninstall
    description: Remove Vault and cleanup
    flags: [--keep-data, --force]

resource_specific_actions:
  - name: unseal
    description: Unseal Vault using unseal keys
    flags: [--key <key>]
    example: resource-vault unseal --key "unseal-key-1"
    
  - name: seal
    description: Seal Vault to protect secrets
    flags: []
    example: resource-vault seal
    
  - name: init
    description: Initialize Vault for first use
    flags: [--key-shares <n>, --key-threshold <n>]
    example: resource-vault init --key-shares 5 --key-threshold 3
    
  - name: content
    description: Manage secrets content
    flags: [add, list, get, remove, execute]
    example: resource-vault content add --path secret/api-keys --data '{"openrouter": "key123"}'
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
  - integration_status: seal status, initialization status
  - configuration: current mode and settings
  
output_formats:
  - default: Human-readable with color coding
  - json: Structured data with --json flag
  - verbose: Detailed diagnostics with --verbose
```

## 🔧 Operational Requirements

### Deployment Standards
```yaml
containerization:
  base_image: hashicorp/vault:1.17
  dockerfile_location: N/A (uses official image)
  build_requirements: none
  
networking:
  required_networks:
    - vrooli-network: Primary inter-resource communication
    
  port_allocation:
    - internal: 8200
    - external: 8200 (from port_registry.sh)
    - protocol: tcp
    - purpose: Vault API and UI
    - registry_integration:
        definition: Port defined in scripts/resources/port_registry.sh
        retrieval: Use vault_get_port function
        no_hardcoding: Never hardcode ports in resource code
    
data_management:
  persistence:
    - volume: vault-data
      mount: /vault/file
      purpose: Secret storage (production mode)
      
    - volume: vault-config
      mount: /vault/config
      purpose: Vault configuration
      
    - volume: vault-logs
      mount: /vault/logs
      purpose: Audit and operational logs
      
  backup_strategy:
    - method: vault operator backup
    - frequency: Daily snapshots
    - retention: 30 days
    
  migration_support:
    - version_compatibility: 1.x series
    - upgrade_path: vault operator migrate
    - rollback_support: Snapshot restore
```

### Performance Standards
```yaml
resource_requirements:
  minimum:
    cpu: 0.5 cores
    memory: 256MB
    disk: 1GB
    
  recommended:
    cpu: 2 cores
    memory: 1GB
    disk: 10GB
    
  scaling:
    horizontal: HA mode with multiple instances
    vertical: Can utilize additional memory for caching
    limits: 5 nodes for HA cluster
    
monitoring_requirements:
  health_checks:
    - endpoint: /v1/sys/health
    - interval: 30s
    - timeout: 5s
    - failure_threshold: 3
    
  metrics:
    - metric: seal_status
      collection: API polling
      alerting: Alert if sealed
      
    - metric: secret_operations
      collection: Audit log analysis
      alerting: Spike detection
```

### Security Standards
```yaml
security_requirements:
  authentication:
    - method: Token-based authentication
    - credential_storage: Encrypted in Vault itself
    - session_management: Token TTL and renewal
    
  authorization:
    - access_control: Policy-based ACL
    - role_based: Path-based permissions
    - resource_isolation: Namespace separation
    
  data_protection:
    - encryption_at_rest: AES-256-GCM
    - encryption_in_transit: TLS 1.2+ (production)
    - key_management: Master key with Shamir sharing
    
  network_security:
    - port_exposure: 8200 (localhost only in dev)
    - firewall_requirements: Restrict to vrooli-network
    - ssl_tls: Required in production mode
    
compliance:
  standards: [SOC2, HIPAA, PCI-DSS compatible]
  auditing: Complete audit trail of all operations
  data_retention: Configurable audit log retention
```

## 🧪 Testing Strategy

### Test Categories
```yaml
unit_tests:
  location: Co-located with source files (lib/*.bats)
  coverage: Individual function testing
  framework: BATS (Bash Automated Testing System)
  
integration_tests:
  location: test/ directory
  coverage: API operations, secret CRUD, seal/unseal
  test_data: Uses shared fixtures from __test/fixtures/data/
  test_scenarios:
    - Vault initialization and unsealing
    - Secret storage and retrieval
    - Token authentication
    - Policy enforcement
  
system_tests:
  location: __test/resources/vault/
  coverage: Full resource lifecycle, HA failover
  automation: Integrated with Vrooli test framework
  
performance_tests:
  load_testing: Secret operations under load
  stress_testing: Memory limits with many secrets
  endurance_testing: Token renewal over time
```

### Test Specifications
```yaml
test_specification:
  resource_name: vault
  test_categories: [unit, integration, system, performance]
  
  test_structure:
    - BATS files co-located with source files
    - Integration tests in test/ directory
    - Shared fixtures from __test/fixtures/data/
    - Test results included in status output with timestamp
    - Examples in examples/ directory
  
  lifecycle_tests:
    - name: "Vault Installation"
      command: resource-vault install
      expect:
        exit_code: 0
        service_running: true
        health_status: healthy
        
    - name: "Vault Status"
      command: resource-vault status --json
      expect:
        exit_code: 0
        json_valid: true
        health_status: healthy
        sealed: false
        
    - name: "Secret Management"
      command: resource-vault content add --path test --data '{"key":"value"}'
      expect:
        exit_code: 0
        secret_stored: true
        
  performance_tests:
    - name: "Service Startup Time"
      measurement: startup_latency
      target: < 5 seconds
      
    - name: "Secret Access Latency"
      measurement: api_response_time
      target: < 50ms
      
  failure_tests:
    - name: "Seal Recovery"
      scenario: Vault gets sealed
      expect: Can unseal with proper keys
```

## 💰 Infrastructure Value

### Technical Value
- **Infrastructure Stability**: Eliminates hardcoded secrets reducing security incidents
- **Integration Simplicity**: Unified secret API for all resources
- **Operational Efficiency**: Automated credential rotation and audit trails
- **Developer Experience**: Simple CLI and API for secret management

### Resource Economics
- **Setup Cost**: 5 minutes for dev mode, 30 minutes for production
- **Operating Cost**: Minimal (< 256MB RAM, < 1% CPU)
- **Integration Value**: Multiplies security posture of entire platform
- **Maintenance Overhead**: Near-zero after initial setup

## 🔄 Resource Lifecycle Integration

### Vrooli Integration Standards
```yaml
resource_discovery:
  registry_entry:
    name: vault
    category: storage
    capabilities: [secrets-management, encryption, audit, dynamic-secrets]
    interfaces:
      - cli: resource-vault (installed via install-resource-cli.sh)
      - api: http://localhost:8200/v1/
      - health: http://localhost:8200/v1/sys/health
      
  metadata:
    description: HashiCorp Vault secrets management
    version: 1.17
    dependencies: [docker]
    enables: [secure-credentials, multi-tenancy, compliance]

resource_framework_compliance:
  - Standard directory structure (/config, /lib, /docs, /test, /examples)
  - CLI framework integration (cli.sh as thin wrapper)
  - Port registry integration  
  - Docker network integration
  - Health monitoring integration
  - Configuration management standards
  
deployment_integration:
  supported_targets:
    - local: Docker container deployment
    - kubernetes: Helm chart available
    - cloud: Vault Cloud or self-managed
    
  configuration_management:
    - Environment-based configuration
    - Template-based setup (dev/prod/test)
    - Self-bootstrapping secret management
```

### Version Management
```yaml
versioning:
  current: 1.17
  compatibility: Backward compatible within 1.x
  upgrade_path: Rolling upgrade supported
  
  breaking_changes: []
  deprecations: []
  migration_guide: See docs/migration.md

release_management:
  release_cycle: Monthly from HashiCorp
  testing_requirements: Integration tests before upgrade
  rollback_strategy: Snapshot restore
```

## 🧬 Evolution Path

### Version 1.17 (Current)
- KV v2 secrets engine
- Dev mode for local development
- Basic policy management
- Token authentication

### Version 2.0 (Planned)
- Production mode with HA
- Dynamic database credentials
- PKI certificate management
- OIDC authentication

### Long-term Vision
- Kubernetes native integration
- Zero-trust service mesh
- Quantum-resistant encryption
- AI-driven anomaly detection

## 🚨 Risk Mitigation

### Technical Risks
| Risk | Probability | Impact | Mitigation |
|------|------------|--------|------------|
| Token exposure | Low | Critical | Short TTL, audit logging |
| Seal event | Low | High | Automated unseal, key escrow |
| Data loss | Low | High | Regular backups, replication |
| Performance degradation | Low | Medium | Caching, connection pooling |

### Operational Risks
- **Configuration Drift**: Managed through IaC and config templates
- **Dependency Failures**: Graceful degradation to cached credentials
- **Resource Conflicts**: Port registry prevents conflicts
- **Update Compatibility**: Version pinning and testing

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
- [x] Configuration management functions properly
- [x] Monitoring and alerting work as expected

### Operational Validation
- [x] Deployment procedures documented and tested
- [ ] Backup and recovery procedures verified
- [ ] Upgrade and rollback procedures validated
- [ ] Troubleshooting documentation complete
- [x] Performance under load verified

## 📝 Implementation Notes

### Design Decisions
**Dev Mode by Default**: Simplified developer experience
- Alternative considered: Production mode
- Decision driver: Lower barrier to entry
- Trade-offs: Less secure but faster onboarding

**Token Authentication**: Simple and secure
- Alternative considered: Username/password
- Decision driver: Better for automation
- Trade-offs: Token management complexity

### Known Limitations
- **Dev Mode Storage**: In-memory only
  - Workaround: Use file backend for persistence
  - Future fix: Production mode template
  
- **Single Node**: No HA in current setup
  - Workaround: Regular backups
  - Future fix: Consul backend for HA

### Integration Considerations
- **Token Distribution**: Use environment variables
- **Network Security**: Restrict to vrooli-network
- **Performance Impact**: Minimal with caching
- **Scaling Considerations**: Move to HA for production

## 🔗 References

### Documentation
- README.md - Quick start and overview
- docs/api.md - API reference
- docs/policies.md - Policy management
- config/defaults.sh - Configuration options
- lib/api.sh - API implementation

### Related Resources
- postgres - Can use as storage backend
- redis - HA coordination backend
- openrouter - Primary consumer of API keys

### External Resources
- [Official Vault Documentation](https://developer.hashicorp.com/vault/docs)
- [Vault Best Practices](https://developer.hashicorp.com/vault/docs/internals/security)
- [Learn Vault](https://developer.hashicorp.com/vault/tutorials)

---

**Last Updated**: 2025-08-22
**Status**: Draft
**Owner**: Vrooli Platform Team
**Review Cycle**: Monthly