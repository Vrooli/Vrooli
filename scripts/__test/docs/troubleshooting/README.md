# 🔧 Troubleshooting Guide

Solutions to common issues with the Vrooli test framework.

## 🚨 Quick Fixes

### Tests Won't Start

**Problem:** `source: can't open fixtures/setup.bash`

**Solution:**
```bash
# Make sure you're in the right directory
cd scripts/__test

# Check that setup file exists
ls fixtures/setup.bash

# If missing, ensure you have the new test structure
ls -la fixtures/
```

**Problem:** `VROOLI_TEST_ROOT: unbound variable`

**Solution:**
```bash
# Set the test root explicitly before sourcing
export VROOLI_TEST_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
source "${VROOLI_TEST_ROOT}/fixtures/setup.bash"
```

### Permission Issues

**Problem:** `Permission denied` when running tests

**Solution:**
```bash
# Make runner scripts executable
chmod +x runners/*.sh
chmod +x integration/**/*.sh

# Check script permissions
ls -la runners/
```

**Problem:** `Cannot write to /tmp`

**Solution:**
```bash
# Check /tmp permissions
ls -ld /tmp

# Alternative: use custom temp directory
export VROOLI_ISOLATION_BASE_DIR="$HOME/tmp/vrooli-tests"
mkdir -p "$VROOLI_ISOLATION_BASE_DIR"
```

### Service Connection Errors

**Problem:** `Connection refused` in integration tests

**Solution:**
```bash
# Check if services are running
./integration/health-check.sh

# Check specific service
nc -z localhost 11434  # Replace with your service port

# Check Docker containers
docker ps

# Start required services
docker-compose up -d
```

## 🔍 Detailed Troubleshooting

### Test Discovery Issues

#### Problem: "No tests found"

**Symptoms:**
- Runner says "Found 0 test files"
- Tests exist but aren't being discovered

**Diagnosis:**
```bash
# Check file patterns
find . -name "*.bats" -type f
find integration/ -name "*.sh" -type f

# Check current directory
pwd  # Should be scripts/__test

# Check file permissions
ls -la fixtures/tests/
```

**Solutions:**
```bash
# Ensure correct directory structure
ls fixtures/tests/     # Should contain .bats files
ls integration/        # Should contain service tests

# Fix file extensions
mv test_file.bat test_file.bats  # Must be .bats, not .bat

# Fix file permissions
chmod +r fixtures/tests/*.bats
```

#### Problem: Tests found but won't execute

**Symptoms:**
- Tests are discovered but fail to start
- "Command not found" errors

**Diagnosis:**
```bash
# Check individual test file
bats fixtures/tests/example.bats

# Check for syntax errors
bash -n fixtures/tests/example.bats
```

**Solutions:**
```bash
# Fix shebang line
#!/usr/bin/env bats  # Correct
#!/bin/bash          # Wrong for BATS files

# Check test file syntax
# Each test must start with @test
@test "description" {
    # test content
}
```

### Setup and Configuration Issues

#### Problem: Setup functions not working

**Symptoms:**
- `vrooli_setup_unit_test: command not found`
- Setup completes but tests fail

**Diagnosis:**
```bash
# Check if setup.bash is being loaded
echo "Setup loaded: $VROOLI_TEST_SETUP_LOADED"

# Check available functions
declare -F | grep vrooli_setup
```

**Solutions:**
```bash
# Ensure proper source line
source "${VROOLI_TEST_ROOT}/fixtures/setup.bash"

# Not this:
load fixtures/setup  # Old BATS load syntax

# Check for conflicting environment
unset VROOLI_TEST_SETUP_LOADED
source "${VROOLI_TEST_ROOT}/fixtures/setup.bash"
```

#### Problem: Configuration not loading

**Symptoms:**
- `vrooli_config_get: command not found`
- Configuration values are wrong

**Diagnosis:**
```bash
# Check config files exist
ls config/test-config.yaml
ls config/environments/

# Test config loading manually
cd scripts/__test
source shared/config-simple.bash
vrooli_config_init
vrooli_config_get "services.ollama.port"
```

**Solutions:**
```bash
# Ensure config files are valid YAML
yaml-lint config/test-config.yaml  # If available

# Check config file permissions
chmod +r config/*.yaml

# Use simple config for BATS compatibility
source shared/config-simple.bash  # Not config-loader.bash
```

