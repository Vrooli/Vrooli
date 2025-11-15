# Product Requirements Document (PRD)

> **Status**: P0 Complete ✅ | **Version**: 1.0.0 | **Last Updated**: 2025-11-14
>
> See [CHANGELOG.md](./CHANGELOG.md) for detailed implementation history.

## 🎯 Capability Definition

### Core Capability
**What permanent capability does this scenario add to Vrooli?**
The PRD Control Tower adds the permanent capability to **comprehensively manage, validate, and publish Product Requirements Documents** across all scenarios and resources. It provides a centralized command center for browsing published PRDs, creating and editing drafts, enforcing structure rules, integrating AI assistance, and seamlessly publishing to the repository. This creates a self-improving documentation system where every PRD maintains consistent quality and becomes a permanent knowledge artifact.

### Intelligence Amplification
**How does this capability make future agents smarter?**
- **Documentation Repository**: Builds permanent knowledge base of scenario and resource capabilities with searchable, structured PRDs
- **Automated Compliance**: New PRDs inherit established structure standards automatically via templates and validation
- **AI-Assisted Authoring**: Natural language prompts generate PRD sections, reducing documentation burden while maintaining quality
- **Quality Feedback Loop**: Structure violations discovered in one PRD prevent similar issues across all others via real-time validation
- **Cross-Reference Intelligence**: Links between PRDs create knowledge graph of scenario dependencies and capabilities

### Recursive Value
**What new scenarios become possible after this exists?**
- **Automated PRD Generation**: New scenarios automatically get well-structured PRDs from ecosystem manager task definitions
- **Knowledge Observatory**: Semantic search across all PRDs to discover capabilities and avoid duplication
- **Compliance Dashboard**: Real-time view of PRD quality across the entire ecosystem
- **Documentation Evolution**: Track PRD changes over time to understand capability maturation
- **AI Documentation Agent**: Autonomous agent that maintains and improves PRDs based on code changes

## 📊 Success Metrics

### Functional Requirements
- **Must Have (P0)**
  - [x] Catalog view listing all scenarios/resources with PRD status (Has PRD, Draft pending, Violations, Missing)
  - [x] Published PRD viewer (read-only, properly formatted markdown)
  - [x] Draft lifecycle (create, edit, save, delete, publish)
  - [x] Draft index page listing all open drafts with metadata (owner, updated timestamp, status)
  - [x] Integration with scenario-auditor for PRD structure validation
  - [x] AI-assisted section generation via resource-openrouter
  - [x] Publishing action that replaces PRD.md and clears draft
  - [x] Health check endpoint for API and UI
  - [x] Basic CLI commands (status, list-drafts, validate)

- **Should Have (P1)**
  - [ ] Diff viewer comparing draft vs published PRD
  - [ ] Violation detail view with actionable fix suggestions
  - [ ] New scenario creation workflow linking to ecosystem-manager
  - [ ] Markdown editor with syntax highlighting and preview
  - [ ] Filter and search in catalog (by name, type, compliance status)
  - [ ] Draft autosave (local + explicit save)

- **Nice to Have (P2)**
  - [ ] Multi-user draft locking (prevent concurrent edits)
  - [ ] PRD version history and rollback
  - [ ] Export PRDs to PDF/HTML
  - [ ] Bulk operations (validate all, fix common violations)
  - [ ] Custom PRD templates per organization

### Performance Criteria
| Metric | Target | Measurement Method |
|--------|--------|-------------------|
| Catalog Load Time | < 2s for 200+ scenarios | Frontend performance monitoring |
| Draft Save Time | < 500ms | API response time tracking |
| Validation Time | < 5s per PRD | scenario-auditor integration timing |
| AI Section Generation | < 10s per section | resource-openrouter response time |
| Publish Operation | < 3s including validation | End-to-end publish flow timing |

### Quality Gates
- [x] All P0 requirements implemented and tested (9 of 9 complete)
- [x] Catalog correctly lists all scenarios and resources
- [x] Draft CRUD operations work reliably
- [x] scenario-auditor integration returns accurate violations
- [x] AI assistance generates valid markdown sections
- [x] Publishing updates PRD.md and clears draft atomically
- [x] Health checks pass for API and UI
- [x] Draft index page displays all drafts with metadata
- [x] CLI commands (status, list-drafts, validate) working

## 🏗️ Technical Architecture

