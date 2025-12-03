# Gold Standard Testing Examples

**Status**: Active
**Last Updated**: 2025-12-02

---

This document showcases exemplary testing implementations that demonstrate best practices.

## Visited Tracker - Complete Testing Suite

**Location**: `/scenarios/visited-tracker/`

### Why It's Gold Standard
- **79.4% Go coverage** with comprehensive test cases
- **Safe BATS implementation** following all safety guidelines
- **Complete phased testing** with all 7 phases implemented
- **Reusable test helpers** for HTTP testing
- **Well-documented** with inline comments and guides

### Key Files

#### API Testing (`api/TESTING_GUIDE.md`)

Demonstrates:
- Comprehensive handler testing with httptest
- Table-driven tests for multiple scenarios
- Gorilla Mux router testing with URL variables
- Reusable HTTP test helpers
- Proper error case coverage

```go
// Example: Table-driven test
func TestVisitHandler(t *testing.T) {
    tests := []struct {
        name           string
        method         string
        body           string
        expectedStatus int
        expectedBody   string
    }{
        {
            name:           "Valid visit",
            method:         "POST",
            body:           `{"files":["test.go"]}`,
            expectedStatus: http.StatusOK,
            expectedBody:   "recorded",
        },
        {
            name:           "Invalid JSON",
            method:         "POST",
            body:           `{invalid}`,
            expectedStatus: http.StatusBadRequest,
            expectedBody:   "Invalid request",
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            // Test implementation
        })
    }
}
```

#### CLI Testing (`cli/visited-tracker.bats`)

Demonstrates:
- **Safe variable handling** - Variables set before skip conditions
- **Guarded teardown** - Multiple validation layers
- **Comprehensive coverage** - 30+ test cases
- **Service state handling** - Tests with/without API running
- **JSON validation** - Proper output validation

```bash
# Example: Safe BATS setup/teardown
setup() {
    # Variables FIRST (critical!)
    export TEST_FILE_PREFIX="/tmp/visited-tracker-cli-test"

    # Skip conditions AFTER
    if ! command -v visited-tracker >/dev/null 2>&1; then
        skip "visited-tracker CLI not installed"
    fi
}

teardown() {
    # Multiple guards for safety
    if [ -n "${TEST_FILE_PREFIX:-}" ]; then
        rm -f "${TEST_FILE_PREFIX}"* 2>/dev/null || true
    fi
}
```

---

## Browser Automation Studio - UI Testing Excellence

**Location**: `/scenarios/browser-automation-studio/`

### Why It's Notable
- **Puppeteer integration** for browser testing
- **Visual regression testing** capabilities
- **Workflow-based testing** with JSON configs
- **Screenshot comparison** for UI validation

### Key Patterns

#### UI Workflow Testing

```json
{
  "name": "user-login-flow",
  "steps": [
    {
      "action": "navigate",
      "url": "http://localhost:3000/login"
    },
    {
      "action": "type",
      "selector": "#username",
      "text": "testuser"
    },
    {
      "action": "click",
      "selector": "#submit"
    },
    {
      "action": "waitFor",
      "selector": ".dashboard"
    },
    {
      "action": "screenshot",
      "name": "dashboard-loaded"
    }
  ]
}
```

---

## Resource Testing Examples

### PostgreSQL Integration Testing

**Location**: `/resources/postgres/test/`

Demonstrates:
- Database connection validation
- Schema migration testing
- Transaction rollback testing
- Connection pool testing

```bash
# Example: PostgreSQL health check
test_postgres_health() {
    local db_url="postgresql://user:pass@localhost:5432/testdb"

    if psql "$db_url" -c "SELECT 1" >/dev/null 2>&1; then
        echo "PostgreSQL is healthy"
        return 0
    else
        echo "PostgreSQL connection failed"
        return 1
    fi
}
```

### Redis Integration Testing

**Location**: `/resources/redis/test/`

Demonstrates:
- Key-value operation testing
- Pub/sub functionality testing
- Cache invalidation testing
- Cluster configuration testing

```bash
# Example: Redis operations test
test_redis_operations() {
    # Set a key
    redis-cli SET test:key "value" || return 1

    # Get the key
    value=$(redis-cli GET test:key)
    [ "$value" = "value" ] || return 1

    # Clean up
    redis-cli DEL test:key

    echo "Redis operations working"
}
```

---

## Testing Anti-Patterns to Avoid

### The File Deletion Bug

**What NOT to do:**
```bash
# DANGEROUS - Can delete everything
teardown() {
    rm -f "${PREFIX}"*  # If PREFIX is empty = disaster
}
```

### Hardcoded Ports

**What NOT to do:**
```bash
# WRONG - Port might be different
API_URL="http://localhost:8080"
```

**Do this instead:**
```bash
# RIGHT - Dynamic port discovery using vrooli CLI
API_PORT=$(vrooli scenario port my-scenario API_PORT)
API_URL="http://localhost:${API_PORT}"
```

### Assuming Test Order

**What NOT to do:**
```bash
@test "step 1" {
    echo "data" > /tmp/shared.txt
}

@test "step 2" {
    cat /tmp/shared.txt  # Might not exist!
}
```

---

## Best Practices Demonstrated

### 1. Comprehensive Coverage
- Test success paths
- Test error conditions
- Test edge cases
- Test timeouts and retries

### 2. Isolation
- Each test is independent
- No shared state between tests
- Clean setup/teardown

### 3. Clarity
- Descriptive test names
- Clear assertions
- Helpful error messages

### 4. Performance
- Fast-running tests (< 5s each)
- Parallel execution where possible
- Minimal external dependencies

### 5. Maintainability
- Reusable test helpers
- DRY principle
- Well-documented

---

## Coverage Standards

### Bronze Level (50-70%)
```bash
go test ./... -cover
# coverage: 55.2% of statements
```

### Silver Level (70-80%)
```bash
go test ./... -cover
# coverage: 74.8% of statements
```

### Gold Level (80-90%)
```bash
go test ./... -cover
# coverage: 85.3% of statements
```

### Diamond Level (90%+)
```bash
go test ./... -cover
# coverage: 92.7% of statements
```

---

## Quick Links to Examples

### Complete Test Suites
- [Visited Tracker Tests](/scenarios/visited-tracker/test/)
- [Browser Automation Studio](/scenarios/browser-automation-studio/test/)

### Specific Test Types
- [Go Handler Tests](/scenarios/visited-tracker/api/main_test.go)
- [BATS CLI Tests](/scenarios/visited-tracker/cli/visited-tracker.bats)
- [Integration Tests](/scenarios/visited-tracker/test/phases/test-integration.sh)

### Test Helpers
- [HTTP Test Helpers](/scenarios/visited-tracker/api/test_helpers.go)

---

## Achieving Gold Standard

To achieve gold standard testing:

### 1. Structure
- All required files present
- Proper directory organization
- Modern configuration (service.json)

### 2. Coverage
- Unit tests > 80%
- Integration tests for all endpoints
- CLI tests with BATS
- Business logic validation

### 3. Safety
- No dangerous patterns
- Proper variable validation
- Path restrictions to /tmp
- Error handling

### 4. Documentation
- README with test instructions
- Inline code comments
- Test case descriptions

### 5. Performance
- Tests complete in < 5 minutes
- Individual tests < 5 seconds
- Parallel execution where possible

---

## See Also

- [Testing Strategy](../concepts/strategy.md) - Three-layer approach
- [Safety Guidelines](../safety/GUIDELINES.md) - Critical safety rules
- [Test Runners](test-runners.md) - Language-specific runners
