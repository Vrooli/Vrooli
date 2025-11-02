# Product Requirements Document (PRD)

## 🎯 Capability Definition

### Core Capability
**What permanent capability does this scenario add to Vrooli?**
A self-improving code quality guardian that continuously detects, tracks, and fixes code smell violations across the entire Vrooli codebase. This includes both general best practices and Vrooli-specific patterns, with intelligent auto-fixing for safe changes and a review queue for risky modifications.

### Intelligence Amplification
**How does this capability make future agents smarter?**
Every code smell pattern discovered and fixed becomes a permanent rule that prevents future scenarios from making the same mistakes. Agents learn what patterns work and what patterns break systems, accumulating institutional knowledge about code quality that compounds over time.

### Recursive Value
**What new scenarios become possible after this exists?**
- **code-review-assistant**: Automated PR reviews with learned quality patterns
- **technical-debt-manager**: Prioritizes and schedules refactoring based on smell metrics
- **scenario-quality-scorer**: Rates scenario code quality before deployment
- **pattern-library-builder**: Extracts good patterns from fixed code for reuse
- **migration-assistant**: Safely migrates old patterns to new standards

## 📊 Success Metrics

### Functional Requirements
- **Must Have (P0)**
  - [ ] Hot-reloadable rules engine that reads from JSON/YAML without restart
  - [ ] Integration with resource-claude-code for AI-powered pattern detection
  - [ ] Integration with visited-tracker to avoid redundant checks
  - [ ] Auto-fix capability for safe patterns (whitespace, imports, etc.)
  - [ ] Review queue UI for approving/denying dangerous changes
  - [ ] API endpoints for other scenarios to request code reviews
  - [ ] Detect and fix Vrooli-specific smells (hard-coded ports, paths)
  
- **Should Have (P1)**
  - [ ] Learning mode that tracks approved/rejected fixes to improve suggestions
  - [ ] Dashboard showing smell statistics across codebase
  - [ ] Rule editor UI for adding patterns without editing files
  - [ ] History view tracking what's been fixed over time
  - [ ] Batch processing mode for fixing multiple files
  - [ ] Configurable risk levels for different fix categories
  
- **Nice to Have (P2)**
  - [ ] Integration with git hooks for pre-commit checks
  - [ ] Smell trend analysis over time
  - [ ] Team-specific rule sets
  - [ ] Export reports in multiple formats

### Performance Criteria
| Metric | Target | Measurement Method |
|--------|--------|-------------------|
| Response Time | < 500ms for single file analysis | API monitoring |
| Throughput | 100 files/minute | Load testing |
| Accuracy | > 95% for pattern detection | Validation suite |
| Resource Usage | < 2GB memory, < 25% CPU | System monitoring |

### Quality Gates
- [ ] All P0 requirements implemented and tested
- [ ] Integration tests pass with resource-claude-code and visited-tracker
- [ ] Performance targets met under load
- [ ] Documentation complete (README, API docs, CLI help)
- [ ] Scenario can be invoked by other agents via API/CLI

## 🏗️ Technical Architecture

### Resource Dependencies
```yaml
required:
  - resource_name: resource-claude-code
    purpose: AI-powered pattern detection beyond regex
    integration_pattern: Direct CLI invocation for complex analysis
    access_method: CLI command - resource-claude-code analyze
    
  - resource_name: postgres
    purpose: Store rules, history, and learning data
    integration_pattern: Direct database access
    access_method: CLI - resource-postgres query
    
optional:
  - resource_name: redis
    purpose: Cache analysis results for performance
    fallback: In-memory cache with shorter TTL
    access_method: CLI - resource-redis get/set
  
  - resource_name: visited-tracker
    purpose: Avoid re-analyzing unchanged files
    fallback: Simple timestamp tracking
    access_method: API - GET /api/v1/visited/check
```

### Resource Integration Standards
```yaml
integration_priorities:
  1_shared_workflows:     # Not applicable - avoiding n8n per requirements
    - note: Moving away from n8n workflows per user directive
  
  2_resource_cli:        # Primary integration method
    - command: resource-claude-code analyze --file <path> --rules <ruleset>
      purpose: Complex pattern analysis using AI
    - command: resource-postgres query --db code_smell
      purpose: Store and retrieve rules and history
    - command: visited-tracker check --path <path>
      purpose: Skip unchanged files
  
  3_direct_api:          # When CLI unavailable
    - justification: Real-time updates to UI require websocket/API
      endpoint: /api/v1/visited/track
```

