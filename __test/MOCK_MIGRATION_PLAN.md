# Mock System Migration Plan

## Executive Summary
Migrate from two extremes (oversimplified new mocks and overengineered legacy mocks) to a three-tier system that balances simplicity with functionality.

## Current State Analysis

### Metrics
| Metric | New Mocks | Legacy Mocks | 
|--------|-----------|--------------|
| **Total Lines** | 1,013 | 27,920 |
| **Files** | 4 | 30+ |
| **Avg Lines/File** | ~250 | ~930 |
| **State Management** | None | File-based persistence |
| **Complexity** | Simple case statements | Full state machines |

### Problems
- **New mocks**: Too simple, can't test workflows, no state, no error handling
- **Legacy mocks**: Overengineered, 30x code size, hard to maintain

## Three-Tier Architecture

### Tier 1: Simple Mocks (100-200 lines)
**Purpose**: Basic connectivity and health checks
**Use for**: CI/CD smoke tests, basic validation
```bash
redis-cli() {
    case "$*" in
        "ping") echo "PONG" ;;
        "info") echo "redis_version:7.0.0" ;;
        *) echo "OK" ;;
    esac
}
```

### Tier 2: Stateful Mocks (300-500 lines)
**Purpose**: Integration testing with minimal state
**Use for**: Workflow validation, integration tests
```bash
declare -gA REDIS_DATA=()
redis-cli() {
    case "$1" in
        "set") REDIS_DATA[$2]="$3"; echo "OK" ;;
        "get") echo "${REDIS_DATA[$2]:-nil}" ;;
        "del") unset REDIS_DATA[$2]; echo "1" ;;
    esac
}
```

### Tier 3: Full Simulation (1000+ lines)
**Purpose**: Complete service simulation
**Use for**: Critical services only, complex edge cases
**Consider**: Using actual test containers instead

## Migration Strategy

### Phase 1: Categorization (Day 1)
- [x] Analyze existing mocks
- [x] Define tier criteria
- [x] Create migration plan

### Phase 2: Templates (Day 2)
- [ ] Create Tier 2 template
- [ ] Define standard structure
- [ ] Add state management helpers

### Phase 3: Critical Services (Week 1)
Priority order based on usage and criticality:

1. **Redis** (Tier 2) - Event bus, caching
2. **PostgreSQL** (Tier 2) - Primary datastore  
3. **N8n** (Tier 2) - Workflow automation
4. **Ollama** (Tier 2) - AI operations
5. **Qdrant** (Tier 2) - Vector operations

### Phase 4: Secondary Services (Week 2)
6. **Minio** (Tier 1) - File storage
7. **Vault** (Tier 1) - Secrets management
8. **BrowserLess** (Tier 1) - Web automation
9. **ComfyUI** (Tier 1) - Image generation
10. **Others** (Tier 1) - Remaining services

## Service Classification

### Tier 2 Services (Stateful)
Services requiring state for meaningful testing:
- Redis - State between commands
- PostgreSQL - Data persistence
- N8n - Workflow state
- Ollama - Model context
- Qdrant - Vector storage

### Tier 1 Services (Stateless)
Services with simple request/response:
- Minio - Simple S3 operations
- Vault - Key/value operations
- Judge0 - Code execution
- Whisper - Transcription
- SearXNG - Search

### Tier 3 Candidates
Consider containers instead:
- Full PostgreSQL with transactions
- Redis with pub/sub and clustering
- Complex N8n workflows

## Standard Mock Structure

### File Organization
```
__test/
├── mocks/                  # Active mocks
│   ├── tier1/             # Simple mocks
│   │   ├── minio.sh
│   │   └── vault.sh
│   ├── tier2/             # Stateful mocks
│   │   ├── redis.sh
│   │   ├── postgres.sh
│   │   └── n8n.sh
│   └── TEMPLATE_TIER2.sh  # Template for new mocks
├── mocks-legacy/          # Keep for reference
└── MOCK_MIGRATION_PLAN.md # This document
```

