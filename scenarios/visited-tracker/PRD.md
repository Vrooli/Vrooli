# Product Requirements Document (PRD)

## 🎯 Capability Definition

### Core Capability
**What permanent capability does this scenario add to Vrooli?**
Persistent file visit tracking with staleness detection that maintains state across Claude Code conversations, AI models, and agent iterations. Enables systematic code coverage for security audits, performance reviews, bug hunting, and any scenario requiring comprehensive file analysis with intelligent prioritization based on visit frequency and file modification patterns.

### Intelligence Amplification
**How does this capability make future agents smarter?**
- **Coverage Guarantee**: Agents never miss files during systematic reviews - every file gets attention
- **Staleness Detection**: Files modified frequently but rarely visited get priority (high bug risk)
- **Cross-Model Memory**: Any AI model can see what others have analyzed and when
- **Smart Prioritization**: Least visited + most stale files surface automatically
- **Historical Context**: Visit timestamps and contexts enable pattern analysis
- **Composition Support**: Scenarios can leverage visit data for their own tracking needs

### Recursive Value
**What new scenarios become possible after this exists?**
1. **Security Audit Scenarios**: Systematic vulnerability scanning with coverage tracking
2. **Performance Analysis Scenarios**: Profile every file, prioritize hotspots
3. **Bug Hunt Scenarios**: Ensure no file escapes review during debugging sessions
4. **Documentation Scenarios**: Track which files have updated docs
5. **Code Quality Scenarios**: Systematic technical debt identification
6. **Compliance Scenarios**: Ensure every file meets regulatory requirements

## 📊 Success Metrics

### Functional Requirements
- **Must Have (P0)**
  - [ ] Track visit count per file with timestamps and context
  - [ ] Calculate staleness score (modifications since last visit × time elapsed)
  - [ ] Persist all data across system restarts and conversation boundaries
  - [ ] Structure sync to handle file additions/deletions/moves
  - [ ] REST API exposing visit tracking and prioritization
  - [ ] CLI interface providing full programmatic access
  - [ ] Support file patterns and explicit file lists
  - [ ] Least visited and most stale prioritization APIs
  
- **Should Have (P1)**
  - [ ] Web UI dashboard with heatmap visualization
  - [ ] Import/export visited maps for backup and sharing
  - [ ] Bulk visit recording for batch operations
  - [ ] Visit context tracking (security/performance/bug/docs)
  - [ ] Content hash tracking to detect file changes
  - [ ] Coverage reports showing visited vs unvisited percentages
  
- **Nice to Have (P2)**
  - [ ] Git integration for modification tracking
  - [ ] Multi-repository support with cross-repo tracking
  - [ ] Visit patterns analysis and recommendations
  - [ ] Integration with IDE plugins for real-time tracking
  - [ ] Webhook notifications for stale file alerts

### Performance Criteria
| Metric | Target | Measurement Method |
|--------|--------|-------------------|
| Response Time | < 50ms for visit recording, < 200ms for prioritization | API monitoring |
| Throughput | 5000+ visit records/second | Load testing |
| Storage Efficiency | < 500 bytes per tracked file | Database analysis |
| Staleness Calculation | < 100ms for 10K files | Performance testing |

### Quality Gates
- [ ] All P0 requirements implemented and tested
- [ ] Integration tests pass with PostgreSQL and CLI
- [ ] Performance targets met under load (50K+ tracked files)
- [ ] Documentation complete (README, API docs, CLI help)
- [ ] Scenario can be invoked by Claude Code and other agents via API/CLI

## 🏗️ Technical Architecture

### Resource Dependencies
```yaml
required:
  - resource_name: postgres
    purpose: Persistent storage of visit tracking, staleness scores, and metadata
    integration_pattern: CLI via resource-postgres
    access_method: SQL schema with efficient indexes for prioritization queries
    
optional:
  - resource_name: redis
    purpose: Caching for improved performance on prioritization queries
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
      purpose: All database operations for visit tracking and staleness
    - command: resource-redis [action] 
      purpose: Optional caching layer for performance
  
  3_direct_api:
    - justification: Direct file system access needed for modification time checks
      endpoint: File system via Go standard library
```

