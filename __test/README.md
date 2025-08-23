# Vrooli Testing Infrastructure v2.0 - Convention-Based Design

## Overview

This testing infrastructure is built on **convention-over-configuration** principles to be completely resource-agnostic and infinitely extensible. Unlike traditional testing frameworks that require hardcoded knowledge of every resource type, this system discovers and tests resources automatically based on naming conventions.

## 🎯 Core Design Principles

### 1. **Resource Agnostic**
The testing framework has ZERO hardcoded knowledge about specific resources (postgres, redis, ollama, etc.). This is intentional and a **feature**, not a flaw.

### 2. **Convention Driven**
Resources are tested by discovering functions that follow standard naming patterns:
- `test_<resource>_connection()` - Basic connectivity/availability test
- `test_<resource>_health()` - Health/status endpoint test  
- `test_<resource>_basic()` - Basic functionality test
- `test_<resource>_advanced()` - Advanced features test (optional)

### 3. **Zero Maintenance**
Adding new resources requires **zero changes** to the testing framework. Simply:
1. Create a mock file in `mocks/`
2. Follow the naming conventions
3. The framework discovers and tests automatically

## 🏗️ Architecture

```
scripts/__test-revised/
├── run-tests.sh              # Main orchestrator
├── phases/
│   ├── test-static.sh        # Static analysis (shellcheck, bash -n)
│   ├── test-resources.sh     # Resource validation (CONVENTION-BASED)
│   ├── test-scenarios.sh     # Scenario validation
│   └── test-bats.sh          # BATS test execution
├── mocks/                    # Resource mocks (convention-driven)
│   ├── postgres.sh           # PostgreSQL mock
│   ├── redis.sh              # Redis mock  
│   ├── ollama.sh             # Ollama mock
│   └── system.sh             # System commands mock
├── shared/                   # Common utilities
│   ├── logging.bash          # Consistent logging
│   ├── test-helpers.bash     # Test utilities
│   └── cache.bash            # Intelligent caching
└── cache/                    # Test result cache
```

## 🔬 Resource Testing (Convention-Based)

### How It Works

1. **Discovery**: Framework reads enabled resources from `service.json`
2. **Function Discovery**: For each resource, looks for standard test functions
3. **Execution**: Runs discovered tests using caching framework
4. **Fallback**: Uses generic validation if no specific tests exist

### Example Resource Flow

```bash
# For resource "postgres":
# 1. Framework looks for these functions:
test_postgres_connection()  # ✅ Found in mocks/postgres.sh
test_postgres_health()      # ✅ Found in mocks/postgres.sh  
test_postgres_basic()       # ✅ Found in mocks/postgres.sh
test_postgres_advanced()    # ❌ Not found (optional)

# 2. Framework runs each discovered function
# 3. Results are cached and reported
```

## 📝 Creating New Resource Mocks

### Step 1: Create Mock File
```bash
# Create mocks/mynewresource.sh
#!/usr/bin/env bash
# Mock for MyNewResource
#
# NAMING CONVENTIONS: Follow standard patterns:
#   test_mynewresource_connection() : Test connectivity
#   test_mynewresource_health()     : Test health
#   test_mynewresource_basic()      : Test basic functionality

# Mock the main command
mynewresource() {
    case "$1" in
        "status") echo "running" ;;
        *) echo "mynewresource: mock command" ;;
    esac
}

# Standard test functions
test_mynewresource_connection() {
    mynewresource status >/dev/null 2>&1
    return $?
}

test_mynewresource_health() {
    # Test health endpoint or functionality
    return 0
}

test_mynewresource_basic() {
    # Test basic operations
    return 0
}

# Export functions
export -f mynewresource
export -f test_mynewresource_connection test_mynewresource_health test_mynewresource_basic
```

### Step 2: Enable Resource
```json
// In .vrooli/service.json
{
  "resources": {
    "category": {
      "mynewresource": {
        "enabled": true
      }
    }
  }
}
```

### Step 3: Run Tests
```bash
./run-tests.sh resources --verbose
```

**That's it!** The framework automatically discovers and tests your resource.

## 🚀 Usage Examples

### Run All Tests
```bash
./run-tests.sh
```

### Run Specific Phase
```bash
./run-tests.sh resources          # Only resource testing
./run-tests.sh static             # Only static analysis
./run-tests.sh scenarios          # Only scenario validation
./run-tests.sh bats               # Only BATS tests
```

### Verbose Mode
```bash
./run-tests.sh resources --verbose  # Shows function discovery process
```

### Clear Cache
```bash
./run-tests.sh resources --clear-cache  # Force re-test everything
```

## 🔄 Why This Design?

### Problems with Traditional Approach
```bash
# Traditional (BAD): Hardcoded resource knowledge
case "$resource" in
    "postgres") test_postgres_specific_logic ;;
    "redis")    test_redis_specific_logic ;;
    # ... 50 more resources
esac
```

**Issues:**
- Every new resource requires code changes
- Framework becomes bloated with resource-specific logic
- Tight coupling between framework and resources
- Maintenance nightmare

### Our Convention-Based Approach
```bash
# Convention-based (GOOD): Generic discovery
for test_type in connection health basic; do
    test_function="test_${resource}_${test_type}"
    if command -v "$test_function" >/dev/null 2>&1; then
        run_test "$test_function"
    fi
done
```

**Benefits:**
- Zero maintenance for new resources
- Framework stays simple and focused
- Loose coupling through conventions
- Self-documenting through function names

## 🧪 Testing the Testing Framework

The framework follows its own principles - it can test itself:

```bash
# Test the static analysis phase
./phases/test-static.sh --dry-run

# Test the resource phase  
./phases/test-resources.sh --verbose

# Test individual components
bash -n phases/test-resources.sh  # Syntax check
shellcheck phases/test-resources.sh  # Static analysis
```

## 💡 Key Insights

1. **Verification Functions Were Removed**: They added complexity without value. If a mock is broken, tests will fail naturally.

2. **No Resource-Specific Code**: The framework doesn't know what postgres, redis, or ollama are. This is intentional.

3. **Convention Discovery**: Functions are discovered at runtime, not compile time.

4. **Extensibility by Design**: Adding resources is a configuration change, not a code change.

5. **Self-Documenting**: Function names clearly indicate what they test.

## 🔍 Debugging

### Enable Debug Mode
```bash
LOG_LEVEL=DEBUG ./run-tests.sh resources --verbose
```

### Check Function Discovery
```bash
# List all available test functions
declare -F | grep test_.*_connection
declare -F | grep test_.*_health  
declare -F | grep test_.*_basic
```

### Validate Mock Syntax
```bash
bash -n mocks/postgres.sh
shellcheck mocks/postgres.sh
```

This design ensures the testing framework remains maintainable, extensible, and focused on its core purpose: validating that resources work as expected, regardless of what those resources are.