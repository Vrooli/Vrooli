# 📱 Pushover Resource

Real-time push notifications for Android, iPhone, iPad, and Desktop applications.

## Overview

Pushover is a cloud-based notification service that enables Vrooli to send real-time push notifications to multiple devices. This resource provides a simple API for scenarios to send alerts, status updates, and notifications to users across all their devices.

## Benefits for Vrooli

- **Real-time Alerts**: Instant notifications for scenario events, errors, or completions
- **Multi-device Support**: Reach users on Android, iOS, desktop, and smartwatches
- **Priority Levels**: From silent notifications to emergency alerts with acknowledgment
- **Rich Content**: Support for HTML, monospace text, images, and clickable URLs
- **Templates**: Pre-configured notification templates for common scenario events
- **Low Latency**: Sub-second delivery for time-critical notifications

## Use Cases

### Scenario Notifications
- Task completion alerts
- Error notifications with stack traces
- Progress updates for long-running operations
- User action required prompts

### System Monitoring
- Resource health alerts
- Service outage notifications
- Performance threshold warnings
- Backup completion confirmations

### Business Automation
- Order confirmations
- Payment notifications
- Customer inquiry alerts
- Inventory level warnings

## Quick Start

### Installation
```bash
# Install the resource
vrooli resource install pushover

# Or via CLI
resource-pushover install
```

### Configuration

Pushover requires API credentials (available from https://pushover.net):
1. **App Token**: Identifies your application
2. **User Key**: Identifies the recipient

Configure via:
```bash
# Interactive setup
resource-pushover configure

# Or manually create credentials file
echo '{
  "app_token": "YOUR_APP_TOKEN",
  "user_key": "YOUR_USER_KEY"
}' > ~/.vrooli/pushover-credentials.json
```

### Basic Usage

```bash
# Send a simple notification
resource-pushover send -m "Task completed successfully"

# Send with title and priority
resource-pushover send -m "Server is down!" -t "Alert" -p 2

# Use a template
resource-pushover send --template task-complete -m "Backup finished"
```

## CLI Commands

### Core Commands
- `install` - Install dependencies and configure
- `status` - Check service status
- `start` - Activate the service
- `stop` - Deactivate the service
- `send` - Send a notification

### Send Options
- `-m, --message` - Notification message (required)
- `-t, --title` - Notification title (default: "Vrooli Notification")
- `-p, --priority` - Priority level (-2 to 2)
- `-s, --sound` - Notification sound
- `--template` - Use a predefined template

## Priority Levels

| Priority | Description | Behavior |
|----------|-------------|----------|
| -2 | Lowest | No sound/vibration, no alert |
| -1 | Low | No sound/vibration |
| 0 | Normal | Default sound and vibration |
| 1 | High | Bypasses quiet hours |
| 2 | Emergency | Requires acknowledgment, repeats every 30s |

## Notification Sounds

21 built-in sounds available:
- `pushover` (default)
- `bike`, `bugle`, `cashregister`, `classical`
- `cosmic`, `falling`, `gamelan`, `incoming`
- `intermission`, `magic`, `mechanical`, `pianobar`
- `siren`, `spacealarm`, `tugboat`, `alien`
- `climb`, `persistent`, `echo`, `updown`

## Templates

### Creating Templates
Create reusable notification templates in `~/.vrooli/pushover/templates/`:

```json
{
  "title": "Task Complete",
  "priority": 0,
  "sound": "magic",
  "message": "Your task has completed successfully"
}
```

### Injecting Templates
You can inject custom templates from any location:
```bash
# Inject a template file
resource-pushover inject /path/to/template.json

# Example: Create and inject a custom template
cat > /tmp/alert.json <<EOF
{
  "title": "System Alert",
  "message": "{{component}} status: {{status}}",
  "priority": 1,
  "sound": "siren"
}
EOF
resource-pushover inject /tmp/alert.json
```

### Using Templates
```bash
# Use a template by name
resource-pushover send --template task-complete

# Override template message
resource-pushover send --template alert -m "Database backup complete"
```

### Pre-installed Templates
- `success.json` - For successful task completions
- `error.json` - For error notifications
- `workflow.json` - For workflow status updates

## Integration with Scenarios

### Python Example
```python
import subprocess
import json

def send_notification(message, title="Scenario Update", priority=0):
    """Send a push notification via Pushover"""
    cmd = [
        "resource-pushover", "send",
        "-m", message,
        "-t", title,
        "-p", str(priority)
    ]
    result = subprocess.run(cmd, capture_output=True, text=True)
    return result.returncode == 0

# Usage
send_notification("Processing complete", "Success", 1)
```

### Workflow Integration
Pushover can be integrated into n8n, Node-RED, and other workflow tools via:
- Direct CLI execution nodes
- HTTP Request nodes to Pushover API
- Custom function nodes using the resource CLI

## Resource Health

The resource reports different health states:
- **Installed**: Dependencies present, awaiting credentials
- **Running**: Configured and ready to send notifications
- **Healthy**: Successfully connected to Pushover API

## Troubleshooting

### Common Issues

**No credentials**
- Status shows "installed" but not "running"
- Solution: Configure API credentials

**Network errors**
- Check internet connectivity
- Verify firewall allows HTTPS to api.pushover.net

**Rate limits**
- Free tier: 10,000 messages/month
- Solution: Implement notification batching or upgrade plan

## Technical Details

- **API Endpoint**: https://api.pushover.net/1/messages.json
- **Dependencies**: Python 3, requests library
- **Data Storage**: `~/.vrooli/pushover/`
- **Credentials**: Stored locally or in Vault
- **Rate Limits**: 10,000 messages/month (free tier)

## Security

- API credentials stored securely in local filesystem
- Optional Vault integration for credential management
- HTTPS-only communication with Pushover API
- No message content stored locally

## Related Resources

- **Twilio**: SMS notifications (higher cost, broader reach)
- **n8n/Node-RED**: Workflow automation with notification triggers
- **Huginn**: Event-based notification automation

## Support

- Pushover Documentation: https://pushover.net/api
- Vrooli Integration Issues: Create issue in Vrooli repository