### Data Models
```yaml
primary_entities:
  - name: TrackedFile
    storage: postgres
    schema: |
      {
        id: UUID,
        file_path: string,           // relative path
        absolute_path: string,        // full path
        visit_count: int,             // total visits
        first_seen: timestamp,        // when file was first tracked
        last_visited: timestamp?,     // last visit time
        last_modified: timestamp,     // file modification time
        content_hash: string?,        // SHA256 for change detection
        size_bytes: int64,            // file size
        staleness_score: float,       // calculated: (mods × time) / (visits + 1)
        visits: Visit[],              // detailed visit history
        deleted: boolean,             // soft delete for removed files
        metadata: map[string]any      // extensible data
      }
    relationships: One-to-many with Visit
    
  - name: Visit
    storage: postgres
    schema: |
      {
        id: UUID,
        file_id: UUID,
        timestamp: timestamp,
        context: string?,             // security/performance/bug/docs/general
        agent: string?,               // which AI model/agent visited
        conversation_id: string?,     // link visits within conversations
        duration_ms: int?,            // how long the visit took
        findings: map[string]any?     // what was found/changed
      }
    relationships: Belongs to TrackedFile
    
  - name: StructureSnapshot
    storage: postgres
    schema: |
      {
        id: UUID,
        timestamp: timestamp,
        total_files: int,
        new_files: string[],          // files added since last snapshot
        deleted_files: string[],      // files removed since last snapshot
        moved_files: map[string]string, // old_path -> new_path
        snapshot_data: JSONB          // full file structure
      }
    relationships: None (audit table)
```

### API Contract
```yaml
endpoints:
  - method: POST
    path: /api/v1/visit
    purpose: Record file visit(s)
    input_schema: |
      {
        files: string[] | {path: string, context?: string}[],
        context?: string,             // applies to all if not per-file
        agent?: string,
        conversation_id?: string,
        metadata?: object
      }
    output_schema: |
      {
        recorded: int,
        files: TrackedFile[]
      }
    sla:
      response_time: 50ms
      availability: 99.9%
      
  - method: POST
    path: /api/v1/structure/sync
    purpose: Update tracked file structure
    input_schema: |
      {
        files?: string[],              // explicit file list
        structure?: object,            // file tree object
        patterns?: string[],           // glob patterns
        remove_deleted?: boolean       // clean up deleted files
      }
    output_schema: |
      {
        added: int,
        removed: int,
        moved: int,
        total: int,
        snapshot_id: UUID
      }
      
  - method: GET
    path: /api/v1/prioritize/least-visited
    purpose: Get files needing attention by visit count
    input_schema: |
      {
        limit?: int,
        context?: string,              // filter by visit context
        include_unvisited?: boolean,
        patterns?: string[]            // filter by patterns
      }
    output_schema: |
      {
        files: TrackedFile[],
        coverage: {
          visited: int,
          unvisited: int,
          percentage: float
        }
      }
      
  - method: GET
    path: /api/v1/prioritize/most-stale
    purpose: Get files by staleness score
    input_schema: |
      {
        limit?: int,
        threshold?: float,             // minimum staleness score
        patterns?: string[]
      }
    output_schema: |
      {
        files: TrackedFile[],
        average_staleness: float,
        critical_count: int           // files above critical threshold
      }
      
  - method: POST
    path: /api/v1/import
    purpose: Import visited map from external source
    input_schema: |
      {
        format: "json" | "csv",
        data: string | object,
        merge_strategy: "replace" | "combine" | "max"
      }
    output_schema: |
      {
        imported: int,
        updated: int,
        conflicts: int
      }
      
  - method: GET
    path: /api/v1/export
    purpose: Export current visited state
    input_schema: |
      {
        format: "json" | "csv",
        include_history?: boolean,
        patterns?: string[]
      }
    output_schema: |
      {
        format: string,
        data: string | object,
        exported_count: int,
        export_timestamp: timestamp
      }
      
  - method: GET
    path: /api/v1/coverage
    purpose: Get coverage statistics
    input_schema: |
      {
        patterns?: string[],
        group_by?: "directory" | "extension" | "context"
      }
    output_schema: |
      {
        total_files: int,
        visited_files: int,
        unvisited_files: int,
        coverage_percentage: float,
        average_visits: float,
        average_staleness: float,
        groups?: object                // if group_by specified
      }
```

### Event Interface
```yaml
published_events:
  - name: file.visited
    payload: { file_id: UUID, path: string, visit_count: int, context: string }
    subscribers: [monitoring-scenarios, coverage-trackers]
    
  - name: file.stale_detected
    payload: { file_id: UUID, path: string, staleness_score: float, last_modified: timestamp }
    subscribers: [alert-scenarios, priority-queues]
    
  - name: coverage.milestone
    payload: { percentage: float, visited: int, total: int }
    subscribers: [notification-scenarios, dashboard-scenarios]
    
consumed_events:
  - name: file_system.changes
    action: Update modification times and recalculate staleness
```