### Core Components
1. **PRD Catalog Service**: Enumerates scenarios/resources by globbing `scenarios/*/PRD.md` and `resources/*/PRD.md`
2. **Draft Store**: CRUD operations over centralized draft files (`data/prd-drafts/{scenario|resource}/<name>.md`) with JSON metadata
3. **Audit Service**: Wraps scenario-auditor API calls, handles timeouts, maps violations to UI hints
4. **AI Service**: Bridges to resource-openrouter CLI/HTTP, handles prompt templating for section generation
5. **Ecosystem Manager Client**: POSTs task creation requests with draft path references for new scenarios
6. **Publishing Engine**: Safely writes PRD.md with optional git diff preview, re-runs validation

### Resource Dependencies
- **PostgreSQL**: Draft metadata (owner, timestamps, entity references), audit results cache
- **scenario-auditor**: PRD structure validation via HTTP API
- **resource-openrouter**: AI assistance for section generation and rewriting
- **ecosystem-manager**: Task creation for new scenarios (optional integration)

### API Endpoints
- `GET /api/v1/catalog` - List all scenarios/resources with PRD status
- `GET /api/v1/catalog/{type}/{name}` - Get published PRD content
- `GET /api/v1/drafts` - List all drafts with metadata
- `GET /api/v1/drafts/{id}` - Get draft content and metadata
- `POST /api/v1/drafts` - Create new draft
- `PUT /api/v1/drafts/{id}` - Update draft content
- `DELETE /api/v1/drafts/{id}` - Delete draft
- `POST /api/v1/drafts/{id}/validate` - Run scenario-auditor validation
- `POST /api/v1/drafts/{id}/ai/generate-section` - AI-generate PRD section
- `POST /api/v1/drafts/{id}/ai/rewrite` - AI-rewrite section with compliance
- `POST /api/v1/drafts/{id}/publish` - Publish draft to PRD.md
- `POST /api/v1/ecosystem-manager/create-scenario` - Create new scenario with draft

### Integration Strategy
- **Draft Storage**: Filesystem-based with metadata JSON sidecar files (`draft.md` + `draft.json`)
- **Validation**: HTTP calls to scenario-auditor with caching to avoid repeated checks
- **AI Integration**: Shell out to resource-openrouter CLI or HTTP API with templated prompts
- **Publishing**: Atomic file write with backup, optional git status check before publish
- **Ecosystem Manager**: HTTP POST to create task, include draft path in notes field

### Health & Monitoring
- **API Health Check**: Database connectivity, draft directory writability, scenario-auditor reachability
- **UI Health Check**: API connectivity verification with 3-second timeout
- **Lifecycle Integration**: Both API and UI managed through service.json develop lifecycle

## 🖥️ CLI Interface Contract

### Command Structure
```bash
prd-control-tower <command> [options]

Commands:
  status              Show service health and statistics
  list-drafts         List all open drafts with metadata
  validate <name>     Validate PRD structure for scenario/resource
  create-draft        Create new draft (interactive wizard)
  publish <draft-id>  Publish draft to PRD.md
  help                Show command help
```

### CLI Capabilities
- Status check showing API health, draft count, validation stats
- List drafts in table format with entity name, type, owner, last update
- Validate PRD and display violations with severity
- Interactive draft creation wizard (choose entity, template, AI prefill)
- Publish command with confirmation and diff preview
- Help text with examples

## 🔄 Integration Requirements

### scenario-auditor Integration
- **Validation**: Call scenario-auditor HTTP API to check PRD structure compliance
- **Caching**: Store validation results in PostgreSQL with timestamp, re-validate on draft changes
- **Error Handling**: Graceful timeout handling (default 240s), fallback to "validation unavailable"
- **Mapping**: Convert scenario-auditor violations to UI-friendly format (line numbers, recommendations)

### resource-openrouter Integration
- **Section Generation**: Prompt template includes: entity type, existing PRD content, missing sections
- **Rewriting**: Prompt template includes: section text, violations found, template structure
- **Context Provision**: Include published PRD, draft content, audit violations in AI context
- **Response Handling**: Extract generated markdown, validate structure, stage with diff approval

### ecosystem-manager Integration
- **Task Creation**: POST to ecosystem-manager API with scenario name, summary, draft path
- **Draft Linking**: Include draft path in task notes so handler can access PRD during generation
- **New Scenario Flow**: UI "Send to ecosystem-manager" action on drafts flagged as "New Scenario"
- **Status Tracking**: Optional: poll ecosystem-manager for task completion status

