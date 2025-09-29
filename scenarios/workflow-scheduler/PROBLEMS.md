# 🐛 Workflow Scheduler - Known Issues & Problems

## Critical Issues (P0)

### 1. Database Connection Failure
**Severity**: Critical  
**Impact**: API cannot start properly  
**Error**: `pq: relation "schedules" does not exist`  
**Root Cause**: Database tables are not being created from schema.sql  
**Workaround**: None currently - requires proper database initialization  
**Fix Required**: 
- Ensure populate.sh runs during setup
- Add automatic schema migration on API startup
- Verify PostgreSQL resource is running and accessible

### 2. Missing Resource Dependencies
**Severity**: Critical  
**Impact**: Scenario cannot function without PostgreSQL and Redis  
**Current State**: Resources are defined but not automatically started  
**Fix Required**:
- Add resource start commands to lifecycle setup
- Implement health checks for resource availability
- Add retry logic for resource connections

## Major Issues (P1)

### 3. API Port Configuration
**Severity**: Major  
**Impact**: API cannot start without API_PORT environment variable  
**Current Behavior**: Fails with "API_PORT environment variable is required"  
**Fix Required**:
- Service.json should provide API_PORT automatically
- Add fallback to default port if not set
- Document port configuration in README

### 4. Scheduler Component Not Starting
**Severity**: Major  
**Impact**: Cron jobs won't execute even if database is available  
**Error**: `Failed to start scheduler:failed to load schedules`  
**Fix Required**:
- Implement graceful startup when database is empty
- Add scheduler health monitoring
- Create background worker for schedule execution

### 5. CLI Tool Not Functional
**Severity**: Major  
**Impact**: Cannot manage schedules via command line  
**Current State**: CLI exists but depends on running API  
**Fix Required**:
- Add API discovery mechanism
- Implement proper error messages
- Add CLI tests to validate functionality

## Minor Issues (P2)

### 6. Missing Test Coverage
**Severity**: Minor  
**Impact**: No automated validation of functionality  
**Current State**: Test files exist but are mostly empty  
**Fix Required**:
- Add Go unit tests for API
- Add integration tests for full workflow
- Add CLI command tests

### 7. UI Server Configuration
**Severity**: Minor  
**Impact**: UI may not connect to correct API port  
**Fix Required**:
- Add dynamic API URL configuration
- Implement proper CORS handling
- Add UI health checks

### 8. Documentation Gaps
**Severity**: Minor  
**Impact**: Difficult to understand setup and usage  
**Missing**:
- Database setup instructions
- Troubleshooting guide
- API authentication details
- Example workflows

## Technical Debt

### 9. No Database Migrations
- Schema changes require manual intervention
- No version tracking for database schema
- Risk of data loss during updates

### 10. Missing Monitoring
- No metrics collection
- No performance tracking
- No alerting for failures

### 11. Security Considerations
- No authentication on API endpoints
- No rate limiting
- No input sanitization for SQL injection prevention

## Recommendations for Next Improver

1. **Priority 1**: Fix database initialization
   - Ensure schema.sql is executed on setup
   - Add connection retry logic
   - Validate tables exist before starting

2. **Priority 2**: Complete lifecycle integration
   - Ensure all components start via service.json
   - Add proper health checks
   - Implement graceful shutdown

3. **Priority 3**: Add core functionality tests
   - Create at least 3 working API tests
   - Validate schedule CRUD operations
   - Test cron execution logic

## Test Results Summary
- API compilation: ✅ Works
- API startup: ❌ Fails (database required)
- CLI installation: ⚠️ Untested
- UI server: ⚠️ Untested
- Integration tests: ❌ Fail (API not running)
- Resource dependencies: ❌ Not automatically managed

## Commands for Testing
```bash
# Start the scenario
vrooli scenario start workflow-scheduler

# Check status
vrooli scenario status workflow-scheduler

# View logs
vrooli scenario logs workflow-scheduler --step start-api

# Run tests
cd scenarios/workflow-scheduler
make test

# Manual API test (if running)
API_PORT=18090
curl http://localhost:$API_PORT/health
```