## 🖥️ CLI Interface Contract

### Command Structure
```yaml
cli_binary: visited-tracker
install_script: cli/install.sh

required_commands:
  - name: status
    description: Show tracking status and coverage summary
    flags: [--json, --verbose, --context]
    
  - name: help
    description: Display command help and usage
    flags: [--all, --command <name>]
    
  - name: version
    description: Show CLI and API version information
    flags: [--json]

custom_commands:
  - name: visit
    description: Record file visit(s)
    api_endpoint: /api/v1/visit
    arguments:
      - name: files
        type: string[]
        required: true
        description: File paths or patterns to mark as visited
    flags:
      - name: --context
        description: Visit context (security/performance/bug/docs)
      - name: --agent
        description: AI model or agent name
      - name: --conversation
        description: Conversation ID for grouping
    output: Visit confirmation with updated counts
    
  - name: sync
    description: Sync file structure and detect changes
    api_endpoint: /api/v1/structure/sync
    arguments: []
    flags:
      - name: --patterns
        description: Glob patterns for files to track
      - name: --structure
        description: JSON file with structure object
      - name: --remove-deleted
        description: Remove deleted files from tracking
    output: Sync summary with added/removed/moved counts
    
  - name: least-visited
    description: Get prioritized list of least visited files
    api_endpoint: /api/v1/prioritize/least-visited
    arguments: []
    flags:
      - name: --limit
        description: Number of files to return
      - name: --context
        description: Filter by visit context
      - name: --include-unvisited
        description: Include never-visited files
      - name: --json
        description: Output raw JSON
    output: Prioritized file list with visit counts
    
  - name: most-stale
    description: Get files with highest staleness scores
    api_endpoint: /api/v1/prioritize/most-stale
    arguments: []
    flags:
      - name: --limit
        description: Number of files to return
      - name: --threshold
        description: Minimum staleness score
      - name: --json
        description: Output raw JSON
    output: Files ranked by staleness with scores
    
  - name: coverage
    description: Show visit coverage statistics
    api_endpoint: /api/v1/coverage
    arguments: []
    flags:
      - name: --patterns
        description: Filter by file patterns
      - name: --group-by
        description: Group by directory/extension/context
      - name: --json
        description: Output raw JSON
    output: Coverage report with percentages and averages
    
  - name: import
    description: Import visited data from file
    api_endpoint: /api/v1/import
    arguments:
      - name: file
        type: string
        required: true
        description: Path to import file (JSON/CSV)
    flags:
      - name: --format
        description: File format (auto-detect if not specified)
      - name: --merge
        description: Merge strategy (replace/combine/max)
    output: Import summary with counts
    
  - name: export
    description: Export visited data to file
    api_endpoint: /api/v1/export
    arguments:
      - name: file
        type: string
        required: true
        description: Output file path
    flags:
      - name: --format
        description: Export format (json/csv)
      - name: --include-history
        description: Include visit history
      - name: --patterns
        description: Filter files to export
    output: Export confirmation with file count
```

## 🔄 Integration Requirements

### Upstream Dependencies
- **PostgreSQL**: Essential for persistent visit tracking and staleness calculation
- **File System Access**: Required for modification time checks and structure scanning

### Downstream Enablement
- **Security Audit Scenarios**: Query unvisited files for comprehensive scanning
- **Performance Review Scenarios**: Track which files have been profiled
- **Bug Hunt Scenarios**: Ensure systematic debugging coverage
- **Documentation Scenarios**: Know which files need doc updates
- **Code Quality Scenarios**: Prioritize technical debt by staleness

### Cross-Scenario Interactions
```yaml
provides_to:
  - scenario: security-scanner
    capability: Track security audit coverage and prioritize unscanned files
    interface: API + CLI
    
  - scenario: performance-optimizer
    capability: Identify performance hotspots needing review
    interface: API staleness scores
    
  - scenario: bug-hunter
    capability: Systematic debugging with coverage guarantee
    interface: API + visit context tracking
    
  - scenario: documentation-generator
    capability: Track documentation coverage per file
    interface: API + CLI with context filtering
    
consumes_from:
  - scenario: git-manager
    capability: Get modification history for staleness calculation
    fallback: File system modification times
```

