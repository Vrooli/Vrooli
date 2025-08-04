# Deployment Guide: Converting Scenarios to Customer Applications

## 🎯 From Validated Scenario to Revenue-Generating App

Once a scenario passes validation, it's ready to become a customer application. This guide covers the complete scenario-to-app conversion process that transforms test artifacts into production-ready deployments.

## 🚀 The Conversion Process

### Overview: Scenario → App Pipeline
```
Validated Scenario              Customer Application
├── metadata.yaml              ├── service.json (minimal)
├── test.sh                    ├── docker-compose.yml
├── initialization/            ├── startup.sh
├── deployment/                ├── monitoring/
└── docs/                      ├── customer-docs/
                              └── support/
```

The conversion process preserves all business functionality while optimizing for customer deployment.

## 🔧 Scenario-to-App Tool

### Basic Usage
```bash
# Convert scenario to deployable app
./tools/scenario-to-app.sh \
  --scenario multi-modal-ai-assistant \
  --output ~/deployments/customer-app

# With customization
./tools/scenario-to-app.sh \
  --scenario multi-modal-ai-assistant \
  --output ~/deployments/customer-app \
  --config customer-config.yaml \
  --branding customer-branding.json
```

### Advanced Options
```bash
# Full customization example
./tools/scenario-to-app.sh \
  --scenario document-intelligence-pipeline \
  --output ~/deployments/legal-firm-app \
  --config configs/legal-firm.yaml \
  --branding branding/legal-firm.json \
  --environment production \
  --monitoring enabled \
  --backup-schedule "0 2 * * *" \
  --ssl-cert /path/to/cert.pem \
  --domain legal-firm.example.com
```

## 📋 Configuration Options

### Customer Configuration (`customer-config.yaml`)
```yaml
customer:
  name: "Acme Legal Services"
  domain: "legal.acme.com"
  contact: "admin@acme.com"

application:
  name: "Legal Document Intelligence"
  description: "AI-powered legal document processing"
  version: "1.0.0"

environment:
  type: "production"  # development, staging, production
  scaling:
    min_instances: 2
    max_instances: 10
    cpu_threshold: 70

resources:
  # Override default resource configuration
  postgres:
    max_connections: 200
    shared_buffers: "1GB"
  ollama:
    models: ["llama3.1:8b", "codellama:7b"]
    max_memory: "8GB"

security:
  ssl: true
  auth_required: true
  jwt_secret: "${CUSTOMER_JWT_SECRET}"
  allowed_origins: ["https://legal.acme.com"]

monitoring:
  enabled: true
  alerts:
    email: "ops@acme.com"
    slack_webhook: "${SLACK_WEBHOOK_URL}"

backup:
  enabled: true
  schedule: "0 2 * * *"  # Daily at 2 AM
  retention_days: 30
  s3_bucket: "acme-legal-backups"
```

### Branding Configuration (`customer-branding.json`)
```json
{
  "colors": {
    "primary": "#1e40af",
    "secondary": "#64748b",
    "accent": "#f59e0b"
  },
  "logo": {
    "url": "https://acme.com/logo.png",
    "alt": "Acme Legal Services"
  },
  "fonts": {
    "heading": "Inter",
    "body": "system-ui"
  },
  "theme": {
    "mode": "professional",
    "sidebar": "dark",
    "animations": false
  },
  "content": {
    "app_name": "Legal Document Intelligence",
    "tagline": "AI-Powered Legal Document Processing",
    "support_email": "support@acme.com",
    "help_url": "https://acme.com/help"
  }
}
```

## 🏗️ Generated Application Structure

