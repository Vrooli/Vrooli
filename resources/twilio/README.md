# Twilio Resource

Cloud communications platform for SMS, voice, and video capabilities.

## Overview

The Twilio resource provides integration with Twilio's cloud communications API, enabling:
- SMS messaging (send/receive)
- Phone number management  
- Message templates
- Delivery tracking
- Secure credential management via Vault

## Quick Start

### Installation

```bash
# Install Twilio CLI and dependencies
vrooli resource twilio manage install

# Configure credentials (recommended: via Vault)
resource-vault secrets init twilio

# Or configure manually (less secure)
# Edit ~/.vrooli/twilio-credentials.json with your account details
```

### Basic Usage

```bash
# Start the resource (validates configuration)
vrooli resource twilio manage start

# Send an SMS
resource-twilio content send-sms "+1234567890" "Hello from Vrooli!"

# Check status
vrooli resource twilio status

# Run tests
vrooli resource twilio test smoke
```

## Configuration

### Secure Configuration (Recommended)

Twilio integrates with Vault for secure credential storage:

```bash
# Initialize Twilio secrets in Vault
resource-vault secrets init twilio

# You'll be prompted for:
# - Account SID (from https://console.twilio.com)
# - Auth Token (from https://console.twilio.com)
# - Default phone number (optional)
```

### Manual Configuration

If Vault is not available, credentials can be stored in `~/.vrooli/twilio-credentials.json`:

```json
{
  "account_sid": "ACxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx",
  "auth_token": "xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx",
  "default_from_number": "+1234567890"
}
```

**Warning**: File-based configuration is less secure. Use Vault when possible.

## Features

### SMS Messaging

Send SMS messages with delivery confirmation:

```bash
# Send with default from number
resource-twilio content send-sms "+1234567890" "Your message here"

# Send with specific from number
resource-twilio content send-sms "+1234567890" "Your message" "+0987654321"
```

### Bulk SMS Messaging

Send messages to multiple recipients efficiently:

```bash
# Send to multiple recipients
resource-twilio content send-bulk "Your message" "+1234567890" "+0987654321" "+5555555555"

# Send from CSV file (phone_number,message format)
resource-twilio content send-from-file /path/to/recipients.csv

# CSV file format:
# phone_number,message
# +1234567890,Hello John!
# +0987654321,Hello Jane!
```

**Features:**
- Automatic rate limiting (100ms between messages)
- Progress tracking with success/failure counts
- Test mode for safe testing without sending real messages
- CSV batch processing for large recipient lists

### Phone Number Management

List and configure Twilio phone numbers:

```bash
# List phone numbers
resource-twilio content numbers

# View phone numbers configuration
cat ~/.vrooli/twilio/phone-numbers.json
```

### Testing

Comprehensive test suite for validation:

```bash
# Quick smoke test (<30s)
vrooli resource twilio test smoke

# Full integration tests
vrooli resource twilio test integration

# Unit tests
vrooli resource twilio test unit

# All tests
vrooli resource twilio test all
```

## Integration with Other Resources

### N8n Workflows

Twilio can be triggered from N8n workflows for automated messaging:

```javascript
// N8n webhook node configuration
{
  "method": "POST",
  "url": "http://localhost:8080/twilio/send",
  "body": {
    "to": "+1234567890",
    "message": "Automated alert from N8n"
  }
}
```

### Scenarios

Any scenario can use Twilio for notifications:

```bash
# From a scenario script
resource-twilio content send-sms "$USER_PHONE" "Task completed successfully"
```

## API Reference

### CLI Commands

| Command | Description |
|---------|-------------|
| `manage install` | Install Twilio CLI and dependencies |
| `manage start` | Start Twilio (validate configuration) |
| `manage stop` | Stop Twilio service |
| `manage restart` | Restart Twilio service |
| `test smoke` | Run quick validation tests |
| `test all` | Run all test suites |
| `content send-sms` | Send an SMS message |
| `content numbers` | List phone numbers |
| `status` | Show detailed status |
| `info` | Show configuration information |

### Environment Variables

| Variable | Description | Required |
|----------|-------------|----------|
| `TWILIO_ACCOUNT_SID` | Twilio Account SID | Yes |
| `TWILIO_AUTH_TOKEN` | Twilio Auth Token | Yes |
| `TWILIO_FROM_NUMBER` | Default from number | No |

## Troubleshooting

### Common Issues

**Issue**: "Twilio credentials not configured"
```bash
# Solution: Configure via Vault
resource-vault secrets init twilio

# Or check credentials file
cat ~/.vrooli/twilio-credentials.json
```

**Issue**: "Cannot connect to Twilio API"
```bash
# Check credentials are valid
vrooli resource twilio test smoke

# Test with Twilio CLI directly
export TWILIO_ACCOUNT_SID="your_sid"
export TWILIO_AUTH_TOKEN="your_token"
twilio api:core:accounts:list --limit 1
```

**Issue**: "SMS not sending"
```bash
# Verify phone number format (must include country code)
# Correct: +1234567890
# Wrong: 234-567-8900

# Check account has SMS capability
# Visit https://console.twilio.com to verify
```

## Security Considerations

1. **Always use Vault for production credentials**
2. **Never commit credentials to version control**
3. **Use test credentials for development**
4. **Rotate auth tokens regularly**
5. **Monitor usage for anomalies**

## Pricing

Twilio charges per message sent. Current rates:
- SMS: ~$0.0075 per message (US)
- International rates vary
- See https://www.twilio.com/pricing for details

## Support

- Twilio Documentation: https://www.twilio.com/docs
- Console: https://console.twilio.com
- Status: https://status.twilio.com

## License

This resource integrates with Twilio's commercial API service. Twilio usage is subject to their terms of service and pricing.