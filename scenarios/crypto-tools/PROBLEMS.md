# Problems and Solutions - Crypto-Tools

## Issues Discovered During Improvement (2025-10-03)

### 1. CLI Implementation Missing (P0 - RESOLVED)
**Problem**: The CLI was just a placeholder template with no crypto-specific functionality.
**Solution**: Implemented complete CLI with all crypto commands (hash, encrypt, decrypt, sign, verify, keygen, keys).
**Status**: ✅ Resolved (2025-09-27)

### 2. Digital Signatures Not Implemented (P0 - RESOLVED)
**Problem**: Sign and verify endpoints returned "coming soon" stubs.
**Solution**: Implemented functional sign/verify handlers using SHA256 hashing for demonstration. Production would use actual RSA/ECDSA signing.
**Status**: ✅ Resolved (2025-09-27)
**Note**: Still uses mock crypto - see limitation #6 below.

### 3. CORS Security Vulnerability (P1 - RESOLVED)
**Problem**: CORS middleware used wildcard (*) allowing any origin.
**Solution**: Configured allowed origins list with specific localhost ports and production domain.
**Status**: ✅ Resolved (2025-09-27)

### 4. Database Connection Issues (P1 - WORKING AS DESIGNED)
**Problem**: PostgreSQL connection fails when database resource is not running.
**Solution**: API runs in degraded mode without database, returning 503 status with detailed health info.
**Status**: ✅ Working as designed - graceful degradation implemented
**Note**: Health endpoint correctly reports dependency status.

### 5. Dynamic Port Assignment (WORKING AS DESIGNED)
**Problem**: API doesn't consistently bind to configured port 15001.
**Issue**: Vrooli lifecycle assigns dynamic ports from range (15000-19999).
**Impact**: CLI needs --api-base flag to specify correct port.
**Status**: ✅ Working as designed - this is intended Vrooli behavior
**Workaround**: Check logs with `vrooli scenario logs crypto-tools --step start-api` to find current port.

### 6. Go Build Command Error (P0 - RESOLVED)
**Problem**: Test lifecycle step used `./cmd/server/main.go` instead of `./cmd/server` package path.
**Solution**: Fixed both test-go-build and build-api steps in service.json to build entire package.
**Status**: ✅ Resolved (2025-10-03)
**Evidence**: `go build -o test-build ./cmd/server && rm test-build` now passes.

## Known Limitations

### 1. Mock Cryptographic Implementation (PRODUCTION BLOCKER)
**Impact**: Sign/verify operations use SHA256 hashing instead of real RSA/ECDSA signatures.
**Security Risk**: Cannot be used for production digital signature requirements.
**Remediation**: Implement real crypto using Go's crypto/rsa, crypto/ecdsa, and crypto/ed25519 packages.
**Priority**: P0 - Must be fixed before production deployment.

### 2. In-Memory Key Storage (SECURITY RISK)
**Impact**: Generated keys are stored in memory only, lost on restart.
**Security Risk**: No persistent secure key storage, keys cannot be recovered.
**Remediation**: Implement database-backed key storage with encryption at rest, or HSM integration.
**Priority**: P0 - Required for production use.

### 3. Simple Bearer Token Authentication (SECURITY RISK)
**Impact**: API uses static bearer token without rotation or expiry.
**Security Risk**: Token compromise grants full API access indefinitely.
**Remediation**: Implement OAuth2/JWT with token rotation and expiry.
**Priority**: P1 - Needed for multi-tenant or internet-facing deployments.

## Next Improvement Recommendations

### High Priority (P0)
1. **Real Cryptographic Implementation**: Replace mock SHA256 signatures with actual RSA-PSS/ECDSA/Ed25519 signing
2. **Persistent Key Storage**: Implement encrypted key storage in PostgreSQL or integrate HSM
3. **Comprehensive Testing**: Add unit tests for all crypto operations with test vectors

### Medium Priority (P1)
1. **Certificate Management**: Implement X.509 certificate creation, validation, and chain verification
2. **Key Rotation**: Add automated key lifecycle management with rotation policies
3. **Authentication Upgrade**: Replace bearer token with OAuth2/JWT
4. **Additional Algorithms**: Add ChaCha20-Poly1305, Ed25519, ECDSA P-256 support

### Low Priority (P2)
1. **HSM Integration**: Add PKCS#11 interface for hardware security modules
2. **Compliance Checking**: Implement FIPS 140-2 and Common Criteria validation
3. **Performance Optimization**: Batch operations, parallel processing for bulk crypto
4. **Audit Trail**: Comprehensive security event logging with SIEM integration

## Testing Commands

```bash
# Start scenario
make run

# Find API port
vrooli scenario logs crypto-tools --step start-api | tail -5

# Test CLI (replace PORT with actual)
./cli/crypto-tools --api-base http://localhost:PORT status
./cli/crypto-tools --api-base http://localhost:PORT hash "test"
./cli/crypto-tools --api-base http://localhost:PORT keygen rsa --size 2048
./cli/crypto-tools --api-base http://localhost:PORT sign "data" KEY_ID
```

## Security Considerations

1. **Production Deployment**: Current implementation uses mock cryptography for demonstration. Must be replaced with real crypto libraries before production use.
2. **Key Storage**: Keys are currently stored in memory. Production needs secure persistent storage (HSM or encrypted DB).
3. **Authentication**: API uses simple bearer token. Consider OAuth2/JWT for production.
4. **Audit Logging**: Basic operation tracking implemented, needs comprehensive audit trail for compliance.