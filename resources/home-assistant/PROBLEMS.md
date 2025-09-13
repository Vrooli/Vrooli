# Home Assistant Resource - Known Issues & Limitations

## Current Limitations

### 1. Port Registry Integration
**Issue**: The port registry returns a full list instead of a single port when queried.
**Workaround**: Currently using the configured default port directly rather than dynamic lookup.
**Impact**: Low - port is fixed and working correctly.

### 2. Multi-user Support
**Issue**: Multi-user support requires manual configuration through the Home Assistant web UI.
**Workaround**: After initial setup, access the web UI at http://localhost:8123 and configure users manually.
**Impact**: Medium - automation required for fully automated deployments.

### 3. File Permissions
**Issue**: Home Assistant container runs as root, creating files with root ownership.
**Solution**: Implemented backup/restore using docker exec to work within container permissions.
**Impact**: Resolved - backup/restore now works correctly.

### 4. CLI Subcommand Dispatch
**Issue**: The CLI framework's subcommand dispatch for "backup list" calls backup instead of the list subcommand.
**Workaround**: Can call the function directly or use separate commands.
**Impact**: Low - functionality works, just CLI ergonomics issue.

## Resolved Issues

### Authentication Not Enforced (RESOLVED)
- **Previous**: API was accessible without authentication
- **Fixed**: API now properly returns 401 without authentication
- **Solution**: Home Assistant correctly enforces auth once configured

### Webhook Support Missing (RESOLVED)
- **Previous**: Webhook endpoints were not tested
- **Fixed**: Webhooks tested and confirmed working
- **Solution**: Added webhook tests to integration suite

### Backup/Restore Not Implemented (RESOLVED)
- **Previous**: No backup/restore functionality
- **Fixed**: Full backup/restore capabilities added
- **Solution**: Used docker exec for proper permissions handling

## Future Improvements

1. **Automated User Setup**: Script initial admin user creation
2. **SSL/TLS Configuration**: Add automated HTTPS setup
3. **Integration Templates**: Pre-configured integrations for common devices
4. **Performance Monitoring**: Add metrics collection for resource usage
5. **Advanced Automation Templates**: Library of common automation patterns