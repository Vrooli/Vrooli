# Troubleshooting Guide

This guide provides step-by-step diagnosis and fixes for common testing issues across the Vrooli ecosystem - from test execution to Docker, networking, and CI/CD problems.

## Quick Diagnosis

### Check Test Execution Status

```bash
# Run tests and capture output
test-genie execute my-scenario --preset comprehensive 2>&1 | tee test_output.log

# Check phase results
cat coverage/phase-results/unit.json | jq '.status'

# View detailed errors
cat coverage/phase-results/unit.json | jq '.errors'
```

### Common Patterns

| Symptom | Likely Cause | Solution |
|---------|--------------|----------|
| Phase times out | Service not running | Check scenario is started |
| Tests pass locally, fail in CI | Environment differences | Verify CI environment matches |
| Requirement not tracked | Missing `[REQ:ID]` tag | Add requirement tag to test |
| Coverage shows 0% | Coverage not enabled | Check vitest/go test config |

## Docker & Container Issues

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

## Network & Connectivity Issues

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

## Environment & Configuration Issues

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
# Set missing variables
export DATABASE_URL="postgresql://localhost:5432/testdb"

# Use .env file
echo "DATABASE_URL=postgresql://localhost:5432/testdb" >> .env
```

### Path Resolution Issues

**Symptoms:**
```
No such file or directory
```

**Solutions:**
```bash
# Verify current directory
pwd
ls -la test/

# Use absolute paths
APP_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"

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

## Performance & Timeout Issues

### Tests Taking Too Long

**Symptoms:**
```
TIMEOUT: Test exceeded 120 seconds
```

**Diagnosis:**
```bash
# Profile test execution
time test-genie execute my-scenario --preset quick

# Check system resources
top
free -h
df -h
```

**Solutions:**
```bash
# Increase timeout via CLI
test-genie execute my-scenario --timeout 300s

# Run phases in parallel where possible
test-genie execute my-scenario --parallel

# Use quick preset for iteration
test-genie execute my-scenario --preset quick
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
rm -rf /tmp/test-* coverage/

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

## CI/CD & Automation Issues

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

## Go Test Troubleshooting

### Test Discovery Issues

**Symptoms:**
```
no Go files in /path/to/api
no test files
```

**Diagnosis:**
```bash
# Check if Go files exist
ls api/*.go

# Verify test files exist
ls api/*_test.go

# Check go.mod location
cat api/go.mod
```

**Solutions:**
```bash
# Ensure test files follow naming convention
# Tests must be named *_test.go

# Initialize go.mod if missing
cd api && go mod init my-scenario/api

# Download dependencies
go mod tidy
```

### Build Errors in Tests

**Symptoms:**
```
api/handler_test.go:15:2: undefined: someFunction
cannot find package "github.com/..."
```

**Diagnosis:**
```bash
# Check for compilation errors
cd api && go build ./...

# Verify imports
go mod verify
```

**Solutions:**
```bash
# Update dependencies
go mod tidy

# Clear module cache if corrupted
go clean -modcache
go mod download

# Check for circular imports
go vet ./...
```

### Coverage Below Threshold

**Symptoms:**
```
coverage: 65% of statements
FAIL: coverage below 70% threshold
```

**Diagnosis:**
```bash
# Generate detailed coverage report
cd api && go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out -o coverage.html
open coverage.html
```

**Solutions:**
1. Identify uncovered code in the HTML report
2. Add tests for critical paths
3. Consider excluding generated code:
   ```bash
   go test -coverprofile=coverage.out -coverpkg=./... ./...
   ```
4. Adjust threshold if appropriate:
   ```json
   {
     "phases": {
       "unit": {
         "coverageWarn": 75,
         "coverageError": 65
       }
     }
   }
   ```

### Race Conditions

**Symptoms:**
```
WARNING: DATA RACE
Read at 0x00c0001a0010 by goroutine 15
```

**Solutions:**
```bash
# Run with race detector
go test -race ./...

# Common fixes:
# 1. Add mutex locks
# 2. Use channels for synchronization
# 3. Use atomic operations for counters
```

### Test Timeout

**Symptoms:**
```
panic: test timed out after 30s
```

**Solutions:**
```bash
# Increase timeout for slow tests
go test -timeout 120s ./...

# For specific long tests, use subtests with their own timeout
func TestLongOperation(t *testing.T) {
    ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
    defer cancel()
    // ...
}
```

---

## Vitest/UI Test Troubleshooting

### Test Discovery Issues

**Symptoms:**
```
No tests found
vitest could not find any test files
```

**Diagnosis:**
```bash
# Check test file locations
ls ui/src/**/*.test.ts ui/src/**/*.test.tsx

