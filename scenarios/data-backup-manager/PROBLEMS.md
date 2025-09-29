# Known Problems and Limitations

## Port Configuration Issue
**Problem**: The scenario may run on different ports than configured in service.json  
**Impact**: Tests and CLI commands may fail to connect  
**Workaround**: Set API_PORT environment variable when running CLI: `API_PORT=20852 ./cli/data-backup-manager status`  
**Solution**: The lifecycle system should ensure consistent port allocation

## MinIO Integration Not Implemented
**Problem**: MinIO backup functionality is stubbed but not actually implemented  
**Impact**: Object storage backups don't work despite being listed as P0 requirement  
**Next Steps**: Implement actual MinIO client integration in backup.go

## PostgreSQL Backup Binary Dependency
**Problem**: PostgreSQL backups require pg_dump binary to be installed system-wide  
**Impact**: Backups may fail if PostgreSQL client tools are not installed  
**Solution**: Either bundle pg_dump or use direct database export methods

## Test Suite Port Mismatch
**Problem**: Test suite uses $API_PORT which may not match actual running port  
**Impact**: `make test` fails even when service is healthy  
**Solution**: Tests should discover the actual port from running service

## Security False Positive
**Problem**: Auditor flags default password values as critical security issue  
**Impact**: Shows as security vulnerability but is actually just a default when env var not set  
**Note**: This is acceptable practice for local development defaults