# Problems & Solutions

## Issues Discovered and Fixed (2025-10-03)

### 1. API Endpoint Returning 500 Errors
**Symptom**: `/api/v1/mcp/endpoints` returned 500 error with "Failed to scan scenarios"

**Root Cause**: The `SCENARIOS_PATH` configuration in `api/main.go` defaulted to `filepath.Join("..", "..", "..")` which was incorrect when running from the `api/` directory. This caused the detector.js path to resolve to a non-existent location.

**Solution**: Updated path resolution to use HOME environment variable with fallback:
```go
defaultScenariosPath := filepath.Join("..", "..")
if homeDir := os.Getenv("HOME"); homeDir != "" {
    defaultScenariosPath = filepath.Join(homeDir, "Vrooli", "scenarios")
}
```

**Files Modified**:
- `api/main.go:434-438`

**Validation**: `curl http://localhost:17961/api/v1/mcp/endpoints` now returns full scenario list

---

### 2. Missing Test File: detector.test.js
**Symptom**: Test suite failed with "Cannot find module '/home/matthalloran8/Vrooli/scenarios/scenario-to-mcp/lib/detector.test.js'"

**Root Cause**: The test lifecycle configuration referenced `lib/detector.test.js` but the file didn't exist.

**Solution**: Created comprehensive test file that validates:
- Scenario listing functionality
- Full scenario scanning
- Individual scenario status checks

**Files Created**:
- `lib/detector.test.js` (41 lines)

**Validation**: `node lib/detector.test.js` passes all 3 test cases

---

### 3. CLI Path Resolution Failure
**Symptom**: CLI commands failed with "cd: /home/matthalloran8/.vrooli/lib: No such file or directory"

**Root Cause**: When CLI is installed as symlink in `~/.vrooli/bin/`, the path calculation `SCENARIO_DIR="$(dirname "$SCRIPT_DIR")"` resolved to `~/.vrooli` instead of the actual scenario directory.

**Solution**: Changed path resolution to explicitly resolve parent directory:
```bash
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SCENARIO_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"
LIB_DIR="$SCENARIO_DIR/lib"
```

Also updated default API_PORT to match actual allocation (17961 vs 3290).

**Files Modified**:
- `cli/scenario-to-mcp:8-12`

**Validation**: `scenario-to-mcp list --json | jq length` returns 132

---

### 4. Missing CLI Test Suite
**Symptom**: Test failed with "Test file scenario-to-mcp.bats does not exist"

**Root Cause**: The test lifecycle referenced `cli/scenario-to-mcp.bats` but file didn't exist.

**Solution**: Created BATS test suite covering:
- Help command
- Version command
- List command with JSON output
- Detect command
- Candidates command

**Files Created**:
- `cli/scenario-to-mcp.bats` (41 lines)

**Validation**: All 5 BATS tests passing

---

## Test Coverage Summary

### Before Session
- 2 of 5 test phases passing (40%)
- API endpoints returning errors
- CLI completely broken

### After Session
- 5 of 5 test phases passing (100%)
- All API endpoints working correctly
- Full CLI functionality restored
- Comprehensive test coverage added

## Issues Fixed (2025-10-05)

### 5. Session ID Type Mismatch in Add MCP Endpoint
**Symptom**: POST `/api/v1/mcp/add` returned "Failed to create session" error

**Root Cause**: The code was generating string session IDs (`mcp-session-1234567890`) but the database schema expected UUIDs for the `agent_sessions.id` column.

**Solution**: Modified session creation to use database-generated UUIDs:
```go
var sessionID string
err := s.db.QueryRow(`
    INSERT INTO mcp.agent_sessions (scenario_name, agent_type, status, start_time)
    VALUES ($1, 'claude-code', 'pending', NOW())
    RETURNING id::text
`, req.ScenarioName).Scan(&sessionID)
```

**Files Modified**:
- `api/main.go:227-233`

**Validation**:
```bash
curl -X POST http://localhost:17961/api/v1/mcp/add \
  -H "Content-Type: application/json" \
  -d '{"scenario_name": "test-scenario", "agent_config": {"auto_detect": true}}'
# Returns: {"success":true,"agent_session_id":"f3119ae5-438d-4dc6-9cbe-7ed8586b9303",...}
```

---

### 6. CLI Symlink Path Resolution Bug
**Symptom**: When installed as symlink in `~/.vrooli/bin/`, CLI commands failed with "cd: /home/user/.vrooli/lib: No such file or directory"

**Root Cause**: The original path resolution used `${BASH_SOURCE[0]}` which points to the symlink location, not the actual script location. This caused `SCENARIO_DIR` to resolve to `~/.vrooli` instead of the actual scenario directory.

**Solution**: Added proper symlink resolution:
```bash
# Resolve symlinks to get the actual script location
SCRIPT_PATH="${BASH_SOURCE[0]}"
while [ -L "$SCRIPT_PATH" ]; do
    SCRIPT_DIR="$(cd "$(dirname "$SCRIPT_PATH")" && pwd)"
    SCRIPT_PATH="$(readlink "$SCRIPT_PATH")"
    [[ $SCRIPT_PATH != /* ]] && SCRIPT_PATH="$SCRIPT_DIR/$SCRIPT_PATH"
done
SCRIPT_DIR="$(cd "$(dirname "$SCRIPT_PATH")" && pwd)"
SCENARIO_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"
```

**Files Modified**:
- `cli/scenario-to-mcp:8-17`

**Validation**:
```bash
scenario-to-mcp list --json | jq length
# Returns: 134 (successful scan of all scenarios)
```

---

### 7. CLI Test Checking for Non-existent Command
**Symptom**: BATS test "CLI help includes all commands" failed with error on line 209

**Root Cause**: Test was checking for "check" command in help output, but the CLI doesn't have a "check" command - it has "test" instead.

**Solution**: Updated test assertion to check for correct command:
```bash
# Before: [[ "$output" =~ "check" ]]
# After:  [[ "$output" =~ "test" ]]
```

**Files Modified**:
- `cli/scenario-to-mcp.bats:209`

**Validation**: All 25 BATS tests now passing
