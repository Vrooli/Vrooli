# Production Environment Setup

This guide covers setting up a production environment for Vrooli, focusing on security, performance, and reliability.

## Overview

A production environment should prioritize:

1. **Security**: Protection against attacks and data breaches
2. **Stability**: Ensuring consistent uptime and performance
3. **Scalability**: Ability to handle increased load
4. **Monitoring**: Detecting and addressing issues before they impact users
5. **Backup & Recovery**: Protecting against data loss

## Architecture Recommendations

For production deployments, we recommend either:

1. **Single Server Deployment** - for smaller installations (see [Single Server Deployment](./single_server.md))
2. **Multiple Servers Deployment** - for higher traffic and better reliability (see [Multiple Servers Deployment](./multiple_servers.md))
3. **Kubernetes Deployment** - for enterprise-level installations (see [Kubernetes Setup](./kubernetes_testing.md))

## Server Requirements

### Minimum Requirements

For a basic production environment:

- **CPU**: 4 cores
- **RAM**: 8GB
- **Storage**: 50GB SSD
- **Network**: 1Gbps connection

### Recommended Requirements

For optimal performance with moderate traffic:

- **CPU**: 8+ cores
- **RAM**: 16GB+
- **Storage**: 100GB+ SSD
- **Network**: 1Gbps+ connection

## Security Considerations

### Server Hardening

Apply these security measures to all production servers:

1. **Firewall Configuration**:
   ```bash
   # Allow only necessary ports
   sudo ufw default deny incoming
   sudo ufw default allow outgoing
   sudo ufw allow ssh
   sudo ufw allow http
   sudo ufw allow https
   sudo ufw enable
   ```

2. **SSH Hardening**:
   ```bash
   # Edit SSH config
   sudo nano /etc/ssh/sshd_config
   ```
   
   Set these values:
   ```
   PermitRootLogin no
   PasswordAuthentication no
   PubkeyAuthentication yes
   ```
   
   Restart SSH:
   ```bash
   sudo systemctl restart sshd
   ```

3. **Automatic Updates**:
   ```bash
   sudo apt install unattended-upgrades
   sudo dpkg-reconfigure unattended-upgrades
   ```

4. **Fail2Ban Installation**:
   ```bash
   sudo apt install fail2ban
   sudo cp /etc/fail2ban/jail.conf /etc/fail2ban/jail.local
   sudo nano /etc/fail2ban/jail.local
   ```
   
   Configure and start:
   ```bash
   sudo systemctl enable fail2ban
   sudo systemctl start fail2ban
   ```

### Application Security

1. **HTTPS Configuration**:
   
   Always use HTTPS in production. Set up with Let's Encrypt:
   ```bash
   sudo apt install certbot python3-certbot-nginx
   sudo certbot --nginx -d yourdomain.com -d www.yourdomain.com
   ```

2. **Secrets Management**:
   
   Use HashiCorp Vault for managing secrets. See [Secrets Management](./secrets_management.md).

3. **User Authentication Security**:
   
   - Use strong password policies
   - Implement rate limiting for login attempts
   - Consider adding two-factor authentication

4. **Container Security**:
   
   - Use non-root users in containers
   - Keep container images updated
   - Scan containers for vulnerabilities:
     ```bash
     # Install trivy
     sudo apt install trivy
     
     # Scan an image
     trivy image vrooli-server:latest
     ```

## Performance Optimization

### NGINX Configuration

Optimize NGINX for production:

```bash
sudo nano /etc/nginx/nginx.conf
```

Add performance settings:

```nginx
worker_processes auto;
worker_rlimit_nofile 65535;

events {
    worker_connections 1024;
    multi_accept on;
}

http {
    sendfile on;
    tcp_nopush on;
    tcp_nodelay on;
    keepalive_timeout 65;
    types_hash_max_size 2048;
    server_tokens off;
    
    # Cache settings
    open_file_cache max=1000 inactive=20s;
    open_file_cache_valid 30s;
    open_file_cache_min_uses 2;
    open_file_cache_errors on;
    
    # Gzip compression
    gzip on;
    gzip_comp_level 5;
    gzip_min_length 256;
    gzip_proxied any;
    gzip_vary on;
    gzip_types
        application/javascript
        application/json
        application/x-javascript
        application/xml
        application/xml+rss
        image/svg+xml
        text/css
        text/javascript
        text/plain
        text/xml;
}
```