### Data Models
```yaml
primary_entities:
  - name: Rule
    storage: postgres
    schema: |
      {
        id: UUID
        name: string
        pattern: string (regex or AI prompt)
        category: enum (auto-fix|needs-approval|info-only)
        risk_level: enum (safe|moderate|dangerous)
        vrooli_specific: boolean
        fix_template: string (optional)
        examples: json[]
        created_at: timestamp
        updated_at: timestamp
        enabled: boolean
      }
    relationships: Applied to Violations, tracked in History

  - name: Violation
    storage: postgres
    schema: |
      {
        id: UUID
        rule_id: UUID
        file_path: string
        line_number: integer
        column_number: integer
        severity: enum (error|warning|info)
        message: string
        suggested_fix: string
        auto_fixable: boolean
        status: enum (pending|fixed|ignored|approved)
        detected_at: timestamp
      }
    relationships: Belongs to Rule, tracked in History

  - name: FixHistory
    storage: postgres  
    schema: |
      {
        id: UUID
        violation_id: UUID
        action: enum (auto-fixed|approved|rejected|ignored)
        fix_applied: string
        applied_by: string (agent|user)
        applied_at: timestamp
        result: enum (success|failure|partial)
        error_message: string (optional)
      }
    relationships: References Violation

  - name: LearnedPattern
    storage: postgres
    schema: |
      {
        id: UUID
        pattern: string
        confidence: float
        positive_examples: integer
        negative_examples: integer  
        last_seen: timestamp
        rule_suggestion: json
      }
    relationships: Can generate new Rules
```

### API Contract
```yaml
endpoints:
  - method: POST
    path: /api/v1/code-smell/analyze
    purpose: Request code smell analysis for files/directories
    input_schema: |
      {
        paths: string[] (required)
        rules: string[] (optional, defaults to all enabled)
        auto_fix: boolean (optional, default false)
        risk_threshold: string (optional: safe|moderate|dangerous)
      }
    output_schema: |
      {
        violations: Violation[]
        auto_fixed: number
        needs_review: number
        total_files: number
        duration_ms: number
      }
    sla:
      response_time: 5000ms
      availability: 99%

  - method: GET
    path: /api/v1/code-smell/rules
    purpose: Get all configured rules
    output_schema: |
      {
        rules: Rule[]
        categories: string[]
        vrooli_specific_count: number
      }

  - method: POST
    path: /api/v1/code-smell/fix
    purpose: Apply a specific fix
    input_schema: |
      {
        violation_id: UUID
        action: enum (approve|reject|ignore)
        modified_fix: string (optional)
      }

  - method: GET
    path: /api/v1/code-smell/queue
    purpose: Get violations needing review
    output_schema: |
      {
        violations: Violation[]
        total: number
        by_severity: object
      }

  - method: POST
    path: /api/v1/code-smell/learn
    purpose: Submit pattern for learning system
    input_schema: |
      {
        pattern: string
        is_positive: boolean
        context: object
      }
```

### Event Interface
```yaml
published_events:
  - name: code-smell.analysis.completed
    payload: { path: string, violations: number, fixed: number }
    subscribers: [technical-debt-manager, ci-cd-healer]
    
  - name: code-smell.fix.applied
    payload: { file: string, rule: string, result: string }
    subscribers: [git-manager, test-genie]
    
  - name: code-smell.pattern.learned
    payload: { pattern: string, confidence: float }
    subscribers: [pattern-library-builder]
    
consumed_events:
  - name: scenario.created
    action: Automatically analyze new scenario code
  
  - name: git.commit.pending
    action: Run pre-commit smell check
```

## 🖥️ CLI Interface Contract

