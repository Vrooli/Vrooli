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

### Technical Details
- [Architecture](../concepts/architecture.md) - Understand the system
- [Requirements Sync](requirements-sync.md) - Debug requirement tracking issues
- [Glossary](../GLOSSARY.md) - Look up unfamiliar terms

### Reference
- [CLI Commands](../reference/cli-commands.md) - CLI reference
- [API Endpoints](../reference/api-endpoints.md) - API reference
- [Test Runners](../reference/test-runners.md) - Execution methods
