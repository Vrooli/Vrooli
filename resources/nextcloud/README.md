# Nextcloud Resource

Self-hosted file sync, share, and collaboration platform providing complete collaboration infrastructure with full data sovereignty.

## Overview

Nextcloud is a comprehensive collaboration platform that combines file storage, document editing, communication, and groupware capabilities in a single, self-hosted solution. It provides scenarios with secure file management, real-time collaboration, and privacy-compliant data handling without relying on third-party services.

## Features

- **File Storage & Sync**: WebDAV-compliant file storage with automatic synchronization
- **Document Collaboration**: Share and collaborate on documents with version control
- **Office Suite Integration**: Built-in Collabora Office for editing Word, Excel, and PowerPoint files
- **Sharing & Permissions**: Granular sharing controls with password protection and expiration
- **External Storage**: Mount S3, FTP, SMB/CIFS, and other storage backends (S3 implemented)
- **Security**: Secure password storage, brute force protection, and session management
- **API Access**: WebDAV, OCS, and REST APIs for programmatic access
- **User Management**: CLI-based user creation and management via OCC commands
- **Backup & Restore**: Automated backup and restore functionality for data protection

## Quick Start

```bash
# Install and start Nextcloud
vrooli resource nextcloud manage install
vrooli resource nextcloud manage start --wait

# Check status
vrooli resource nextcloud status

# Upload a file
vrooli resource nextcloud content add --file ./document.pdf

# List files
vrooli resource nextcloud content list

# Share a file
vrooli resource nextcloud content execute --name share --options "file=document.pdf,user=bob"
```

## Configuration

Default configuration is provided in `config/defaults.sh`:

```bash
NEXTCLOUD_PORT=8086
NEXTCLOUD_ADMIN_USER=admin
NEXTCLOUD_ADMIN_PASSWORD=changeme
NEXTCLOUD_DATA_DIR=/var/www/html/data
```

Override defaults by setting environment variables:

```bash
export NEXTCLOUD_PORT=8090
export NEXTCLOUD_ADMIN_PASSWORD=mysecurepassword
vrooli resource nextcloud manage start
```

## Dependencies

- **PostgreSQL**: Metadata storage (automatically configured)
- **Redis**: Cache and file locking (automatically configured)
- **MinIO** (optional): S3-compatible object storage

## API Access

### WebDAV
Access files via WebDAV at `http://localhost:8086/remote.php/dav/files/[username]/`

```bash
# Upload via WebDAV
curl -u admin:password -T file.pdf \
  http://localhost:8086/remote.php/dav/files/admin/file.pdf

# List files
curl -u admin:password -X PROPFIND \
  http://localhost:8086/remote.php/dav/files/admin/
```

### OCS API
Share files and manage users via OCS API at `http://localhost:8086/ocs/v2.php/`

```bash
# Create share link
curl -u admin:password -X POST \
  http://localhost:8086/ocs/v2.php/apps/files_sharing/api/v1/shares \
  -d "path=/file.pdf&shareType=3"
```

## Usage Examples

### User Management
```bash
# Create new user
vrooli resource nextcloud users add --username john --password "SecurePass123!"

# List users
vrooli resource nextcloud users list

# Delete user
vrooli resource nextcloud users delete john
```

### Backup & Restore
```bash
# Create backup
vrooli resource nextcloud content execute --name backup

# Restore from backup
vrooli resource nextcloud content execute --name restore --options "nextcloud_backup_20250912.tar.gz"
```

### External Storage (S3)
```bash
# Mount S3 bucket
vrooli resource nextcloud content execute --name mount-s3 --options "bucket=mybucket,key=ACCESS_KEY,secret=SECRET_KEY"

# Mount MinIO bucket
vrooli resource nextcloud content execute --name mount-s3 --options "bucket=data,endpoint=http://localhost:9000,key=minioadmin,secret=minioadmin"
```

## CLI Commands

### Lifecycle Management
- `manage install` - Install Nextcloud and dependencies
- `manage start [--wait]` - Start Nextcloud services
- `manage stop` - Stop services
- `manage restart` - Restart services
- `manage uninstall [--keep-data]` - Remove Nextcloud

### Content Management
- `content add --file <path>` - Upload file
- `content list [--filter <pattern>]` - List files
- `content get --name <file>` - Download file
- `content remove --name <file>` - Delete file
- `content execute --name <operation>` - Execute operations (share, backup, restore, mount-s3, enable-office, configure-security)

### Testing
- `test smoke` - Quick health check
- `test integration` - Integration tests
- `test all` - Complete test suite

### Administration
- `occ <command>` - Execute Nextcloud OCC commands
- `users <operation>` - User management
- `apps <operation>` - App management
- `config <operation>` - Configuration management

