# Known Problems & Risks

Track issues, blockers, and deferred decisions here. Keep open issues at the top and move resolved items to the bottom.

## Open Issues

### BAS Workflow Validation Blocker
**Status**: Open (Blocked by UI implementation)
**Severity**: Medium
**Description**: Integration phase failing because 4 BAS UI workflows reference selectors that don't exist in the template UI yet. Workflows have been converted to new BAS schema (nodes/edges format) but BAS strict mode validation rejects @selector/ tokens when the underlying data-testid selectors aren't defined in ui/src/consts/selectors.ts.
**Affected Workflows**:
- test/playbooks/capabilities/01-light-scanning/ui/global-dashboard.json (TM-UI-001)
- test/playbooks/capabilities/01-light-scanning/ui/scenario-detail.json (TM-UI-003, TM-UI-004)
- test/playbooks/capabilities/04-ui-dashboard/ui/accessibility.json (TM-UI-007)
- test/playbooks/capabilities/04-ui-dashboard/ui/theme.json (TM-UI-006)
**Root Cause**: UI is placeholder React template. Needs actual dashboard implementation with scenario table, file table, and interactive components to define proper selectors.
**Mitigation**: Workflows are properly structured and ready to execute once UI components exist.
**Next Steps**: Implement OT-P0-009 UI Dashboard module to add real components and selectors (TM-UI-001, TM-UI-003, TM-UI-004, TM-UI-006, TM-UI-007).

---

### Vitest 2.x CLI Coverage Incompatibility
**Status**: Open (Systemic issue)
**Severity**: Low
**Description**: The test runner (scripts/scenarios/testing/unit/node.sh) passes vitest 1.x-style CLI coverage flags (`--coverage.reporter`, `--coverage.thresholds.*`) which are not supported in vitest 2.x. This causes Node.js unit tests to fail during phased testing.
**Mitigation**:
- Coverage configuration moved to vite.config.ts for vitest 2.x compatibility
- Placeholder tests created to verify test infrastructure works
- Manual `pnpm test` works correctly (without CLI coverage flags)
**Root Cause**: Systemic - affects all scenarios using vitest 2.x. The test runner script needs to detect vitest version and adjust flags accordingly.
**Next Steps**:
- File issue with test framework maintainers (outside scenario scope)
- For now, configure coverage in vite.config.ts (already done)
- Update test runner to detect vitest 2.x and skip incompatible flags (cross-scenario fix)

---


### AI Cost Control
**Status**: Open
**Severity**: High
**Description**: Smart scanning could incur runaway AI costs if not properly batched and throttled.
**Mitigation**:
- Strict batch limits (10 files/batch, 5 concurrent batches)
- Global campaign concurrency limit (K=3)
- Force-scan queueing to prevent agent loops
- Optional: per-scenario monthly token budgets

**Next Steps**: Implement batch configuration (TM-SS-001) and campaign concurrency (TM-AC-002) in P0.

---