### Command Structure
```yaml
cli_binary: code-smell
install_script: cli/install.sh

required_commands:
  - name: status
    description: Show operational status and statistics
    flags: [--json, --verbose]
    
  - name: help
    description: Display command help and usage
    flags: [--all, --command <name>]
    
  - name: version
    description: Show CLI and API version information
    flags: [--json]

custom_commands:
  - name: analyze
    description: Analyze files/directories for code smells
    api_endpoint: /api/v1/code-smell/analyze
    arguments:
      - name: path
        type: string
        required: true
        description: File or directory to analyze
    flags:
      - name: --auto-fix
        description: Automatically fix safe violations
      - name: --rules
        description: Comma-separated list of rules to apply
      - name: --risk
        description: Maximum risk level for auto-fixes (safe|moderate|dangerous)
    output: List of violations found and fixed

  - name: fix
    description: Apply or reject a specific fix
    api_endpoint: /api/v1/code-smell/fix
    arguments:
      - name: violation-id
        type: string
        required: true
    flags:
      - name: --action
        description: Action to take (approve|reject|ignore)

  - name: rules
    description: Manage smell detection rules
    subcommands:
      - list: Show all rules
      - add: Add new rule from file
      - enable: Enable a rule
      - disable: Disable a rule
      - reload: Hot-reload rules from disk

  - name: queue
    description: Show violations awaiting review
    flags:
      - name: --severity
        description: Filter by severity
      - name: --file
        description: Filter by file pattern

  - name: stats
    description: Show code smell statistics
    flags:
      - name: --period
        description: Time period (day|week|month|all)
```

## 🔄 Integration Requirements

### Upstream Dependencies
- **resource-claude-code**: Provides AI-powered pattern analysis beyond regex
- **visited-tracker**: Prevents redundant analysis of unchanged files
- **postgres**: Persistent storage for rules and history

### Downstream Enablement
- **code-review-assistant**: Uses learned patterns for automated PR reviews
- **technical-debt-manager**: Prioritizes refactoring based on smell metrics
- **scenario-quality-scorer**: Rates code quality before deployment
- **ci-cd-healer**: Fixes build issues caused by code smells

### Cross-Scenario Interactions
```yaml
provides_to:
  - scenario: git-manager
    capability: Pre-commit code quality check
    interface: CLI - code-smell analyze --auto-fix
    
  - scenario: scenario-improvement-loop
    capability: Automated quality fixes during improvement
    interface: API - POST /api/v1/code-smell/analyze
    
  - scenario: agent-dashboard
    capability: Code quality metrics display
    interface: API - GET /api/v1/code-smell/stats

consumes_from:
  - scenario: visited-tracker
    capability: File change tracking
    fallback: Analyze all files (slower)
    
  - scenario: prompt-manager
    capability: Rule prompts for AI analysis
    fallback: Use built-in prompts
```

## 🎨 Style and Branding Requirements

### UI/UX Style Guidelines
```yaml
style_profile:
  category: technical
  inspiration: Modern code review tools (GitHub PR interface)
  
  visual_style:
    color_scheme: dark
    typography: monospace for code, modern sans for UI
    layout: dashboard with sidebar navigation
    animations: subtle transitions only
  
  personality:
    tone: technical but approachable
    mood: focused and efficient
    target_feeling: Confidence in code quality

ui_components:
  - Dashboard: Overview cards with smell metrics
  - Review Queue: Split-pane diff viewer
  - Rule Editor: Monaco editor with syntax highlighting
  - History Timeline: Visualization of fixes over time
  - Settings Panel: Clean form layout
```

### Target Audience Alignment
- **Primary Users**: Developers, DevOps engineers, AI agents
- **User Expectations**: GitHub-like interface, keyboard shortcuts
- **Accessibility**: WCAG AA compliance, keyboard navigation
- **Responsive Design**: Desktop primary, tablet supported

## 💰 Value Proposition

### Business Value
- **Primary Value**: Prevents technical debt accumulation saving 30% maintenance time
- **Revenue Potential**: $20K - $40K per enterprise deployment
- **Cost Savings**: 100+ developer hours/month in code review time
- **Market Differentiator**: Self-learning pattern detection unique to each codebase

### Technical Value
- **Reusability Score**: 10/10 - Every scenario benefits from code quality
- **Complexity Reduction**: Makes code review 80% faster
- **Innovation Enablement**: Allows focus on features instead of fixes

## 🧬 Evolution Path

### Version 1.0 (Current)
- Core smell detection with hot-reload rules
- Auto-fix for safe patterns
- Review queue UI
- Claude-code integration

### Version 2.0 (Planned)
- Machine learning pattern detection
- Git integration with hooks
- Team-specific rule sets
- Cross-repository analysis

