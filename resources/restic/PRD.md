# Product Requirements Document (PRD) - Restic

## 🎯 Infrastructure Definition

### Core Infrastructure Capability
**What permanent infrastructure capability does this resource add to Vrooli?**
Restic provides enterprise-grade, encrypted backup and recovery services with client-side encryption, efficient deduplication, and incremental snapshots. It enables automated, ransomware-resistant protection for all Vrooli resources, scenarios, and data.

### System Amplification
**How does this resource make the entire Vrooli system more capable?**
- **Data Protection**: Automatic, encrypted backups of all critical Vrooli data including scenarios, databases, and configurations
- **Disaster Recovery**: Rapid restoration capabilities enable quick recovery from failures, attacks, or data corruption
- **Version Control**: Snapshot-based backups provide point-in-time recovery for any resource or scenario
- **Resource Independence**: Works with any storage backend (local, S3, MinIO, SFTP) giving flexibility in backup strategies
- **Developer Confidence**: Developers can experiment freely knowing rollback is always possible

### Enabling Value
**What new scenarios become possible when this resource is available?**
1. **Production Scenarios**: Safe deployment of revenue-generating scenarios with guaranteed rollback capability
2. **Compliance Scenarios**: Audit-compliant data retention and recovery for regulated industries
3. **Multi-Environment Scenarios**: Clone production data to development/testing environments safely
4. **Ransomware Protection**: Immutable, versioned backups protect against ransomware attacks
5. **Cross-Region Replication**: Geographic redundancy for business continuity planning

## 📊 Infrastructure Metrics

### Functional Requirements
- **Must Have (P0)**
  - [x] Client-side AES-256 encryption for all backups ✅ 2025-01-10
  - [x] Automated daily backup scheduling for critical resources ✅ 2025-01-10
  - [x] Incremental snapshot-based backups with deduplication ✅ 2025-01-10
  - [x] Integration with Vrooli resource framework ✅ 2025-01-10
  - [x] Standard CLI interface (resource-restic) ✅ 2025-01-10
  - [x] Health monitoring and backup status reporting ✅ 2025-01-10
  - [x] Docker containerization with persistent repository storage ✅ 2025-01-10
  
- **Should Have (P1)**
  - [x] Content management commands (add/list/get/remove/execute) ✅ 2025-09-16
  - [x] Multi-backend support (S3, MinIO, local filesystem, SFTP, REST) ✅ 2025-09-16
  - [x] Retention policy management (keep-daily/weekly/monthly) ✅ 2025-09-16
  - [x] Backup verification and integrity checking ✅ 2025-09-16
  - [x] Resource-specific backup hooks for databases (PostgreSQL, Redis, MinIO) ✅ 2025-09-16
  
- **Nice to Have (P2)**
  - [ ] Backup encryption key rotation
  - [ ] Bandwidth throttling for network backups
  - [ ] Integration with monitoring systems for alerts

### Performance Criteria
| Metric | Target | Measurement Method |
|--------|--------|-------------------|
| Service Startup Time | < 10s | Container initialization |
| Health Check Response | < 500ms | API/CLI status checks |
| Backup Speed | > 50MB/s | Local filesystem backups |
| Deduplication Ratio | > 10:1 | Typical scenario data |
| Restore Speed | > 100MB/s | Local filesystem restore |

### Quality Gates
- [x] All P0 requirements implemented and tested ✅ 2025-01-10
- [x] Backup and restore operations verified ✅ 2025-01-10
- [x] Encryption validated with test data ✅ 2025-01-10
- [x] Performance targets met for local backups ✅ 2025-01-10
- [ ] Documentation complete with recovery procedures

## 🏗️ Technical Architecture

### Resource Dependencies
```yaml
required:
  - resource_name: minio
    purpose: Primary backup storage backend
    integration_pattern: S3-compatible API for remote backups
    access_method: API
    
optional:
  - resource_name: postgres
    purpose: Backup target for database dumps
    integration_pattern: Pre-backup hooks for consistent snapshots
    access_method: CLI
    
  - resource_name: vault
    purpose: Secure storage of backup encryption keys
    fallback: Local encrypted key storage
    access_method: API
```