### Database Optimization

Tune PostgreSQL for production:

```bash
sudo nano /etc/postgresql/13/main/postgresql.conf
```

Adjust these settings based on server resources:

```
# Memory settings
shared_buffers = 2GB                  # 25% of RAM up to 8GB
work_mem = 20MB                       # Depends on query complexity
maintenance_work_mem = 256MB          # For maintenance operations
effective_cache_size = 6GB            # 75% of RAM

# Checkpoint settings
checkpoint_timeout = 15min
checkpoint_completion_target = 0.9

# Planner settings
random_page_cost = 1.1                # For SSD storage

# WAL settings
wal_buffers = 16MB
```

### Redis Configuration

Optimize Redis for production:

```bash
sudo nano /etc/redis/redis.conf
```

Adjust these settings:

```
maxmemory 2gb
maxmemory-policy allkeys-lru
```

## Monitoring and Logging

### System Monitoring

Set up Prometheus and Grafana for monitoring:

```bash
# Install Prometheus
sudo apt update
sudo apt install -y prometheus

# Install Grafana
sudo apt-get install -y software-properties-common
sudo add-apt-repository "deb https://packages.grafana.com/oss/deb stable main"
wget -q -O - https://packages.grafana.com/gpg.key | sudo apt-key add -
sudo apt-get update
sudo apt-get install -y grafana

# Enable and start services
sudo systemctl enable prometheus
sudo systemctl start prometheus
sudo systemctl enable grafana-server
sudo systemctl start grafana-server
```

Access Grafana at http://your-server-ip:3000 (default credentials: admin/admin)

### Log Management

Set up centralized logging with the ELK stack:

```bash
# Install Elasticsearch
sudo apt install -y elasticsearch

# Install Logstash
sudo apt install -y logstash

# Install Kibana
sudo apt install -y kibana

# Enable and start services
sudo systemctl enable elasticsearch
sudo systemctl start elasticsearch
sudo systemctl enable logstash
sudo systemctl start logstash
sudo systemctl enable kibana
sudo systemctl start kibana
```

Configure Docker to send logs to Logstash:

```bash
sudo nano /etc/docker/daemon.json
```

Add:

```json
{
  "log-driver": "syslog",
  "log-opts": {
    "syslog-address": "tcp://localhost:5000",
    "tag": "{{.Name}}/{{.ID}}"
  }
}
```

Restart Docker:

```bash
sudo systemctl restart docker
```

### Health Checks

Set up regular health checks:

```bash
# Create health check script
sudo nano /usr/local/bin/check_vrooli.sh
```

Add:

```bash
#!/bin/bash
ENDPOINT="https://yourdomain.com/api/healthcheck"
RESPONSE=$(curl -s -o /dev/null -w "%{http_code}" $ENDPOINT)

if [ $RESPONSE -ne 200 ]; then
    echo "Health check failed: HTTP $RESPONSE"
    # Send alert (e.g., via email or webhook)
    curl -X POST -H "Content-Type: application/json" \
      -d '{"text":"⚠️ Health check failed: HTTP '"$RESPONSE"'"}' \
      https://your-notification-webhook
    exit 1
fi

echo "Health check passed: HTTP $RESPONSE"
exit 0
```

Make executable and add to crontab:

```bash
sudo chmod +x /usr/local/bin/check_vrooli.sh
sudo crontab -e
```

Add:

```
*/5 * * * * /usr/local/bin/check_vrooli.sh > /var/log/health_check.log 2>&1
```

## Backup and Recovery

### Database Backups

Set up automated backups:

```bash
# Create backup script
sudo nano /usr/local/bin/backup_db.sh
```

Add:

```bash
#!/bin/bash
DATE=$(date +%Y%m%d_%H%M%S)
BACKUP_DIR="/var/backups/postgres"
FILENAME="vrooli_db_$DATE.sql.gz"
KEEP_DAYS=7

# Create backup directory if it doesn't exist
mkdir -p $BACKUP_DIR

# Create backup
sudo -u postgres pg_dump vrooli | gzip > "$BACKUP_DIR/$FILENAME"

# Upload to remote storage (optional)
# aws s3 cp "$BACKUP_DIR/$FILENAME" s3://your-bucket/backups/

# Remove old backups
find $BACKUP_DIR -name "vrooli_db_*.sql.gz" -type f -mtime +$KEEP_DAYS -delete
```

