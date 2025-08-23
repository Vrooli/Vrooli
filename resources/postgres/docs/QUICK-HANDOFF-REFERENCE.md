# Quick Client Handoff Reference

Fast reference for common PostgreSQL client delivery scenarios.

## 🚀 One-Command Client Delivery

### Standard E-commerce Client
```bash
# Create, configure, and package e-commerce database
./manage.sh --action create --instance client-ecommerce --template ecommerce --yes yes && \
./manage.sh --action create-db --instance client-ecommerce --database store_app --yes yes && \
./manage.sh --action create-user --instance client-ecommerce --username app_user --password "$(openssl rand -base64 32)" --yes yes && \
./manage.sh --action backup --instance client-ecommerce --backup-name "delivery-$(date +%Y%m%d)" --yes yes && \
./manage.sh --action connect --instance client-ecommerce --format json > client-ecommerce-connection.json && \
echo "✅ E-commerce database ready for delivery!"
```

### Real Estate Lead Management
```bash
# Create, configure, and package real estate database
./manage.sh --action create --instance client-realestate --template real-estate --yes yes && \
./manage.sh --action create-db --instance client-realestate --database leads_app --yes yes && \
./manage.sh --action create-user --instance client-realestate --username leads_user --password "$(openssl rand -base64 32)" --yes yes && \
./manage.sh --action backup --instance client-realestate --backup-name "delivery-$(date +%Y%m%d)" --yes yes && \
./manage.sh --action connect --instance client-realestate --format json > client-realestate-connection.json && \
echo "✅ Real estate database ready for delivery!"
```

### SaaS Application Database
```bash
# Create, configure, and package SaaS database
./manage.sh --action create --instance client-saas --template saas --yes yes && \
./manage.sh --action create-db --instance client-saas --database saas_app --yes yes && \
./manage.sh --action create-user --instance client-saas --username saas_user --password "$(openssl rand -base64 32)" --yes yes && \
./manage.sh --action backup --instance client-saas --backup-name "delivery-$(date +%Y%m%d)" --yes yes && \
./manage.sh --action connect --instance client-saas --format json > client-saas-connection.json && \
echo "✅ SaaS database ready for delivery!"
```

## 📋 Quick Validation Checklist

```bash
# Verify instance health
./manage.sh --action status --instance CLIENT_NAME

# Check databases created
./manage.sh --action db-stats --instance CLIENT_NAME

# Verify backup exists
./manage.sh --action list-backups --instance CLIENT_NAME

# Test connection
./manage.sh --action connect --instance CLIENT_NAME
```

## 🔧 Common Client Configurations

### High-Performance E-commerce
```bash
# Optimal for high transaction volume
TEMPLATE="ecommerce"
DATABASES=("products" "orders" "customers" "analytics")
USERS=("app_user" "readonly_user" "analytics_user")
```

### Real Estate CRM
```bash
# Optimal for lead management and property searches
TEMPLATE="real-estate"
DATABASES=("leads" "properties" "contacts" "analytics")
USERS=("crm_user" "readonly_user")
```

### Multi-Tenant SaaS
```bash
# Optimal for SaaS with row-level security
TEMPLATE="saas"
DATABASES=("app_data" "tenant_configs" "usage_metrics")
USERS=("app_user" "metrics_user")
```

## 🎁 Client Delivery Package Generator

