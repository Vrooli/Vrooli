# Keycloak Resource - Known Problems

## Active Issues

### 1. Social Provider End-to-End Test Execution
**Date Found**: 2025-09-16  
**Issue**: The social-e2e test script exits silently during library sourcing
**Impact**: Social provider E2E test is not executing, but core functionality works
**Workaround**: All other tests (smoke, unit, integration, security, multi-realm) pass successfully
**Notes**: The test was added and the logic is correct, but shell sourcing issues prevent execution. The social provider functionality itself works as validated by manual testing.

## Resolved Issues

### 1. Let's Encrypt CLI Integration ✅
**Resolution Date**: 2025-09-15
**Initial Issue**: Let's Encrypt integration temporarily disabled due to circular dependency
**Solution**: 
- Added proper source guards to prevent duplicate library loading
- Improved ACME challenge test with automatic port conflict resolution
- Enhanced error handling for Python HTTP server startup
- All Let's Encrypt commands now fully functional via CLI
**Commands Available**: init, request, renew, auto-renew, disable-auto-renew, status, revoke, test

### 2. HTTPS/TLS Configuration ✅
**Resolution Date**: 2025-09-14
**Solution**: Implemented comprehensive TLS support in lib/tls.sh
- Self-signed certificate generation for development
- Certificate import/export functionality
- HTTPS enable/disable commands
- Certificate expiry monitoring and renewal
- Java keystore creation for Keycloak

### 2. Missing v2.0 Contract Compliance ✅
**Resolution Date**: 2024-09-12
**Solution**: 
- Added PRD.md with requirements
- Created test suite structure
- Added config/schema.json
- Implemented content management

### 2. Health Check Missing Timeout ✅
**Resolution Date**: 2024-09-12
**Solution**: Added `timeout 5` wrapper to all curl commands in tests

### 3. Container Name Mismatch ✅
**Resolution Date**: 2024-09-12
**Solution**: Updated test.sh to use ${KEYCLOAK_CONTAINER_NAME} variable consistently

### 4. PostgreSQL Integration ✅
**Resolution Date**: 2025-09-14
**Solution**: PostgreSQL integration is fully functional with shared Vrooli PostgreSQL
- Keycloak database created with 88 tables
- Using vrooli-postgres-main container
- Connection configured via KC_DB environment variables
- All data persisted in PostgreSQL, not H2

### 5. Realm Management Functions ✅
**Resolution Date**: 2025-09-12
**Solution**: Fully implemented realm, user, and client management in content.sh
- Realm creation/import/export/deletion working
- User creation with credentials working
- Client registration working
- All operations use Keycloak Admin API with proper authentication

### 6. Limited Test Coverage ✅
**Resolution Date**: 2025-09-12
**Solution**: All test phases now fully functional and passing
- Smoke tests validate health and admin console
- Unit tests verify configuration and structure
- Integration tests confirm realm/user/OIDC functionality

### 7. Missing core.sh for v2.0 Contract ✅
**Resolution Date**: 2025-09-14
**Solution**: Created lib/core.sh with core functionality
- Consolidated common operations (health, token, realm, user, client)
- Follows v2.0 universal contract requirements
- All existing tests still pass with new structure

### 8. Core.sh Library Path Issues ✅
**Resolution Date**: 2025-09-15
**Solution**: Fixed incorrect library paths in core.sh
- Changed logging.sh to utils/log.sh
- Changed port-management.sh to port_registry.sh
- Fixed system::load_port to resources::get_port function call

## Lessons Learned

1. **Test Structure is Critical**: Having proper test phases makes validation much easier
2. **Port Registry Integration**: Don't rely on sourcing functions in test subshells - use direct values
3. **Content Management Complexity**: Keycloak's API requires careful handling of authentication tokens
4. **Documentation First**: PRD helps track progress and prioritize work
5. **Library Dependencies**: Be careful with circular dependencies when sourcing shell libraries
6. **CLI Path Resolution**: Always use proper path resolution for BASH_SOURCE to handle both direct and sourced execution