# PyBullet Physics Simulation Resource

PyBullet is a Python-based physics simulation engine that provides fast and accurate rigid body dynamics, collision detection, and robotics simulation capabilities for AI and robotics development.

## Quick Start

```bash
# Install PyBullet
vrooli resource pybullet manage install

# Start API server
vrooli resource pybullet manage start --wait

# Run smoke test
vrooli resource pybullet test smoke

# Execute demo simulation
vrooli resource pybullet content execute pendulum
```

## Features

- **Bullet Physics Engine**: Industry-standard physics simulation
- **Python API**: Complete Python bindings for easy integration
- **URDF/SDF Support**: Load standard robot model formats
- **Headless Mode**: Run simulations without GUI for automation
- **Real-time Control**: Precise simulation time stepping at 240Hz
- **REST API**: HTTP endpoints for remote simulation control

## Use Cases

- **Robotics Development**: Test robot behaviors safely
- **Reinforcement Learning**: Train AI agents in physics environments
- **Game Physics**: Prototype game mechanics
- **Education**: Teach physics and robotics concepts
- **Research**: Validate control algorithms

## API Endpoints

- `GET /health` - Service health check
- `POST /simulation/create` - Create new simulation
- `DELETE /simulation/{name}` - Destroy simulation
- `POST /simulation/{name}/step` - Advance simulation (with real-time control)
- `GET /simulation/{name}/state` - Get current state
- `POST /simulation/{name}/spawn` - Add objects (box, sphere, cylinder)
- `POST /simulation/{name}/apply_force` - Apply forces to bodies
- `POST /simulation/{name}/set_joint` - Control robot joints
- `GET /simulation/{name}/sensors` - Read sensor data

## Configuration

Default configuration in `config/defaults.sh`:
- Physics timestep: 240Hz (0.004167s)
- Gravity: [0, 0, -9.81] m/s²
- Max simulations: 10 concurrent
- API timeout: 300 seconds

## Examples

### Create and Run Simulation

```python
import httpx

# Create simulation
response = httpx.post("http://localhost:11457/simulation/create", 
    json={"name": "test", "gravity": [0, 0, -9.81]})

# Spawn a box
httpx.post("http://localhost:11457/simulation/test/spawn",
    json={"shape": "box", "position": [0, 0, 2]})

# Step simulation
httpx.post("http://localhost:11457/simulation/test/step",
    json={"steps": 100})

# Get state
state = httpx.get("http://localhost:11457/simulation/test/state").json()
print(f"Bodies: {state['num_bodies']}")
```

### Available Demo Simulations

- `pendulum` - Simple pendulum physics demonstration
- `bouncing_ball` - Three balls with different restitution values
- `robotic_arm` - 3-DOF robotic arm with inverse kinematics control
- `multi_body_chain` - Chain and cloth simulation with soft constraints

Run with: `vrooli resource pybullet content execute <name>`

## Testing

```bash
# Quick health check
vrooli resource pybullet test smoke

# Full integration tests
vrooli resource pybullet test integration

# All tests
vrooli resource pybullet test all
```

## Troubleshooting

### Installation Issues
- Ensure Python 3.8+ and pip are installed
- If python3-venv is not available, the resource will use a fallback installation method
- Check virtual environment creation: `ls -la .venv/`

### API Server Not Starting
- Check port availability: `lsof -i :11457`
- View logs: `vrooli resource pybullet logs`

### Simulation Errors
- Verify PyBullet import: `source .venv/bin/activate && python -c "import pybullet"`
- Check API health: `curl http://localhost:11457/health`

## Integration with Vrooli

PyBullet integrates with other Vrooli resources:
- Use with reinforcement learning scenarios
- Combine with computer vision for robotic perception
- Export simulation data to analytics platforms

## Performance

- Simulation speed: 240Hz stable for 10+ rigid bodies
- Memory usage: <500MB typical
- Startup time: <10 seconds
- Concurrent simulations: Up to 10

## License

PyBullet is released under the zlib license. This resource wrapper follows Vrooli's licensing terms.