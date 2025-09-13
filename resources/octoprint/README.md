# OctoPrint Resource

Web-based 3D printer management platform providing remote control, monitoring, and automation capabilities for additive manufacturing workflows.

## Overview

OctoPrint enables Vrooli scenarios to integrate 3D printing into their workflows, supporting everything from rapid prototyping to distributed manufacturing. It provides a REST API and web interface for complete printer control.

## Features

### Core Capabilities
- **Remote Control**: Start, pause, cancel prints from anywhere
- **G-Code Management**: Upload, organize, and execute print files
- **Temperature Monitoring**: Real-time hotend and bed temperature tracking
- **Print Progress**: Track print status, time remaining, and completion
- **Serial Communication**: Direct control via USB/serial connection

### Advanced Features
- **Webcam Support**: Live video monitoring of prints
- **Plugin System**: Extend functionality with community plugins
- **Print Queue**: Manage multiple jobs with scheduling
- **Slicing Integration**: Convert STL to G-code on-the-fly
- **API Access**: Full REST API for automation

## Quick Start

```bash
# Install and start OctoPrint
vrooli resource octoprint manage install
vrooli resource octoprint manage start --wait

# Check status
vrooli resource octoprint status

# Access web interface
# Open http://localhost:8197 in browser

# Run tests
vrooli resource octoprint test all
```

## Configuration

### Environment Variables
```bash
OCTOPRINT_PORT=8197              # Web interface port
OCTOPRINT_PRINTER_PORT=/dev/ttyUSB0  # Printer serial port
OCTOPRINT_CAMERA_URL=http://localhost:8080/?action=stream  # Webcam URL
OCTOPRINT_API_KEY=auto           # API key (auto-generated if 'auto')
```

### Printer Connection
OctoPrint automatically detects printers on common serial ports:
- `/dev/ttyUSB*` - USB serial adapters
- `/dev/ttyACM*` - Arduino-based printers
- Network printers via RFC2217

## API Usage

### Basic Operations
```bash
# Get printer status
curl -H "X-Api-Key: $API_KEY" http://localhost:8197/api/printer

# Upload G-code file
curl -H "X-Api-Key: $API_KEY" -F "file=@model.gcode" \
  http://localhost:8197/api/files/local

# Start print
curl -H "X-Api-Key: $API_KEY" -X POST \
  http://localhost:8197/api/job -d '{"command":"start"}'

# Monitor temperature
curl -H "X-Api-Key: $API_KEY" \
  http://localhost:8197/api/printer/tool
```

### WebSocket Events
Connect to WebSocket for real-time updates:
```javascript
const ws = new WebSocket('ws://localhost:8197/sockjs/websocket');
ws.onmessage = (event) => {
  const data = JSON.parse(event.data);
  // Handle temperature updates, print progress, etc.
};
```

## Integration Examples

### With N8n Workflow
```yaml
# Automated print workflow
1. Monitor folder for new STL files
2. Slice to G-code using configured slicer
3. Upload to OctoPrint
4. Start print when printer idle
5. Send completion notification
```

### With Postgres Analytics
```sql
-- Track print metrics
INSERT INTO print_jobs (filename, duration, material_used, success)
VALUES ('model.gcode', 7200, 45.3, true);
```

## Troubleshooting

### Common Issues

**Printer Not Detected**
```bash
# Check serial ports
ls -la /dev/tty*

# Test connection manually
vrooli resource octoprint cli test-serial /dev/ttyUSB0
```

**Permission Denied on Serial Port**
```bash
# Add user to dialout group
sudo usermod -a -G dialout $USER
# Logout and login again
```

**Webcam Not Working**
```bash
# Test camera stream
curl http://localhost:8080/?action=stream

# Check ffmpeg installation
which ffmpeg
```

## Development

### Virtual Printer Testing
```bash
# Enable virtual printer for development
export OCTOPRINT_VIRTUAL_PRINTER=true
vrooli resource octoprint manage start
```

### Plugin Development
```python
# Example OctoPrint plugin
import octoprint.plugin

class CustomPlugin(octoprint.plugin.StartupPlugin):
    def on_after_startup(self):
        self._logger.info("Custom plugin started!")
```

## Security Notes

- API key required for all operations
- Serial port access needs privileged mode
- File uploads validated for safety
- WebSocket connections authenticated
- Consider firewall rules for remote access

## Resource Dependencies

- **Optional**: postgres (print history), minio (file storage), redis (job queue)
- **Recommended**: webcam software (mjpg-streamer or similar)

## Performance Considerations

- G-code files can be large (100MB+)
- Temperature polling impacts printer performance
- Webcam streaming uses significant bandwidth
- Plugin execution affects response time

## See Also

- [OctoPrint Documentation](https://docs.octoprint.org)
- [G-Code Reference](https://reprap.org/wiki/G-code)
- [Marlin Firmware](https://marlinfw.org)
- [Printer Calibration Guide](https://teachingtechyt.github.io/calibration.html)