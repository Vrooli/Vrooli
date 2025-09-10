# Restic Resource

Enterprise-grade encrypted backup and recovery service for Vrooli.

## Overview

Restic provides automated, encrypted backups with deduplication and incremental snapshots. It protects all Vrooli resources, scenarios, and data with client-side AES-256 encryption and supports multiple storage backends.

## Features

- **Client-side encryption**: AES-256-GCM encryption before data leaves the system ✅
- **Deduplication**: Efficient storage with content-defined chunking ✅
- **Incremental snapshots**: Fast backups with point-in-time recovery ✅
- **Multiple backends**: Local, S3, MinIO, SFTP support (local implemented) ✅
- **Automated scheduling**: Configurable backup schedules via cron ✅
- **Retention policies**: Automatic pruning of old snapshots ✅
- **Health monitoring**: REST API with health endpoint ✅
- **Docker containerization**: Fully containerized with persistent volumes ✅

## Quick Start

```bash
# Install and initialize
resource-restic manage install

# Start the service
resource-restic manage start --wait

# Create a backup
resource-restic backup --paths /data --tags daily

# List snapshots
resource-restic snapshots

# Restore from latest snapshot
resource-restic restore --snapshot latest --target /restore

# Check status
resource-restic status
```

## Configuration

Configuration is managed through environment variables in `config/defaults.sh`:

- `RESTIC_REPOSITORY`: Repository path (default: `/repository`)
- `RESTIC_PASSWORD`: Encryption password (override in production!)
- `RESTIC_BACKUP_SCHEDULE`: Cron expression (default: daily at 2 AM)
- `RESTIC_KEEP_DAILY`: Daily snapshots to keep (default: 7)
- `RESTIC_KEEP_WEEKLY`: Weekly snapshots to keep (default: 4)
- `RESTIC_KEEP_MONTHLY`: Monthly snapshots to keep (default: 12)

### S3/MinIO Backend

To use S3 or MinIO as the backend:

```bash
export RESTIC_BACKEND=s3
export RESTIC_S3_ENDPOINT=http://vrooli-minio:9000
export AWS_ACCESS_KEY_ID=your-access-key
export AWS_SECRET_ACCESS_KEY=your-secret-key
export RESTIC_REPOSITORY=s3:http://vrooli-minio:9000/backup-bucket
```

## Management Commands

### Lifecycle Management
- `resource-restic manage install` - Install and initialize repository
- `resource-restic manage start` - Start the service
- `resource-restic manage stop` - Stop the service
- `resource-restic manage restart` - Restart the service
- `resource-restic manage uninstall` - Remove restic (keeps repository by default)

### Backup Operations
- `resource-restic backup` - Create a backup
- `resource-restic restore` - Restore from snapshot
- `resource-restic snapshots` - List available snapshots
- `resource-restic prune` - Remove old snapshots per retention policy

### Testing
- `resource-restic test smoke` - Quick health check
- `resource-restic test integration` - Full integration test
- `resource-restic test all` - Run all tests

## Integration with Other Resources

### PostgreSQL Backup
```bash
# Dump database before backup
pg_dump -h vrooli-postgres -U postgres mydb > /tmp/mydb.sql
resource-restic backup --paths /tmp/mydb.sql --tags postgres-daily
```

### MinIO Integration
```bash
# Configure MinIO as backend
export RESTIC_BACKEND=s3
export RESTIC_S3_ENDPOINT=http://vrooli-minio:9000
export RESTIC_REPOSITORY=s3:http://vrooli-minio:9000/backups
```

### Vault Integration
```bash
# Store encryption key in Vault
vault kv put secret/restic password=$RESTIC_PASSWORD
```

## Disaster Recovery

### Full System Restore
```bash
# 1. Install restic
resource-restic manage install

# 2. List available snapshots
resource-restic snapshots

# 3. Restore to target directory
resource-restic restore --snapshot <snapshot-id> --target /recovery

# 4. Verify restored data
ls -la /recovery
```

### Point-in-Time Recovery
```bash
# Find snapshot from specific date
resource-restic snapshots --tags daily | grep "2025-01-09"

# Restore that specific snapshot
resource-restic restore --snapshot <snapshot-id> --target /recovery
```

## Performance Tuning

### Compression
```bash
# Disable compression for already-compressed data
export RESTIC_COMPRESSION=off

# Maximum compression for text data
export RESTIC_COMPRESSION=max
```

### Parallelism
```bash
# Increase parallel operations for faster backups
export RESTIC_PARALLEL=4
```

## Security Best Practices

1. **Never commit passwords**: Use environment variables or Vault
2. **Rotate encryption keys**: Periodically change repository passwords
3. **Test restores regularly**: Ensure backups are restorable
4. **Monitor backup status**: Set up alerts for failed backups
5. **Secure repository access**: Limit network access to backup storage

## Troubleshooting

### Repository Locked
```bash
# If repository is locked from interrupted operation
docker exec vrooli-restic restic unlock
```

### Verify Repository Integrity
```bash
# Check repository for errors
docker exec vrooli-restic restic check
```

### Memory Issues
```bash
# Increase memory limit for large backups
export RESTIC_MEMORY_LIMIT=4g
resource-restic manage restart
```

## Monitoring

Check backup status and metrics:
```bash
# View status
resource-restic status --verbose

# Check last backup time
resource-restic snapshots | head -n 2

# View logs
resource-restic logs --tail 50
```

## API Endpoints

- `GET /health` - Health check endpoint
- `POST /api/backup` - Trigger manual backup
- `POST /api/restore` - Restore from backup
- `GET /api/snapshots` - List snapshots

## Support

For issues or questions:
- Check logs: `resource-restic logs`
- View status: `resource-restic status --verbose`
- Run tests: `resource-restic test all`
- See PRD.md for detailed requirements and architecture