### Integration Standards
```yaml
resource_category: infrastructure

standard_interfaces:
  management:
    - cli: cli.sh (using CLI framework)
    - actions: [help, install, uninstall, start, stop, restart, status, validate, test, content]
    - configuration: config/defaults.sh
    - documentation: README.md + docs/
    
  networking:
    - docker_networks: [vrooli-network]
    - port_registry: Ports defined in scripts/resources/port_registry.sh only
    - hostname: vrooli-restic
    
  monitoring:
    - health_check: HTTP endpoint on port from registry
    - status_reporting: resource-restic status
    - logging: Docker container logs
    
  data_persistence:
    - volumes: 
      - restic-repository: /repository
      - restic-cache: /cache
      - restic-config: /config
    - backup_strategy: Repository is the backup itself
    - migration_support: Restic handles version compatibility

integration_patterns:
  scenarios_using_resource:
    - scenario_name: disaster-recovery-orchestrator
      usage_pattern: Automated recovery workflows
      
    - scenario_name: compliance-auditor
      usage_pattern: Retention policy enforcement
      
  resource_to_resource:
    - postgres → restic: Database dumps before backup
    - restic → minio: Store backup repositories
    - vault → restic: Encryption key management
```

### Configuration Schema
```yaml
resource_configuration:
  defaults:
    enabled: true
    port: [retrieve from port_registry.sh]
    networks: [vrooli-network]
    volumes: 
      - restic-repository:/repository
      - restic-cache:/cache
    environment:
      - RESTIC_REPOSITORY=/repository
      - RESTIC_CACHE_DIR=/cache
      - RESTIC_COMPRESSION=auto
    
  templates:
    development:
      - description: Local filesystem backups only
      - overrides:
          backup_schedule: "0 2 * * *"  # Daily at 2 AM
          retention_days: 7
      
    production:
      - description: Remote S3 backups with encryption
      - overrides:
          backup_schedule: "0 */6 * * *"  # Every 6 hours
          retention_days: 30
          retention_weeks: 12
          retention_months: 12
      
    testing:
      - description: Minimal retention for testing
      - overrides:
          backup_schedule: "*/15 * * * *"  # Every 15 minutes
          retention_days: 1
      
customization:
  user_configurable:
    - parameter: backup_schedule
      description: Cron expression for backup frequency
      default: "0 2 * * *"
      
    - parameter: repository_backend
      description: Storage backend (local/s3/sftp)
      default: local
      
    - parameter: retention_policy
      description: How long to keep backups
      default: "7d 4w 12m"
      
  environment_variables:
    - var: RESTIC_PASSWORD
      purpose: Repository encryption password
      
    - var: AWS_ACCESS_KEY_ID
      purpose: S3 backend authentication
      
    - var: AWS_SECRET_ACCESS_KEY
      purpose: S3 backend authentication
```

### API Contract
```yaml
api_endpoints:
  - method: GET
    path: /health
    purpose: Health check endpoint
    output_schema: |
      {
        "status": "healthy|degraded|unhealthy",
        "repository_status": "initialized|locked|error",
        "last_backup": "ISO-8601 timestamp",
        "snapshots_count": "integer"
      }
    authentication: none
    
  - method: POST
    path: /api/backup
    purpose: Trigger manual backup
    input_schema: |
      {
        "paths": ["array of paths to backup"],
        "tags": ["optional tags"],
        "exclude": ["optional exclusions"]
      }
    output_schema: |
      {
        "snapshot_id": "string",
        "files_new": "integer",
        "files_changed": "integer",
        "bytes_added": "integer"
      }
    authentication: bearer token
    
  - method: POST
    path: /api/restore
    purpose: Restore from backup
    input_schema: |
      {
        "snapshot_id": "string or latest",
        "target_path": "restoration path",
        "include": ["optional file patterns"]
      }
    output_schema: |
      {
        "files_restored": "integer",
        "bytes_restored": "integer",
        "duration_seconds": "float"
      }
    authentication: bearer token
```

