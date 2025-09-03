# Product Requirements Document (PRD)

## 🎯 Capability Definition

### Core Capability
**What permanent capability does this scenario add to Vrooli?**
Persistent campaign-based mass file operation tracking that maintains state across Claude Code conversations and agent iterations. Enables systematic tracking of large-scale refactoring, feature additions, and batch operations across multiple files with progress persistence.

### Intelligence Amplification
**How does this capability make future agents smarter?**
- **Context Preservation**: Agents never lose track of multi-file operations when conversations restart
- **Progress Resumption**: Any agent can pick up where another left off on mass operations
- **Pattern Recognition**: Historical campaign data enables learning optimal file operation strategies
- **Risk Mitigation**: Failed operations are tracked and can inform retry strategies
- **Coordination**: Multiple agents can work on different files within the same campaign

### Recursive Value
**What new scenarios become possible after this exists?**
1. **Large-scale Migration Scenarios**: Database schema changes, API version upgrades, framework migrations
2. **Code Standardization Scenarios**: Linting fixes, style guide enforcement, security pattern adoption
3. **Feature Rollout Scenarios**: Adding new capabilities across many components simultaneously
4. **Documentation Scenarios**: Generating or updating docs across entire codebases
5. **Testing Scenarios**: Adding tests to untested files, updating test patterns

## 📊 Success Metrics

### Functional Requirements
- **Must Have (P0)**
  - [ ] Create, read, update, delete campaigns with file patterns and metadata
  - [ ] Track individual file status (pending/in-progress/completed/failed) within campaigns
  - [ ] Persist all data across system restarts and conversation boundaries
  - [ ] REST API exposing all campaign and file operations
  - [ ] CLI interface providing full programmatic access to campaigns
  - [ ] Support glob patterns for file selection (e.g., `src/**/*.ts`, `docs/*.md`)
  
- **Should Have (P1)**
  - [ ] Web UI dashboard for visual campaign management and progress tracking
  - [ ] Bulk file status updates and batch operations
  - [ ] Campaign templates for common operation types
  - [ ] Search and filter campaigns by status, patterns, or metadata
  - [ ] Export campaign progress to JSON/CSV formats
  - [ ] Integration with git to track which files have uncommitted changes
  
- **Nice to Have (P2)**
  - [ ] Campaign scheduling and automated retries
  - [ ] Integration with external tools (linters, formatters, test runners)
  - [ ] Campaign analytics and performance metrics
  - [ ] Webhook notifications for campaign completion
  - [ ] Campaign cloning and template sharing

### Performance Criteria
| Metric | Target | Measurement Method |
|--------|--------|-------------------|
| Response Time | < 100ms for campaign CRUD, < 500ms for bulk operations | API monitoring |
| Throughput | 1000+ file status updates/second | Load testing |
| Storage Efficiency | < 1KB per tracked file | Database analysis |
| UI Responsiveness | < 200ms for dashboard interactions | Frontend performance testing |

### Quality Gates
- [ ] All P0 requirements implemented and tested
- [ ] Integration tests pass with PostgreSQL and CLI
- [ ] Performance targets met under load (10K+ files per campaign)
- [ ] Documentation complete (README, API docs, CLI help)
- [ ] Scenario can be invoked by Claude Code and other agents via API/CLI

## 🏗️ Technical Architecture

### Resource Dependencies
```yaml
required:
  - resource_name: postgres
    purpose: Persistent storage of campaigns, file tracking, and metadata
    integration_pattern: CLI via resource-postgres
    access_method: SQL schema with transactions for data consistency
    
optional:
  - resource_name: redis
    purpose: Caching for improved performance on large campaigns
    fallback: Direct database queries (acceptable performance degradation)
    access_method: resource-redis CLI commands
```

