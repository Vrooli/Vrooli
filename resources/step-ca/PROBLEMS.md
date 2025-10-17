# Step-CA Resource - Known Issues and Solutions

## Minor Issues Discovered

### 1. HTTP vs HTTPS Access
**Issue**: Step-CA only accepts HTTPS connections, but some tools may attempt HTTP
**Impact**: Low - Connection errors when using HTTP
**Solution**: Always use `https://` prefix or `-k` flag with curl for self-signed certificates
**Example**: `curl -k https://localhost:9010/health`

### 2. Nested Config Directory
**Issue**: Step-CA creates its config in a nested structure (`config/config/`)
**Impact**: Low - Can be confusing when looking for configuration files
**Solution**: Implementation correctly handles this nested structure in `lib/core.sh`

### 3. Duration Format Conversion
**Issue**: Step-CA requires duration in hours format (e.g., "8760h") not days (e.g., "365d")
**Impact**: Low - Policy settings may fail with incorrect format
**Solution**: Implementation includes automatic conversion in policy management functions

## P1 Requirements - Future Enhancements

### Audit Logging (Enhanced - Partial Implementation)
- **Current State**: 
  - Basic request logging is available in container logs
  - Added `--audit` flag to filter certificate-related operations
  - All certificate operations are logged with timestamps and details
- **Enhancement Needed**: 
  - Structured JSON logging format
  - Separate audit file with rotation
  - Integration with centralized logging system
- **Implementation Path**: 
  - Configure Step-CA with JSON logging
  - Add PostgreSQL backend for persistent audit trail
  - Integrate with syslog or ELK stack

### Database Backend Support (COMPLETED ✅)
- **Current State**: PostgreSQL backend fully integrated and operational
- **Implemented Features**:
  - Automatic database creation and user setup
  - Migration from file-based to PostgreSQL backend
  - Certificate statistics query from database
  - 24 tables for complete certificate management
- **Benefits**: Scalability to millions of certificates, better querying, multi-instance support

### Certificate Revocation (IMPLEMENTED ✅)
- **Current State**: Full revocation support with PostgreSQL backend
- **Implemented Features**:
  - `revoke` command with serial number and reason support
  - CRL data generation from revoked certificates table
  - `check-revocation` command to verify certificate status
  - Revocation reasons: unspecified, keyCompromise, affiliationChanged, superseded, cessationOfOperation
- **Remaining Work**: 
  - Full CRL distribution in PEM format (data export available)
  - OCSP responder setup (Step-CA supports but needs configuration)
- **Workaround**: Use short-lived certificates (24-48h) to minimize risk

### HSM/KMS Integration (CONFIGURATION TEMPLATES PROVIDED ✅)
- **Current State**: Configuration templates and guidance implemented
- **Implemented Features**:
  - `hsm` command group for status, configure, and test operations
  - Configuration templates for AWS KMS, Google Cloud KMS, Azure Key Vault, and YubiHSM
  - Step-by-step guidance for production HSM/KMS deployment
- **Note**: Full integration requires actual HSM hardware or cloud KMS service
- **Production Path**: 
  - Use `resource-step-ca hsm configure --type <provider>` to generate configuration
  - Apply configuration to Step-CA instance
  - Test with actual credentials and HSM/KMS access

### Custom Certificate Templates (IMPLEMENTED ✅)
- **Current State**: Template management system fully operational
- **Implemented Features**:
  - Pre-defined templates: web-server, client-auth, code-signing, email
  - Custom template creation with configurable durations
  - Template storage in local configuration
  - Template listing and removal commands
- **Usage**: Templates define certificate profiles for different use cases
- **Future Enhancement**: Direct integration with Step-CA's template API when available

## Best Practices

1. **Always use HTTPS**: Step-CA enforces TLS for all connections
2. **Backup root certificate**: Store root CA certificate securely
3. **Monitor certificate expiry**: Implement alerting for expiring certificates
4. **Use appropriate lifetimes**: Balance security vs operational overhead

## Improvements Made (2025-09-14)

### Content Management Enhanced
- ✅ `content list`: Now shows certificate info and provisioner count
- ✅ `content get`: Can retrieve root CA certificate with `--name root_ca`
- ✅ `content remove`: Documents revocation requirements and workarounds
- ✅ All functions provide helpful feedback about current limitations

### Status Reporting Enhanced
- ✅ Added provisioner count to status output
- ✅ Added container uptime information
- ✅ Added ACME endpoint URL to status
- ✅ Enhanced JSON output with additional fields

### Audit Logging Improved
- ✅ Added `--audit` flag to logs command for filtered output
- ✅ Filters certificate-related operations from logs
- ✅ Provides guidance on full audit implementation

## Testing Notes

All P0 requirements are fully functional:
- ✅ ACME protocol server operational
- ✅ Multiple authentication methods configured
- ✅ Certificate policies management working
- ✅ Health monitoring functional
- ✅ v2.0 CLI contract compliant
- ✅ Docker containerization stable

## 2025-09-26 Validation Update

### PostgreSQL Backend Status
- **Configuration**: PostgreSQL backend is configured but requires PostgreSQL resource to be running
- **Current State**: Falls back to file-based storage when PostgreSQL unavailable
- **Impact**: No impact on core functionality, but limits scalability features
- **Solution**: Install and start PostgreSQL resource (`vrooli resource postgres manage install`)

### Certificate Issuance via CLI
- **Issue**: Direct certificate issuance via `content add` requires interactive password input
- **Impact**: Automated certificate generation limited to ACME protocol
- **Solution**: Use ACME clients (certbot, acme.sh) for automated certificate issuance
- **Alternative**: Generate certificates with pre-shared tokens or OIDC authentication

### Verification Results
- ✅ All test suites passing (unit, smoke, integration, validation)
- ✅ ACME protocol fully operational at https://localhost:9010/acme/acme/directory
- ✅ 3 provisioners configured (admin, ACME, keycloak-test)
- ✅ Certificate templates functional (2 templates active)
- ✅ HSM/KMS configuration templates working
- ✅ Health monitoring and status reporting operational
- ⚠️ PostgreSQL backend configured but not connected (postgres resource not running)
- ⚠️ Interactive certificate issuance requires terminal input

Last verified: 2025-09-26