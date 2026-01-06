# VPS Setup Guide

Prepare your VPS for Scenario-to-Cloud deployments.

## Requirements

### Minimum Specifications

- **OS**: Ubuntu 22.04+ or Debian 11+
- **RAM**: 2GB minimum (4GB+ recommended)
- **Storage**: 20GB minimum
- **CPU**: 1 vCPU minimum

### Required Tools

The following tools must be available (installed automatically if missing):

- `curl`
- `git`
- `docker` (optional, for containerized resources)
- `systemd`

## SSH Configuration

### 1. Create SSH Key (if needed)

```bash
# On your local machine
ssh-keygen -t ed25519 -C "your-email@example.com"
```

### 2. Copy Key to VPS

```bash
ssh-copy-id -i ~/.ssh/id_ed25519.pub root@your-vps.com
```

### 3. Verify Access

```bash
ssh root@your-vps.com "echo 'SSH working!'"
```

## DNS Configuration

### Point Domain to VPS

Add an A record for your domain pointing to your VPS IP:

| Type | Name | Value |
|------|------|-------|
| A | app | 123.45.67.89 |
| A | @ | 123.45.67.89 |

Allow 5-30 minutes for DNS propagation.

### Verify DNS

```bash
dig +short app.yourdomain.com
# Should return your VPS IP
```

## Firewall Configuration

### Required Ports

| Port | Protocol | Purpose |
|------|----------|---------|
| 22 | TCP | SSH access |
| 80 | TCP | HTTP (Caddy redirect) |
| 443 | TCP | HTTPS |
| 3000-9999 | TCP | Scenario services (internal) |

### UFW Example

```bash
sudo ufw allow 22/tcp
sudo ufw allow 80/tcp
sudo ufw allow 443/tcp
sudo ufw enable
```

## Security Recommendations

### Disable Password Auth

Edit `etc/ssh/sshd_config` (path may vary by distro):

```
PasswordAuthentication no
PubkeyAuthentication yes
```

Restart SSH:

```bash
sudo systemctl restart sshd
```

### Keep System Updated

```bash
sudo apt update && sudo apt upgrade -y
```

### Enable Automatic Security Updates

```bash
sudo apt install unattended-upgrades
sudo dpkg-reconfigure -plow unattended-upgrades
```

## Verification Checklist

Before deploying, verify:

- [ ] SSH key authentication works
- [ ] Domain resolves to VPS IP
- [ ] Ports 80 and 443 are open
- [ ] Root or sudo access available
- [ ] At least 10GB free disk space

## Cloud Provider Guides

### DigitalOcean

1. Create a Droplet with Ubuntu 22.04
2. Add your SSH key during creation
3. Note the assigned IP address
4. Configure DNS in your domain registrar

### Hetzner

1. Create a Cloud server with Ubuntu 22.04
2. Add your SSH key in the setup
3. Configure firewall in Cloud Console
4. Note the assigned IP address

### Linode

1. Create a Linode with Ubuntu 22.04
2. Add SSH key via Cloud Manager
3. Configure firewall rules
4. Note the assigned IP address
