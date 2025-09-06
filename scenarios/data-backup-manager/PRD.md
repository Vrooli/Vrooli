# Product Requirements Document (PRD)

## 🎯 Capability Definition

### Core Capability
**What permanent capability does this scenario add to Vrooli?**
The Data Backup Manager adds comprehensive data protection and recovery capabilities to Vrooli, providing automated, intelligent backup management for all resources, scenarios, and user data. It ensures data integrity and business continuity across the entire Vrooli ecosystem.

### Intelligence Amplification
**How does this capability make future agents smarter?**
- Enables risk-free experimentation by providing reliable rollback capabilities
- Allows agents to safely modify critical data knowing backups exist
- Provides historical data access for analysis and learning from past states
- Enables disaster recovery capabilities for autonomous operations
- Creates data lineage tracking for better decision-making

### Recursive Value
**What new scenarios become possible after this exists?**
- **Time Travel Debugging**: Scenarios that can revert to previous states for debugging
- **Audit Trail Analytics**: Deep analysis of data changes over time
- **Automated Recovery Systems**: Self-healing scenarios that auto-restore from corruption
- **Compliance Manager**: Automated compliance reporting with historical data preservation
- **Data Migration Orchestrator**: Safe migration of scenarios between environments

## 📊 Success Metrics

### Functional Requirements
- **Must Have (P0)**
  - [ ] Automated daily backups of all PostgreSQL databases
  - [ ] Scenario file and configuration backup with versioning
  - [ ] MinIO object storage backup with integrity verification
  - [ ] Point-in-time recovery capability (7-day retention minimum)
  - [ ] Backup health monitoring and alerting system
  
- **Should Have (P1)**
  - [ ] Incremental backup support to reduce storage overhead
  - [ ] Cross-environment backup synchronization
  - [ ] Backup encryption at rest and in transit
  - [ ] Automated backup testing and validation
  - [ ] Backup performance optimization and compression
  
- **Nice to Have (P2)**
  - [ ] Cloud storage integration (AWS S3, Google Cloud)
  - [ ] Backup analytics and trend reporting
  - [ ] Custom backup schedules per scenario
  - [ ] Integration with external monitoring systems

### Performance Criteria
| Metric | Target | Measurement Method |
|--------|--------|-------------------|
| Backup Completion Time | < 30 minutes for full system backup | Automated monitoring |
| Recovery Time Objective (RTO) | < 15 minutes for critical data | Recovery testing |
| Recovery Point Objective (RPO) | < 4 hours maximum data loss | Backup frequency validation |
| Storage Efficiency | > 70% compression ratio | Backup size analysis |
| System Impact | < 5% CPU/Memory during backup | Resource monitoring |

### Quality Gates
- [ ] All P0 requirements implemented and tested
- [ ] Integration tests pass with all required resources
- [ ] Performance targets met under load
- [ ] Documentation complete (README, API docs, CLI help)
- [ ] Scenario can be invoked by other agents via API/CLI
- [ ] Disaster recovery procedures documented and tested

## 🏗️ Technical Architecture

### Resource Dependencies
```yaml
required:
  - resource_name: postgres
    purpose: Store backup metadata, schedules, and logs
    integration_pattern: Direct API and pg_dump utilities
    access_method: postgres CLI commands and Go database driver
    
  - resource_name: minio
    purpose: Object storage for backup files and archives
    integration_pattern: MinIO client API for object operations
    access_method: MinIO Go client library
    
  - resource_name: n8n
    purpose: Backup orchestration and scheduling workflows
    integration_pattern: Shared workflows for automation
    access_method: N8n workflow API and webhooks
    
optional:
  - resource_name: redis
    purpose: Caching backup status and job queues
    fallback: Use PostgreSQL for job queuing if unavailable
    access_method: Redis Go client library
```

