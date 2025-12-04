# Watchdog Design

How Autoheal survives crashes and reboots through OS-level watchdog integration.

## The Problem

Without a watchdog:

```mermaid
sequenceDiagram
    participant OS
    participant Autoheal
    participant Services

    OS->>Autoheal: Start
    Autoheal->>Services: Monitor health

    Note over Autoheal: Process crashes!

    Services->>Services: Failures go undetected

    Note over OS,Services: Manual intervention required
```

## The Solution

With OS-level watchdog:

```mermaid
sequenceDiagram
    participant OS
    participant Watchdog
    participant Autoheal
    participant Services

    OS->>Watchdog: Start on boot
    Watchdog->>Autoheal: Start & monitor
    Autoheal->>Services: Monitor health

    Note over Autoheal: Process crashes!

    Watchdog->>Watchdog: Detect crash
    Watchdog->>Autoheal: Restart
    Autoheal->>Services: Resume monitoring

    Note over OS,Services: Automatic recovery!
```

## Platform-Specific Implementations

### Linux (systemd)

```mermaid
graph LR
    subgraph "systemd"
        Service["vrooli-autoheal.service"]
        Timer["Optional: timer unit"]
    end

    Service --> Binary["autoheal binary"]

    style Service fill:#1e40af,color:#fff
```

Service file (`/etc/systemd/system/vrooli-autoheal.service`):

```ini
[Unit]
Description=Vrooli Autoheal - Self-healing infrastructure supervisor
After=network.target docker.service

[Service]
Type=simple
ExecStart=/usr/local/bin/vrooli-autoheal loop
Restart=always
RestartSec=10
Environment=VROOLI_LIFECYCLE_MANAGED=true

[Install]
WantedBy=multi-user.target
```

### macOS (launchd)

Plist file (`~/Library/LaunchAgents/com.vrooli.autoheal.plist`):

```xml
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" ...>
<plist version="1.0">
<dict>
    <key>Label</key>
    <string>com.vrooli.autoheal</string>
    <key>ProgramArguments</key>
    <array>
        <string>/usr/local/bin/vrooli-autoheal</string>
        <string>loop</string>
    </array>
    <key>KeepAlive</key>
    <true/>
    <key>RunAtLoad</key>
    <true/>
</dict>
</plist>
```

### Windows (Scheduled Task or Service)

Using Windows Task Scheduler:

```powershell
$action = New-ScheduledTaskAction -Execute "vrooli-autoheal.exe" -Argument "loop"
$trigger = New-ScheduledTaskTrigger -AtStartup
$settings = New-ScheduledTaskSettingsSet -RestartCount 3 -RestartInterval (New-TimeSpan -Minutes 1)
Register-ScheduledTask -TaskName "VrooliAutoheal" -Action $action -Trigger $trigger -Settings $settings
```

## Installation Flow

```mermaid
flowchart TD
    A[watchdog install] --> B{Detect platform}
    B -->|Linux| C{Has systemd?}
    B -->|macOS| D[Generate launchd plist]
    B -->|Windows| E[Create scheduled task]

    C -->|Yes| F[Generate systemd unit]
    C -->|No| G[Use cron fallback]

    F --> H[systemctl enable]
    D --> I[launchctl load]
    E --> J[Register-ScheduledTask]
    G --> K[Add crontab entry]

    H --> L[Watchdog active]
    I --> L
    J --> L
    K --> L
```

## Verification

Check watchdog status:

```bash
# Linux
systemctl status vrooli-autoheal

# macOS
launchctl list | grep vrooli

# Windows
Get-ScheduledTask -TaskName "VrooliAutoheal"
```

## Self-Protection

The watchdog protects autoheal, but autoheal also monitors the watchdog:

```mermaid
graph TB
    subgraph "Mutual Protection"
        Watchdog["OS Watchdog<br/>Restarts autoheal"]
        Autoheal["Autoheal<br/>Monitors watchdog status"]
    end

    Watchdog -->|restart on crash| Autoheal
    Autoheal -->|verify running| Watchdog
```

The `watchdog` health check verifies the OS service is properly configured:

```go
func (c *WatchdogCheck) Run(ctx context.Context) checks.Result {
    if !c.isWatchdogInstalled() {
        return checks.Result{
            Status:  checks.StatusWarning,
            Message: "OS watchdog not installed",
        }
    }
    if !c.isWatchdogRunning() {
        return checks.Result{
            Status:  checks.StatusCritical,
            Message: "OS watchdog not running",
        }
    }
    return checks.Result{
        Status:  checks.StatusOK,
        Message: "OS watchdog active",
    }
}
```

## Design Principles

1. **Idempotent Installation**: Running `watchdog install` multiple times is safe
2. **Graceful Degradation**: Works without watchdog (just with reduced resilience)
3. **Platform Abstraction**: Same CLI command, platform-specific implementation
4. **Minimal Privileges**: Only requests elevated access when needed
5. **Visible Status**: Watchdog status shown in dashboard
