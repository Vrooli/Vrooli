# Deployment Guide: Converting Scenarios to Running Applications

## 🎯 From Validated Scenario to Live Application

Once a scenario passes validation, it's ready to become a running application. This guide covers the resource-based scenario-to-app conversion process that transforms test artifacts into live, running applications using existing resource infrastructure.

## 🚀 The Conversion Process

### Overview: Scenario → Live App Pipeline
```
Validated Scenario              Running Application
├── service.json               ├── Running Resources (via manage.sh)
├── test.sh                    │   ├── postgres
├── initialization/            │   ├── n8n
├── deployment/                │   ├── windmill  
└── README.md                  │   └── ollama
                              ├── Injected Data (via inject.sh)
                              │   ├── Database schemas
                              │   ├── Workflows
                              │   └── UI apps
                              └── Application Services
                                  ├── Custom startup scripts
                                  └── Access URLs
```

The conversion process uses existing resource infrastructure to create live applications.

## 🔧 Scenario-to-App Tool

### Basic Usage
```bash
# Convert scenario to running application
./tools/scenario-to-app.sh multi-modal-ai-assistant

# With verbose output
./tools/scenario-to-app.sh multi-modal-ai-assistant --verbose
```

### Advanced Options
```bash
# Preview what would be done without executing
./tools/scenario-to-app.sh secure-document-processing --dry-run

# Keep resources running after errors for debugging
./tools/scenario-to-app.sh secure-document-processing --no-cleanup

# Get help and see all options
./tools/scenario-to-app.sh --help
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

## 🏗️ Running Application Architecture

### Resource-Based Runtime Structure
```
Running Application State:
├── Required Resources (managed by existing scripts)
│   ├── postgres (localhost:5432)    # Started via scripts/resources/storage/postgres/manage.sh
│   ├── n8n (localhost:5678)         # Started via scripts/resources/automation/n8n/manage.sh
│   ├── windmill (localhost:8000)    # Started via scripts/resources/automation/windmill/manage.sh
│   ├── ollama (localhost:11434)     # Started via scripts/resources/ai/ollama/manage.sh
│   └── ... (other resources as needed)
├── Data Injection (via existing inject.sh scripts)
│   ├── Database schemas and seeds   # Injected via postgres/inject.sh
│   ├── n8n workflows               # Injected via n8n/inject.sh
│   ├── Windmill applications       # Injected via windmill/inject.sh
│   └── Configuration files         # Loaded from scenario/initialization/
├── Application Services
│   ├── Custom startup scripts      # From scenario/deployment/startup.sh
│   ├── Health monitoring          # Via resource manage.sh status commands
│   └── Access point URLs          # Provided by scenario-to-app.sh
└── Scenario-Specific Features
    ├── Business logic (via workflows)
    ├── User interfaces (via Windmill)
    ├── Data processing (via n8n)
    └── AI capabilities (via Ollama/Whisper)
```

## ⚙️ Resource Optimization

### Optimized Resource Configuration
The conversion process analyzes scenario requirements and generates optimized `service.json`:

```json
{
  "resources": {
    "ai": {
      "ollama": {
        "enabled": true,
        "baseUrl": "http://localhost:11434",
        "expectedModels": ["llama3.1:8b"],
        "type": "ollama"
      }
    },
    "storage": {
      "postgres": {
        "enabled": true,
        "type": "postgresql",
        "host": "localhost",
        "port": 5432,
        "database": "customer_app"
      },
      "qdrant": {
        "enabled": true,
        "type": "qdrant",
        "baseUrl": "http://localhost:6333"
      }
    },
    "automation": {
      "n8n": {
        "enabled": true,
        "type": "n8n",
        "baseUrl": "http://localhost:5678"
      }
    }
  }
}
```

### Resource Configuration
Resource scaling and configuration is handled by the individual resource manage.sh scripts:

```bash
# Configure resource limits before starting
export OLLAMA_MAX_MEMORY="8GB"
export POSTGRES_MAX_CONNECTIONS="200"
export POSTGRES_SHARED_BUFFERS="1GB"

# Start resources with configuration
./tools/scenario-to-app.sh your-scenario-name

# Resources inherit configuration from:
# - Environment variables
# - Resource-specific config files in scripts/resources/
# - Scenario-specific settings in .vrooli/service.json
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
Monitoring is built into the resource management system:

```bash
# Check overall application health
./tools/scenario-to-app.sh your-scenario --dry-run  # Preview health checks

# Monitor individual resource health
./scripts/resources/ai/ollama/manage.sh --action status
./scripts/resources/storage/postgres/manage.sh --action status
./scripts/resources/automation/n8n/manage.sh --action status

# View resource logs
./scripts/resources/ai/ollama/manage.sh --action logs
./scripts/resources/automation/windmill/manage.sh --action logs

# Access built-in monitoring UIs
# - n8n workflows: http://localhost:5678
# - Windmill apps: http://localhost:8000
# - Resource-specific monitoring endpoints
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
    
    # Stop all resources gracefully
    ./scripts/resources/automation/n8n/manage.sh --action stop
    ./scripts/resources/storage/postgres/manage.sh --action stop
    ./scripts/resources/ai/ollama/manage.sh --action stop
    
    # Restore database
    psql "${DATABASE_URL}" < "${BACKUP_DIR}/database.sql"
    
    # Restore configuration files
    tar -xzf "${BACKUP_DIR}/config.tar.gz" -C ./
    
    # Restore workflows and data
    tar -xzf "${BACKUP_DIR}/workflows.tar.gz" -C data/
    
    # Restart application using scenario-to-app
    ./tools/scenario-to-app.sh "${SCENARIO_NAME}"
    
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
    command -v jq >/dev/null 2>&1 || { echo "jq required but not installed"; exit 1; }
    command -v curl >/dev/null 2>&1 || { echo "curl required but not installed"; exit 1; }
    [[ -d "./scripts/resources" ]] || { echo "Vrooli resource directory not found"; exit 1; }
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
    echo "🔧 Starting application using resource infrastructure..."
    
    # Use scenario-to-app.sh to start all resources
    if [[ -f "./tools/scenario-to-app.sh" ]]; then
        ./tools/scenario-to-app.sh "$(basename "$PWD")"
    else
        echo "❌ scenario-to-app.sh not found. Please run from Vrooli root directory."
        exit 1
    fi
    
    echo "🎉 Application is ready!"
    echo "📱 Resource access points will be displayed by scenario-to-app.sh"
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
   - Application endpoints will be displayed during startup
   - n8n workflows: http://localhost:5678
   - Windmill UI: http://localhost:8000
   - Individual resource interfaces as shown by the startup script

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
- **Logs**: Use resource-specific log commands shown during startup
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
    
    # Stop current resources
    killall scenario-to-app.sh || true
    
    # Update Vrooli resource scripts
    git pull origin main
    
    # Restart application
    ./tools/scenario-to-app.sh "${SCENARIO_NAME}"
    
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