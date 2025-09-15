# CNCjs Resource

Web-based interface for CNC milling controllers running Grbl, Marlin, Smoothieware, or TinyG.

## Overview

CNCjs provides a modern web interface for controlling CNC machines, featuring:
- **3D G-code visualization**: Interactive WebGL previews with path analysis
- **Macro automation**: Define and execute repetitive task sequences
- **Workflow management**: Chain multiple G-code files for production runs
- **Job queue management**: Automated scheduling with priority-based execution
- **Camera integration**: Real-time monitoring with snapshot and timelapse
- **Custom widgets**: Extensible UI with gauges, buttons, charts, and terminals
- **Multi-controller support**: Grbl, Marlin, Smoothieware, and TinyG
- **Remote access**: Control machines from anywhere via web interface
- **Serial communication**: USB, Bluetooth, and network connections

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

### Macro Automation

```bash
# Create a macro for common operations
vrooli resource cncjs macro add "home" "G28 ; Home all axes"
vrooli resource cncjs macro add "probe_z" "G38.2 Z-10 F50 ; Probe Z axis"
vrooli resource cncjs macro add "park" "G0 Z5 F500 ; Raise Z\nG0 X0 Y0 F2000 ; Go to origin"

# List available macros
vrooli resource cncjs macro list

# Execute a macro (requires controller connection)
vrooli resource cncjs macro run "home"

# Remove a macro
vrooli resource cncjs macro remove "home"
```

### Workflow Management

```bash
# Create a workflow for multi-step jobs
vrooli resource cncjs workflow create "production_run" "Complete production sequence"

# Add G-code steps to workflow
vrooli resource cncjs workflow add-step production_run roughing.gcode "Rough cut"
vrooli resource cncjs workflow add-step production_run finishing.gcode "Finish pass"
vrooli resource cncjs workflow add-step production_run drilling.gcode "Drill holes"

# List workflows
vrooli resource cncjs workflow list

# Execute workflow (queues all steps)
vrooli resource cncjs workflow execute production_run

# Export workflow for sharing
vrooli resource cncjs workflow export production_run production.tar.gz

# Import workflow from archive
vrooli resource cncjs workflow import production.tar.gz
```

### 3D Visualization

```bash
# Generate interactive 3D preview of G-code
vrooli resource cncjs visualization preview mypart.gcode

# Analyze G-code for statistics and bounds
vrooli resource cncjs visualization analyze mypart.gcode

# Render static visualization (SVG)
vrooli resource cncjs visualization render mypart.gcode output.svg

# Export standalone HTML visualization
vrooli resource cncjs visualization export mypart.gcode visualization.html

# Start visualization server (browse all visualizations)
vrooli resource cncjs visualization server
# Access at http://localhost:8195/
```

### Controller Configuration

```bash
# List supported controllers
vrooli resource cncjs controller list

# Create controller profiles
vrooli resource cncjs controller configure "3d_printer" "marlin" "/dev/ttyUSB1" "250000"
vrooli resource cncjs controller configure "laser_cutter" "grbl" "/dev/ttyUSB2" "115200"
vrooli resource cncjs controller configure "mill" "tinyg" "/dev/ttyUSB3" "115200"

# Show profile details
vrooli resource cncjs controller show 3d_printer

# Apply controller profile (updates CNCjs configuration)
vrooli resource cncjs controller apply laser_cutter

# Test connectivity
vrooli resource cncjs controller test

# Remove profile
vrooli resource cncjs controller remove 3d_printer
```

### Camera Monitoring

```bash
# List available camera devices
vrooli resource cncjs camera list

# Enable camera monitoring
vrooli resource cncjs camera enable /dev/video0

# Capture a snapshot
vrooli resource cncjs camera snapshot snapshot.jpg

# Start live video stream
vrooli resource cncjs camera stream start
# View stream at http://localhost:8195/feed.mjpg

# Stop video stream
vrooli resource cncjs camera stream stop

# Start timelapse capture (10 second intervals)
vrooli resource cncjs camera timelapse start 10

# Stop timelapse and compile video
vrooli resource cncjs camera timelapse stop
vrooli resource cncjs camera timelapse compile
```

### Custom Widgets

```bash
# List available widgets
vrooli resource cncjs widget list

# Create custom widgets
vrooli resource cncjs widget create spindle-speed gauge
vrooli resource cncjs widget create emergency-stop button
vrooli resource cncjs widget create temperature-graph chart
vrooli resource cncjs widget create gcode-terminal terminal
vrooli resource cncjs widget create machine-status status

# Show widget definition
vrooli resource cncjs widget show spindle-speed

# Install widget to CNCjs interface
vrooli resource cncjs widget install spindle-speed

# Export widget for sharing
vrooli resource cncjs widget export spindle-speed spindle-widget.tar.gz

# Import widget from package
vrooli resource cncjs widget import spindle-widget.tar.gz

# Uninstall widget
vrooli resource cncjs widget uninstall spindle-speed

# Remove widget definition
vrooli resource cncjs widget remove spindle-speed
```

### Job Queue Management

```bash
# List job queue status
vrooli resource cncjs jobqueue list

# Add G-code file to queue with priority (1-10, 1=highest)
vrooli resource cncjs jobqueue add file.gcode 3 job_name

# Check specific job status
vrooli resource cncjs jobqueue status job_name

# Start job queue processor
vrooli resource cncjs jobqueue start

# Stop job queue processor
vrooli resource cncjs jobqueue stop

# Clear queues (pending/completed/failed/all)
vrooli resource cncjs jobqueue clear all

# Remove specific job from queue
vrooli resource cncjs jobqueue remove job_name
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

### P1 Requirements (Implemented)
- ✅ Macro automation system - Create and execute reusable G-code macros
- ✅ Multi-controller support - Configure profiles for different CNC controllers
- ✅ Workflow storage - Save and manage multi-step CNC job sequences
- ✅ 3D visualization - Interactive WebGL-based G-code preview and analysis

### P2 Requirements (Implemented)
- ✅ Camera integration - Real-time monitoring via webcam with timelapse capture
- ✅ Custom widgets - Extensible UI with gauges, buttons, charts, and terminals
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