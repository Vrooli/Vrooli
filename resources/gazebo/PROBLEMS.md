# Gazebo Resource - Known Issues and Problems

## Current Implementation Status

The Gazebo resource currently provides a minimal implementation focused on v2.0 contract compliance and health check functionality. The full physics simulation engine is not yet integrated.

## Known Issues

### 1. Physics Engine Not Installed
**Problem**: The full Gazebo Fortress physics simulation is not installed
**Impact**: Cannot run actual physics simulations
**Workaround**: Using simulation stub for health checks
**Resolution**: Need to install gz-fortress package and dependencies

### 2. Python Dependencies Incomplete
**Problem**: gymnasium, pygazebo, and protobuf not installed
**Impact**: Python API integration limited
**Workaround**: Basic Python health server works without these
**Resolution**: Install full Python dependencies when physics engine available

### 3. ROS 2 Integration Missing
**Problem**: ROS 2 bridge not implemented
**Impact**: Cannot communicate with Robot Operating System
**Workaround**: None currently
**Resolution**: Implement after physics engine integration

### 4. GPU Acceleration Not Available
**Problem**: CUDA/OpenCL support not configured
**Impact**: Simulations limited to CPU processing
**Workaround**: Run smaller simulations
**Resolution**: Add GPU support configuration

## Installation Challenges

### Ubuntu Package Dependencies
The full Gazebo Fortress installation requires numerous packages:
- gz-fortress (main package)
- libgz-* libraries (15+ packages)
- DART physics engine libraries
- Qt5 for GUI components
- OGRE for rendering

These have complex dependency chains that may conflict with other system packages.

## Future Improvements

1. **Phase 1**: Install minimal Gazebo command-line tools
2. **Phase 2**: Add DART physics engine
3. **Phase 3**: Integrate Python API fully
4. **Phase 4**: Add ROS 2 bridge
5. **Phase 5**: GPU acceleration support

## Testing Limitations

- Integration tests cannot validate actual physics simulation
- Performance benchmarks not meaningful without real engine
- Sensor simulation tests not possible

## Workarounds

For users needing immediate simulation capabilities:
1. Use Docker container with Gazebo pre-installed
2. Install Gazebo manually following official documentation
3. Use alternative simulators like PyBullet for simple physics

## References

- [Gazebo Installation Guide](https://gazebosim.org/docs/fortress/install_ubuntu)
- [DART Physics Engine](https://dartsim.github.io/)
- [ROS 2 Gazebo Integration](https://gazebosim.org/docs/fortress/ros2_integration)