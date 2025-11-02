# Product Requirements Document (PRD)

## 🎯 Capability Definition

### Core Capability
**What permanent capability does this scenario add to Vrooli?**
Code Tidiness Manager adds automated technical debt detection, cleanup suggestion generation, and maintenance intelligence to Vrooli. It continuously scans the codebase for inefficiencies, redundancies, and cleanup opportunities, generating both automated cleanup scripts for simple issues and detailed analysis for complex architectural improvements.

### Intelligence Amplification
**How does this capability make future agents smarter?**
This capability provides agents with:
- **Clean workspace guarantee**: Agents can trust the codebase is maintained, reducing context noise
- **Pattern recognition**: Learns what cleanup patterns are accepted/rejected, improving suggestions
- **Technical debt awareness**: Agents can query accumulated debt before making architectural decisions
- **Resource optimization**: Identifies underutilized resources that agents can repurpose
- **Code quality baseline**: Establishes standards that all future code generation follows

### Recursive Value
**What new scenarios become possible after this exists?**
1. **code-quality-enforcer**: Automated PR reviews with quality gates based on tidiness standards
2. **resource-optimizer**: Dynamic resource allocation based on actual usage patterns detected
3. **scenario-deduplication-manager**: Intelligent merging of overlapping scenario capabilities
4. **legacy-code-modernizer**: Systematic upgrade of outdated patterns to modern approaches
5. **performance-bottleneck-hunter**: Deep analysis of inefficiencies identified by tidiness scans

## 📊 Success Metrics

### Functional Requirements
- **Must Have (P0)**
  - [ ] Scan entire Vrooli codebase for cleanup opportunities (backup files, temp files, empty dirs)
  - [ ] Generate safe cleanup scripts for automated issues with confidence scoring
  - [ ] Detect complex issues requiring human judgment (duplicate scenarios, architectural drift)
  - [ ] Provide REST API for other scenarios to request targeted scans
  - [ ] Store scan history and track accepted/rejected suggestions for learning
  - [ ] CLI interface for manual triggering and configuration
  
- **Should Have (P1)**
  - [ ] Scheduled automatic scanning with configurable frequency
  - [ ] Integration with git hooks for pre-commit cleanup suggestions
  - [ ] Dashboard UI showing cleanup categories, counts, and trends
  - [ ] Batch operations for applying multiple similar cleanups
  - [ ] Custom rule registration API for scenarios to define their own cleanup patterns
  - [ ] Notification system for critical technical debt accumulation
  
- **Nice to Have (P2)**
  - [ ] Machine learning model to predict cleanup acceptance likelihood
  - [ ] "Spring cleaning mode" for comprehensive deep scans
  - [ ] Integration with CI/CD for automated cleanup in pipelines
  - [ ] Cost analysis showing resource waste from inefficiencies
  - [ ] Gamification with cleanliness scores and leaderboards

### Performance Criteria
| Metric | Target | Measurement Method |
|--------|--------|-------------------|
| Full Scan Time | < 60s for entire codebase | Time from start to completion |
| Suggestion Generation | < 500ms per issue | API response time |
| False Positive Rate | < 5% for automated cleanups | Rejected suggestions / total |
| Memory Usage | < 512MB during scan | Resource monitoring |
| Cleanup Success Rate | > 95% for automated scripts | Successful executions / attempts |

### Quality Gates
- [ ] All P0 requirements implemented and tested
- [ ] Can scan and analyze 10,000+ files without crashing
- [ ] Zero data loss from cleanup operations
- [ ] API response times consistently under 1 second
- [ ] CLI provides comprehensive help and examples

## 🏗️ Technical Architecture

### Resource Dependencies
```yaml
required:
  - resource_name: postgres
    purpose: Store scan history, patterns, and learning data
    integration_pattern: Direct database access for persistence
    access_method: resource-postgres CLI and SQL queries
    
  - resource_name: redis
    purpose: Cache scan results and temporary analysis data
    integration_pattern: Key-value storage for performance
    access_method: resource-redis CLI commands
    
optional:
  - resource_name: qdrant
    purpose: Vector storage for code similarity detection
    fallback: Skip advanced duplicate detection features
    access_method: API for embedding storage/retrieval
    
  - resource_name: ollama
    purpose: AI-powered code analysis and suggestion refinement
    fallback: Use rule-based analysis only
    access_method: Shared workflow ollama.json
```