## 🖥️ Management Interface Contract

### Required Management Actions
```yaml
standard_actions:
  - name: install
    description: Install restic and initialize repository
    flags: [--force, --backend <type>, --repository <path>]
    
  - name: start  
    description: Start restic service and scheduler
    flags: [--wait]
    
  - name: stop
    description: Stop restic service gracefully
    flags: [--force]
    
  - name: status
    description: Show backup status and statistics
    flags: [--json, --verbose]
    
  - name: uninstall
    description: Remove restic (keeps backup repository)
    flags: [--keep-repository, --force]

resource_specific_actions:
  - name: backup
    description: Trigger immediate backup
    flags: [--paths <paths>, --tags <tags>]
    example: resource-restic backup --paths /data --tags daily
    
  - name: restore
    description: Restore from snapshot
    flags: [--snapshot <id>, --target <path>]
    example: resource-restic restore --snapshot latest --target /restore
    
  - name: snapshots
    description: List available snapshots
    flags: [--tags <tags>, --json]
    example: resource-restic snapshots --tags postgres
    
  - name: prune
    description: Remove old snapshots per retention policy
    flags: [--dry-run, --keep-daily <n>]
    example: resource-restic prune --keep-daily 7
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
  - repository_info: size, snapshot count, last backup time
  - backup_statistics: deduplication ratio, compression ratio
  - scheduled_backups: next run time, schedule status
  
output_formats:
  - default: Human-readable with color coding
  - json: Structured data with --json flag
  - verbose: Detailed diagnostics with --verbose
```

## 🔧 Operational Requirements

### Deployment Standards
```yaml
containerization:
  base_image: restic/restic:latest
  dockerfile_location: docker/Dockerfile (if customization needed)
  build_requirements: none (official image sufficient)
  
networking:
  required_networks:
    - vrooli-network: Inter-resource communication
    
  port_allocation:
    - internal: 8000
    - external: [retrieved from port_registry.sh]
    - protocol: tcp
    - purpose: REST API and health checks
    - registry_integration:
        definition: Port defined in scripts/resources/port_registry.sh
        retrieval: Use port_registry functions
        no_hardcoding: Never hardcode ports
    
data_management:
  persistence:
    - volume: restic-repository
      mount: /repository
      purpose: Backup repository storage
      
    - volume: restic-cache
      mount: /cache
      purpose: Performance cache
      
    - volume: restic-config
      mount: /config
      purpose: Configuration and keys
      
  backup_strategy:
    - method: Repository is self-contained
    - frequency: Not applicable (is the backup)
    - retention: User-defined policy
    
  migration_support:
    - version_compatibility: Restic handles forward compatibility
    - upgrade_path: Automatic migration on first use
    - rollback_support: Keep old binary for emergency
```

### Performance Standards
```yaml
resource_requirements:
  minimum:
    cpu: 0.5 cores
    memory: 512MB
    disk: 1GB + repository size
    
  recommended:
    cpu: 2 cores
    memory: 2GB
    disk: 10GB + 2x data size
    
  scaling:
    horizontal: Not applicable (single repository)
    vertical: More CPU/RAM improves compression/encryption
    limits: Repository size limited by storage backend
    
monitoring_requirements:
  health_checks:
    - endpoint: http://localhost:${PORT}/health
    - interval: 60s
    - timeout: 5s
    - failure_threshold: 3
    
  metrics:
    - metric: backup_success_rate
      collection: Parse restic output
      alerting: Alert if <95%
      
    - metric: repository_size
      collection: du -sh /repository
      alerting: Alert if >80% disk
      
    - metric: last_backup_age
      collection: Check snapshot timestamps
      alerting: Alert if >25 hours old
```