```bash
#!/bin/bash
# Quick client package generator

create_client_package() {
    local client_name="$1"
    local template="$2"
    local package_dir="client-${client_name}-$(date +%Y%m%d)"
    
    echo "🚀 Creating client package: $package_dir"
    mkdir -p "$package_dir"/{config,backup,docs}
    
    # Generate connection configs
    ./manage.sh --action connect --instance "$client_name" --format json > "$package_dir/config/connection.json"
    ./manage.sh --action connect --instance "$client_name" --format n8n > "$package_dir/config/n8n-credentials.json"
    
    # Copy backup
    latest_backup=$(./manage.sh --action list-backups --instance "$client_name" | grep -E "^[a-zA-Z]" | head -1 | awk '{print $1}')
    if [[ -n "$latest_backup" ]]; then
        cp -r ~/.vrooli/backups/postgres/"$client_name"/"$latest_backup" "$package_dir/backup/"
    fi
    
    # Generate documentation
    cat > "$package_dir/docs/README.md" << EOF
# $client_name Database Package

## Quick Start
\`\`\`bash
# Start database
docker-compose up -d

# Restore data  
docker exec ${client_name}-postgres psql -U vrooli -d vrooli_client -f /backup/full.sql
\`\`\`

## Connection Details
See \`config/connection.json\` for complete connection information.

## Support
Contact: your-support-email
EOF
    
    # Generate docker-compose.yml
    local port=$(jq -r '.port_external' "$package_dir/config/connection.json")
    local password=$(jq -r '.password' "$package_dir/config/connection.json")
    
    cat > "$package_dir/docker-compose.yml" << EOF
version: '3.8'
services:
  postgres:
    image: postgres:16-alpine
    container_name: ${client_name}-postgres
    environment:
      POSTGRES_DB: vrooli_client
      POSTGRES_USER: vrooli
      POSTGRES_PASSWORD: $password
    ports:
      - "$port:5432"
    volumes:
      - postgres_data:/var/lib/postgresql/data
      - ./backup:/backup:ro
    restart: unless-stopped

volumes:
  postgres_data:
EOF
    
    echo "✅ Client package created: $package_dir"
    echo "📦 Package contents:"
    find "$package_dir" -type f | sort
}

# Usage: create_client_package "client-name" "template"
```

## 🔒 Security Quick Setup

```bash
# Generate secure random password
SECURE_PASSWORD=$(openssl rand -base64 32)

# Create user with limited permissions
./manage.sh --action create-user \
  --instance CLIENT_NAME \
  --username app_user \
  --password "$SECURE_PASSWORD" \
  --yes yes

# Store password securely (example with 1Password CLI)
op item create --category login \
  --title "Client Database - CLIENT_NAME" \
  --username app_user \
  --password "$SECURE_PASSWORD" \
  --url "postgresql://localhost:PORT/DATABASE"
```

## 📊 Performance Monitoring Setup

```bash
# Quick performance check queries
PERFORMANCE_QUERIES="
-- Active connections
SELECT count(*) as active_connections FROM pg_stat_activity WHERE state = 'active';

-- Database size
SELECT pg_size_pretty(pg_database_size(current_database())) as database_size;

-- Top 5 largest tables
SELECT schemaname,tablename,pg_size_pretty(size) as size
FROM (SELECT schemaname,tablename,pg_total_relation_size(schemaname||'.'||tablename) AS size
      FROM pg_tables) AS sizes
ORDER BY size DESC LIMIT 5;

-- Long running queries
SELECT query, now() - query_start as duration
FROM pg_stat_activity 
WHERE state = 'active' AND query_start < now() - interval '5 minutes';
"

# Save monitoring queries for client
echo "$PERFORMANCE_QUERIES" > client-monitoring-queries.sql
```

## 🚨 Emergency Procedures

### Rapid Backup Creation
```bash
# Emergency backup with timestamp
./manage.sh --action backup \
  --instance CLIENT_NAME \
  --backup-name "emergency-$(date +%Y%m%d-%H%M%S)" \
  --yes yes
```

### Quick Instance Recovery
```bash
# Stop, restore, and restart
./manage.sh --action stop --instance CLIENT_NAME --yes yes && \
./manage.sh --action restore --instance CLIENT_NAME --backup-name BACKUP_NAME --yes yes && \
./manage.sh --action start --instance CLIENT_NAME --yes yes
```

### Health Check Script
```bash
#!/bin/bash
# Quick health check for client instances

check_client_health() {
    local client_name="$1"
    
    echo "🏥 Health Check: $client_name"
    echo "================================"
    
    # Instance status
    ./manage.sh --action status --instance "$client_name"
    
    # Connection test
    if ./manage.sh --action connect --instance "$client_name" >/dev/null 2>&1; then
        echo "✅ Connection: OK"
    else
        echo "❌ Connection: FAILED"
    fi
    
    # Disk usage
    local backup_dir=~/.vrooli/backups/postgres/"$client_name"
    if [[ -d "$backup_dir" ]]; then
        echo "💾 Backup disk usage: $(du -sh "$backup_dir" | cut -f1)"
    fi
    
    echo ""
}

# Usage: check_client_health "client-name"
```

---

**⚡ Pro Tip**: Save this file as executable scripts for repeated use:
```bash
chmod +x quick-handoff-scripts.sh
./quick-handoff-scripts.sh create_client_package "my-client" "ecommerce"
```