## 🗝️ Key Take-aways *(one-paragraph overview)*

A functional development environment is the foundation of productive coding. This maintenance task **starts the Vrooli development environment, monitors startup logs for errors, investigates the 5 most critical issues, and delivers a detailed report with root cause analysis and proposed fixes**. Use this task to proactively catch configuration problems, dependency conflicts, database issues, and service failures before they block development work. ([expressjs.com][1], [docs.docker.com][2], [pnpm.io][3], [winston.js.org][4], [prisma.io][5])

---

## 1 Purpose & Scope

| Field            | Value                                                                                                                |
| ---------------- | -------------------------------------------------------------------------------------------------------------------- |
| **Task ID**      | `STARTUP_ERRORS`                                                                                                     |
| **Goal**         | **Identify and analyze startup errors** in development mode, providing actionable fixes for the top 5 critical issues. |
| **Out of scope** | Production deployment issues, performance optimization (see `PERF_GENERAL`), test failures (see `TEST_QUALITY`).    |

---

## 2 Hard Rules

| Rule                                                                                                 | Why                                                                |
| ---------------------------------------------------------------------------------------------------- | ------------------------------------------------------------------ |
| **Always use extended timeouts** (5+ minutes) for startup commands.                                  | Services can take significant time to initialize. ([docs.docker.com][2]) |
| **Never run Git commands** (commit, stash, push, rebase) during maintenance.                         | Human review required before committing changes. ([CLAUDE.md][6])  |
| **Keep `.js` extensions in TypeScript imports** when making fixes.                                   | Required for Node ESM compatibility. ([CLAUDE.md][6])              |
| **Use the shared Winston logger** for all error logging investigations.                              | Ensures consistent JSON output. ([winston.js.org][4])              |
| **Stop development environment cleanly** before investigation ends.                                  | Prevents resource conflicts for subsequent development work.        |

---

## 3 Development Environment Architecture

Vrooli uses a sophisticated multi-service architecture with several startup targets:

### Available Targets
| Target | Description | Best For | Services |
|--------|-------------|----------|----------|
| `native-linux` | Direct Node.js execution | Fast iteration | PostgreSQL/Redis containers + native Node services |
| `docker-daemon` | Full containerization | Production parity | All services in containers |
| `k8s-cluster` | Kubernetes deployment | Advanced testing | Minikube with Helm charts |

### Service Dependencies
```
Environment Setup → Database Services → PostgreSQL/Redis → Server/Jobs/UI → Full Stack Ready
```

---

## 4 Startup Command Reference

### Primary Development Commands
```bash
# Start development environment (recommended)
bash scripts/main/develop.sh --target native-linux

# With specific target
bash scripts/main/develop.sh --target docker-daemon --detached no

# Root package commands
pnpm develop                    # Wrapper for scripts/main/develop.sh
```

### Package-Level Commands
```bash
# Server package (requires pre-built shared package)
cd packages/server && pnpm run start-development

# UI package 
cd packages/ui && pnpm run start-development

# Jobs package (requires server dependencies)
cd packages/jobs && pnpm run start-development
```

### Infrastructure Commands
```bash
# Database containers only
docker-compose up -d postgres redis

# Check service health
curl http://localhost:5329/healthcheck    # Server health
curl http://localhost:3000                # UI health
```

---

## 5 Error Categories & Investigation

### 5.1 Environment Configuration Errors
**Symptoms:** Missing environment variables, invalid paths, key generation failures

**Common Issues:**
- Missing `.env-dev` file
- Invalid `PROJECT_DIR` path
- Missing JWT keys (`JWT_PRIV`, `JWT_PUB`)
- Database connection strings malformed

**Investigation Commands:**
```bash
# Check environment file exists
ls -la .env-dev

# Verify critical environment variables
grep -E "^(JWT_PRIV|JWT_PUB|PROJECT_DIR|DB_|REDIS_)" .env-dev

# Test environment loading
source .env-dev && echo "PROJECT_DIR: $PROJECT_DIR, DB_HOST: $DB_HOST"
```

### 5.2 Database Connection Errors
**Symptoms:** `ECONNREFUSED`, Prisma connection failures, migration errors

**Common Issues:**
- PostgreSQL container not running
- Port conflicts (5432 already in use)
- Database credentials mismatch
- Schema migration failures

