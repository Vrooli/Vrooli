# Data Backup Manager

A comprehensive backup management system for all Vrooli resources and scenarios, providing automated data protection and disaster recovery capabilities.

## 🎯 Purpose

The Data Backup Manager is a critical infrastructure component that ensures data integrity and business continuity across the entire Vrooli ecosystem. It provides:

- **Automated Backups**: Scheduled backups of PostgreSQL databases, scenario files, and MinIO object storage
- **Point-in-Time Recovery**: Restore data to any point within the retention period
- **Intelligent Monitoring**: Health checks and alerting for backup operations
- **Multi-Target Support**: Backup databases, files, and object storage with a unified interface

## 🚀 Quick Start

### Prerequisites
- PostgreSQL (for backup metadata storage)
- MinIO (for backup file storage)
- N8n (for backup orchestration)
- Go 1.19+ (for building the API)

### Installation

1. **Setup the scenario**:
   ```bash
   cd scenarios/data-backup-manager
   vrooli scenario setup data-backup-manager
   ```

2. **Start the service**:
   ```bash
   vrooli scenario develop data-backup-manager
   ```

3. **Verify installation**:
   ```bash
   data-backup-manager status
   ```

## 📚 Usage

### CLI Commands

#### Check System Status
```bash
# Basic status
data-backup-manager status

# Detailed status with JSON output
data-backup-manager status --json --verbose
```

#### Create Backups
```bash
# Full backup of all targets
data-backup-manager backup postgres,files,scenarios --type full

# Incremental backup with custom retention
data-backup-manager backup postgres --type incremental --retention-days 30
```

#### Restore Data
```bash
# Restore from latest backup
data-backup-manager restore --targets postgres --verify

# Restore from specific backup job
data-backup-manager restore --backup-id <job-id> --targets files
```

#### Manage Schedules
```bash
# List all backup schedules
data-backup-manager schedule list

# Create new schedule
data-backup-manager schedule create \
  --name "daily-full-backup" \
  --cron "0 2 * * *" \
  --targets postgres,files \
  --retention 7

# Enable/disable schedules
data-backup-manager schedule enable daily-full-backup
data-backup-manager schedule disable daily-full-backup
```

#### Verify Backups
```bash
# Verify specific backup
data-backup-manager verify <backup-job-id>

# Verify latest backup for target
data-backup-manager verify --target postgres --latest
```

### API Usage

The Data Backup Manager provides a REST API for programmatic access:

```bash
# Check system status
curl http://localhost:20010/api/v1/backup/status

# Create immediate backup
curl -X POST http://localhost:20010/api/v1/backup/create \
  -H "Content-Type: application/json" \
  -d '{
    "type": "full",
    "targets": ["postgres", "files"],
    "description": "Manual backup"
  }'

# List available backups
curl http://localhost:20010/api/v1/backup/list?target=postgres

# Start restore operation
curl -X POST http://localhost:20010/api/v1/restore/create \
  -H "Content-Type: application/json" \
  -d '{
    "backup_job_id": "<job-id>",
    "targets": ["postgres"],
    "verify_before_restore": true
  }'
```

## 🔧 Configuration

### Backup Targets

The system supports backing up multiple target types:

- **postgres**: All PostgreSQL databases with schema and data
- **files**: Scenario files, configurations, and user data
- **scenarios**: Complete scenario directories with metadata
- **minio**: Object storage buckets and metadata

### Backup Types

- **Full**: Complete backup of all data in target
- **Incremental**: Only changes since last backup (reduces storage and time)
- **Differential**: Changes since last full backup

### Storage Configuration

Backups are stored in MinIO with the following structure:
```
backups/
├── postgres/
│   ├── YYYY-MM-DD/
│   └── incremental/
├── files/
│   ├── scenarios/
│   └── configuration/
├── metadata/
│   └── job-manifests/
└── verification/
    └── checksums/
```

## 🏗️ Architecture

### Components

- **API Server** (Go): RESTful API for backup operations and status
- **CLI Tool** (Go): Command-line interface for interactive use
- **N8n Workflows**: Automated scheduling and orchestration
- **PostgreSQL**: Metadata storage for jobs, schedules, and restore points
- **MinIO**: Object storage for backup files and archives

