# ROS2 Resource

Robot Operating System 2 (ROS2) middleware for building distributed robotics applications with real-time control, sensor integration, and multi-robot coordination.

## Overview

ROS2 provides industrial-grade middleware for robotics applications, enabling:
- **Distributed Computing**: Multi-machine robot control
- **Real-time Control**: Deterministic communication with QoS
- **Hardware Abstraction**: Unified interface for diverse hardware
- **Sensor Fusion**: Combine data from multiple sensors
- **Multi-robot Coordination**: Fleet management and swarm robotics

## Quick Start

```bash
# Install ROS2
vrooli resource ros2 manage install

# Start ROS2 services
vrooli resource ros2 manage start

# Check health
vrooli resource ros2 test smoke

# Launch demo nodes
vrooli resource ros2 content launch-node talker
vrooli resource ros2 content launch-node listener

# List active topics
vrooli resource ros2 content list-topics
```

## Current Status

**Working**: Full P0 functionality complete (100% P0 completion) ✅
- Health endpoint responds correctly
- DDS middleware configured and operational
- Node, topic, and service management via API
- Service client/server implementation complete
- Parameter server with persistence operational
- All tests passing (smoke, integration, unit)
- Docker container deployment working

## Features

### Core Capabilities
- ✅ DDS-based communication (Fast-DDS/CycloneDDS)
- ✅ Node lifecycle management
- ✅ Topic publishing/subscribing
- ✅ Service client/server patterns
- ✅ Parameter server
- ✅ Health monitoring API
- ✅ v2.0 contract compliance
- ✅ Docker-based deployment for compatibility
- ✅ RESTful API for all operations
- ✅ Comprehensive test coverage

### Integration Points
- **Gazebo**: 3D simulation environment
- **PyBullet**: Physics simulation
- **ESPHome**: IoT sensor nodes
- **Home Assistant**: Smart home integration

## Configuration

Configuration is managed through environment variables:

```bash
# Port configuration
export ROS2_PORT=11501          # API server port

# ROS2 settings
export ROS2_DOMAIN_ID=0         # Network isolation domain
export ROS2_MIDDLEWARE=fastdds  # DDS implementation
export ROS2_DISTRO=humble       # ROS2 distribution

# Performance tuning
export ROS2_MAX_NODES=100       # Maximum concurrent nodes
export ROS2_BUFFER_SIZE=65536   # Message buffer size
```

## API Endpoints

The ROS2 resource provides a REST API on port 11501:

### Health & Status
- `GET /health` - Service health check
- `GET /nodes` - List active nodes
- `GET /topics` - List available topics
- `GET /services` - List available services

### Node Management
- `POST /nodes/launch` - Launch a new node
- `DELETE /nodes/{id}` - Stop a node

### Communication
- `POST /topics/{name}/publish` - Publish to topic
- `GET /topics/{name}/subscribe` - Subscribe via WebSocket
- `POST /services/{name}/call` - Call service

### Parameters
- `GET /params/{node}` - Get node parameters
- `PUT /params/{node}` - Set node parameters
- `GET /params/list` - List all parameters across nodes
- `POST /params/save` - Save parameters to persistent storage
- `POST /params/load` - Load parameters from persistent storage

### Service Management
- `POST /services/create` - Create a new service server
- `DELETE /services/{name}` - Remove a service

## Usage Examples

### Basic Talker/Listener
```bash
# Terminal 1: Launch talker node
vrooli resource ros2 content launch-node talker

# Terminal 2: Launch listener node
vrooli resource ros2 content launch-node listener

# View communication
vrooli resource ros2 content list-topics
```

### Multi-Robot Coordination
```bash
# Set different domain IDs for robot isolation
export ROS2_DOMAIN_ID=1
vrooli resource ros2 content launch-node robot1_controller

export ROS2_DOMAIN_ID=2
vrooli resource ros2 content launch-node robot2_controller
```

### Sensor Integration
```bash
# Launch sensor nodes
vrooli resource ros2 content launch-node camera_driver
vrooli resource ros2 content launch-node lidar_driver

# Launch fusion node
vrooli resource ros2 content launch-node sensor_fusion
```

