# Research: tidiness-manager

**Date**: 2025-11-21
**Author**: Generator Agent
**Purpose**: Document uniqueness analysis and competitive landscape for the tidiness-manager scenario

## Uniqueness Check

### Within Vrooli Repository

Searched for existing scenarios that might overlap with tidiness-manager's core capabilities:

1. **visited-tracker** (`/scenarios/visited-tracker/`)
   - **Purpose**: Persistent file visit tracking with staleness detection for systematic code analysis
   - **Overlap**: File tracking, visit counting, staleness scoring
   - **Differentiation**: tidiness-manager USES visited-tracker as a dependency; doesn't duplicate it. visited-tracker provides the "which files to analyze next" service, while tidiness-manager focuses on "what issues exist in files"

2. **scenario-auditor** (`/scenarios/scenario-auditor/`)
   - **Purpose**: Comprehensive standards enforcement (security, configuration, UI testing, development standards)
   - **Overlap**: Quality checking, rules engine, violation reporting
   - **Differentiation**: scenario-auditor focuses on **standards compliance** (security, schema validation, best practices enforcement). tidiness-manager focuses on **code cleanliness** (file length, dead code, duplication, refactoring opportunities). Complementary tools - auditor says "does this meet our standards?" while tidiness says "is this code getting messy?"

3. **code-smell** (`/scenarios/code-smell/`)
   - **Purpose**: Self-improving code quality guardian detecting and fixing code smell violations
   - **Overlap**: Code quality, pattern detection, auto-fixing, AI-powered analysis
   - **Differentiation**: SIGNIFICANT OVERLAP. code-smell focuses on **pattern-based violations** (bad practices, anti-patterns, smell patterns). tidiness-manager focuses on **structural issues** (length, organization, dead code, duplication).
   - **Integration Opportunity**: These should likely work together - code-smell detects WHAT is wrong (patterns), tidiness-manager detects WHERE things are getting messy (structure). tidiness-manager could call code-smell for deep pattern analysis.

4. **prd-control-tower** (`/scenarios/prd-control-tower/`)
   - **Purpose**: PRD and requirements management with quality metrics
   - **Overlap**: Quality scoring, validation
   - **Differentiation**: Focused on documentation and requirements, not code. No overlap.

5. **app-monitor** (`/scenarios/app-monitor/`)
   - **Purpose**: Application health monitoring and diagnostics
   - **Overlap**: Monitoring, dashboard
   - **Differentiation**: Runtime monitoring vs. static code analysis. No overlap.

### Unique Value Proposition

tidiness-manager is unique because it:

1. **Orchestrates multiple quality tools** - Combines cheap static analysis (lint, type checking, line counts) with expensive AI analysis (code-smell, resource-claude-code) in a coordinated way
2. **Progressive coverage** - Uses visited-tracker to ensure comprehensive, non-redundant analysis across large codebases
3. **Campaign management** - Provides automatic, long-running tidiness campaigns that systematically improve code health
4. **Makefile integration** - Leverages existing scenario Makefiles (`make lint`, `make type`) as the foundation for light scanning
5. **Agent-callable interface** - Designed to be invoked by other agents/scenarios for "give me refactor suggestions"
6. **Multi-tier scanning** - Light (cheap, always-on) + Smart (expensive, targeted) scanning model

### Related Scenarios/Resources

**Dependencies:**
- `visited-tracker` - File visit tracking and prioritization
- `resource-claude-code` - AI-powered code analysis
- `resource-codes` - Additional AI analysis capabilities
- `postgres` (optional) - Data storage
- `redis` (optional) - Caching

**Complementary:**
- `scenario-auditor` - Standards compliance (different focus)
- `code-smell` - Pattern-based smell detection (integration partner)
- `app-issue-tracker` - Could receive tidiness issues as tasks

**Consumers:**
- Any maintenance/improvement scenarios
- Development agents needing refactor guidance
- CI/CD pipelines wanting code health metrics

## External Research

### Existing Code Quality Tools

1. **SonarQube**
   - Comprehensive quality platform
   - Our differentiation: Lightweight, CLI-first, agent-optimized, integrated with Vrooli ecosystem

2. **CodeClimate**
   - Automated code review platform
   - Our differentiation: Local-first, campaign-based coverage, visited-tracker integration

3. **Sourcery** (AI refactoring)
   - AI-powered refactoring suggestions
   - Our differentiation: Multi-tool orchestration, systematic coverage, Makefile integration

4. **Resharper / ReSharper**
   - IDE-based refactoring
   - Our differentiation: Cross-scenario, campaign management, agent API

### Key Features to Incorporate

1. **Configurable thresholds** - Allow customization per scenario (file length limits, complexity thresholds)
2. **Issue prioritization** - Severity scoring for tidiness issues
3. **Historical tracking** - See code health trends over time
4. **Batch operations** - Handle large refactoring campaigns
5. **Integration-first** - Make it easy for other agents to consume

## Technical Considerations

### Performance

- Light scans should complete in <60-120 seconds for typical scenarios
- Smart scans will be slower (AI-dependent); need queue management
- Cache lint/type results to avoid redundant Makefile runs
- Respect visited-tracker to avoid re-analyzing unchanged files

### Scalability

- Support scenarios with 1000+ files
- Batch AI analysis to control costs
- Configurable concurrency limits
- Campaign throttling (max K scenarios at once)

### Data Storage

- PostgreSQL for issue tracking, campaign state
- File-based JSON for light scan results (portability)
- Redis for caching expensive operations

### Integration Patterns

- CLI for programmatic access
- HTTP API for web UI and external integrations
- Agent-friendly response format (ranked, categorized issues)
- Webhook/event support for CI/CD integration

## Implementation Risks

1. **Overlap with code-smell** - Need clear boundaries or integration strategy
2. **AI cost control** - Smart scans could get expensive; need strict batching
3. **False positives** - Long files aren't always bad; need context-aware flagging
4. **Campaign runaway** - Auto-campaigns must have kill switches
5. **Makefile dependency** - Not all scenarios have standardized Makefiles yet

## Recommendations

1. **Coordinate with code-smell** - Define clear integration points or merge capabilities
2. **Start with light scanning** - Prove value with cheap analysis before heavy AI
3. **Build API-first** - CLI and UI consume the same API
4. **Configurable everything** - Thresholds, rules, campaigns should be tunable
5. **Monitor resource usage** - Track AI tokens, scan times, storage growth
6. **Progressive rollout** - Start with a few scenarios, expand based on learnings

## References

- visited-tracker PRD: `/scenarios/visited-tracker/PRD.md`
- scenario-auditor README: `/scenarios/scenario-auditor/README.md`
- code-smell PRD: `/scenarios/code-smell/PRD.md`
- Vrooli context: `/docs/context.md`
- Template documentation: `vrooli scenario template show react-vite`