# Verify vitest config
cat ui/vite.config.ts | grep -A 20 "test:"
```

**Solutions:**
```typescript
// vite.config.ts - ensure test config exists
export default defineConfig({
  test: {
    include: ['src/**/*.{test,spec}.{js,ts,tsx}'],
    environment: 'jsdom',
  },
});
```

### Module Resolution Errors

**Symptoms:**
```
Cannot find module '@/components/Button'
Failed to resolve import "@/lib/utils"
```

**Solutions:**
```bash
# Check path aliases in vite.config.ts
# Ensure tsconfig.json paths match

# Verify node_modules
rm -rf ui/node_modules ui/pnpm-lock.yaml
cd ui && pnpm install
```

### React Testing Library Issues

**Symptoms:**
```
Unable to find an element with the text: "Submit"
TestingLibraryElementError: Unable to find role="button"
```

**Solutions:**
```typescript
// Use findBy* for async elements
const button = await screen.findByRole('button', { name: /submit/i });

// Use waitFor for state changes
await waitFor(() => {
  expect(screen.getByText('Success')).toBeInTheDocument();
});

// Debug the DOM
screen.debug();
```

### Store/State Issues

**Symptoms:**
```
Cannot read property 'x' of undefined
Store not initialized
```

**Solutions:**
```typescript
// Wrap tests with providers
import { renderWithProviders } from '../test-utils';

it('renders component', () => {
  renderWithProviders(<MyComponent />);
});

// Reset store between tests
beforeEach(() => {
  useStore.setState(initialState);
});
```

### Snapshot Mismatches

**Symptoms:**
```
Snapshot `Component 1` mismatched
```

**Solutions:**
```bash
# Update snapshots if changes are intentional
cd ui && pnpm test -- -u

# Review diff carefully before updating
pnpm test -- --reporter=verbose
```

### Requirement Reporter Not Working

**Symptoms:**
- `coverage/vitest-requirements.json` not created
- Requirements not syncing after tests

**Diagnosis:**
```bash
# Check reporter configuration
grep -r "RequirementReporter" ui/vite.config.ts

# Verify requirement tags exist
grep -r "\[REQ:" ui/src/**/*.test.ts
```

**Solutions:**
```typescript
// vite.config.ts
import { RequirementReporter } from './test/reporters/RequirementReporter';