### Application Directory Layout
```
customer-app/
├── docker-compose.yml         # Main deployment configuration
├── .env.example              # Environment variables template
├── startup.sh                # Application startup script
├── service.json      # Minimal resource configuration
├── docs/                     # Customer documentation
│   ├── README.md             # Getting started guide
│   ├── CONFIGURATION.md      # Configuration options
│   ├── TROUBLESHOOTING.md    # Common issues and solutions
│   └── API.md                # API documentation
├── config/                   # Application configuration
│   ├── app-config.json       # Runtime application settings
│   ├── nginx.conf            # Web server configuration
│   └── monitoring.yml        # Monitoring configuration
├── scripts/                  # Operational scripts
│   ├── backup.sh             # Backup procedures
│   ├── restore.sh            # Restore procedures
│   ├── update.sh             # Update procedures
│   └── health-check.sh       # Health monitoring
├── data/                     # Application data
│   ├── database/             # Database initialization
│   ├── workflows/            # Deployed workflows
│   ├── ui/                   # UI applications
│   └── storage/              # File storage setup
└── monitoring/               # Monitoring and logging
    ├── prometheus.yml        # Metrics configuration
    ├── grafana-dashboard.json # Monitoring dashboard
    └── alerting-rules.yml    # Alert configurations
```

## ⚙️ Resource Optimization

### Minimal Resource Configuration
The conversion process analyzes scenario requirements and generates optimized `service.json`:

```json
{
  "services": {
    "ai": {
      "ollama": {
        "enabled": true,
        "baseUrl": "http://localhost:11434",
        "models": ["llama3.1:8b"],
        "max_memory": "4gb"
      }
    },
    "storage": {
      "postgres": {
        "enabled": true,
        "baseUrl": "postgres://localhost:5432",
        "database": "customer_app",
        "max_connections": 100
      },
      "qdrant": {
        "enabled": true,
        "baseUrl": "http://localhost:6333",
        "collection": "documents"
      }
    },
    "automation": {
      "n8n": {
        "enabled": true,
        "baseUrl": "http://localhost:5678",
        "workflows": ["document-processing"]
      }
    }
  }
}
```

### Resource Scaling Configuration
```yaml
# docker-compose.yml scaling section
services:
  ollama:
    deploy:
      replicas: 2
      resources:
        limits:
          memory: 4g
          cpus: '2'
        reservations:
          memory: 2g
          cpus: '1'
          
  postgres:
    deploy:
      resources:
        limits:
          memory: 2g
          cpus: '1'
        reservations:
          memory: 1g
          cpus: '0.5'
```

## 🔒 Security Configuration

### SSL/TLS Setup
```bash
# Generate SSL certificates
./scripts/setup-ssl.sh \
  --domain customer.example.com \
  --email admin@customer.com

# Configure nginx for SSL
nginx_ssl_config() {
  cat > config/nginx.conf << EOF
server {
    listen 443 ssl;
    server_name customer.example.com;
    
    ssl_certificate /etc/ssl/certs/customer.crt;
    ssl_certificate_key /etc/ssl/private/customer.key;
    
    location / {
        proxy_pass http://app:3000;
        proxy_set_header Host \$host;
        proxy_set_header X-Real-IP \$remote_addr;
    }
}
EOF
}
```

### Authentication & Authorization
```yaml
# config/auth-config.yml
auth:
  provider: "local"  # local, oauth2, saml
  jwt:
    secret: "${JWT_SECRET}"
    expiry: "24h"
  
  roles:
    admin:
      permissions: ["*"]
    user:
      permissions: ["read", "create", "update"]
    viewer:
      permissions: ["read"]

  password_policy:
    min_length: 8
    require_uppercase: true
    require_numbers: true
    require_special_chars: true
```

## 📊 Monitoring & Observability

### Monitoring Stack
```yaml
# monitoring/docker-compose.monitoring.yml
version: '3.8'
services:
  prometheus:
    image: prom/prometheus:latest
    volumes:
      - ./prometheus.yml:/etc/prometheus/prometheus.yml
    ports:
      - "9090:9090"
      
  grafana:
    image: grafana/grafana:latest
    environment:
      - GF_SECURITY_ADMIN_PASSWORD=${GRAFANA_PASSWORD}
    volumes:
      - ./grafana-dashboard.json:/var/lib/grafana/dashboards/
    ports:
      - "3001:3000"
      
  alertmanager:
    image: prom/alertmanager:latest
    volumes:
      - ./alerting-rules.yml:/etc/alertmanager/alertmanager.yml
    ports:
      - "9093:9093"
```

