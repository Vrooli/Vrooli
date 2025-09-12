# Keycloak Resource - Known Problems

## Active Issues

### 1. Missing PostgreSQL Integration  
**Severity**: Low (for development)
**Description**: Currently using H2 embedded database for development. Production should use PostgreSQL.

**Next Steps**: 
- Add PostgreSQL connection configuration
- Create migration script from H2 to PostgreSQL
- Test with shared Vrooli PostgreSQL instance

### 2. No HTTPS/TLS Configuration
**Severity**: Medium (for production)
**Description**: Currently only HTTP is configured. Production deployments need HTTPS.

**Requirements**:
- Certificate management
- TLS configuration
- HTTPS port exposure


## Resolved Issues

### 1. Missing v2.0 Contract Compliance ✅
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

### 4. Realm Management Functions ✅
**Resolution Date**: 2025-09-12
**Solution**: Fully implemented realm, user, and client management in content.sh
- Realm creation/import/export/deletion working
- User creation with credentials working
- Client registration working
- All operations use Keycloak Admin API with proper authentication

### 5. Limited Test Coverage ✅
**Resolution Date**: 2025-09-12
**Solution**: All test phases now fully functional and passing
- Smoke tests validate health and admin console
- Unit tests verify configuration and structure
- Integration tests confirm realm/user/OIDC functionality

## Lessons Learned

1. **Test Structure is Critical**: Having proper test phases makes validation much easier
2. **Port Registry Integration**: Don't rely on sourcing functions in test subshells - use direct values
3. **Content Management Complexity**: Keycloak's API requires careful handling of authentication tokens
4. **Documentation First**: PRD helps track progress and prioritize work