### Resource Integration Standards
```yaml
integration_priorities:
  1_shared_workflows:
    - workflow: Not applicable - this is a foundational data management scenario
  
  2_resource_cli:
    - command: resource-postgres [action]
      purpose: All database operations for campaigns and file tracking
    - command: resource-redis [action] 
      purpose: Optional caching layer for performance
  
  3_direct_api:
    - justification: Direct file system access needed for glob pattern resolution
      endpoint: File system via Go standard library
```

### Data Models
```yaml
primary_entities:
  - name: Campaign
    storage: postgres
    schema: |
      {
        id: UUID,
        name: string,
        description: string,
        file_patterns: string[], // glob patterns like ["src/**/*.ts", "docs/*.md"]
        working_directory: string,
        scenario_name: string, // which scenario is using this campaign
        metadata: map[string]any, // custom fields for specific use cases
        status: enum[active, paused, completed, failed],
        created_at: timestamp,
        updated_at: timestamp,
        completed_at: timestamp?
      }
    relationships: One-to-many with FileEntry
    
  - name: FileEntry
    storage: postgres
    schema: |
      {
        id: UUID,
        campaign_id: UUID,
        file_path: string, // relative to working_directory
        absolute_path: string, // computed full path
        status: enum[pending, in_progress, completed, failed, skipped],
        error_message: string?,
        attempts: int,
        last_attempt_at: timestamp?,
        completed_at: timestamp?,
        metadata: map[string]any // task-specific data
      }
    relationships: Belongs to Campaign
```

### API Contract
```yaml
endpoints:
  - method: POST
    path: /api/v1/campaigns
    purpose: Create new mass update campaign
    input_schema: |
      {
        name: string,
        description: string?,
        file_patterns: string[],
        working_directory: string,
        scenario_name: string?,
        metadata: object?
      }
    output_schema: |
      {
        id: UUID,
        file_count: int,
        campaign: Campaign
      }
    sla:
      response_time: 200ms
      availability: 99.9%
      
  - method: GET
    path: /api/v1/campaigns
    purpose: List campaigns with filtering and pagination
    input_schema: |
      {
        status?: string,
        scenario?: string,
        page?: int,
        limit?: int
      }
    output_schema: |
      {
        campaigns: Campaign[],
        total: int,
        page: int,
        limit: int
      }
      
  - method: PATCH
    path: /api/v1/campaigns/{id}/files/{file_id}/status
    purpose: Update individual file status within campaign
    input_schema: |
      {
        status: enum[pending, in_progress, completed, failed, skipped],
        error_message?: string,
        metadata?: object
      }
    output_schema: |
      {
        success: bool,
        file: FileEntry
      }
      
  - method: GET
    path: /api/v1/campaigns/{id}/files
    purpose: Get files needing work or matching filter
    input_schema: |
      {
        status?: string[],
        limit?: int
      }
    output_schema: |
      {
        files: FileEntry[],
        total: int,
        remaining: int
      }
```

### Event Interface
```yaml
published_events:
  - name: campaign.file.status_changed
    payload: { campaign_id: UUID, file_id: UUID, old_status: string, new_status: string }
    subscribers: [monitoring-scenarios, progress-trackers]
    
  - name: campaign.completed
    payload: { campaign_id: UUID, total_files: int, success_count: int, failure_count: int }
    subscribers: [notification-scenarios, workflow-orchestrators]
    
consumed_events:
  - name: file_system.changes
    action: Auto-update file existence and validate patterns
```

## 🖥️ CLI Interface Contract