### Parallel Execution Issues

#### Problem: Tests fail in parallel but pass individually

**Symptoms:**
- Sequential execution works fine
- Parallel execution has random failures
- Port conflict errors

**Diagnosis:**
```bash
# Test sequentially first
./runners/run-unit.sh --verbose

# Then test with minimal parallelism
./runners/run-unit.sh --parallel --jobs 2
```

**Solutions:**
```bash
# Use proper test isolation
setup() {
    vrooli_setup_unit_test  # Provides isolation
}

# Use dynamic port allocation
local port=$(vrooli_port_allocate "my-service")

# Avoid shared resources
# Don't write to same files from parallel tests
local tmpfile=$(vrooli_isolation_create_file "data")
```

#### Problem: Port conflicts

**Symptoms:**
- "Address already in use" errors
- Random test failures

**Solutions:**
```bash
# Use port manager
setup() {
    vrooli_setup_service_test "my-service"
    # Port is allocated automatically
}

# Or manual allocation
local port=$(vrooli_port_allocate "service-name")
start_service_on_port "$port"
# Port released automatically on test completion

# Check port availability
vrooli_port_is_in_use 8080
```

### Cleanup and Resource Issues

#### Problem: Tests leave resources behind

**Symptoms:**
- `/tmp` fills up with test files
- Docker containers remain running
- Processes don't terminate

**Diagnosis:**
```bash
# Check current resources
./cleanup-all.sh --status

# Show isolation statistics
vrooli_isolation_stats

# Check for running test processes
ps aux | grep vrooli_test
```

**Solutions:**
```bash
# Use automatic cleanup registration
setup() {
    vrooli_setup_unit_test  # Registers cleanup automatically
}

# Register custom cleanup
vrooli_isolation_register_cleanup "my_cleanup_function"
vrooli_isolation_register_directory "/tmp/my_test_dir"
vrooli_isolation_register_process "my_daemon"

# Manual cleanup if needed
./cleanup-all.sh --aggressive
./cleanup-all.sh --full  # Includes Docker
```

#### Problem: Cleanup doesn't work

**Symptoms:**
- Resources remain after tests
- Cleanup functions not called

**Diagnosis:**
```bash
# Check if traps are set up
trap  # Should show cleanup handlers

# Check cleanup registration
vrooli_isolation_stats
```

**Solutions:**
```bash
# Ensure proper setup
setup() {
    vrooli_setup_unit_test
    vrooli_isolation_setup_traps  # Should be automatic
}

# Manual trap setup if needed
vrooli_isolation_setup_traps
```

### Mock and Assertion Issues

#### Problem: Assertions not working

**Symptoms:**
- `assert_command_success: command not found`
- Tests pass when they should fail

**Diagnosis:**
```bash
# Check if assertions are loaded
declare -F | grep assert_

# Test assertion manually
assert_equals "actual" "expected"
```

**Solutions:**
```bash
# Ensure proper setup loading
source "${VROOLI_TEST_ROOT}/fixtures/setup.bash"

# Check for function conflicts
unset assert_equals  # If overridden somewhere
source "${VROOLI_TEST_ROOT}/fixtures/setup.bash"
```

#### Problem: Mocks not working

**Symptoms:**
- Real services called instead of mocks
- Mock functions undefined

**Solutions:**
```bash
# Use service-specific setup for automatic mocking
setup() {
    vrooli_setup_service_test "ollama"  # Loads ollama mocks
}

# Or load mocks manually
source "${VROOLI_TEST_ROOT}/fixtures/mocks/docker.bash"

# Check mock loading
declare -F | grep docker  # Should show mock functions
```

## 🐛 Debug Mode

### Enable Verbose Debugging

```bash
# Enable debug mode
export VROOLI_TEST_DEBUG=true
export VROOLI_TEST_VERBOSE=true

# Run with maximum verbosity
./runners/run-unit.sh --verbose

# Debug specific test
bats --verbose --tap fixtures/tests/my_test.bats
```

### Check Environment

```bash
# Show all test environment variables
env | grep VROOLI_

# Show loaded functions
declare -F | grep vrooli_

# Show test isolation status
vrooli_isolation_stats
```

### Trace Test Execution