### Git Integration (Optional)
- **Diff Preview**: Show git diff before publishing if repository is clean
- **Commit Hook**: Optionally create commit after publish (disabled by default)
- **Safety Check**: Warn if uncommitted changes exist in target PRD.md

## 🎨 Style and Branding Requirements

### UI Design
- **Layout**: Clean dashboard with sidebar navigation (Catalog, Drafts, Settings)
- **Component Library**: shadcn/UI for consistent, accessible components
- **Icons**: Lucide React icons for status indicators (✅ Has PRD, ⚠️ Violations, ❌ Missing)
- **Color Coding**: Green=compliant, Yellow=violations, Red=missing, Blue=draft pending
- **Responsive**: Mobile-friendly with collapsible sidebar

### Markdown Editor
- **Editor**: TipTap or MDX-based with toolbar for formatting
- **Live Preview**: Split-pane with markdown source on left, rendered on right
- **Syntax Highlighting**: Code blocks with language detection
- **Diff Viewer**: Monaco diff editor or react-diff-viewer for publish preview

### Status Chips
- **Has PRD**: Green badge with checkmark
- **Draft Pending**: Blue badge with pencil icon
- **Violations**: Yellow badge with warning icon and count
- **Missing**: Red badge with X icon

## 💰 Value Proposition

### Direct Revenue Impact
- **Time Savings**: Reduces PRD creation time from 2 hours to 30 minutes (75% reduction)
- **Quality Improvement**: Ensures 100% PRD structure compliance (currently ~60%)
- **AI Efficiency**: 10x faster documentation with AI-assisted generation
- **Onboarding Acceleration**: New scenarios get compliant PRDs in minutes, not days

### Ecosystem Value
- **Documentation Quality**: Permanent improvement in PRD consistency and completeness
- **Knowledge Preservation**: Every capability documented to standard, searchable, discoverable
- **Reduced Duplication**: Easy PRD browsing prevents building duplicate scenarios
- **Improved Handoffs**: Clear PRD structure enables smooth agent transitions

### Business Justification
- **Current Pain**: PRDs created manually, inconsistent structure, missing sections, no validation
- **Opportunity Cost**: Poor PRDs lead to wasted development time, duplicated work, knowledge loss
- **Market Differentiation**: No other agent ecosystem provides centralized PRD management
- **Scalability**: Enables managing 1000+ scenarios without documentation chaos

**Conservative Estimate**: Saves 1.5 hours per PRD × 100 PRDs/year = 150 hours = $15K-20K value annually

## 🧬 Evolution Path

### Phase 1: Foundation (Current)
- Catalog view and published PRD viewer
- Basic draft CRUD operations
- scenario-auditor integration
- Simple AI section generation

### Phase 2: Intelligence
- Advanced diff viewer with conflict resolution
- Smart AI prompts based on context analysis
- Violation auto-fix suggestions
- Draft templates with placeholders

### Phase 3: Automation
- Automatic PRD updates from code changes
- CI/CD integration for validation
- Bulk operations across all PRDs
- Quality scoring and gamification

### Phase 4: Integration
- Full ecosystem-manager workflow
- Git commit automation
- Multi-user collaboration with locking
- Version history and rollback

## 🔄 Scenario Lifecycle Integration

### Lifecycle Steps
1. **setup**: Create draft directory, initialize PostgreSQL tables, verify dependencies
2. **develop**: Start API (Go service), start UI (React dev server with Vite)
3. **test**: Run structure tests, integration tests, API tests, UI build
4. **stop**: Stop API and UI processes, clean up PIDs

### Port Allocation
- **API**: 18600 (HTTP server)
- **UI**: 36300 (Vite dev server)

### Health Checks
- **API**: `GET /api/v1/health` → `{"status": "healthy", "readiness": true}`
- **UI**: `GET /health` → `{"status": "healthy", "api_connectivity": {"connected": true}}`

### Resource Requirements
- **PostgreSQL**: Draft metadata tables (drafts, audit_results)
- **Disk Space**: Draft storage in `data/prd-drafts/` directory
- **Network**: HTTP access to scenario-auditor, resource-openrouter

## 🚨 Risk Mitigation

### Technical Risks
1. **Draft Corruption**: Mitigate with atomic writes, backup before publish
2. **Concurrent Edits**: Detect with file modification timestamps, warn user
3. **scenario-auditor Timeout**: Cache previous results, show stale data with warning
4. **AI Service Failure**: Graceful degradation, manual editing always available
5. **Git Conflicts**: Check for uncommitted changes before publish, abort if dirty

