# Godot Engine Resource

Free, open-source game engine with built-in GDScript language, node-based scene system, and comprehensive 2D/3D development tools.

## Overview

Godot Engine provides a complete game development platform with:
- GDScript, C#, C++, and GDExtension support
- Node and Scene system architecture  
- Vulkan/OpenGL rendering
- AI assistant integration for code generation
- Export to desktop, mobile, web, and consoles

## Quick Start

```bash
# Install and start Godot
vrooli resource godot manage install
vrooli resource godot manage start --wait

# Check health status
vrooli resource godot status

# Create a new project
curl -X POST http://localhost:11457/api/projects \
  -H "Content-Type: application/json" \
  -d '{"name": "MyGame", "template": "2d-platformer"}'

# Access GDScript LSP
# Configure your editor to connect to localhost:6005
```

## Usage

### Managing Projects

```bash
# List all projects
curl http://localhost:11457/api/projects

# Get project details
curl http://localhost:11457/api/projects/my-game

# Build project
curl -X POST http://localhost:11457/api/projects/my-game/build

# Export for web
curl -X POST http://localhost:11457/api/projects/my-game/export \
  -H "Content-Type: application/json" \
  -d '{"platform": "HTML5"}'
```

### AI Code Generation

```bash
# Generate GDScript code with AI
curl -X POST http://localhost:11457/api/gdscript/generate \
  -H "Content-Type: application/json" \
  -d '{"prompt": "Create a player controller with jump and movement"}'
```

### GDScript Language Server

The Language Server Protocol (LSP) runs on port 6005 and provides:
- Code completion
- Go to definition
- Hover documentation
- Diagnostics and error reporting

Configure your editor to connect to `localhost:6005` for GDScript support.

## Configuration

### Environment Variables

- `GODOT_API_PORT`: API server port (default: 11457)
- `GODOT_LSP_PORT`: Language Server Protocol port (default: 6005)
- `GODOT_VERSION`: Godot engine version (default: 4.3)
- `GODOT_PROJECTS_DIR`: Projects storage directory
- `GODOT_EXPORTS_DIR`: Export outputs directory

### Resource Dependencies

- **minio**: Asset storage for textures, models, audio
- **ollama**: AI-powered code generation (optional)

## Testing

```bash
# Run all tests
vrooli resource godot test all

# Quick health check
vrooli resource godot test smoke

# Integration tests
vrooli resource godot test integration
```

## API Reference

### Health Check
- **GET** `/health` - Service health status

### Projects
- **POST** `/api/projects` - Create new project
- **GET** `/api/projects` - List all projects
- **GET** `/api/projects/{id}` - Get project details
- **DELETE** `/api/projects/{id}` - Delete project

### Building
- **POST** `/api/projects/{id}/build` - Build project
- **POST** `/api/projects/{id}/export` - Export to platform
- **GET** `/api/projects/{id}/export/{job}` - Check export status

### Assets
- **GET** `/api/projects/{id}/assets` - List project assets
- **POST** `/api/projects/{id}/assets` - Upload asset
- **DELETE** `/api/projects/{id}/assets/{name}` - Delete asset

### AI Features
- **POST** `/api/gdscript/generate` - Generate code with AI
- **POST** `/api/gdscript/explain` - Explain code with AI
- **POST** `/api/gdscript/optimize` - Optimize code with AI

## Export Platforms

Supported export targets:
- **HTML5**: Web browser deployment
- **Linux**: Desktop Linux builds
- **Windows**: Windows executables
- **Android**: Mobile APK (requires templates)
- **iOS**: iOS builds (requires templates)

## Project Templates

Available starter templates:
- `empty` - Blank project
- `2d-platformer` - 2D platform game
- `3d-fps` - 3D first-person shooter
- `puzzle` - Puzzle game template
- `rpg` - RPG game framework

## Troubleshooting

### Service won't start
```bash
# Check logs
vrooli resource godot logs

# Verify port availability
netstat -tlnp | grep 11457
```

### LSP not connecting
```bash
# Ensure LSP is running
curl http://localhost:11457/health

# Check LSP port
netstat -tlnp | grep 6005
```

### Export fails
```bash
# Verify export templates are installed
vrooli resource godot content list

# Install missing templates
vrooli resource godot content add --type template --name HTML5
```

## Performance Tuning

### Memory Limits
```bash
# Set memory limit for Godot process
export GODOT_MEMORY_LIMIT=2G
```

### Concurrent Projects
```bash
# Limit concurrent project builds
export GODOT_MAX_CONCURRENT_BUILDS=2
```

## Security Considerations

- Projects run in isolated containers
- File system access is restricted
- API requires authentication for production use
- Export outputs are scanned for security

## License

Godot Engine is licensed under the MIT License. This resource wrapper follows the same licensing terms.