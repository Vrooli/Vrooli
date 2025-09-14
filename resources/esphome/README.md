# ESPHome Resource

ESP32/ESP8266 firmware framework using YAML configuration for creating custom IoT devices.

## Overview

ESPHome enables Vrooli to create custom firmware for ESP32 and ESP8266 microcontrollers without programming. Using simple YAML configuration files, you can define sensors, actuators, and automation logic that runs directly on the microcontroller.

## Quick Start

```bash
# Install ESPHome
vrooli resource esphome manage install

# Start the service
vrooli resource esphome manage start --wait

# Check status
vrooli resource esphome status

# Access dashboard
# Open http://localhost:6587 in your browser
```

## Usage

### Using Templates

```bash
# List available device templates
vrooli resource esphome template::list

# Apply a template to create new device
vrooli resource esphome template::apply temperature-sensor living-room "Living Room Sensor"

# Available templates:
# - temperature-sensor: DHT22 temperature/humidity sensor
# - motion-sensor: PIR motion detection with LED indicator
# - smart-switch: WiFi-controlled relay switch
```

### Monitoring & Metrics

```bash
# Display device telemetry and statistics
vrooli resource esphome metrics

# Set up alert system for device failures
vrooli resource esphome alerts::setup

# Check for active device alerts
vrooli resource esphome alerts::check
```

### Custom Components

```bash
# Add a new custom component template
vrooli resource esphome custom::add my_sensor

# List available custom components
vrooli resource esphome custom::list
```

### Managing Configurations

```bash
# Add a new device configuration
vrooli resource esphome content add my-sensor.yaml

# List all configurations
vrooli resource esphome content list

# View a configuration
vrooli resource esphome content get my-sensor

# Compile firmware
vrooli resource esphome content execute my-sensor.yaml

# Remove configuration
vrooli resource esphome content remove my-sensor
```

### Device Management

```bash
# Discover ESP devices on network
vrooli resource esphome discover

# Upload firmware via OTA
vrooli resource esphome upload my-sensor.yaml 192.168.1.100

# Monitor device logs
vrooli resource esphome monitor /dev/ttyUSB0

# Validate configuration
vrooli resource esphome validate my-sensor.yaml
```

## Configuration

Default settings are stored in `config/defaults.sh`. Key configurations:

- **Port**: 6587 (dashboard)
- **Container**: esphome
- **OTA Password**: vrooli_ota (default)
- **Parallel Builds**: 2
- **Compile Timeout**: 300s

## Example Configuration

Create a temperature sensor:

```yaml
# temperature-sensor.yaml
esphome:
  name: temp-sensor
  platform: ESP32
  board: esp32dev

wifi:
  ssid: "YourWiFi"
  password: "YourPassword"

sensor:
  - platform: dht
    pin: GPIO23
    temperature:
      name: "Room Temperature"
      unit_of_measurement: "°C"
    humidity:
      name: "Room Humidity"
      unit_of_measurement: "%"
    update_interval: 60s

# Enable logging
logger:

# Enable OTA updates
ota:
  password: "vrooli_ota"

# Enable Home Assistant API
api:

# Web server for device status
web_server:
  port: 80
```

## Features

### Core Capabilities
- **YAML-based Configuration**: Define devices using simple YAML files
- **Firmware Compilation**: Build custom firmware for ESP32/ESP8266 devices
- **OTA Updates**: Deploy firmware updates over the network
- **Template System**: Quick-start templates for common IoT scenarios
- **Web Dashboard**: Visual interface for device management at port 6587
- **Device Discovery**: Automatic detection of ESP devices on network

### Advanced Features
- **Home Assistant Integration**: Auto-discovery and entity creation
- **Backup/Restore**: Save and restore device configurations
- **Bulk Operations**: Manage multiple devices simultaneously
- **Metrics Dashboard**: Real-time device telemetry and statistics
- **Alert System**: Automated monitoring and failure notifications
- **Custom Components**: Create your own ESP components

## Supported Components

### Sensors
- Temperature/Humidity (DHT, BME280, BMP280)
- Motion (PIR, radar)
- Light (BH1750, TSL2561)
- Distance (ultrasonic, VL53L0x)
- Air quality (MQ sensors, CCS811)
- Energy monitoring (PZEM, INA219)

### Actuators
- Relays and switches
- LEDs (single, RGB, addressable)
- Servos and motors
- Displays (OLED, LCD, e-paper)
- Buzzers and speakers

### Communication
- WiFi and Ethernet
- MQTT for messaging
- HTTP/REST API
- Home Assistant integration
- I2C, SPI, UART protocols

## Testing

```bash
# Run all tests
vrooli resource esphome test all

# Quick health check
vrooli resource esphome test smoke

# Integration tests
vrooli resource esphome test integration

# Unit tests
vrooli resource esphome test unit
```

## Integration Examples

### With Home Assistant

ESPHome devices automatically integrate with Home Assistant:

1. Configure device with `api:` component
2. Home Assistant auto-discovers the device
3. Entities appear in Home Assistant UI
4. Control devices through automations

### With Node-RED

Create IoT workflows:

```yaml
# Enable MQTT in ESPHome config
mqtt:
  broker: 192.168.1.100
  discovery: true
  
# Node-RED can now:
# - Subscribe to sensor data
# - Send commands to devices
# - Create complex automations
```

### With Vault

Secure credential storage:

```bash
# Store WiFi credentials in Vault
vrooli resource vault content add wifi-password "MySecurePassword"

# Reference in ESPHome config
# wifi:
#   ssid: "MyNetwork"
#   password: !secret wifi_password
```

## Troubleshooting

### Dashboard Not Accessible

1. Check container is running: `vrooli resource esphome status`
2. Verify port 6587 is available: `ss -tlnp | grep 6587`
3. Check logs: `vrooli resource esphome logs`

### Compilation Fails

1. Validate YAML syntax: `vrooli resource esphome validate config.yaml`
2. Check board type is correct
3. Ensure all referenced pins exist on your board
4. Review error messages in logs

### OTA Upload Fails

1. Verify device is on same network
2. Check OTA password matches
3. Ensure device has sufficient free space
4. Try USB upload for initial firmware

### Device Not Discovered

1. Enable mDNS on your network
2. Check firewall rules
3. Verify device successfully connected to WiFi
4. Use manual IP address if discovery fails

## Security Notes

- Always set strong OTA passwords
- Use API passwords for Home Assistant
- Enable web authentication for dashboard
- Validate all YAML configurations
- Keep firmware updated

## Performance Tips

- Limit update intervals to reduce network traffic
- Use deep sleep for battery-powered devices
- Optimize WiFi power settings
- Batch sensor readings
- Use local automation when possible

## Resources

- [ESPHome Documentation](https://esphome.io)
- [Supported Devices](https://esphome.io/devices/)
- [Component Reference](https://esphome.io/components/)
- [Home Assistant Integration](https://www.home-assistant.io/integrations/esphome/)

## License

ESPHome is licensed under the MIT License. See the [ESPHome repository](https://github.com/esphome/esphome) for details.