### Resource Integration Standards
```yaml
# Priority order for resource access (MUST follow this hierarchy):
integration_priorities:
  1_shared_workflows:     # FIRST: Use existing shared n8n workflows
    - workflow: postgres-utils.json
      location: initialization/automation/n8n/
      purpose: PostgreSQL backup and restore operations
    - workflow: file-system-ops.json
      location: initialization/automation/n8n/
      purpose: File system backup operations
    - workflow: storage-manager.json
      location: initialization/automation/n8n/
      purpose: MinIO backup storage management
  
  2_resource_cli:        # SECOND: Use resource CLI commands
    - command: resource-postgres backup
      purpose: Create PostgreSQL database dumps
    - command: resource-minio upload
      purpose: Upload backup files to object storage
  
  3_direct_api:          # LAST: Direct API only when necessary
    - justification: Real-time backup status monitoring requires direct API access
      endpoint: /api/v1/backup/status

# Shared workflow guidelines:
shared_workflow_criteria:
  - Must be truly reusable across multiple scenarios
  - Place in initialization/automation/n8n/ if generic
  - Document reusability in workflow description
  - List all scenarios that will use this workflow: [disaster-recovery, maintenance-orchestrator, scenario-migrator]
```

### Data Models
```yaml
# Core data structures that define the capability
primary_entities:
  - name: BackupJob
    storage: postgres
    schema: |
      {
        id: UUID,
        type: "full|incremental|differential",
        target: "postgres|files|scenario|minio",
        target_identifier: string,
        status: "pending|running|completed|failed",
        started_at: timestamp,
        completed_at: timestamp,
        size_bytes: bigint,
        compression_ratio: float,
        storage_path: string,
        checksum: string,
        retention_until: timestamp,
        metadata: jsonb
      }
    relationships: Links to BackupSchedule and RestorePoint entities
    
  - name: BackupSchedule
    storage: postgres
    schema: |
      {
        id: UUID,
        name: string,
        cron_expression: string,
        backup_type: "full|incremental",
        targets: array<string>,
        retention_days: integer,
        enabled: boolean,
        last_run: timestamp,
        next_run: timestamp,
        created_by: string
      }
    relationships: Has many BackupJob executions
    
  - name: RestorePoint
    storage: postgres
    schema: |
      {
        id: UUID,
        name: string,
        backup_job_ids: array<UUID>,
        created_at: timestamp,
        description: string,
        verified: boolean,
        verification_date: timestamp
      }
    relationships: References multiple BackupJob records
```

### API Contract
```yaml
# Defines how other scenarios/agents can use this capability
endpoints:
  - method: POST
    path: /api/v1/backup/create
    purpose: Trigger immediate backup of specified targets
    input_schema: |
      {
        type: "full|incremental",
        targets: ["postgres", "files", "scenarios"],
        description?: string,
        retention_days?: integer
      }
    output_schema: |
      {
        job_id: UUID,
        estimated_duration: "15m",
        status: "pending",
        targets: array<string>
      }
    sla:
      response_time: 500ms
      availability: 99.9%
      
  - method: POST
    path: /api/v1/restore/create
    purpose: Restore data from backup to specified point in time
    input_schema: |
      {
        restore_point_id?: UUID,
        backup_job_id?: UUID,
        targets: array<string>,
        destination?: string,
        verify_before_restore: boolean
      }
    output_schema: |
      {
        restore_id: UUID,
        estimated_duration: "10m",
        status: "pending",
        validation_results: array<object>
      }
    sla:
      response_time: 1000ms
      availability: 99.9%
      
  - method: GET
    path: /api/v1/backup/status
    purpose: Get current backup system status and recent jobs
    output_schema: |
      {
        system_status: "healthy|degraded|critical",
        active_jobs: array<BackupJob>,
        last_successful_backup: timestamp,
        storage_usage: {
          used_gb: number,
          available_gb: number,
          compression_ratio: number
        }
      }
    sla:
      response_time: 200ms
      availability: 99.95%
```

