# Zigbee2MQTT Resource

Open-source Zigbee to MQTT bridge that allows you to use your Zigbee devices without the vendor's bridge or gateway. It bridges events and allows you to control your Zigbee devices via MQTT.

## Overview

Zigbee2MQTT supports over 3000 devices from various manufacturers including Philips Hue, IKEA TRÅDFRI, Xiaomi Aqara, and many more. It provides local control, eliminates cloud dependencies, and preserves privacy in smart home automation.

## Features

- ✅ **3000+ Supported Devices** - Extensive device compatibility
- ✅ **Local Control** - No cloud dependencies or internet required
- ✅ **MQTT Bridge** - Universal protocol for IoT communication
- ✅ **Home Assistant Integration** - Auto-discovery support
- ✅ **Web Dashboard** - User-friendly device management interface
- ✅ **Network Visualization** - See your Zigbee mesh topology
- ✅ **OTA Updates** - Update device firmware over the air
- ✅ **Groups & Scenes** - Manage multiple devices together

## Prerequisites

### Hardware Requirements
- **Zigbee USB Adapter** (one of the following):
  - CC2652 based (recommended): SONOFF ZBDongle-P, Electrolama zzh!
  - CC2531 based: Cheap but limited to ~20 devices
  - ConBee II: Good alternative with strong mesh
  - CC2530/CC2538: Other supported options

### Software Requirements
- Docker installed and running
- MQTT broker (mosquitto or similar)
- USB device passthrough permissions

## Quick Start

### 1. Install Zigbee2MQTT
```bash
vrooli resource zigbee2mqtt manage install
```

### 2. Configure USB Adapter
```bash
# Find your adapter (usually /dev/ttyACM0 or /dev/ttyUSB0)
ls -la /dev/tty*

# Set adapter path
export ZIGBEE_ADAPTER=/dev/ttyACM0
```

### 3. Start MQTT Broker (if not running)
```bash
# Using mosquitto
docker run -d --name mosquitto -p 1883:1883 eclipse-mosquitto

# Or use Vrooli's mosquitto resource if available
vrooli resource mosquitto manage start
```

### 4. Start Zigbee2MQTT
```bash
vrooli resource zigbee2mqtt manage start --wait
```

### 5. Access Web UI
Open browser to: http://localhost:8090

## Usage

### Device Management

#### Pair New Device
```bash
# Enable pairing mode (120 seconds)
vrooli resource zigbee2mqtt device pair

# Then press pairing button on your Zigbee device
```

#### List Devices
```bash
vrooli resource zigbee2mqtt content list
```

#### Remove Device
```bash
vrooli resource zigbee2mqtt device unpair "device_name"
```

#### Rename Device
```bash
vrooli resource zigbee2mqtt device rename "old_name" "new_name"
```

### Device Control

#### Basic On/Off Control
```bash
# Turn device on
vrooli resource zigbee2mqtt device control "living_room_light" on

# Turn device off
vrooli resource zigbee2mqtt device control "living_room_light" off

# Toggle device state
vrooli resource zigbee2mqtt device control "living_room_light" toggle
```

#### Brightness Control
```bash
# Set brightness (0-255)
vrooli resource zigbee2mqtt device brightness "bedroom_lamp" 128

# Dim to 20%
vrooli resource zigbee2mqtt device brightness "bedroom_lamp" 51
```

#### Color Control
```bash
# Set color using hex
vrooli resource zigbee2mqtt device color "rgb_bulb" "#FF0000"  # Red

# Set color using RGB values
vrooli resource zigbee2mqtt device color "rgb_bulb" "0,255,0"  # Green
```

#### Color Temperature
```bash
# Set color temperature in Kelvin
vrooli resource zigbee2mqtt device temperature "white_bulb" 2700  # Warm white
vrooli resource zigbee2mqtt device temperature "white_bulb" 6500  # Cool white
```

### Network Management

#### View Network Map
```bash
vrooli resource zigbee2mqtt network map
```

#### Backup Coordinator
```bash
vrooli resource zigbee2mqtt network backup
```

