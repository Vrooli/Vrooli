# Development Progress

Agent-maintained document tracking development milestones.

## Last Updated
2026-02-15

---

## Completed Milestones

### Phase 1: Core Implementation
- [x] API server with skill CRUD operations
- [x] File-based skill storage with metadata.json
- [x] SQLite integration for tags, metrics, test results
- [x] React UI with folder-based navigation
- [x] Basic CLI with skill operations

### Phase 2: Screaming Architecture Alignment (2025-01-25)
- [x] API refactored to domain-driven structure
- [x] Interfaces added to all domains:
  - `skills/interfaces.go` - SkillStore, MetricsService
  - `tags/interfaces.go` - TagRepository
  - `agents/interfaces.go` - AgentStore
  - `teams/interfaces.go` - TeamStore
  - `store/interfaces.go` - RelationStore, IndexStore
  - `testing/interfaces.go` - TestRepository, LLMClient
  - `metrics/interfaces.go` - MetricsRepository
- [x] Handlers refactored to depend on interfaces
- [x] Search domain added with full-text search
- [x] Version history endpoints added to skills domain

### Phase 3: CLI Completion (2025-01-25)
- [x] Noun-verb subcommand pattern implemented
- [x] All API endpoints have CLI coverage:
  - `skill` - list, show, read, add, update, delete, use, sync, rate, versions, revert
  - `tag` - list, create
  - `agent` - list, show, create, update, delete
  - `team` - list, show, create, add-member (commands implemented)
  - `test` - run, history
  - `search` - query
  - `metadata` - fetch
- [x] Output formatting helpers in `internal/output/`
- [x] Clipboard integration for `use` and combined `read` output

### Phase 4: Documentation (2025-01-25)
- [x] docs/QUICKSTART.md - Getting started guide
- [x] docs/concepts/ARCHITECTURE.md - System overview
- [x] docs/reference/api-endpoints.md - Complete API docs
- [x] docs/reference/cli-commands.md - Complete CLI docs
- [x] docs/reference/configuration.md - Configuration guide
- [x] docs/internal/ directory setup with SEAMS, PROBLEMS, PROGRESS

---

### Phase 5: Skills + Agents + Teams Evolution (2025-01-30)
- [x] Agent domain with appearance, SOUL.md, capabilities
- [x] Team domain with roles, members, org charts
- [x] Agent-skill relations with pins and enablement
- [x] Team-member relations with role assignments
- [x] Effective skills computation API
- [x] 3D world visualization with React Three Fiber
- [x] Documentation update to reflect 3-domain architecture

### Phase 6: Heartbeat Lifecycle Hardening (2026-02-03)
- [x] Scheduler resolves per-member profile keys
- [x] Manual trigger and executor handle missing configs safely
- [x] Team/member deletion unschedules heartbeats and cleans member data
- [x] Added regression tests for scheduler, executor, and cleanup flows

### Phase 7: Relationship Graph (2026-02-12)
- [x] Graph data model (nodes: team/agent/skill/cli; edges: 6 kinds)
- [x] Scanner with 4 regex patterns for reference extraction from markdown
- [x] CLI detector for code-usage edge extraction
- [x] Builder pipeline (collect nodes → scan edges → extract CLI nodes → score)
- [x] Health scoring with weighted composite factors
- [x] Analytical queries (orphans, skillless, empty teams, unaffiliated, popular, cycles)
- [x] Index persistence with lazy generation and mutation-driven invalidation
- [x] REST API (12 endpoints under `/api/v1/graph`)
- [x] CLI commands (`graph show|dump|node|regenerate|orphaned-skills|...`)
- [x] React Flow UI with Dagre layout, custom node shapes, health tinting
- [x] Query panel, toolbar filters, legend, and tooltip components
- [x] Zustand store + service layer with 10s client cache
- [x] Comprehensive test coverage (builder, scanner, scoring, queries, handlers, index)

### Phase 8: Team Execution Model & Coordination Skills (2026-02-15)
- [x] Team-level execution context with bounded FIFO queue
- [x] Policy-aware per-team execution queue with `serialized` and `bounded-parallel` modes
- [x] Dedup protection: 409 Conflict when member already queued/running
- [x] Queue persistence to disk with crash recovery
- [x] Scheduler routes heartbeats through TeamExecutionStore
- [x] GET /teams/{id}/execution-status endpoint
- [x] Three coordination skills: `team-coordination-independent`, `team-coordination-peer`, and `team-coordination-leader-led`
- [x] PromptBuilder injects coordination skill reference based on team coordination pattern
- [x] Member context endpoint: GET /teams/{id}/members/{agentId}/context (excludes HEARTBEAT.md)
- [x] CLI command: `prompt-manager team member-context <team-id> <agent-id>`
- [x] Comprehensive test coverage for queue behavior, scheduler integration, prompt builder, and handlers
- [x] Documentation: TEAM-EXECUTION.md concept doc, heartbeat-api.md and heartbeat-cli.md reference updates

