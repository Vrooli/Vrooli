# Dashboard Guide

How to use the Autoheal web dashboard effectively.

## Accessing the Dashboard

After starting the scenario, open the UI URL in your browser:

```bash
# Get the UI port
vrooli scenario status vrooli-autoheal

# Look for: UI_PORT=XXXXX
# Open: http://localhost:XXXXX
```

## Dashboard Layout

The dashboard is organized into several sections:

### Header

- **Logo & Title**: "Vrooli Autoheal" with the shield icon
- **Status Badge**: Overall system status (OK/Warning/Critical)
- **System Protection**: Shows if OS watchdog is active
- **Auto Toggle**: Enable/disable 30-second auto-refresh
- **Run Tick**: Manually trigger a health check cycle

### Tab Navigation

- **Dashboard**: Main health overview (default)
- **Trends**: Historical data and charts
- **Docs**: This documentation

### Summary Cards

Four cards showing check counts:

| Card | Color | Meaning |
|------|-------|---------|
| Total Checks | Gray | All registered health checks |
| Healthy | Green | Checks returning OK status |
| Warnings | Amber | Checks returning Warning status |
| Critical | Red | Checks returning Critical status |

### Health Checks Section

Checks are grouped by severity:

1. **Critical Issues** (red) - Problems requiring immediate attention
2. **Warnings** (amber) - Degraded but functional
3. **Healthy** (green) - Everything working correctly

Each check card shows:
- Check name and description
- Current status with color indicator
- Status message explaining the result
- Time since last check
- Duration of the check

### Sidebar

- **System Protection**: Detailed watchdog status
- **Uptime Stats**: 24h/7d/30d availability percentages
- **Platform Info**: Detected OS and capabilities
- **Last Updated**: Timestamp and refresh interval

### Events Timeline

Chronological list of recent events:
- Health check results
- Status changes (e.g., OK → Critical)
- Auto-heal actions taken

## Common Tasks

### Running a Manual Health Check

Click the **Run Tick** button in the header. This:
1. Executes all eligible health checks
2. Ignores interval restrictions (force mode)
3. Updates the dashboard with new results

### Investigating a Failed Check

1. Find the check in the "Critical Issues" or "Warnings" section
2. Read the status message for details
3. Click the check card to expand details
4. Check the Events Timeline for recent history
5. Go to Trends tab for historical patterns

### Checking Historical Data

1. Click the **Trends** tab
2. View uptime percentages over time
3. See health check trends
4. Identify patterns (e.g., daily failures at specific times)

### Understanding Platform Capabilities

The Platform Info card shows what Autoheal detected:

| Capability | Meaning |
|------------|---------|
| systemd | Can use systemd for watchdog |
| Docker | Docker daemon available |
| cloudflared | Cloudflare tunnel available |
| RDP | Remote desktop available |
| WSL | Running in Windows Subsystem for Linux |

## Status Colors

The color scheme is consistent throughout:

| Color | Status | Action Needed |
|-------|--------|---------------|
| Green | OK | None - healthy |
| Amber | Warning | Monitor - may need attention |
| Red | Critical | Immediate - investigate now |

## Keyboard Shortcuts

| Key | Action |
|-----|--------|
| `r` | Refresh data |
| `t` | Run tick |
| `1` | Go to Dashboard tab |
| `2` | Go to Trends tab |
| `3` | Go to Docs tab |

## Tips

1. **Enable Auto-Refresh**: Keep it on during active monitoring
2. **Use Run Tick**: After making changes, manually trigger to verify
3. **Check Trends**: Before investigating, see if it's a recurring issue
4. **Read Messages**: Check status messages often contain the solution
5. **Platform Context**: Some checks only run on certain platforms
