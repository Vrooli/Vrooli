# Gazebo Robotics Simulation Platform

Gazebo (formerly Ignition) is a modern open-source 3D robotics simulator with advanced physics engines, sensor simulation, and robot modeling capabilities. This resource enables safe virtual testing of robot behaviors, autonomous systems, and control algorithms before real-world deployment.

> **Current Status**: Minimal implementation with health check endpoint and v2.0 contract compliance. Full physics engine integration pending.

## Quick Start

```bash
# Install Gazebo and dependencies
vrooli resource gazebo manage install

# Start Gazebo server
vrooli resource gazebo manage start --wait

# Check status
vrooli resource gazebo status

# Run example simulation
vrooli resource gazebo content execute run-world cart_pole

# Stop Gazebo
vrooli resource gazebo manage stop
```

## Features

### Currently Implemented
- ✅ **v2.0 Contract Compliance**: All required CLI commands working
- ✅ **Health Check Endpoint**: HTTP endpoint at port 11456
- ✅ **Lifecycle Management**: Install, start, stop, restart commands
- ✅ **Basic Testing**: Smoke, unit, and integration tests
- ✅ **Configuration Management**: Runtime and defaults configuration

### Planned Features (Not Yet Implemented)
- ⏳ **3D Physics Simulation**: DART, Bullet, and ODE engines
- ⏳ **Robot Modeling**: URDF and SDF robot descriptions
- ⏳ **Sensor Simulation**: Cameras, LiDAR, IMU, GPS
- ⏳ **Python API**: Full bindings for simulation control
- ⏳ **Distributed Simulation**: Multi-machine support

### Simulation Features
- **Real-time Factor Control**: Speed up or slow down simulations
- **Multi-physics Support**: Switch between physics engines
- **World Management**: Load, save, and modify simulation environments
- **Model Spawning**: Dynamically add robots and objects
- **Performance Monitoring**: Track simulation metrics and resource usage

## Architecture

```
┌─────────────────────────┐
│    Gazebo Resource      │
├─────────────────────────┤
│  Health Monitor (11456) │
│  gRPC Server (11457)    │
│  WebSocket (11458)      │
├─────────────────────────┤
│    Physics Engines      │
│  - DART (default)       │
│  - Bullet               │
│  - ODE                  │
├─────────────────────────┤
│   Integration Points    │
│  - Python API           │
│  - ROS 2 Bridge         │
│  - REST API             │
└─────────────────────────┘
```

## Usage Examples

### Basic Simulation

```bash
# Load a world file
vrooli resource gazebo content add world my_world.world
vrooli resource gazebo content execute run-world my_world

# Spawn a robot model
vrooli resource gazebo content add model robot.sdf
vrooli resource gazebo content execute spawn-robot robot 0 0 1

# Control simulation
vrooli resource gazebo content execute pause
vrooli resource gazebo content execute resume
```

### Python API

```python
#!/usr/bin/env python3
import requests
import json

# Health check
response = requests.get('http://localhost:11456/health')
print(json.dumps(response.json(), indent=2))

# Create simulation
sim_data = {
    "world": "cart_pole",
    "physics_engine": "dart",
    "real_time_factor": 1.0
}
response = requests.post('http://localhost:11456/simulations', json=sim_data)
sim_id = response.json()['id']

# Spawn robot
robot_data = {
    "model": "quadcopter",
    "position": [0, 0, 1],
    "orientation": [0, 0, 0]
}
requests.post(f'http://localhost:11456/models', json=robot_data)

# Get sensor data
sensor_data = requests.get(f'http://localhost:11456/sensors/camera/data')
print(sensor_data.json())
```

### Integration with AI Agents

```python
# Example: Reinforcement Learning Environment
import gymnasium as gym
import numpy as np

class GazeboEnv(gym.Env):
    def __init__(self):
        self.action_space = gym.spaces.Box(-1, 1, (4,))
        self.observation_space = gym.spaces.Box(-np.inf, np.inf, (10,))
        
    def reset(self):
        # Reset simulation
        requests.put('http://localhost:11456/simulations/reset')
        return self._get_observation()
    
    def step(self, action):
        # Apply action
        self._apply_action(action)
        obs = self._get_observation()
        reward = self._calculate_reward()
        done = self._check_done()
        return obs, reward, done, {}
```

## Configuration

### Environment Variables

```bash
# Ports
GAZEBO_PORT=11456              # API and health check
GAZEBO_GRPC_PORT=11457         # gRPC communication
GAZEBO_WS_PORT=11458           # WebSocket updates

# Simulation
GAZEBO_PHYSICS_ENGINE=dart     # Physics engine (dart/bullet/ode)
GAZEBO_REAL_TIME_FACTOR=1.0    # Simulation speed
GAZEBO_MAX_STEP_SIZE=0.001     # Step size in seconds

# Resources
GAZEBO_MAX_MEMORY=2G           # Memory limit
GAZEBO_MAX_CPU=2               # CPU cores
GAZEBO_MAX_SIMULATIONS=5       # Concurrent simulations

# Rendering
GAZEBO_HEADLESS=true           # Run without GUI
GAZEBO_ENABLE_GPU=false        # GPU acceleration
```

### Configuration Files