```bash
# Enable bash tracing
export BASH_XTRACING=true

# Or use bash -x
bash -x runners/run-unit.sh
```

## 📊 Performance Issues

### Problem: Tests are too slow

**Symptoms:**
- Tests take minutes instead of seconds
- Timeouts occur frequently

**Diagnosis:**
```bash
# Time individual components
time ./runners/run-unit.sh --list  # Just discovery
time bats fixtures/tests/simple.bats  # Individual test

# Check resource usage
htop  # During test execution
```

**Solutions:**
```bash
# Use parallel execution
./runners/run-unit.sh --parallel --jobs $(nproc)

# Filter tests during development
./runners/run-unit.sh --filter "fast"
./runners/run-changed.sh  # Only changed tests

# Use performance test setup for minimal overhead
setup() {
    vrooli_setup_performance_test
}

# Increase timeouts if needed
./runners/run-unit.sh --timeout 60
```

### Problem: High memory usage

**Solutions:**
```bash
# Use cleanup between tests
export VROOLI_TEST_AGGRESSIVE_CLEANUP=true

# Limit parallel jobs
./runners/run-unit.sh --parallel --jobs 2

# Monitor memory during tests
watch 'free -h && ps aux | grep bats | wc -l'
```

## 🔧 Environment-Specific Issues

### CI/CD Environment

**Common Issues:**
- Services not available
- Different file permissions
- Limited resources

**Solutions:**
```bash
# Skip integration tests in CI if services unavailable
if [[ "${CI:-false}" == "true" ]]; then
    ./runners/run-unit.sh  # Unit tests only
else
    ./runners/run-all.sh   # Full test suite
fi

# Use CI-specific configuration
export VROOLI_TEST_CONFIG_ENV=ci
```

### Docker Environment

**Common Issues:**
- Network connectivity
- Volume mounting
- Permission differences

**Solutions:**
```bash
# Use host networking in Docker
docker run --network host ...

# Mount test directory properly
docker run -v "$(pwd)/scripts/__test:/test" ...

# Check container permissions
docker run -it --rm -v "$(pwd):/app" ubuntu:20.04 bash
```

## 🆘 Getting Help

### Self-Diagnosis Script

Create this helper script for quick diagnosis:

```bash
#!/bin/bash
# diagnose-tests.sh

echo "=== Test Environment Diagnosis ==="
echo "Working directory: $(pwd)"
echo "Test root exists: $(test -d scripts/__test && echo "✅" || echo "❌")"
echo "Setup file exists: $(test -f scripts/__test/fixtures/setup.bash && echo "✅" || echo "❌")"
echo "Runners exist: $(test -d scripts/__test/runners && echo "✅" || echo "❌")"

echo -e "\n=== Required Tools ==="
echo "BATS: $(bats --version 2>/dev/null || echo "❌ Not installed")"
echo "curl: $(curl --version 2>/dev/null | head -1 || echo "❌ Not installed")"
echo "jq: $(jq --version 2>/dev/null || echo "❌ Not installed")"

echo -e "\n=== Test Discovery ==="
echo "Unit tests found: $(find scripts/__test/fixtures/tests -name "*.bats" 2>/dev/null | wc -l)"
echo "Integration tests found: $(find scripts/__test/integration -name "*.sh" 2>/dev/null | wc -l)"

echo -e "\n=== Configuration ==="
echo "Config file exists: $(test -f scripts/__test/config/test-config.yaml && echo "✅" || echo "❌")"
echo "Environment: ${VROOLI_TEST_CONFIG_ENV:-default}"

echo -e "\n=== Resources ==="
./scripts/__test/cleanup-all.sh --status 2>/dev/null || echo "Cleanup status unavailable"
```

### Get Support

1. **Run the diagnosis script** above
2. **Check [common patterns](../patterns/)** for examples
3. **Review [migration guide](../migration/)** if upgrading
4. **Enable debug mode** and capture output
5. **Check logs** in `/tmp/vrooli_test_*` directories

### Report Issues

When reporting issues, include:

- **Environment info:** OS, bash version, BATS version
- **Diagnosis output:** From the script above
- **Error messages:** Complete error output
- **Test file:** Minimal example that reproduces the issue
- **Expected vs actual behavior**

---

**Still stuck?** Check the [quick start guide](../quick-start.md) for basic setup, or the [API reference](../reference/) for complete function documentation!