### Event Interface
```yaml
# Events this capability publishes for others to consume
published_events:
  - name: backup.job.started
    payload: { job_id: UUID, type: string, targets: array }
    subscribers: [monitoring systems, notification services]
    
  - name: backup.job.completed
    payload: { job_id: UUID, status: string, duration: string, size_bytes: number }
    subscribers: [audit systems, reporting dashboards]
    
  - name: backup.job.failed
    payload: { job_id: UUID, error: string, retry_count: number }
    subscribers: [alerting systems, maintenance orchestrator]
    
  - name: restore.completed
    payload: { restore_id: UUID, targets: array, success: boolean }
    subscribers: [audit systems, scenario managers]
    
consumed_events:
  - name: scenario.deployed
    action: Create automatic backup of new scenario files
  - name: postgres.schema_updated
    action: Trigger incremental backup of affected databases
```

## 🖥️ CLI Interface Contract

### Command Structure
```yaml
# Primary CLI executable name and pattern
cli_binary: data-backup-manager
install_script: cli/install.sh

# Core commands that MUST be implemented:
required_commands:
  - name: status
    description: Show backup system status and recent jobs
    flags: [--json, --verbose, --targets <list>]
    
  - name: help
    description: Display command help and usage
    flags: [--all, --command <name>]
    
  - name: version
    description: Show CLI and API version information
    flags: [--json]

# Scenario-specific commands (must mirror API endpoints):
custom_commands:
  - name: backup
    description: Create backup of specified targets
    api_endpoint: /api/v1/backup/create
    arguments:
      - name: targets
        type: string
        required: true
        description: Comma-separated list of backup targets (postgres,files,scenarios)
    flags:
      - name: --type
        description: Backup type (full, incremental)
      - name: --retention-days
        description: How long to keep this backup
      - name: --description
        description: Human-readable backup description
    output: Backup job details and progress tracking
    
  - name: restore
    description: Restore data from backup
    api_endpoint: /api/v1/restore/create
    arguments:
      - name: restore-point
        type: string
        required: false
        description: Restore point ID or backup job ID
    flags:
      - name: --targets
        description: Specific targets to restore
      - name: --verify
        description: Verify backup integrity before restore
      - name: --destination
        description: Custom restore destination path
    output: Restore progress and verification results
    
  - name: list
    description: List available backups and restore points
    api_endpoint: /api/v1/backup/list
    flags:
      - name: --type
        description: Filter by backup type
      - name: --target
        description: Filter by backup target
      - name: --since
        description: Show backups since date (YYYY-MM-DD)
    output: Formatted table of available backups
    
  - name: verify
    description: Verify backup integrity
    api_endpoint: /api/v1/backup/verify
    arguments:
      - name: backup-id
        type: string
        required: true
        description: Backup job ID to verify
    output: Verification results and any issues found
    
  - name: schedule
    description: Manage backup schedules
    api_endpoint: /api/v1/schedules
    arguments:
      - name: action
        type: string
        required: true
        description: Action to perform (list, create, update, delete, enable, disable)
    flags:
      - name: --name
        description: Schedule name
      - name: --cron
        description: Cron expression for schedule
      - name: --targets
        description: Backup targets for schedule
      - name: --retention
        description: Retention period in days
    output: Schedule configuration and status
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
  - language: Go for consistency with API
  - dependencies: Minimal - reuse API client libraries
  - error_handling: Consistent exit codes (0=success, 1=error, 2=warning)
  - configuration: 
      - Read from ~/.vrooli/data-backup-manager/config.yaml
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
- **PostgreSQL**: Database storage for backup metadata and job tracking
- **MinIO**: Object storage for backup files and archives
- **N8n**: Workflow orchestration for scheduled and automated backups
- **File System Access**: Read/write permissions for scenario directories and data

### Downstream Enablement
**What future capabilities does this unlock?**
- **Disaster Recovery Manager**: Comprehensive disaster recovery orchestration
- **Compliance Reporter**: Automated compliance reporting with data retention
- **Data Migration Tools**: Safe scenario and environment migration capabilities
- **Time Travel Debugger**: Point-in-time system state restoration for debugging

### Cross-Scenario Interactions
```yaml
# How this scenario enhances other scenarios
provides_to:
  - scenario: maintenance-orchestrator
    capability: Automated backup scheduling and monitoring
    interface: API/CLI/Events
    
  - scenario: scenario-generator
    capability: Automatic backup of generated scenarios
    interface: Events/API
    
  - scenario: system-monitor
    capability: Backup health metrics and alerting
    interface: Events/API
    
