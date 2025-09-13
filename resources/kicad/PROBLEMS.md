# KiCad Resource - Known Issues & Solutions

## Overview
This document tracks known issues, limitations, and workarounds for the KiCad resource.

## Current Limitations

### 1. KiCad Binary Not Installed
**Issue**: The actual KiCad application is not installed in the Docker environment.
**Impact**: 
- Export functionality operates in mock mode
- Python API is unavailable
- 3D visualization not possible

**Workaround**: The resource provides mock implementations for testing and development. When deployed to a real environment with KiCad installed, full functionality will be available.

**Solution**: To install KiCad:
```bash
# Ubuntu/Debian
sudo apt-get update
sudo apt-get install kicad kicad-cli

# macOS
brew install --cask kicad

# Or use the installer from https://www.kicad.org/
```

### 2. Python API Requires KiCad Installation
**Issue**: The Python API (pcbnew, eeschema modules) requires full KiCad installation.
**Impact**: Automated scripting capabilities limited.
**Workaround**: Use CLI commands for automation where possible.

### 3. Export Formats Limited in Mock Mode
**Issue**: Without kicad-cli binary, export operations complete silently but don't produce output files.
**Impact**: Cannot generate actual Gerber, PDF, SVG, or STEP files.
**Workaround**: Files are created in the correct output directory structure for testing purposes.

## Performance Considerations

### Memory Usage
- Mock mode: <10MB
- With KiCad installed: 200-500MB typical
- Large projects: Can exceed 1GB

### Startup Time
- Mock mode: <1 second
- With KiCad: 8-15 seconds (GUI components)

## Integration Notes

### Docker Deployment
KiCad is a desktop application that requires:
- X11 forwarding for GUI (optional for CLI-only usage)
- Significant disk space (~2GB installed)
- Graphics capabilities for 3D rendering

Consider using a separate container or VM for KiCad if full functionality is needed.

### Headless Operation
The resource is designed to work headless using kicad-cli. GUI is not required for:
- Project import/export
- Library management
- Automated workflows

## Testing

All tests pass in mock mode. When KiCad is installed:
- Integration tests will use actual binary
- Export tests will generate real files
- Python API tests will execute

## Future Improvements

1. **Docker Image**: Create a custom Docker image with KiCad pre-installed
2. **Remote KiCad**: Support connecting to remote KiCad instances
3. **Web Interface**: Add web-based project viewer
4. **CI/CD Integration**: Automated PCB validation in pipelines

## Support

For KiCad-specific issues:
- KiCad Documentation: https://docs.kicad.org/
- KiCad Forums: https://forum.kicad.info/
- Issue Tracker: https://gitlab.com/kicad/code/kicad/-/issues