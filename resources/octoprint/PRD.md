# OctoPrint Resource PRD

## Executive Summary
**What**: Web-based 3D printer management platform with REST API for remote control, monitoring, and automation  
**Why**: Enable automated 3D printing workflows, print farm management, and integration of physical manufacturing into digital services  
**Who**: Scenarios requiring 3D printing automation, prototyping services, or distributed manufacturing capabilities  
**Value**: $40K+ (automated print services, prototyping-as-a-service, distributed manufacturing coordination)  
**Priority**: P0 - Essential infrastructure for additive manufacturing and rapid prototyping scenarios

## Requirements Checklist

### P0 Requirements (Must Have)
- [x] **Health Check**: Responds within 1s with service status and printer connection state
- [x] **Lifecycle Management**: setup/develop/test/stop commands work reliably
- [ ] **Printer Connection**: Can connect to 3D printers via USB serial ports (requires physical printer)
- [x] **G-Code Upload**: Accept and manage G-code files for printing
- [x] **Print Control**: Start, pause, cancel print jobs via CLI commands (API has auth issues)
- [x] **Temperature Monitoring**: Temperature monitoring command implemented (requires printer for data)
- [x] **Web Interface**: Accessible browser-based control panel on configured port

### P1 Requirements (Should Have)
- [ ] **Webcam Integration**: Live video feed of print progress
- [ ] **Slicing Support**: Integrate with slicing engines for STL to G-code conversion
- [ ] **Print Queue**: Manage multiple print jobs with scheduling
- [ ] **Plugin System**: Support OctoPrint plugins for extended functionality

### P2 Requirements (Nice to Have)
- [ ] **Multi-Printer**: Control multiple printers from single instance
- [ ] **Time-Lapse**: Automatic time-lapse video generation of prints
- [ ] **Failure Detection**: AI-based print failure detection using camera

## Technical Specifications

### Architecture
```yaml
deployment:
  primary: docker_container
  fallback: native_installation
  port: 8197  # From port_registry.sh
  
dependencies:
  runtime:
    - python: ">=3.7"
    - flask: "web framework"
    - pyserial: "serial communication"
  optional:
    - ffmpeg: "webcam streaming"
    - mjpg-streamer: "camera feed"
    
printer_support:
  firmware:
    - marlin: ["1.x", "2.x"]
    - repetier: ["0.92", "1.0"]
    - smoothieware: ["latest"]
    - klipper: ["0.10+"]
  connections:
    - serial: "/dev/ttyUSB*, /dev/ttyACM*"
    - network: "RFC2217, telnet"
    
interfaces:
  api:
    - rest: "full printer control"
    - websocket: "real-time updates"
  plugins:
    - python: "plugin API"
    - javascript: "UI extensions"
```

### Performance Requirements
```yaml
metrics:
  response_time: <200ms      # API responsiveness
  command_latency: <100ms    # Printer command execution
  stream_fps: 15             # Webcam streaming
  max_file_size: 500MB       # G-code files
  concurrent_connections: 20 # WebSocket clients
```

### Security Considerations
- Serial port access requires privileged container mode
- API key authentication for remote access
- File upload validation for G-code safety
- Secure WebSocket connections for real-time data
- Access control for print operations

## Success Metrics

### Completion Targets
- **Phase 1**: 30% - Core structure and health checks ✅
- **Phase 2**: 60% - Printer connection and basic control (55% complete - commands work, API auth issues)
- **Phase 3**: 80% - Full print management and monitoring
- **Phase 4**: 100% - Webcam integration and plugins

### Quality Metrics
- First-time printer connection success >85%
- Print job reliability >95%
- Temperature reading accuracy ±1°C
- Zero data corruption in G-code transfers

### Performance Benchmarks
- 10MB G-code files upload in <5s
- Real-time temperature updates at 1Hz minimum
- Webcam stream stable at 15 FPS
- API response for status queries <200ms

## Revenue Model

### Direct Revenue Streams
- **Print-on-Demand Service**: $20-100 per print job
- **Prototype Services**: $500-5000 per project
- **Print Farm Management**: $100-500/month per printer
- **Custom Manufacturing**: $1000-10000 per batch

### Indirect Value Creation
- Enables physical product scenarios
- Supports rapid prototyping workflows
- Integrates with e-commerce for custom products
- Provides manufacturing capability for IoT devices

### Market Opportunities
- Small-batch manufacturing
- Custom merchandise production
- Architectural model printing
- Medical device prototyping
- Educational STEM programs

## Integration Points

### Vrooli Resources
- **postgres**: Store print history and analytics
- **minio**: Store G-code files and time-lapses
- **redis**: Cache printer status and job queue
- **n8n**: Automate print workflows
- **browserless**: Generate print previews

### Scenarios
- **e-commerce**: Custom product manufacturing
- **design-tools**: CAD to print pipeline
- **iot-factory**: Case printing for devices
- **stem-education**: Educational printing services

## Risk Mitigation

### Technical Risks
- **Printer Compatibility**: Test with virtual printer first
- **Serial Access**: Provide USB passthrough configuration
- **Plugin Security**: Sandbox plugin execution
- **Print Failures**: Implement monitoring and alerts

### Business Risks
- **Material Costs**: Track filament usage
- **Print Time**: Accurate time estimation
- **Quality Control**: Implement verification steps
- **Maintenance**: Schedule regular calibration

## Implementation Notes

### Development Priorities
1. Docker container with Python environment
2. Basic API for printer control
3. Serial communication setup
4. Web interface integration
5. G-code file management
6. Temperature monitoring
7. Webcam support (optional)

### Testing Strategy
- Use virtual printer for development
- Test with common G-code files
- Validate temperature control
- Stress test with large files
- Security audit API endpoints

## Current Implementation Status (2025-09-14)

Successfully improved with:
- ✅ Docker-based deployment working
- ✅ Virtual printer support for testing  
- ✅ Web interface accessible on port 8197
- ✅ G-code file management (upload/list/remove)
- ✅ v2.0 contract compliance
- ✅ Smoke and unit tests passing
- ✅ Temperature monitoring command added
- ✅ Print control commands implemented (start/pause/resume/cancel)
- ✅ Job status command added
- ✅ API key generation improved
- ⚠️ API authentication has configuration issues (see PROBLEMS.md)
- ⚠️ Physical printer support untested

## Success Indicators

### Short Term (1 month)
- Successfully control virtual printer ✅
- Upload and execute G-code ✅
- Monitor temperatures (pending API config)
- Basic web interface working ✅

### Medium Term (3 months)
- Real printer integration
- Webcam streaming
- Plugin system active
- Print queue management

### Long Term (6 months)
- Multi-printer support
- AI failure detection
- Full automation workflows
- Production-ready stability