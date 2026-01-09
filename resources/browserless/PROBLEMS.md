# Browserless Resource - Known Problems

## Purpose
This document tracks active issues and unresolved problems with the Browserless resource. These issues are used for task generation and prioritization by the ecosystem manager.

## Active Problems

### 1. CLI Dispatch Debug Trace Order (Cosmetic)
**Severity**: Cosmetic  
**Impact**: No functional impact  
**Description**: When running browserless CLI commands with bash debug flag (-x), the command output appears before the debug trace, making it look like output precedes execution. This is purely cosmetic and doesn't affect functionality.  
**Workaround**: Run CLI commands without -x flag for cleaner output.  
**Fix Required**: None - this is expected bash behavior when output is buffered differently than stderr trace.

### 2. Session Persistence Limitation
**Severity**: Low  
**Impact**: Performance optimization limited  
**Description**: The current browserless implementation doesn't support true persistent browser sessions that can be programmatically closed and reused. The session-manager.sh has a TODO noting this limitation.  
**Workaround**: Sessions are managed at the metadata level, but actual browser contexts are recreated each time.  
**Fix Required**: Would need browserless service to add persistent session API support.

### 3. Function Execution API Interface Changed
**Severity**: Low  
**Impact**: Some advanced automation features unavailable  
**Description**: The `/function` endpoint exists but returns an HTML-based function runner interface when accessed via GET. POST requests with JavaScript code return "Not Found". This appears to be a change in the browserless API where the function execution is now handled differently than the documented REST approach.  
**Workaround**: Use JavaScript evaluation through CDP protocol or the HTML interface for manual testing.  
**Fix Required**: Either adapt to the new API format or use alternative methods for JavaScript execution.

### 4. CDP Protocol vs REST API Mismatch
**Severity**: Medium  
**Impact**: Documentation doesn't match implementation in some areas  
**Description**: Current browserless version uses Chrome DevTools Protocol (CDP) for many operations that were previously REST-based. Some documented REST endpoints like screenshot/pdf/content work, but others don't.  
**Workaround**: Most functionality works through the available endpoints.  
**Fix Required**: Update documentation to clearly indicate which endpoints are available.

### 5. Health Endpoint Clarification
**Severity**: Documentation  
**Impact**: Potential confusion about health checks  
**Description**: The browserless service doesn't provide a traditional `/health` endpoint. The `/pressure` endpoint serves as the comprehensive health and metrics endpoint, providing CPU, memory, browser pool status, and service availability.  
**Workaround**: None needed - `/pressure` endpoint provides all required health information.  
**Fix Required**: Documentation update to clarify that `/pressure` is the standard health endpoint for browserless.

### 6. CLI Subcommand Output Routing (RESOLVED 2025-09-26)
**Severity**: Low  
**Impact**: Some CLI subcommands ran but produced no visible output  
**Description**: Pool management subcommands (metrics, start, stop, recover, prewarm, etc.) executed successfully but their output was not displayed to the user due to CLI framework dispatch issues.  
**Resolution**: 
- Improved wrapper functions to properly capture and display output
- Fixed autoscaler background process to properly detach and log to file
- Added `pool logs` command to view autoscaler logs
- All pool commands now provide proper user feedback
**Improvements Made**:
- `pool start`: Now shows success message and pool stats
- `pool stop`: Displays confirmation of stop operation
- `pool prewarm`: Shows pre-warm status and updated pool stats
- `pool smart-prewarm`: Displays completion status with pool stats
- `pool recover`: Shows recovery status and pool health
- `pool logs`: New command to view autoscaler activity logs


## Resolved Problems

### 1. Integration Test False Positives (RESOLVED 2025-09-26)
**Resolution**: Updated integration tests to use stable URL (example.com) instead of httpbin.org. All tests now pass consistently.

### 2. Missing schema.json (RESOLVED 2025-01-13)
**Resolution**: Created comprehensive schema.json file with all configuration options for v2.0 contract compliance.

### 3. Health Check Timing Issues (RESOLVED 2025-01-11)
**Resolution**: Added retry logic with proper timeout handling for cold starts.

### 4. v2.0 Contract Violations (RESOLVED 2025-01-11)
**Resolution**: Fixed lib/test.sh delegation and added all required CLI commands.