### Makefile Inconsistency
**Status**: Open
**Severity**: Medium
**Description**: Not all scenarios have standardized Makefiles with `make lint` and `make type` targets.
**Mitigation**:
- Gracefully skip scenarios without Makefiles (warn but don't fail)
- Document Makefile standards for new scenarios
- Consider fallback to direct tool invocation (eslint, golangci-lint) if Makefile absent

**Next Steps**: Implement Makefile detection and fallback logic in light scanning (TM-LS-001, TM-LS-002).

---

### Code-Smell Overlap
**Status**: Open
**Severity**: Medium
**Description**: Significant overlap with code-smell scenario (both do AI-powered code analysis).
**Options**:
1. **Integration**: tidiness-manager calls code-smell for pattern analysis (P2 feature)
2. **Differentiation**: tidiness focuses on structure (length, organization), code-smell focuses on patterns (anti-patterns, smells)
3. **Merge**: Combine scenarios into single code-quality-manager

**Next Steps**: Coordinate with code-smell maintainers, define clear boundaries, implement integration in P2 (TM-FF-003).

---

### False Positive Noise
**Status**: Open
**Severity**: Low
**Description**: Long files aren't always bad; generated code, complex algorithms, etc. may legitimately exceed thresholds.
**Mitigation**:
- Configurable per-scenario thresholds (TM-DA-004, TM-DA-005)
- Mark-as-ignored capability (TM-IM-002)
- Context-aware flagging (e.g., don't flag generated files)

**Next Steps**: Implement configurable thresholds and ignore workflow in P1.

---

### visited-tracker Availability
**Status**: Open
**Severity**: Low
**Description**: visited-tracker is optional dependency; campaign management should gracefully degrade if unavailable.
**Mitigation**:
- Fallback to timestamp-based tracking if visited-tracker unavailable
- Document visited-tracker as "recommended" not "required"
- Test both code paths (with and without visited-tracker)

**Next Steps**: Implement fallback logic in smart scanning (TM-SS-003, TM-SS-004).

---

## Deferred Ideas

### CI/CD Integration (P2)
**Decision**: Deferred to P2 (OT-P2-007)
**Rationale**: Focus on core scanning and campaign management first; CI/CD hooks can be added later once agent API proven.
**Revisit**: After P1 completion or if early adopters request it.

---

### Custom Rule Engine (P2)
**Decision**: Deferred to P2 (OT-P2-005)
**Rationale**: Built-in categories (length, duplication, dead_code, complexity) cover 80% of use cases. Custom rules add complexity without proportional value early on.
**Revisit**: After P1 if users report specific missing rule types.

---

### Remediation Automation (P2)
**Decision**: Deferred to P2 (OT-P2-004)
**Rationale**: Auto-fixing is risky; better to prove detection/recommendation value first. Integration with code-smell (which has auto-fix capability) may be sufficient.
**Revisit**: If manual remediation becomes bottleneck.

---

## Resolved Issues

*(None yet - initialization only)*

---

## Architecture Decisions

### Postgres for Everything vs. File-Based Caching
**Decision**: Postgres for persistent data (issues, campaigns, scan history); file-based JSON for light scan caching.
**Rationale**: Postgres provides queryability for API; file-based JSON provides portability and transparency for debugging. Hybrid approach balances performance and flexibility.
**Trade-offs**: Light scan caching won't survive postgres restarts, but that's acceptable (re-scan is cheap).

---

### CLI-First vs. UI-First
**Decision**: Build agent API and CLI first (P0), then UI (also P0 but can lag slightly).
**Rationale**: Agent integration is the primary use case; UI is for human oversight. CLI can validate API contracts before UI development starts.
**Trade-offs**: Early testers won't have visual feedback, but CLI output is sufficient for development/validation.

---

### Visited-Tracker Integration vs. Custom Tracking
**Decision**: Integrate with visited-tracker scenario (optional dependency).
**Rationale**: Visited-tracker already solves campaign-based file tracking; no need to duplicate. Making it optional ensures tidiness-manager works standalone.
**Trade-offs**: Two code paths to maintain (with and without visited-tracker), but avoids reinventing the wheel.

---

### React UI vs. Terminal UI
**Decision**: React dashboard for tidiness-manager.
**Rationale**: Complex data tables, sortable columns, and campaign management are better suited to web UI than TUI. Agents use CLI; humans use dashboard.
**Trade-offs**: Heavier dependency footprint, but better UX for non-trivial interactions.

---

## Risk Register

| Risk | Probability | Impact | Mitigation Status |
|------|-------------|--------|-------------------|
| AI cost runaway | Medium | High | Planned (batch limits, concurrency) |
| False positive noise | High | Medium | Planned (configurable thresholds, ignore workflow) |
| Makefile inconsistency | Medium | Low | Planned (graceful fallback) |
| visited-tracker unavailable | Low | Low | Planned (timestamp fallback) |
| Code-smell overlap confusion | Low | Medium | Open (coordination needed) |

---

## Instructions for Future Agents

1. **Add new issues** to "Open Issues" section with status, severity, description, mitigation, next steps
2. **Move resolved issues** to "Resolved Issues" with resolution date and approach
3. **Update risk register** when implementing mitigations (change status to "Complete")
4. **Document architecture decisions** when making non-obvious design choices
5. **Keep deferred ideas** visible for P2+ planning
