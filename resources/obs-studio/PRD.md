# OBS Studio Resource PRD

## Overview
OBS Studio (Open Broadcaster Software) is a professional streaming and recording software resource that enables Vrooli to create, manage, and automate video production workflows. This resource provides programmatic control over streaming, recording, scene management, and audio/video sources through WebSocket APIs.

## Purpose & Value
- **Video Automation**: Enables scenarios to record tutorials, demos, and documentation videos automatically
- **Streaming Workflows**: Supports live streaming scenarios for presentations, events, and broadcasts
- **Content Production**: Automates video content creation for marketing, education, and entertainment
- **Screen Capture**: Provides professional screen recording capabilities for documentation and training
- **Scene Management**: Allows dynamic scene switching for complex production workflows

## Standard Interfaces

### CLI Commands
- [x] `resource-obs-studio install` - Install OBS Studio with WebSocket plugin
- [x] `resource-obs-studio start` - Start OBS Studio service and WebSocket server
- [x] `resource-obs-studio stop` - Stop OBS Studio service
- [x] `resource-obs-studio status` - Check health and configuration (supports --format json)
- [x] `resource-obs-studio content add --file <scene.json>` - Add a scene configuration
- [x] `resource-obs-studio content list` - List all scene configurations
- [x] `resource-obs-studio content get --name <scene-name>` - Get scene configuration
- [x] `resource-obs-studio content remove --name <scene-name>` - Remove scene
- [x] `resource-obs-studio content execute --name <scene-name>` - Activate a scene
- [x] `resource-obs-studio test` - Run integration tests

### Deprecated Commands (for backwards compatibility)
- [x] `resource-obs-studio content add` - Legacy command, redirects to content add

## Configuration Requirements

### Port Registry Integration
- [x] Uses dynamic port allocation from port registry
- [x] Default port: 4455 (WebSocket)
- [x] No hardcoded ports in implementation
- [x] Respects OBS_PORT environment variable

### Docker Image Versioning
- [x] Mock mode for testing (no Docker required)
- [ ] Docker support with specific version tags
- [ ] No 'latest' tags in production

### Environment Variables
- `OBS_PORT` - WebSocket server port (default: 4455)
- `OBS_PASSWORD` - WebSocket authentication password
- `OBS_CONFIG_DIR` - Configuration directory (default: ~/.vrooli/obs-studio)
- `OBS_RECORDINGS_DIR` - Recording output directory (default: ~/Videos/obs-recordings)

## Implementation Status

### Core Functionality
- [x] Installation and setup
- [x] Start/stop management
- [x] Health checks and status reporting
- [x] WebSocket connectivity
- [x] Mock mode for testing
- [x] Scene configuration management
- [x] Recording management
- [x] Content management (add/list/get/remove/execute)
- [ ] Streaming control (basic stub, not fully implemented)
- [ ] Source management (cameras, screens, media)
- [ ] Audio mixer control
- [ ] Transition effects

### Testing
- [x] v2.0 contract compliant test structure
- [x] lib/test.sh with smoke/integration/unit functions
- [x] test/run-tests.sh main runner
- [x] test/phases/ directory with test-smoke.sh, test-integration.sh, test-unit.sh
- [x] All test commands functional (smoke <30s, integration <120s, unit <60s)
- [x] Test results included in status output
- [x] Content management tests (basic)
- [x] Scene management tests (mock mode)
- [x] Recording tests (mock mode)

### Documentation
- [x] Main README.md with overview
- [x] Examples directory with sample workflows
- [x] config/schema.json with complete configuration schema
- [ ] API documentation
- [ ] Scene configuration guide
- [ ] Streaming setup guide

### v2.0 Contract Compliance (2025-09-12)
- [x] Required files: cli.sh, lib/core.sh, lib/test.sh
- [x] Required config files: defaults.sh, schema.json, runtime.json
- [x] Required test structure: run-tests.sh, phases/test-smoke.sh, phases/test-integration.sh, phases/test-unit.sh
- [x] All required CLI commands: help, info, manage (install/uninstall/start/stop/restart), test (smoke/integration/unit/all), content, status, logs
- [x] Proper timeout handling in health checks
- [x] Test time limits enforced (smoke <30s, integration <120s, unit <60s)

## Content Management Specification

### Supported Content Types
1. **Scene Configurations** (.json)
   - Scene layouts and sources
   - Audio/video settings
   - Transition configurations

2. **Recording Profiles** (.json)
   - Output formats and quality
   - Encoder settings
   - File naming patterns

3. **Streaming Profiles** (.json)
   - Stream keys and servers
   - Bitrate and quality settings
   - Platform-specific configurations

### Content Operations
- **Add**: Store scene/profile configurations
- **List**: Show available scenes and profiles
- **Get**: Retrieve specific configuration
- **Remove**: Delete scene or profile
- **Execute**: Activate scene or start recording/streaming

## Integration Points

### Dependencies
- Python 3 (for WebSocket server and control scripts)
- ffmpeg (for video processing)
- Optional: Docker for containerized deployment

### Resource Interactions
- **Browserless**: Can capture browser windows as sources
- **n8n/Huginn**: Trigger recording/streaming from workflows
- **MinIO**: Store recorded videos
- **Whisper**: Transcribe recorded audio

## Quality Requirements

### Performance
- WebSocket response time < 100ms
- Scene switching < 500ms
- Recording start/stop < 2s
- Support for 1080p @ 60fps recording

### Reliability
- Automatic reconnection on WebSocket disconnect
- Graceful handling of missing sources
- Recovery from recording failures
- Persistent scene configurations

### Security
- WebSocket authentication with secure passwords
- No exposure of stream keys in logs
- Secure storage of credentials
- Isolated recording directory

## Migration Notes

### inject → content Migration
- [ ] Implement content management alongside inject
- [ ] Update all examples to use content commands
- [ ] Add deprecation notice to inject command
- [ ] Provide migration script for existing scenes

## Success Metrics
- All validation gates pass
- WebSocket connectivity stable
- Scene management functional
- Recording/streaming capabilities verified
- Integration tests passing
- Documentation complete and accurate

## Progress History

### 2025-09-13: Content Management Verification (85% → 90%)
**Improvements Made:**
- ✅ Verified content management commands fully functional (add/list/get/remove/execute)
- ✅ Fixed example scripts to use correct v2.0 command structure
- ✅ Updated streaming-workflow.sh to demonstrate proper content management
- ✅ Updated basic-recording.sh to use content commands correctly
- ✅ Confirmed all tests passing including content management tests
- ✅ Updated PRD checkboxes to accurately reflect working functionality

**Net Progress:** +5 features verified working, 0 regressions = +5 net

### 2025-09-12: v2.0 Contract Compliance (70% → 85%)
**Improvements Made:**
- ✅ Created lib/test.sh with proper test implementations
- ✅ Created config/schema.json with complete configuration schema
- ✅ Created test/run-tests.sh and phases/ directory structure
- ✅ Fixed all test command implementations (smoke/integration/unit/all)
- ✅ Ensured all tests pass and meet time requirements
- ✅ Updated CLI to properly wire test handlers

**Net Progress:** +6 features working, 0 regressions = +6 net

## Known Issues & Improvements
- WebSocket occasionally fails to start (fixed by restart)
- Mock mode needed for testing environments
- Need to implement content management system
- Scene templates would improve usability

## Future Enhancements
- Virtual camera support
- Advanced audio processing
- Multi-stream output
- Browser source integration
- Automated thumbnail generation
- Stream health monitoring