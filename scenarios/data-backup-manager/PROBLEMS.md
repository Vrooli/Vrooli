# Known Problems and Limitations

## Port Discovery in Tests (PARTIALLY RESOLVED)
**Problem**: The lifecycle system's JSON output doesn't consistently include allocated port information
**Impact**: Tests need to query actual port from `lsof` or check multiple ports
**Workaround**: Test steps now attempt port discovery via JSON with fallback to default port
**Status**: Tests pass when port discovery works, but JSON structure inconsistencies remain
**Solution**: Lifecycle system needs to consistently populate allocated_ports in JSON output

## MinIO Integration (IMPLEMENTED with Dependencies)
**Problem**: MinIO backup implementation requires `mc` (MinIO Client) CLI tool to be installed
**Impact**: MinIO backups will fail if mc tool is not available
**Status**: Implementation complete in api/backup.go:243-304, integrated in api/main.go:503-506
**Workaround**: Install mc CLI tool before using MinIO backup: `wget https://dl.min.io/client/mc/release/linux-amd64/mc && chmod +x mc && sudo mv mc /usr/local/bin/`
**Next Steps**: Consider bundling mc binary or using MinIO Go SDK

## PostgreSQL Backup Binary Dependency
**Problem**: PostgreSQL backups require pg_dump binary to be installed system-wide
**Impact**: Backups may fail if PostgreSQL client tools are not installed
**Fallback**: System attempts docker exec to postgres container if pg_dump fails
**Solution**: Either bundle pg_dump or use direct database export methods

## Dynamic Port Allocation
**Problem**: Scenarios run on dynamically allocated ports that change between restarts
**Impact**: Hardcoded port references fail; clients must discover actual port
**Workaround**: Use `vrooli scenario status <name>` to find current port, or set API_PORT env var
**Best Practice**: Always use environment variables for port configuration

## Security False Positive
**Problem**: Auditor flags default password values as critical security issue
**Impact**: Shows as security vulnerability but is actually just a default when env var not set
**Note**: This is acceptable practice for local development defaults
**Recommendation**: Always set explicit passwords via environment variables in production