### Data Flow

1. **Backup Creation**: API receives backup request → Creates job record → Triggers appropriate backup tool
2. **File Processing**: Data is compressed and encrypted → Uploaded to MinIO → Checksum verification
3. **Metadata Update**: Job status updated → Events published → Cleanup old backups based on retention

### Integration Points

- **System Monitor**: Provides backup health metrics and alerts
- **Maintenance Orchestrator**: Schedules automated backup jobs
- **Notification Hub**: Sends backup completion and failure notifications
- **Scenario Generator**: Automatically backs up new scenarios

## 📊 Monitoring

### Health Checks

The system provides multiple health check endpoints:

- `/health` - Overall system health
- `/api/v1/backup/status` - Backup system status with details
- `/api/v1/storage/status` - Storage usage and availability

### Metrics

Key metrics tracked:
- Backup success/failure rates
- Backup duration and size trends  
- Storage usage and growth
- Recovery time objectives (RTO)
- Recovery point objectives (RPO)

### Alerting

Automatic alerts for:
- Backup job failures
- Storage capacity warnings
- Backup verification failures
- Missed scheduled backups
- Long-running backup operations

## 🛠️ Development

### Building

```bash
# Build API server
cd api
go mod download
go build -o data-backup-manager-api .

# Make CLI executable
chmod +x cli/data-backup-manager
```

### Testing

```bash
# Run scenario tests
vrooli test scenario data-backup-manager

# Run specific test categories
vrooli test scenario data-backup-manager --structure
vrooli test scenario data-backup-manager --integration
vrooli test scenario data-backup-manager --performance
```

### Adding New Backup Targets

1. Define target configuration in `lib/targets/`
2. Implement backup and restore logic
3. Add CLI command support
4. Update API endpoints
5. Add integration tests

## 🔐 Security

### Encryption
- **At Rest**: AES-256 encryption for all backup files
- **In Transit**: TLS 1.3 for all data transfers
- **Key Management**: Automatic key rotation every 90 days

### Access Control
- Backup files accessible only to authorized processes
- Role-based access control for API endpoints
- Audit logging for all backup and restore operations

### Compliance
- Data retention policies enforced automatically
- Immutable backup records for audit trails
- GDPR-compliant data handling procedures

## 📈 Performance

### Optimization Features
- Incremental backups reduce storage by 70%+
- Parallel processing for large datasets
- Compression reduces backup size by 60%+
- Deduplication across backup jobs

### Scalability
- Horizontal scaling through multiple backup workers
- Storage scaling through MinIO clustering
- Database scaling through PostgreSQL read replicas

## 🆘 Troubleshooting

### Common Issues

#### Backup Jobs Stuck in "Running" Status
```bash
# Check for stuck processes
data-backup-manager status --verbose

# Restart stuck jobs
data-backup-manager jobs restart <job-id>
```

#### Storage Space Warnings
```bash
# Check storage usage
data-backup-manager status --storage

# Clean up old backups
data-backup-manager cleanup --older-than 30d --dry-run
data-backup-manager cleanup --older-than 30d --confirm
```

#### Restore Failures
```bash
# Verify backup integrity first
data-backup-manager verify <backup-id>

# Check restore destination permissions
ls -la /path/to/restore/destination

# Attempt restore with verbose logging
data-backup-manager restore --backup-id <id> --targets postgres --verbose
```

### Log Locations
- API logs: `data/logs/api.log`
- CLI logs: `~/.local/share/vrooli/data-backup-manager/cli.log`
- Backup job logs: `data/logs/jobs/<job-id>.log`

## 🤝 Contributing

1. Follow the PRD specifications in `PRD.md`
2. Ensure all tests pass: `vrooli test scenario data-backup-manager`
3. Update documentation for new features
4. Follow Go coding standards and best practices

## 📄 License

MIT License - see LICENSE file for details

---

**Service Status**: 🟡 In Development  
**Last Updated**: 2025-09-05  
**API Version**: v1  
**Maintenance Window**: Daily 2:00-2:30 AM UTC