### Resource Integration Standards
```yaml
integration_priorities:
  1_shared_workflows:
    - workflow: ollama.json
      location: initialization/automation/n8n/
      purpose: Intelligent analysis of complex code patterns
    - workflow: rate-limiter.json
      location: initialization/automation/n8n/
      purpose: Throttle scan operations to avoid system overload
  
  2_resource_cli:
    - command: resource-postgres query
      purpose: Store and retrieve scan history
    - command: resource-redis get/set
      purpose: Cache intermediate scan results
  
  3_direct_api:
    - justification: Real-time scan progress updates
      endpoint: /api/v1/tidiness/scan/progress

shared_workflow_criteria:
  - code-scanner.json will be created for reusable scanning logic
  - Will be used by: code-quality-enforcer, legacy-code-modernizer
  - Provides parameterized scanning with custom rule sets
```

### Data Models
```yaml
primary_entities:
  - name: ScanResult
    storage: postgres
    schema: |
      {
        id: UUID
        scan_id: UUID
        timestamp: DateTime
        issue_type: string // backup_file|duplicate_code|unused_import|etc
        severity: string // low|medium|high|critical
        file_path: string
        description: text
        cleanup_script: text // nullable for complex issues
        confidence_score: float
        requires_human_review: boolean
        metadata: jsonb
      }
    relationships: Links to ScanHistory, CleanupAction
    
  - name: CleanupAction
    storage: postgres
    schema: |
      {
        id: UUID
        scan_result_id: UUID
        action_taken: string // accepted|rejected|modified|deferred
        executed_at: DateTime
        execution_result: jsonb
        user_feedback: text
      }
    relationships: Belongs to ScanResult
    
  - name: CleanupRule
    storage: postgres  
    schema: |
      {
        id: UUID
        rule_name: string
        pattern: string // regex or glob
        action_template: text
        priority: integer
        enabled: boolean
        created_by: string // scenario that registered it
      }
    relationships: Used by ScanEngine
```

### API Contract
```yaml
endpoints:
  - method: POST
    path: /api/v1/tidiness/scan
    purpose: Trigger a new scan with optional filters
    input_schema: |
      {
        paths?: string[]  // Specific paths to scan
        types?: string[]  // Issue types to detect
        deep_scan?: boolean
        exclude_patterns?: string[]
      }
    output_schema: |
      {
        scan_id: string
        status: "started"
        estimated_time: number
      }
    sla:
      response_time: 500ms
      availability: 99.9%
      
  - method: GET
    path: /api/v1/tidiness/suggestions
    purpose: Retrieve cleanup suggestions from latest scan
    input_schema: |
      {
        scan_id?: string
        severity?: string
        limit?: number
        offset?: number
      }
    output_schema: |
      {
        suggestions: [{
          id: string
          type: string
          description: string
          cleanup_script?: string
          confidence: number
          files_affected: string[]
          requires_human_review: boolean
        }]
        total: number
      }
      
  - method: POST
    path: /api/v1/tidiness/cleanup
    purpose: Execute cleanup scripts
    input_schema: |
      {
        suggestion_ids: string[]
        dry_run?: boolean
      }
    output_schema: |
      {
        executed: number
        skipped: number
        errors: string[]
      }
```

### Event Interface
```yaml
published_events:
  - name: tidiness.scan.completed
    payload: { scan_id, issues_found, duration }
    subscribers: code-quality-enforcer, scenario-health-monitor
    
  - name: tidiness.cleanup.executed
    payload: { suggestion_id, result, files_modified }
    subscribers: git-manager, deployment-manager
    
  - name: tidiness.debt.threshold
    payload: { debt_score, categories, alert_level }
    subscribers: product-manager-agent, system-monitor
    
consumed_events:
  - name: scenario.created
    action: Register new scenario's cleanup patterns
    
  - name: git.pre_commit
    action: Run targeted scan on changed files
```

