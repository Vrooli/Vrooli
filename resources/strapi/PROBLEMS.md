# Known Problems - Strapi Resource

## Installation Issues

### Node.js Version Requirements
**Problem**: Strapi v5 requires Node.js 18+ (recommended 20 LTS)
**Solution**: Install Node.js 20 LTS before attempting installation
```bash
# Check Node version
node --version

# Install via nvm if needed
nvm install 20
nvm use 20
```

### PostgreSQL Connection
**Problem**: Installation requires PostgreSQL to be running
**Solution**: Ensure PostgreSQL resource is active
```bash
vrooli resource postgres manage start --wait
```

## Runtime Issues

### Port Conflicts
**Problem**: Port 1337 may be in use by another service
**Solution**: Check and free the port or use Docker deployment
```bash
# Check port usage
lsof -i :1337

# Or use Docker which manages ports internally
vrooli resource strapi docker start
```

### Memory Usage
**Problem**: Strapi can consume significant memory (>512MB)
**Solution**: Monitor and adjust PM2 memory limits in ecosystem.config.js
```javascript
max_memory_restart: '1G'  // Increase limit if needed
```

## Docker Issues

### Network Creation
**Problem**: Docker network 'vrooli-network' must exist
**Solution**: Network is auto-created, but can be manually created:
```bash
docker network create vrooli-network
```

### Volume Permissions
**Problem**: Docker volumes may have permission issues
**Solution**: Ensure proper ownership of mounted directories
```bash
# Fix permissions if needed
sudo chown -R $USER:$USER ~/.vrooli/strapi
```

## API Issues

### Admin Registration
**Problem**: Admin user can only be created once
**Solution**: Use existing credentials or reset database
```bash
# View existing credentials
vrooli resource strapi credentials

# Or reset (destructive)
vrooli resource strapi manage uninstall
vrooli resource strapi manage install
```

### GraphQL Not Available
**Problem**: GraphQL endpoint returns 404
**Solution**: GraphQL plugin must be installed and enabled
```bash
# GraphQL is included in v5 but must be configured
# Check admin panel: Settings > Plugins
```

## Performance Issues

### Slow Startup
**Problem**: Initial startup can take 30-60 seconds
**Solution**: This is normal for first run as Strapi builds assets
- Subsequent starts are faster
- Use `--wait` flag for automated scripts

### Build Times
**Problem**: `npm run build` takes several minutes
**Solution**: This is expected for production builds
- Use development mode for faster iteration
- Consider using Docker pre-built images

## Storage Issues

### MinIO Connection Failed
**Problem**: S3 storage enable fails with connection error
**Solution**: Ensure MinIO is running and accessible
```bash
# Start MinIO
vrooli resource minio manage start

# Test MinIO connection
curl -I http://localhost:9000/minio/health/live

# Then enable storage
vrooli resource strapi storage enable
```

### S3 Bucket Creation Failed
**Problem**: Cannot create or access S3 bucket
**Solution**: Check MinIO credentials and permissions
```bash
# Verify MinIO credentials match
export MINIO_ACCESS_KEY=minioadmin
export MINIO_SECRET_KEY=minioadmin

# Test storage integration
vrooli resource strapi storage test
```

## Backup/Restore Issues

### Backup Fails - Database Export
**Problem**: pg_dump command not found
**Solution**: Use Docker-based backup or install PostgreSQL client
```bash
# Install PostgreSQL client
sudo apt-get install postgresql-client

# Or use Docker if PostgreSQL is containerized
docker exec postgres pg_dump -U postgres strapi > backup.sql
```

### Restore Fails - Permission Denied
**Problem**: Cannot write to restore directories
**Solution**: Check file permissions and ownership
```bash
# Fix permissions
chmod 755 /app/public/uploads
chown -R $USER:$USER /app
```

### Scheduled Backups Not Running
**Problem**: Cron job not executing
**Solution**: Verify crontab entry and script permissions
```bash
# Check crontab
crontab -l | grep strapi

# Test backup script manually
~/.vrooli/backups/strapi/auto-backup.sh
```

## Troubleshooting Commands

```bash
# Check service health
vrooli resource strapi status

# View logs
vrooli resource strapi logs --tail 100

# Test basic functionality
vrooli resource strapi test smoke

# Reset everything
vrooli resource strapi manage uninstall
vrooli resource strapi manage install
```

## Contact Support

For issues not covered here:
1. Check Strapi documentation: https://docs.strapi.io/
2. Review Vrooli logs: `vrooli resource strapi logs`
3. Submit issue to Vrooli repository