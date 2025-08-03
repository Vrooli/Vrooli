# Resource Structure Standardization Plan

## Executive Summary
Standardize the internal structure of all resources in scripts/resources/ to eliminate redundancy, consolidate test infrastructure, and improve maintainability.

## Current State Analysis

### Problems Identified
1. **Test Infrastructure Duplication**
   - Centralized test infrastructure exists at `scripts/__test/resources/`
   - Individual resources have redundant test directories and fixtures
   - Multiple naming conventions: `test/`, `tests/`, `testing/`, `test_fixtures/`, `test-fixtures/`

2. **PostgreSQL Instance Explosion**
   - 16 test/debug instances cluttering postgres/instances/
   - Names like `test-client`, `test-comprehensive`, `test-debug`, `test-fix`

3. **Examples vs Fixtures Confusion**
   - 28 resources have examples/ directories
   - Examples often contain test data that should be fixtures
   - Actual usage examples mixed with test artifacts

4. **Inconsistent Structure**
   - Some resources have unique directories without clear purpose
   - Backup directories with timestamped files (agent-s2)
   - Root-level test files that should be organized

## Target Standardized Structure

```
resource-name/
├── README.md              # Brief overview + link to docs
├── manage.sh              # Main management script  
├── manage.bats            # Tests for manage.sh
│
├── config/                # Configuration (KEEP AS-IS)
│   ├── defaults.sh
│   ├── defaults.bats
│   ├── messages.sh
│   └── messages.bats
│
├── lib/                   # Core functionality (KEEP AS-IS)
│   ├── common.sh
│   ├── api.sh
│   ├── install.sh
│   ├── status.sh
│   └── *.bats            # Tests for each lib file
│
├── docs/                  # Documentation (if extensive)
│   └── *.md
│
└── [OPTIONAL - ONLY IF NEEDED]
    ├── docker/            # Docker configs (if containerized)
    ├── templates/         # Resource-specific templates
    └── schemas/           # Data schemas (for databases)
```

## Implementation Plan

### Phase 1: PostgreSQL Cleanup
**Priority: HIGH - Quick win, immediate impact**

1. Remove test instances from postgres/instances/:
   - test-client
   - test-comprehensive
   - test-comprehensive-2
   - test-debug
   - test-fix
   - test-fix-2
   - test-auto-port
   - test-claude
   - testing-template
   - And any other test-* instances

2. Keep only production instances:
   - development
   - production
   - local
   - vrooli-db

### Phase 2: Test Directory Consolidation
**Priority: HIGH - Major cleanup**

Resources with test directories to clean:
1. **agent-s2**
   - Remove: testing/, tests/, examples/*/testing/
   - Remove: backups/ directory
   - Keep: .bats files in lib/ and root

2. **unstructured-io**
   - Remove: test-fixtures/, tests/
   - Move unique fixtures to scripts/__test/resources/fixtures/
   - Keep: .bats files only

3. **windmill**
   - Remove: fixtures/, test_fixtures/
   - Keep: .bats files only

4. **node-red**
   - Remove: test_fixtures/
   - Keep: .bats files only

5. **huginn**
   - Remove: test-fixtures/, test_fixtures/
   - Keep: .bats files only

6. **whisper**
   - Remove: tests/ directory
   - Move audio test files to centralized fixtures

7. **comfyui**
   - Remove: test/ directory
   - Keep: .bats files only

### Phase 3: Examples Rationalization
**Priority: MEDIUM - Documentation improvement**

For each resource with examples/:
1. Review content to separate:
   - Real usage examples → Move to docs/examples.md
   - Test data → Move to scripts/__test/resources/fixtures/
   - Redundant content → Remove

2. Create standardized docs/examples.md format:
   ```markdown
   # Examples
   
   ## Basic Usage
   [code example]
   
   ## Advanced Configuration
   [code example]
   
   ## Integration with Other Resources
   [code example]
   ```

### Phase 4: Root File Cleanup
**Priority: LOW - Final polish**

1. Move root-level test files to appropriate locations
2. Remove redundant documentation files (keep only README.md + docs/)
3. Ensure consistent file naming

## Resources Requiring Changes

### High Impact (Most Changes Needed)
1. agent-s2 - Complex structure, multiple test dirs, backups
2. postgres - 16 test instances to remove
3. unstructured-io - Multiple test directories and fixtures

### Medium Impact
4. windmill - fixtures/ and test_fixtures/
5. node-red - test_fixtures/
6. huginn - test-fixtures/ and test_fixtures/
7. whisper - tests/ directory
8. comfyui - test/ directory

### Low Impact (Already Clean)
- ollama
- redis
- minio
- searxng
- judge0
- browserless
- vault
- qdrant
- questdb

## Success Metrics
- [ ] No test/fixture directories in individual resources
- [ ] All test data centralized in scripts/__test/resources/fixtures/
- [ ] PostgreSQL instances reduced from 20+ to 4-5
- [ ] Consistent structure across all 30+ resources
- [ ] Examples converted to documentation

## Rollback Plan
- All deletions will be done with git tracking
- Can revert changes if issues arise
- Test resource functionality after each phase

## Testing Strategy
After each phase:
1. Run resource management tests: `./resource/manage.bats`
2. Verify resource can start/stop: `./resource/manage.sh --action start/stop`
3. Check that centralized tests still pass

## Notes
- Preserve all .bats test files in lib/ and root
- Keep config/ structure unchanged (most consistent part)
- Maintain backward compatibility for resource management scripts