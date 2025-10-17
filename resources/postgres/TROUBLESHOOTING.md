# PostgreSQL Troubleshooting Guide

## Connection Issues

### Problem: Connection pool exhausted
**Frequency:** Occurs weekly during peak usage
**Symptoms:**
- "FATAL: remaining connection slots are reserved" errors
- Application timeouts when trying to connect
- Database becomes unresponsive

**Current Workaround:**
Restart the PostgreSQL service to reset connections. This is a temporary fix.

**TODO:** Implement proper connection pooling with PgBouncer and automatic cleanup.

**Solution:**
```bash
# Temporary fix - increase max connections
docker exec -it vrooli-postgres psql -U postgres -c "ALTER SYSTEM SET max_connections = 200;"
docker restart vrooli-postgres
```

### Problem: Slow queries on large tables
**Known issue:** Queries on tables over 1GB take >30 seconds
**FIXME:** Missing indexes on frequently queried columns

**Workaround:**
```sql
-- Add these indexes manually until permanent fix
CREATE INDEX CONCURRENTLY idx_users_email ON users(email);
CREATE INDEX CONCURRENTLY idx_tasks_status ON tasks(status, created_at);
```

## Backup and Recovery

### Problem: Backup script fails silently
**WARNING:** Backups may not be running as expected
**Frequency:** Intermittent failures, about 20% of runs

**Temporary solution:**
Add verbose logging to backup script:
```bash
pg_dump -v -h localhost -U postgres dbname > backup.sql 2> backup.log
```

**NOTE:** Need to implement backup monitoring and alerting

## Performance Issues

### Problem: High memory usage leading to OOM kills
**Known issue:** PostgreSQL consuming >80% of available memory
**Impact:** Container gets killed by Docker when memory limit exceeded

**Current mitigation:**
- Reduced shared_buffers from 1GB to 512MB
- Set work_mem to 4MB (was 16MB)

**TODO:** Properly tune PostgreSQL memory settings based on container limits

## Replication Problems

### Problem: Replication lag increasing over time
**FIXME:** Streaming replication falls behind during bulk operations
**Workaround:** Pause bulk operations periodically to let replica catch up

## Common Error Messages

### "could not extend file: No space left on device"
**Solution:**
1. Check disk space: `df -h`
2. Clean up old WAL files: `pg_archivecleanup /var/lib/postgresql/data/pg_wal`
3. Vacuum large tables: `VACUUM FULL;`

### "FATAL: password authentication failed"
**Common causes:**
- Wrong password in connection string
- pg_hba.conf misconfiguration
- User doesn't exist

**Fix:**
```bash
# Reset password
docker exec -it vrooli-postgres psql -U postgres
ALTER USER youruser WITH PASSWORD 'newpassword';
```

## Monitoring Gaps

**TODO:** Implement monitoring for:
- Connection pool usage
- Query performance metrics
- Replication lag
- Backup success/failure
- Disk space usage

## Integration Issues

### Problem: N8N can't connect after PostgreSQL restart
**Known issue:** N8N holds stale connections that don't reconnect
**Workaround:** Always restart N8N after PostgreSQL restart
**FIXME:** Configure connection pooling with automatic reconnection