# Claude Code Cache Check (system-claude-cache)

Monitors the Claude Code cache directory for excessive files that can cause inotify watcher exhaustion (ENOSPC errors).

## Overview

| Property | Value |
|----------|-------|
| Check ID | `system-claude-cache` |
| Category | System |
| Interval | 1 hour |
| Platforms | All |

## What It Monitors

Claude Code maintains a cache in `~/.claude/` containing:
- **todos/**: Task and todo data from conversations
- **file-history/**: File change tracking for conversations
- **shell-snapshots/**: Shell state snapshots

Over time, this cache can grow to contain 10,000+ files, exhausting the system's inotify watcher limit.

## Status Meanings

| Status | Meaning |
|--------|---------|
| **OK** | Cache has < 5,000 files |
| **Warning** | Cache has 5,000-10,000 files - cleanup recommended |
| **Critical** | Cache has > 10,000 files - may cause ENOSPC errors |

## Why It Matters

When the cache grows too large:
- Claude Code startup fails with: `ENOSPC: System limit for number of file watchers reached`
- IDE file watching breaks (VS Code, other editors)
- Build tools that use file watching fail
- Development workflow is severely impacted

## Recovery Actions

| Action | Description | Risk |
|--------|-------------|------|
| **Cleanup Stale Files** | Remove files older than thresholds | Safe - preserves recent data |
| **Full Cleanup** | Remove all cleanable cache files (keeps last 24h) | Medium - more aggressive |
| **Analyze Cache** | Get detailed breakdown of cache contents | Safe |

## Default Cleanup Thresholds

| Directory | Max Age |
|-----------|---------|
| todos/ | 7 days |
| file-history/ | 30 days |
| shell-snapshots/ | 7 days |

## Troubleshooting Steps

### 1. Check Current Cache Size
```bash
# Count files
find ~/.claude -type f | wc -l

# Check disk usage
du -sh ~/.claude
du -sh ~/.claude/*
```

### 2. Check inotify Limit
```bash
# Current limit
cat /proc/sys/fs/inotify/max_user_watches

# Current usage (approximate)
cat /proc/sys/fs/inotify/max_user_instances
```

### 3. Manual Cleanup
```bash
# Remove old todos (7+ days)
find ~/.claude/todos -type f -mtime +7 -delete

# Remove old history (30+ days)
find ~/.claude/file-history -type f -mtime +30 -delete

# Remove old shell snapshots (7+ days)
find ~/.claude/shell-snapshots -type f -mtime +7 -delete

# Remove empty directories
find ~/.claude -type d -empty -delete
```

### 4. Increase inotify Limit (Temporary Fix)
```bash
# Temporary increase
sudo sysctl fs.inotify.max_user_watches=524288

# Permanent (add to /etc/sysctl.conf)
echo "fs.inotify.max_user_watches=524288" | sudo tee -a /etc/sysctl.conf
sudo sysctl -p
```

## Prevention Tips

1. **Regular Cleanup**: Run this check and cleanup action weekly
2. **Monitor Growth**: Watch for warning status before it becomes critical
3. **Increase Limits**: Consider raising inotify limits on development machines

## The ENOSPC Error

When you see:
```
Error: ENOSPC: System limit for number of file watchers reached
```

This means the kernel has run out of inotify watches. Common causes:
1. Claude Code cache (this check catches this)
2. Large `node_modules` directories
3. Many VS Code workspaces open
4. Other file-watching tools

## Related Checks

- **system-inode**: Inode exhaustion (related but different issue)
- **system-disk**: Disk space usage

---

*Back to [Check Catalog](../check-catalog.md)*
