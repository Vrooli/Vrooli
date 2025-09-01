# SSH Key Setup Guide

This guide provides comprehensive SSH key setup for Vrooli deployment and CI/CD operations.

## Overview

SSH keys are used for:
- **Server Deployment**: Secure access to staging and production servers
- **CI/CD Pipeline**: Automated deployment from GitHub Actions
- **Development**: Secure Git operations and server management

## SSH Key Types

### Deployment Keys
Environment-specific keys for server access:
- `vrooli_staging_deploy` - For staging server access
- `vrooli_production_deploy` - For production server access

### Personal Keys
For individual developer access:
- `id_ed25519` - Default personal key for Git and server access

## Generating SSH Keys

### For Server Deployment

Generate environment-specific deployment keys:

```bash
# Generate staging deployment key
ssh-keygen -t ed25519 -f ~/.ssh/vrooli_staging_deploy -C "vrooli-staging-deploy"

# Generate production deployment key
ssh-keygen -t ed25519 -f ~/.ssh/vrooli_production_deploy -C "vrooli-production-deploy"

# Set proper permissions
chmod 600 ~/.ssh/vrooli_*_deploy
chmod 644 ~/.ssh/vrooli_*_deploy.pub
```

### For Personal Development

Generate personal SSH key if not already present:

```bash
# Generate personal key (if not exists)
if [ ! -f ~/.ssh/id_ed25519 ]; then
    ssh-keygen -t ed25519 -C "your-email@example.com"
fi

# Set proper permissions
chmod 600 ~/.ssh/id_ed25519
chmod 644 ~/.ssh/id_ed25519.pub
```

### Key Generation Options

```bash
# Ed25519 (recommended - modern, secure, fast)
ssh-keygen -t ed25519 -f ~/.ssh/keyname -C "comment"

# RSA (legacy compatibility, if Ed25519 not supported)
ssh-keygen -t rsa -b 4096 -f ~/.ssh/keyname -C "comment"

# With passphrase (more secure)
ssh-keygen -t ed25519 -f ~/.ssh/keyname -C "comment"
# Enter passphrase when prompted

# Without passphrase (for automation)
ssh-keygen -t ed25519 -f ~/.ssh/keyname -C "comment" -N ""
```

## Adding Keys to Servers

### Manual Key Installation

```bash
# Copy public key to server (interactive)
ssh-copy-id -i ~/.ssh/vrooli_staging_deploy.pub user@staging-server-ip

# Copy public key to server (manual)
cat ~/.ssh/vrooli_staging_deploy.pub | ssh user@server-ip "mkdir -p ~/.ssh && cat >> ~/.ssh/authorized_keys"

# Or upload via server provider interface
cat ~/.ssh/vrooli_staging_deploy.pub
# Copy output and paste into server provider's SSH key field
```

### Automated Key Installation

```bash
# Using the provided authorize_key.sh script
ssh user@server-ip 'bash -s' < scripts/lib/network/authorize_key.sh
# Then paste the public key content and press Ctrl+D
```

### Server-Side Key Verification

```bash
# On the server, verify authorized keys
cat ~/.ssh/authorized_keys
ls -la ~/.ssh/

# Check permissions (should be 700 for .ssh, 600 for authorized_keys)
chmod 700 ~/.ssh
chmod 600 ~/.ssh/authorized_keys
```

## SSH Configuration

### Client Configuration

Create or update `~/.ssh/config` for easier access:

```bash
# Staging server configuration
Host vrooli-staging
    HostName staging-server-ip
    User deploy
    IdentityFile ~/.ssh/vrooli_staging_deploy
    IdentitiesOnly yes
    Port 22
    ServerAliveInterval 60
    ServerAliveCountMax 3

# Production server configuration  
Host vrooli-production
    HostName production-server-ip
    User deploy
    IdentityFile ~/.ssh/vrooli_production_deploy
    IdentitiesOnly yes
    Port 22
    ServerAliveInterval 60
    ServerAliveCountMax 3

# Default settings for all Vrooli hosts
Host vrooli-*
    StrictHostKeyChecking no
    UserKnownHostsFile /dev/null
    LogLevel ERROR
```

### Setting File Permissions

```bash
# Set correct permissions for SSH config
chmod 600 ~/.ssh/config
chmod 700 ~/.ssh

# Verify permissions
ls -la ~/.ssh/
```

## Testing SSH Connections

### Basic Connection Test

```bash
# Test staging connection
ssh vrooli-staging "echo 'SSH connection to staging successful'"

# Test production connection
ssh vrooli-production "echo 'SSH connection to production successful'"

# Test with specific key
ssh -i ~/.ssh/vrooli_staging_deploy user@server-ip "echo 'Connection test'"
```

### Detailed Connection Debug

```bash
# Debug SSH connection issues
ssh -vvv vrooli-staging

# Test specific components
ssh -o PasswordAuthentication=no vrooli-staging "whoami"
ssh -o PubkeyAuthentication=yes vrooli-staging "whoami"
```

### Connection Verification Script

```bash
#!/bin/bash
# SSH connection verification script

echo "=== SSH Connection Verification ==="

# Test staging connection
if ssh -o ConnectTimeout=10 vrooli-staging "exit 0" 2>/dev/null; then
    echo "✅ Staging server connection: SUCCESS"
else
    echo "❌ Staging server connection: FAILED"
fi

# Test production connection
if ssh -o ConnectTimeout=10 vrooli-production "exit 0" 2>/dev/null; then
    echo "✅ Production server connection: SUCCESS"
else
    echo "❌ Production server connection: FAILED"
fi

echo "=== Verification Complete ==="
```