## 🎨 Style and Branding Requirements

### UI/UX Style Guidelines
```yaml
style_profile:
  category: technical
  inspiration: Code coverage tools meets heatmap visualization
  
  visual_style:
    color_scheme: dark with heat gradient (blue=cold/unvisited, red=hot/stale)
    typography: technical monospace
    layout: dashboard with treemap/heatmap
    animations: subtle transitions on data updates
  
  personality:
    tone: analytical
    mood: focused
    target_feeling: "Complete visibility into code coverage"

style_references:
  technical:
    - codecov: "Coverage percentage badges and sunburst charts" 
    - github-contributions: "Activity heatmap visualization"
    
  inspiration: "NYC coverage reporter meets GitHub activity tracker with staleness alerts"
```

### Target Audience Alignment
- **Primary Users**: AI agents and developers doing systematic code reviews
- **User Expectations**: Dense information display, quick prioritization, clear coverage metrics
- **Accessibility**: Keyboard navigation, high contrast mode, screen reader support
- **Responsive Design**: Desktop-first for code review, API-first for agent integration

### Brand Consistency Rules
- **Scenario Identity**: Professional analysis tool focused on systematic coverage
- **Vrooli Integration**: Foundational capability that other scenarios build upon
- **Professional vs Fun**: Strongly professional - critical infrastructure for code quality

## 💰 Value Proposition

### Business Value
- **Primary Value**: Prevents bugs by ensuring no code escapes review, reduces security vulnerabilities by 40-60%
- **Revenue Potential**: $20K - $35K per deployment (enterprise security/quality teams)
- **Cost Savings**: Prevents production bugs that cost $5K-$50K each to fix
- **Market Differentiator**: First tool to combine visit tracking with staleness detection for AI agents

### Technical Value
- **Reusability Score**: Extreme - every analysis scenario can leverage this
- **Complexity Reduction**: Transforms chaotic ad-hoc reviews into systematic coverage
- **Innovation Enablement**: Enables true "comprehensive" analysis previously impossible

## 🧬 Evolution Path

### Version 1.0 (Current)
- Core visit tracking with counts
- Staleness detection and scoring
- Structure sync and change detection
- REST API and CLI interface
- Basic coverage dashboard

### Version 2.0 (Planned)  
- Git integration for precise modification tracking
- Multi-repository federation
- ML-based staleness prediction
- IDE plugin integration
- Advanced visualization (3D codebases, VR review)

### Long-term Vision
- **Predictive Analysis**: AI predicts which files will become problematic
- **Autonomous Review**: Agents automatically review stale files
- **Cross-Project Intelligence**: Learn patterns across all Vrooli projects
- **Becomes the Memory**: Central memory system for all code analysis in Vrooli

## 🔄 Scenario Lifecycle Integration

### Direct Scenario Deployment
```yaml
direct_execution:
  supported: true
  structure_compliance:
    - service.json with PostgreSQL dependency
    - API server with staleness calculation
    - CLI tool with prioritization commands
    - UI dashboard with heatmap visualization
    
  deployment_targets:
    - local: Docker Compose with PostgreSQL
    - kubernetes: Helm chart with persistent volumes
    - cloud: AWS RDS + Lambda for serverless staleness calculation
    
  revenue_model:
    - type: subscription
    - pricing_tiers: 
      - Team: $299/month (up to 100K files)  
      - Enterprise: $1499/month (unlimited, advanced analytics)
    - trial_period: 30 days
```

### Capability Discovery
```yaml
discovery:
  registry_entry:
    name: visited-tracker
    category: analysis
    capabilities: 
      - visit_count_tracking
      - staleness_detection
      - coverage_guarantee
      - intelligent_prioritization
      - cross_model_memory
    interfaces:
      - api: http://localhost:20252/api/v1
      - cli: visited-tracker
      - events: file.* namespace
      
  metadata:
    description: "File visit tracking with staleness detection for systematic code analysis"
    keywords: [visit-tracking, staleness, coverage, prioritization, code-analysis]
    dependencies: [postgres]
    enhances: [all-analysis-scenarios, security-scenarios, quality-scenarios]
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
| Performance degradation with huge codebases | Medium | Medium | Efficient indexing, Redis cache, pagination |
| Staleness calculation accuracy | Low | Medium | Multiple calculation methods, tunable weights |
| File system changes during tracking | High | Low | Regular structure sync, change detection |
| Cross-platform path issues | Medium | Low | Path normalization, absolute path storage |

### Operational Risks
- **Drift Prevention**: PRD serves as single source of truth, validated by tests
- **Data Loss**: Regular backups, export capability for disaster recovery
- **Resource Conflicts**: PostgreSQL shared safely via resource management
- **Privacy**: No file contents stored, only metadata and visit counts

## ✅ Validation Criteria

### Declarative Test Specification
```yaml
version: 1.0
scenario: visited-tracker

