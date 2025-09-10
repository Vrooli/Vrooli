# Product Requirements Document (PRD) - Step-CA

## 🎯 Infrastructure Definition

### Core Infrastructure Capability
**What permanent infrastructure capability does this resource add to Vrooli?**
Step-CA provides a modern, open-source private certificate authority (CA) that enables automated certificate lifecycle management through the ACME protocol. It serves as the foundation for secure internal communications, mutual TLS authentication, and zero-trust architectures within the Vrooli ecosystem.

### System Amplification
**How does this resource make the entire Vrooli system more capable?**
- **Security Infrastructure**: Provides automated X.509 and SSH certificate management for all internal services
- **Zero-Trust Foundation**: Enables mutual TLS authentication between resources and scenarios
- **Automated Certificate Lifecycle**: Eliminates manual certificate management through ACME protocol
- **Identity Management**: Integrates with OIDC providers for certificate-based identity verification
- **Compliance Enablement**: Meets enterprise PKI requirements for regulated environments

### Enabling Value
**What new scenarios become possible when this resource is available?**
1. **Secure Service Mesh**: Automatic mTLS for all inter-service communication
2. **Enterprise Authentication**: Certificate-based authentication for enterprise scenarios
3. **IoT Security**: Device certificate management for IoT scenarios
4. **DevOps Automation**: Automated certificate provisioning for CI/CD pipelines
5. **Multi-Tenant Security**: Isolated certificate namespaces for tenant isolation

## 📊 Infrastructure Metrics

### Functional Requirements
- **Must Have (P0)**
  - [ ] ACME protocol server for automated certificate enrollment and renewal
  - [ ] Support for X.509 and SSH certificate generation
  - [ ] Multiple authentication methods (OIDC, tokens, cloud APIs)
  - [ ] Configurable certificate lifetime and renewal policies
  - [ ] Health monitoring endpoint returning CA status
  - [ ] Standard CLI interface (resource-step-ca)
  - [ ] Docker containerization with persistent storage
  
- **Should Have (P1)**
  - [ ] HSM/KMS integration for secure key storage
  - [ ] Certificate revocation and CRL management
  - [ ] Multiple database backends (Badger, BoltDB, PostgreSQL)
  - [ ] Custom certificate templates and profiles
  - [ ] Audit logging for all certificate operations
  
- **Nice to Have (P2)**
  - [ ] Web UI for certificate management
  - [ ] Prometheus metrics export
  - [ ] Active Directory integration

### Performance Criteria
| Metric | Target | Measurement Method |
|--------|--------|-------------------|
| Service Startup Time | < 10s | Container initialization |
| Health Check Response | < 500ms | API health endpoint |
| Certificate Issuance | < 2s | ACME enrollment time |
| Resource Utilization | < 20% CPU/512MB Memory | Resource monitoring |
| Availability | > 99.9% uptime | Service monitoring |

### Quality Gates
- [ ] All P0 requirements implemented and tested
- [ ] ACME protocol compliance verified
- [ ] Certificate generation and renewal working
- [ ] Security standards met for CA operations
- [ ] Documentation complete with examples

## 🏗️ Technical Architecture

### Resource Dependencies
```yaml
required:
  - resource_name: postgres
    purpose: Certificate and configuration storage
    integration_pattern: Database backend for persistent data
    access_method: Direct database connection
    
optional:
  - resource_name: vault
    purpose: HSM integration for root key protection
    fallback: Use local key storage
    access_method: API integration
    
  - resource_name: keycloak
    purpose: OIDC provider for authentication
    fallback: Use token-based authentication
    access_method: OIDC protocol
```

### Integration Standards
```yaml
resource_category: security

standard_interfaces:
  management:
    - cli: cli.sh (v2.0 compliant)
    - actions: [help, manage, test, content, status, logs, credentials]
    - configuration: config/defaults.sh
    - documentation: README.md + docs/
    
  networking:
    - docker_networks: [vrooli-network]
    - port_registry: Ports from scripts/resources/port_registry.sh
    - hostname: vrooli-step-ca
    
  monitoring:
    - health_check: GET /health
    - status_reporting: resource-step-ca status
    - logging: Docker container logs
    
  data_persistence:
    - volumes: [step-ca-data, step-ca-config, step-ca-certs]
    - backup_strategy: Database and certificate backup
    - migration_support: Version upgrade path

integration_patterns:
  scenarios_using_resource:
    - scenario_name: api-gateway
      usage_pattern: mTLS for API authentication
      
    - scenario_name: microservices-mesh
      usage_pattern: Service-to-service certificates
      
  resource_to_resource:
    - postgres → step-ca: Certificate storage backend
    - step-ca → all-resources: Certificate provisioning
```

