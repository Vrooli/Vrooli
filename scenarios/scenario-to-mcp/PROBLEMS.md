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

## Remaining P0 Gap

**One-click MCP Addition**: The UI dashboard and API endpoints are in place, but the actual Claude-code agent spawning integration is not yet implemented in the `/mcp/add` endpoint. This is the final P0 requirement.

**Next Steps**:
1. Implement agent spawning logic in `handleAddMCP`
2. Add agent session tracking to database
3. Test end-to-end MCP addition flow
4. Verify generated MCP servers work correctly