consumes_from:
  - scenario: system-monitor
    capability: Resource usage monitoring during backups
    fallback: Use internal resource monitoring
    
  - scenario: notification-hub
    capability: Backup completion and failure notifications
    fallback: Log to system logs only
```

## 🎨 Style and Branding Requirements

### UI/UX Style Guidelines
```yaml
# Define the visual and experiential personality of this scenario
style_profile:
  category: technical
  inspiration: Infrastructure monitoring dashboards like Grafana, DataDog
  
  # Visual characteristics:
  visual_style:
    color_scheme: dark
    typography: technical
    layout: dashboard
    animations: subtle
  
  # Personality traits:
  personality:
    tone: serious
    mood: focused
    target_feeling: Confidence in data protection

# Style examples from existing scenarios:
style_references:
  technical:
    - system-monitor: "Matrix-style green terminal aesthetic"
    - agent-dashboard: "NASA mission control vibes"
```

### Target Audience Alignment
- **Primary Users**: System administrators, DevOps engineers, scenario developers
- **User Expectations**: Professional, reliable, no-nonsense interface focused on critical data
- **Accessibility**: WCAG AA compliance, high contrast for monitoring scenarios
- **Responsive Design**: Desktop-first design for dashboard use

### Brand Consistency Rules
- **Scenario Identity**: Professional infrastructure tool aesthetic
- **Vrooli Integration**: Must feel part of the critical infrastructure ecosystem
- **Professional vs Fun**: Strictly professional - data protection is serious business

## 💰 Value Proposition

### Business Value
- **Primary Value**: Protects all Vrooli investments by preventing data loss
- **Revenue Potential**: $25K - $75K per deployment (enterprise data protection value)
- **Cost Savings**: Prevents catastrophic data loss events (potentially $100K+ in recovery costs)
- **Market Differentiator**: Built-in data protection for AI automation platforms

### Technical Value
- **Reusability Score**: 9/10 - Every scenario benefits from backup capabilities
- **Complexity Reduction**: Makes data recovery simple and automated
- **Innovation Enablement**: Enables risk-free experimentation and rapid rollback

## 🧬 Evolution Path

### Version 1.0 (Current)
- Core backup and restore capabilities for PostgreSQL, files, and MinIO
- Basic scheduling and automation via N8n workflows
- Essential CLI and API interface for backup management
- Local storage with compression and verification

### Version 2.0 (Planned)
- Multi-site backup replication and disaster recovery
- Advanced backup analytics and optimization recommendations
- Integration with cloud storage providers (AWS S3, GCP, Azure)
- Backup performance optimization with parallel processing

### Long-term Vision
- Predictive backup optimization using machine learning
- Integration with blockchain for immutable backup verification
- Cross-platform scenario migration and environment synchronization
- Real-time backup streaming for zero-RPO scenarios

## 🔄 Scenario Lifecycle Integration

### Direct Scenario Deployment
```yaml
# Requirements for direct scenario execution:
direct_execution:
  supported: true
  structure_compliance:
    - service.json with complete metadata including maintenance tag
    - All required initialization files for databases and workflows
    - Deployment scripts (startup.sh, monitor.sh)
    - Health check endpoints for backup system monitoring
    
  deployment_targets:
    - local: Docker Compose with persistent storage volumes
    - kubernetes: Helm chart with PVC for backup storage
    - cloud: Terraform templates for multi-region backup storage
    
  revenue_model:
    - type: subscription
    - pricing_tiers: 
      - basic: $2K/month (local backups, 30-day retention)
      - professional: $5K/month (cloud integration, 90-day retention)
      - enterprise: $10K/month (multi-site, unlimited retention)
    - trial_period: 14 days
