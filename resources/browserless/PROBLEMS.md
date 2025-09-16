# Browserless Resource - Known Problems

## Purpose
This document tracks active issues and unresolved problems with the Browserless resource. These issues are used for task generation and prioritization by the ecosystem manager.

## Active Problems

### 1. Session Persistence Limitation
**Severity**: Low  
**Impact**: Performance optimization limited  
**Description**: The current browserless implementation doesn't support true persistent browser sessions that can be programmatically closed and reused. The session-manager.sh has a TODO noting this limitation.  
**Workaround**: Sessions are managed at the metadata level, but actual browser contexts are recreated each time.  
**Fix Required**: Would need browserless service to add persistent session API support.

### 2. Function Execution API Not Available
**Severity**: Low  
**Impact**: Some advanced automation features unavailable  
**Description**: The `/function` endpoint returns 404 in the current ghcr.io/browserless/chrome:latest image. This endpoint was documented but isn't available in the current version.  
**Workaround**: Use JavaScript evaluation through CDP protocol instead.  
**Fix Required**: Either update to a different browserless image or remove references to this endpoint.

### 3. CDP Protocol vs REST API Mismatch
**Severity**: Medium  
**Impact**: Documentation doesn't match implementation in some areas  
**Description**: Current browserless version uses Chrome DevTools Protocol (CDP) for many operations that were previously REST-based. Some documented REST endpoints like screenshot/pdf/content work, but others don't.  
**Workaround**: Most functionality works through the available endpoints.  
**Fix Required**: Update documentation to clearly indicate which endpoints are available.

### 4. Health Endpoint Clarification
**Severity**: Documentation  
**Impact**: Potential confusion about health checks  
**Description**: The browserless service doesn't provide a traditional `/health` endpoint. The `/pressure` endpoint serves as the comprehensive health and metrics endpoint, providing CPU, memory, browser pool status, and service availability.  
**Workaround**: None needed - `/pressure` endpoint provides all required health information.  
**Fix Required**: Documentation update to clarify that `/pressure` is the standard health endpoint for browserless.

## Resolved Problems

### 1. Missing schema.json (RESOLVED 2025-01-13)
**Resolution**: Created comprehensive schema.json file with all configuration options for v2.0 contract compliance.

### 2. Health Check Timing Issues (RESOLVED 2025-01-11)
**Resolution**: Added retry logic with proper timeout handling for cold starts.

### 3. v2.0 Contract Violations (RESOLVED 2025-01-11)
**Resolution**: Fixed lib/test.sh delegation and added all required CLI commands.

### 4. Browser Pool Pre-warming (RESOLVED 2025-01-15)
**Resolution**: Implemented comprehensive pre-warming with intelligent idle detection, startup pre-warming, and CLI commands.

### 5. Workflow Result Caching (RESOLVED 2025-01-15)
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
- Integration with n8n adapter system works but could be more robust
- Performance benchmarks meet targets but could be optimized further
- Auto-scaling works but needs real-world load testing

## Resolved Problems (January 15, 2025)

### 6. bc Command Dependency (RESOLVED)
**Resolution**: Replaced all bc command usage with awk for better portability. The pool-manager.sh now uses awk for floating-point arithmetic instead of bc, ensuring it works on systems without bc installed.

### 7. Browser Crash Recovery (RESOLVED)
**Resolution**: Added automatic browser crash detection and recovery with health monitoring integrated into the auto-scaler loop.

## Maintenance History

- **2025-01-16**: Fixed pool CLI command dispatcher to properly handle all subcommands (recover, prewarm, smart-prewarm)
- **2025-01-15 (Session 2)**: Fixed bc dependency, added browser crash recovery, improved error handling
- **2025-01-15**: Implemented browser pool pre-warming and workflow result caching
- **2025-01-14**: Fixed shellcheck warnings, enhanced input validation and error handling
- **2025-01-13**: Added schema.json for v2.0 compliance, created PROBLEMS.md
- **2025-01-12**: Achieved 100% PRD completion with conditional workflows
- **2025-01-11**: Fixed v2.0 compliance issues, added session management
- **2025-01-10**: Initial improvements and adapter system implementation

---

**Last Updated**: 2025-01-16  
**Maintained By**: Ecosystem Manager