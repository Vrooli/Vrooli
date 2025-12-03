---
title: "Deployment Guide"
description: "How to deploy your landing page to production"
category: "getting-started"
order: 4
audience: ["users", "developers"]
---

# Deployment Guide

This guide covers how to deploy your landing page from development to production.

## Table of Contents

1. [Deployment Overview](#deployment-overview)
2. [Local Development](#local-development)
3. [Cloud Deployment](#cloud-deployment)
4. [Domain Configuration](#domain-configuration)
5. [SSL/HTTPS Setup](#sslhttps-setup)
6. [Environment Configuration](#environment-configuration)
7. [Health Checks & Monitoring](#health-checks--monitoring)
8. [Rollback Procedures](#rollback-procedures)

---

## Deployment Overview

Landing pages can be deployed through multiple methods:

| Method | Best For | Complexity |
|--------|----------|------------|
| **Vrooli Managed** | Quick deployment, tunneled access | Low |
| **Docker** | Containerized deployments | Medium |
| **Traditional VPS** | Full control | Medium |
| **PaaS (Railway, Render)** | Managed infrastructure | Low |

---

## Local Development

### Starting Your Landing Page

Start the landing page for local testing:

```bash
# Using Makefile (recommended)
cd scenarios/<your-slug>
make start

# Or using Vrooli CLI
vrooli scenario start <your-slug>
```

### Accessing Local Services

Once started, your landing page is available at:

| Service | URL | Purpose |
|---------|-----|---------|
| Public Landing | `http://localhost:<UI_PORT>/` | What visitors see |
| Admin Portal | `http://localhost:<UI_PORT>/admin` | Manage content |
| API Health | `http://localhost:<API_PORT>/health` | Service status |

Get the actual ports:
```bash
vrooli scenario port <your-slug> UI_PORT
vrooli scenario port <your-slug> API_PORT
```

### Development Workflow

```
+-------------------------------------------------------------+
|                    LOCAL DEVELOPMENT                         |
+-------------------------------------------------------------+
|                                                             |
|  1. Edit content via Admin Portal                           |
|     --> Changes save to database immediately                |
|                                                             |
|  2. Edit code (if needed)                                   |
|     --> Vite hot-reloads UI changes                         |
|     --> Go API requires: make stop && make start            |
|                                                             |
|  3. Test your changes                                       |
|     --> make test                                           |
|                                                             |
|  4. Check performance                                       |
|     --> npx lighthouse http://localhost:<port> --view       |
|                                                             |
+-------------------------------------------------------------+
```

---

## Cloud Deployment

### Option 1: Vrooli Managed (Recommended)

Vrooli can deploy your landing page via Cloudflare Tunnel:

```bash
# Enable cloud deployment
vrooli deploy <slug> --tunnel

# This creates a public URL like:
# https://<slug>.vrooli.app
```

### Option 2: Docker Deployment

Export your landing page as a Docker container:

```bash
# Build Docker image
cd scenarios/<slug>
docker build -t my-landing:latest .

# Run container
docker run -d \
  -p 3000:3000 \
  -e DATABASE_URL="postgres://..." \
  -e STRIPE_SECRET_KEY="sk_live_..." \
  my-landing:latest
```

### Option 3: Traditional VPS

Deploy to any Linux server:

```bash
# 1. Copy files to server
rsync -avz scenarios/<slug>/ user@server:/opt/landing/

# 2. SSH to server
ssh user@server

# 3. Install dependencies
cd /opt/landing
pnpm install --production
cd api && go build -o landing-api .

# 4. Set up systemd service (see below)
sudo cp landing.service /etc/systemd/system/
sudo systemctl enable landing
sudo systemctl start landing
```

**Systemd Service Template** (`landing.service`):

```ini
[Unit]
Description=Landing Page Service
After=network.target postgresql.service

[Service]
Type=simple
User=www-data
WorkingDirectory=/opt/landing
Environment=API_PORT=3000
Environment=DATABASE_URL=postgres://...
ExecStart=/opt/landing/api/landing-api
Restart=always
RestartSec=5

[Install]
WantedBy=multi-user.target
```

### Option 4: Platform as a Service

Deploy to platforms like Railway, Render, or Fly.io:

**Railway Example:**
```bash
# Install Railway CLI
npm install -g @railway/cli

# Deploy
cd scenarios/<slug>
railway login
railway init
railway up
```

---

## Domain Configuration

### Setting Up a Custom Domain

1. **Register/Configure DNS**:

   Add an A record pointing to your server:
   ```
   Type: A
   Name: landing (or @ for root)
   Value: <your-server-ip>
   TTL: 300
   ```

   Or for Cloudflare proxied:
   ```
   Type: CNAME
   Name: landing
   Value: <your-tunnel-id>.cfargotunnel.com
   Proxied: Yes
   ```

2. **Update Landing Page Configuration**:

   Edit `.vrooli/branding` or via Admin Portal:
   ```json
   {
     "canonical_base_url": "https://landing.yourdomain.com"
   }
   ```

3. **Configure Reverse Proxy** (if applicable):

   Nginx example:
   ```nginx
   server {
       listen 80;
       server_name landing.yourdomain.com;

       location / {
           proxy_pass http://localhost:3000;
           proxy_http_version 1.1;
           proxy_set_header Upgrade $http_upgrade;
           proxy_set_header Connection 'upgrade';
           proxy_set_header Host $host;
           proxy_cache_bypass $http_upgrade;
       }
   }
   ```

---

## SSL/HTTPS Setup

### Option 1: Let's Encrypt with Certbot

```bash
# Install certbot
sudo apt install certbot python3-certbot-nginx

# Get certificate
sudo certbot --nginx -d landing.yourdomain.com

# Auto-renewal is configured automatically
```

### Option 2: Cloudflare (Recommended)

If using Cloudflare:
1. Enable "Full (strict)" SSL mode in Cloudflare dashboard
2. Cloudflare handles SSL automatically
3. No server-side certificate needed

### Option 3: Custom Certificate

```nginx
server {
    listen 443 ssl http2;
    server_name landing.yourdomain.com;

    ssl_certificate /etc/ssl/certs/landing.crt;
    ssl_certificate_key /etc/ssl/private/landing.key;

    # ... rest of config
}
```

---

## Environment Configuration

### Production Environment Variables

Create `initialization/configuration/<slug>.env` with production values:

```bash
# Database
DATABASE_URL=postgres://prod_user:secure_password@db.example.com:5432/landing_prod

# Stripe (LIVE keys)
STRIPE_PUBLISHABLE_KEY=pk_live_...
STRIPE_SECRET_KEY=sk_live_...
STRIPE_WEBHOOK_SECRET=whsec_...

# Application
NODE_ENV=production
API_PORT=3000
UI_PORT=3001

# Security
ADMIN_EMAIL=admin@yourdomain.com
SESSION_SECRET=<random-64-char-string>

# Optional
ANALYTICS_ENABLED=true
SENTRY_DSN=https://...@sentry.io/...
```

### Secrets Management

**Never commit secrets to git!**

Options:
1. **Environment files**: Use `.env` files (gitignored)
2. **Vrooli Secrets Manager**: `vrooli secret set <slug> STRIPE_SECRET_KEY`
3. **Cloud provider**: Use Railway/Render/Fly secrets
4. **Vault**: For enterprise deployments

---

## Health Checks & Monitoring

### Built-in Health Endpoint

```bash
curl http://localhost:3000/health

# Response:
{
  "status": "healthy",
  "service": "My Landing API",
  "version": "1.0.0",
  "readiness": true,
  "timestamp": "2024-01-15T10:30:00Z",
  "dependencies": {
    "database": "connected"
  }
}
```

### Setting Up Monitoring

**Uptime Monitoring** (recommended services):
- UptimeRobot (free tier available)
- Pingdom
- Better Uptime

Configure to check:
- `https://landing.yourdomain.com/health` every 5 minutes
- Alert on 5xx errors or timeout

**Error Tracking**:

Add Sentry for error tracking:
```bash
# Add to environment
SENTRY_DSN=https://...@sentry.io/...
```

### Lighthouse CI

Set up automated performance checks:

```yaml
# .github/workflows/lighthouse.yml
name: Lighthouse CI
on: [push]
jobs:
  lighthouse:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      - name: Run Lighthouse
        uses: treosh/lighthouse-ci-action@v9
        with:
          urls: |
            https://landing.yourdomain.com
          budgetPath: ./.vrooli/lighthouse.json
```

---

## Rollback Procedures

### Quick Rollback

If something goes wrong:

```bash
# 1. Stop the broken deployment
vrooli scenario stop <slug>

# 2. If you have a backup:
cp -r backups/<slug>-<date> scenarios/<slug>

# 3. Restart
vrooli scenario start <slug>
```

### Database Rollback

```bash
# Restore from backup
pg_restore -d landing_prod backups/landing-2024-01-15.dump

# Or restore specific tables
pg_restore -d landing_prod -t sections backups/landing-2024-01-15.dump
```

### Blue-Green Deployment

For zero-downtime deployments:

```
+--------------+     +--------------+
|   BLUE       |     |   GREEN      |
| (Current)    |     |   (New)      |
| Port 3000    |     | Port 3001    |
+------+-------+     +------+-------+
       |                   |
       +--------+----------+
                |
       +--------v---------+
       |  Load Balancer   |
       |  (nginx/HAProxy) |
       +------------------+
```

1. Deploy new version to GREEN
2. Test GREEN thoroughly
3. Switch load balancer to GREEN
4. Keep BLUE running for quick rollback
5. After verification, retire BLUE

---

## Pre-Deployment Checklist

Before going live:

### Content & Branding
- [ ] All placeholder text replaced
- [ ] Logo and favicon uploaded
- [ ] Meta titles and descriptions set
- [ ] OG images configured for social sharing
- [ ] Footer links working
- [ ] Privacy policy and terms linked

### Technical
- [ ] Stripe live keys configured
- [ ] Webhook endpoint accessible
- [ ] Database backed up
- [ ] SSL certificate valid
- [ ] Health endpoint responding
- [ ] Error tracking configured

### Testing
- [ ] All sections render correctly
- [ ] A/B variants working
- [ ] Checkout flow complete
- [ ] Admin portal accessible
- [ ] Mobile responsive
- [ ] Lighthouse score >= 80

### Security
- [ ] Admin password changed from default
- [ ] Sensitive files not exposed
- [ ] CORS configured correctly
- [ ] Rate limiting enabled
- [ ] Secrets not in git

---

## Troubleshooting Deployment

### "502 Bad Gateway"

The API isn't responding:
```bash
# Check if API is running
ps aux | grep landing-api

# Check logs
journalctl -u landing -n 100
```

### "Database connection refused"

```bash
# Verify DATABASE_URL
echo $DATABASE_URL

# Test connection
psql $DATABASE_URL -c "SELECT 1"

# Check PostgreSQL is running
systemctl status postgresql
```

### "SSL certificate errors"

```bash
# Check certificate validity
openssl s_client -connect landing.yourdomain.com:443 -servername landing.yourdomain.com

# Renew Let's Encrypt
sudo certbot renew --dry-run
```

### Webhook failures

```bash
# Check webhook endpoint is accessible
curl -X POST https://landing.yourdomain.com/api/v1/webhooks/stripe

# Should return 400 (no signature), not 404 or 5xx
```

---

## See Also

- [Quick Start](../TEMPLATE.md) - Initial setup
- [Configuration Guide](CONFIGURATION_GUIDE.md) - All config options
- [Troubleshooting](TROUBLESHOOTING.md) - Common issues
- [Admin Guide](ADMIN_GUIDE.md) - Managing your landing page
