# Product Requirements Document (PRD)

## 🎯 Capability Definition

### Core Capability
**What permanent capability does this scenario add to Vrooli?**
Vrooli Bridge enables external projects on the same machine to become Vrooli-aware consumers, allowing them to leverage Vrooli's full suite of intelligent scenarios and capabilities. It manages documentation injection, tracks integration health, and ensures external projects can effectively use Vrooli's software engineering prowess.

### Intelligence Amplification
**How does this capability make future agents smarter?**
This capability creates a bidirectional learning loop where:
- Agents learn from diverse codebases and project structures outside Vrooli
- External project patterns inform better scenario design
- Cross-project intelligence accumulates to identify common development needs
- Vrooli's capabilities become universally applicable, not just internal

### Recursive Value
**What new scenarios become possible after this exists?**
1. **cross-project-refactor** - Safely refactor code patterns across multiple Vrooli-enabled projects
2. **dependency-orchestrator** - Manage dependencies and breaking changes across integrated projects
3. **project-health-monitor** - Track and improve health metrics across all Vrooli-enabled codebases
4. **pattern-harvester** - Extract successful patterns from external projects to improve Vrooli scenarios
5. **multi-project-test-runner** - Coordinate testing across interdependent Vrooli-enabled projects

## 📊 Success Metrics

### Functional Requirements
- **Must Have (P0)**
  - [ ] Scan filesystem to discover projects (git repos, package.json, Cargo.toml, etc.)
  - [ ] Generate Vrooli integration documentation for external projects
  - [ ] Update/create CLAUDE.md files to reference Vrooli documentation
  - [ ] Track integration status and last update timestamps
  - [ ] Provide CLI interface for manual project registration
  - [ ] Store project registry in PostgreSQL for persistence
  
- **Should Have (P1)**
  - [ ] Auto-detect project type and recommend relevant scenarios
  - [ ] Version tracking for Vrooli documentation updates
  - [ ] Bulk operations for updating multiple projects
  - [ ] Integration health checks (verify docs are still present)
  - [ ] Project categorization and tagging system
  
- **Nice to Have (P2)**
  - [ ] Auto-update projects when new Vrooli scenarios are added
  - [ ] Project dependency graph visualization
  - [ ] Custom documentation templates per project type
  - [ ] Integration analytics and usage tracking

### Performance Criteria
| Metric | Target | Measurement Method |
|--------|--------|-------------------|
| Scan Time | < 5s for 100 projects | CLI timing |
| Documentation Generation | < 500ms per project | API monitoring |
| UI Response Time | < 200ms for list operations | Frontend monitoring |
| Database Queries | < 50ms for project lookups | PostgreSQL monitoring |

### Quality Gates
- [ ] All P0 requirements implemented and tested
- [ ] Can successfully integrate 10 different project types
- [ ] Documentation generation is idempotent
- [ ] UI displays accurate integration status
- [ ] CLI commands have comprehensive help text

## 🏗️ Technical Architecture

### Resource Dependencies
```yaml
required:
  - resource_name: postgres
    purpose: Store project registry and integration metadata
    integration_pattern: Direct SQL via Go API
    access_method: resource-postgres CLI for setup, direct connection for runtime
    
optional:
  - resource_name: redis
    purpose: Cache project scanning results
    fallback: Re-scan filesystem on each request
    access_method: resource-redis CLI
```

### Data Models
```yaml
primary_entities:
  - name: Project
    storage: postgres
    schema: |
      {
        id: UUID
        path: string (absolute path)
        name: string
        type: string (npm|cargo|python|go|generic)
        vrooli_version: string
        last_updated: timestamp
        integration_status: enum (active|outdated|missing)
        metadata: jsonb
      }
    relationships: One-to-many with IntegrationHistory
    
  - name: IntegrationHistory
    storage: postgres  
    schema: |
      {
        id: UUID
        project_id: UUID (FK)
        action: enum (created|updated|removed)
        vrooli_version: string
        timestamp: timestamp
        files_modified: jsonb
      }
```

### API Contract
```yaml
endpoints:
  - method: GET
    path: /api/v1/projects
    purpose: List all Vrooli-enabled projects
    output_schema: |
      {
        projects: [{
          id: string
          name: string
          path: string
          type: string
          status: string
          last_updated: string
        }]
      }
    
  - method: POST
    path: /api/v1/projects/scan
    purpose: Scan filesystem for new projects
    input_schema: |
      {
        directories: [string] (optional, defaults to home)
        depth: number (optional, max depth to scan)
      }
    output_schema: |
      {
        found: number
        new: number
        projects: [Project]
      }
      
  - method: POST
    path: /api/v1/projects/{id}/integrate
    purpose: Add/update Vrooli integration for a project
    input_schema: |
      {
        force: boolean (optional, overwrite existing)
      }
    output_schema: |
      {
        success: boolean
        files_created: [string]
        files_updated: [string]
      }
```

## 🖥️ CLI Interface Contract

