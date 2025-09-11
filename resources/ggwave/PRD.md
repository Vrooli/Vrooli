# Product Requirements Document (PRD) - GGWave

## 🎯 Infrastructure Definition

### Core Infrastructure Capability
**What permanent infrastructure capability does this resource add to Vrooli?**
GGWave provides secure air-gapped data transmission using frequency-shift keying (FSK) modulated audio signals. This enables scenarios to communicate data through sound waves without network connectivity, supporting 8-500 bytes/sec data rates with Reed-Solomon error correction for reliable transfers.

### System Amplification
**How does this resource make the entire Vrooli system more capable?**
- **Air-Gapped Security**: Enables secure data transfer without network exposure, eliminating attack vectors
- **Proximity Authentication**: Natural physical proximity requirement for sound-based authentication
- **Network-Free Communication**: Scenarios can operate in environments with no network access
- **Creative Integration**: Opens new possibilities for IoT pairing, secure handshakes, and novel UX
- **Cross-Platform Support**: Works across iOS, Android, Linux, Arduino with standard audio hardware

### Enabling Value
**What new scenarios become possible when this resource is available?**
1. **Secure Credential Exchange**: Transfer API keys, passwords, tokens via air-gapped sound
2. **IoT Device Pairing**: Network-free device pairing and configuration using ultrasonic mode
3. **Offline Payment Systems**: Transmit payment data in network-dead zones
4. **Emergency Communications**: Send critical data when all networks are down
5. **Creative Installations**: Interactive audio-based experiences and art installations

## 📊 Infrastructure Metrics

### Functional Requirements
- **Must Have (P0)**
  - [ ] **v2.0 Contract Compliance**: Full lifecycle management with all required commands
  - [ ] **Health Check Endpoint**: Responds within 5 seconds with service status
  - [ ] **FSK Modulation Engine**: Core audio signal generation and decoding
  - [ ] **Multiple Transmission Modes**: Normal (8-64 bps), Fast (32-64 bps), DT (64-500 bps), Ultrasonic (inaudible)
  - [ ] **Reed-Solomon Error Correction**: Reliable data transmission with error recovery
  - [ ] **REST API**: HTTP endpoints for encoding/decoding operations
  - [ ] **WebSocket Support**: Real-time bidirectional audio streaming
  
- **Should Have (P1)**
  - [ ] **Protocol Selection**: Automatic mode selection based on environment
  - [ ] **Multi-Channel Support**: Parallel transmission on different frequencies
  - [ ] **Session Management**: Track and manage active transmission sessions
  - [ ] **Performance Monitoring**: Transmission success rates and error statistics
  
- **Nice to Have (P2)**
  - [ ] **Custom Protocols**: User-defined modulation schemes
  - [ ] **Audio Watermarking**: Embed data in existing audio streams

### Performance Criteria
| Metric | Target | Measurement Method |
|--------|--------|-------------------|
| Service Startup Time | < 10s | Container initialization |
| Health Check Response | < 500ms | API status endpoint |
| Encoding Latency | < 100ms | Time to generate audio |
| Decoding Latency | < 200ms | Time to extract data |
| Error Rate | < 1% at 1m | Test transmission accuracy |

### Quality Gates
- [ ] All P0 requirements implemented and tested
- [ ] Integration tests pass with audio hardware
- [ ] Performance targets met for all transmission modes
- [ ] Security validation for air-gapped operation
- [ ] Documentation complete with examples

## 🏗️ Technical Architecture

### Resource Dependencies
```yaml
required:
  # No required dependencies - standalone service
    
optional:
  - resource_name: ffmpeg
    purpose: Advanced audio processing and format conversion
    fallback: Use built-in audio processing
    access_method: CLI when available
```

### Integration Standards
```yaml
resource_category: communication

standard_interfaces:
  management:
    - cli: cli.sh (using CLI framework)
    - actions: [help, info, manage, test, content, status, logs]
    - configuration: config/defaults.sh
    - documentation: README.md + docs/
    
  networking:
    - docker_networks: [vrooli-network]
    - port_registry: Port 8196 (defined in scripts/resources/port_registry.sh)
    - hostname: vrooli-ggwave
    
  monitoring:
    - health_check: GET /health
    - status_reporting: resource-ggwave status
    - logging: Docker container logs
    
  data_persistence:
    - volumes: [ggwave-data]
    - backup_strategy: Session logs and configuration
    - migration_support: Version-agnostic data format

integration_patterns:
  scenarios_using_resource:
    - scenario_name: secure-credential-manager
      usage_pattern: Air-gapped credential transfer
      
    - scenario_name: iot-device-configurator
      usage_pattern: Ultrasonic device pairing
      
  resource_to_resource:
    - ffmpeg → ggwave: Audio format conversion
    - ggwave → vault: Secure credential storage after transfer
```

