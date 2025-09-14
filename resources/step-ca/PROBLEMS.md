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

### Audit Logging (Partially Present)
- **Current State**: Basic request logging is available in container logs
- **Enhancement Needed**: Structured audit log with certificate lifecycle events
- **Implementation Path**: Configure Step-CA logging to separate audit file

### Database Backend Support
- **Current State**: Using default embedded database
- **Enhancement Needed**: PostgreSQL backend integration for scalability
- **Implementation Path**: Add PostgreSQL configuration during initialization

### Certificate Revocation
- **Current State**: No CRL/OCSP support
- **Enhancement Needed**: Certificate revocation list distribution
- **Workaround**: Use short-lived certificates (24-48h) to minimize risk

## Best Practices

1. **Always use HTTPS**: Step-CA enforces TLS for all connections
2. **Backup root certificate**: Store root CA certificate securely
3. **Monitor certificate expiry**: Implement alerting for expiring certificates
4. **Use appropriate lifetimes**: Balance security vs operational overhead

## Testing Notes

All P0 requirements are fully functional:
- ✅ ACME protocol server operational
- ✅ Multiple authentication methods configured
- ✅ Certificate policies management working
- ✅ Health monitoring functional
- ✅ v2.0 CLI contract compliant
- ✅ Docker containerization stable

Last verified: 2025-09-14