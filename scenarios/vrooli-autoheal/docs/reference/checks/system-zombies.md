# Zombie Process Check (system-zombies)

Detects zombie (defunct) processes that indicate resource leaks and potential process table exhaustion.

## Overview

| Property | Value |
|----------|-------|
| Check ID | `system-zombies` |
| Category | System |
| Interval | 300 seconds (5 minutes) |
| Platforms | Linux |

## What It Monitors

This check scans `/proc` to count processes in the "Z" (zombie) state:

- Process state from `/proc/[pid]/stat`
- Parent process ID for each zombie
- Process name (command) of zombies

## Status Meanings

| Status | Meaning |
|--------|---------|
| **OK** | No zombies, or fewer than 5 zombies |
| **Warning** | Between 5-20 zombie processes |
| **Critical** | More than 20 zombie processes |

## Why It Matters

Zombie processes indicate programming bugs or resource management issues:

- **Parent not reaping**: Parent process isn't calling `wait()` on children
- **Process table pressure**: Too many zombies can exhaust the process table
- **Resource indicator**: Often a symptom of larger issues

While zombies don't consume CPU or memory (they're just process table entries), they:
- Indicate buggy software that needs attention
- Can eventually prevent new process creation
- May mask the root cause of other issues

## Understanding Zombie Processes

```
Process Lifecycle:
1. Parent forks child process
2. Child executes and eventually exits
3. Child becomes zombie (Z state) holding exit code
4. Parent calls wait() to read exit code
5. Zombie is finally removed (reaped)

If step 4 never happens, zombie persists.
```

## Common Failure Causes

### 1. Buggy Parent Process

```bash
# Find parent processes of zombies
ps aux | awk '$8=="Z" {print $2}' | while read pid; do
  cat /proc/$pid/stat 2>/dev/null | awk -F')' '{print $2}' | awk '{print $2}'
done | sort | uniq -c | sort -rn
```

### 2. Signal Handling Issues

Parent may not handle SIGCHLD properly:
```bash
# Check parent process
ps -o pid,ppid,state,comm -p $(pgrep -P 1 --list-full | grep defunct | awk '{print $1}')
```

### 3. Long-Running Daemons

Daemons that spawn many children may accumulate zombies:
```bash
# Find which daemons have zombie children
for pid in $(pgrep -f "defunct" 2>/dev/null); do
  ppid=$(ps -o ppid= -p $pid 2>/dev/null)
  if [ -n "$ppid" ]; then
    ps -o comm= -p $ppid 2>/dev/null
  fi
done | sort | uniq -c | sort -rn
```

## Troubleshooting Steps

1. **List current zombies**
   ```bash
   ps aux | grep -w Z
   # or
   ps aux | awk '$8=="Z"'
   ```

2. **Find their parent processes**
   ```bash
   ps -eo pid,ppid,state,comm | awk '$3=="Z" {print $2}' | sort -u | while read ppid; do
     ps -o pid,comm -p $ppid
   done
   ```

3. **Send SIGCHLD to parents**
   ```bash
   # This tells the parent to check for dead children
   ps -eo pid,ppid,state | awk '$3=="Z" {print $2}' | sort -u | xargs -r kill -SIGCHLD
   ```

4. **If parent is unresponsive, kill parent**
   ```bash
   # WARNING: This kills the parent, not the zombie
   # The zombie will be adopted by init and reaped
   ps -eo pid,ppid,state | awk '$3=="Z" {print $2}' | sort -u | xargs -r kill
   ```

5. **Check for recurring zombies**
   ```bash
   # Monitor zombie count over time
   watch -n 5 'ps aux | awk "\$8==\"Z\"" | wc -l'
   ```

## Configuration

The check accepts the following options:

| Option | Default | Description |
|--------|---------|-------------|
| `warningThreshold` | 5 | Warning at this count |
| `criticalThreshold` | 20 | Critical at this count |

## Details Returned

```json
{
  "zombieCount": 3,
  "warningThreshold": 5,
  "criticalThreshold": 20,
  "zombies": [
    {"pid": "12345", "ppid": "1234", "name": "defunct-process"},
    {"pid": "12346", "ppid": "1234", "name": "defunct-process"}
  ]
}
```

## Health Score

The health score is calculated as `100 - (zombieCount * 5)`, with a minimum of 0.

## Related Checks

- **system-ports**: Both indicate resource exhaustion risks
- **infra-docker**: Container processes may create zombies

## Prevention Tips

1. **Code review for fork() usage**: Ensure wait() is called
2. **Use double-fork for daemons**: Prevents zombie accumulation
3. **Set SIGCHLD handler**: Properly reap children in signal handler
4. **Monitor trends**: Gradual increase indicates a leak

## Auto-Heal Actions

When zombies exceed thresholds:
1. Send SIGCHLD to parent processes (attempt to trigger reaping)
2. Alert administrators for investigation
3. If persistent, consider restarting the parent service

---

*Back to [Check Catalog](../check-catalog.md)*