### Command Structure
```yaml
cli_binary: vrooli-bridge
install_script: cli/install.sh

required_commands:
  - name: status
    description: Show integration statistics
    flags: [--json, --verbose]
    
  - name: help
    description: Display command help
    flags: [--all, --command <name>]
    
  - name: version
    description: Show version information
    flags: [--json]

custom_commands:
  - name: scan
    description: Scan for projects to integrate
    api_endpoint: /api/v1/projects/scan
    arguments:
      - name: path
        type: string
        required: false
        description: Directory to scan (defaults to ~)
    flags:
      - name: --depth
        description: Max depth for scanning
      - name: --type
        description: Filter by project type
    
  - name: integrate
    description: Add Vrooli to a project
    api_endpoint: /api/v1/projects/{id}/integrate
    arguments:
      - name: project-path
        type: string
        required: true
        description: Path to project
    flags:
      - name: --force
        description: Overwrite existing integration
        
  - name: list
    description: List integrated projects
    api_endpoint: /api/v1/projects
    flags:
      - name: --status
        description: Filter by status (active|outdated|missing)
      - name: --type
        description: Filter by project type
        
  - name: update
    description: Update Vrooli docs in projects
    arguments:
      - name: project-path
        type: string
        required: false
        description: Specific project or all if omitted
```

## 🔄 Integration Requirements

### Upstream Dependencies
- **PostgreSQL**: Database for project registry
- **Filesystem Access**: Read/write to project directories
- **Git**: Detect repository boundaries

### Downstream Enablement
- **cross-project-refactor**: Uses project registry to coordinate changes
- **dependency-orchestrator**: Leverages project type detection
- **project-health-monitor**: Builds on integration tracking

### Cross-Scenario Interactions
```yaml
provides_to:
  - scenario: test-genie
    capability: List of testable projects
    interface: API
    
  - scenario: code-sleuth
    capability: Cross-project code search scope
    interface: CLI
    
consumes_from:
  - scenario: prompt-manager
    capability: Documentation templates
    fallback: Use built-in templates
```

## 🎨 Style and Branding Requirements

### UI/UX Style Guidelines
```yaml
style_profile:
  category: professional
  inspiration: GitHub Desktop meets VS Code Extensions
  
  visual_style:
    color_scheme: dark with accent colors per project type
    typography: modern, monospace for paths
    layout: dashboard with sidebar navigation
    animations: subtle state transitions
  
  personality:
    tone: helpful and informative
    mood: focused and efficient
    target_feeling: "I have full control over my integrations"
```

### Target Audience Alignment
- **Primary Users**: Developers using Claude Code on multiple projects
- **User Expectations**: Clean, fast, developer-focused interface
- **Accessibility**: WCAG AA compliance
- **Responsive Design**: Desktop-first, tablet-compatible

## 💰 Value Proposition

### Business Value
- **Primary Value**: Multiplies Vrooli's reach to entire development ecosystem
- **Revenue Potential**: $25K - $100K per enterprise deployment
- **Cost Savings**: 20-40% reduction in cross-project development time
- **Market Differentiator**: First AI system that enhances ALL projects on a machine

### Technical Value
- **Reusability Score**: 10/10 - Every scenario benefits from external project access
- **Complexity Reduction**: Makes multi-project coordination trivial
- **Innovation Enablement**: Allows Vrooli to learn from entire codebases

## 🧬 Evolution Path

### Version 1.0 (Current)
- Basic project discovery and integration
- Manual documentation injection
- Simple tracking dashboard

### Version 2.0 (Planned)
- Automatic integration updates
- Project dependency mapping
- Cross-project scenario recommendations
- Integration health monitoring

### Long-term Vision
- Vrooli becomes the universal development intelligence layer
- Every project on a machine contributes to collective intelligence
- Cross-organizational pattern sharing (with privacy controls)

## ✅ Validation Criteria

### Declarative Test Specification
```yaml
version: 1.0
scenario: vrooli-bridge

structure:
  required_files:
    - .vrooli/service.json
    - PRD.md
    - api/main.go
    - api/go.mod
    - cli/vrooli-bridge
    - cli/install.sh
    - initialization/storage/postgres/schema.sql
    - ui/index.html
    
tests:
  - name: "Project scanner finds test projects"
    type: exec
    command: ./cli/vrooli-bridge scan --depth 2 /tmp/test-projects
    expect:
      exit_code: 0
      output_contains: ["found", "projects"]
      
  - name: "Integration adds Vrooli docs"
    type: exec
    command: ./cli/vrooli-bridge integrate /tmp/test-project
    expect:
      exit_code: 0
      files_created:
        - /tmp/test-project/VROOLI_INTEGRATION.md
        
  - name: "UI dashboard loads"
    type: http
    service: ui
    endpoint: /
    method: GET
    expect:
      status: 200
      content_contains: ["Vrooli Bridge", "Projects"]
```

## 📝 Implementation Notes

### Design Decisions
**Documentation Injection**: Chose file-based approach over API
- Alternative considered: REST API for doc serving
- Decision driver: Works offline, git-trackable
- Trade-offs: More disk usage for better reliability

**Project Detection**: Filesystem scanning over registry service
- Alternative considered: System-wide project registry
- Decision driver: No daemon required, simpler deployment
- Trade-offs: Slower discovery for easier maintenance

### Known Limitations
- **Filesystem Permissions**: Requires write access to project directories
  - Workaround: User must have appropriate permissions
  - Future fix: Optional read-only mode with instructions
  
- **Large Repositories**: Scanning can be slow for huge codebases
  - Workaround: Use --depth flag to limit scan
  - Future fix: Incremental scanning with cache

### Security Considerations
- **Data Protection**: No project code is stored, only metadata
- **Access Control**: Respects filesystem permissions
- **Audit Trail**: All integrations logged with timestamps

---

**Last Updated**: 2025-01-06  
**Status**: Draft  
**Owner**: AI Agent  
**Review Cycle**: After each Vrooli version release