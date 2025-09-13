# Home Assistant Resource

Open-source home automation platform that puts local control and privacy first.

## 🚀 Quick Start

```bash
# Install and start Home Assistant
vrooli resource home-assistant manage install
vrooli resource home-assistant manage start --wait

# Check status
vrooli resource home-assistant status

# View logs
vrooli resource home-assistant logs --tail 50

# Run tests
vrooli resource home-assistant test all
```

## 📋 Features

- **Local Control**: Full control of smart home devices without cloud dependency
- **Privacy-Focused**: Your data stays on your local network
- **Extensive Integrations**: 2000+ supported devices and services
- **Automation Engine**: Powerful automation with triggers, conditions, and actions
- **Energy Management**: Track and optimize energy consumption
- **Voice Control**: Compatible with various voice assistants
- **Web Interface**: Modern, responsive UI accessible from any browser

## 🔧 Configuration

### Environment Variables

```bash
# Container settings
HOME_ASSISTANT_CONTAINER_NAME=home-assistant
HOME_ASSISTANT_IMAGE=homeassistant/home-assistant:stable

# Network settings
HOME_ASSISTANT_PORT=8123
HOME_ASSISTANT_BASE_URL=http://localhost:8123

# Data directories
HOME_ASSISTANT_DATA_DIR=${var_DATA_DIR}/resources/home-assistant
HOME_ASSISTANT_CONFIG_DIR=${HOME_ASSISTANT_DATA_DIR}/config

# Runtime settings
HOME_ASSISTANT_TIME_ZONE=America/New_York
HOME_ASSISTANT_RESTART_POLICY=unless-stopped
```

### Secrets Management

Secrets are managed through `config/secrets.yaml`:

```yaml
secrets:
  api_keys:
    - name: "long_lived_token"
      path: "secret/resources/home-assistant/api/token"
      description: "Long-lived access token for API authentication"
      required: false
```

To configure secrets:
```bash
# Check secret status
vrooli resource vault secrets check home-assistant

# Initialize secrets
vrooli resource vault secrets init home-assistant
```

## 🔌 API Access

### Authentication

Home Assistant starts with no authentication by default. To secure it:

1. Access the web UI at http://localhost:8123
2. Create your admin account on first access
3. Generate a long-lived access token:
   - Go to Profile → Security → Long-Lived Access Tokens
   - Create a new token and save it securely

### API Examples

```bash
# Check API status (no auth required initially)
curl http://localhost:8123/api/

# With authentication token
curl -H "Authorization: Bearer YOUR_TOKEN" \
  http://localhost:8123/api/states

# Call a service
curl -X POST -H "Authorization: Bearer YOUR_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"entity_id": "light.living_room"}' \
  http://localhost:8123/api/services/light/turn_on
```

## 💾 Backup & Restore

### Creating Backups

```bash
# Create a backup of current configuration
vrooli resource home-assistant backup

# List available backups
vrooli resource home-assistant backup list

# Backups are automatically rotated (keeps last 5)
```

### Restoring Backups

```bash
# List available backups to restore from
vrooli resource home-assistant restore

# Restore a specific backup
vrooli resource home-assistant restore /path/to/backup_20250912_190857.tar.gz
```

## 🔗 Webhook Support

Home Assistant can receive webhooks for external integrations:

```bash
# Send a webhook (no auth required for webhook endpoints)
curl -X POST http://localhost:8123/api/webhook/your_webhook_id \
  -H "Content-Type: application/json" \
  -d '{"event": "button_pressed", "data": {"button": "doorbell"}}'
```

Configure webhooks in automations to trigger actions from external systems.

## 🤖 Automation Management

### Adding Automations

```bash
# Add a YAML automation file
vrooli resource home-assistant content add path/to/automation.yaml

# List all injected files
vrooli resource home-assistant content list

# Clear all automations (requires --force)
vrooli resource home-assistant content remove --force

# Reload automations without restart
vrooli resource home-assistant content reload
```

### Example Automation

```yaml
# motion_light.yaml
alias: Motion Light
description: Turn on light when motion detected
trigger:
  - platform: state
    entity_id: binary_sensor.motion_sensor
    to: 'on'
condition:
  - condition: sun
    after: sunset
action:
  - service: light.turn_on
    target:
      entity_id: light.hallway
```

## 🏗️ Integration with Other Resources

### PostgreSQL (Recorder Backend)

```yaml
# In Home Assistant configuration.yaml
recorder:
  db_url: postgresql://user:password@postgres:5432/homeassistant
  purge_keep_days: 30
```

### MQTT (Device Communication)

```yaml
mqtt:
  broker: localhost
  port: 1883
  username: mqtt_user
  password: !secret mqtt_password
```

### Redis (Session Cache)

```yaml
http:
  session_cache:
    type: redis
    host: localhost
    port: 6379
```

## 🧪 Testing

### Test Phases

```bash
# Quick health check (< 30s)
vrooli resource home-assistant test smoke

# Full functionality test (< 120s)
vrooli resource home-assistant test integration

# Library function tests (< 60s)
vrooli resource home-assistant test unit

# Run all tests
vrooli resource home-assistant test all
```

### Health Checks

The resource provides multiple health check endpoints:

- **Container Health**: Checks if Docker container is running
- **API Health**: Verifies API endpoint responds (401 or 200)
- **Web UI Health**: Ensures web interface is accessible
- **Service Health**: Validates core services are operational

## 📊 Monitoring

```bash
# Detailed status with metrics
vrooli resource home-assistant status

# JSON format for automation
vrooli resource home-assistant status --format json

# Quick health check
vrooli resource home-assistant test smoke
```

## 🔐 Security Considerations

1. **Network Security**:
   - Default uses host network for device discovery
   - Consider bridge network for isolation if discovery not needed
   
2. **Authentication**:
   - Always set up authentication in production
   - Use long-lived tokens for API access
   - Enable multi-factor authentication for admin users

3. **SSL/TLS**:
   - Configure HTTPS for production deployments
   - Use reverse proxy for SSL termination

4. **Trusted Networks**:
   - Configure trusted_networks for local access
   - Be cautious with network ranges

## 🛠️ Troubleshooting

### Common Issues

1. **Container won't start**:
   ```bash
   # Check logs
   vrooli resource home-assistant logs --tail 100
   
   # Verify port availability
   lsof -i :8123
   ```

2. **API not responding**:
   ```bash
   # Check health
   vrooli resource home-assistant test smoke
   
   # Restart container
   vrooli resource home-assistant manage restart
   ```

3. **Configuration errors**:
   ```bash
   # Check configuration syntax
   docker exec home-assistant python -m homeassistant --script check_config
   ```

4. **Discovery not working**:
   - Ensure container uses host network mode
   - Check firewall rules for mDNS (port 5353)
   - Verify devices are on same network

## 📚 Additional Resources

- [Official Documentation](https://www.home-assistant.io/docs/)
- [Integration Gallery](https://www.home-assistant.io/integrations/)
- [Automation Examples](https://www.home-assistant.io/examples/)
- [Community Forum](https://community.home-assistant.io/)
- [Discord Server](https://discord.gg/home-assistant)

## 📄 License

Home Assistant is released under the Apache 2.0 License.

## 🤝 Contributing

This resource follows the Vrooli v2.0 Universal Contract. Improvements should:
- Maintain backward compatibility
- Include comprehensive tests
- Update PRD with progress
- Follow security best practices