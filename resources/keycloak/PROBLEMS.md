# Keycloak Resource - Known Problems

## Active Issues

None currently identified. All major functionality is working.


## Resolved Issues

### 1. HTTPS/TLS Configuration ✅
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

## Lessons Learned

1. **Test Structure is Critical**: Having proper test phases makes validation much easier
2. **Port Registry Integration**: Don't rely on sourcing functions in test subshells - use direct values
3. **Content Management Complexity**: Keycloak's API requires careful handling of authentication tokens
4. **Documentation First**: PRD helps track progress and prioritize work