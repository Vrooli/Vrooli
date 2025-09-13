# Home Assistant Resource - Product Requirements Document

## Executive Summary
**What**: Open-source home automation platform that integrates smart devices and automates household processes  
**Why**: Enable unified control of IoT devices, automation workflows, and smart home intelligence  
**Who**: Developers building home automation solutions and AI agents creating smart environments  
**Value**: $15K-25K per deployment through energy savings, security automation, and productivity gains  
**Priority**: High - Core infrastructure for smart home scenarios and IoT integration

## Requirements Checklist

### P0 Requirements (Must Have)
- [x] **Core Platform**: Home Assistant container runs with proper configuration
- [x] **Health Monitoring**: Health endpoint responds within 5 seconds with service status
- [x] **Integration API**: REST API accessible for device control and automation
- [x] **Automation Engine**: Supports loading custom automations and scripts
- [x] **Data Persistence**: Configuration and state persist across restarts
- [x] **Secure Access**: Authentication required for API access (401 returned without auth)
- [x] **v2.0 Contract**: Full compliance with universal resource contract

### P1 Requirements (Should Have)
- [x] **Device Discovery**: Auto-discovers compatible devices on network (via host network mode)
- [x] **Webhook Support**: Receives webhooks for external triggers (tested and working)
- [x] **Backup/Restore**: Configuration backup and restoration capabilities (implemented with docker exec)
- [ ] **Multi-user Support**: Different access levels for users (requires manual configuration)

### P2 Requirements (Nice to Have)
- [ ] **Custom Components**: Support for community integrations
- [ ] **Voice Control**: Integration with voice assistants
- [ ] **Energy Management**: Track and optimize energy usage

## Technical Specifications

### Architecture
- **Container**: Docker-based deployment with homeassistant/home-assistant:stable
- **Port**: 8123 (registered in port_registry.sh)
- **API**: RESTful API with JSON responses
- **Storage**: SQLite for recorder, JSON for configuration
- **Authentication**: Long-lived access tokens or OAuth2

### Dependencies
- Docker for containerization
- PostgreSQL (optional) for recorder backend
- Redis (optional) for session management

### API Endpoints
- `GET /api/` - API status (401 when auth required, 200 when open)
- `GET /api/states` - List all entity states
- `POST /api/services/{domain}/{service}` - Call service
- `GET /api/config` - Get configuration
- `POST /api/config/core/check_config` - Validate configuration

### Configuration Structure
```yaml
# config/secrets.yaml
version: "1.0"
resource: "home-assistant"
description: "Authentication tokens and API keys for integrations"

secrets:
  api_keys:
    - name: "long_lived_token"
      path: "secret/resources/home-assistant/api/token"
      description: "Long-lived access token for API access"
      required: false
      format: "string"
      default_env: "HOME_ASSISTANT_TOKEN"
```

### Resource Lifecycle
1. **Install**: Pull Docker image, create directories
2. **Start**: Launch container with volume mounts
3. **Health Check**: Verify API endpoint responds
4. **Configure**: Load automations and integrations
5. **Stop**: Graceful shutdown with state preservation

## Success Metrics

### Completion Targets
- **P0 Completion**: 100% required for production
- **P1 Completion**: 50% for enhanced functionality
- **P2 Completion**: Optional improvements

### Quality Metrics
- Health check response time < 1 second
- API response time < 500ms for standard calls
- Container startup time < 30 seconds
- Zero data loss on restart
- 99.9% uptime when running

### Performance Targets
- Support 100+ devices/entities
- Handle 1000+ automation rules
- Process 100+ events/second
- < 500MB memory usage baseline
- < 5% CPU usage at idle

## Testing Requirements

### Smoke Tests (< 30s)
- Container starts successfully
- Health endpoint responds
- API is accessible
- Web UI loads

### Integration Tests (< 120s)
- Create and trigger automation
- Device state changes
- Webhook reception
- Configuration persistence
- API authentication

### Validation Commands
```bash
# Quick health check
timeout 5 curl -sf http://localhost:8123/api/

# Full test suite
vrooli resource home-assistant test all

# Check automation loading
vrooli resource home-assistant content list
```

## Revenue Justification

### Direct Value ($15K-25K per deployment)
- **Energy Savings**: $3-5K/year through intelligent HVAC, lighting control
- **Security Automation**: $5-8K value in monitoring and response
- **Productivity Gains**: $2-3K/year in time savings from automation
- **Property Value**: $5-10K increase in smart home capability

### Scenario Integration Value
- **Smart Office**: Occupancy-based climate control, meeting room automation
- **Retail Analytics**: Customer flow tracking, energy optimization
- **Healthcare Monitoring**: Patient room automation, equipment tracking
- **Industrial IoT**: Sensor integration, predictive maintenance

## Implementation Notes

### Current Status
- Core container management implemented
- Basic health checks functional
- CLI follows v2.0 patterns
- Missing: test infrastructure, secrets management, PRD

### Priority Improvements
1. Add test suite (test/run-tests.sh and phases/)
2. Implement secrets management (config/secrets.yaml)
3. Add configuration schema validation
4. Enhance health monitoring with timeout handling
5. Document automation injection patterns

## Progress History
- 2025-01-12: Initial PRD creation, 0% complete
- 2025-01-12: 85% P0 complete (6/7), 25% P1 complete (1/4)
  - ✅ Implemented full v2.0 contract compliance
  - ✅ Created comprehensive test suite (smoke/integration/unit)  
  - ✅ Added secrets management via config/secrets.yaml
  - ✅ Fixed lifecycle management and health checks
  - ✅ Added config/schema.json for validation
  - ✅ Container management fully operational
  - ✅ Automation injection system working
  - ⚠️ Authentication available but not enforced by default
  - 📝 Created complete documentation
- 2025-09-12: 100% P0 complete (7/7), 75% P1 complete (3/4)
  - ✅ Fixed authentication - API properly returns 401 without auth
  - ✅ Implemented backup/restore functionality using docker exec for permissions
  - ✅ Added webhook support - tested and confirmed working
  - ✅ Enhanced integration tests with backup/restore and webhook validation
  - ✅ Added CLI commands for backup/restore/list operations
  - ⚠️ Multi-user support requires manual configuration in Home Assistant UI
  - ⚠️ Port registry integration simplified due to registry limitations