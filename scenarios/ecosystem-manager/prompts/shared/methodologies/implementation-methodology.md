# Implementation Methodology

## Core Principles
**Incremental**: One change at a time; test after each; never break working functionality
**Standards**: Follow codebase patterns; maintain consistency; adhere to the lastest standards  
**Validation**: Test continuously; ensure gates pass; document verification

## Implementation Phases

### Phase 1: Setup (10% effort)
- Verify dependencies, tools, permissions
- Define success criteria aligned with PRD requirement (see 'prd-protocol' section)
- Prepare rollback procedure and test commands

### Phase 2: Core Implementation (60% effort)

**Generators (Creating New):**
- Start from appropriate template/reference
- Implement minimal viable functionality first
- Integrate incrementally: dependencies → service.json → CLI → API

**Improvers (Enhancing Existing):**
- Focus on ONE PRD requirement per iteration (see 'prd-protocol' section)
- Preserve existing functionality (no regressions)
- Update integration points as needed

**Quality Standards:**
- Clear descriptive naming; helpful error messages
- Graceful error handling with recovery hints
- Follow established CLI/API patterns

### Phase 3: Testing & Validation (20% effort)
**Required Tests:**
- Functional: core features, edge cases, error handling
- Integration: dependencies, APIs, CLI commands, UI (if applicable)
- Performance: response times, resource usage, scaling

**Test Commands:**
- **Resources**: See 'resource-testing-reference' section
- **Scenarios**: See 'scenario-testing-reference' section

### Phase 4: Documentation & Finalization (10% effort)
- Update README: features, usage, troubleshooting
- Update relevant docs: e.g. PROBLEMS.md with problems you discovered
- Update PRD per 'prd-protocol' section guidelines

- First-time success rate >80%
- 100% test pass rate
- PRD advancement with each iteration (see 'prd-protocol' section)
- Clear, complete documentation
- Reusable components

Quality over speed. Small steps win. Test everything.