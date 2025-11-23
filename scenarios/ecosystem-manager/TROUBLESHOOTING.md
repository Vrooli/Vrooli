# Ecosystem Manager Troubleshooting

## Auto Steer Profile Selector Not Loading

### Symptoms
- Auto Steer profile dropdown is empty or doesn't load
- Profiles show up intermittently

### Diagnosis

1. **Check if tables exist:**
   ```bash
   # The API now checks on startup. Look for this log message:
   # "Auto Steer database tables verified"
   # OR an error like:
   # "Auto Steer database tables missing: table auto_steer_profiles does not exist"
   ```

2. **If tables don't exist, initialize the database:**
   ```bash
   # From the scenario root directory:
   psql -h $POSTGRES_HOST -p $POSTGRES_PORT \
        -U $POSTGRES_USER -d $POSTGRES_DB \
        -f initialization/postgres/schema.sql
   ```

3. **Check table counts:**
   The API logs table counts on startup:
   ```
   Auto Steer table counts: profiles=0, executions=0, active_states=0
   ```

4. **Verify API endpoint works:**
   ```bash
   # Should return an array (even if empty: [])
   curl http://localhost:$API_PORT/api/auto-steer/profiles
   ```

5. **Check browser console for errors:**
   Open DevTools → Console and look for:
   - Network errors (failed to fetch)
   - JavaScript errors in `AutoSteerManager.loadProfiles`

### Common Fixes

**Issue:** Database tables don't exist
```bash
# Solution: Run schema initialization
cd scenarios/ecosystem-manager
psql ... -f initialization/postgres/schema.sql
```

**Issue:** Empty profiles but no errors
```bash
# Solution: Create a profile from a template
# 1. Go to Auto Steer tab in UI
# 2. Click on a template (e.g., "Balanced Development")
# 3. Click "Use Template"
# 4. Save the profile
```

**Issue:** Intermittent loading
- This was caused by error handling issues (now fixed)
- Frontend now gracefully handles failures and returns empty array
- Check API logs for connection errors to Postgres

## Phase Updates Not Persisting

### Symptoms
- Phase advances in UI but reverts when refreshing
- `current_phase_index` stays at 0

### Diagnosis

1. **Check if execution control endpoints exist:**
   ```bash
   # These should all return proper responses:
   curl -X POST http://localhost:$API_PORT/api/auto-steer/execution/start \
     -H "Content-Type: application/json" \
     -d '{"task_id":"test","profile_id":"xxx","scenario_name":"test"}'

   curl -X POST http://localhost:$API_PORT/api/auto-steer/execution/advance \
     -H "Content-Type: application/json" \
     -d '{"task_id":"test","scenario_name":"test"}'
   ```

2. **Check database state directly:**
   ```sql
   -- View active execution states
   SELECT task_id, current_phase_index, current_phase_iteration
   FROM profile_execution_state;
   ```

### Common Fixes

**Issue:** Endpoints return 404
- Solution: Restart the API (endpoints were added recently)
- Make sure you're running the latest build

**Issue:** Execution state doesn't update
- Check API logs for database errors
- Verify `profile_execution_state` table exists
- Check that task_id matches between calls

## Running Tests

```bash
# Test all Auto Steer components
cd api
go test ./pkg/autosteer/... -v

# Test specific functionality
go test ./pkg/autosteer/ -run TestExecutionEngine_PhaseAdvancementPersistence -v
go test ./pkg/autosteer/ -run TestAutoSteerHandlers_ExecutionFlow -v
go test ./pkg/autosteer/ -run TestEnsureTablesExist -v
```

## API Endpoints Reference

### Execution Control
- `POST /api/auto-steer/execution/start` - Start a profile execution
- `POST /api/auto-steer/execution/evaluate` - Evaluate current iteration
- `POST /api/auto-steer/execution/advance` - Advance to next phase
- `GET  /api/auto-steer/execution/:taskId` - Get current execution state

### Profile Management
- `GET  /api/auto-steer/profiles` - List all profiles
- `POST /api/auto-steer/profiles` - Create new profile
- `GET  /api/auto-steer/profiles/:id` - Get profile details
- `PUT  /api/auto-steer/profiles/:id` - Update profile
- `DELETE /api/auto-steer/profiles/:id` - Delete profile

### Templates
- `GET  /api/auto-steer/templates` - Get built-in templates
