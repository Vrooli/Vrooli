# Keycloak Resource

Enterprise-grade identity and access management with SSO, OIDC, SAML, and social login support.

## Overview

Keycloak provides centralized authentication and authorization for Vrooli scenarios requiring:
- Single Sign-On (SSO)
- Multi-tenant authentication
- OAuth2/OIDC integration
- SAML support
- Social login providers
- User federation (LDAP/AD)

## Quick Start

```bash
# Install and start
vrooli resource keycloak manage install
vrooli resource keycloak manage start --wait

# Check status
vrooli resource keycloak status

# Access admin console
# http://localhost:8070/admin
# Default: admin/admin

# Run tests
vrooli resource keycloak test all

# Run security validation
vrooli resource keycloak test security

# Run social provider end-to-end tests
vrooli resource keycloak test social-e2e
```

## Architecture

- **Port**: 8070 (HTTP), 8443 (HTTPS - when configured)
- **Database**: PostgreSQL (production), H2 (fallback)
- **Container**: keycloak/keycloak:latest
- **Network**: vrooli-network

## Content Management

```bash
# Create a realm
vrooli resource keycloak content add --file realm.json --type realm

# Add users to a realm
vrooli resource keycloak content add --file users.json --type user --name test-realm

# Add OAuth2 clients
vrooli resource keycloak content add --file client.json --type client --name test-realm

# List all realms and their content
vrooli resource keycloak content list

# Export a realm configuration
vrooli resource keycloak content get --name test-realm --output realm-export.json

# Remove a realm
vrooli resource keycloak content remove --name test-realm
```

## Social Login Providers

```bash
# Add GitHub provider
vrooli resource keycloak social add-github \
  --client-id <your-github-client-id> \
  --client-secret <your-github-client-secret> \
  --realm master

# Add Google provider
vrooli resource keycloak social add-google \
  --client-id <your-google-client-id> \
  --client-secret <your-google-client-secret> \
  --realm master

# List configured providers
vrooli resource keycloak social list --realm master

# Test provider configuration
vrooli resource keycloak social test --alias github --realm master

# Remove a provider
vrooli resource keycloak social remove --alias github --realm master
```

## Multi-Realm Tenant Isolation

```bash
# Create an isolated tenant realm
vrooli resource keycloak realm create-tenant "Acme Corp" admin@acme.com SecurePass123!

# List all tenant realms
vrooli resource keycloak realm list-tenants

# Get tenant details
vrooli resource keycloak realm get-tenant acme-corp

# Export tenant configuration
vrooli resource keycloak realm export-tenant acme-corp /tmp/acme-backup.json

# Delete a tenant realm
vrooli resource keycloak realm delete-tenant acme-corp
```

Multi-realm features:
- Complete tenant isolation with separate user databases
- Tenant-specific roles and groups
- Default OAuth2 client per tenant
- Tenant admin user creation
- Export/import tenant configurations

## LDAP/AD Federation

```bash
# Add LDAP provider
vrooli resource keycloak ldap add \
  --url ldap://ldap.example.com:389 \
  --users-dn ou=users,dc=example,dc=com \
  --bind-dn cn=admin,dc=example,dc=com \
  --bind-password admin-password \
  --name my-ldap \
  --realm master

# Add Active Directory provider
vrooli resource keycloak ldap add \
  --url ldap://ad.example.com:389 \
  --users-dn CN=Users,DC=example,DC=com \
  --bind-dn admin@example.com \
  --bind-password admin-password \
  --name my-ad \
  --type ad \
  --realm master

# List configured LDAP/AD providers
vrooli resource keycloak ldap list --realm master

# Test connection
vrooli resource keycloak ldap test --name my-ldap --realm master

# Sync users (incremental)
vrooli resource keycloak ldap sync --name my-ldap --realm master

# Sync users (full)
vrooli resource keycloak ldap sync --name my-ldap --realm master --full

# Remove provider
vrooli resource keycloak ldap remove --name my-ldap --realm master
```

## Theme Customization

```bash
# Create a custom theme
vrooli resource keycloak theme create my-brand-theme

# List available themes
vrooli resource keycloak theme list

# Deploy theme to container
vrooli resource keycloak theme deploy my-brand-theme

# Apply theme to a realm
vrooli resource keycloak theme apply test-realm my-brand-theme login

# Customize theme properties
vrooli resource keycloak theme customize my-brand-theme primary-color "#ff6600"
vrooli resource keycloak theme customize my-brand-theme logo /path/to/logo.png

# Export theme for sharing
vrooli resource keycloak theme export my-brand-theme /tmp/theme-export.tar.gz

# Import theme from archive
vrooli resource keycloak theme import /tmp/theme-export.tar.gz

# Remove a theme
vrooli resource keycloak theme remove my-brand-theme
```

Theme features:
- Custom CSS styling and gradients
- Logo and branding customization
- Multiple theme types (login, account, admin, email)
- Export/import for sharing themes
- Hot deployment to running container

## Backup and Restore

```bash
# Create backup of a realm
vrooli resource keycloak backup create master

# List available backups
vrooli resource keycloak backup list

# Restore from backup
vrooli resource keycloak backup restore keycloak_master_20250914.json.gz

# Clean up old backups (with rotation policies)
vrooli resource keycloak backup cleanup  # Uses retention/max/min policies

# Schedule automatic backups
vrooli resource keycloak backup schedule master "0 2 * * *"  # 2 AM daily
```

Backup features:
- Compressed backups (gzip)
- Intelligent rotation policies
- Configurable retention (KEYCLOAK_BACKUP_RETENTION_DAYS)
- Maximum backup limit (KEYCLOAK_BACKUP_MAX_COUNT)
- Minimum backup guarantee (KEYCLOAK_BACKUP_MIN_COUNT)
- Scheduled backups via cron