```

### Capability Discovery
```yaml
# How other scenarios/agents discover and use this capability:
discovery:
  registry_entry:
    name: data-backup-manager
    category: maintenance
    capabilities: [backup, restore, data-protection, disaster-recovery]
    interfaces:
      - api: http://localhost:20010/api/v1
      - cli: data-backup-manager
      - events: backup.*
      
  metadata:
    description: Comprehensive backup management for all Vrooli data
    keywords: [backup, restore, disaster-recovery, data-protection, maintenance]
    dependencies: [postgres, minio, n8n]
    enhances: [all scenarios - provides data protection]
```

### Version Management
```yaml
# Compatibility and upgrade paths:
versioning:
  current: 1.0.0
  minimum_compatible: 1.0.0
  
  breaking_changes: []
      
  deprecations: []
```

## 🚨 Risk Mitigation

### Technical Risks
| Risk | Probability | Impact | Mitigation |
|------|------------|--------|------------|
| Backup corruption | Low | Critical | Multiple verification methods, checksums, test restores |
| Storage exhaustion | Medium | High | Automated cleanup, compression, storage monitoring |
| Long backup times | Medium | Medium | Incremental backups, parallel processing, optimization |
| Recovery failure | Low | Critical | Automated recovery testing, multiple restore methods |

### Operational Risks
- **Drift Prevention**: PRD serves as single source of truth, validated by scenario-test.yaml
- **Version Compatibility**: Semantic versioning with clear breaking change documentation
- **Resource Conflicts**: Resource allocation managed through service.json priorities
- **Data Integrity**: Multiple verification layers and automated testing
- **Security**: Encryption at rest and in transit, access control for backup files

## ✅ Validation Criteria

### Declarative Test Specification
```yaml
# REQUIRED: scenario-test.yaml in scenario root
version: 1.0
scenario: data-backup-manager

# Structure validation - files and directories that MUST exist:
structure:
  required_files:
    - .vrooli/service.json
    - PRD.md
    - api/main.go
    - api/go.mod
    - cli/data-backup-manager
    - cli/install.sh
    - initialization/postgres/schema.sql
    - scenario-test.yaml
    
  required_dirs:
    - api
    - cli
    - initialization
    - initialization/n8n
    - initialization/postgres
    - data/backups

# Resource validation:
resources:
  required: [postgres, minio, n8n]
  optional: [redis]
  health_timeout: 60

# Declarative tests:
tests:
  # Resource health checks:
  - name: "PostgreSQL is accessible"
    type: http
    service: postgres
    endpoint: /health
    method: GET
    expect:
      status: 200
      
  - name: "MinIO is accessible"
    type: http
    service: minio
    endpoint: /minio/health/live
    method: GET
    expect:
      status: 200
      
  # API endpoint tests:
  - name: "API health endpoint responds"
    type: http
    service: api
    endpoint: /health
    method: GET
    expect:
      status: 200
      body:
        status: "healthy"
        
  - name: "Backup status endpoint responds"
    type: http
    service: api
    endpoint: /api/v1/backup/status
    method: GET
    expect:
      status: 200
      body:
        system_status: ["healthy", "degraded", "critical"]
        
  # CLI command tests:
  - name: "CLI status command executes"
    type: exec
    command: ./cli/data-backup-manager status --json
    expect:
      exit_code: 0
      output_contains: ["system_status"]
      
  - name: "CLI backup command validation"
    type: exec
    command: ./cli/data-backup-manager backup --help
    expect:
      exit_code: 0
      output_contains: ["targets", "type", "retention"]
      
  # Database tests:
  - name: "Backup metadata schema exists"
    type: sql
    service: postgres
    query: "SELECT COUNT(*) FROM information_schema.tables WHERE table_name IN ('backup_jobs', 'backup_schedules', 'restore_points')"
    expect:
      rows: 
        - count: 3
        
  # Functional tests:
  - name: "Create test backup"
    type: http
    service: api
    endpoint: /api/v1/backup/create
    method: POST
    body:
      type: "full"
      targets: ["test-data"]
      description: "Integration test backup"
    expect:
      status: 201
      body:
        job_id: "not_null"
        status: "pending"
