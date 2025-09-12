# ESPHome Resource - Product Requirements Document

## Executive Summary
- **What**: ESPHome firmware framework for ESP32/ESP8266 microcontrollers using YAML configuration
- **Why**: Enable Vrooli to create custom IoT devices and edge computing nodes without complex programming
- **Who**: Scenarios needing physical sensors, actuators, or edge computing capabilities
- **Value**: $15,000+ through custom IoT hardware development, edge computing services, and sensor network deployment
- **Priority**: Medium - Expands Vrooli's capabilities from software to hardware control

## Core Features

### Current State (2025-09-12)
Resource has been scaffolded with v2.0 contract compliance and basic functionality working.

### Target Capabilities
- YAML-based firmware generation for ESP32/ESP8266
- Over-the-air (OTA) updates for deployed devices
- Home Assistant integration for smart home scenarios
- Extensive sensor/actuator library support
- Web dashboard for device management
- Captive portal for WiFi configuration

## Revenue Opportunities

### Direct Revenue
- **Custom IoT Devices**: $500-2,000 per device type developed
- **Edge Computing Nodes**: $100-500 per node deployed
- **Sensor Networks**: $5,000-20,000 for industrial monitoring
- **Smart Agriculture**: $10,000+ for irrigation/monitoring systems

### Indirect Value
- Enables physical-world interaction for scenarios
- Creates data collection points for AI/ML workflows
- Reduces cloud dependency through edge processing
- Provides hardware control for automation scenarios

## Requirements Checklist

### P0 Requirements (Must Have)
- [x] **Health Monitoring**: Service responds to health checks (dashboard availability)
- [x] **Lifecycle Management**: Full support for install/start/stop/restart/uninstall commands
- [x] **YAML Processing**: Compile YAML configurations to firmware binaries
- [x] **Device Discovery**: Scan network for ESP devices using mDNS
- [ ] **OTA Updates**: Deploy firmware updates over the network (partial - command exists, needs testing)
- [x] **Configuration Validation**: Validate YAML syntax and component compatibility
- [x] **Web Dashboard**: Provide web UI at port 6587 for device management

### P1 Requirements (Should Have)
- [ ] **Home Assistant Integration**: Auto-discovery and entity creation
- [ ] **Sensor Templates**: Pre-configured templates for common sensors
- [ ] **Backup/Restore**: Device configuration backup and restoration
- [ ] **Bulk Operations**: Update multiple devices simultaneously

### P2 Requirements (Nice to Have)
- [ ] **Custom Components**: Support for user-defined components
- [ ] **Metrics Dashboard**: Device telemetry and statistics
- [ ] **Alert System**: Notifications for device failures

## Technical Specifications

### Architecture
```
┌─────────────────────────────────────────┐
│           ESPHome Container             │
│                                         │
│  ┌─────────────┐    ┌────────────────┐ │
│  │   Compiler  │───▶│  Firmware Gen  │ │
│  └─────────────┘    └────────────────┘ │
│         │                    │          │
│         ▼                    ▼          │
│  ┌─────────────┐    ┌────────────────┐ │
│  │ Web Dashboard│    │   OTA Server   │ │
│  └─────────────┘    └────────────────┘ │
│         │                    │          │
└─────────│────────────────────│──────────┘
          │                    │
          ▼                    ▼
    ┌──────────┐        ┌──────────┐
    │ ESP32 #1 │        │ ESP32 #2 │
    └──────────┘        └──────────┘
```

### Dependencies
- **Runtime**: Python 3.9+, platformio
- **Build Tools**: gcc-arm-none-eabi, esptool
- **Network**: mDNS for device discovery
- **Storage**: Local filesystem for configurations

### API Endpoints
- `GET /health` - Health check endpoint
- `GET /devices` - List discovered devices
- `POST /compile` - Compile YAML to firmware
- `POST /upload/{device_id}` - OTA firmware upload
- `GET /logs/{device_id}` - Stream device logs
- `GET /dashboard` - Web UI dashboard

### Integration Points
- **Home Assistant**: MQTT discovery protocol
- **Node-RED**: MQTT/HTTP webhooks for events
- **Vault**: Secure storage of WiFi credentials
- **Prometheus**: Export device metrics

## Success Metrics

### Completion Criteria
- [x] 0% - PRD defined
- [x] 20% - Basic structure scaffolded
- [x] 40% - Health checks and lifecycle working
- [x] 60% - YAML compilation functional
- [ ] 80% - OTA updates working (partial implementation)
- [ ] 100% - All P0 requirements met (5/7 P0 requirements complete)

### Quality Metrics
- Health check response < 500ms
- Firmware compilation < 60s
- OTA update success rate > 95%
- Device discovery < 10s

