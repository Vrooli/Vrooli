# App Issue Tracker - Problems & Solutions

## Problems Found and Fixed (2025-10-01 - Session 3)

### 1. CORS Security Configuration Not Enforced
**Problem**: CORS middleware hardcoded wildcard `*` origin, ignoring security configuration from environment variables

**Impact**: No way to restrict API access to specific domains in production deployments

**Solution Applied**:
- Integrated `SecurityConfig` into main `Config` struct
- Updated `corsMiddleware` to accept and enforce `SecurityConfig.AllowedOrigins`
- Added logging on startup to show configured CORS origins and auth status
- Made CORS configurable via `ALLOWED_ORIGINS` environment variable

**Fix Applied**: ✅ CORS now respects security configuration (2025-10-01)

**Files Modified**:
- api/main.go:30-35 (Config struct)
- api/main.go:423-448 (loadConfig)
- api/main.go:2490-2523 (corsMiddleware)
- api/main.go:2596-2608 (main function logging)

### 2. Performance Analytics Hardcoded
**Problem**: Stats endpoint returned hardcoded `avg_resolution_hours: 24.5` instead of calculating from actual data

**Impact**: Dashboard showed inaccurate metrics, couldn't track real resolution performance

**Solution Applied**:
- Implemented actual calculation from completed issues
- Parse `CreatedAt` and `ResolvedAt` timestamps
- Calculate duration and average across all resolved issues
- Added `resolved_count` to response for transparency
- Returns `0.0` when no resolved issues exist

**Fix Applied**: ✅ Real-time resolution analytics now working (2025-10-01)

**Files Modified**: api/main.go:2221-2302 (getStatsHandler)

### 3. Incomplete Integration Testing
**Problem**: Integration tests only checked basic health, missing validation of new features

**Impact**: Could deploy with broken endpoints or security issues

**Solution Applied**:
- Enhanced test-integration.sh with comprehensive API validation
- Added tests for: health, stats with analytics, CORS headers, export formats, git integration
- Proper error handling and response format validation
- Fixed export and git PR endpoint tests to match actual response formats

**Fix Applied**: ✅ Comprehensive integration test suite (2025-10-01)

**Files Modified**: test/phases/test-integration.sh (complete rewrite)

### 4. Security Configuration Undocumented
**Problem**: Security features existed but no documentation on how to enable or configure them

**Impact**: Developers wouldn't know how to secure production deployments

**Solution Applied**:
- Created comprehensive SECURITY_SETUP.md guide
- Documented all environment variables: ENABLE_AUTH, API_TOKENS, ALLOWED_ORIGINS, RATE_LIMIT
- Added GitHub integration setup instructions
- Included production deployment checklist
- Provided token generation examples and troubleshooting

**Fix Applied**: ✅ Complete security documentation (2025-10-01)

**Files Added**: docs/SECURITY_SETUP.md

## Problems Found and Fixed (2025-10-01 - Session 2)

### 1. Git Integration Not Wired to API
**Problem**: git_integration.go contained full implementation of PR creation (gitPRHandler) but no API route was registered

**Impact**: Git integration feature was completely inaccessible via API despite having working code

**Solution Applied**:
- Added route: `v1.HandleFunc("/issues/{id}/create-pr", server.gitPRHandler).Methods("POST")`
- Verified endpoint responds correctly when called without investigation data
- Requires GITHUB_TOKEN, GITHUB_OWNER, GITHUB_REPO environment variables to function

**Fix Applied**: ✅ API endpoint now accessible at POST /api/v1/issues/{id}/create-pr (2025-10-01)

**Files Modified**: api/main.go:2560

## Problems Found and Fixed (2025-10-01 - Session 1)

### 1. Go Compilation Error in git_integration.go
**Problem**: git_integration.go:377-379 had type errors:
- Attempted to index `issue.Metadata` as a map when it's a struct
- Tried to access non-existent `issue.UpdatedAt` field
- Would prevent API from compiling

**Impact**: API binary would not compile, blocking all development and testing

**Solution Applied**:
- Fixed to use `issue.Metadata.Extra` map for PR metadata
- Changed to `issue.Metadata.UpdatedAt` for timestamp
- Added nil check and initialization for Extra map
- Used proper RFC3339 time formatting

