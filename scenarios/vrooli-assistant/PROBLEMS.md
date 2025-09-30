# Vrooli Assistant - Known Issues

## Current Problems (2025-09-30)

### 1. Hotkey Registration (P0 - Critical)
**Issue**: Global hotkey registration fails during test mode
- **Symptom**: `globalShortcut.isRegistered()` returns false even after registration
- **Root Cause**: Electron app lifecycle timing issue in test mode
- **Impact**: Cannot verify hotkey functionality automatically
- **Workaround**: Manual testing shows hotkey works in daemon mode
- **Fix Needed**: Refactor test-hotkey mode to properly wait for app ready state

### 2. Screenshot Capture (P0 - High)
**Issue**: Screenshot capture fails on Linux without proper tools
- **Symptom**: "Warning: Could not capture screenshot" message
- **Root Cause**: System missing screenshot utilities (gnome-screenshot, import)
- **Impact**: Cannot capture visual context for issues
- **Attempted Fix**: Added xwd support but convert command not available
- **Workaround**: API accepts issues without screenshots
- **Fix Needed**: Bundle electron-based screenshot solution or install ImageMagick

### 3. Port Configuration (P1 - Medium)
**Issue**: API port not consistently passed to CLI
- **Symptom**: CLI commands fail to connect to API
- **Root Cause**: Service runs on dynamic port (17835) but CLI defaults to 3250
- **Impact**: CLI commands fail unless API_PORT is explicitly set
- **Workaround**: Set API_PORT=17835 before running CLI commands
- **Fix Needed**: Read port from service registry or environment

### 4. Database Schema (P1 - Medium)
**Issue**: Schema uses VECTOR type without pgvector extension
- **Symptom**: Table creation might fail on some Postgres installations
- **Impact**: Pattern matching features won't work
- **Workaround**: Tables create without vector column
- **Fix Needed**: Conditional vector column creation or pgvector installation

### 5. Test Infrastructure (P2 - Low)
**Issue**: Legacy test format (scenario-test.yaml) still in use
- **Symptom**: Warning about minimal test infrastructure
- **Impact**: Missing comprehensive test coverage
- **Fix Needed**: Migrate to phased testing architecture

## Resolution Priority
1. Fix port configuration (quick win)
2. Bundle proper screenshot solution
3. Fix hotkey test mode
4. Upgrade test infrastructure
5. Handle database extensions properly

## Next Agent Instructions
- Start with port configuration fix - check how other scenarios handle dynamic ports
- For screenshot, consider using Electron's desktopCapturer API instead of system tools
- Test mode needs proper async handling for Electron app lifecycle