export default defineConfig({
  test: {
    reporters: ['default', new RequirementReporter()],
  },
});
```

---

## BAS Automation Troubleshooting

### Workflow Not Found

**Symptoms:**
```
workflow "login-flow" not found in playbooks
no matching playbook for phase
```

**Diagnosis:**
```bash
# List available playbooks
ls test/playbooks/*.yaml

# Check playbook naming
cat test/playbooks/registry.json
```

**Solutions:**
```bash
# Ensure playbook exists
# Naming convention: test/playbooks/<name>.yaml

# Register in registry.json
{
  "playbooks": {
    "login-flow": {
      "path": "login-flow.yaml",
      "phase": "e2e"
    }
  }
}
```

### Selector Not Found

**Symptoms:**
```
Element not found: [data-testid="login-button"]
Timeout waiting for selector
```

**Diagnosis:**
```bash
# Check if UI is running
curl -I http://localhost:${UI_PORT}

# Verify selector exists in UI code
grep -r "data-testid=\"login-button\"" ui/src/
```

**Solutions:**
1. Verify selector exists in the component
2. Wait for element to be visible:
   ```yaml
   - action: wait_for_selector
     selector: "[data-testid='login-button']"
     timeout: 10000
   ```
3. Check for dynamic loading:
   ```yaml
   - action: wait_for_network_idle
   ```

### Screenshot/Evidence Issues

**Symptoms:**
```
Failed to capture screenshot
Evidence directory not writable
```

**Solutions:**
```bash
# Create evidence directory
mkdir -p coverage/ui-smoke

# Check permissions
chmod 755 coverage

# Verify browserless is running
curl http://localhost:3000/json/version
```

### Browser Connection Failures

**Symptoms:**
```
Failed to connect to browser
WebSocket connection failed
```

**Solutions:**
```bash
# Check browserless resource
vrooli resource status browserless

# Restart browserless
vrooli resource restart browserless

# Verify connection
curl http://localhost:3000/health
```

---

## Python Test Troubleshooting

### Pytest Not Found

**Symptoms:**
```
pytest: command not found
```

**Solutions:**
```bash
# Install pytest
pip install pytest pytest-cov

# Or use pipenv/poetry
pipenv install --dev pytest
poetry add --dev pytest
```

### Import Errors

**Symptoms:**
```
ModuleNotFoundError: No module named 'mypackage'
```

**Solutions:**
```bash
# Install in development mode
pip install -e .

# Or set PYTHONPATH
export PYTHONPATH="${PYTHONPATH}:$(pwd)"

# Create pyproject.toml or setup.py for proper installation
```

### Fixture Issues

**Symptoms:**
```
fixture 'db_session' not found
```

**Solutions:**
```python
# Ensure conftest.py is in the test directory
# test/conftest.py
import pytest

@pytest.fixture
def db_session():
    # Setup
    yield session
    # Teardown
```

---

## Phase-Specific Troubleshooting

### Structure Phase Failures

| Error | Cause | Solution |
|-------|-------|----------|
| "service.json not found" | Missing config | Create `.vrooli/service.json` |
| "Invalid JSON" | Syntax error | Validate: `jq empty .vrooli/service.json` |
| "Missing required field" | Incomplete config | Check schema requirements |
| "README.md not found" | Missing docs | Create README.md in scenario root |

### Dependencies Phase Failures

| Error | Cause | Solution |
|-------|-------|----------|
| "go: not found" | Missing Go | Install Go toolchain |
| "pnpm: not found" | Missing pnpm | `npm install -g pnpm` |
| "resource X unavailable" | Resource not running | `vrooli resource start X` |

### Unit Phase Failures

| Error | Cause | Solution |
|-------|-------|----------|
| "no test files" | Missing tests | Add `*_test.go` or `*.test.ts` files |
| "compilation error" | Code issues | Fix build errors first |
| "coverage too low" | Insufficient tests | Add tests or adjust threshold |

### Integration Phase Failures

| Error | Cause | Solution |
|-------|-------|----------|
| "connection refused" | Scenario not running | `vrooli scenario start X` |
| "health check failed" | API error | Check scenario logs |
| "port discovery failed" | Service not registered | Verify service.json ports |

### E2E Phase Failures

| Error | Cause | Solution |
|-------|-------|----------|
| "browserless unavailable" | Resource down | `vrooli resource start browserless` |
| "element not found" | UI changed | Update selectors |
| "timeout exceeded" | Slow UI | Increase wait times |

### Business Phase Failures

| Error | Cause | Solution |
|-------|-------|----------|
| "workflow failed" | Business logic error | Debug the specific workflow |
| "data integrity issue" | State pollution | Add proper test isolation |

### Performance Phase Failures

| Error | Cause | Solution |
|-------|-------|----------|
| "build exceeded threshold" | Slow compilation | Profile dependencies |
| "lighthouse failed" | Chrome issue | Install chromium |
| "score below threshold" | Performance regression | Optimize or adjust threshold |

---

## Test-Genie Specific Issues

### Phase Results Not Generated

**Symptoms:**
- `coverage/phase-results/` directory is empty
- Phase status shows "skipped"

**Diagnosis:**
```bash
# Check if scenario exists in catalog
test-genie scenarios list | grep my-scenario

# Verify scenario configuration
cat scenarios/my-scenario/.vrooli/service.json | jq '.lifecycle.test'
```

**Solutions:**
```bash
# Ensure scenario is registered
test-genie scenarios refresh

# Check API is running
curl http://localhost:${TEST_GENIE_PORT}/health

# Run with verbose output
test-genie execute my-scenario --verbose
```

### Requirements Not Syncing

**Symptoms:**
- Requirements status not updating after tests
- `coverage/vitest-requirements.json` not created

**Diagnosis:**
```bash
# Check if vitest reporter is configured
grep -r "RequirementReporter" ui/vite.config.ts

# Verify requirement tags exist
grep -r "\[REQ:" ui/src/**/*.test.ts
```

**Solutions:**
```bash
# Run comprehensive preset (triggers sync)
test-genie execute my-scenario --preset comprehensive

