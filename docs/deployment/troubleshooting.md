# Deployment Troubleshooting Guide

This guide provides solutions for common issues you might encounter when deploying Vrooli in different environments.

## Table of Contents

- [Docker Issues](#docker-issues)
- [Database Issues](#database-issues)
- [Networking Issues](#networking-issues)
- [CI/CD Pipeline Issues](#cicd-pipeline-issues)
- [Performance Issues](#performance-issues)
- [SSL/TLS Issues](#ssltls-issues)
- [Environment Variables Issues](#environment-variables-issues)

## Docker Issues

### Containers Not Starting

**Symptoms:**
- Docker containers remain in Created or Exited state
- Error messages in logs about permissions or configuration

**Solutions:**

1. Check container status and logs:
   ```bash
   docker ps -a
   docker logs container_name
   ```

2. Verify disk space:
   ```bash
   df -h
   ```

3. Check Docker service:
   ```bash
   sudo systemctl status docker
   ```

4. Reset Docker if necessary:
   ```bash
   sudo systemctl restart docker
   ```

### Image Pull Errors

**Symptoms:**
- Error messages about failed image pulls
- "Image not found" or authentication errors

**Solutions:**

1. Verify network connectivity:
   ```bash
   ping docker.io
   ```

2. Check Docker Hub authentication:
   ```bash
   docker login
   ```

3. Try pulling the image manually:
   ```bash
   docker pull image_name:tag
   ```

### Container Networking Issues

**Symptoms:**
- Containers cannot communicate with each other
- "Connection refused" errors

**Solutions:**

1. Inspect Docker networks:
   ```bash
   docker network ls
   docker network inspect network_name
   ```

2. Verify container connectivity:
   ```bash
   docker exec container_name ping other_container
   ```

3. Check if ports are exposed correctly:
   ```bash
   docker port container_name
   ```

## Database Issues

### Connection Failures

**Symptoms:**
- Application logs show database connection errors
- "Connection refused" or timeout errors

**Solutions:**

1. Verify database container is running:
   ```bash
   docker ps | grep db
   ```

2. Check database logs:
   ```bash
   docker logs db
   ```

3. Verify connection details:
   ```bash
   # Test PostgreSQL connection
   psql -h hostname -U username -d database_name
   ```

4. Check network connectivity:
   ```bash
   telnet hostname 5432
   ```

### Migration Issues

**Symptoms:**
- Errors about missing tables or columns
- Schema version mismatches

**Solutions:**

1. Check migration logs:
   ```bash
   # In the server container logs
   docker logs server | grep migration
   ```

2. Manually apply migrations:
   ```bash
   docker exec -it server /bin/bash
   cd packages/server
   yarn run migrate
   ```

3. Verify database schema:
   ```bash
   # Connect to PostgreSQL and list tables
   psql -h hostname -U username -d database_name -c "\dt"
   ```

### Database Performance Issues

**Symptoms:**
- Slow query response times
- High CPU usage in database container

**Solutions:**

1. Check query execution times:
   ```sql
   SELECT query, calls, total_time, mean_time 
   FROM pg_stat_statements 
   ORDER BY total_time DESC 
   LIMIT 10;
   ```

2. Analyze table statistics:
   ```sql
   ANALYZE;
   ```

3. Check for missing indexes:
   ```sql
   SELECT relname, seq_scan, idx_scan 
   FROM pg_stat_user_tables 
   ORDER BY seq_scan DESC;
   ```

## Networking Issues

### Load Balancer Issues

**Symptoms:**
- 502 Bad Gateway or 504 Gateway Timeout errors
- Inconsistent access to application

**Solutions:**

1. Check load balancer configuration:
   ```bash
   sudo nano /etc/nginx/sites-available/vrooli
   sudo nginx -t
   ```

2. Verify backend services are running:
   ```bash
   curl http://backend_ip:port/healthcheck
   ```

3. Check NGINX error logs:
   ```bash
   sudo tail -f /var/log/nginx/error.log
   ```

### DNS Resolution Issues

**Symptoms:**
- Domain name doesn't resolve to correct IP
- Unable to access application via domain name

**Solutions:**

1. Check DNS records:
   ```bash
   dig your_domain.com
   ```

2. Verify DNS propagation:
   ```bash
   dig your_domain.com @8.8.8.8
   ```

3. Check local DNS resolution:
   ```bash
   cat /etc/resolv.conf
   ```

### Firewall Issues

**Symptoms:**
- Connection timeouts
- Services unreachable despite being online

**Solutions:**

1. Check firewall status:
   ```bash
   sudo ufw status
   ```

2. Verify required ports are open:
   ```bash
   sudo ufw allow 80/tcp
   sudo ufw allow 443/tcp
   sudo ufw allow 5432/tcp  # For PostgreSQL
   sudo ufw allow 6379/tcp  # For Redis
   ```

3. Test connectivity:
   ```bash
   telnet hostname port
   ```

## CI/CD Pipeline Issues

### GitHub Actions Failures

**Symptoms:**
- Pipeline fails in specific steps
- Error messages in GitHub Actions logs

**Solutions:**

1. Check logs in GitHub Actions interface

2. Verify secret configuration:
   - Ensure all required secrets are set in GitHub repository settings
   - Verify secret values are correct (without exposing them)

3. Test commands locally:
   ```bash
   # Try building locally
   ./scripts/build.sh -v test-version -t y
   ```

### Deployment Script Failures

**Symptoms:**
- Deployment script fails with error messages
- Script completes but application doesn't work

**Solutions:**

1. Run script with extra verbosity:
   ```bash
   bash -x ./scripts/deploy.sh
   ```

2. Check environment variables:
   ```bash
   # Verify environment file exists
   cat .env-prod
   ```

3. Ensure all dependencies are installed:
   ```bash
   # Check for common dependencies
   command -v docker docker-compose git jq curl
   ```

### SSH Connection Issues

**Symptoms:**
- "Permission denied" errors during deployment
- SSH connection failures

**Solutions:**

1. Verify SSH key permissions:
   ```bash
   chmod 600 ~/.ssh/id_ed25519
   ```

2. Check SSH configuration:
   ```bash
   cat ~/.ssh/config
   ```

3. Test SSH connection manually:
   ```bash
   ssh -v username@hostname
   ```

4. Check authorized_keys on the server:
   ```bash
   cat ~/.ssh/authorized_keys
   ```

## Performance Issues

### Slow Application Response

**Symptoms:**
- Pages load slowly
- API requests take too long

**Solutions:**

1. Check container resources:
   ```bash
   docker stats
   ```

2. Monitor system resources:
   ```bash
   top
   htop
   ```

3. Check application logs for slow operations:
   ```bash
   docker logs server | grep "slow"
   ```

4. Increase container resources:
   ```yaml
   # In docker-compose.yml
   services:
     server:
       deploy:
         resources:
           limits:
             cpus: '2'
             memory: 2G
   ```

### Memory Leaks

**Symptoms:**
- Increasing memory usage over time
- Container crashes with OOM (Out Of Memory) errors

**Solutions:**

1. Monitor memory usage:
   ```bash
   docker stats
   ```

2. Restart containers periodically:
   ```bash
   # Add to cron
   0 3 * * * docker restart container_name
   ```

3. Increase swap space:
   ```bash
   sudo fallocate -l 2G /swapfile
   sudo chmod 600 /swapfile
   sudo mkswap /swapfile
   sudo swapon /swapfile
   ```

## SSL/TLS Issues

### Certificate Errors

**Symptoms:**
- Browser shows "Your connection is not private"
- SSL handshake failures

**Solutions:**

1. Verify certificate expiration:
   ```bash
   openssl x509 -enddate -noout -in /etc/letsencrypt/live/your_domain.com/cert.pem
   ```

2. Check certificate chain:
   ```bash
   openssl verify -CAfile /etc/letsencrypt/live/your_domain.com/chain.pem /etc/letsencrypt/live/your_domain.com/cert.pem
   ```

3. Renew Let's Encrypt certificate:
   ```bash
   sudo certbot renew
   ```

### Mixed Content Warnings

**Symptoms:**
- Browser console shows mixed content warnings
- Some resources load over HTTP instead of HTTPS

**Solutions:**

1. Update application to use HTTPS URLs:
   ```bash
   grep -r "http://" --include="*.ts" --include="*.tsx" ./packages
   ```

2. Configure NGINX to redirect HTTP to HTTPS:
   ```nginx
   server {
     listen 80;
     server_name your_domain.com;
     return 301 https://$host$request_uri;
   }
   ```

## Environment Variables Issues

### Missing Environment Variables

**Symptoms:**
- Application shows errors about missing configuration
- Certain features don't work

**Solutions:**

1. Verify environment file exists:
   ```bash
   ls -la .env-prod
   ```

2. Check for missing variables:
   ```bash
   # Compare with example
   diff -y .env-example .env-prod
   ```

3. Set environment variables manually:
   ```bash
   export VARIABLE_NAME=value
   ```

### Secrets Management Issues

**Symptoms:**
- Application can't access secrets
- Vault connection errors

**Solutions:**

1. Check Vault status:
   ```bash
   vault status
   ```

2. Verify token validity:
   ```bash
   VAULT_TOKEN=your_token vault token lookup
   ```

3. Test secret retrieval:
   ```bash
   VAULT_TOKEN=your_token vault kv get secret/path/to/secret
   ```

4. Restart Vault if needed:
   ```bash
   sudo systemctl restart vault
   ```

## Still Having Issues?

If you've tried the solutions above and are still experiencing problems:

1. Check the project's GitHub Issues to see if others have encountered similar problems
2. Create a detailed issue report with:
   - Exact error messages
   - Steps to reproduce
   - Environment details (OS, Docker version, etc.)
   - Logs from relevant containers

Remember that more information leads to faster solutions! 