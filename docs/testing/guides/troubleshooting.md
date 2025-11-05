# Comprehensive Testing Troubleshooting Guide

This guide provides step-by-step diagnosis and fixes for common testing issues across the entire Vrooli ecosystem - from unit tests to integration testing, Docker issues, and CI/CD problems.

## 🚨 Emergency Quick Fixes

### Test Exits Immediately After First Success
```bash
# Symptom: Script exits right after first ✓ message
# Quick Fix: Replace all arithmetic expansions
sed -i 's/((tests_passed++))/tests_passed=$((tests_passed + 1))/g' test/phases/test-unit.sh
sed -i 's/((tests_failed++))/tests_failed=$((tests_failed + 1))/g' test/phases/test-unit.sh
```

### All Function Tests Fail
```bash
# Quick Fix: Check if you're sourcing libraries correctly
# Make sure config comes before libraries:
head -20 test/phases/test-unit.sh | grep -E "(source|config|lib)"
```

## 🔍 Systematic Diagnosis Process

### Step 1: Identify the Failure Pattern

Run your unit test and capture the output:
```bash
./cli.sh test unit 2>&1 | tee unit_test_output.log
```

Look for these patterns in the output:

#### Pattern A: Immediate Exit After First Test
```
[INFO] Test 1: Core lifecycle functions...
[SUCCESS] ✓ Install function exists
[ERROR] unit tests failed
```
→ **Go to [Arithmetic Expansion Issues](#arithmetic-expansion-issues)**

#### Pattern B: Function Not Found Errors
```
[ERROR] ✗ Content add function missing
[ERROR] ✗ API process file function missing
```
→ **Go to [Function Name Mismatches](#function-name-mismatches)**

#### Pattern C: Configuration Variable Errors
```
[ERROR] ✗ Port not configured
[ERROR] ✗ Base URL not configured
```
→ **Go to [Configuration Issues](#configuration-issues)**

#### Pattern D: CLI Handler Registration Failures
```
[ERROR] ✗ Install handler not registered
[ERROR] ✗ Test smoke handler not registered
```
→ **Go to [CLI Framework Issues](#cli-framework-issues)**

#### Pattern E: Library Sourcing Errors
```
./test/phases/test-unit.sh: line 32: RESOURCE_PORT: unbound variable
```
→ **Go to [Library Sourcing Issues](#library-sourcing-issues)**

## 🛠️ Detailed Fix Procedures

### Arithmetic Expansion Issues

**Cause:** Using `((variable++))` with `set -e` causes early script termination in some bash versions.

**Diagnosis:**
```bash
# Find all problematic arithmetic expansions
grep -n '((' test/phases/test-unit.sh
```

**Fix:**
```bash
# Replace ALL instances - don't miss any!
sed -i 's/((tests_passed++))/tests_passed=$((tests_passed + 1))/g' test/phases/test-unit.sh
sed -i 's/((tests_failed++))/tests_failed=$((tests_failed + 1))/g' test/phases/test-unit.sh

# For any custom counters (like size_valid++):
sed -i 's/((size_valid++))/size_valid=$((size_valid + 1))/g' test/phases/test-unit.sh
```

**Verification:**
```bash
# Should return no results:
grep -n '((' test/phases/test-unit.sh | grep -v '$((.*))' | grep -v 'if \[' | grep -v 'for \('
```

### Function Name Mismatches

**Cause:** Unit test expects function names that don't match the actual implementation.

**Diagnosis:**
```bash
# Step 1: See what functions the test expects
grep -E "test_function_exists.*\".*\"" test/phases/test-unit.sh

# Step 2: See what functions actually exist
grep -E "^[a-z_]+::[a-z_:]+\(\)" lib/*.sh

# Step 3: Compare them manually or with a script
```

**Automated Diagnosis Script:**
```bash
#!/bin/bash
echo "=== Functions Expected by Test ==="
grep -oE '"[^"]*::.*"' test/phases/test-unit.sh | sed 's/"//g' | sort

echo -e "\n=== Functions Actually Implemented ==="
grep -oE '^[a-z_]+::[a-z_:]+\(\)' lib/*.sh | sed 's/()$//' | sort

echo -e "\n=== Missing Functions ==="
comm -23 \
    <(grep -oE '"[^"]*::.*"' test/phases/test-unit.sh | sed 's/"//g' | sort) \
    <(grep -oE '^[a-z_]+::[a-z_:]+\(\)' lib/*.sh | sed 's/()$//' | sort)
```

**Fix Options:**

**Option 1: Update Test to Match Implementation**
```bash
# Example: Test expects resource::api::process_file but implementation has resource::process_document
sed -i 's/resource::api::process_file/resource::process_document/g' test/phases/test-unit.sh
```

**Option 2: Create Function Aliases** (if backward compatibility needed)
```bash
# Add to appropriate lib file:
resource::api::process_file() {
    resource::process_document "$@"
}
```

### Configuration Issues

**Cause:** Configuration variables not properly defined or sourced.

**Diagnosis:**
```bash
# Check if config file exists and has the expected variables
grep -E "^[A-Z_]+=.*" config/defaults.sh

# Check if config is sourced before it's used in test
head -30 test/phases/test-unit.sh | grep -n -E "(source.*defaults|config)"
```

**Common Fixes:**

1. **Missing export statements:**
```bash
# Wrong:
RESOURCE_PORT=8080
# Right: 
export RESOURCE_PORT=8080
```

2. **Wrong variable names:**
```bash
# Check test expectations vs actual config
grep -E "RESOURCE.*PORT" test/phases/test-unit.sh config/defaults.sh
```

3. **Config not sourced:**
```bash
# Add near top of test file:
source "${RESOURCE_DIR}/config/defaults.sh"
```

### CLI Framework Issues

**Cause:** CLI framework not properly initialized or handlers not registered.

**Diagnosis:**
```bash
# Check if CLI framework is initialized
grep -n "cli::init" cli.sh

# Check if handlers are registered after cli::init
grep -A 10 "cli::init" cli.sh | grep "CLI_COMMAND_HANDLERS"

# Check specific handler registration
grep "CLI_COMMAND_HANDLERS\[test::smoke\]" cli.sh
```

**Fix:**
```bash
# Ensure handlers are registered AFTER cli::init in cli.sh:
CLI_COMMAND_HANDLERS["test::smoke"]="resource::test::smoke"
CLI_COMMAND_HANDLERS["test::unit"]="resource::test::unit"
CLI_COMMAND_HANDLERS["manage::install"]="resource::install"
```

### Library Sourcing Issues

**Cause:** Libraries sourced in wrong order or with wrong paths.

**Diagnosis:**
```bash
# Check sourcing order in test
head -40 test/phases/test-unit.sh | grep -E "source.*\.(sh|bash)"

# Check if paths are correct
ls -la lib/*.sh config/*.sh
```

**Fix:**
```bash
# Correct order (config MUST come before libraries):
source "${RESOURCE_DIR}/config/defaults.sh"
source "${RESOURCE_DIR}/lib/core.sh"
source "${RESOURCE_DIR}/lib/install.sh"
# ... other libs

# Handle missing libraries gracefully:
for lib in core install status api content; do
    lib_file="${RESOURCE_DIR}/lib/${lib}.sh"
    [[ -f "$lib_file" ]] && source "$lib_file" 2>/dev/null || true
done
```

## 🧪 Test Your Fix

After applying any fix, verify it works:

```bash
# Run unit test
./cli.sh test unit

# Should see:
# - No immediate exits
# - Proper test counts
# - Clear pass/fail summary
```

## 📋 Prevention Checklist

Use this checklist when writing new unit tests:

### Before Writing Tests:
- [ ] List all functions that actually exist: `grep -r "^function_name()" lib/`
- [ ] Check configuration variables: `grep "^export" config/defaults.sh`
- [ ] Verify CLI handler registration: `grep "CLI_COMMAND_HANDLERS" cli.sh`

### While Writing Tests:
- [ ] Use safe arithmetic: `var=$((var + 1))` not `((var++))`
- [ ] Test function existence with: `declare -f function_name`
- [ ] Source config before libraries
- [ ] Handle missing libraries gracefully
- [ ] Use consistent naming patterns

### After Writing Tests:
- [ ] Run test multiple times to ensure consistency
- [ ] Check test completes in <60 seconds
- [ ] Verify all expected functions are tested
- [ ] Ensure proper exit codes (0 for success, 1 for failure)

## 🔧 Debug Tools

### Enable Debug Mode
```bash
# Run test with full debug output:
bash -x test/phases/test-unit.sh 2>&1 | tee debug.log

# Or set debug in the test itself:
set -x  # Add after set -euo pipefail
```

### Function Existence Checker
```bash
# Quick script to check if all expected functions exist:
check_functions() {
    local test_file="$1"
    while IFS= read -r func; do
        if declare -f "$func" &>/dev/null; then
            echo "✓ $func"
        else
            echo "✗ $func"
        fi
    done < <(grep -oE '"[^"]*::[^"]*"' "$test_file" | sed 's/"//g')
}

# Usage:
check_functions test/phases/test-unit.sh
```

### Variable Checker
```bash
# Check if all expected variables are defined:
check_vars() {
    local test_file="$1"
    while IFS= read -r var; do
        if [[ -n "${!var:-}" ]]; then
            echo "✓ $var = ${!var}"
        else
            echo "✗ $var (undefined)"
        fi
    done < <(grep -oE '\$\{[A-Z_]+[:-]' "$test_file" | sed 's/\${\(.*\)[:-].*/\1/' | sort -u)
}

# Usage (run from resource directory after sourcing config):
source config/defaults.sh
check_vars test/phases/test-unit.sh
```

## ❓ Still Stuck?

If none of these solutions work:

1. **Compare with a working resource:**
   ```bash
   # Find resources with passing unit tests:
   find /path/to/vrooli/resources -name "test-unit.sh" -exec bash -c 'echo "Testing {}"; timeout 10 bash "{}" && echo "✓ PASSED" || echo "✗ FAILED"' \;
   ```

2. **Start with a minimal test:**
   ```bash
   # Create a minimal test that just counts:
   tests_passed=0
   tests_failed=0
   
   tests_passed=$((tests_passed + 1))
   echo "Passed: $tests_passed"
   ```

3. **Enable maximum debugging:**
   ```bash
   set -euxo pipefail  # Add 'x' for trace mode
   ```

4. **Check resource-specific documentation** in the resource's README or PRD files.

Remember: Unit test failures are almost always implementation bugs, not framework limitations!

## 🐳 Docker & Container Issues

### Container Won't Start
**Symptoms:**
```
Error starting userland proxy: listen tcp 0.0.0.0:8080: bind: address already in use
```

**Diagnosis:**
```bash
# Check what's using the port
lsof -i :8080
netstat -tlnp | grep :8080

# Check if container already exists
docker ps -a | grep container-name
```

**Solutions:**
```bash
# Kill process using the port
sudo kill -9 $(lsof -t -i:8080)

# Use different port
export RESOURCE_PORT=8081

# Remove existing container
docker rm -f container-name
```

### Permission Denied in Container
**Symptoms:**
```
Permission denied: /var/lib/resource/data
mkdir: cannot create directory '/app/data': Permission denied
```

**Solutions:**
```bash
# Fix ownership
docker exec -u root container-name chown -R app:app /app/data

# Run with proper user
docker run --user $(id -u):$(id -g) ...

# Add to docker-compose.yml
user: "${UID:-1000}:${GID:-1000}"
```

### Docker Build Failures
**Symptoms:**
```
ERROR [stage 2/5] COPY . /app/
```

**Solutions:**
```bash
# Check .dockerignore
cat .dockerignore

# Clean Docker cache
docker system prune -a

# Build with no cache
docker build --no-cache -t image-name .
```

## 🌐 Network & Connectivity Issues

### API Not Responding
**Symptoms:**
```
curl: (7) Failed to connect to localhost port 8080: Connection refused
```

**Diagnosis:**
```bash
# Check if service is running
curl -I http://localhost:8080/health

# Check port binding
docker port container-name

# Check container logs
docker logs container-name --tail 50
```

**Solutions:**
```bash
# Wait for service to be ready
timeout 30 bash -c 'until curl -f http://localhost:8080/health; do sleep 1; done'

# Check correct port mapping
docker run -p 8080:8080 image-name

# Verify service configuration
grep -E "port|PORT" .env config/*.sh
```

### DNS Resolution Issues
**Symptoms:**
```
getaddrinfo: Name or service not known
```

**Solutions:**
```bash
# Use IP addresses instead of hostnames
export DATABASE_URL="postgresql://user:pass@127.0.0.1:5432/db"

# Check /etc/hosts
echo "127.0.0.1 service-name" | sudo tee -a /etc/hosts

# Use Docker network
docker network create vrooli-network
docker run --network vrooli-network ...
```

### Firewall/Security Issues
**Symptoms:**
```
Connection timed out
curl: (28) Operation timed out
```

**Solutions:**
```bash
# Check firewall status
sudo ufw status

# Allow specific port
sudo ufw allow 8080

# Check iptables
sudo iptables -L

# Disable firewall temporarily (development only)
sudo ufw disable
```

## 🔧 Environment & Configuration Issues

### Environment Variables Not Set
**Symptoms:**
```
Error: DATABASE_URL is required but not set
```

**Diagnosis:**
```bash
# Check current environment
env | grep -E "DATABASE|REDIS|API"

# Check .env files
cat .env .env.local 2>/dev/null

# Check shell exports
declare -x | grep -E "DATABASE|REDIS|API"
```

**Solutions:**
```bash
# Source configuration
source config/defaults.sh

# Set missing variables
export DATABASE_URL="postgresql://localhost:5432/testdb"

# Use .env file
echo "DATABASE_URL=postgresql://localhost:5432/testdb" >> .env

# Check variable loading order
grep -n "source\|export" scripts/setup.sh
```

### Path Resolution Issues
**Symptoms:**
```
./test/run-tests.sh: No such file or directory
scripts/lib/utils/log.sh: No such file or directory
```

**Solutions:**
```bash
# Verify current directory
pwd
ls -la test/

# Use absolute paths
APP_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
source "$APP_ROOT/scripts/lib/utils/log.sh"

# Check script permissions
chmod +x test/run-tests.sh

# Verify symlinks
ls -la test/ | grep -E "^l"
```

### Version Conflicts
**Symptoms:**
```
Node version v14.x is not supported
Python 3.6 is too old
```

**Solutions:**
```bash
# Check versions
node --version
python3 --version
go version

# Use Docker for consistent environment
docker run --rm node:18 node --version

# Update using version managers
nvm use 18
pyenv global 3.9.0

# Check .nvmrc or .python-version files
cat .nvmrc .python-version 2>/dev/null
```

## ⚡ Performance & Timeout Issues

### Tests Taking Too Long
**Symptoms:**
```
TIMEOUT: Test exceeded 120 seconds
```

**Diagnosis:**
```bash
# Profile test execution
time ./test/run-tests.sh

# Find slow tests
bats --timing test/cli/*.bats

# Check system resources
top
free -h
df -h
```

**Solutions:**
```bash
# Increase timeout
export TEST_TIMEOUT=300

# Run tests in parallel
bats --jobs 4 test/cli/*.bats

# Skip slow tests
export SKIP_SLOW_TESTS=true

# Use test caching
export ENABLE_TEST_CACHE=true
```

### Memory Issues
**Symptoms:**
```
OOMKilled
Cannot allocate memory
```

**Solutions:**
```bash
# Check memory usage
docker stats

# Increase Docker memory limit
docker run -m 2g image-name

# Clean up test data
rm -rf /tmp/test-* test/fixtures/temp/

# Monitor memory in tests
ps aux --sort=-%mem | head -10
```

### Resource Leaks
**Symptoms:**
```
Too many open files
Address already in use (after tests)
```

**Solutions:**
```bash
# Check open files
lsof | wc -l
ulimit -n

# Proper cleanup in tests
trap 'cleanup_resources' EXIT

# Kill orphaned processes
pkill -f test-process-name

# Clean up containers
docker ps -q | xargs -r docker rm -f
```

## 🔄 CI/CD & Automation Issues

### GitHub Actions Failures
**Symptoms:**
```
Error: Resource not available
Error: Docker daemon not available
```

**Solutions:**
```yaml
# Use services for databases
services:
  postgres:
    image: postgres:13
    env:
      POSTGRES_PASSWORD: postgres

# Wait for services
- name: Wait for PostgreSQL
  run: |
    timeout 30 bash -c 'until pg_isready; do sleep 1; done'

# Use setup-* actions
- uses: actions/setup-node@v3
  with:
    node-version: '18'
```

### Build Cache Issues
**Symptoms:**
```
Package not found
Cache miss on dependencies
```

**Solutions:**
```yaml
# Cache dependencies
- uses: actions/cache@v3
  with:
    path: ~/.npm
    key: ${{ runner.os }}-node-${{ hashFiles('**/package-lock.json') }}

# Clear cache manually
- run: npm cache clean --force

# Disable cache temporarily
- run: npm ci --no-cache
```

### Secrets & Environment
**Symptoms:**
```
Error: API_KEY is not set
Unauthorized: Invalid credentials
```

**Solutions:**
```yaml
# Set repository secrets
env:
  API_KEY: ${{ secrets.API_KEY }}

# Use default values for testing
env:
  API_KEY: ${{ secrets.API_KEY || 'test-key' }}

# Conditional steps
if: ${{ secrets.API_KEY != '' }}
```

## 🔍 Advanced Debugging Techniques

### Enable Comprehensive Logging
```bash
# Set debug mode
export DEBUG=1
export VERBOSE=1

# Shell tracing
set -x

# Log everything
exec > >(tee -a test-debug.log) 2>&1

# Docker logging
docker logs container-name -f --since 5m
```

### Reproduce Issues Locally
```bash
# Match CI environment
export CI=true
export NODE_ENV=test

# Use same Docker images
docker run --rm -it -v "$(pwd):/app" node:18-alpine /bin/bash

# Run single test
bats test/cli/specific-test.bats --verbose

# Isolate the problem
bash -x test/phases/test-unit.sh
```

### Get Help from Logs
```bash
# Collect relevant logs
mkdir debug-info
cp test-debug.log debug-info/
docker logs container-name > debug-info/docker.log
env > debug-info/environment.txt

# Create issue with debug info
zip -r debug-info.zip debug-info/
```

## 📋 Prevention Checklist

### Before Writing Tests
- [ ] Read [Safety Guidelines](../safety/GUIDELINES.md)
- [ ] Use safe BATS templates
- [ ] Set proper timeouts
- [ ] Plan cleanup strategy

### Before Committing
- [ ] Run safety linter: `scripts/scenarios/testing/lint-tests.sh`
- [ ] Test locally with clean environment
- [ ] Check for hardcoded values (ports, paths)
- [ ] Verify all tests pass: `make test`

### Before Production
- [ ] Test in CI environment
- [ ] Verify performance under load
- [ ] Check resource cleanup
- [ ] Document known limitations

## 🆘 When All Else Fails

1. **Start with minimal reproduction**:
   ```bash
   # Create smallest possible test
   echo "#!/bin/bash\necho 'test'" > minimal-test.sh
   chmod +x minimal-test.sh && ./minimal-test.sh
   ```

2. **Compare with working examples**:
   - Check [visited-tracker tests](../../../scenarios/visited-tracker/test/)
   - Use [gold standard examples](../reference/examples.md)

3. **Get help**:
   - Create GitHub issue with "TESTING" label
   - Include debug logs and environment info
   - Mention what you've already tried

Remember: Most testing issues are environment/configuration problems, not fundamental framework issues!

## See Also

### Related Guides
- [Quick Start Guide](quick-start.md) - Start with basics
- [Safety Guidelines](../safety/GUIDELINES.md) - Prevent common mistakes
- [Scenario Testing Guide](scenario-testing.md) - Complete workflow

### Technical Details
- [Phased Testing Architecture](../architecture/PHASED_TESTING.md) - Understand phase system
- [Requirement Flow](../architecture/REQUIREMENT_FLOW.md) - Debug requirement tracking issues
- [Testing Glossary](../GLOSSARY.md) - Look up unfamiliar terms

### Common Resources
- [Shell Libraries Reference](../reference/shell-libraries.md) - Helper function docs
- [Test Runners Reference](../reference/test-runners.md) - Execution methods
- [Gold Standard Examples](../reference/examples.md) - Learn from working code