### Configuration Schema
```yaml
resource_configuration:
  defaults:
    enabled: true
    port: 8196  # Retrieved from port_registry.sh
    networks: [vrooli-network]
    volumes: [ggwave-data:/data]
    environment:
      GGWAVE_MODE: auto
      GGWAVE_SAMPLE_RATE: 48000
      GGWAVE_ERROR_CORRECTION: true
    
  templates:
    development:
      - description: Dev-optimized with verbose logging
      - overrides:
          GGWAVE_LOG_LEVEL: debug
          GGWAVE_MODE: normal
      
    production:
      - description: Production-optimized for reliability
      - overrides:
          GGWAVE_LOG_LEVEL: info
          GGWAVE_MODE: robust
      
    testing:
      - description: Test-optimized with all modes enabled
      - overrides:
          GGWAVE_LOG_LEVEL: debug
          GGWAVE_TEST_MODE: true
      
customization:
  user_configurable:
    - parameter: mode
      description: Transmission mode (normal/fast/dt/ultrasonic)
      default: auto
      
    - parameter: sample_rate
      description: Audio sample rate in Hz
      default: 48000
      
    - parameter: volume
      description: Output volume (0.0-1.0)
      default: 0.8
      
  environment_variables:
    - var: GGWAVE_MODE
      purpose: Default transmission mode
      
    - var: GGWAVE_PORT
      purpose: API server port
```

### API Contract
```yaml
api_endpoints:
  - method: POST
    path: /api/encode
    purpose: Encode data into audio signal
    input_schema: |
      {
        "data": "string or base64",
        "mode": "normal|fast|dt|ultrasonic",
        "format": "wav|raw|base64"
      }
    output_schema: |
      {
        "audio": "base64 encoded audio",
        "duration_ms": 1000,
        "mode": "normal",
        "bytes": 64
      }
    authentication: none
    rate_limiting: 100/minute
    
  - method: POST
    path: /api/decode
    purpose: Decode data from audio signal
    input_schema: |
      {
        "audio": "base64 encoded audio",
        "mode": "auto|normal|fast|dt|ultrasonic"
      }
    output_schema: |
      {
        "data": "decoded string or base64",
        "confidence": 0.95,
        "mode_detected": "normal",
        "errors_corrected": 0
      }
    authentication: none
    rate_limiting: 100/minute
    
  - method: GET
    path: /health
    purpose: Health check endpoint
    output_schema: |
      {
        "status": "healthy",
        "version": "0.4.0",
        "modes_available": ["normal", "fast", "dt", "ultrasonic"]
      }
```

## 🖥️ Management Interface Contract

### Required Management Actions
```yaml
standard_actions:
  - name: install
    description: Install and configure ggwave
    flags: [--force]
    
  - name: start  
    description: Start the ggwave service
    flags: [--wait]
    
  - name: stop
    description: Stop the service gracefully
    flags: [--force]
    
  - name: status
    description: Show detailed service status
    flags: [--json, --verbose]
    
  - name: uninstall
    description: Remove ggwave and cleanup
    flags: [--keep-data, --force]

resource_specific_actions:
  - name: test-transmission
    description: Test audio transmission between microphone and speaker
    flags: [--mode <mode>, --data <test-string>]
    example: resource-ggwave test-transmission --mode normal --data "Hello World"
    
  - name: benchmark
    description: Benchmark transmission rates and error rates
    flags: [--duration <seconds>, --mode <mode>]
    example: resource-ggwave benchmark --duration 60 --mode all
```

## 🔧 Operational Requirements

### Deployment Standards
```yaml
containerization:
  base_image: ubuntu:22.04
  dockerfile_location: docker/Dockerfile
  build_requirements: Audio libraries, GGWave C++ library
  
networking:
  required_networks:
    - vrooli-network: Primary inter-resource communication
    
  port_allocation:
    - internal: 8196
    - external: 8196
    - protocol: tcp
    - purpose: REST API and WebSocket
    - registry_integration:
        definition: Port 8196 defined in scripts/resources/port_registry.sh
        retrieval: Use port_registry functions
        no_hardcoding: Never hardcode ports
    
data_management:
  persistence:
    - volume: ggwave-data
      mount: /data
      purpose: Session logs and configurations
      
  backup_strategy:
    - method: Volume snapshots
    - frequency: Daily
    - retention: 7 days
    
  migration_support:
    - version_compatibility: All versions
    - upgrade_path: Automatic
    - rollback_support: Yes
```

### Performance Standards
```yaml
resource_requirements:
  minimum:
    cpu: 0.5 cores
    memory: 256MB
    disk: 100MB
    
  recommended:
    cpu: 1 core
    memory: 512MB
    disk: 500MB
    
  scaling:
    horizontal: Multiple instances for parallel sessions
    vertical: More CPU for faster processing
    limits: 10 concurrent sessions per instance
    
monitoring_requirements:
  health_checks:
    - endpoint: /health
    - interval: 30s
    - timeout: 5s
    - failure_threshold: 3
    
  metrics:
    - metric: transmission_success_rate
      collection: API metrics
      alerting: < 95%
      
    - metric: decoding_latency
      collection: API timing
      alerting: > 500ms
```

