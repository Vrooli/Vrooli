# Watchdog Installation

How to set up OS-level watchdog protection for Autoheal.

## Why Install a Watchdog?

Without a watchdog:
- Autoheal stops if the process crashes
- Autoheal doesn't start after system reboot
- Manual intervention required to recover

With a watchdog:
- Automatic restart on crash
- Automatic start on boot
- True self-healing infrastructure

## Installation Commands

### Quick Install

```bash
# Project setup owns the complete autoheal protection stack.
sudo vrooli setup
```

### Verify Installation

```bash
# Inspect the setup-owned safeguard and boot policy
vrooli setup status
vrooli setup explain autoheal_watchdog
```

## Platform-Specific Details

### Linux (systemd)

Project setup creates a user-scoped `vrooli-autoheal.service` using the shared
platform-go definition. On dedicated Linux hosts it also enables user lingering
so the service can start before login. Shared-host policy is explicitly
login-scoped and does not enable lingering.

Native scheduler mutation is intentionally not an operator procedure. Use
`vrooli setup` to reconcile it; setup reports a degraded state when the user
systemd bus is unavailable.

### macOS (launchd)

The installer creates `~/Library/LaunchAgents/com.vrooli.autoheal.plist`:

```xml
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN"
    "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>Label</key>
    <string>com.vrooli.autoheal</string>
    <key>ProgramArguments</key>
    <array>
        <string>/usr/local/bin/vrooli-autoheal</string>
        <string>loop</string>
        <string>--interval-seconds=60</string>
    </array>
    <key>KeepAlive</key>
    <true/>
    <key>RunAtLoad</key>
    <true/>
    <key>StandardOutPath</key>
    <string>/tmp/vrooli-autoheal.log</string>
    <key>StandardErrorPath</key>
    <string>/tmp/vrooli-autoheal.err</string>
</dict>
</plist>
```

**Manual Management:**

```bash
# Load (start)
launchctl bootstrap user/$(id -u) ~/Library/LaunchAgents/com.vrooli.autoheal.plist

# Unload (stop)
launchctl unload ~/Library/LaunchAgents/com.vrooli.autoheal.plist

# View logs
tail -f /tmp/vrooli-autoheal.log
```

### Windows (Task Scheduler)

The installer creates a scheduled task named "VrooliAutoheal":

**Manual Management (PowerShell as Admin):**

```powershell
# View task
Get-ScheduledTask -TaskName "VrooliAutoheal"

# Start manually
Start-ScheduledTask -TaskName "VrooliAutoheal"

# Stop
Stop-ScheduledTask -TaskName "VrooliAutoheal"

# Disable
Disable-ScheduledTask -TaskName "VrooliAutoheal"

# Remove
Unregister-ScheduledTask -TaskName "VrooliAutoheal" -Confirm:$false
```

## Troubleshooting

### "Permission denied" during install

**Linux:**
```bash
sudo vrooli setup
```

**macOS:** User-level launchd doesn't require sudo. For system-wide:
```bash
sudo vrooli setup
```

**Windows:** Run PowerShell as Administrator.

### Watchdog not starting

1. Check if another instance is running:
   ```bash
   # Linux
   pgrep -f vrooli-autoheal

   # macOS
   pgrep -f vrooli-autoheal

   # Windows
   Get-Process | Where-Object {$_.ProcessName -like "*autoheal*"}
   ```

2. Check logs for errors:
   ```bash
   # Linux
   journalctl -u vrooli-autoheal -n 50

   # macOS
   cat /tmp/vrooli-autoheal.err

   # Windows
   Get-EventLog -LogName Application -Source "VrooliAutoheal" -Newest 10
   ```

### Watchdog keeps restarting

This usually means the autoheal process is crashing. Check:

1. Environment variables (especially `VROOLI_LIFECYCLE_MANAGED`)
2. Database connectivity
3. Port conflicts

```bash
# View crash reasons (Linux)
journalctl -u vrooli-autoheal --since "10 minutes ago" | grep -i error
```

## Uninstalling

### Linux

There is no root-owned autoheal unit to remove. Keep or change the policy in
operator state, then reconcile with `sudo vrooli setup`. The setup-owned user
unit and lingering state must be inspected through `vrooli setup status`.

### macOS
```bash
launchctl unload ~/Library/LaunchAgents/com.vrooli.autoheal.plist
rm ~/Library/LaunchAgents/com.vrooli.autoheal.plist
```

### Windows (PowerShell as Admin)
```powershell
Stop-ScheduledTask -TaskName "VrooliAutoheal"
Unregister-ScheduledTask -TaskName "VrooliAutoheal" -Confirm:$false
```

## Verifying Protection

After installation, test the protection:

1. **Find the autoheal process:**
   ```bash
   pgrep -f vrooli-autoheal
   ```

2. **Kill it:**
   ```bash
   pkill -f vrooli-autoheal
   ```

3. **Wait 10-15 seconds and check again:**
   ```bash
   pgrep -f vrooli-autoheal
   ```

If the process restarts automatically, the watchdog is working correctly.

## Dashboard Indicator

The dashboard shows watchdog status in the "System Protection" card:
- **Protected** (green): Watchdog active and running
- **Unprotected** (amber): Watchdog not installed or not running
- **Error** (red): Watchdog configuration issue