### Application Metrics
```javascript
// Application metrics collection
const metrics = {
  // Business metrics
  documents_processed: counter('documents_processed_total'),
  processing_time: histogram('document_processing_duration_seconds'),
  user_sessions: gauge('active_user_sessions'),
  
  // Technical metrics
  api_requests: counter('api_requests_total'),
  api_response_time: histogram('api_response_duration_seconds'),
  database_connections: gauge('database_connections_active'),
  
  // Resource metrics
  ollama_requests: counter('ollama_requests_total'),
  ollama_response_time: histogram('ollama_response_duration_seconds'),
  vector_searches: counter('vector_searches_total')
};
```

## 🔄 Backup & Recovery

### Automated Backup Setup
```bash
#!/bin/bash
# scripts/backup.sh

backup_application() {
    BACKUP_DATE=$(date +%Y%m%d_%H%M%S)
    BACKUP_DIR="/backups/${BACKUP_DATE}"
    
    echo "Starting backup: ${BACKUP_DATE}"
    
    # Backup database
    pg_dump "${DATABASE_URL}" > "${BACKUP_DIR}/database.sql"
    
    # Backup file storage
    tar -czf "${BACKUP_DIR}/files.tar.gz" data/storage/
    
    # Backup configuration
    tar -czf "${BACKUP_DIR}/config.tar.gz" config/
    
    # Backup workflows
    tar -czf "${BACKUP_DIR}/workflows.tar.gz" data/workflows/
    
    # Upload to S3 (if configured)
    if [[ -n "$S3_BACKUP_BUCKET" ]]; then
        aws s3 sync "${BACKUP_DIR}" "s3://${S3_BACKUP_BUCKET}/backups/${BACKUP_DATE}/"
    fi
    
    echo "Backup completed: ${BACKUP_DATE}"
}

# Schedule with cron
# 0 2 * * * /path/to/backup.sh
```

### Recovery Procedures
```bash
#!/bin/bash
# scripts/restore.sh

restore_application() {
    BACKUP_DATE=$1
    BACKUP_DIR="/backups/${BACKUP_DATE}"
    
    echo "Restoring from backup: ${BACKUP_DATE}"
    
    # Stop application
    docker-compose down
    
    # Restore database
    psql "${DATABASE_URL}" < "${BACKUP_DIR}/database.sql"
    
    # Restore files
    tar -xzf "${BACKUP_DIR}/files.tar.gz" -C data/
    
    # Restore configuration
    tar -xzf "${BACKUP_DIR}/config.tar.gz" -C ./
    
    # Restore workflows
    tar -xzf "${BACKUP_DIR}/workflows.tar.gz" -C data/
    
    # Start application
    docker-compose up -d
    
    echo "Restore completed: ${BACKUP_DATE}"
}
```

## 📱 Customer Experience

### One-Click Deployment
```bash
#!/bin/bash
# startup.sh - Customer deployment script

echo "🚀 Starting Customer Application Deployment..."

# Pre-flight checks
check_requirements() {
    command -v docker >/dev/null 2>&1 || { echo "Docker required but not installed"; exit 1; }
    command -v docker-compose >/dev/null 2>&1 || { echo "Docker Compose required but not installed"; exit 1; }
}

# Environment setup
setup_environment() {
    if [[ ! -f .env ]]; then
        cp .env.example .env
        echo "📝 Please edit .env file with your configuration"
        exit 1
    fi
}

# Application startup
start_application() {
    echo "🔧 Starting application services..."
    docker-compose up -d
    
    echo "⏳ Waiting for services to be ready..."
    ./scripts/health-check.sh --wait
    
    echo "🎉 Application is ready!"
    echo "📱 Access your application at: http://localhost"
    echo "📊 Monitoring dashboard: http://localhost:3001"
    echo "📚 Documentation: docs/README.md"
}

# Main execution
main() {
    check_requirements
    setup_environment
    start_application
}

main "$@"
```