**Investigation Commands:**
```bash
# Check database container status
docker ps | grep postgres
docker logs postgres -n 50

# Test database connectivity
pg_isready -h localhost -p 5432
psql -h localhost -U vrooli_dev -d postgres -c "SELECT version();"

# Check for port conflicts  
sudo lsof -i :5432
```

### 5.3 Redis Connection Errors
**Symptoms:** Cache service failures, queue initialization errors, `ECONNREFUSED` to Redis

**Common Issues:**
- Redis container not running
- Port conflicts (6379 already in use)
- Redis authentication failures
- Memory/disk space issues

**Investigation Commands:**
```bash
# Check Redis container status
docker ps | grep redis
docker logs redis -n 50

# Test Redis connectivity
redis-cli -h localhost -p 6379 ping
docker exec -it redis redis-cli ping

# Check for port conflicts
sudo lsof -i :6379
```

### 5.4 Package Build & Dependency Errors
**Symptoms:** Module not found, TypeScript compilation failures, PNPM workspace issues

**Common Issues:**
- Shared package not built before server/jobs
- Node modules corruption
- TypeScript compilation errors
- PNPM cache issues

**Investigation Commands:**
```bash
# Check shared package build status
ls -la packages/shared/dist/

# Verify package dependencies
cd packages/server && pnpm list @vrooli/shared
cd packages/jobs && pnpm list @vrooli/shared

# Check for TypeScript errors
cd packages/server && pnpm run type-check
cd packages/shared && pnpm run type-check

# PNPM diagnostics
pnpm store status
pnpm install --frozen-lockfile
```

### 5.5 Service Port Conflicts
**Symptoms:** `EADDRINUSE`, port already in use errors, services failing to bind

**Common Issues:**
- Previous development session not cleaned up
- Other applications using required ports
- Multiple instances running simultaneously

**Investigation Commands:**
```bash
# Check all required ports
sudo lsof -i :3000    # UI port
sudo lsof -i :5329    # Server port  
sudo lsof -i :5432    # PostgreSQL port
sudo lsof -i :6379    # Redis port

# Identify what processes are using ports (safer than blind killing)
for port in 3000 5329 5432 6379; do
    pid=$(sudo lsof -ti:$port 2>/dev/null)
    if [ -n "$pid" ]; then
        echo "Port $port used by:"
        ps -p "$pid" -o pid,ppid,cmd --no-headers
    fi
done

# Stop processes gracefully (avoid pkill which can kill this process)
# Only kill clearly identified development processes
```

## 6 Step-by-Step Investigation Procedure

### Step 1: Clean Environment Setup
```bash
# Navigate to project root
cd /path/to/Vrooli

# Clean previous state safely (docker first, then check processes)
echo "🧹 Cleaning up previous development environment..."
docker-compose down 2>/dev/null || true

# Check what Node processes are actually running before killing
echo "📋 Checking for running Node processes..."
ps aux | grep -E "node.*(server|jobs)" | grep -v grep || echo "No server/jobs processes found"
ps aux | grep -E "vite.*ui" | grep -v grep || echo "No UI processes found"

# Only kill specific processes if they exist (safer than pkill)
# Kill by specific process patterns, excluding this script's process
for pid in $(ps aux | grep -E "node.*packages/(server|jobs)" | grep -v grep | awk '{print $2}'); do
    echo "🔄 Stopping Node process: $pid"
    kill -TERM "$pid" 2>/dev/null || true
done

for pid in $(ps aux | grep -E "vite.*packages/ui" | grep -v grep | awk '{print $2}'); do
    echo "🔄 Stopping Vite process: $pid"
    kill -TERM "$pid" 2>/dev/null || true
done

# Wait a moment for graceful shutdown
sleep 2

# Verify environment file exists
if [[ ! -f .env-dev ]]; then
    echo "❌ Missing .env-dev file - run setup first"
    exit 1
fi
```

### Step 2: Start Development Environment with Monitoring
```bash
# Start with log capture (use appropriate timeout: 5+ minutes)
timeout 300 bash scripts/main/develop.sh --target native-linux --detached no 2>&1 | tee startup_logs.txt

# Alternative: Start in detached mode for longer monitoring
bash scripts/main/develop.sh --target native-linux --detached yes
sleep 60  # Allow services to initialize
```