- `config/defaults.sh`: Default environment variables
- `config/runtime.json`: Runtime configuration and dependencies
- `config/schema.json`: Configuration validation schema

## API Reference

### Health Check
```
GET /health
Returns: {
  "status": "healthy",
  "service": "gazebo",
  "version": "fortress",
  "uptime": 3600,
  "simulation_running": true
}
```

### Simulations
```
POST /simulations         - Create new simulation
GET /simulations/{id}     - Get simulation state
PUT /simulations/{id}/pause   - Pause simulation
PUT /simulations/{id}/resume  - Resume simulation
DELETE /simulations/{id}  - Stop simulation
```

### World Management
```
POST /worlds             - Load world file
GET /worlds              - List available worlds
PUT /worlds/{id}         - Modify world parameters
```

### Model Management
```
POST /models             - Spawn robot/object
GET /models              - List spawned models
PUT /models/{id}/pose    - Set model position
DELETE /models/{id}      - Remove model
```

### Sensor Data
```
GET /sensors             - List available sensors
GET /sensors/{id}/data   - Get sensor readings
POST /sensors/{id}/subscribe - Subscribe to sensor stream
```

## Testing

```bash
# Run smoke tests (< 30s)
vrooli resource gazebo test smoke

# Run integration tests
vrooli resource gazebo test integration

# Run all tests
vrooli resource gazebo test all

# View test results
vrooli resource gazebo logs
```

## Troubleshooting

### Common Issues

**Gazebo won't start**
```bash
# Check if port is in use
netstat -tlnp | grep 11456

# Check logs
vrooli resource gazebo logs

# Restart with verbose output
GAZEBO_VERBOSE=true vrooli resource gazebo manage restart
```

**Simulation running slowly**
```bash
# Reduce real-time factor
export GAZEBO_REAL_TIME_FACTOR=0.5

# Increase step size
export GAZEBO_MAX_STEP_SIZE=0.01

# Disable rendering
export GAZEBO_HEADLESS=true
```

**Python API not connecting**
```bash
# Check health endpoint
curl http://localhost:11456/health

# Verify Python dependencies
python3 -c "import gymnasium, numpy, protobuf"

# Check firewall settings
sudo ufw status
```

## Performance Optimization

### Resource Limits
- Set appropriate memory limits: `GAZEBO_MAX_MEMORY=4G`
- Limit CPU cores: `GAZEBO_MAX_CPU=4`
- Control concurrent simulations: `GAZEBO_MAX_SIMULATIONS=3`

### Physics Optimization
- Use larger step sizes for faster simulation
- Choose appropriate physics engine for your use case
- Disable unused sensors and plugins

### Headless Operation
- Always use headless mode for production: `GAZEBO_HEADLESS=true`
- Disable GPU if not needed: `GAZEBO_ENABLE_GPU=false`

## Examples

### Cart-Pole Balance
```python
# Simple cart-pole control example
import requests
import time

# Start simulation
requests.post('http://localhost:11456/simulations', 
              json={"world": "cart_pole"})

# Control loop
for _ in range(1000):
    # Get cart position
    state = requests.get('http://localhost:11456/models/cart/state').json()
    
    # Simple control law
    force = -0.1 * state['position'] - 0.01 * state['velocity']
    
    # Apply force
    requests.put('http://localhost:11456/models/cart/force',
                 json={"force": [force, 0, 0]})
    
    time.sleep(0.01)
```

### Multi-Robot Coordination
```python
# Spawn multiple robots
for i in range(3):
    requests.post('http://localhost:11456/models',
                  json={
                      "model": "quadcopter",
                      "name": f"drone_{i}",
                      "position": [i*2, 0, 1]
                  })

# Coordinate movement
target = [5, 5, 3]
for i in range(3):
    requests.put(f'http://localhost:11456/models/drone_{i}/target',
                 json={"position": target})
```

## Integration with Other Resources

### With ROS 2 (when available)
```bash
# Enable ROS 2 bridge
export GAZEBO_ENABLE_ROS2=true
export GAZEBO_ROS2_DOMAIN_ID=0

# Launch with ROS 2 integration
vrooli resource gazebo manage restart
```

### With Ollama for AI Control
```python
# Use Ollama to generate control commands
import requests

# Get scene description
scene = requests.get('http://localhost:11456/scene/description').json()

# Ask Ollama for control strategy
prompt = f"Given this robot scene: {scene}, what control commands should I send?"
ollama_response = requests.post('http://localhost:11434/generate',
                                json={"model": "llama2", "prompt": prompt})

# Parse and apply commands
commands = parse_ollama_response(ollama_response.json())
apply_commands(commands)
```

## Security Considerations

- Simulations run in isolated process space
- No direct hardware access without explicit permission
- API authentication available for multi-user scenarios
- Resource limits prevent runaway simulations
- Logs sanitized to prevent information leakage

## Support

- [Official Gazebo Documentation](https://gazebosim.org/docs)
- [DART Physics Documentation](https://dartsim.github.io/)
- [ROS 2 Integration Guide](https://gazebosim.org/docs/fortress/ros2_integration)
- [SDF Format Specification](http://sdformat.org/)

## License

Gazebo is licensed under Apache 2.0. This resource wrapper follows the same license.