### Command Structure
```yaml
cli_binary: mass-update-tracker
install_script: cli/install.sh

required_commands:
  - name: status
    description: Show operational status and active campaigns summary
    flags: [--json, --verbose, --campaign-id]
    
  - name: help
    description: Display command help and usage
    flags: [--all, --command <name>]
    
  - name: version
    description: Show CLI and API version information
    flags: [--json]

custom_commands:
  - name: create
    description: Create new mass update campaign
    api_endpoint: /api/v1/campaigns
    arguments:
      - name: name
        type: string
        required: true
        description: Campaign name
      - name: patterns
        type: string[]
        required: true
        description: File glob patterns to track
    flags:
      - name: --description
        description: Campaign description
      - name: --working-dir
        description: Base directory for patterns (default: current)
      - name: --scenario
        description: Name of scenario using this campaign
    output: Campaign ID and initial file count
    
  - name: list
    description: List campaigns with optional filtering
    api_endpoint: /api/v1/campaigns
    arguments: []
    flags:
      - name: --status
        description: Filter by campaign status
      - name: --scenario
        description: Filter by scenario name
      - name: --json
        description: Output raw JSON
    output: Formatted campaign list or JSON
    
  - name: files
    description: List files in campaign with optional status filter
    api_endpoint: /api/v1/campaigns/{id}/files
    arguments:
      - name: campaign_id
        type: string
        required: true
        description: Campaign UUID
    flags:
      - name: --status
        description: Filter by file status
      - name: --pending
        description: Show only pending files
      - name: --failed
        description: Show only failed files
      - name: --json
        description: Output raw JSON
    output: Filtered file list with paths and status
    
  - name: update-file
    description: Update status of specific file in campaign
    api_endpoint: /api/v1/campaigns/{campaign_id}/files/{file_id}/status
    arguments:
      - name: campaign_id
        type: string
        required: true
        description: Campaign UUID
      - name: file_path
        type: string  
        required: true
        description: File path (relative to working directory)
      - name: status
        type: string
        required: true
        description: New status (pending/in_progress/completed/failed/skipped)
    flags:
      - name: --error
        description: Error message if status is failed
      - name: --metadata
        description: JSON metadata to store
    output: Updated file status confirmation
```

## 🔄 Integration Requirements

### Upstream Dependencies
- **PostgreSQL**: Essential for persistent campaign and file status storage
- **File System Access**: Required for glob pattern resolution and file validation

### Downstream Enablement
- **Claude Code Workflows**: Can query pending files and update progress programmatically
- **Batch Operation Scenarios**: Any scenario needing mass file operations can leverage this
- **Monitoring Scenarios**: Can track large operation progress and alert on failures
- **CI/CD Integration**: Can coordinate with deployment workflows for staged rollouts

### Cross-Scenario Interactions
```yaml
provides_to:
  - scenario: any_refactoring_scenario
    capability: Persistent progress tracking across conversation boundaries
    interface: API + CLI
    
  - scenario: code_migration_scenarios
    capability: File-by-file migration status with rollback capability
    interface: API events + CLI
    
  - scenario: testing_scenarios
    capability: Track which files have tests added/updated
    interface: API + CLI integration
    
consumes_from:
  - scenario: file_system_monitor
    capability: Auto-detect file changes to validate campaign accuracy
    fallback: Manual refresh via API call
```

## 🎨 Style and Branding Requirements

### UI/UX Style Guidelines
```yaml
style_profile:
  category: technical
  inspiration: Modern DevOps dashboard (similar to GitHub Actions, GitLab CI)
  
  visual_style:
    color_scheme: dark
    typography: technical
    layout: dashboard
    animations: subtle
  
  personality:
    tone: technical
    mood: focused
    target_feeling: "Confident control over complex operations"

style_references:
  technical:
    - system-monitor: "Matrix-style green terminal aesthetic" 
    - agent-dashboard: "NASA mission control vibes"
    
  inspiration: "GitHub Actions workflow view meets file explorer with progress tracking"
```

### Target Audience Alignment
- **Primary Users**: Developers using Claude Code for large refactoring tasks
- **User Expectations**: Clean, efficient interface prioritizing information density over visual flair
- **Accessibility**: WCAG AA compliance, keyboard navigation, screen reader support
- **Responsive Design**: Desktop-first (primary use case), tablet secondary, mobile basic

### Brand Consistency Rules
- **Scenario Identity**: Professional developer tool with focus on clarity and efficiency
- **Vrooli Integration**: Consistent with other technical scenarios (system-monitor, agent-dashboard)
- **Professional vs Fun**: Strongly professional - this is a productivity tool for serious development work