## CI/CD Integration

### GitHub Actions Setup

For automated deployment, add SSH private keys to GitHub Secrets:

1. **Copy Private Key Content**:
```bash
# Copy staging private key
cat ~/.ssh/vrooli_staging_deploy
# Copy entire output including headers

# Copy production private key  
cat ~/.ssh/vrooli_production_deploy
# Copy entire output including headers
```

2. **Add to GitHub Secrets**:
   - Go to Repository Settings → Secrets and Variables → Actions
   - Add secrets:
     - `VPS_SSH_PRIVATE_KEY` (for the target environment)
     - `VPS_DEPLOY_USER` (username, typically `deploy` or `root`)
     - `VPS_DEPLOY_HOST` (server IP or hostname)

3. **Environment-Specific Secrets**:
```bash
# Staging environment secrets
VPS_SSH_PRIVATE_KEY=<staging-private-key-content>
VPS_DEPLOY_USER=deploy
VPS_DEPLOY_HOST=staging-server-ip

# Production environment secrets
VPS_SSH_PRIVATE_KEY=<production-private-key-content>
VPS_DEPLOY_USER=deploy
VPS_DEPLOY_HOST=production-server-ip
```

### Key Format for CI/CD

Ensure private keys are in OpenSSH format:

```bash
# Check key format (should start with -----BEGIN OPENSSH PRIVATE KEY-----)
head -1 ~/.ssh/vrooli_staging_deploy

# Convert RSA to OpenSSH format if needed
ssh-keygen -p -m OpenSSH -f ~/.ssh/vrooli_staging_deploy
```

## Security Best Practices

### Key Management

```bash
# Use different keys for different environments
# Never reuse production keys for staging

# Rotate keys regularly (quarterly recommended)
# Keep old keys until new ones are confirmed working

# Use passphrases for personal keys
# Use passwordless keys only for automation
```

### Server Security

```bash
# Disable password authentication (server-side)
sudo nano /etc/ssh/sshd_config
# Set: PasswordAuthentication no
# Set: PubkeyAuthentication yes
# Set: PermitRootLogin no

# Restart SSH service
sudo systemctl restart sshd
```

### Access Control

```bash
# Limit SSH access to specific users
sudo nano /etc/ssh/sshd_config
# Add: AllowUsers deploy

# Use fail2ban for additional protection
sudo apt install fail2ban

# Monitor SSH access logs
sudo tail -f /var/log/auth.log
```

## Troubleshooting SSH Issues

### Common Connection Problems

#### Permission Denied

```bash
# Check key permissions
ls -la ~/.ssh/
chmod 600 ~/.ssh/private_key
chmod 644 ~/.ssh/public_key.pub

# Verify key is being offered
ssh -v user@server 2>&1 | grep "Offering public key"

# Test key authentication specifically
ssh -o PreferredAuthentications=publickey user@server
```

#### Connection Refused

```bash
# Check if SSH service is running
ssh user@server -p 22  # Try default port
ssh user@server -p 2222  # Try alternate port

# Check server firewall
sudo ufw status
sudo iptables -L
```

#### Host Key Verification Failed

```bash
# Remove old host key
ssh-keygen -R server-hostname
ssh-keygen -R server-ip

# Or disable host key checking (less secure)
ssh -o StrictHostKeyChecking=no user@server
```

### Debug Connection Issues

```bash
# Verbose SSH debugging
ssh -vvv user@server

# Test from server side
sudo tail -f /var/log/auth.log

# Check SSH daemon configuration
sudo sshd -T | grep -i key
```

### Key Authentication Failures

```bash
# Verify public key is in authorized_keys
ssh user@server "cat ~/.ssh/authorized_keys | grep 'comment-from-your-key'"

# Check authorized_keys permissions
ssh user@server "ls -la ~/.ssh/"

# Regenerate authorized_keys if corrupted
cat ~/.ssh/public_key.pub | ssh user@server "cat > ~/.ssh/authorized_keys"
```

## Key Rotation Procedure

### Rotating Deployment Keys

```bash
# 1. Generate new key pair
ssh-keygen -t ed25519 -f ~/.ssh/vrooli_staging_deploy_new -C "vrooli-staging-deploy-new"

# 2. Add new public key to server
ssh-copy-id -i ~/.ssh/vrooli_staging_deploy_new.pub user@server

# 3. Test new key
ssh -i ~/.ssh/vrooli_staging_deploy_new user@server "echo 'New key works'"

# 4. Update SSH config and CI/CD secrets
# 5. Remove old public key from server
ssh user@server "sed -i '/old-key-comment/d' ~/.ssh/authorized_keys"

# 6. Remove old local keys
rm ~/.ssh/vrooli_staging_deploy ~/.ssh/vrooli_staging_deploy.pub
mv ~/.ssh/vrooli_staging_deploy_new ~/.ssh/vrooli_staging_deploy
mv ~/.ssh/vrooli_staging_deploy_new.pub ~/.ssh/vrooli_staging_deploy.pub
```

## Integration with Deployment Scripts

The Vrooli deployment scripts automatically use SSH keys configured in this guide:

- **CI/CD Pipeline**: Uses keys from GitHub Secrets
- **Manual Deployment**: Uses keys from SSH config
- **Development**: Uses personal keys for Git and server access

For more information:
- **CI/CD Setup**: See [CI/CD Pipeline Guide](../build-deploy/ci-cd.md)
- **Server Deployment**: See [Server Deployment Guide](../operations/server-deployment.md)
- **Troubleshooting**: See [Troubleshooting Guide](../troubleshooting.md#ssh-deployment-issues) 