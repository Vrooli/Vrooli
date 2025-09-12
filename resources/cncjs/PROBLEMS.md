# CNCjs Known Problems and Limitations

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