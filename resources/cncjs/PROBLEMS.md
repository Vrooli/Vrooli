# CNCjs Known Problems and Limitations

## Update: 2025-09-16 Improvements

### Latest Enhancements
- **Error Handling**: Added recovery hints for common startup issues (port conflicts, permissions)
- **Health Check Compliance**: Updated all timeouts to v2.0 standard (5 seconds)
- **Logs Command**: Enhanced with --follow, --tail options and better user guidance
- **PRD Accuracy**: Updated to reflect actual working features (position tracking, emergency stop)
- **Code Robustness**: Improved error detection and user feedback throughout

## Update: 2025-09-12 Improvements

### What Was Implemented
- **Workflow Storage**: Full workflow management system for multi-step CNC jobs
  - Create, manage, and execute job sequences
  - Export/import workflows for sharing
  - Automatic step queuing
- **Multi-Controller Support**: Profile-based configuration for different CNC controllers
  - Support for Grbl, Marlin, Smoothieware, TinyG, g2core
  - Controller-specific default settings
  - Easy profile switching
- **Enhanced Testing**: Comprehensive test coverage for all new features

### Key Improvements Made
1. Added `workflow` command for managing job sequences
2. Added `controller` command for configuring different CNC types
3. Export/import capability for sharing workflows
4. Controller profile system with type-specific defaults
5. All tests passing without hardware requirements

## Hardware Requirements
- **Serial Port Access**: Requires actual CNC hardware connected via USB/serial for controller features
- **Docker Privileged Mode**: Needs --privileged flag for serial port access (security consideration)

## Feature Limitations
- **Controller State**: Cannot test real-time position, emergency stop, or actual G-code execution without hardware
- **WebSocket API**: Full socket.io integration not tested (requires authentication setup)
- **Authentication**: API endpoints require token authentication which is not yet implemented in CLI

## Workarounds
- **Hardware Testing**: Use simulator mode or virtual serial ports for development
- **Macro Execution**: Macros are queued but require controller connection to actually run
- **3D Visualization**: Use web interface directly for WebGL visualization features

## Future Improvements
- Add authentication token management for API access
- Implement virtual controller mode for testing
- Add WebSocket client for real-time status monitoring
- Support for network-connected controllers (ESP32, Raspberry Pi)