### Performance Targets
- Support 100+ devices per instance
- Handle 10 concurrent compilations
- Stream logs from 20 devices simultaneously
- Web dashboard loads in < 2s

## Security Considerations

### Required Security
- WiFi credentials encrypted in Vault
- OTA updates use password protection
- Web dashboard requires authentication
- Device-to-server communication encrypted

### Best Practices
- Rotate OTA passwords regularly
- Validate all YAML inputs
- Sandbox compilation process
- Rate limit API endpoints

## Implementation Approach

### Phase 1: Foundation (This Task)
1. Create v2.0 contract structure
2. Implement health checks
3. Setup basic web server
4. Create lifecycle commands

### Phase 2: Core Functionality (Future)
1. YAML parser and validator
2. PlatformIO integration
3. Firmware compilation pipeline
4. Basic web dashboard

### Phase 3: Advanced Features (Future)
1. OTA update system
2. Device discovery via mDNS
3. Home Assistant integration
4. Metrics and monitoring

## Testing Strategy

### Smoke Tests
- Service starts successfully
- Health endpoint responds
- Web dashboard accessible
- Lifecycle commands work

### Integration Tests
- YAML validation works
- Mock compilation succeeds
- Device discovery simulation
- API endpoints respond correctly

### End-to-End Tests
- Full compilation pipeline
- OTA update simulation
- Multi-device management
- Home Assistant discovery

## Documentation Requirements

### User Documentation
- Quick start guide
- YAML configuration reference
- Sensor/actuator catalog
- Troubleshooting guide

### Developer Documentation
- API reference
- Custom component guide
- Integration examples
- Security best practices

## Known Limitations

### Current Limitations
- Requires physical ESP32/ESP8266 for full testing
- USB access needed for initial flashing
- Network requirements for OTA updates
- Limited to ESP-compatible components

### Future Improvements
- Support for ESP32-S3/C3 variants
- Bluetooth provisioning
- Cloud compilation service
- Mobile app for management

## Competitive Analysis

### vs Tasmota
- **Advantage**: YAML configuration vs web UI
- **Advantage**: Better Home Assistant integration
- **Disadvantage**: Requires compilation

### vs Arduino IDE
- **Advantage**: No programming required
- **Advantage**: Built-in OTA updates
- **Disadvantage**: Less flexibility

### vs PlatformIO
- **Advantage**: Higher-level abstraction
- **Advantage**: Automatic dependency management
- **Disadvantage**: Limited to ESP platforms

## Resource Planning

### Estimated Effort
- Generator: 4 hours (PRD + scaffolding)
- Improver Round 1: 4 hours (core features)
- Improver Round 2: 4 hours (integrations)
- Improver Round 3: 4 hours (polish)

### Required Skills
- Docker containerization
- Python development
- Web dashboard creation
- IoT protocols (MQTT, mDNS)

## Appendix

### Example YAML Configuration
```yaml
esphome:
  name: temperature-sensor
  platform: ESP32
  board: esp32dev

wifi:
  ssid: "MyNetwork"
  password: !secret wifi_password

sensor:
  - platform: dht
    pin: GPIO23
    temperature:
      name: "Room Temperature"
    humidity:
      name: "Room Humidity"
    update_interval: 60s

mqtt:
  broker: 192.168.1.100
  discovery: true
```

### Example Use Cases
1. **Environmental Monitoring**: Temperature, humidity, air quality sensors
2. **Security System**: Motion sensors, door contacts, cameras
3. **Garden Automation**: Soil moisture, irrigation control
4. **Energy Management**: Power monitoring, smart switches
5. **Industrial IoT**: Machine monitoring, predictive maintenance

## Progress History

### 2025-09-12: Initial Implementation (60% Complete)
**Improvements Made:**
- ✅ Implemented v2.0 contract compliance with all required commands
- ✅ Added health monitoring (via dashboard availability check)
- ✅ Full lifecycle management (install/start/stop/restart/uninstall)
- ✅ Configuration management (add/list/get/remove/validate)
- ✅ Web dashboard accessible at port 6587
- ✅ Device discovery command implemented
- ✅ Fixed YAML syntax for ESPHome 2025.8.4 compatibility
- ✅ All tests passing (smoke, integration, unit)
- ✅ Added `info` command for structured resource information

**Issues Resolved:**
- Fixed deprecated `platform` key in YAML configs
- Updated OTA configuration to use list format with platform specification
- Added framework specification to avoid deprecation warnings
- Fixed test suite variable references

**Known Limitations:**
- OTA updates implemented but require physical devices for testing
- Device discovery requires mDNS network support
- USB access requires privileged container mode

---

**Last Updated**: 2025-09-12
**Status**: Functional - 60% Complete (5/7 P0 requirements met)
**Next Steps**: Test OTA updates with physical devices, enhance device discovery