### Phase 9: Proto/Connect Re-platform and Runtime Adoption (2026-08-19)
- [x] Normalized the domain layout and removed empty scaffolding
- [x] Re-platformed actions, skills, teams, tags, topics, agents, metrics, testing, AI search, graph, heartbeat, memberflow, and experiments onto generated Connect services
- [x] Retired domain REST mirrors, leaving six non-domain registrations
- [x] Declared 120 Connect commands, 10 intentionally local commands, 75 typed omissions, and 18 architecture exceptions
- [x] Declared and probed nine measures over real stored data
- [x] Program Runtime doctor resolves 117 of 117 Prompt Manager bindings as callable
- [x] Program Runtime `guide` composes `prompt-manager/discover/discover` in a live kernel

**Validation limitation:** CLI Health earns PASSED/L3 and its proto/runtime
surface is clean, but the current provider classifies 138 architecture-primitive
findings as required maturity debt. Measures Health passes behavioral and domain
coverage while reporting four canonical-parameter hardening findings.

### Phase 10: Reliability Closeout Evidence (2026-08-19)
- [x] Durable workflow execution `668b8aac-55b5-476a-b6c3-1222ac3882f3` passed 6/6 at L5 with isolated fixtures and no primary-data leaks
- [x] Business Health passed at L3 with zero findings
- [x] Program Runtime enumerated 117 Prompt Manager bindings and reported zero uncallable bindings globally
- [x] Live `guide` discovery and all Program Runtime examples and verbs succeeded against the Connect binding
- [x] Immutable collection diff `phase19-final-comparison` classified Prompt Manager's remaining Test Genie failures as pre-existing, with no regression

**Closeout limitation:** The final Prompt Manager suite
`20260819-110355-3e69cd28` passed 12/21; the collection diff proves the remaining
phases pre-existing rather than clean. Nine scenarios added to later plan
validation scopes were absent from the immutable pre-change collection and are
therefore non-comparable.

---

## Current Focus

- Documentation completeness and accuracy
- Adding DOC: comments to critical code paths
- Adding [CODE: ...] references in documentation
- Added core steer skills `quality-health` and `tidiness` so standards and tidiness Test Genie findings can route to dedicated remediation doctrine with `programmaticHome` metadata.

---

## Upcoming Work

### Testing
- [ ] Add tests for tags, agents, teams, testing, search handlers
- [ ] Add CLI integration tests
- [ ] Add E2E tests with test containers

### Features
- [ ] Semantic search with Qdrant (optional)
- [ ] CLI shell completion scripts
- [ ] Batch operations (bulk import/export)

### Infrastructure
- [ ] Docker compose setup
- [ ] CI/CD pipeline
- [ ] Health check improvements

---

## Architecture Decisions

### ADR-001: File-based Skill Storage
**Decision:** Store skill content as .md files, metadata in metadata.json
**Rationale:**
- Human-readable and version control friendly
- Easy to edit outside the application
- Clear separation from database-stored data

### ADR-002: Interface-based Handlers
**Decision:** All handlers depend on interfaces, not concrete types
**Rationale:**
- Enables unit testing without external dependencies
- Clear contracts between layers
- Supports implementation swapping

### ADR-003: CLI as Contract-Aware API Client
**Decision:** CLI commands remain API-first, but own lightweight contract-aware flag resolution for team policy presets before sending requests
**Rationale:**
- API validation remains authoritative
- CLI can expose ergonomic presets and migrations without duplicating server validation rules
- Shared `teamconfig` helpers keep API and CLI policy defaults aligned

---

## Version History

| Version | Date | Changes |
|---------|------|---------|
| 2.1.0 | 2026-04-09 | Team runtime/coordination/execution contract migration, independent/peer/leader-led coordination skills, CLI/API policy alignment |
| 2.0.0 | 2025-01-25 | Screaming architecture alignment, full CLI coverage |
| 1.0.0 | 2024-12-01 | Initial release |
