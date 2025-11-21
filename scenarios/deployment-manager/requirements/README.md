# Requirements Registry: deployment-manager

**Total**: 99 requirements (37 P0, 36 P1, 26 P2)
**Last Updated**: 2025-11-21 (Generator Phase)

---

## Module Structure

Requirements organized by PRD operational targets into 14 functional modules:

| Module | Directory | Count | Priority | PRD Section |
|--------|-----------|-------|----------|-------------|
| Dependency Analysis & Fitness Scoring | `01-dependency-analysis` | 6 | P0 | Dependency analysis, fitness scoring engine, resource tallies |
| Dependency Swapping | `02-dependency-swapping` | 5 | P0 | Swap suggestions, impact analysis, cascading detection |
| Deployment Profile Management | `03-profile-management` | 6 | P0 | Profile CRUD, versioning, export/import |
| Secret Management Integration | `04-secret-management` | 5 | P0 | Secret identification, categorization, template generation |
| Pre-Deployment Validation | `05-pre-deployment-validation` | 5 | P0 | Validation checks, cost estimation, remediation UX |
| Deployment Orchestration | `06-deployment-orchestration` | 7 | P0 | Deploy trigger, log streaming, scenario-to-* integration |
| Dependency Graph Visualization | `07-dependency-visualization` | 3 | P0 | Interactive graph, table view, rendering performance |
| Post-Deployment Monitoring | `08-post-deployment-monitoring` | 8 | P1 | Health tracking, metrics, alerting, dashboards |
| Update & Rollback Management | `09-update-rollback` | 6 | P1 | Change detection, zero-downtime updates, auto-rollback |
| Multi-Tier Deployment Orchestration | `10-multi-tier-orchestration` | 5 | P1 | Multi-tier profiles, parallel orchestration, per-tier status |
| Agent Integration & Custom Swaps | `11-agent-integration` | 17 | P1 | AI migration proposals, custom swap requests, UX/accessibility |
| Enterprise Compliance & Licensing | `12-enterprise-compliance` | 8 | P2 | License validation, audit logging, approval workflows |
| CLI Automation & CI/CD | `13-cli-automation` | 4 | P2 | Headless deployment, machine-readable output, exit codes |
| Advanced Features & Visual Builder | `14-advanced-features` | 14 | P2 | Cost tracking, visual editor, templates, tier recommendations |

---

## Requirement ID Pattern

- **P0 (Core)**: `REQ-P0-001` through `REQ-P0-037` — Must ship for viability
- **P1 (Enhanced)**: `REQ-P1-001` through `REQ-P1-036` — Should have post-launch
- **P2 (Advanced)**: `REQ-P2-001` through `REQ-P2-026` — Future/expansion ideas

Each requirement links to its PRD operational target via `prd_ref` field (e.g., `OT-P0-001`).

---

## Test Tagging

Tag all tests with `[REQ:ID]` for automated tracking:

**Go (API tests)**:
```go
// [REQ:REQ-P0-001] Dependency Aggregation
func TestDependencyAggregation(t *testing.T) { ... }
```

**TypeScript (UI tests)**:
```typescript
// [REQ:REQ-P0-035] Interactive Dependency Graph
it('[REQ:REQ-P0-035] renders graph in <3s for 100 deps', () => { ... });
```

**Bash (Integration)**:
```bash
# [REQ:REQ-P0-028] One-Click Deployment Trigger
test_one_click_deployment() { ... }
```

---

## Lifecycle & Auto-Sync

1. **Operational targets** (PRD.md) map to requirement modules (this directory)
2. **Tests execute** with `[REQ:ID]` tags → results captured per phase
3. **Auto-sync updates** `requirements/index.json` validation arrays after test runs
4. **PRD checkboxes** flip automatically when requirements validated
5. **Never edit validation status by hand** — always via test runs + sync

### Key Commands
```bash
# Run full test suite
make test

# Sync requirement statuses from test results
vrooli scenario requirements sync deployment-manager

# Lint PRD to ensure all OT-* targets have requirements
vrooli scenario requirements lint-prd deployment-manager

# Review requirement coverage snapshot
vrooli scenario requirements snapshot deployment-manager
```

---

## Implementation Priority

### Phase 1: P0 Core (Modules 01-07)
Focus: Dependency analysis → fitness scoring → profile management → validation → orchestration → visualization

**Critical Path**:
1. Module 01 (dependency data from scenario-dependency-analyzer)
2. Module 02 (swap engine with swap database)
3. Module 03 (profile storage in postgres)
4. Module 06 (orchestrator calling scenario-to-* packagers)
5. Module 07 (React Flow dependency graph UI)

### Phase 2: P1 Enhanced (Modules 08-11)
Focus: Monitoring, updates/rollbacks, multi-tier, agent integration

### Phase 3: P2 Advanced (Modules 12-14)
Focus: Enterprise compliance, CLI automation, visual builder

---

## Key Integration Points

- **scenario-dependency-analyzer**: Dependency tree data source (REQ-P0-001, REQ-P1-031)
- **secrets-manager**: Secret classification and templates (REQ-P0-018 through REQ-P0-022)
- **app-issue-tracker**: Migration task creation when swaps approved (REQ-P1-022, REQ-P1-024, REQ-P1-035)
- **scenario-to-*** packagers: Platform-specific deployment execution (REQ-P0-033, REQ-P0-034)
- **claude-code/ollama**: AI migration strategy suggestions (REQ-P1-020, REQ-P1-021)

---

## Related Documentation

- **PRD.md**: Operational targets and feature descriptions (canonical source)
- **docs/PROGRESS.md**: Implementation progress log
- **docs/PROBLEMS.md**: Blockers and deferred decisions
- **docs/RESEARCH.md**: Uniqueness check, integration analysis, external references
- **Repo-level guides**:
  - `docs/testing/guides/requirement-tracking.md` — Full requirement system docs
  - `docs/testing/guides/requirement-tracking-quick-start.md` — Quick reference
  - `docs/testing/guides/ui-automation-with-bas.md` — Browser automation for UI validation

---

**Next Steps**: Implement P0 requirements first, tag all tests with `[REQ:ID]`, run test suite, sync statuses, validate coverage.
