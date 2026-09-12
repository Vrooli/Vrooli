# VPS Setup Guide

Prepare your VPS for Scenario-to-Cloud deployments.

## Requirements

### Minimum Specifications

- **OS**: Ubuntu LTS (24.04 recommended; 22.04/20.04 compatibility mode)
- **RAM**: 2GB minimum (4GB+ recommended)
- **Storage**: 20GB minimum
- **CPU**: 1 vCPU minimum

### Required Tools

The following tools must be available (installed automatically if missing):

- `curl`
- `git`
- `docker` (optional, for containerized resources)
- `systemd`

## Target Access

The cloud never holds a private key and the manifest never names one: `target.vps` carries the locator only (`host`, `port`, `user`, `workdir`). Access is bound one of two ways.

### 1. Enroll the host with Vrooli Bridge (recommended)

```bash
# on the VPS, after installing vrooli
vrooli-bridge onboard
```

Onboarding pairs the machine with your Bridge, records the enrollment (machine id, node id, generation) on the deployment's target binding and lets every cloud operation reach the host through the Bridge relay with scoped grants. A revoked enrollment is a typed refusal (`enrollment_revoked`); the cloud never falls back to SSH.

### 2. SSH with a credential binding

When the deployment's transport is `ssh`, the cloud authenticates with the key its credential binding names: `vrooli/scenario-to-cloud:ssh-key`, a binding of class `machine_enrollment_credential` whose file target is the operator-held key path. The binding never carries key bytes. A deployment converted from an older manifest that carried `target.vps.key_path` receives this binding automatically at API start. A target with no binding (a host you are preflighting before the deployment exists) is reached with your ambient SSH identity: the agent (`SSH_AUTH_SOCK`) or your default identity files.

Create the key and authorise it on the host with the standard tools:

```bash
ssh-keygen -t ed25519 -C "your-email@example.com"
ssh-copy-id -i ~/.ssh/id_ed25519.pub root@your-vps.com
ssh root@your-vps.com "echo 'SSH working!'"
```

Host keys are trusted on first use into the scenario's own `known_hosts` store (`$VROOLI_STATE_DIR/known_hosts`); a changed host key is refused as `target_offline` with the ssh diagnostic in the detail.

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

1. Create a Droplet with Ubuntu 24.04
2. Add your SSH key during creation
3. Note the assigned IP address
4. Configure DNS in your domain registrar

### Hetzner

1. Create a Cloud server with Ubuntu 24.04
2. Add your SSH key in the setup
3. Configure firewall in Cloud Console
4. Note the assigned IP address

### Linode

1. Create a Linode with Ubuntu 24.04
2. Add SSH key via Cloud Manager
3. Configure firewall rules
4. Note the assigned IP address