#### Change Zigbee Channel
```bash
# Use channel 11, 15, 20, or 25 for best WiFi coexistence
vrooli resource zigbee2mqtt network channel 15
```

### MQTT Topics

Control devices via MQTT:
```bash
# Turn on a light
mosquitto_pub -t "zigbee2mqtt/living_room_light/set" -m '{"state":"ON"}'

# Set brightness
mosquitto_pub -t "zigbee2mqtt/living_room_light/set" -m '{"brightness":128}'

# Set color
mosquitto_pub -t "zigbee2mqtt/living_room_light/set" -m '{"color":{"hex":"#FF0000"}}'
```

Subscribe to device states:
```bash
# All device updates
mosquitto_sub -t "zigbee2mqtt/#"

# Specific device
mosquitto_sub -t "zigbee2mqtt/living_room_light"
```

## Configuration

Configuration file: `${VROOLI_ROOT}/data/zigbee2mqtt/configuration.yaml`

### Basic Configuration
```yaml
mqtt:
  server: mqtt://localhost:1883
  base_topic: zigbee2mqtt

serial:
  port: /dev/ttyACM0
  
frontend:
  port: 8080
  
homeassistant: true
```

### Environment Variables
```bash
ZIGBEE2MQTT_PORT=8090        # Web UI port
MQTT_HOST=localhost          # MQTT broker host
MQTT_PORT=1883              # MQTT broker port
ZIGBEE_ADAPTER=/dev/ttyACM0 # USB adapter path
```

## Integration

### Home Assistant
Zigbee2MQTT automatically publishes Home Assistant discovery information. Devices appear automatically in Home Assistant when both are connected to the same MQTT broker.

### Node-RED
Install the Zigbee2MQTT nodes:
```bash
vrooli resource node-red content add node-red-contrib-zigbee2mqtt
```

### Grafana Monitoring
Metrics are exposed at `/api/metrics` in Prometheus format for monitoring device availability, message rates, and network health.

## Testing

```bash
# Quick health check
vrooli resource zigbee2mqtt test smoke

# Full test suite
vrooli resource zigbee2mqtt test all
```

## Troubleshooting

### Adapter Not Found
```bash
# Check USB devices
lsusb

# Check serial ports
ls -la /dev/tty*

# Set correct path
export ZIGBEE_ADAPTER=/dev/ttyUSB0
```

### Permission Denied
```bash
# Add user to dialout group
sudo usermod -a -G dialout $USER

# Logout and login again
```

### Devices Not Pairing
1. Check coordinator firmware is up to date
2. Move device closer to coordinator
3. Reset device to factory settings
4. Check Zigbee channel for interference

### MQTT Connection Failed
```bash
# Test MQTT broker
mosquitto_pub -h localhost -t test -m "test"

# Check broker logs
docker logs mosquitto
```

## Advanced Topics

### Custom Device Converters
Add support for unsupported devices by creating external converters in `data/zigbee2mqtt/converters/`.

### Touchlink Commissioning
For devices that support Touchlink:
```bash
vrooli resource zigbee2mqtt content execute touchlink/factory_reset
```

### Network Optimization
- Use Zigbee channel 11, 15, 20, or 25 to avoid WiFi interference
- Add router devices (powered devices) to strengthen mesh
- Keep coordinator central in your home
- Avoid metal enclosures and interference sources

## Security

- Network key is auto-generated on first start
- Enable frontend authentication for production
- Use MQTT with TLS in production environments
- Regularly backup coordinator configuration

## Resources

- [Supported Devices Database](https://www.zigbee2mqtt.io/supported-devices/)
- [Configuration Documentation](https://www.zigbee2mqtt.io/guide/configuration/)
- [MQTT Topics Reference](https://www.zigbee2mqtt.io/guide/usage/mqtt_topics_and_messages.html)
- [Coordinator Firmware](https://github.com/Koenkk/Z-Stack-firmware)

## License

This resource integrates Zigbee2MQTT which is licensed under GPL-3.0.

---

For more information about Vrooli resources, see the [main documentation](/docs/README.md).