structure:
  required_files:
    - .vrooli/service.json
    - PRD.md
    - api/main.go
    - api/go.mod
    - cli/visited-tracker
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
      
  - name: "Visit recording works"
    type: http
    service: api
    endpoint: /api/v1/visit
    method: POST
    body:
      files: ["test.js", "main.go"]
      context: "security"
    expect:
      status: 200
      body:
        recorded: 2
        
  - name: "Staleness calculation works"
    type: http
    service: api
    endpoint: /api/v1/prioritize/most-stale
    method: GET
    params:
      limit: 5
    expect:
      status: 200
      body:
        files: "[array-with-length-max-5]"
        
  - name: "CLI visit command executes"
    type: exec
    command: ./cli/visited-tracker visit test.js --context security
    expect:
      exit_code: 0
      output_contains: ["recorded", "test.js"]
      
  - name: "Database schema is initialized"
    type: sql
    service: postgres
    query: "SELECT COUNT(*) FROM information_schema.tables WHERE table_name IN ('tracked_files', 'visits', 'structure_snapshots')"
    expect:
      rows: 
        - count: 3
```

### Performance Validation
- [ ] Visit recording under 50ms for single file
- [ ] Staleness calculation under 100ms for 10K files
- [ ] Structure sync handles 50K+ files efficiently
- [ ] UI dashboard remains responsive with large datasets

### Integration Validation
- [ ] Discoverable via Vrooli resource registry
- [ ] All API endpoints documented and functional
- [ ] All CLI commands executable with comprehensive --help
- [ ] Events published correctly for downstream consumption

### Capability Verification
- [ ] Successfully tracks visits across conversation restarts
- [ ] Accurately calculates staleness based on modifications
- [ ] Structure sync handles file additions/deletions/moves
- [ ] Prioritization APIs return meaningful results
- [ ] Coverage metrics accurately reflect codebase state

## 📝 Implementation Notes

### Design Decisions
**Visit Tracking vs Campaign Tracking**: Shifted from campaigns to visit counts
- Alternative considered: Keep campaign-based approach
- Decision driver: Visit counts more flexible for varied use cases
- Trade-offs: Slightly different mental model but much more powerful

**Staleness Formula**: Time × modifications / (visits + 1)
- Alternative considered: Simple time since last visit
- Decision driver: Need to account for both time and change frequency
- Trade-offs: More complex but much more accurate prioritization

### Known Limitations
- **File Content**: Does not analyze file contents, only metadata
  - Workaround: Integrate with code analysis tools via API
  - Future fix: Optional AST analysis for deeper insights

- **Real-time Tracking**: Batch updates, not real-time
  - Workaround: Frequent syncs, webhook integration
  - Future fix: File system watchers for instant updates

### Security Considerations
- **Data Protection**: Visit history may reveal sensitive patterns - encrypt at rest
- **Access Control**: Token-based auth planned for v2
- **Audit Trail**: All visits logged with full context for compliance

## 🔗 References

### Documentation
- README.md - User-facing overview and quick start
- docs/api.md - Complete API specification
- docs/cli.md - CLI command reference
- docs/integration.md - How scenarios leverage visit tracking

### Related PRDs
- security-scanner: Primary consumer of visit tracking
- performance-optimizer: Uses staleness for hotspot detection

### External Resources
- [Git File History](https://git-scm.com/book/en/v2/Git-Basics-Viewing-the-Commit-History) - Modification tracking patterns
- [Code Coverage Tools](https://github.com/istanbuljs/nyc) - Coverage visualization inspiration
- [GitHub Activity Heatmap](https://docs.github.com/en/account-and-profile/setting-up-and-managing-your-github-profile/managing-contribution-settings-on-your-profile/viewing-contributions-on-your-profile) - Heatmap visualization reference

---

**Last Updated**: 2025-09-05  
**Status**: Active Development  
**Owner**: Claude Code Assistant  
**Review Cycle**: Validate against implementation after each major milestone