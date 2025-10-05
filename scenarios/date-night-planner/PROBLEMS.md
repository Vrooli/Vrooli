# Date Night Planner - Known Issues and Solutions

## Security Fixes Completed (2025-10-03)

### Fixed Security Vulnerabilities ✅
1. **SQL Injection Vulnerability** (FIXED 2025-10-03)
   - **Location**: api/main.go:251
   - **Issue**: String concatenation in SQL query allowed SQL injection attacks
   - **Fix**: Refactored to use parameterized placeholders with proper argument counting
   - **Validation**: All tests passing, endpoint functionality verified

2. **Hardcoded Password** (FIXED 2025-10-03)
   - **Location**: api/main.go:105
   - **Issue**: Database password hardcoded in source code
   - **Fix**: Removed hardcoded default, now requires POSTGRES_PASSWORD environment variable
   - **Validation**: API starts correctly with environment variable, logs warning if missing

## Outstanding Issues

### Performance Issues
1. **Test Hanging**: Performance and business logic tests hang without completing
   - **Cause**: Tests may be waiting for resources that aren't running
   - **Solution**: Ensure all required resources are started before tests

### Standards Violations (from audit)
1. **319 Standards Violations**
   - Code quality issues that need addressing
   - Likely formatting, error handling, or documentation issues

### Missing Features (P1 Requirements)
1. **Calendar Integration** (Checkbox marked but not implemented)
   - No actual calendar integration code found
   - Need to implement proper calendar API endpoints

2. **Social Media Integration** (Not yet implemented)
   - No social media API connections
   - Would require OAuth setup and API integrations

### Infrastructure Issues
1. **N8N Workflow Import**
   - Workflows created but not actively imported/running
   - Need to verify n8n resource is running and workflows are deployed

2. **Database Population**
   - Schema exists but no seed data
   - Need to add sample venues and date ideas

## Recommendations

### Immediate Priorities (P0)
1. Fix hanging tests by ensuring proper test setup
2. Address security vulnerabilities
3. Verify all claimed features actually work

### Next Steps (P1)
1. Implement actual calendar integration or uncheck the box
2. Fix major standards violations
3. Import and activate n8n workflows
4. Add database seed data

### Future Improvements (P2)
1. Social media integration
2. AR/VR experiences
3. Gift suggestions
4. Anniversary reminders

## Validation Commands
```bash
# Check health
curl http://localhost:19454/health

# Test suggestions
curl -X POST http://localhost:19454/api/v1/dates/suggest \
  -H 'Content-Type: application/json' \
  -d '{"couple_id":"test","date_type":"romantic","budget_max":100}'

# Test surprise mode
curl -X POST http://localhost:19454/api/v1/dates/surprise \
  -H 'Content-Type: application/json' \
  -d '{"couple_id":"test","planned_by":"partner1"}'

# Run tests
make test

# Check CLI
./cli/date-night-planner --help
```