## 🖥️ CLI Interface Contract

### Command Structure
```yaml
cli_binary: code-tidiness-manager
install_script: cli/install.sh

required_commands:
  - name: status
    description: Show scan history and system health
    flags: [--json, --verbose]
    
  - name: help
    description: Display command help and usage
    flags: [--all, --command <name>]
    
  - name: version
    description: Show CLI and API version
    flags: [--json]

custom_commands:
  - name: scan
    description: Run a tidiness scan
    api_endpoint: /api/v1/tidiness/scan
    arguments:
      - name: path
        type: string
        required: false
        description: Specific path to scan (default: entire codebase)
    flags:
      - name: --types
        description: Comma-separated issue types to detect
      - name: --deep
        description: Enable deep scan for complex patterns
      - name: --exclude
        description: Patterns to exclude from scan
    output: Scan ID and progress updates
    
  - name: suggestions
    description: List cleanup suggestions
    api_endpoint: /api/v1/tidiness/suggestions
    flags:
      - name: --severity
        description: Filter by severity (low|medium|high|critical)
      - name: --limit
        description: Number of suggestions to show
      - name: --json
        description: Output as JSON
    output: Formatted list of suggestions
    
  - name: cleanup
    description: Execute cleanup suggestions
    api_endpoint: /api/v1/tidiness/cleanup
    arguments:
      - name: suggestion-ids
        type: string
        required: true
        description: Comma-separated suggestion IDs or "all"
    flags:
      - name: --dry-run
        description: Show what would be cleaned without executing
      - name: --force
        description: Skip confirmation prompts
    output: Cleanup execution results
    
  - name: register-rule
    description: Register custom cleanup rule
    arguments:
      - name: name
        type: string
        required: true
      - name: pattern
        type: string
        required: true
      - name: action
        type: string
        required: true
    output: Rule registration confirmation
```

### CLI-API Parity Requirements
- Every API endpoint has corresponding CLI command
- CLI provides both interactive and scriptable modes
- JSON output available for all commands
- Progress indicators for long-running operations

### Implementation Standards
```yaml
implementation_requirements:
  - architecture: Go binary wrapping lib/ functions
  - dependencies: Minimal - uses standard library primarily
  - error_handling: Clear error messages with suggested fixes
  - configuration: 
      - ~/.vrooli/code-tidiness-manager/config.yaml
      - Environment variables: CTM_* prefix
      - Command flags override all
  
installation:
  - install_script: Creates symlink in ~/.vrooli/bin/
  - path_update: Adds to PATH if needed
  - permissions: 755 on binary
  - documentation: Comprehensive --help
```

## 🔄 Integration Requirements

### Upstream Dependencies
**What capabilities must exist before this can function?**
- **File System Access**: Read access to entire Vrooli codebase
- **PostgreSQL**: Persistence layer for scan history and learning
- **Redis**: Caching for performance optimization
- **Git**: Understanding of repository structure and history

### Downstream Enablement
**What future capabilities does this unlock?**
- **Automated Code Quality**: Enables quality gates and enforcement
- **Resource Optimization**: Identifies waste for optimization scenarios
- **Technical Debt Management**: Quantifies and tracks debt over time
- **Intelligent Refactoring**: Provides data for large-scale improvements

### Cross-Scenario Interactions
```yaml
provides_to:
  - scenario: git-manager
    capability: Pre-commit cleanup suggestions
    interface: Event/API
    
  - scenario: scenario-health-monitor
    capability: Technical debt metrics
    interface: API
    
  - scenario: code-quality-enforcer
    capability: Quality baseline and rules
    interface: API/Events
    
consumes_from:
  - scenario: ecosystem-manager
    capability: New scenario notifications
    fallback: Periodic full scans
    
  - scenario: resource-monitor
    capability: System load metrics
    fallback: Run without throttling
```

## 🎨 Style and Branding Requirements

