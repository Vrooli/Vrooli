# Home Assistant Resource

## Overview

Home Assistant is an open-source home automation platform that enables local control and automation of smart home devices. This resource provides Home Assistant as a containerized service within the Vrooli ecosystem, enabling scenarios to leverage home automation, IoT device control, and intelligent home management capabilities.

## Key Features

- **3000+ Integrations**: Connect with smart devices from hundreds of manufacturers
- **Local Control**: Everything runs locally with optional cloud connections
- **Visual Automation**: Create complex automations through UI or YAML
- **Energy Management**: Monitor and optimize home energy usage
- **Voice Control**: Integrate with voice assistants
- **Dashboards**: Create custom interfaces for different users and devices

## Benefits for Vrooli

Home Assistant enables Vrooli to:

- **IoT Scenarios**: Build scenarios that interact with physical devices
- **Energy Optimization**: Create intelligent energy management workflows
- **Security Automation**: Implement presence-based security scenarios
- **Data Collection**: Gather sensor data for analysis and ML workflows
- **Notification Systems**: Send alerts through multiple channels
- **Cross-Platform Integration**: Bridge different smart home ecosystems

## Scenario Examples

1. **Smart Energy Manager**: Optimize home energy usage based on solar production, battery storage, and grid prices
2. **Security Assistant**: Automated security responses based on presence, cameras, and sensors
3. **Health Monitor**: Track environmental conditions and suggest improvements
4. **Predictive Maintenance**: Alert when appliances need service based on usage patterns
5. **Voice-Controlled Workflows**: Trigger Vrooli scenarios through voice commands

## Quick Start

### Installation

```bash
# Install Home Assistant
vrooli resource home-assistant install

# Check status
vrooli resource home-assistant status

# Access web UI
# Open http://localhost:8123 in your browser
```

### Basic Usage

```bash
# View logs
vrooli resource home-assistant logs

# Inject an automation
vrooli resource home-assistant inject my-automation.yaml

# List injected files
vrooli resource home-assistant list

# Restart to apply changes
vrooli resource home-assistant restart
```

## Data Injection

Home Assistant accepts several file types for configuration and automation:

### YAML Automations (.yaml, .yml)

Create automations that respond to triggers and conditions:

```yaml
# smart-lights.yaml
alias: Motion Activated Lights
description: Turn on lights when motion detected
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
      entity_id: light.living_room
```

### Python Scripts (.py)

Create custom logic for complex scenarios:

```python
# energy_optimizer.py
# Optimize energy usage based on solar production
solar_power = hass.states.get('sensor.solar_power').state
battery_level = hass.states.get('sensor.battery_level').state

if float(solar_power) > 3000 and float(battery_level) > 80:
    hass.services.call('switch', 'turn_on', {'entity_id': 'switch.pool_pump'})
```

### JSON Configurations (.json)

Import device configurations and service definitions:

```json
{
  "automation": {
    "alias": "Temperature Alert",
    "trigger": {
      "platform": "numeric_state",
      "entity_id": "sensor.temperature",
      "above": 30
    },
    "action": {
      "service": "notify.email",
      "data": {
        "message": "High temperature detected!"
      }
    }
  }
}
```

## Integration with Vrooli

### Using in Scenarios

Home Assistant can be integrated into Vrooli scenarios through:

1. **REST API**: Direct API calls for device control and data retrieval
2. **Webhooks**: Trigger Vrooli workflows from Home Assistant events
3. **MQTT**: Real-time bidirectional communication
4. **File-based**: Import/export configurations and data

### Example Workflow Integration

```javascript
// In a Vrooli workflow
const homeAssistant = {
  url: 'http://localhost:8123',
  token: process.env.HOME_ASSISTANT_TOKEN
};

// Get current temperature
const temperature = await fetch(`${homeAssistant.url}/api/states/sensor.temperature`, {
  headers: { 'Authorization': `Bearer ${homeAssistant.token}` }
}).then(r => r.json());

// Turn on AC if too hot
if (parseFloat(temperature.state) > 25) {
  await fetch(`${homeAssistant.url}/api/services/climate/set_temperature`, {
    method: 'POST',
    headers: { 
      'Authorization': `Bearer ${homeAssistant.token}`,
      'Content-Type': 'application/json'
    },
    body: JSON.stringify({
      entity_id: 'climate.ac',
      temperature: 22
    })
  });
}
```

## Architecture

```
┌─────────────────────────────────────┐
│          Vrooli Platform            │
├─────────────────────────────────────┤
│         Home Assistant              │
│  ┌─────────────────────────────┐   │
│  │     Core Engine             │   │
│  ├─────────────────────────────┤   │
│  │  Integrations │ Automations │   │
│  ├──────────────┼──────────────┤   │
│  │   Devices    │   Scripts    │   │
│  └──────────────┴──────────────┘   │
├─────────────────────────────────────┤
│    Physical Devices & Services      │
│  (Lights, Sensors, Thermostats)     │
└─────────────────────────────────────┘
```

## Configuration

### Environment Variables

- `HOME_ASSISTANT_PORT`: Web UI port (default: 8123)
- `HOME_ASSISTANT_TIME_ZONE`: System timezone (default: America/New_York)
- `HOME_ASSISTANT_CONFIG_DIR`: Configuration directory

### Data Persistence

All configuration and state data is stored in:
- Config: `~/Vrooli/data/home-assistant/config/`
- Automations: `~/Vrooli/data/home-assistant/config/automations/`
- Scripts: `~/Vrooli/data/home-assistant/config/python_scripts/`

## Troubleshooting

### Common Issues

1. **Port Conflict**: If port 8123 is in use, change `HOME_ASSISTANT_PORT`
2. **Discovery Issues**: Use `--network host` for device discovery
3. **Slow Startup**: First run downloads components, wait 2-3 minutes
4. **Permission Errors**: Container runs privileged for hardware access

### Debug Commands

```bash
# Check container status
docker ps -a | grep home-assistant

# View detailed logs
docker logs home-assistant -f

# Check configuration
docker exec home-assistant python -m homeassistant --script check_config

# Access container shell
docker exec -it home-assistant bash
```

## Security Considerations

1. **Authentication**: Set up users and long-lived tokens
2. **Network**: Use HTTPS for external access
3. **Secrets**: Store credentials in secrets.yaml
4. **Updates**: Regularly update for security patches
5. **Backups**: Regular configuration backups recommended

## Additional Resources

- [Home Assistant Documentation](https://www.home-assistant.io/docs/)
- [Integration Gallery](https://www.home-assistant.io/integrations/)
- [Automation Examples](https://www.home-assistant.io/examples/)
- [Community Forums](https://community.home-assistant.io/)
- [YAML Reference](https://www.home-assistant.io/docs/configuration/yaml/)

## Best Practices

1. **Start Simple**: Begin with basic automations, add complexity gradually
2. **Use Packages**: Organize complex configurations into packages
3. **Version Control**: Track automation changes in git
4. **Test Locally**: Test automations in development before production
5. **Monitor Performance**: Use built-in diagnostics for optimization
6. **Document Automations**: Add descriptions to all automations