## Performance Monitoring

```bash
# Check health status
vrooli resource keycloak monitor health

# View performance metrics
vrooli resource keycloak monitor metrics

# Analyze performance
vrooli resource keycloak monitor performance

# View realm statistics
vrooli resource keycloak monitor realms

# Full monitoring dashboard (includes history)
vrooli resource keycloak monitor dashboard
```

Monitoring features:
- Real-time health checks with response times
- JVM and container metrics
- Connection statistics
- Historical metrics with averaging
- Performance benchmarking (token generation <100ms)
- Metrics persistence and trend analysis

## Webhook Integration

```bash
# List available webhook event types
vrooli resource keycloak webhook list-events

# Register a webhook for login events
vrooli resource keycloak webhook register master LOGIN https://your-app.com/webhook secret123

# Configure multiple events for a realm
vrooli resource keycloak webhook configure-events master LOGIN,LOGOUT,REGISTER

# List configured webhooks
vrooli resource keycloak webhook list master

# Test webhook connectivity
vrooli resource keycloak webhook test master https://your-app.com/webhook

# View event history
vrooli resource keycloak webhook history master 20

# Configure retry policy
vrooli resource keycloak webhook configure-retry master webhook-name 5 2000

# Remove a webhook
vrooli resource keycloak webhook remove master webhook-name
```

Webhook features:
- 18+ event types (LOGIN, LOGOUT, REGISTER, UPDATE_PROFILE, etc.)
- Secret-based signature validation
- Configurable retry policies with exponential backoff
- Event filtering per realm
- Event history tracking
- Connectivity testing

## TLS/HTTPS Configuration

```bash
# Generate self-signed certificate (for development)
vrooli resource keycloak tls generate

# Import existing certificate
vrooli resource keycloak tls import /path/to/cert.pem /path/to/key.pem

# Enable HTTPS
vrooli resource keycloak tls enable

# Check certificate expiry
vrooli resource keycloak tls check

# Show certificate details
vrooli resource keycloak tls show

# Renew certificate
vrooli resource keycloak tls renew

# Disable HTTPS (revert to HTTP only)
vrooli resource keycloak tls disable
```

### Let's Encrypt Certificate Automation

```bash
# Initialize Let's Encrypt with email
vrooli resource keycloak letsencrypt init "admin@example.com"

# Request certificate for domain
vrooli resource keycloak letsencrypt request "yourdomain.com" "admin@example.com"

# Renew certificates
vrooli resource keycloak letsencrypt renew

# Setup automatic renewal (daily/weekly/monthly)
vrooli resource keycloak letsencrypt auto-renew daily
vrooli resource keycloak letsencrypt auto-renew weekly
vrooli resource keycloak letsencrypt auto-renew monthly

# Disable automatic renewal
vrooli resource keycloak letsencrypt disable-auto-renew

# Check certificate status
vrooli resource keycloak letsencrypt status

# Revoke certificate
vrooli resource keycloak letsencrypt revoke "yourdomain.com" "keycompromise"

# Test ACME challenge setup
vrooli resource keycloak letsencrypt test 8899
```

## Multi-Factor Authentication (MFA)

```bash
# Enable MFA for a realm (totp/webauthn/sms)
vrooli resource keycloak mfa enable master totp

# Configure MFA policy (always/conditional/optional)
vrooli resource keycloak mfa configure master conditional

# Enable MFA for specific user
vrooli resource keycloak mfa enable-user master john.doe

# Check MFA status
vrooli resource keycloak mfa status master

# List users with MFA status
vrooli resource keycloak mfa list-users master

# Disable MFA
vrooli resource keycloak mfa disable master
```

## Password Policy Management

```bash
# Set custom password policy
vrooli resource keycloak password-policy set master \
  --length 12 \
  --digits 2 \
  --uppercase 2 \
  --lowercase 2 \
  --special 1 \
  --not-username \
  --history 5

# Apply preset policy (basic/moderate/strong/paranoid)
vrooli resource keycloak password-policy preset master strong

# View current policy
vrooli resource keycloak password-policy get master

# Validate password against policy
vrooli resource keycloak password-policy validate master "MyP@ssw0rd123"

# Force password reset for user(s)
vrooli resource keycloak password-policy force-reset master john.doe

# Clear password policy
vrooli resource keycloak password-policy clear master
```

## Production Configuration

For production deployments, set these environment variables:

```bash
# Use secure admin password
export KEYCLOAK_ADMIN_PASSWORD="your-secure-password"

# Enable production mode
export KEYCLOAK_ENV="production"

# Load production settings
source resources/keycloak/config/production.sh

# Start with production configuration
vrooli resource keycloak manage start
```

Production settings include:
- HTTPS-only mode with strict hostname checking
- PostgreSQL database (required)
- Increased JVM memory allocation
- Brute force protection enabled
- Session timeout configuration
- Rate limiting enabled
- Automated backup scheduling

## Security Testing

Run security validation tests:
```bash
vrooli resource keycloak test security
```

Validates:
- Admin password strength
- SSL/TLS configuration
- Security headers
- Realm security settings
- Database security
- Port exposure

## Documentation

- [Installation Guide](docs/installation.md)
- [Configuration](docs/configuration.md)
- [Integration Guide](docs/integration.md)
- [API Reference](docs/api.md)
- [Production Setup](config/production.sh)

## Use Cases

1. **Multi-tenant SaaS**: Isolate customer data with realms
2. **Enterprise SSO**: Connect LDAP/AD directories
3. **API Security**: OAuth2 token validation
4. **B2B Scenarios**: SAML federation
5. **Social Login**: GitHub, Google, Facebook integration