### UI/UX Style Guidelines
```yaml
style_profile:
  category: professional
  inspiration: GitHub Insights crossed with Marie Kondo minimalism
  
  visual_style:
    color_scheme: light with accent colors for severity
    typography: modern, clean sans-serif
    layout: dashboard with cards and charts
    animations: subtle transitions, progress animations
  
  personality:
    tone: helpful and encouraging
    mood: calm and organized
    target_feeling: "My codebase is under control"

style_references:
  professional: 
    - "Clean, data-driven dashboard like GitHub Insights"
    - "Color coding: green=clean, yellow=attention, red=action needed"
  unique_elements:
    - "Cleanup progress as satisfying animations"
    - "Before/after visualization of codebase health"
    - "Gamified cleanliness score with trends"
```

### Target Audience Alignment
- **Primary Users**: Developers, DevOps engineers, AI agents
- **User Expectations**: Professional tool with clear actionable insights
- **Accessibility**: WCAG AA compliance, keyboard navigation
- **Responsive Design**: Desktop-first, tablet supported

### Brand Consistency Rules
- Professional design reflecting importance of code quality
- Clean, organized interface mirroring the goal of clean code
- Positive reinforcement for cleanup actions
- Non-judgmental presentation of technical debt

## 💰 Value Proposition

### Business Value
- **Primary Value**: Reduces maintenance costs by 30-40%
- **Revenue Potential**: $15K - $30K per enterprise deployment
- **Cost Savings**: 10-20 developer hours per month
- **Market Differentiator**: Self-learning cleanup intelligence

### Technical Value
- **Reusability Score**: 9/10 - Every scenario benefits from clean code
- **Complexity Reduction**: Makes large codebases manageable
- **Innovation Enablement**: Clean code accelerates development

## 🧬 Evolution Path

### Version 1.0 (Current)
- Core scanning engine with rule-based detection
- Basic cleanup script generation
- API and CLI interfaces
- Simple web dashboard

### Version 2.0 (Planned)
- ML-based pattern recognition
- Cross-scenario duplicate detection
- Automated refactoring suggestions
- Integration with CI/CD pipelines

### Long-term Vision
- Fully autonomous code maintenance
- Predictive technical debt prevention
- Cross-repository analysis for enterprises
- Industry-specific cleanup patterns

## 🔄 Scenario Lifecycle Integration

### Direct Scenario Deployment
```yaml
direct_execution:
  supported: true
  structure_compliance:
    - service.json with resource dependencies
    - Database initialization scripts
    - n8n workflows for scanning
    - Health check endpoints
    
  deployment_targets:
    - local: Docker Compose with postgres/redis
    - kubernetes: StatefulSet for persistence
    - cloud: Lambda for serverless scanning
    
  revenue_model:
    - type: subscription
    - pricing_tiers:
        - free: 100 scans/month
        - pro: Unlimited scans, $99/month
        - enterprise: Custom rules, $499/month
    - trial_period: 14 days
```

### Capability Discovery
```yaml
discovery:
  registry_entry:
    name: code-tidiness-manager
    category: maintenance
    capabilities: 
      - technical_debt_detection
      - cleanup_automation
      - code_quality_metrics
    interfaces:
      - api: http://localhost:{port}/api/v1/tidiness
      - cli: code-tidiness-manager
      - events: tidiness.*
      
  metadata:
    description: Automated code cleanup and technical debt management
    keywords: [cleanup, maintenance, quality, technical-debt, refactoring]
    dependencies: [postgres, redis]
    enhances: [all scenarios benefit from clean code]
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
| False positive cleanups | Medium | High | Confidence scoring, dry-run mode |
| Performance impact on large codebases | Low | Medium | Incremental scanning, caching |
| Data loss from cleanup | Low | Critical | Backup before cleanup, rollback capability |

### Operational Risks
- **Over-aggressive cleaning**: Conservative defaults, human review for complex issues
- **Resource contention**: Throttling, off-peak scheduling
- **Suggestion fatigue**: Prioritization, batching similar issues

## ✅ Validation Criteria

### Declarative Test Specification
```yaml
version: 1.0
scenario: code-tidiness-manager