### 5. Browser Pool Pre-warming (RESOLVED 2025-01-15)
**Resolution**: Implemented comprehensive pre-warming with intelligent idle detection, startup pre-warming, and CLI commands.

### 6. Workflow Result Caching (RESOLVED 2025-01-15)
**Resolution**: Added full caching system with TTL, automatic cleanup, cache statistics, and cached extraction operations.

## Future Improvements

### 1. Enhanced Error Recovery (RESOLVED 2025-01-15)
**Resolution**: Implemented `pool::health_check_and_recover` function that:
- Detects unresponsive browser pools
- Automatically restarts containers with configurable retry attempts
- Clears stuck sessions when high rejection rates detected
- Pre-warms pool after recovery for immediate readiness
- Added CLI command `resource-browserless pool recover` for manual recovery


## Testing Notes

- All core functionality tests pass consistently
- Performance benchmarks meet targets but could be optimized further
- Auto-scaling works but needs real-world load testing

## Resolved Problems (January 15, 2025)

### 6. bc Command Dependency (RESOLVED)
**Resolution**: Replaced all bc command usage with awk for better portability. The pool-manager.sh now uses awk for floating-point arithmetic instead of bc, ensuring it works on systems without bc installed.

### 7. Browser Crash Recovery (RESOLVED)
**Resolution**: Added automatic browser crash detection and recovery with health monitoring integrated into the auto-scaler loop.

### 8. Enhanced Error Handling (RESOLVED 2025-09-26)
**Resolution**: Improved error handling and user feedback throughout the resource:
- Added service availability checks before operations
- Improved error messages with helpful recovery suggestions
- Added HTTP status code handling with specific error messages
- Enhanced JavaScript execution error reporting
- Better feedback for common failure scenarios

## Maintenance History

- **2025-09-26 (Final Assessment)**: 
  - Investigated function execution API - confirmed it serves HTML interface, not REST API
  - Updated documentation to reflect actual API behavior vs documented expectations
  - Validated all test phases pass with 100% success rate
  - Confirmed all P0, P1, and P2 requirements functioning correctly
  - v2.0 contract fully compliant
  - Updated PRD version to 1.2.10
- **2025-09-26 (Final Validation)**: 
  - Enhanced test coverage by adding pool management tests to integration suite
  - Added cache functionality tests to unit test suite
  - Validated all PRD requirements are accurate and working
  - Confirmed 100% test pass rate across all test phases
  - CLI dispatch cosmetic issue noted but functionality confirmed working
- **2025-09-26 (Additional Improvements)**:
  - Enhanced test coverage further: added advanced screenshot test (integration) and pool recovery test (unit)
  - Both integration (9 tests) and unit (9 tests) suites expanded with edge cases
  - All new tests pass successfully, maintaining 100% pass rate
- **2025-09-26**: 
  - Fixed CLI pool command output routing - all pool commands now display proper output
  - Added `pool logs` command to view autoscaler activity
  - Fixed autoscaler background process to properly detach from terminal
  - Enhanced error handling with helpful recovery suggestions
  - Improved user feedback for all browser operations
- **2025-01-17**: Created comprehensive pool-manager.sh implementation with all registered functions (auto-scaling, recovery, pre-warming). Verified all test suites pass. Pool management CLI commands now functional but CLI dispatcher needs investigation for proper output routing.
- **2025-01-16**: Fixed pool CLI command dispatcher to properly handle all subcommands (recover, prewarm, smart-prewarm)
- **2025-01-15 (Session 2)**: Fixed bc dependency, added browser crash recovery, improved error handling
- **2025-01-15**: Implemented browser pool pre-warming and workflow result caching
- **2025-01-14**: Fixed shellcheck warnings, enhanced input validation and error handling
- **2025-01-13**: Added schema.json for v2.0 compliance, created PROBLEMS.md
- **2025-01-12**: Achieved 100% PRD completion with conditional workflows
- **2025-01-11**: Fixed v2.0 compliance issues, added session management
- **2025-01-10**: Initial improvements and adapter system implementation

---

**Last Updated**: 2025-09-26  
**Version**: 1.2.10  
**Maintained By**: Ecosystem Manager