### Customer Documentation
```markdown
# Getting Started with Your AI Application

## Quick Start

1. **Copy environment file**:
   ```bash
   cp .env.example .env
   ```

2. **Edit configuration**:
   ```bash
   nano .env  # Add your settings
   ```

3. **Start application**:
   ```bash
   ./startup.sh
   ```

4. **Access your application**:
   - Application: http://localhost
   - Admin panel: http://localhost/admin
   - Monitoring: http://localhost:3001

## Configuration

### Required Settings
- `DATABASE_URL`: Database connection string
- `JWT_SECRET`: Authentication secret key
- `ADMIN_EMAIL`: Administrator email address

### Optional Settings
- `SSL_ENABLED`: Enable HTTPS (true/false)
- `BACKUP_ENABLED`: Enable automated backups (true/false)
- `MONITORING_ENABLED`: Enable monitoring stack (true/false)

## Support

- **Email**: support@yourcompany.com
- **Documentation**: docs/
- **Health Check**: ./scripts/health-check.sh
- **Logs**: docker-compose logs -f
```

## 🔄 Update & Maintenance

### Application Updates
```bash
#!/bin/bash
# scripts/update.sh

update_application() {
    echo "🔄 Updating application..."
    
    # Backup current state
    ./scripts/backup.sh
    
    # Pull latest images
    docker-compose pull
    
    # Restart with new images
    docker-compose up -d
    
    # Verify health
    ./scripts/health-check.sh
    
    echo "✅ Update completed"
}
```

### Health Monitoring
```bash
#!/bin/bash
# scripts/health-check.sh

check_application_health() {
    echo "🏥 Checking application health..."
    
    # Check core services
    check_service "postgres" "5432"
    check_service "ollama" "11434"
    check_service "app" "3000"
    
    # Check business functionality
    test_api_endpoints
    test_ai_responses
    test_database_connectivity
    
    # Check resource usage
    check_resource_usage
    
    echo "✅ Health check completed"
}

check_service() {
    local service=$1
    local port=$2
    
    if nc -z localhost "$port"; then
        echo "✅ $service is healthy"
    else
        echo "❌ $service is not responding on port $port"
        exit 1
    fi
}
```

## 💼 Business Considerations

### Pricing Models
```yaml
# Different deployment tiers
pricing_tiers:
  starter:
    price: "$299/month"
    features:
      - Basic AI processing
      - Standard UI
      - Email support
    limits:
      documents_per_month: 1000
      users: 5
      
  professional:
    price: "$999/month"
    features:
      - Advanced AI processing
      - Custom branding
      - Priority support
      - SSL included
    limits:
      documents_per_month: 10000
      users: 50
      
  enterprise:
    price: "Custom"
    features:
      - Full customization
      - On-premise deployment
      - 24/7 support
      - SLA guarantee
    limits:
      documents_per_month: "Unlimited"
      users: "Unlimited"
```

### Customer Onboarding
```bash
# Customer onboarding checklist
onboarding_checklist() {
    echo "📋 Customer Onboarding Checklist:"
    echo "□ Application deployed successfully"
    echo "□ Customer can access the application"
    echo "□ Admin account created and tested"
    echo "□ Basic functionality demonstrated"
    echo "□ Monitoring alerts configured"
    echo "□ Backup procedures verified"
    echo "□ Customer documentation provided"
    echo "□ Support contacts shared"
    echo "□ Training session scheduled"
    echo "□ First invoice sent"
}
```

## 🎯 Success Metrics

### Deployment Success Indicators
- ✅ Application starts without errors
- ✅ All health checks pass
- ✅ Customer can access UI
- ✅ Basic workflows function
- ✅ Monitoring shows green status
- ✅ Customer receives welcome email

### Business Success Indicators
- 💰 Customer pays first invoice
- 📈 Usage metrics show adoption
- 😊 Customer satisfaction survey > 8/10
- 🔄 Customer requests additional features
- 📞 Minimal support tickets (< 5 in first month)

## 🚀 Next Steps

**Ready to deploy customer applications?**

1. **Test the Process**: Convert a simple scenario to understand the workflow
2. **Customize Templates**: Adapt deployment templates for your business
3. **Automate Operations**: Set up monitoring, backups, and updates
4. **Scale Operations**: Process multiple customer deployments efficiently

The scenario-to-app conversion process transforms validated scenarios into revenue-generating customer applications, completing the journey from idea to deployed business value.

**Next**: Explore specific scenarios in the [core directory](../core/) or dive into [advanced customization topics](ADVANCED.md).