structure:
  required_files:
    - .vrooli/service.json
    - PRD.md
    - api/main.go
    - api/go.mod
    - cli/code-tidiness-manager
    - cli/install.sh
    - initialization/storage/postgres/schema.sql
    - test/run-tests.sh
    
  required_dirs:
    - api
    - cli
    - initialization
    - initialization/automation/n8n
    - initialization/storage/postgres
    - ui
    - lib
    - docs

resources:
  required: [postgres, redis]
  optional: [qdrant, ollama]
  health_timeout: 60

tests:
  - name: "PostgreSQL is accessible"
    type: http
    service: postgres
    endpoint: /health
    method: GET
    expect:
      status: 200
      
  - name: "Scan API endpoint responds"
    type: http
    service: api
    endpoint: /api/v1/tidiness/scan
    method: POST
    body:
      paths: ["test/"]
    expect:
      status: 201
      body:
        status: "started"
        
  - name: "CLI status command works"
    type: exec
    command: ./cli/code-tidiness-manager status --json
    expect:
      exit_code: 0
      output_contains: ["healthy"]
      
  - name: "Database schema initialized"
    type: sql
    service: postgres
    query: "SELECT COUNT(*) FROM information_schema.tables WHERE table_name IN ('scan_results', 'cleanup_actions', 'cleanup_rules')"
    expect:
      rows: 
        - count: 3
        
  - name: "Can detect backup files"
    type: exec
    command: ./cli/code-tidiness-manager scan --types backup_files test/
    expect:
      exit_code: 0
      output_contains: ["scan_id"]
```

### Test Execution Gates
```bash
./test.sh --scenario code-tidiness-manager --validation complete
```

### Performance Validation
- [ ] Scan completes in under 60 seconds for 10,000 files
- [ ] API responses under 500ms for 95th percentile
- [ ] Memory usage stays under 512MB
- [ ] Zero data corruption in cleanup operations

### Integration Validation
- [ ] Discoverable via scenario registry
- [ ] All API endpoints return expected schemas
- [ ] CLI commands provide comprehensive help
- [ ] Events published to message bus
- [ ] Cleanup suggestions are actionable

### Capability Verification
- [ ] Detects all common cleanup patterns
- [ ] Generates safe cleanup scripts
- [ ] Tracks suggestion acceptance/rejection
- [ ] Provides useful technical debt metrics
- [ ] UI clearly shows cleanup opportunities

## 📝 Implementation Notes

### Design Decisions
**Static Analysis vs Runtime**: Chose static analysis for safety and predictability
- Alternative considered: Runtime monitoring
- Decision driver: Safer to analyze without execution
- Trade-offs: May miss runtime-only issues

**Rule-based vs ML**: Starting with rules, adding ML in v2
- Alternative considered: ML from start
- Decision driver: Predictability and explainability
- Trade-offs: Less sophisticated initially

### Known Limitations
- **Large binary files**: Skipped to avoid memory issues
  - Workaround: Separate binary cleanup mode
  - Future fix: Streaming analysis in v2
  
- **Cross-repository dependencies**: Cannot detect
  - Workaround: Manual dependency mapping
  - Future fix: Repository relationship tracking

### Security Considerations
- **Read-only by default**: Explicit permission for cleanup
- **Audit trail**: All cleanups logged with rollback info
- **No credential scanning**: Avoid security tool overlap

## 🔗 References

### Documentation
- README.md - User guide and quick start
- docs/api.md - Complete API specification
- docs/rules.md - Cleanup rule documentation
- docs/patterns.md - Common pattern library

### Related PRDs
- scenario-health-monitor - Consumes tidiness metrics
- git-manager - Integrates with pre-commit
- code-quality-enforcer - Uses tidiness as baseline

### External Resources
- Clean Code principles by Robert Martin
- Technical Debt Quadrant by Martin Fowler
- SonarQube rule definitions for inspiration

---

**Last Updated**: 2024-09-04  
**Status**: Draft  
**Owner**: AI Agent  
**Review Cycle**: Weekly validation against implementation
