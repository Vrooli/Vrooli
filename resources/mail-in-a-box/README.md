# Mail-in-a-Box Resource

All-in-one email server with webmail, contacts, calendar, and file storage capabilities for Vrooli scenarios.

## Features

- **Complete Email Server**: SMTP, IMAP, POP3 with TLS support
- **Webmail Interface**: Roundcube for web-based email access (port 8880)
- **Calendar & Contacts**: CalDAV and CardDAV support (port 5232)
- **Spam Protection**: SpamAssassin, greylisting, and fail2ban
- **Auto-configuration**: Automatic email client configuration
- **Multi-domain Support**: Host email for multiple domains
- **Web Admin Panel**: Easy management interface

## Quick Start

```bash
# Install and start Mail-in-a-Box
resource-mail-in-a-box install
resource-mail-in-a-box start

# Check status
resource-mail-in-a-box status

# Add email account
resource-mail-in-a-box add-account user@example.com SecurePassword123

# Add email alias
resource-mail-in-a-box add-alias info@example.com user@example.com
```

## Configuration

Environment variables can be set to customize Mail-in-a-Box:

- `MAILINABOX_PRIMARY_HOSTNAME`: Primary mail server hostname (default: mail.local)
- `MAILINABOX_ADMIN_EMAIL`: Administrator email (default: admin@mail.local)
- `MAILINABOX_ADMIN_PASSWORD`: Administrator password
- `MAILINABOX_BIND_ADDRESS`: Bind address for services (default: 127.0.0.1)
- `MAILINABOX_POSTMASTER_ALIAS`: Forwarding target for postmaster@ and abuse@ aliases (defaults to the admin mailbox)
- `MAILINABOX_DEFAULT_MAILBOX`: Optional mailbox (user@domain) to create on first start
- `MAILINABOX_DEFAULT_MAILBOX_PASSWORD`: Password for the optional mailbox

> **Vault integration:** Use the Secrets Manager scenario (`vrooli scenario run secrets-manager`) to populate these values in Vault. The resource automatically loads the secrets, regenerates the docker-mailserver config, and provisions the admin + optional mailboxes on startup.

## Injection Support

Mail-in-a-Box supports injecting email configurations:

### CSV Format
```csv
user1@example.com,Password123
user2@example.com,Password456
```

### JSON Format
```json
[
  {"email": "user1@example.com", "password": "Password123"},
  {"email": "user2@example.com", "password": "Password456"}
]
```

### Usage
```bash
resource-mail-in-a-box inject accounts.csv
resource-mail-in-a-box inject config.json
```

## Access Points

- **Admin Panel**: https://localhost:8543/admin
- **Webmail**: https://localhost/mail
- **SMTP**: localhost:25 (or 587 for submission)
- **IMAP**: localhost:143 (or 993 for IMAPS)
- **POP3**: localhost:110 (or 995 for POP3S)

## Integration with Scenarios

Mail-in-a-Box can be used in scenarios for:
- Email automation and testing
- Newsletter distribution
- Customer communication workflows
- Alert and notification systems
- Multi-user collaboration scenarios

## Security Notes

- Default installation uses self-signed certificates
- Enable TLS for all connections in production
- Configure SPF, DKIM, and DMARC for domain reputation
- Regular backups are stored in `~/.mailinabox/backup`