### Operational Risks
1. **Data Loss**: Regular backups of draft directory, PostgreSQL dumps
2. **Performance**: Index draft metadata tables, cache validation results
3. **Security**: Validate all file paths, sanitize draft content before publish
4. **Compliance**: Audit trail of all publishes with timestamp and user

## ✅ Validation Criteria

### Capability Validated When:
- [x] Catalog correctly lists all scenarios and resources with accurate status
- [x] Published PRD viewer renders markdown with proper formatting
- [x] Draft creation from template or existing PRD works
- [x] Draft index page lists all drafts with metadata
- [x] scenario-auditor integration returns violations with line numbers
- [x] AI section generation produces valid markdown
- [x] Publishing atomically updates PRD.md and clears draft
- [x] Health checks pass and services start via lifecycle system
- [x] CLI commands work correctly (status, list-drafts, validate)

**This scenario becomes Vrooli's permanent PRD command center - ensuring every scenario has compliant, AI-assisted, high-quality documentation.**

## 📝 Implementation Notes

### Technical Decisions
- **Storage**: Filesystem for drafts (simple, version-controllable), PostgreSQL for metadata (queryable)
- **Editor**: TipTap chosen for extensibility and markdown support
- **AI**: Shell out to resource-openrouter CLI for simplicity (can upgrade to HTTP API later)
- **Validation**: Periodic polling (not real-time) to avoid overwhelming scenario-auditor
- **Publishing**: File write with backup, no automatic git commit to preserve user control

### Development Priorities
1. **P0 Scaffold**: Catalog, draft CRUD, basic publishing (no AI, no validation)
2. **P0 Integration**: scenario-auditor validation, AI section generation
3. **P1 Polish**: Diff viewer, search/filter, autosave
4. **P2 Advanced**: Multi-user, version history, bulk operations

### Known Limitations
- Single-user only initially (no draft locking)
- No real-time collaboration features
- AI assistance requires resource-openrouter running
- Validation requires scenario-auditor running
- No automatic git operations (manual commit after publish)

### Testing Strategy
- **Unit Tests**: Draft CRUD, catalog enumeration, validation caching
- **Integration Tests**: scenario-auditor API calls, resource-openrouter integration
- **E2E Tests**: Full publish workflow, AI-assisted draft creation
- **Manual Tests**: UI usability, editor performance, diff viewer accuracy

## 🔗 References

### Internal Documentation
- `/docs/scenarios/README.md` - Scenario development standards
- `/scripts/scenarios/templates/react-vite/PRD.md` - PRD template structure
- `/scenarios/scenario-auditor/PRD.md` - Auditor integration details
- `/docs/context.md` - Vrooli ecosystem overview

### External References
- **TipTap**: https://tiptap.dev/ - Markdown editor framework
- **shadcn/UI**: https://ui.shadcn.com/ - React component library
- **Monaco Editor**: https://microsoft.github.io/monaco-editor/ - Diff viewer
- **PRD Best Practices**: https://www.productplan.com/glossary/product-requirements-document/
- **Markdown Specification**: https://spec.commonmark.org/

### Existing Patterns
- `scenarios/scenario-auditor` - Rule validation and enforcement patterns
- `scenarios/document-manager` - Document lifecycle management
- `scenarios/ecosystem-manager` - Task creation and agent spawning
- `scenarios/funnel-builder` - React/TypeScript/Vite setup
- `scenarios/invoice-generator` - Draft workflow patterns

**Session 8 (2025-10-14)**: Code quality and maintainability improvements
- Removed 12 duplicate Content-Type header declarations across all API files
- Replaced custom string manipulation functions with Go standard library
- Improved code maintainability by eliminating unnecessary custom implementations
- All tests pass with no regressions
- Functional verification: All API endpoints working correctly

**Session 9 (2025-10-14)**: Code deduplication and maintainability enhancements
- Created `getVrooliRoot()` helper function to eliminate duplicate VROOLI_ROOT/HOME resolution logic
- Refactored 3 locations (catalog.go, publish.go) to use centralized helper
- Reduced code duplication and improved consistency
- Standards violations reduced from 625 to 620 (5 violations eliminated)
- All tests pass with no regressions
- Security scan: PASSED (0 vulnerabilities)