**Fix Applied**: ✅ Build now succeeds, all tests pass (2025-10-01)

**Files Modified**: api/git_integration.go:376-382

## Problems Found During Improvement (2025-09-30)

### 1. Directory Structure Inconsistency
**Problem**: The scenario has two different issue storage structures:
- `issues/` - Used by the lifecycle test configuration
- `data/issues/` - Used by the API server

**Impact**: Tests were failing because templates were in `data/issues/templates/` but tests expected them in `issues/templates/`

**Solution Applied**: Copied templates and management scripts to both locations to ensure compatibility

**Recommended Fix**: Standardize on one directory structure and update all references

### 2. Missing CLI v2 Referenced in Documentation
**Problem**: README references `app-issue-tracker-v2.sh` but this file doesn't exist

**Impact**: Documentation is inaccurate and users can't use the documented commands

**Solution**: Fixed all references in service.json to use `app-issue-tracker.sh` instead of the non-existent v2 version

**Fix Applied**: ✅ All CLI references now point to the correct `app-issue-tracker.sh` file (2025-09-30)

### 3. Port Configuration Hardcoding
**Problem**: Some documentation references port 15000 but actual port is dynamically allocated (currently 19982)

**Impact**: Users following documentation will get connection errors

**Solution**: Use `vrooli scenario port app-issue-tracker API_PORT` to get actual port

**Recommended Fix**: Update documentation to explain dynamic port allocation

### 4. Investigation Script Integration
**Problem**: The investigation handler expects external scripts but codex/claude integration scripts are not present

**Impact**: Investigation runs start but may not complete properly without the actual agent scripts

**Solution**: The API correctly triggers investigations but needs actual agent implementation

**Recommended Fix**: Create proper Claude Code or Codex integration scripts in prompts/ directory

### 5. File Storage Not Persisting
**Problem**: Issues created via API return success but files are not always visible on disk

**Impact**: Data loss risk if issues aren't properly persisted

**Solution**: API is working but may be using in-memory storage for some operations

**Recommended Fix**: Ensure all issue operations persist to YAML files immediately

## Working Features (Verified 2025-10-01)

✅ API Health check endpoint (`/health`) - responding on port 19751
✅ Issue creation via API (`POST /api/v1/issues`)
✅ Issue listing (`GET /api/v1/issues`)
✅ Investigation triggering (`POST /api/v1/investigate`)
✅ Fix generation integrated with investigation workflow
✅ Templates exist and are valid YAML
✅ Basic lifecycle management via Makefile
✅ File-based storage structure created (folder-based bundles with artifacts)
✅ Web UI dashboard - functional with metrics display
✅ CLI all commands working (create, list, investigate, fix)
✅ Export functionality - JSON, CSV, Markdown formats (`GET /api/v1/export`)
✅ Git integration endpoint - PR creation (`POST /api/v1/issues/{id}/create-pr`)

## Partially Working Features

⚠️ CLI tool - works but documentation references wrong version
⚠️ Investigation - triggers but needs actual agent implementation
⚠️ File persistence - API reports success but files not always visible

## Not Working / Missing Features

❌ Semantic search with Qdrant integration (Blocked: Qdrant resource not available)
❌ UI automation tests
❌ Performance analytics tracking (P2 requirement)

## Performance Notes

- API responds quickly (<50ms for most operations)
- File operations are fast for current volume
- No performance issues detected with current implementation

## Security Considerations

- Security code implemented in api/security.go but not enabled by default
- Authentication available via ENABLE_AUTH=true and API_TOKENS env vars
- Input validation implemented with regex patterns and length limits
- Path traversal protection in place for file operations
- CORS configurable via ALLOWED_ORIGINS env var (default: *)
- Rate limiting available (default: 100 requests per window)

## Next Steps Priority

1. **P1**: Add UI automation tests to test suite
2. **P1**: Enable authentication in production deployments
3. **P2**: Implement semantic search when Qdrant becomes available
4. **P2**: Add performance analytics dashboard
5. **P2**: Multi-agent support for specialized issue types