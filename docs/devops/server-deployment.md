# Server Deployment Guide

This comprehensive guide covers setting up staging and production servers for Vrooli, including VPS configuration, security hardening, performance optimization, and deployment using both Docker and Kubernetes methods.

> **Prerequisites**: See [Prerequisites Guide](./getting-started/prerequisites.md) for required tools installation.
> **SSH Setup**: For comprehensive SSH key configuration, see [SSH Setup Guide](./getting-started/ssh-setup.md).
> **Environment Variables**: For complete variable reference, see [Environment Variables Guide](./getting-started/environment-variables.md).
> **CI/CD Integration**: For automated deployment setup, see [CI/CD Pipeline Guide](./ci-cd.md).
> **Troubleshooting**: For deployment issues, see [Troubleshooting Guide](./troubleshooting.md#server-deployment-issues).

## Overview

Server deployment for Vrooli supports multiple deployment strategies:

- 🖥️ **Docker Deployment**: Container-based deployment using Docker Compose
- ☸️ **Kubernetes Deployment**: Container orchestration with Helm charts
- 🔐 **Advanced Security**: Server hardening and application security
- 📊 **Production Monitoring**: Comprehensive monitoring and logging
- 🚀 **CI/CD Integration**: Automated deployment from GitHub Actions

## Server Requirements

### Minimum Requirements (Staging/Small Production)

- **CPU**: 4 cores
- **RAM**: 8GB
- **Storage**: 50GB SSD
- **OS**: Ubuntu 22.04 LTS
- **Network**: 1Gbps connection

### Recommended Requirements (Production)

- **CPU**: 8+ cores  
- **RAM**: 16GB+
- **Storage**: 100GB+ SSD
- **OS**: Ubuntu 22.04 LTS
- **Network**: 1Gbps+ connection

### High-Traffic Production

- **CPU**: 16+ cores
- **RAM**: 32GB+
- **Storage**: 200GB+ NVMe SSD
- **Network**: 10Gbps connection
- **Load Balancer**: Multiple servers with load balancing

## Server Provider Selection

> **Note**: This section provides guidance on cloud providers. For detailed setup of each provider, refer to their official documentation.

### Recommended Providers

#### **DigitalOcean** (Recommended for simplicity)
- **Pros**: Simple UI, good documentation, predictable pricing
- **Cons**: Limited advanced services
- **Best for**: Small to medium deployments

#### **Linode (Akamai)**
- **Pros**: Excellent performance-to-cost ratio, reliable
- **Cons**: Smaller ecosystem than AWS/GCP
- **Best for**: Cost-effective production deployments

#### **AWS EC2**
- **Pros**: Extensive services, global reach, enterprise features
- **Cons**: Complex pricing, steeper learning curve
- **Best for**: Enterprise deployments, complex requirements

#### **Google Cloud Platform**
- **Pros**: Excellent Kubernetes integration, competitive pricing
- **Cons**: Smaller market share, fewer tutorials
- **Best for**: Kubernetes-heavy deployments

#### **Vultr**
- **Pros**: Competitive pricing, global availability, good performance
- **Cons**: Smaller support ecosystem
- **Best for**: Budget-conscious deployments

### Provider-Specific Setup Examples

#### DigitalOcean Setup

1. **Create Account** and navigate to Droplets
2. **Choose Image**: Ubuntu 22.04 LTS
3. **Select Plan**: 
   - Basic plan for staging: 4GB RAM, 2 vCPUs, 80GB SSD
   - Premium plan for production: 8GB RAM, 4 vCPUs, 160GB SSD
4. **Choose Region**: Closest to your target audience
5. **Authentication**: Add SSH keys (see [SSH Setup Guide](./getting-started/ssh-setup.md))
6. **Additional Options**: Enable monitoring, backups (recommended for production)

#### AWS EC2 Setup

1. **Launch Instance** from EC2 Dashboard
2. **Choose AMI**: Ubuntu Server 22.04 LTS
3. **Instance Type**: 
   - t3.large (2 vCPUs, 8GB) for staging
   - t3.xlarge (4 vCPUs, 16GB) for production
4. **Configure Security Groups**: HTTP (80), HTTPS (443), SSH (22)
5. **Key Pair**: Create or select existing key pair
6. **Storage**: 50GB+ gp3 SSD

## SSH Key Setup

> **Complete SSH Guide**: For comprehensive SSH key setup, testing, and troubleshooting, see [SSH Setup Guide](./getting-started/ssh-setup.md).

### Quick SSH Setup

```bash
# Generate deployment keys (see SSH Setup Guide for details)
ssh-keygen -t ed25519 -f ~/.ssh/vrooli_staging_deploy -C "vrooli-staging-deploy"
ssh-keygen -t ed25519 -f ~/.ssh/vrooli_production_deploy -C "vrooli-production-deploy"

# Add to server (see SSH Setup Guide for multiple methods)
ssh-copy-id -i ~/.ssh/vrooli_staging_deploy.pub user@server-ip

# Test connection (see SSH Setup Guide for debugging)
ssh -i ~/.ssh/vrooli_staging_deploy user@server-ip "echo 'SSH connection successful'"
```

## Domain Name & DNS Setup

### Domain Registration

Recommended domain registrars:
- **Namecheap**: Good value, simple interface
- **Google Domains**: Integrated with Google services
- **Cloudflare Registrar**: Best for performance and security

### DNS Configuration

#### Cloudflare Worker Setup (Recommended)

For dynamic Open Graph metadata support using Cloudflare Workers, configure DNS as follows:

| Type | Name | Value | Proxy | Purpose |
|------|------|-------|-------|---------|
| A | @ | server_ip | **Proxied 🟠** | Main domain (Worker intercepts) |
| A | do-origin | server_ip | **DNS-only ⚪** | Origin server (Worker calls this) |
| A | api | server_ip | **Proxied 🟠** | API subdomain |
| CNAME | www | @ | **Proxied 🟠** | WWW redirect |

**Critical Setup Notes:**
- `@` (apex domain) **must be orange** so Cloudflare and the Worker sit in front of traffic
- `do-origin` **must be grey** to prevent recursive loops when the Worker calls the origin server
- The Worker serves dynamic OG metadata for crawlers, proxies regular traffic to `do-origin`

#### Advanced DNS with Cloudflare

1. **Transfer DNS to Cloudflare** (required for Worker functionality)
2. **Enable Proxy** for web traffic (orange cloud) on main domains
3. **Configure SSL/TLS**: Full (strict) mode
4. **Security Settings**: 
   - Bot Fight Mode: On
   - Security Level: Medium
   - Always Use HTTPS: On

```bash
# Complete DNS record configuration for Cloudflare
# Main production domain (Worker intercepts crawler traffic)
Type: A, Name: @, Content: production_ip, TTL: Auto, Proxy: On

# Origin server (Worker calls this for actual content)
Type: A, Name: do-origin, Content: production_ip, TTL: Auto, Proxy: Off

# API subdomain (proxied for security)
Type: A, Name: api, Content: production_ip, TTL: Auto, Proxy: On

# WWW redirect
Type: CNAME, Name: www, Content: @, TTL: Auto, Proxy: On
```

#### Worker Configuration Requirements

Your server deployment must accommodate the Cloudflare Worker setup:

```bash
# Environment variables for Worker integration
VIRTUAL_HOST=example.com,do-origin.example.com
LETSENCRYPT_HOST=example.com,do-origin.example.com
```

#### Alternative: Basic DNS Setup (Without Workers)

If not using Cloudflare Workers for OG metadata, use this simpler configuration:

| Type | Name | Value | TTL | Purpose |
|------|------|-------|-----|---------|
| A | @ | server_ip | 300 | Main domain |
| A | api | server_ip | 300 | API subdomain |
| CNAME | www | @ | 300 | WWW redirect |

## Server Security Hardening

> **Note**: For security troubleshooting, see [Troubleshooting Guide](./troubleshooting.md).

### Initial Security Setup

```bash
# Connect to server
ssh root@server-ip

# Update system
apt update && apt upgrade -y

# Create non-root user
adduser deploy
usermod -aG sudo deploy

# Copy SSH keys to new user
mkdir -p /home/deploy/.ssh
cp ~/.ssh/authorized_keys /home/deploy/.ssh/
chown -R deploy:deploy /home/deploy/.ssh
chmod 700 /home/deploy/.ssh
chmod 600 /home/deploy/.ssh/authorized_keys
```

### Firewall Configuration

```bash
# Configure UFW firewall
ufw default deny incoming
ufw default allow outgoing
ufw allow ssh
ufw allow http
ufw allow https

# For Kubernetes (if using)
ufw allow 6443    # Kubernetes API server
ufw allow 30000:32767/tcp  # NodePort range

# Enable firewall
ufw --force enable

# Check status
ufw status verbose
```

### SSH Hardening

```bash
# Edit SSH configuration
sudo nano /etc/ssh/sshd_config
```

Apply these security settings:

```bash
# Disable root login
PermitRootLogin no

# Disable password authentication
PasswordAuthentication no
PubkeyAuthentication yes

# Change default port (optional but recommended)
Port 2222

# Limit login attempts
MaxAuthTries 3
MaxStartups 10:30:60

# Disable unused authentication methods
ChallengeResponseAuthentication no
UsePAM no

# Set login timeout
LoginGraceTime 30

# Allow specific users only
AllowUsers deploy
```

Restart SSH service:

```bash
sudo systemctl restart sshd

# Test new connection on new port (if changed)
ssh -p 2222 deploy@server-ip
```

### Fail2Ban Installation

```bash
# Install Fail2Ban
sudo apt install fail2ban

# Create local configuration
sudo cp /etc/fail2ban/jail.conf /etc/fail2ban/jail.local

# Edit configuration
sudo nano /etc/fail2ban/jail.local
```

Configure Fail2Ban:

```ini
[DEFAULT]
bantime = 3600        # 1 hour ban
findtime = 600        # 10 minute window
maxretry = 3          # 3 attempts

[sshd]
enabled = true
port = ssh,2222       # Include custom SSH port
logpath = /var/log/auth.log
```

Start and enable Fail2Ban:

```bash
sudo systemctl enable fail2ban
sudo systemctl start fail2ban

# Check status
sudo fail2ban-client status
sudo fail2ban-client status sshd
```

### Automatic Security Updates

```bash
# Install unattended upgrades
sudo apt install unattended-upgrades

# Configure automatic updates
sudo dpkg-reconfigure unattended-upgrades

# Edit configuration
sudo nano /etc/apt/apt.conf.d/50unattended-upgrades
```

Enable security updates:

```bash
Unattended-Upgrade::Allowed-Origins {
    "${distro_id}:${distro_codename}-security";
    "${distro_id}ESMApps:${distro_codename}-apps-security";
    "${distro_id}ESM:${distro_codename}-infra-security";
};

Unattended-Upgrade::AutoFixInterruptedDpkg "true";
Unattended-Upgrade::MinimalSteps "true";
Unattended-Upgrade::Remove-Unused-Dependencies "true";
Unattended-Upgrade::Automatic-Reboot "false";
```

## Server Preparation

> **Prerequisites**: Ensure you have completed the [Prerequisites](./getting-started/prerequisites.md) installation.

### Install Required Dependencies

```bash
# Connect as deploy user
ssh deploy@server-ip

# Clone repository
cd ~
git clone https://github.com/Vrooli/Vrooli.git
cd Vrooli

# Make scripts executable
chmod +x scripts/main/*.sh
chmod +x scripts/helpers/**/*.sh

# Run server setup
bash scripts/main/setup.sh --location remote --environment production --target docker-daemon
```

The setup script will automatically install:
- Docker and Docker Compose
- Node.js and PNPM
- PostgreSQL client tools
- Redis client tools
- SSL certificates (via Let's Encrypt)
- Monitoring tools

### Environment Configuration

> **Complete Environment Guide**: For comprehensive environment variables documentation, see [Environment Variables Guide](./getting-started/environment-variables.md).

```bash
# Create production environment file
cp .env-example .env-prod

# Edit production configuration
nano .env-prod
```

Essential production environment variables:

```bash
# Environment
ENVIRONMENT=production
NODE_ENV=production
LOCATION=remote

# Security
SECRETS_SOURCE=vault  # or 'file' for simple setup
VIRTUAL_HOST=example.com
LETSENCRYPT_HOST=example.com
LETSENCRYPT_EMAIL=admin@example.com

# Database (auto-generated secure values)
DB_HOST=postgres
DB_NAME=postgres
DB_USER=vrooli_prod
DB_PASSWORD=<secure-random-password>

# Redis (auto-generated secure values)
REDIS_HOST=redis
REDIS_PASSWORD=<secure-random-password>

# Application URLs
API_URL=https://example.com
UI_URL=https://example.com
SITE_EMAIL_FROM=noreply@example.com

# External Services (add your keys)
OPENAI_API_KEY=your-openai-key
STRIPE_PUBLIC_KEY=your-stripe-public-key
STRIPE_SECRET_KEY=your-stripe-secret-key

# Email Configuration (if using)
SITE_EMAIL_USERNAME=your-email@domain.com
SITE_EMAIL_PASSWORD=your-email-password
SITE_EMAIL_HOST=smtp.domain.com
SITE_EMAIL_PORT=587
```

## Deployment Methods

### Docker Deployment

#### Manual Deployment

```bash
# Connect to server
ssh deploy@server-ip
cd ~/Vrooli

# Build and deploy
bash scripts/main/build.sh --environment production --artifacts docker --bundles zip
bash scripts/main/deploy.sh --source docker --environment production
```

#### CI/CD Deployment

> **Complete CI/CD Guide**: For automated deployment setup, see [CI/CD Pipeline Guide](./ci-cd.md).

The GitHub Actions workflow automatically deploys when you push to the appropriate branch:

```bash
# For staging (dev branch)
git push origin dev

# For production (master branch) 
git push origin master
```

#### Docker Compose Configuration

The deployment creates a `docker-compose.prod.yml` with:

```yaml
version: '3.8'
services:
  nginx-proxy:
    image: nginxproxy/nginx-proxy
    ports:
      - "80:80"
      - "443:443"
    volumes:
      - /var/run/docker.sock:/tmp/docker.sock:ro
      - ./nginx/certs:/etc/nginx/certs:ro
      - ./nginx/vhost.d:/etc/nginx/vhost.d
      - ./nginx/html:/usr/share/nginx/html
    restart: unless-stopped

  letsencrypt:
    image: nginxproxy/acme-companion
    volumes:
      - /var/run/docker.sock:/var/run/docker.sock:ro
      - ./nginx/certs:/etc/nginx/certs
      - ./nginx/vhost.d:/etc/nginx/vhost.d
      - ./nginx/html:/usr/share/nginx/html
    environment:
      - DEFAULT_EMAIL=admin@example.com
    restart: unless-stopped

  postgres:
    image: postgres:15
    environment:
      POSTGRES_DB: postgres
      POSTGRES_USER: vrooli_prod
      POSTGRES_PASSWORD: ${DB_PASSWORD}
    volumes:
      - postgres_data:/var/lib/postgresql/data
    restart: unless-stopped

  redis:
    image: redis:7
    command: redis-server --requirepass ${REDIS_PASSWORD}
    volumes:
      - redis_data:/data
    restart: unless-stopped

  server:
    image: vrooli/server:prod
    environment:
      - VIRTUAL_HOST=example.com
      - LETSENCRYPT_HOST=example.com
      - LETSENCRYPT_EMAIL=admin@example.com
    env_file:
      - .env-prod
    depends_on:
      - postgres
      - redis
    restart: unless-stopped

volumes:
  postgres_data:
  redis_data:
```

### Kubernetes Deployment

> **Complete Kubernetes Guide**: For detailed Kubernetes setup, see [Kubernetes Deployment](./kubernetes.md).

#### Prerequisites

```bash
# Install Kubernetes tools
curl -LO "https://dl.k8s.io/release/$(curl -L -s https://dl.k8s.io/release/stable.txt)/bin/linux/amd64/kubectl"
sudo install -o root -g root -m 0755 kubectl /usr/local/bin/kubectl

# Install Helm
curl https://get.helm.sh/helm-v3.12.0-linux-amd64.tar.gz | tar xz
sudo mv linux-amd64/helm /usr/local/bin/

# Install k3s (lightweight Kubernetes)
curl -sfL https://get.k3s.io | sh -s - --write-kubeconfig-mode 644

# Configure kubectl
mkdir -p ~/.kube
sudo cp /etc/rancher/k3s/k3s.yaml ~/.kube/config
sudo chown deploy:deploy ~/.kube/config
```

#### Deploy with Kubernetes

```bash
# Build and deploy with Kubernetes
bash scripts/main/build.sh --environment production --artifacts k8s --version 1.0.0
bash scripts/main/deploy.sh --source k8s --environment production --version 1.0.0
```
#### Kubernetes Resources

The deployment creates:

- **Deployments**: Server, UI, Jobs
- **StatefulSets**: PostgreSQL, Redis
- **Services**: Load balancing and service discovery
- **Ingress**: SSL termination and routing
- **ConfigMaps**: Configuration management
- **Secrets**: Sensitive data management

## Performance Optimization

### NGINX Configuration

Optimize reverse proxy for production:

```bash
# Edit NGINX configuration
sudo nano /etc/nginx/conf.d/performance.conf
```

Add performance settings:

```nginx
# Worker processes
worker_processes auto;
worker_rlimit_nofile 65535;

# Connection settings
events {
    worker_connections 1024;
    multi_accept on;
    use epoll;
}

http {
    # Basic settings
    sendfile on;
    tcp_nopush on;
    tcp_nodelay on;
    keepalive_timeout 65;
    types_hash_max_size 2048;
    server_tokens off;

    # Buffer settings
    client_body_buffer_size 16K;
    client_header_buffer_size 1k;
    client_max_body_size 8m;
    large_client_header_buffers 2 1k;

    # Timeout settings
    client_body_timeout 12;
    client_header_timeout 12;
    send_timeout 10;

    # Gzip compression
    gzip on;
    gzip_vary on;
    gzip_min_length 1024;
    gzip_comp_level 6;
    gzip_types
        text/plain
        text/css
        text/xml
        text/javascript
        application/javascript
        application/xml+rss
        application/json;

    # File caching
    open_file_cache max=1000 inactive=20s;
    open_file_cache_valid 30s;
    open_file_cache_min_uses 2;
    open_file_cache_errors on;

    # Rate limiting
    limit_req_zone $binary_remote_addr zone=api:10m rate=10r/s;
    limit_req_zone $binary_remote_addr zone=login:10m rate=1r/s;
}
```

### Database Optimization

Optimize PostgreSQL for production:

```bash
# Edit PostgreSQL configuration
sudo nano /etc/postgresql/15/main/postgresql.conf
```

Production settings:

```ini
# Memory settings (adjust based on available RAM)
shared_buffers = 4GB                    # 25% of RAM
effective_cache_size = 12GB             # 75% of RAM
work_mem = 64MB                         # Per connection
maintenance_work_mem = 1GB              # For maintenance

# Checkpoint settings
checkpoint_timeout = 15min
checkpoint_completion_target = 0.9
wal_buffers = 16MB

# Connection settings
max_connections = 200
shared_preload_libraries = 'pg_stat_statements'

# Logging
log_statement = 'ddl'
log_checkpoints = on
log_connections = on
log_disconnections = on
log_lock_waits = on
log_temp_files = 0

# Performance
random_page_cost = 1.1                  # For SSD storage
effective_io_concurrency = 200          # For SSD storage
```

Apply configuration:

```bash
sudo systemctl restart postgresql
```

### Redis Optimization

Configure Redis for production:

```bash
# Edit Redis configuration
sudo nano /etc/redis/redis.conf
```

Production settings:

```ini
# Memory management
maxmemory 2gb
maxmemory-policy allkeys-lru

# Persistence
save 900 1
save 300 10
save 60 10000

# Security
requirepass your-secure-redis-password
rename-command FLUSHDB ""
rename-command FLUSHALL ""
rename-command DEBUG ""

# Network
bind 127.0.0.1
port 6379
timeout 300
keepalive 60
```

## Monitoring & Observability

### System Monitoring with Prometheus

```bash
# Install Prometheus
sudo apt update
sudo apt install prometheus

# Configure Prometheus
sudo nano /etc/prometheus/prometheus.yml
```

Basic Prometheus configuration:

```yaml
global:
  scrape_interval: 15s
  evaluation_interval: 15s

rule_files:
  - "first_rules.yml"

scrape_configs:
  - job_name: 'prometheus'
    static_configs:
      - targets: ['localhost:9090']

  - job_name: 'node'
    static_configs:
      - targets: ['localhost:9100']

  - job_name: 'vrooli-server'
    static_configs:
      - targets: ['localhost:5329']
    metrics_path: '/metrics'
```

### Grafana Dashboard

```bash
# Install Grafana
sudo apt-get install -y software-properties-common
sudo add-apt-repository "deb https://packages.grafana.com/oss/deb stable main"
wget -q -O - https://packages.grafana.com/gpg.key | sudo apt-key add -
sudo apt-get update
sudo apt-get install grafana

# Enable and start Grafana
sudo systemctl enable grafana-server
sudo systemctl start grafana-server
```

Access Grafana at `http://server-ip:3000` (admin/admin)

### Application Monitoring

```bash
# Install Node Exporter for system metrics
wget https://github.com/prometheus/node_exporter/releases/download/v1.6.0/node_exporter-1.6.0.linux-amd64.tar.gz
tar xvfz node_exporter-1.6.0.linux-amd64.tar.gz
sudo mv node_exporter-1.6.0.linux-amd64/node_exporter /usr/local/bin/
sudo useradd -rs /bin/false node_exporter

# Create systemd service
sudo tee /etc/systemd/system/node_exporter.service > /dev/null <<EOF
[Unit]
Description=Node Exporter
After=network.target

[Service]
User=node_exporter
Group=node_exporter
Type=simple
ExecStart=/usr/local/bin/node_exporter

[Install]
WantedBy=multi-user.target
EOF

# Enable and start
sudo systemctl daemon-reload
sudo systemctl enable node_exporter
sudo systemctl start node_exporter
```

### Log Management

For detailed information on application logging types, access methods, and best practices, refer to the [Logging Guide](./logging.md). The following details specifically cover Docker daemon log configuration for server environments.

Set up centralized logging:

```bash
# Configure Docker logging
sudo nano /etc/docker/daemon.json
```

```json
{
  "log-driver": "json-file",
  "log-opts": {
    "max-size": "10m",
    "max-file": "3"
  }
}
```

```bash
sudo systemctl restart docker
```

## Backup Strategy

### Database Backups

Create automated backup script:

```bash
# Create backup script
sudo nano /usr/local/bin/backup_vrooli.sh
```

```bash
#!/bin/bash
set -e

# Configuration
BACKUP_DIR="/var/backups/vrooli"
DATE=$(date +%Y%m%d_%H%M%S)
KEEP_DAYS=7

# Create backup directory
mkdir -p $BACKUP_DIR

# Database backup
docker exec postgres pg_dump -U vrooli_prod postgres | gzip > "$BACKUP_DIR/db_backup_$DATE.sql.gz"

# Environment files backup
tar -czf "$BACKUP_DIR/config_backup_$DATE.tar.gz" .env-prod jwt_*.pem

# Optional: Upload to cloud storage
# aws s3 cp "$BACKUP_DIR/db_backup_$DATE.sql.gz" s3://your-backup-bucket/
# aws s3 cp "$BACKUP_DIR/config_backup_$DATE.tar.gz" s3://your-backup-bucket/

# Cleanup old backups
find $BACKUP_DIR -name "*.gz" -type f -mtime +$KEEP_DAYS -delete

echo "Backup completed: $DATE"
```

Make executable and schedule:

```bash
sudo chmod +x /usr/local/bin/backup_vrooli.sh

# Add to crontab
sudo crontab -e
# Add: 0 2 * * * /usr/local/bin/backup_vrooli.sh > /var/log/backup.log 2>&1
```

### Disaster Recovery

Document recovery procedures:

```bash
# Create recovery script
sudo nano /usr/local/bin/restore_vrooli.sh
```

```bash
#!/bin/bash
set -e

BACKUP_FILE=$1
if [ -z "$BACKUP_FILE" ]; then
    echo "Usage: $0 <backup_file.sql.gz>"
    exit 1
fi

echo "Restoring from backup: $BACKUP_FILE"

# Stop application
docker-compose -f docker-compose.prod.yml down

# Restore database
gunzip -c "$BACKUP_FILE" | docker exec -i postgres psql -U vrooli_prod postgres

# Start application
docker-compose -f docker-compose.prod.yml up -d

echo "Restore completed"
```

## Health Monitoring

### Application Health Checks

Create health monitoring script:

```bash
# Create health check script
sudo nano /usr/local/bin/health_check.sh
```

```bash
#!/bin/bash

DOMAIN="example.com"
ENDPOINTS=(
    "https://$DOMAIN/api/healthcheck"
    "https://$DOMAIN/"
)

for endpoint in "${ENDPOINTS[@]}"; do
    status=$(curl -s -o /dev/null -w "%{http_code}" "$endpoint")
    if [ "$status" != "200" ]; then
        echo "$(date): Health check FAILED for $endpoint (HTTP $status)"
        # Send notification (webhook, email, etc.)
        curl -X POST -H "Content-Type: application/json" \
            -d "{\"text\":\"🚨 Health check failed: $endpoint returned HTTP $status\"}" \
            "$WEBHOOK_URL"
        exit 1
    fi
done

echo "$(date): All health checks PASSED"
```

Schedule health checks:

```bash
sudo chmod +x /usr/local/bin/health_check.sh

# Add to crontab for regular checks
sudo crontab -e
# Add: */5 * * * * /usr/local/bin/health_check.sh >> /var/log/health_check.log 2>&1
```

## SSL Certificate Management

### Let's Encrypt with Docker

SSL certificates are automatically managed by the `acme-companion` container in the Docker setup. For manual management:

```bash
# Install certbot
sudo apt install certbot python3-certbot-nginx

# Generate certificates
sudo certbot certonly --standalone -d example.com -d api.example.com

# Test renewal
sudo certbot renew --dry-run

# Automatic renewal is handled by the acme-companion container
```

## Maintenance Procedures

### Regular Maintenance Tasks

```bash
# Weekly maintenance script
sudo nano /usr/local/bin/weekly_maintenance.sh
```

```bash
#!/bin/bash

echo "Starting weekly maintenance: $(date)"

# System updates
apt update && apt upgrade -y

# Docker cleanup
docker system prune -f
docker volume prune -f

# Database maintenance
docker exec postgres psql -U vrooli_prod -c "VACUUM ANALYZE;"

# Log rotation
journalctl --vacuum-time=30d

# Check disk space
df -h
docker system df

echo "Weekly maintenance completed: $(date)"
```

Schedule maintenance:

```bash
sudo chmod +x /usr/local/bin/weekly_maintenance.sh

# Add to crontab
sudo crontab -e
# Add: 0 3 * * 0 /usr/local/bin/weekly_maintenance.sh >> /var/log/maintenance.log 2>&1
```

## Scaling Strategies

### Horizontal Scaling (Multiple Servers)

For high-traffic deployments:

1. **Load Balancer Setup**:
   - Use cloud provider load balancer (AWS ALB, GCP Load Balancer)
   - Or self-hosted HAProxy/NGINX load balancer

2. **Database Scaling**:
   - Read replicas for PostgreSQL
   - Redis clustering for session storage

3. **Application Scaling**:
   - Multiple server instances
   - Container orchestration with Kubernetes

### Vertical Scaling

Monitor resource usage and scale up when needed:

```bash
# Monitor resource usage
htop
docker stats
kubectl top nodes  # For Kubernetes
```

## Production Checklist

### Pre-Deployment Checklist

- [ ] Server meets minimum requirements
- [ ] SSH keys configured and tested
- [ ] Domain name configured with proper DNS
- [ ] SSL certificates working
- [ ] Firewall configured and enabled
- [ ] Security hardening applied
- [ ] Monitoring systems installed
- [ ] Backup strategy implemented
- [ ] Health checks configured

### Post-Deployment Verification

- [ ] Application accessible via domain
- [ ] SSL certificate valid and auto-renewing
- [ ] Database connections working
- [ ] API endpoints responding correctly
- [ ] Monitoring systems collecting data
- [ ] Backup scripts running successfully
- [ ] Log aggregation working

### Security Audit

- [ ] SSH hardening applied
- [ ] Firewall rules verified
- [ ] SSL/TLS configuration secure
- [ ] Database access restricted
- [ ] Application secrets properly managed
- [ ] Regular security updates enabled

This comprehensive server deployment guide ensures your Vrooli installation is secure, performant, and production-ready for both Docker and Kubernetes deployment strategies.

## Related Documentation

- **Prerequisites**: [Prerequisites Guide](./getting-started/prerequisites.md)
- **SSH Setup**: [SSH Setup Guide](./getting-started/ssh-setup.md)
- **Environment Variables**: [Environment Variables Reference](./getting-started/environment-variables.md)
- **CI/CD Pipeline**: [CI/CD Pipeline Guide](./ci-cd.md)
- **Kubernetes Deployment**: [Kubernetes Guide](./kubernetes.md)
- **Build System**: [Build System Guide](./build-system.md)
- **Troubleshooting**: [Troubleshooting Guide](./troubleshooting.md)