### Security Standards
```yaml
security_requirements:
  authentication:
    - method: Repository password
    - credential_storage: Environment variable or Vault
    - session_management: Not applicable
    
  authorization:
    - access_control: Repository-level encryption
    - role_based: Not applicable
    - resource_isolation: Separate repositories per tenant
    
  data_protection:
    - encryption_at_rest: AES-256-GCM client-side
    - encryption_in_transit: TLS for remote backends
    - key_management: PBKDF2 key derivation
    
  network_security:
    - port_exposure: API port for health/management only
    - firewall_requirements: Outbound to backup destinations
    - ssl_tls: Required for S3/remote backends
    
compliance:
  standards: [GDPR data protection, HIPAA for healthcare data]
  auditing: All backup/restore operations logged
  data_retention: Configurable retention policies
```

## 🧪 Testing Strategy

### Test Categories
```yaml
unit_tests:
  location: lib/ directory (e.g., lib/backup.bats)
  coverage: Individual function testing
  framework: BATS
  
integration_tests:
  location: test/ directory
  coverage: Backup/restore workflows
  test_data: Sample files from __test/fixtures/data/
  test_scenarios: 
    - Initialize repository
    - Create backup
    - List snapshots
    - Restore files
    - Prune old snapshots
  
system_tests:
  location: __test/resources/
  coverage: Full backup/restore cycle
  automation: Integrated with Vrooli test framework
  
performance_tests:
  - backup_speed: Test with 1GB data
  - deduplication: Test with duplicate data
  - compression: Test with various file types
```

### Test Specifications
```yaml
test_specification:
  resource_name: restic
  test_categories: [unit, integration, system, performance]
  
  lifecycle_tests:
    - name: "Repository Initialization"
      command: resource-restic install
      expect:
        exit_code: 0
        repository_created: true
        health_status: healthy
        
    - name: "Create Backup"  
      command: resource-restic backup --paths /test/data
      expect:
        exit_code: 0
        snapshot_created: true
        files_backed_up: true
        
    - name: "Restore Backup"
      command: resource-restic restore --snapshot latest --target /tmp/restore
      expect:
        exit_code: 0
        files_restored: true
        data_integrity: verified
        
  performance_tests:
    - name: "Backup Speed"
      measurement: throughput_mbps
      target: > 50 MB/s
      
    - name: "Deduplication Ratio"
      measurement: storage_efficiency
      target: > 10:1 for similar data
      
  failure_tests:
    - name: "Repository Lock Handling"
      scenario: Concurrent backup attempts
      expect: Second backup waits or fails gracefully
      
    - name: "Network Interruption"
      scenario: Connection lost during backup
      expect: Resume on reconnection
```

## 💰 Infrastructure Value

### Technical Value
- **Infrastructure Stability**: Enables rapid recovery from any failure, minimizing downtime
- **Integration Simplicity**: Simple API for any resource to trigger backups
- **Operational Efficiency**: Automated backup scheduling reduces manual intervention
- **Developer Experience**: Confidence to experiment knowing rollback is available

### Resource Economics
- **Setup Cost**: ~30 minutes to configure and initialize
- **Operating Cost**: Minimal CPU/memory, storage costs for repositories
- **Integration Value**: Protects all high-value scenario data and configurations
- **Maintenance Overhead**: Automated pruning minimizes manual maintenance

## 🔄 Resource Lifecycle Integration

### Vrooli Integration Standards
```yaml
resource_discovery:
  registry_entry:
    name: restic
    category: infrastructure
    capabilities: [backup, restore, encryption, deduplication, snapshots]
    interfaces:
      - cli: resource-restic
      - api: http://vrooli-restic:8000
      - health: http://vrooli-restic:8000/health
      
  metadata:
    description: Enterprise-grade encrypted backup and recovery
    version: 0.16.4
    dependencies: [storage backend (minio/s3/local)]
    enables: [disaster recovery, compliance, data protection]

resource_framework_compliance:
  - Standard directory structure (/config, /lib, /docs, /test)
  - CLI framework integration
  - Port registry integration
  - Docker network integration  
  - Health monitoring integration
  - Configuration management standards
  
deployment_integration:
  supported_targets:
    - local: Docker Compose deployment
    - kubernetes: StatefulSet with persistent volumes
    - cloud: Integration with cloud storage services
    
  configuration_management:
    - Environment-based configuration
    - Template-based setup
    - Secret management via Vault
```