### Configuration Schema
```yaml
resource_configuration:
  defaults:
    enabled: true
    port: [from port_registry.sh]
    networks: [vrooli-network]
    volumes: [step-ca-data, step-ca-config]
    environment:
      - DOCKER_STEPCA_INIT_NAME: "Vrooli CA"
      - DOCKER_STEPCA_INIT_DNS_NAMES: "localhost,vrooli-step-ca"
      - DOCKER_STEPCA_INIT_PROVISIONER_NAME: "admin"
      - DOCKER_STEPCA_INIT_PASSWORD: "changeme"
    
  templates:
    development:
      - description: Short-lived certificates for development
      - overrides:
          default_cert_duration: "24h"
          max_cert_duration: "72h"
      
    production:
      - description: Production-grade certificate policies
      - overrides:
          default_cert_duration: "720h"
          max_cert_duration: "8760h"
          require_oidc: true
      
    testing:
      - description: Instant certificates for testing
      - overrides:
          default_cert_duration: "1h"
          skip_validation: true
      
customization:
  user_configurable:
    - parameter: default_cert_duration
      description: Default certificate lifetime
      default: "24h"
      
    - parameter: provisioner_type
      description: Authentication method (JWK, OIDC, ACME)
      default: "JWK"
      
  environment_variables:
    - var: STEPCA_PASSWORD
      purpose: Admin password for CA operations
      
    - var: STEPCA_ROOT_CERT
      purpose: Path to root certificate
```

### API Contract
```yaml
api_endpoints:
  - method: GET
    path: /health
    purpose: CA health and readiness check
    output_schema: |
      {
        "status": "ok",
        "ca_status": "running",
        "certificate_count": 42
      }
    authentication: none
    
  - method: POST
    path: /acme/new-account
    purpose: ACME account registration
    input_schema: |
      {
        "contact": ["mailto:admin@example.com"],
        "termsOfServiceAgreed": true
      }
    authentication: ACME protocol
    
  - method: POST
    path: /1.0/sign
    purpose: Certificate signing request
    input_schema: |
      {
        "csr": "base64-encoded-csr",
        "ott": "one-time-token"
      }
    authentication: Token or OIDC
```

## 🖥️ Management Interface Contract

### Required Management Actions
```yaml
standard_actions:
  - name: install
    description: Install Step-CA and initialize root certificate
    flags: [--force, --template <name>]
    
  - name: start  
    description: Start Step-CA service
    flags: [--wait]
    
  - name: stop
    description: Stop Step-CA service gracefully
    flags: [--force]
    
  - name: status
    description: Show CA status and certificate statistics
    flags: [--json, --verbose]
    
  - name: uninstall
    description: Remove Step-CA and optionally preserve certificates
    flags: [--keep-data, --force]

resource_specific_actions:
  - name: init-ca
    description: Initialize new certificate authority
    flags: [--name, --dns, --address, --provisioner]
    example: resource-step-ca init-ca --name "Vrooli CA"
    
  - name: add-provisioner
    description: Add authentication provisioner
    flags: [--type, --name, --client-id]
    example: resource-step-ca add-provisioner --type OIDC --name keycloak
    
  - name: issue-cert
    description: Issue certificate manually
    flags: [--cn, --san, --duration]
    example: resource-step-ca issue-cert --cn service.local --duration 30d
```

### Management Standards
```yaml
implementation_requirements:
  - cli_location: cli.sh (v2.0 compliant)
  - configuration: config/defaults.sh
  - dependencies: lib/ directory with modular functions
  - error_handling: Exit codes (0=success, 1=error, 2=config error)
  - logging: Structured output with levels (INFO, WARN, ERROR)
  - idempotency: Safe to run commands multiple times
  
status_reporting:
  - health_status: healthy|degraded|unhealthy|unknown
  - service_info: version, uptime, certificates issued
  - integration_status: database connectivity
  - configuration: current CA configuration
  
output_formats:
  - default: Human-readable with color coding
  - json: Structured data with --json flag
  - verbose: Detailed diagnostics with --verbose
```

## 🔧 Operational Requirements

### Deployment Standards
```yaml
containerization:
  base_image: smallstep/step-ca:latest
  dockerfile_location: Not needed (official image)
  
networking:
  required_networks:
    - vrooli-network: Primary inter-resource communication
    
  port_allocation:
    - internal: 9000
    - external: [from port_registry.sh]
    - protocol: tcp
    - purpose: HTTPS API and ACME server
    
data_management:
  persistence:
    - volume: step-ca-data
      mount: /home/step
      purpose: CA database and certificates
      
    - volume: step-ca-config
      mount: /home/step/config
      purpose: CA configuration
      
  backup_strategy:
    - method: Volume snapshots
    - frequency: Daily
    - retention: 30 days
    
  migration_support:
    - version_compatibility: Backward compatible
    - upgrade_path: In-place upgrade
    - rollback_support: Volume restore
```