Make executable and add to crontab:

```bash
sudo chmod +x /usr/local/bin/backup_db.sh
sudo crontab -e
```

Add:

```
0 2 * * * /usr/local/bin/backup_db.sh > /var/log/db_backup.log 2>&1
```

### Disaster Recovery Plan

Create a disaster recovery plan document outlining:

1. **Recovery Point Objective (RPO)**: How much data loss is acceptable
2. **Recovery Time Objective (RTO)**: How quickly services must be restored
3. **Backup Verification**: How and when to verify backups
4. **Restoration Procedures**: Step-by-step instructions for restoring from backups
5. **Contact Information**: Who to contact in case of an emergency

Test your recovery plan regularly to ensure it works as expected.

## Deployment Process

Use a systematic approach to production deployments:

### Pre-Deployment Checklist

1. All tests pass in CI/CD pipeline
2. Database migrations are tested
3. Backup is taken before deployment
4. Deployment is scheduled and team is notified

### Deployment Steps

1. **Create a deployment plan**:
   - Document changes being deployed
   - List potential risks and mitigation strategies
   - Define rollback procedure

2. **Execute deployment**:
   ```bash
   # SSH into production server
   ssh user@production-server
   
   # Navigate to project directory
   cd /path/to/vrooli
   
   # Pull latest changes
   git pull origin main
   
   # Run deployment script
   ./scripts/deploy.sh
   ```

3. **Verify deployment**:
   - Check application health endpoint
   - Verify critical functionality works
   - Monitor error logs for new issues

4. **Rollback if necessary**:
   ```bash
   # Revert to previous version
   git checkout previous-tag
   
   # Run deployment script
   ./scripts/deploy.sh
   
   # Restore database if needed
   # (using backup from pre-deployment)
   ```

## Scaling Strategies

### Vertical Scaling

When to scale vertically:
- Application is CPU or memory bound
- Database performance is bottleneck
- Single server setup is preferred

Steps:
1. Monitor resource usage
2. Increase resources on cloud provider dashboard
3. Restart services to utilize new resources

### Horizontal Scaling

When to scale horizontally:
- Need higher availability
- Traffic exceeds single server capacity
- Want to eliminate single points of failure

Steps:
1. Follow [Multiple Servers Deployment](./multiple_servers.md) guide
2. Add more application servers as needed
3. Use load balancer to distribute traffic

## Maintenance Procedures

### Regular Maintenance Tasks

1. **System Updates**:
   ```bash
   # Monthly system updates
   sudo apt update
   sudo apt upgrade -y
   ```

2. **Database Maintenance**:
   ```bash
   # Weekly VACUUM ANALYZE
   sudo -u postgres psql -c "VACUUM ANALYZE;"
   ```

3. **Log Rotation**:
   ```bash
   # Configure logrotate
   sudo nano /etc/logrotate.d/vrooli
   ```
   
   Add:
   ```
   /var/log/vrooli/*.log {
       weekly
       rotate 12
       compress
       delaycompress
       missingok
       notifempty
       create 0640 www-data adm
   }
   ```

4. **SSL Certificate Renewal**:
   ```bash
   # Test renewal
   sudo certbot renew --dry-run
   ```
   
   Certbot typically adds a cron job for automatic renewal.

### Planned Maintenance Windows

For major updates or changes:

1. **Notify users** at least 24 hours in advance
2. **Schedule during low-traffic periods**
3. **Document** maintenance procedures and rollback plans
4. **Test** procedures in staging environment first
5. **Monitor** closely during and after maintenance

## Best Practices

1. **Use Infrastructure as Code** to document and version infrastructure changes
2. **Keep environments consistent** between development, staging, and production
3. **Document everything**, including system architecture, procedures, and incidents
4. **Monitor proactively** to catch issues before they affect users
5. **Test regularly**, including performance and security testing
6. **Train multiple team members** on production procedures
7. **Review logs and metrics** regularly to spot trends
8. **Stay current** with security updates and patches

## Compliance Considerations

If your application handles sensitive data, ensure compliance with relevant regulations:

1. **GDPR**: Data protection for EU citizens
2. **HIPAA**: Health information privacy (if applicable)
3. **PCI DSS**: Payment card security (if processing payments)

Document compliance measures and conduct regular audits. 