# Apache Superset - Enterprise Analytics Platform

Apache Superset is a modern data exploration and visualization platform capable of handling data at petabyte scale. It provides an intuitive interface for visualizing datasets and crafting interactive dashboards.

## 🚀 Quick Start

```bash
# Install and start Superset
vrooli resource apache-superset manage install
vrooli resource apache-superset manage start --wait

# Check status
vrooli resource apache-superset status

# Access web UI (default: admin/admin)
open http://localhost:8088

# Note: Dashboard creation via CLI requires CSRF handling
# Use the web UI for creating dashboards and charts
```

## 🎯 Use Cases

### Business Intelligence Dashboards
Create executive-level KPI dashboards combining data from multiple sources for strategic decision making.

### Real-Time Monitoring
Build live dashboards that update as data flows through Vrooli's ecosystem, providing instant visibility.

### Multi-Tenant Analytics  
Generate per-client usage reports and billing dashboards with row-level security.

### Cross-Scenario Intelligence
Combine metrics from energy management, supply chain, and healthcare scenarios into unified views.

## 🔌 Pre-Configured Connections

Superset comes pre-configured with connections to Vrooli's data infrastructure:

- **PostgreSQL** (port 5433): Main relational database
- **QuestDB** (port 9009): Time-series metrics database  
- **MinIO** (port 9000): S3-compatible object storage

## 📊 Key Features

- **Drag & Drop Dashboard Builder**: No-code dashboard creation
- **SQL Lab**: Advanced SQL IDE for data exploration
- **Semantic Layer**: Define business metrics once, use everywhere
- **Rich Visualizations**: 40+ chart types out of the box
- **Caching & Performance**: Redis-powered query caching
- **Scheduled Reports**: Email/Slack delivery of dashboards
- **Row-Level Security**: Fine-grained access control
- **API Access**: Full REST API for automation

## 🛠️ CLI Commands

```bash
# Lifecycle management
vrooli resource apache-superset manage install
vrooli resource apache-superset manage start
vrooli resource apache-superset manage stop
vrooli resource apache-superset manage uninstall

# Content management
vrooli resource apache-superset content list           # List dashboards/charts
vrooli resource apache-superset content add            # Create new content
vrooli resource apache-superset content get            # Export dashboard
vrooli resource apache-superset content remove         # Delete content

# Testing
vrooli resource apache-superset test smoke            # Quick health check
vrooli resource apache-superset test integration      # Full integration test
vrooli resource apache-superset test all              # Complete test suite

# Status and logs
vrooli resource apache-superset status                # Service status
vrooli resource apache-superset logs                  # View logs
```

## 🔧 Configuration

Default configuration is in `config/defaults.sh`. Override with environment variables:

```bash
export SUPERSET_PORT=8088
export SUPERSET_ADMIN_PASSWORD=your-secure-password
export SUPERSET_SECRET_KEY=$(openssl rand -base64 42)
```

## 📝 API Usage

```bash
# Authenticate and get token
TOKEN=$(vrooli resource apache-superset credentials --format json | jq -r .token)

# Create datasource via API
curl -X POST http://localhost:8088/api/v1/database/ \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"database_name": "metrics", "sqlalchemy_uri": "postgresql://user:pass@questdb:8812/qdb"}'

# Export dashboard
curl -X GET http://localhost:8088/api/v1/dashboard/1/export/ \
  -H "Authorization: Bearer $TOKEN" > dashboard.json
```

## 🔗 Integration Examples

### Embed Dashboard in Scenario UI
```javascript
// In your scenario's UI
const dashboardUrl = 'http://localhost:8088/superset/dashboard/1/?standalone=true';
<iframe src={dashboardUrl} width="100%" height="600" />
```

### Trigger n8n Workflow on Alert
Configure alerts in Superset to webhook n8n when thresholds are breached for automated response.

## 🏗️ Architecture

- **Superset App**: Main application server (port 8088)
- **PostgreSQL**: Metadata storage for dashboards, users, queries
- **Redis**: Caching layer for performance and Celery message broker
- **Celery Workers**: Async query execution and scheduled tasks

## 📚 Resources

- [Official Documentation](https://superset.apache.org/docs/intro)
- [API Reference](https://superset.apache.org/docs/api)
- [Security Guide](https://superset.apache.org/docs/security)
- [Chart Types Gallery](https://superset.apache.org/docs/charts/)