### Step 3: Collect Service Status
```bash
# Check all containers
docker ps -a --format "table {{.Names}}\t{{.Status}}\t{{.Ports}}"

# Check Node processes  
ps aux | grep -E "node.*(server|jobs|ui)" | grep -v grep

# Test service endpoints
curl -s http://localhost:5329/healthcheck || echo "❌ Server not responding"
curl -s http://localhost:3000 | head -n 5     || echo "❌ UI not responding"
```

### Step 4: Extract and Categorize Errors
```bash
# Extract errors from logs by severity
grep -i "error\|fatal\|exception\|fail" startup_logs.txt > errors_all.txt

# Critical system errors (prevents startup)
grep -iE "(ECONNREFUSED|EADDRINUSE|Cannot connect|Fatal|Exit code)" startup_logs.txt > errors_critical.txt

# Database/Redis errors
grep -iE "(postgres|redis|prisma|database|cache)" startup_logs.txt | grep -i error > errors_db.txt

# Package/dependency errors
grep -iE "(module not found|typescript|build failed|pnpm)" startup_logs.txt > errors_deps.txt

# Service binding errors
grep -iE "(port.*use|bind.*fail|address.*use)" startup_logs.txt > errors_ports.txt
```

### Step 5: Investigate Top 5 Critical Issues
For each error category, perform detailed analysis:

```bash
# 1. Root cause analysis
echo "Error: $(head -n 1 errors_critical.txt)"
echo "Context: $(grep -B 5 -A 5 "$(head -n 1 errors_critical.txt)" startup_logs.txt)"

# 2. Determine affected service
echo "Affected service: $(grep -E "(server|ui|jobs|postgres|redis)" errors_critical.txt | head -n 1)"

# 3. Check service-specific logs
docker logs postgres -n 20 2>&1 | grep -i error
docker logs redis -n 20 2>&1 | grep -i error

# 4. Identify fix category (config, dependency, resource, etc.)
```

### Step 6: Generate and Save Report
```bash
# Create the report with today's date
REPORT_DATE=$(date +%Y-%m-%d)
REPORT_FILE="/root/Vrooli/docs/scratch/startup-errors-${REPORT_DATE}.md"

echo "📝 Generating error analysis report..."
echo "💾 Report will be saved to: $REPORT_FILE"

# Generate report content using the template (shown in section 7)
# Save the completed report
```

### Step 7: Clean Shutdown
```bash
# Stop development environment cleanly
echo "🛑 Shutting down development environment..."
docker-compose down

# Stop Node processes gracefully (avoid pkill to prevent killing this process)
for pid in $(ps aux | grep -E "node.*packages/(server|jobs)" | grep -v grep | awk '{print $2}'); do
    echo "🔄 Stopping Node process: $pid"
    kill -TERM "$pid" 2>/dev/null || true
done

for pid in $(ps aux | grep -E "vite.*packages/ui" | grep -v grep | awk '{print $2}'); do
    echo "🔄 Stopping Vite process: $pid"
    kill -TERM "$pid" 2>/dev/null || true
done

# Wait for graceful shutdown
sleep 3

# Only use force kill as absolute last resort (with very specific patterns)
# Check if any stubborn processes remain
remaining=$(ps aux | grep -E "node.*packages/(server|jobs|ui)" | grep -v grep | wc -l)
if [ "$remaining" -gt 0 ]; then
    echo "⚠️  Some processes still running, force killing..."
    ps aux | grep -E "node.*packages/(server|jobs|ui)" | grep -v grep | awk '{print $2}' | xargs -r kill -9
fi

# Clean up log files
rm -f startup_logs.txt errors_*.txt
```

---

## 7 Report Template

### Report Storage Location
The error analysis report should be saved to:
```
/root/Vrooli/docs/scratch/startup-errors-YYYY-MM-DD.md
```

This keeps the report separate from the guide itself, allowing:
- Historical tracking of startup issues over time
- Clean separation between instructions and results
- Easy reference to past investigations

### Error Analysis Report
```markdown
# Startup Error Analysis Report

**Date:** [Current Date]
**Target:** native-linux
**Duration:** X minutes
**Services Checked:** Server, UI, Jobs, PostgreSQL, Redis

## Summary
- **Total Errors Found:** X
- **Critical Issues:** X  
- **Services Affected:** [List]
- **Environment Status:** [Functional/Partially Functional/Non-functional]

## Top 5 Critical Issues

### 1. [Error Category] - [Service]
**Error:** `[Exact error message]`
**Root Cause:** [Analysis of underlying issue]
**Impact:** [How this affects development]
**Proposed Fix:** 
```bash
[Specific commands to resolve]
```
**Prevention:** [How to avoid in future]

### 2. [Continue for remaining issues...]

## Quick Fixes Summary
```bash
# Fix 1: [Brief description]
[command]

