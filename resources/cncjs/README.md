# CNCjs Resource

Web-based interface for CNC milling controllers running Grbl, Marlin, Smoothieware, or TinyG.

## Overview

CNCjs provides a modern web interface for controlling CNC machines, featuring:
- Real-time G-code visualization with WebGL
- Macro automation system for repetitive tasks
- Remote access capability for distributed manufacturing
- Support for multiple CNC controller firmwares
- Serial port communication with USB/Bluetooth support

## Quick Start

```bash
# Install CNCjs
vrooli resource cncjs manage install

# Start the service
vrooli resource cncjs manage start --wait

# Check status
vrooli resource cncjs status

# Access web interface
# Open http://localhost:8194 in your browser
```

## Usage

### Managing G-code Files

```bash
# Upload G-code file
vrooli resource cncjs content add mypart.gcode

# List available files
vrooli resource cncjs content list

# Execute G-code job (requires controller connection)
vrooli resource cncjs content execute mypart.gcode

# Remove file
vrooli resource cncjs content remove mypart.gcode
```

### Configuration

Default configuration is stored in `~/.cncjs/.cncrc`. Key settings:

- **Port**: 8194 (web interface)
- **Controller**: Grbl (default)
- **Serial Port**: /dev/ttyUSB0
- **Baud Rate**: 115200
- **Remote Access**: Enabled

### Supported Controllers

- **Grbl**: Versions 0.9 and 1.1
- **Marlin**: 1.x and 2.x firmware
- **Smoothieware**: Latest version
- **TinyG**: Version 0.97
- **g2core**: Latest version

## Features

### P0 Requirements (Implemented)
- ✅ Health monitoring endpoint
- ✅ Lifecycle management (start/stop/restart)
- ✅ Serial port communication
- ✅ G-code file management
- ✅ Web-based control interface

### P1 Requirements (Planned)
- ⏳ Macro automation system
- ⏳ Multi-controller support
- ⏳ 3D visualization
- ⏳ Workflow storage

### P2 Requirements (Future)
- ⏳ Camera integration
- ⏳ Custom widgets
- ⏳ Job queue management

## Testing

```bash
# Run all tests
vrooli resource cncjs test all

# Quick health check
vrooli resource cncjs test smoke

# Full integration test
vrooli resource cncjs test integration

# Unit tests
vrooli resource cncjs test unit
```

## Troubleshooting

### Serial Port Access

If you encounter serial port permission issues:

1. Ensure Docker is running with `--privileged` flag (handled automatically)
2. Check serial device exists: `ls -la /dev/ttyUSB*`
3. Add user to dialout group: `sudo usermod -a -G dialout $USER`

### Controller Connection

1. Verify correct controller type in configuration
2. Check baud rate matches controller settings
3. Ensure no other software is using the serial port
4. Try different USB ports or cables

### Web Interface Issues

1. Check service is running: `vrooli resource cncjs status`
2. Verify port 8194 is not in use: `ss -tlnp | grep 8194`
3. Check firewall settings allow access
4. View logs: `vrooli resource cncjs logs`

## Integration Examples

### With Blender (3D to CNC)

```bash
# Generate 3D model in Blender
vrooli resource blender content execute generate_model.py

# Convert to G-code (using CAM software)
# Then upload to CNCjs
vrooli resource cncjs content add model.gcode
```

### With Node-RED (Automation)

Create automated manufacturing workflows:
- Monitor CNC job completion
- Queue multiple jobs
- Send notifications on completion
- Integrate with inventory management

## Security Notes

- Serial port access requires privileged container mode
- Remote access can be disabled in configuration
- Authentication tokens expire after 30 days by default
- Always validate G-code before execution

## Resources

- [Official Documentation](https://cnc.js.org/docs/)
- [G-code Reference](https://reprap.org/wiki/G-code)
- [Controller Firmware Guides](https://github.com/cncjs/cncjs/wiki)

## License

CNCjs is licensed under the MIT License. See the [CNCjs repository](https://github.com/cncjs/cncjs) for details.