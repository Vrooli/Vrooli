# Known Issues and Problems

## Issues Discovered During Enhancement (2025-09-28)

### 1. Dynamic Port Assignment
- **Problem**: The scenario uses dynamic port allocation that changes on each restart
- **Impact**: UI configuration needs to dynamically discover the correct API port
- **Status**: Working - UI server provides config endpoint with current ports
- **Note**: This is by design for the lifecycle system but can confuse users

### 2. Code Standards Violations  
- **Problem**: Scenario auditor reports 520 standards violations and 2 security issues
- **Impact**: Code quality metrics are below acceptable thresholds
- **Next Steps**: Need dedicated cleanup pass to address linting and security findings
- **Note**: Auditor output too large for jq processing (Argument list too long error)

### 3. Test Script Mismatch
- **Problem**: scenario-test.yaml expects `/api/health` but actual endpoint is `/health`
- **Status**: Fixed during this session
- **Resolution**: Updated scenario-test.yaml to use correct endpoint

### 4. Formatting Issues
- **Problem**: Go code had 1164+ lines needing formatting
- **Status**: Fixed during this session using gofmt
- **Resolution**: Applied standard Go formatting to all API files

## Historical Issues (Resolved)

### Go HTTP Server Hanging (Fixed)
- **Previous Issue**: Go HTTP servers were hanging when started
- **Resolution**: Issue was environmental and has been resolved
- **Current State**: API starts and responds normally

## Recommendations for Next Improvement

1. **Standards Compliance**: Address the 313 standards violations reported by scenario-auditor
2. **Security Fixes**: Investigate and fix the 2 security vulnerabilities
3. **Test Reliability**: Fix the test lifecycle to properly start services before testing
4. **Performance Testing**: Add performance benchmarks for API endpoints
5. **Integration Testing**: Add tests for scenario activation/deactivation workflows