### Security Standards
```yaml
security_requirements:
  authentication:
    - method: None (air-gapped by design)
    - credential_storage: Not applicable
    - session_management: Ephemeral sessions only
    
  authorization:
    - access_control: Physical proximity required
    - role_based: Not applicable
    - resource_isolation: Process isolation in container
    
  data_protection:
    - encryption_at_rest: Not stored
    - encryption_in_transit: Physical audio signal
    - key_management: Not applicable
    
  network_security:
    - port_exposure: API port only (8196)
    - firewall_requirements: None
    - ssl_tls: Optional for API
    
compliance:
  standards: Air-gap security standards
  auditing: Transmission logs only
  data_retention: No persistent data storage
```

## 🧪 Testing Strategy

### Test Categories
```yaml
unit_tests:
  location: lib/*.bats files
  coverage: Individual function testing
  framework: BATS
  
integration_tests:
  location: test/phases/
  coverage: Audio hardware integration, API endpoints
  test_scenarios: 
    - All transmission modes
    - Error correction validation
    - Range and interference testing
    - API request/response
  
performance_tests:
  load_testing: Concurrent transmissions
  stress_testing: Maximum data rates
  endurance_testing: 24-hour operation
```

## 💰 Infrastructure Value

### Technical Value
- **Security Enhancement**: True air-gap security for sensitive operations
- **Network Independence**: Operates without any network infrastructure
- **Universal Compatibility**: Works with any device with audio I/O
- **Novel Capabilities**: Enables completely new interaction patterns

### Resource Economics
- **Setup Cost**: 30 minutes (minimal dependencies)
- **Operating Cost**: < 256MB RAM, minimal CPU
- **Integration Value**: $15K+ for secure credential scenarios
- **Maintenance Overhead**: Minimal (stable C++ library)

## 🔄 Resource Lifecycle Integration

### Vrooli Integration Standards
```yaml
resource_discovery:
  registry_entry:
    name: ggwave
    category: communication
    capabilities: [air-gap, audio-transmission, error-correction]
    interfaces:
      - cli: resource-ggwave
      - api: http://localhost:8196
      - health: /health
      
  metadata:
    description: Air-gapped data transmission via audio
    version: 0.4.0
    dependencies: []
    enables: [secure-credential-exchange, iot-pairing, offline-payments]

resource_framework_compliance:
  - Standard directory structure
  - CLI framework integration
  - Port registry integration
  - Docker network integration
  - Health monitoring
  - Configuration management
```

## 🧬 Evolution Path

### Version 0.1 (Current - MVP)
- Basic FSK encoding/decoding
- Normal and fast modes
- REST API
- Docker deployment

### Version 0.2 (Planned)
- Ultrasonic mode support
- WebSocket streaming
- Session management
- Performance metrics

### Long-term Vision
- Custom modulation schemes
- Multi-channel transmission
- Audio watermarking
- Hardware acceleration

## 🚨 Risk Mitigation

### Technical Risks
| Risk | Probability | Impact | Mitigation |
|------|------------|--------|------------|
| Audio hardware compatibility | Medium | Medium | Test on multiple platforms |
| Environmental interference | High | Low | Reed-Solomon error correction |
| Limited data rate | High | Medium | Use for small payloads only |
| Range limitations | High | Low | Document 1-5m typical range |

## ✅ Validation Criteria

### Infrastructure Validation
- [ ] Resource installs and starts successfully
- [ ] Audio transmission works in normal mode
- [ ] API endpoints respond correctly
- [ ] Error correction functions properly
- [ ] Health checks pass

### Integration Validation  
- [ ] Can transmit data via sound
- [ ] Integrates with Vrooli framework
- [ ] Port allocation works correctly
- [ ] Docker networking functions
- [ ] Status reporting accurate

## 📝 Implementation Notes

### Design Decisions
**FSK Modulation Choice**: Robust against noise and interference
- Alternative considered: Phase-shift keying
- Decision driver: Better performance in noisy environments
- Trade-offs: Lower data rate but higher reliability

**Docker Deployment**: Ensures audio library compatibility
- Alternative considered: Native installation
- Decision driver: Consistent environment across platforms
- Trade-offs: Requires audio device passthrough

### Known Limitations
- **Data Rate**: Maximum 500 bytes/sec in fastest mode
  - Workaround: Use for small payloads only
  - Future fix: Implement parallel channels
  
- **Range**: 1-5 meters typical indoor range
  - Workaround: Position devices appropriately
  - Future fix: Adaptive volume control

## 🔗 References

### Documentation
- README.md - Quick start and overview
- docs/MODES.md - Transmission mode details
- docs/API.md - Complete API reference
- config/defaults.sh - Configuration options

### Related Resources
- ffmpeg - Audio processing when available
- vault - Secure storage after transfer
- iot-device-configurator - Example integration

### External Resources
- [GGWave GitHub](https://github.com/ggerganov/ggwave)
- [FSK Modulation Theory](https://en.wikipedia.org/wiki/Frequency-shift_keying)
- [Reed-Solomon Codes](https://en.wikipedia.org/wiki/Reed%E2%80%93Solomon_error_correction)

---

**Last Updated**: 2025-01-10  
**Status**: Draft  
**Owner**: Ecosystem Manager  
**Review Cycle**: Monthly