# Fix 2: [Brief description] 
[command]

# [etc.]
```

## Recommendations
- [Immediate actions needed]
- [Configuration changes suggested]
- [Documentation updates needed]
```

---

## 8 Common Fix Patterns

### Environment Configuration
```bash
# Regenerate missing environment
rm .env-dev
bash scripts/main/setup.sh --target native-linux

# Fix JWT keys
openssl genrsa -out jwt_private.pem 2048
openssl rsa -in jwt_private.pem -pubout -out jwt_public.pem
```

### Database Issues
```bash
# Reset database container
docker-compose down
docker volume rm $(docker volume ls -q | grep postgres) 2>/dev/null || true
docker-compose up -d postgres

# Run migrations
cd packages/server && pnpm run prisma:migrate
```

### Package Dependencies
```bash
# Clean rebuild all packages
rm -rf node_modules packages/*/node_modules packages/*/dist
pnpm install
pnpm --filter @vrooli/shared run build
pnpm --filter @vrooli/server run build
pnpm --filter @vrooli/jobs run build
```

### Port Conflicts
```bash
# Find what's using the ports (safer approach)
echo "🔍 Checking port usage..."
sudo lsof -i :3000  # UI port
sudo lsof -i :5329  # Server port  
sudo lsof -i :5432  # PostgreSQL port
sudo lsof -i :6379  # Redis port

# Kill conflicting processes carefully (check what they are first)
for port in 3000 5329 5432 6379; do
    pid=$(sudo lsof -ti:$port)
    if [ -n "$pid" ]; then
        echo "🔄 Port $port in use by PID $pid"
        ps -p "$pid" -o pid,ppid,cmd | head -2
        # Only kill if it's clearly a development process
        if ps -p "$pid" -o cmd --no-headers | grep -qE "(node|vite|postgres|redis)"; then
            echo "🔄 Stopping process on port $port (PID: $pid)"
            sudo kill -TERM "$pid" 2>/dev/null || true
        else
            echo "⚠️  Unknown process on port $port, skipping automatic kill"
        fi
    fi
done

# Alternative: Use different ports (edit .env-dev)
# PORT_UI=3001
# PORT_API=5330
```

---

## 9 Update AI_CHECK Comment

After completing investigation, update the file comment:

```typescript
// AI_CHECK: STARTUP_ERRORS=<incrementedCount> | LAST: YYYY-MM-DD
```

Place this comment at the top of relevant files that were investigated or fixed during the process.

---

## 10 Fast Checklist

| ✔︎ | Item |
| -- | ---- |
| Environment file (`.env-dev`) exists and contains required variables |
| Database containers start successfully |
| Redis container starts successfully |
| Shared package builds without errors |
| Server starts and responds to health checks |
| UI starts and serves content |
| Jobs service initializes queue connections |
| All port conflicts resolved |
| Top 5 errors identified and analyzed |
| Report generated with actionable fixes |
| Report saved to `/docs/scratch/startup-errors-YYYY-MM-DD.md` |
| Development environment stopped cleanly |
| `AI_CHECK` comment updated |

---

## 12 Reference Links

* Express.js error handling guide ([expressjs.com][1])
* Docker Compose troubleshooting ([docs.docker.com][2])  
* PNPM workspace documentation ([pnpm.io][3])
* Winston logging best practices ([winston.js.org][4])
* Prisma connection troubleshooting ([prisma.io][5])
* Vrooli development guidelines ([CLAUDE.md][6])

---

### 🧠 Remember

**Startup errors are early warning signals:** catch configuration problems, dependency conflicts, and service failures before they impact development velocity. A 10-minute investigation can save hours of debugging later.

[1]: https://expressjs.com/en/guide/error-handling.html "Express.js Error Handling"
[2]: https://docs.docker.com/compose/troubleshooting/ "Docker Compose Troubleshooting"
[3]: https://pnpm.io/workspaces "PNPM Workspaces"
[4]: https://winston.js.org/#usage "Winston Usage Guide"
[5]: https://prisma.io/docs/reference/api-reference/environment-variables-reference "Prisma Environment Variables"
[6]: /CLAUDE.md "Vrooli Development Guidelines" 