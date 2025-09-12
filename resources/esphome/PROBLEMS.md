# ESPHome Resource - Known Problems and Solutions

## YAML Configuration Changes (ESPHome 2025.8.4)

### Problem: Deprecated platform key
**Error**: `Please remove the platform key from the [esphome] block`

**Solution**: Use separate platform blocks instead of inline platform:
```yaml
# OLD (deprecated)
esphome:
  name: device
  platform: ESP32
  board: esp32dev

# NEW (correct)
esphome:
  name: device

esp32:
  board: esp32dev
```

### Problem: OTA requires platform specification
**Error**: `'ota' requires a 'platform' key but it was not specified`

**Solution**: Use list format with platform:
```yaml
# OLD
ota:
  password: "password"

# NEW
ota:
  - platform: esphome
    password: "password"
```

### Problem: Framework warning for ESP32
**Warning**: Device doesn't have a framework specified

**Solution**: Explicitly specify framework:
```yaml
esp32:
  board: esp32dev
  framework:
    type: arduino  # or esp-idf
```

## Container and Network Issues

### Problem: No native /health endpoint
ESPHome doesn't provide a dedicated health endpoint.

**Solution**: Check dashboard availability as health indicator:
- Use dashboard root URL for health checks
- Implement wrapper function to return JSON health status

### Problem: mDNS discovery may fail
Device discovery requires mDNS support on the network.

**Symptoms**: 
- Discovery returns no devices
- `zeroconf` module errors

**Solutions**:
1. Ensure mDNS/Bonjour is enabled on network
2. Check firewall rules for port 5353 (mDNS)
3. Use direct IP addresses as fallback

### Problem: USB device access
Serial monitoring requires privileged container access.

**Solution**: Enable USB access in container:
```bash
export ESPHOME_USB_ACCESS=true
# Container will run with --privileged and -v /dev:/dev
```

## Testing Issues

### Problem: Integration tests fail with real devices
Testing OTA updates and device compilation requires physical hardware.

**Solutions**:
1. Use mock/simulation for CI testing
2. Skip hardware-dependent tests when devices unavailable
3. Document manual testing procedures

## Performance Considerations

### Compilation Timeout
Firmware compilation can take several minutes, especially for first builds.

**Default timeout**: 300 seconds (5 minutes)
**Adjust if needed**: `export ESPHOME_COMPILE_TIMEOUT=600`

### PlatformIO Cache
First compilation downloads many dependencies.

**Solution**: Cache directory persists between container restarts at:
`~/.vrooli/esphome/.cache`

## Migration Notes

### From older ESPHome versions
1. Update all YAML configurations to new syntax
2. Remove deprecated keys (`platform` in esphome block)
3. Add framework specification for ESP32
4. Update OTA configuration to list format

### Dashboard Authentication
Optional but recommended for production:
```bash
export ESPHOME_DASHBOARD_PASSWORD="secure_password"
export ESPHOME_DASHBOARD_USERNAME="admin"
```

## Troubleshooting Commands

```bash
# Check container logs
vrooli resource esphome logs

# Validate configuration
vrooli resource esphome validate config.yaml

# Clean build artifacts if compilation fails
vrooli resource esphome clean

# Manual compilation test
docker exec esphome esphome compile /config/example.yaml

# Check PlatformIO installation
docker exec esphome which platformio
```