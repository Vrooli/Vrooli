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

## Future Improvements

### 1. Browser Pool Pre-warming Optimization
**Priority**: P2  
**Description**: Implement pre-warming of browser instances to reduce cold start latency.  
**Benefit**: Faster response times for first requests after idle periods.

### 2. Enhanced Error Recovery
**Priority**: P1  
**Description**: Add more sophisticated error recovery for browser crashes and network issues.  
**Benefit**: Better reliability under high load or unstable conditions.

### 3. Workflow Result Caching
**Priority**: P2  
**Description**: Cache workflow execution results for improved performance on repeated operations.  
**Benefit**: Significant performance improvement for repeated automation tasks.

## Testing Notes

- All core functionality tests pass consistently
- Integration with n8n adapter system works but could be more robust
- Performance benchmarks meet targets but could be optimized further
- Auto-scaling works but needs real-world load testing

## Maintenance History

- **2025-01-13**: Added schema.json for v2.0 compliance, created PROBLEMS.md
- **2025-01-12**: Achieved 100% PRD completion with conditional workflows
- **2025-01-11**: Fixed v2.0 compliance issues, added session management
- **2025-01-10**: Initial improvements and adapter system implementation

---

**Last Updated**: 2025-01-13  
**Maintained By**: Ecosystem Manager