```

### Test Execution Gates
```bash
# All tests must pass via:
vrooli test scenario data-backup-manager --validation complete

# Individual test categories:
vrooli test scenario data-backup-manager --structure    # Verify file/directory structure
vrooli test scenario data-backup-manager --resources    # Check resource health
vrooli test scenario data-backup-manager --integration  # Run integration tests
vrooli test scenario data-backup-manager --performance  # Validate performance targets
```

### Performance Validation
- [ ] API response times meet SLA targets
- [ ] Backup completion within 30 minutes for full system
- [ ] Recovery operations complete within 15 minutes
- [ ] System resource usage remains under 5% during backups
- [ ] No memory leaks detected over 24-hour backup cycles

### Integration Validation
- [ ] Discoverable via resource registry
- [ ] All API endpoints documented and functional
- [ ] All CLI commands executable with --help
- [ ] N8n workflows properly registered and active
- [ ] Events published/consumed correctly with other scenarios

### Capability Verification
- [ ] Successfully backs up PostgreSQL databases with data integrity
- [ ] File system backups preserve permissions and structure
- [ ] MinIO backup includes metadata and access policies
- [ ] Point-in-time recovery works accurately
- [ ] Scheduled backups execute automatically without manual intervention
- [ ] Failed backups trigger appropriate alerts and retry mechanisms

## 📝 Implementation Notes

### Design Decisions
**Storage Architecture**: Chosen MinIO for backup storage over traditional file systems
- Alternative considered: Local file system with rsync
- Decision driver: MinIO provides better integrity verification, versioning, and future cloud integration
- Trade-offs: Additional resource dependency but significantly better reliability

**Backup Strategy**: Implemented full + incremental backup pattern
- Alternative considered: Full backups only or continuous backup streaming
- Decision driver: Balance between storage efficiency and recovery complexity
- Trade-offs: More complex restore logic but 70%+ storage savings

### Known Limitations
- **Large Database Limitations**: PostgreSQL dumps may take extended time for databases >100GB
  - Workaround: Use pg_basebackup for large databases instead of pg_dump
  - Future fix: Implement parallel dump/restore capabilities in version 2.0

- **Cross-Platform Limitations**: File permissions may not transfer correctly between different OS platforms
  - Workaround: Document platform-specific restore procedures
  - Future fix: Implement metadata-driven permission restoration

### Security Considerations
- **Data Protection**: All backups encrypted at rest using AES-256, in transit using TLS 1.3
- **Access Control**: Backup files accessible only to authorized Vrooli processes and administrators
- **Audit Trail**: All backup and restore operations logged with user attribution and timestamps
- **Key Management**: Encryption keys rotated automatically every 90 days

## 🔗 References

### Documentation
- README.md - User-facing overview and quick start guide
- docs/api.md - Complete API specification with examples
- docs/cli.md - CLI command reference and usage examples
- docs/disaster-recovery.md - Disaster recovery procedures and best practices

### Related PRDs
- maintenance-orchestrator - Backup scheduling integration
- system-monitor - Health monitoring and alerting integration
- scenario-generator - Automatic backup of new scenarios

### External Resources
- PostgreSQL Backup Documentation: https://www.postgresql.org/docs/current/backup.html
- MinIO Client Reference: https://docs.min.io/docs/minio-client-complete-guide.html
- N8n Workflow API: https://docs.n8n.io/api/

---

**Last Updated**: 2025-09-05  
**Status**: Draft  
**Owner**: Claude AI Agent  
**Review Cycle**: Weekly validation against implementation progress