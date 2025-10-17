# Vrooli Assistant - Known Issues

## Resolved (2025-10-03)

### ✅ Hotkey Registration Timing
**Fixed**: Electron app lifecycle issue causing crashes
- **Solution**: Added `app.isReady()` check before calling `globalShortcut.unregisterAll()`
- **Status**: No longer crashes, but hotkey registration fails in headless test environment
- **Note**: This is expected - global hotkeys require full display server access

### ✅ Port Configuration
**Fixed**: CLI now reads dynamic port from service registry
- **Solution**: Added `vrooli scenario port` lookup at CLI startup
- **Status**: CLI automatically detects correct API port

### ✅ Screenshot Capture
**Improved**: Better error handling and tool detection
- **Solution**: Added support for scrot, improved error messages, graceful degradation
- **Status**: Works when screen capture tools available, degrades gracefully when not

### ✅ Test Infrastructure
**Fixed**: Migrated to phased testing architecture
- **Solution**: Updated test structure files to match vrooli-assistant's custom structure
- **Status**: All standard test phases passing

## Current Problems (2025-10-03)

### 1. Hotkey Registration in Test Mode (P2 - Low)
**Issue**: Global hotkey registration fails in test environment
- **Symptom**: "Failed to register hotkey" in automated tests
- **Root Cause**: Electron requires full display server for global shortcuts
- **Impact**: Cannot automatically test hotkey in CI/headless environments
- **Workaround**: Hotkey works in normal daemon mode
- **Fix Needed**: Mock or skip hotkey test in headless environments

### 2. Screenshot Tools Not Bundled (P1 - Medium)
**Issue**: Screenshot capture requires system tools to be installed
- **Symptom**: "Warning: Could not capture screenshot" when tools missing
- **Root Cause**: CLI relies on scrot/gnome-screenshot/imagemagick
- **Impact**: Users must install additional tools (scrot recommended)
- **Workaround**: Issue capture works without screenshots
- **Fix Needed**: Bundle Electron-based screenshot via IPC to daemon

### 3. Database Schema (P1 - Medium)
**Issue**: Schema uses VECTOR type without pgvector extension
- **Symptom**: Table creation might fail on some Postgres installations
- **Impact**: Pattern matching features won't work
- **Workaround**: Tables create without vector column
- **Fix Needed**: Conditional vector column creation or pgvector installation

## Resolution Priority
1. Bundle Electron-based screenshot solution
2. Handle database vector extensions properly
3. Skip/mock hotkey test in headless environments

## Next Agent Instructions
- For screenshot: Implement IPC bridge from CLI to Electron daemon's desktopCapturer
- For database: Add conditional schema creation based on pgvector availability
- For hotkey test: Add environment detection and skip when headless
