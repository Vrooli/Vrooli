# Twilio Resource

SMS, voice, and messaging automation via Twilio API.

## Overview
The Twilio resource provides programmatic SMS, voice, and messaging capabilities through Twilio's comprehensive API. It enables scenarios to send/receive SMS messages, make voice calls, and manage phone numbers.

## Installation
```bash
vrooli resource twilio install
```

## Configuration
1. Get your Twilio credentials from https://console.twilio.com
2. Provision the credentials through the canonical Vrooli credential authority:
```bash
printf '%s' "$TWILIO_ACCOUNT_SID" | vrooli credentials provision \
  --identity vrooli/twilio --field account-sid
printf '%s' "$TWILIO_AUTH_TOKEN" | vrooli credentials provision \
  --identity vrooli/twilio --field auth-token
```

3. Configure phone numbers in `~/.config/vrooli/resources/twilio/phone-numbers.json`:
```json
{
  "numbers": [
    {
      "number": "+1234567890",
      "friendly_name": "Main Line"
    }
  ]
}
```

## Usage

### Check Status
```bash
vrooli resource twilio status
vrooli resource twilio status --json
```

### Send SMS
```bash
# With default from number
vrooli resource twilio send-sms "+1234567890" "Hello from Vrooli!"

# With specific from number
vrooli resource twilio send-sms "+1234567890" "Hello!" "+19876543210"
```

### Inject Data
```bash
# Inject phone numbers
vrooli resource twilio inject phone-numbers.json

# Inject workflow
vrooli resource twilio inject workflow.json
```

### Start/Stop
```bash
# Start monitor (validates credentials)
vrooli resource twilio start

# Stop monitor
vrooli resource twilio stop
```

## Workflow Format
Workflows define automated SMS/voice operations:
```json
{
  "workflow": {
    "name": "customer-notification",
    "description": "Send customer notifications",
    "trigger": "api",
    "steps": [
      {
        "id": "send_sms",
        "type": "send_sms",
        "to": "{{customer_phone}}",
        "from": "{{sender}}",
        "body": "{{message}}"
      }
    ]
  }
}
```

## Integration with Scenarios
Scenarios can use Twilio for:
- Customer notifications
- Two-factor authentication
- Appointment reminders
- Alert systems
- Marketing campaigns
- Voice call automation

## Best Practices
1. Store credentials through the Vrooli credential authority and keep recovery bundles separate from the host
2. Use messaging services for bulk SMS
3. Implement rate limiting for high-volume sends
4. Handle delivery status callbacks
5. Validate phone numbers before sending
6. Use templates for consistent messaging

## Troubleshooting
- **Auth Error**: Check account_sid and auth_token
- **Invalid Number**: Ensure number is E.164 format (+1234567890)
- **Rate Limit**: Implement queuing for bulk sends
- **No From Number**: Configure default_from_number in credentials