### Long-term Vision
- Becomes the quality gatekeeper for all Vrooli code
- Learns organization-specific patterns and standards
- Prevents entire categories of bugs before they exist

## 🔄 Scenario Lifecycle Integration

### Direct Scenario Deployment
```yaml
direct_execution:
  supported: true
  structure_compliance:
    - service.json with code-smell metadata
    - Rules configuration in initialization/rules/
    - API and CLI fully functional
    - Health check at /api/v1/health
    
  deployment_targets:
    - local: Docker Compose with postgres
    - kubernetes: Helm chart with persistent volumes
    - cloud: Serverless for analysis, RDS for storage
    
  revenue_model:
    - type: subscription
    - pricing_tiers:
        - free: 1000 files/month
        - pro: unlimited files, learning mode
        - enterprise: custom rules, priority support
    - trial_period: 30 days
```

### Capability Discovery
```yaml
discovery:
  registry_entry:
    name: code-smell
    category: analysis
    capabilities: [smell-detection, auto-fix, pattern-learning]
    interfaces:
      - api: http://localhost:{PORT}/api/v1
      - cli: code-smell
      - events: code-smell.*
      
  metadata:
    description: Self-improving code quality guardian with AI-powered analysis
    keywords: [quality, lint, smell, technical-debt, auto-fix]
    dependencies: [resource-claude-code, postgres]
    enhances: [all scenarios]
```

## 🚨 Risk Mitigation

### Technical Risks
| Risk | Probability | Impact | Mitigation |
|------|------------|--------|------------|
| False positives in detection | Medium | Medium | Learning system + manual review queue |
| Breaking code with auto-fix | Low | High | Risk levels + comprehensive testing |
| Claude-code unavailability | Low | Medium | Fallback to regex-only rules |
| Rule conflicts | Medium | Low | Priority system + conflict detection |

### Operational Risks
- **Drift Prevention**: Rules versioned in git, hot-reload for updates
- **Performance Impact**: Incremental analysis with visited-tracker
- **Resource Conflicts**: Dedicated postgres schema

## ✅ Validation Criteria

### Phased Testing (Modern Architecture)
- **Structure** – Validates required files and directories (`test/phases/test-structure.sh`).
- **Dependencies** – Confirms Go module graph and npm toolchain health without mutating state (`test/phases/test-dependencies.sh`).
- **Unit** – Executes Go unit tests with coverage thresholds via the shared runner (`test/phases/test-unit.sh`).
- **Integration** – Verifies API/UI health endpoints and CLI status while the scenario is running (`test/phases/test-integration.sh`).
- **Business** – Exercises rule hot-reload, CLI analysis, learning endpoints, and statistics reporting (`test/phases/test-business.sh`).
- **Performance** – Placeholder phase tracks pending benchmark work (`test/phases/test-performance.sh`).

```bash
# Run the full phased suite
cd scenarios/code-smell
test/run-tests.sh --preset comprehensive

# Or via lifecycle configuration
vrooli scenario test code-smell
```

## 📝 Implementation Notes

### Design Decisions
**Hot-reload mechanism**: File watcher with inotify for instant rule updates
- Alternative considered: Polling every N seconds
- Decision driver: Real-time updates critical for development flow
- Trade-offs: More complex but better UX

**No n8n workflows**: Direct CLI/API integration per user requirements
- Alternative considered: n8n for orchestration
- Decision driver: Moving away from n8n platform-wide
- Trade-offs: More code but simpler deployment

### Known Limitations
- **Initial analysis slow**: First run analyzes entire codebase
  - Workaround: Use visited-tracker from start
  - Future fix: Incremental analysis from git history

### Security Considerations
- **Data Protection**: Fix history anonymized in learning mode
- **Access Control**: API keys for external scenario access
- **Audit Trail**: Every fix logged with timestamp and actor

## 🔗 References

### Documentation
- README.md - Quick start guide
- docs/rules.md - Rule writing guide
- docs/api.md - API specification
- docs/integration.md - Integration patterns

### Related PRDs
- visited-tracker - File change tracking
- git-manager - Git integration
- technical-debt-manager - Debt prioritization

---

**Last Updated**: 2025-09-07  
**Status**: Draft  
**Owner**: AI Agent  
**Review Cycle**: Weekly validation against implementation