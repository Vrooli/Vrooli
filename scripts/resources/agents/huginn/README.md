# Huginn Agent-Based Monitoring and Automation

Huginn is a system for building agents that perform automated tasks for you online. Think of it as your own private IFTTT or Zapier, but with a fundamentally different approach: **autonomous agents** that continuously monitor, learn, and react to events.

## 🤖 What Makes Huginn Different?

Unlike traditional workflow automation (n8n, Node-RED), Huginn uses:

- **Autonomous Agents**: Self-contained units that run continuously
- **Event-Driven Architecture**: Agents communicate through events
- **Stateful Processing**: Agents can remember and learn from past events
- **Mesh Communication**: Non-linear agent interactions create emergent behaviors

## 🚀 Quick Start

```bash
# Install Huginn with default settings
./manage.sh --action install

# Install with custom admin credentials
./manage.sh --action install \
  --admin-email your@email.com \
  --admin-password secure_password

# Check status
./manage.sh --action status

# View logs
./manage.sh --action logs
```

Default access:
- **URL**: http://localhost:4111
- **Username**: admin
- **Password**: vrooli_huginn_secure_2025
- **Login Field**: Use username, not email

## 📦 Installation Options

### Basic Installation
```bash
./manage.sh --action install
```

### Custom Configuration
```bash
# Method 1: Environment variables (recommended)
export HUGINN_ADMIN_EMAIL="admin@company.com"
export HUGINN_ADMIN_USERNAME="admin" 
export HUGINN_ADMIN_PASSWORD="V3ryS3cur3P@ss"
./manage.sh --action install

# Method 2: Command line arguments
./manage.sh --action install \
  --admin-email admin@company.com \
  --admin-password "V3ryS3cur3P@ss" \
  --db-password "Str0ngDBP@ss"
```

### Docker Compose Alternative
```bash
# Use docker-compose directly
cd scripts/resources/agents/huginn
docker-compose up -d
```

## 🧩 Core Concepts

### Agents
Self-contained units that perform specific tasks:
- Monitor websites for changes
- Process RSS feeds
- Send notifications
- Transform data
- Make API calls

### Events
Messages passed between agents containing:
- Structured data (JSON)
- Metadata (timestamps, sources)
- Custom fields

### Scenarios
Logical groupings of related agents that work together to accomplish complex tasks.

### Agent Types

#### Input Agents
- **Website Agent**: Monitor web pages for changes
- **RSS Agent**: Parse RSS/Atom feeds
- **Email Agent**: Receive emails
- **Webhook Agent**: Accept HTTP webhooks
- **Twitter Agent**: Monitor Twitter streams

#### Processing Agents
- **Trigger Agent**: Filter events based on rules
- **Event Formatting Agent**: Transform event data
- **JavaScript Agent**: Custom logic with JS code
- **Sentiment Agent**: Analyze text sentiment
- **Translation Agent**: Translate text

#### Output Agents
- **Email Agent**: Send emails
- **Slack Agent**: Post to Slack
- **Webhook Agent**: Send HTTP requests
- **Pushover Agent**: Mobile notifications
- **Twilio Agent**: Send SMS

#### Utility Agents
- **Digest Agent**: Aggregate events
- **Peak Detector Agent**: Detect anomalies
- **De-duplication Agent**: Remove duplicates
- **CSV Agent**: Import/export CSV data

## 📚 Example Use Cases

### 1. Website Monitoring
```bash
# Import website monitor example
./manage.sh --action import --file agents/website-monitor.json
```
Monitor any website for changes and get notified when content updates.

### 2. RSS Intelligence
```bash
# Import RSS aggregator example
./manage.sh --action import --file agents/rss-aggregator.json
```
Aggregate multiple RSS feeds, filter by keywords, and detect trending topics.

### 3. Price Tracking
```bash
# Import price tracker example
./manage.sh --action import --file agents/price-tracker.json
```
Track product prices across e-commerce sites and alert on drops.

### 4. Complete Monitoring Suite
```bash
# Import monitoring scenario
./manage.sh --action import --file scenarios/monitoring-suite.json
```
Comprehensive monitoring of websites, APIs, and security feeds with intelligent alerting.

## 🔧 Management Commands

### Basic Operations
```bash
# Start/stop services
./manage.sh --action start
./manage.sh --action stop
./manage.sh --action restart

# View status and logs
./manage.sh --action status
./manage.sh --action logs
```