## 💰 Value Proposition

### Business Value
- **Primary Value**: Eliminates lost progress in large refactoring tasks, reduces development time by 30-50%
- **Revenue Potential**: $15K - $25K per deployment (enterprise development teams)
- **Cost Savings**: Prevents context loss that typically costs 2-4 hours per large refactoring task
- **Market Differentiator**: First tool to provide persistent state for AI-assisted mass file operations

### Technical Value
- **Reusability Score**: High - any scenario involving multiple files can leverage this
- **Complexity Reduction**: Transforms complex multi-conversation workflows into manageable campaigns
- **Innovation Enablement**: Unlocks scenarios requiring coordination across hundreds of files

## 🧬 Evolution Path

### Version 1.0 (Current)
- Core campaign and file tracking
- REST API and CLI interface
- Basic web dashboard
- PostgreSQL persistence

### Version 2.0 (Planned)  
- Git integration for tracking uncommitted changes
- Campaign templates and preset patterns
- Webhook notifications and external tool integration
- Advanced analytics and performance metrics

### Long-term Vision
- **AI-Powered Orchestration**: Intelligent scheduling of file operations based on dependencies
- **Cross-Repository Campaigns**: Track operations across multiple repositories
- **Collaborative Campaigns**: Multiple developers working on same mass operation
- **Integration Hub**: Becomes central coordination point for all mass operations in Vrooli ecosystem

## 🔄 Scenario Lifecycle Integration

### Direct Scenario Deployment
```yaml
direct_execution:
  supported: true
  structure_compliance:
    - service.json with PostgreSQL dependency
    - API server with health checks
    - CLI tool with full campaign management
    - UI dashboard for visual management
    
  deployment_targets:
    - local: Docker Compose with PostgreSQL
    - kubernetes: Helm chart with persistent volumes
    - cloud: AWS RDS + ECS deployment
    
  revenue_model:
    - type: subscription
    - pricing_tiers: 
      - Team: $199/month (up to 10 developers)  
      - Enterprise: $999/month (unlimited, advanced features)
    - trial_period: 30 days
```

### Capability Discovery
```yaml
discovery:
  registry_entry:
    name: mass-update-tracker
    category: automation
    capabilities: 
      - persistent_campaign_tracking
      - file_operation_coordination
      - progress_state_management
      - cross_conversation_continuity
    interfaces:
      - api: http://localhost:20251/api/v1
      - cli: mass-update-tracker
      - events: campaign.* namespace
      
  metadata:
    description: "Campaign-based mass file operation tracking with persistent state"
    keywords: [mass-update, file-tracking, campaign, refactoring, batch-operations]
    dependencies: [postgres]
    enhances: [all-refactoring-scenarios, migration-scenarios, batch-operation-scenarios]
```

### Version Management
```yaml
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
| Database corruption during large campaigns | Low | High | Atomic transactions, regular backups |
| Performance degradation with massive file counts | Medium | Medium | Pagination, indexing, optional Redis cache |
| Glob pattern conflicts | Medium | Low | Pattern validation, conflict detection |
| Concurrent access conflicts | Low | Medium | Database-level locking, optimistic updates |

### Operational Risks
- **Drift Prevention**: PRD serves as single source of truth, validated by scenario-test.yaml
- **Version Compatibility**: Semantic versioning with clear API contracts
- **Resource Conflicts**: PostgreSQL shared safely via resource management
- **Style Drift**: Technical dashboard style maintained via design system
- **CLI Consistency**: Automated testing ensures CLI-API parity

## ✅ Validation Criteria

### Declarative Test Specification
```yaml
version: 1.0
scenario: mass-update-tracker

structure:
  required_files:
    - .vrooli/service.json
    - PRD.md
    - api/main.go
    - api/go.mod
    - cli/mass-update-tracker
    - cli/install.sh
    - initialization/storage/postgres/schema.sql
    - scenario-test.yaml
    - ui/index.html
    
  required_dirs:
    - api
    - cli
    - initialization
    - initialization/storage
    - ui
    - test

