# Date Night Planner - Known Issues and Solutions

## Issues Discovered (2025-09-28)

### Performance Issues
1. **Test Hanging**: Performance and business logic tests hang without completing
   - **Cause**: Tests may be waiting for resources that aren't running
   - **Solution**: Ensure all required resources are started before tests

### Security Issues (from audit)
1. **2 Security Vulnerabilities Found**
   - Need to run detailed security scan to identify specific issues
   - Likely related to input validation or SQL injection protection

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