## Integration Examples

### Scenario Integration
```javascript
// Upload file from scenario
const response = await fetch('http://localhost:8086/remote.php/dav/files/admin/report.pdf', {
  method: 'PUT',
  headers: {
    'Authorization': 'Basic ' + btoa('admin:password')
  },
  body: fileContent
});
```

### N8n Workflow Integration
Use the HTTP Request node with WebDAV endpoints for file operations.

### Python Integration
```python
import requests
from requests.auth import HTTPBasicAuth

# Upload file
with open('document.pdf', 'rb') as f:
    response = requests.put(
        'http://localhost:8086/remote.php/dav/files/admin/document.pdf',
        auth=HTTPBasicAuth('admin', 'password'),
        data=f
    )
```

## Troubleshooting

### Health Check Fails
```bash
# Check container status
docker ps | grep nextcloud

# View logs
vrooli resource nextcloud logs

# Test health endpoint
curl -sf http://localhost:8086/status.php
```

### Database Connection Issues
```bash
# Verify PostgreSQL is running
vrooli resource postgres status

# Check database connectivity
docker exec nextcloud_nextcloud_1 php occ db:connection
```

### Permission Errors
```bash
# Fix data directory permissions
docker exec nextcloud_nextcloud_1 chown -R www-data:www-data /var/www/html
```

## Security Considerations

1. **Change default passwords** immediately after installation
2. **Enable HTTPS** in production environments
3. **Configure firewall rules** to restrict access
4. **Enable two-factor authentication** for admin accounts
5. **Regular backups** of data and database
6. **Keep Nextcloud updated** for security patches

## Performance Tuning

### Redis Optimization
```bash
# Increase Redis memory limit
export REDIS_MAXMEMORY=2gb
vrooli resource redis config set maxmemory 2gb
```

### PHP Memory Limits
```bash
# Increase PHP memory for large files
docker exec nextcloud_nextcloud_1 sed -i 's/memory_limit = .*/memory_limit = 512M/' /usr/local/etc/php/php.ini
```

### Database Tuning
```sql
-- Optimize PostgreSQL for Nextcloud
ALTER SYSTEM SET shared_buffers = '256MB';
ALTER SYSTEM SET effective_cache_size = '1GB';
```

## Advanced Features

### External Storage
Mount external storage sources:
```bash
vrooli resource nextcloud occ files_external:create \
  "S3 Storage" amazons3 null::null \
  -c bucket=my-bucket \
  -c key=access_key \
  -c secret=secret_key
```

### Collabora Office Integration
Collabora Office is now included and can be enabled with a single command:
```bash
# Enable Collabora Office integration
vrooli resource nextcloud content execute --name enable-office

# Access Collabora Admin UI
# URL: http://localhost:9980/browser/dist/admin/admin.html
# Username: admin, Password: changeme
```

Once enabled, you can:
- Create new documents directly in Nextcloud (Files → + → New document/spreadsheet/presentation)
- Edit existing Office files by clicking on them
- Collaborate in real-time with other users
- Export documents in various formats

### Calendar and Contacts (CalDAV/CardDAV)
Nextcloud now includes full calendar and contacts support:
```bash
# CalDAV endpoint for calendars
http://localhost:8086/remote.php/dav/calendars/[username]/

# CardDAV endpoint for contacts  
http://localhost:8086/remote.php/dav/addressbooks/users/[username]/

# Access via web interface
# Calendar: Apps → Calendar
# Contacts: Apps → Contacts
```

### Security Configuration
Harden your Nextcloud installation with security best practices:
```bash
# Apply security configuration
vrooli resource nextcloud content execute --name configure-security

# Enable encryption (optional, affects performance)
vrooli resource nextcloud content execute --name configure-security --options "encrypt=true"
```

This configures:
- HTTPS headers (requires reverse proxy for full HTTPS)
- CSP and security headers
- Brute force protection
- Strong password policy (10+ chars, special, numeric, mixed case)
- Session security hardening
- Two-factor authentication support

### Backup and Restore
```bash
# Backup data and database
vrooli resource nextcloud content execute --name backup

# Restore from backup
vrooli resource nextcloud content execute --name restore --options "backup=20250110.tar.gz"
```

## Support

For issues or questions:
1. Check the [PRD.md](./PRD.md) for requirements and specifications
2. Review [Nextcloud documentation](https://docs.nextcloud.com)
3. Run diagnostic tests: `vrooli resource nextcloud test all`
4. Check container logs: `vrooli resource nextcloud logs`

## License

Nextcloud is licensed under AGPLv3. This resource wrapper follows Vrooli's licensing terms.