resources:
  required: [postgres]
  optional: [redis]
  health_timeout: 60

tests:
  - name: "PostgreSQL is accessible"
    type: http
    service: postgres
    endpoint: /health
    method: GET
    expect:
      status: 200
      
  - name: "API health endpoint responds"
    type: http
    service: api
    endpoint: /health
    method: GET
    expect:
      status: 200
      
  - name: "Campaign creation works"
    type: http
    service: api
    endpoint: /api/v1/campaigns
    method: POST
    body:
      name: "test-campaign"
      file_patterns: ["*.md"]
      working_directory: "/tmp"
    expect:
      status: 201
      body:
        id: "[uuid-pattern]"
        
  - name: "CLI status command executes"
    type: exec
    command: ./cli/mass-update-tracker status --json
    expect:
      exit_code: 0
      output_contains: ["campaigns"]
      
  - name: "Database schema is initialized"
    type: sql
    service: postgres
    query: "SELECT COUNT(*) FROM information_schema.tables WHERE table_name IN ('campaigns', 'file_entries')"
    expect:
      rows: 
        - count: 2
```

### Performance Validation
- [ ] API response times meet SLA targets under load
- [ ] Campaign with 10,000+ files completes operations within performance bounds
- [ ] UI remains responsive with large datasets
- [ ] Database queries remain performant with proper indexing

### Integration Validation
- [ ] Discoverable via Vrooli resource registry
- [ ] All API endpoints documented and functional
- [ ] All CLI commands executable with comprehensive --help
- [ ] Events published correctly for downstream consumption

### Capability Verification
- [ ] Successfully tracks file operations across conversation restarts
- [ ] Enables resumable mass operations for Claude Code workflows
- [ ] Provides reliable progress state for complex refactoring tasks
- [ ] Maintains data consistency under concurrent access

## 📝 Implementation Notes

### Design Decisions
**Database vs File Storage**: PostgreSQL chosen over file-based storage
- Alternative considered: JSON files for simplicity
- Decision driver: Need for ACID transactions and concurrent access safety
- Trade-offs: Slightly more complex setup for significantly better reliability

**API-First Design**: REST API as primary interface with CLI as wrapper
- Alternative considered: CLI-first with optional API
- Decision driver: Enables easier integration with other scenarios and external tools
- Trade-offs: More initial complexity for better long-term extensibility

### Known Limitations
- **File System Scope**: Cannot track files across different filesystems or remote locations
  - Workaround: Use absolute paths and ensure working directory accessibility
  - Future fix: Add support for remote file systems via plugins

- **Pattern Complexity**: Very complex glob patterns may impact performance
  - Workaround: Recommend simpler patterns, provide pattern optimization guidance
  - Future fix: Pattern compilation and caching

### Security Considerations
- **Data Protection**: Campaign metadata may contain sensitive information - encrypt at rest
- **Access Control**: No authentication in v1 (local use), add token-based auth in v2
- **Audit Trail**: All file status changes logged with timestamps for debugging

## 🔗 References

### Documentation
- README.md - User-facing overview and quick start
- docs/api.md - Complete API specification
- docs/cli.md - CLI command reference
- docs/integration.md - How other scenarios can leverage campaigns

### Related PRDs
- agent-metareasoning-manager: Similar coordination patterns
- app-debugger: File-based operation tracking inspiration

### External Resources
- [GitHub Actions Workflow Syntax](https://docs.github.com/en/actions/using-workflows/workflow-syntax-for-github-actions) - Inspiration for campaign definition
- [PostgreSQL JSON Support](https://www.postgresql.org/docs/current/datatype-json.html) - Metadata storage patterns
- [Glob Pattern Specification](https://pkg.go.dev/path/filepath#Match) - File pattern matching reference

---

**Last Updated**: 2025-09-03  
**Status**: Draft  
**Owner**: Claude Code Assistant  
**Review Cycle**: Validate against implementation after each major milestone