### Service & Parameter Management
```bash
# Create a service
curl -X POST http://localhost:11501/services/create \
  -H "Content-Type: application/json" \
  -d '{"name":"robot_status", "type":"std_srvs/srv/Trigger"}'

# Call the service
curl -X POST http://localhost:11501/services/robot_status/call \
  -H "Content-Type: application/json" \
  -d '{}'

# Set parameters for a node
curl -X PUT http://localhost:11501/params/navigation_controller \
  -H "Content-Type: application/json" \
  -d '{"max_velocity": 2.0, "obstacle_distance": 0.5}'

# Save parameters persistently
curl -X POST http://localhost:11501/params/save

# Run the comprehensive demo
python3 /home/matthalloran8/Vrooli/resources/ros2/examples/service_param_demo.py
```

## Testing

```bash
# Quick health check (<30s)
vrooli resource ros2 test smoke

# Full integration test (<120s)
vrooli resource ros2 test integration

# Unit tests (<60s)
vrooli resource ros2 test unit

# All tests (<600s)
vrooli resource ros2 test all
```

## Development

### Directory Structure
```
ros2/
├── cli.sh              # CLI interface
├── config/            
│   ├── defaults.sh     # Default configuration
│   ├── runtime.json    # Runtime metadata
│   └── schema.json     # Configuration schema
├── lib/
│   ├── core.sh         # Core functionality
│   ├── test.sh         # Test functions
│   └── api_server.py   # REST API server
├── test/
│   ├── run-tests.sh    # Test runner
│   └── phases/         # Test phases
├── examples/           # Usage examples
├── docs/               # Documentation
├── PRD.md              # Product requirements
└── README.md           # This file
```

### Adding Custom Nodes

Create a Python node in `examples/`:

```python
#!/usr/bin/env python3
import rclpy
from rclpy.node import Node
from std_msgs.msg import String

class CustomPublisher(Node):
    def __init__(self):
        super().__init__('custom_publisher')
        self.publisher_ = self.create_publisher(String, 'custom_topic', 10)
        timer_period = 1.0
        self.timer = self.create_timer(timer_period, self.timer_callback)
        
    def timer_callback(self):
        msg = String()
        msg.data = 'Custom message'
        self.publisher_.publish(msg)

def main():
    rclpy.init()
    node = CustomPublisher()
    rclpy.spin(node)
    node.destroy_node()
    rclpy.shutdown()

if __name__ == '__main__':
    main()
```

## Troubleshooting

### Common Issues

**ROS2 not starting:**
```bash
# Check if ports are available
ss -tuln | grep 11501

# Check logs
vrooli resource ros2 logs

# Restart with verbose output
export ROS2_DEBUG=true
vrooli resource ros2 manage restart
```

**Domain ID conflicts:**
```bash
# Use different domain IDs for isolation
export ROS2_DOMAIN_ID=42
vrooli resource ros2 manage restart
```

**DDS discovery issues:**
```bash
# Check multicast is enabled
ip link show | grep MULTICAST

# Use localhost-only mode
export ROS_LOCALHOST_ONLY=1
```

## Performance Tuning

### Message Throughput
```bash
# Increase buffer sizes for high-frequency topics
export ROS2_BUFFER_SIZE=262144  # 256KB

# Tune DDS QoS settings
export FASTRTPS_DEFAULT_PROFILES_FILE=/path/to/qos_profiles.xml
```

### Multi-Machine Setup
```bash
# Configure DDS discovery for multi-machine
export ROS_DISCOVERY_SERVER=192.168.1.100:11811
```

## Security

ROS2 supports DDS Security for encrypted communication:

```bash
# Enable security
export ROS2_SECURITY_ENABLE=true
export ROS_SECURITY_KEYSTORE=/path/to/keystore

# Generate security artifacts
ros2 security create_keystore /path/to/keystore
ros2 security create_key /path/to/keystore /talker
```

## Integration with Vrooli

ROS2 enables sophisticated robotics scenarios:

1. **Autonomous Navigation**: Path planning and obstacle avoidance
2. **Swarm Robotics**: Coordinate multiple robots
3. **Sensor Processing**: Real-time sensor data fusion
4. **Industrial Automation**: Factory robot control
5. **Educational Robotics**: Teaching robotics concepts

## Resources

- [ROS2 Documentation](https://docs.ros.org/en/humble/)
- [ROS2 Tutorials](https://docs.ros.org/en/humble/Tutorials.html)
- [DDS Specification](https://www.omg.org/spec/DDS/)
- [Fast-DDS Documentation](https://fast-dds.docs.eprosima.com/)

## License

ROS2 is licensed under Apache 2.0. This resource wrapper follows Vrooli's licensing.