### Agent Management
```bash
# List all agents
./manage.sh --action agents

# List scenarios
./manage.sh --action scenarios

# Import agents from JSON
./manage.sh --action import --file my-agents.json

# Export current configuration
./manage.sh --action export --file backup.json
```

### Backup and Restore
```bash
# Create full backup (config + data)
./manage.sh --action backup --file huginn-backup.tar.gz

# Backup configuration only
./manage.sh --action backup --file config-only.tar.gz --include-data no

# Restore from backup
./manage.sh --action restore --file huginn-backup.tar.gz
```

## 🛠️ Advanced Configuration

### Environment Variables
Edit `~/.huginn/.env`:

```bash
# Core Settings
DB_PASSWORD=secure_database_password
HUGINN_PORT=4111
DOMAIN=huginn.yourdomain.com

# Admin Account
ADMIN_EMAIL=admin@yourdomain.com
ADMIN_PASSWORD=very_secure_password

# Email Configuration (for notifications)
SMTP_DOMAIN=yourdomain.com
SMTP_USER_NAME=huginn@yourdomain.com
SMTP_PASSWORD=smtp_password
SMTP_SERVER=smtp.gmail.com
SMTP_PORT=587

# Performance Tuning
WORKER_PROCESSES=4  # Increase for more agents
```

### Security Considerations

1. **Change Default Passwords**: Always change admin and database passwords
2. **Use HTTPS**: Configure reverse proxy with SSL for production
3. **Limit Access**: Use firewall rules to restrict access
4. **Regular Backups**: Schedule automated backups
5. **Monitor Resources**: Agents can consume significant CPU/memory

### Resource Limits

For large deployments, consider:
- PostgreSQL tuning for many events
- Docker memory limits
- Agent scheduling optimization
- Event retention policies

## 🔌 Integration with Vrooli

Huginn is automatically configured in Vrooli's resource registry:

```json
{
  "agents": {
    "huginn": {
      "baseUrl": "http://localhost:4111",
      "capabilities": {
        "autonomousAgents": true,
        "continuousMonitoring": true,
        "eventDriven": true,
        "webhooks": true
      }
    }
  }
}
```

### Webhook Integration
Huginn can receive events from Vrooli:
1. Create a Webhook Agent in Huginn
2. Use the webhook URL in Vrooli automations
3. Process events with Huginn's agent network

### API Access
Access Huginn programmatically:
- REST API for agent management
- Event API for data flow
- Credential API for secure storage

## 🐛 Troubleshooting

### Container Won't Start
```bash
# Check logs
docker logs huginn
docker logs huginn-postgres

# Verify network
docker network ls | grep huginn

# Check ports
lsof -i :4111
```

### Database Connection Issues
```bash
# Test PostgreSQL connection
docker exec huginn-postgres pg_isready -U huginn

# Reset database
docker-compose down -v
docker-compose up -d
```

### Agent Not Working
1. Check agent logs in Huginn UI
2. Verify agent schedule and sources
3. Test with dry run feature
4. Check event propagation

### Memory Issues
```bash
# Monitor container resources
docker stats huginn

# Increase memory limit in docker-compose.yml
deploy:
  resources:
    limits:
      memory: 2G
```

## 📈 Performance Tips

1. **Agent Scheduling**: Spread agent schedules to avoid peaks
2. **Event Retention**: Set appropriate `keep_events_for` values
3. **Database Maintenance**: Regular PostgreSQL vacuuming
4. **Selective Agents**: Disable unused agents
5. **Efficient Queries**: Use specific CSS selectors in Website Agents

## 🤝 Community Resources

- [Official Documentation](https://github.com/huginn/huginn/wiki)
- [Agent Examples](https://github.com/huginn/huginn/wiki/Agent-examples)
- [Huginn Scenarios](https://github.com/huginn/huginn/wiki/Agent-scenarios)
- [Docker Guide](https://github.com/huginn/huginn/tree/master/docker)

## 🎯 Next Steps

1. Access Huginn at http://localhost:4111
2. Change default passwords
3. Import example agents to get started
4. Create your first custom agent
5. Build scenarios for your use cases

Need help? Run `./manage.sh --help` or check the logs with `./manage.sh --action logs`