### Version Management
```yaml
versioning:
  current: 0.16.4
  compatibility: Forward compatible, backward compatible to 0.14.x
  upgrade_path: Automatic repository migration on first use
  
  breaking_changes: None in 0.16.x series
  deprecations: None
  migration_guide: Not required for 0.14.x → 0.16.x

release_management:
  release_cycle: Following upstream restic releases
  testing_requirements: Full backup/restore cycle test
  rollback_strategy: Keep previous binary, repositories compatible
```

## 🧬 Evolution Path

### Version 0.1.0 (Current)
- Basic backup/restore functionality
- Local and S3 backend support
- Daily scheduled backups
- CLI management interface

### Version 0.2.0 (Planned)
- Database-specific backup hooks
- Vault integration for key management
- Multi-repository support
- Backup verification jobs

### Long-term Vision
- ML-based retention optimization
- Cross-region replication
- Integrated disaster recovery workflows
- Real-time continuous data protection

## 🚨 Risk Mitigation

### Technical Risks
| Risk | Probability | Impact | Mitigation |
|------|------------|--------|------------|
| Repository corruption | Low | Critical | Regular verification, multiple repositories |
| Lost encryption key | Low | Critical | Key backup in Vault, recovery procedures |
| Storage backend failure | Medium | High | Multiple backend support, local cache |
| Backup window missed | Medium | Medium | Alert on missed backups, catch-up logic |

### Operational Risks
- **Configuration Drift**: Automated configuration validation
- **Storage Exhaustion**: Monitoring and alerts for repository size
- **Performance Impact**: Scheduled during low-usage periods
- **Recovery Time**: Regular restore testing to ensure RTO

## ✅ Validation Criteria

### Infrastructure Validation
- [ ] Restic installs and initializes repository
- [ ] All management commands work correctly
- [ ] Backup and restore operations verified
- [ ] Performance meets targets for local backups
- [ ] Security encryption validated
- [ ] Documentation complete with recovery procedures

### Integration Validation  
- [ ] Successfully backs up Postgres databases
- [ ] Integrates with MinIO for remote storage
- [ ] Works with Vrooli resource framework
- [ ] Health monitoring functions properly
- [ ] Scheduled backups execute automatically

### Operational Validation
- [ ] Deployment procedures documented
- [ ] Recovery procedures tested
- [ ] Retention policies work correctly
- [ ] Monitoring alerts configured
- [ ] Performance verified under load

## 📝 Implementation Notes

### Design Decisions
**Client-side encryption**: Chosen for zero-trust security
- Alternative considered: Server-side encryption
- Decision driver: Data sovereignty and security
- Trade-offs: Slight performance overhead acceptable

**Incremental snapshots**: Efficient storage and fast backups
- Alternative considered: Full backups
- Decision driver: Storage efficiency and speed
- Trade-offs: More complex restore process

### Known Limitations
- **Single repository lock**: Only one operation at a time
  - Workaround: Queue backup requests
  - Future fix: Multiple repository support
  
- **Memory usage with large files**: Can consume significant RAM
  - Workaround: Increase container memory limits
  - Future fix: Streaming mode for large files

### Integration Considerations
- **Database Consistency**: Use pre-backup hooks for consistent snapshots
- **Network Bandwidth**: Consider throttling for remote backups
- **Storage Planning**: Repository grows over time, plan capacity
- **Key Management**: Critical to never lose encryption keys

## 🔗 References

### Documentation
- README.md - Quick start and usage guide
- docs/RECOVERY.md - Disaster recovery procedures
- docs/SCHEDULING.md - Backup scheduling configuration
- config/defaults.sh - Configuration options

### Related Resources
- minio - Primary storage backend for repositories
- postgres - Database backup integration
- vault - Encryption key management

### External Resources
- [Restic Documentation](https://restic.readthedocs.io/)
- [Restic GitHub](https://github.com/restic/restic)
- [Backup Best Practices](https://restic.net/blog/2017-05-08/best-practices)

---

**Last Updated**: 2025-09-16  
**Status**: Production Ready  
**Owner**: Infrastructure Team  
**Review Cycle**: Quarterly