### Performance Standards
```yaml
resource_requirements:
  minimum:
    cpu: 0.5 cores
    memory: 256MB
    disk: 1GB
    
  recommended:
    cpu: 1 core
    memory: 512MB
    disk: 10GB
    
  scaling:
    horizontal: Multiple CAs with shared backend
    vertical: Scales with certificate volume
    limits: 100K certificates per instance
    
monitoring_requirements:
  health_checks:
    - endpoint: http://localhost:9000/health
    - interval: 30s
    - timeout: 5s
    - failure_threshold: 3
    
  metrics:
    - metric: certificate_count
      collection: API query
      alerting: >90% of limit
      
    - metric: certificate_expiry
      collection: Database query
      alerting: <7 days to expiry
```

### Security Standards
```yaml
security_requirements:
  authentication:
    - method: Multiple (JWK, OIDC, ACME, Cloud)
    - credential_storage: Encrypted in database
    - session_management: Token-based with expiry
    
  authorization:
    - access_control: Provisioner-based
    - role_based: Admin vs user provisioners
    - resource_isolation: Namespace separation
    
  data_protection:
    - encryption_at_rest: Database encryption
    - encryption_in_transit: TLS 1.3
    - key_management: HSM/KMS optional
    
  network_security:
    - port_exposure: HTTPS API only
    - firewall_requirements: Inbound 9000/tcp
    - ssl_tls: Self-signed root certificate
    
compliance:
  standards: [PKI best practices, RFC 5280]
  auditing: All certificate operations logged
  data_retention: Configurable retention policies
```

## 🧪 Testing Strategy

### Test Categories
```yaml
unit_tests:
  location: lib/*.bats co-located with scripts
  coverage: Individual function testing
  framework: BATS
  
integration_tests:
  location: test/phases/
  coverage: ACME protocol, certificate issuance
  test_scenarios: 
    - CA initialization
    - Certificate enrollment via ACME
    - Certificate renewal
    - Provisioner management
  
system_tests:
  location: test/
  coverage: Full lifecycle, mTLS verification
  automation: Integrated with Vrooli test framework
  
performance_tests:
  load_testing: 100 certificates/minute
  stress_testing: 10K active certificates
  endurance_testing: 30-day certificate lifecycle
```

### Test Specifications
```yaml
test_specification:
  resource_name: step-ca
  test_categories: [unit, integration, system]
  
  lifecycle_tests:
    - name: "CA Installation"
      command: resource-step-ca manage install
      expect:
        exit_code: 0
        service_running: true
        health_status: healthy
        
    - name: "Certificate Issuance"
      command: resource-step-ca content add --type cert --cn test.local
      expect:
        exit_code: 0
        certificate_created: true
        
    - name: "ACME Enrollment"
      command: certbot certonly --server http://localhost:9000/acme/acme/directory
      expect:
        exit_code: 0
        certificate_issued: true
        
  performance_tests:
    - name: "Service Startup Time"
      measurement: startup_latency
      target: < 10 seconds
      
    - name: "Certificate Issuance Rate"
      measurement: certificates_per_minute
      target: > 100
```

## 💰 Infrastructure Value

### Technical Value
- **Security Enhancement**: Automated PKI eliminates manual certificate management errors
- **Zero-Trust Enablement**: Foundation for service mesh and microsegmentation
- **Compliance Support**: Meets enterprise PKI requirements for regulated industries
- **Developer Productivity**: Self-service certificate provisioning via ACME

### Resource Economics
- **Setup Cost**: 30 minutes initial configuration
- **Operating Cost**: Minimal (256MB RAM, 0.5 CPU)
- **Integration Value**: Enables $100K+ in secure scenario value
- **Maintenance Overhead**: Automated renewal reduces ongoing work

## 🔄 Resource Lifecycle Integration

### Vrooli Integration Standards
```yaml
resource_discovery:
  registry_entry:
    name: step-ca
    category: security
    capabilities: [PKI, ACME, X.509, SSH, mTLS]
    interfaces:
      - cli: resource-step-ca
      - api: https://localhost:9000
      - health: /health
      
  metadata:
    description: Private certificate authority with ACME support
    version: 0.25.0
    dependencies: [postgres]
    enables: [secure service mesh, mTLS, zero-trust]

resource_framework_compliance:
  - v2.0 contract compliance
  - Standard directory structure
  - CLI framework integration
  - Port registry integration
  - Docker network integration
  - Health monitoring
  
deployment_integration:
  supported_targets:
    - local: Docker Compose
    - kubernetes: Helm chart available
    - cloud: AWS KMS, Google Cloud KMS support
```