# Force sync manually
test-genie sync my-scenario --force

# Check reporter output
cat ui/coverage/vitest-requirements.json | jq .
```

## Advanced Debugging

### Enable Comprehensive Logging

```bash
# Set debug mode
export DEBUG=1
export VERBOSE=1

# Run with tracing
test-genie execute my-scenario --verbose --trace

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

# Isolate the problem
test-genie execute my-scenario --phases unit --verbose
```

### Collect Debug Information

```bash
# Collect relevant logs
mkdir debug-info
cp test_output.log debug-info/
docker logs container-name > debug-info/docker.log 2>&1
env > debug-info/environment.txt
test-genie version > debug-info/version.txt

# Create archive
zip -r debug-info.zip debug-info/
```

## Prevention Checklist

### Before Writing Tests
- [ ] Read [Safety Guidelines](../safety/GUIDELINES.md)
- [ ] Tag tests with requirement IDs
- [ ] Set appropriate timeouts
- [ ] Plan cleanup strategy

### Before Committing
- [ ] Run quick preset locally: `test-genie execute my-scenario --preset quick`
- [ ] Check for hardcoded values (ports, paths)
- [ ] Verify all tests pass: `make test`

### Before Production
- [ ] Run comprehensive preset
- [ ] Test in CI environment
- [ ] Verify performance under load
- [ ] Check resource cleanup
- [ ] Document known limitations

## See Also

### Related Guides
- [QUICKSTART](../QUICKSTART.md) - Start with basics
- [Safety Guidelines](../safety/GUIDELINES.md) - Prevent common mistakes
- [Phased Testing](phased-testing.md) - Complete workflow
- [Performance Testing](performance-testing.md) - Performance phase details
- [Dashboard Guide](dashboard-guide.md) - UI troubleshooting
- [Custom Presets](custom-presets.md) - Preset configuration

### Technical Details
- [Architecture](../concepts/architecture.md) - Understand the system
- [Requirements Sync](../phases/business/requirements-sync.md) - Debug requirement tracking issues
- [Glossary](../GLOSSARY.md) - Look up unfamiliar terms
- [Scenario Unit Testing](../phases/unit/scenario-unit-testing.md) - Writing effective tests

### Reference
- [CLI Commands](../reference/cli-commands.md) - CLI reference
- [API Endpoints](../reference/api-endpoints.md) - API reference
- [Test Runners](../phases/unit/test-runners.md) - Execution methods
- [Phases Overview](../phases/README.md) - Phase specifications
