# System Health Checks

System checks monitor operating system resources to detect capacity issues before they cause failures.

---

## system-disk: Disk Space

**Interval:** 300 seconds (5 minutes)
**Platforms:** Linux

Monitors disk usage on mounted partitions.

### Why It Matters
Low disk space causes:
- Database write failures
- Log rotation failures
- Container image pull failures
- Swap file exhaustion

### Thresholds
- **Warning**: 80% usage
- **Critical**: 95% usage

### Status Meanings
- **OK**: Below warning threshold
- **Warning**: Above 80%, below 95%
- **Critical**: Above 95%

### Troubleshooting
1. Check usage: `df -h`
2. Find large files: `du -h --max-depth=1 / | sort -rh | head -20`
3. Clean Docker: `docker system prune -a`
4. Clear old logs: `sudo journalctl --vacuum-time=7d`
5. Check /tmp: `du -sh /tmp/*`

---

## system-inode: Inode Usage

**Interval:** 300 seconds (5 minutes)
**Platforms:** Linux

Monitors inode (file metadata) usage on filesystems.

### Why It Matters
Inode exhaustion prevents file creation even with free disk space:
- Many small files exhaust inodes
- Container layers use many inodes
- Log rotation creates many files

### Thresholds
- **Warning**: 85% usage
- **Critical**: 95% usage

### Status Meanings
- **OK**: Below warning threshold
- **Warning**: High inode usage
- **Critical**: Near exhaustion

### Troubleshooting
1. Check usage: `df -i`
2. Find directories with many files:
   ```bash
   find / -xdev -type d -size +100k 2>/dev/null
   ```
3. Clean up old files in /tmp, /var/log

---

## system-swap: Swap Memory

**Interval:** 300 seconds (5 minutes)
**Platforms:** Linux

Monitors swap usage as an indicator of memory pressure.

### Why It Matters
High swap usage indicates:
- Memory pressure
- Potential memory leaks
- Performance degradation
- Risk of OOM killer activation

### Thresholds
- **Warning**: 50% swap usage

### Status Meanings
- **OK**: Low swap usage
- **Warning**: Elevated swap indicating memory pressure

### Troubleshooting
1. Check swap: `free -h`
2. Find memory consumers: `ps aux --sort=-%mem | head -10`
3. Check for memory leaks in long-running processes
4. Consider adding more RAM or increasing swap

---

## system-zombies: Zombie Processes

**Interval:** 300 seconds (5 minutes)
**Platforms:** Linux, macOS

Detects zombie (defunct) processes that indicate resource leaks.

### Why It Matters
Zombie processes:
- Indicate parent process bugs
- Consume process table entries
- Can exhaust PID namespace
- Suggest unhandled child exits

### Thresholds
- **Warning**: 5 zombies
- **Critical**: 20 zombies

### Status Meanings
- **OK**: Few or no zombies
- **Warning**: Moderate zombie count
- **Critical**: High zombie count

### Recovery Actions
- **Reap Zombies**: Send SIGCHLD to parent processes
- **List Zombies**: Show zombie processes and parents

### Troubleshooting
1. List zombies: `ps aux | awk '$8 ~ /Z/'`
2. Find parent: `ps -o ppid= -p <zombie_pid>`
3. Signal parent: `kill -SIGCHLD <parent_pid>`
4. If parent is unresponsive, consider restarting it

---

## system-ports: Ephemeral Port Usage

**Interval:** 300 seconds (5 minutes)
**Platforms:** Linux

Monitors ephemeral port exhaustion which can prevent new connections.

### Why It Matters
Port exhaustion causes:
- "Cannot assign requested address" errors
- Connection failures for new requests
- Service unavailability

### Thresholds
- **Warning**: 70% port range used
- **Critical**: 85% port range used

### Status Meanings
- **OK**: Healthy port availability
- **Warning**: Elevated usage
- **Critical**: Near exhaustion

### Recovery Actions
- **Analyze Connections**: Show port usage by process
- **Show TIME_WAIT**: List connections holding ports

### Troubleshooting
1. Check port range: `cat /proc/sys/net/ipv4/ip_local_port_range`
2. Count connections: `ss -tan | wc -l`
3. Find consumers: `ss -tan | awk '{print $5}' | sort | uniq -c | sort -rn`
4. Check for connection leaks in applications
5. Tune TIME_WAIT: `sysctl net.ipv4.tcp_tw_reuse=1`
