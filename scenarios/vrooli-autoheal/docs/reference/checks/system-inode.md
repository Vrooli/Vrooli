# Inode Usage Check (system-inode)

Monitors inode usage to prevent file creation failures even when disk space is available.

## Overview

| Property | Value |
|----------|-------|
| Check ID | `system-inode` |
| Category | System |
| Interval | 300 seconds (5 minutes) |
| Platforms | Linux |

## What It Monitors

This check uses the `statfs` system call to measure:

- Total inodes per partition
- Used inodes per partition
- Usage percentage

Inodes are filesystem data structures that store metadata about files. Each file requires one inode, regardless of file size.

## Status Meanings

| Status | Meaning |
|--------|---------|
| **OK** | All partitions below 80% inode usage |
| **Warning** | At least one partition between 80-90% inode usage |
| **Critical** | At least one partition above 90% inode usage |

## Why It Matters

Inode exhaustion causes "No space left on device" errors even when `df -h` shows plenty of free space. This happens because:

- Each file consumes one inode
- Many small files can exhaust inodes before disk space
- Session files, cache entries, and temporary files are common culprits

This is particularly problematic for:
- Applications with many session files
- Build systems with many small artifacts
- Package managers with many cached packages

## Common Failure Causes

### 1. Many Small Files

```bash
# Count files per directory
find /var -xdev -type f | cut -d/ -f3 | sort | uniq -c | sort -rn | head

# Find directories with most files
for dir in /*; do
  echo "$(find "$dir" -xdev -type f 2>/dev/null | wc -l) $dir"
done | sort -rn | head
```

### 2. PHP/Node Session Files

```bash
# Check PHP sessions
find /var/lib/php/sessions -type f | wc -l

# Check temporary directories
find /tmp -type f | wc -l
```

### 3. Mail Queue

```bash
# Check mail spool
find /var/spool/mail -type f | wc -l
find /var/spool/postfix -type f | wc -l
```

### 4. Package Manager Cache

```bash
# NPM cache files
find ~/.npm -type f | wc -l

# PNPM store files
find ~/.local/share/pnpm -type f 2>/dev/null | wc -l
```

## Troubleshooting Steps

1. **Check inode usage**
   ```bash
   df -i
   ```

2. **Find directories with most files**
   ```bash
   find / -xdev -type d -exec sh -c 'echo "$(find "$0" -maxdepth 1 -type f | wc -l) $0"' {} \; 2>/dev/null | sort -rn | head -20
   ```

3. **Check for file count in common problem areas**
   ```bash
   # Session directories
   find /tmp /var/tmp -type f | wc -l

   # Log directories
   find /var/log -type f | wc -l
   ```

4. **Remove old files**
   ```bash
   # Remove files older than 7 days in /tmp
   find /tmp -type f -mtime +7 -delete
   ```

## Configuration

The check accepts the following options:

| Option | Default | Description |
|--------|---------|-------------|
| `partitions` | `["/"]` | List of mount points to monitor |
| `warningThreshold` | 80 | Warning at this percentage |
| `criticalThreshold` | 90 | Critical at this percentage |

## Details Returned

```json
{
  "partitions": [
    {
      "partition": "/",
      "usedPercent": 15,
      "usedInodes": 1234567,
      "totalInodes": 8388608,
      "freeInodes": 7154041,
      "status": "ok"
    }
  ],
  "warningThreshold": 80,
  "criticalThreshold": 90
}
```

## Health Score

The health score is calculated as `100 - worstPartitionUsagePercent`.

## Related Checks

- **system-disk**: Disk space and inode usage are related but independent
- **infra-docker**: Docker can create many small files (layers, logs)

## Windows Note

This check only runs on Linux. Windows uses NTFS which doesn't have the concept of inodes in the same way.

---

*Back to [Check Catalog](../check-catalog.md)*