### Version Management
```yaml
versioning:
  current: 0.25.0
  compatibility: Backward compatible to 0.20.x
  upgrade_path: In-place upgrade supported
  
  breaking_changes: None in 0.25.x series
  deprecations: Legacy provisioner API in 0.26.0
  migration_guide: Update provisioner configuration

release_management:
  release_cycle: Monthly
  testing_requirements: ACME compliance tests
  rollback_strategy: Volume restore from backup
```

## 🧬 Evolution Path

### Version 0.25.0 (Current)
- ACME protocol support
- Multiple provisioner types
- Basic HSM integration
- PostgreSQL backend support

### Version 0.26.0 (Planned)
- Enhanced OIDC integration
- Certificate transparency logs
- Improved revocation handling
- Prometheus metrics export

### Long-term Vision
- Full HSM/KMS integration
- Multi-region CA federation
- Quantum-resistant algorithms
- Automated compliance reporting

## 🚨 Risk Mitigation

### Technical Risks
| Risk | Probability | Impact | Mitigation |
|------|------------|--------|------------|
| Root key compromise | Low | Critical | HSM integration, key rotation |
| Service availability | Low | High | Database replication, HA mode |
| Certificate expiry | Medium | Medium | Automated renewal, monitoring |
| Performance degradation | Low | Low | Database indexing, caching |

### Operational Risks
- **Configuration Drift**: Version-controlled configuration with validation
- **Dependency Failures**: Graceful degradation without database
- **Certificate Sprawl**: Automated cleanup of expired certificates
- **Update Compatibility**: Staged rollout with validation

## ✅ Validation Criteria

### Infrastructure Validation
- [ ] Step-CA installs and starts successfully
- [ ] Root certificate generated and stored
- [ ] ACME server responds to requests
- [ ] Health endpoint returns status
- [ ] CLI commands function correctly
- [ ] Documentation complete with examples

### Integration Validation  
- [ ] PostgreSQL backend connection works
- [ ] ACME protocol compliance verified
- [ ] Certificate issuance successful
- [ ] Renewal automation functioning
- [ ] mTLS verification between services

### Operational Validation
- [ ] Backup and restore procedures work
- [ ] Certificate lifecycle managed properly
- [ ] Performance meets targets
- [ ] Security requirements satisfied
- [ ] Monitoring and alerting functional

## 📝 Implementation Notes

### Design Decisions
**ACME-First Approach**: Prioritize ACME for automation
- Alternative considered: Manual certificate management
- Decision driver: Automation and standard compliance
- Trade-offs: Complexity vs automation benefits

**PostgreSQL Backend**: Use PostgreSQL for production
- Alternative considered: BoltDB embedded database  
- Decision driver: Scalability and backup capabilities
- Trade-offs: External dependency vs reliability

### Known Limitations
- **Single Root CA**: No intermediate CA support yet
  - Workaround: Use separate Step-CA instances
  - Future fix: Intermediate CA support in v0.27.0
  
- **No CRL Distribution**: Revocation lists not distributed
  - Workaround: Short certificate lifetimes
  - Future fix: CRL/OCSP support planned

### Integration Considerations
- **Database Backend**: PostgreSQL recommended for production
- **Root Certificate Distribution**: Must distribute to all clients
- **ACME Clients**: Support standard ACME clients (certbot, acme.sh)
- **HSM Integration**: Optional but recommended for production

## 🔗 References

### Documentation
- README.md - Quick start and configuration
- docs/ACME.md - ACME protocol implementation
- docs/PROVISIONERS.md - Authentication methods
- config/defaults.sh - Configuration options

### Related Resources
- postgres - Database backend for certificates
- vault - HSM integration for key protection
- keycloak - OIDC provider for authentication

### External Resources
- [Step-CA Documentation](https://smallstep.com/docs/step-ca)
- [ACME Protocol RFC 8555](https://tools.ietf.org/html/rfc8555)
- [PKI Best Practices](https://nvlpubs.nist.gov/nistpubs/SpecialPublications/NIST.SP.800-57pt1r5.pdf)

---

**Last Updated**: 2025-01-10  
**Status**: Draft  
**Owner**: Ecosystem Manager  
**Review Cycle**: Monthly