### Tier 2 Template Structure
```bash
#!/usr/bin/env bash
# Resource: [RESOURCE_NAME]
# Tier: 2 (Stateful Mock)
# Coverage: [List operations covered]

# === Configuration ===
declare -gA [RESOURCE]_STATE=()
declare -g [RESOURCE]_DEBUG="${[RESOURCE]_DEBUG:-}"

# === Core Mock ===
[main_command]() {
    [[ -n "$[RESOURCE]_DEBUG" ]] && echo "[MOCK] $0 $*" >&2
    
    # State management
    local key="$1" value="$2"
    
    case "${1:-}" in
        # Basic operations
        [operation1]) 
            [RESOURCE]_STATE[$key]="$value"
            echo "[success_response]"
            ;;
        [operation2])
            echo "${[RESOURCE]_STATE[$key]:-[default]}"
            ;;
        *)
            echo "[safe_default]"
            ;;
    esac
}

# === Convention-based Test Functions ===
test_[resource]_connection() {
    # Test basic connectivity
}

test_[resource]_health() {
    # Test service health
}

test_[resource]_basic() {
    # Test basic operations with state
}

# === State Management ===
[resource]_mock_reset() {
    [RESOURCE]_STATE=()
}

[resource]_mock_set_state() {
    local key="$1" value="$2"
    [RESOURCE]_STATE[$key]="$value"
}

[resource]_mock_get_state() {
    local key="$1"
    echo "${[RESOURCE]_STATE[$key]:-}"
}

# === Export Functions ===
export -f [main_command]
export -f test_[resource]_connection
export -f test_[resource]_health
export -f test_[resource]_basic
export -f [resource]_mock_reset
```

## Testing Strategy

### Compatibility Requirements
1. Maintain convention-based test functions
2. Support existing integration tests
3. Provide migration path from legacy

### Test Coverage Goals
- Tier 1: Connection, health, single operation
- Tier 2: Workflows, state changes, basic errors
- Tier 3: Transactions, concurrency, edge cases

## Success Metrics

### Code Reduction
- Target: 70% reduction in total mock code
- Current: 28,933 lines → Target: 8,000 lines

### Test Coverage
- Maintain 100% of critical path coverage
- Accept reduced coverage for edge cases

### Performance
- Tier 1: < 10ms per operation
- Tier 2: < 50ms per operation  
- Tier 3: < 100ms per operation

## Implementation Timeline

| Week | Tasks | Deliverables |
|------|-------|--------------|
| 1 | Templates, Redis, PostgreSQL | 2 Tier 2 mocks |
| 2 | N8n, Ollama, Qdrant | 3 Tier 2 mocks |
| 3 | Remaining services | 25+ Tier 1 mocks |
| 4 | Testing, documentation | Complete migration |

## Risk Mitigation

### Backward Compatibility
- Keep legacy mocks during transition
- Run parallel tests to verify parity
- Gradual rollout with feature flags

### Knowledge Transfer
- Document patterns in templates
- Create migration guide
- Record common pitfalls

## Decision Log

### Why Tier 2 for Critical Services?
- Balance between simplicity and functionality
- 80% of test needs with 20% of complexity
- Fast enough for CI/CD, functional enough for integration

### Why Keep Some Legacy?
- Reference implementation
- Complex edge case testing
- Gradual migration reduces risk

### Why Not Just Use Containers?
- Speed: Mocks are 100x faster
- Simplicity: No Docker required
- Deterministic: Predictable responses
- But: Consider for Tier 3 needs

## Next Steps

1. ✅ Store this plan
2. ⏳ Create Tier 2 template
3. ⏳ Migrate Redis to Tier 2
4. ⏳ Migrate PostgreSQL to Tier 2
5. ⏳ Test and validate
6. ⏳ Continue migration

---

*Last Updated: 2024-08-23*
*Version: 1.0*
*Author: Vrooli Test Infrastructure Team*