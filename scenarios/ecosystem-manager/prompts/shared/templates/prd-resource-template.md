# PRD: {{RESOURCE_NAME}}

## Executive Summary

### Purpose
{{ONE_SENTENCE_PURPOSE}}

### Problem Statement
{{WHAT_PROBLEM_DOES_THIS_SOLVE}}

### Solution Overview
{{HIGH_LEVEL_SOLUTION}}

### Integration Value
- **Scenarios Enabled**: {{SCENARIOS_THAT_NEED_THIS}}
- **Capabilities Provided**: {{KEY_CAPABILITIES}}
- **Performance Impact**: {{PERFORMANCE_BENEFIT}}

## Success Metrics

### Operational Metrics
- [ ] Uptime: >{{UPTIME_TARGET}}%
- [ ] Response time: <{{RESPONSE_TIME}}ms
- [ ] Resource usage: <{{RESOURCE_LIMIT}}
- [ ] Error rate: <{{ERROR_THRESHOLD}}%

### Adoption Metrics
- [ ] Scenarios using: >{{ADOPTION_TARGET}}
- [ ] API calls/day: >{{API_VOLUME}}
- [ ] Data processed: >{{DATA_VOLUME}}

## Requirements

### P0 Requirements (v2.0 Contract Compliance)
These are mandatory for v2.0 compliance:

- [ ] **REQ-P0-001**: Health check endpoint
  - Endpoint: `/health`
  - Response time: <1s
  - Includes: status, version, uptime
  
- [ ] **REQ-P0-002**: Lifecycle hooks
  - [ ] setup - Initialize resource
  - [ ] develop - Start in dev mode
  - [ ] test - Run tests
  - [ ] stop - Graceful shutdown
  
- [ ] **REQ-P0-003**: CLI integration
  - [ ] status command
  - [ ] health command
  - [ ] logs command
  - [ ] content commands (add/remove/list)
  
- [ ] **REQ-P0-004**: Service configuration
  - [ ] Valid service.json
  - [ ] Port allocation
  - [ ] Environment variables
  
- [ ] **REQ-P0-005**: Error handling
  - [ ] Graceful degradation
  - [ ] Clear error messages
  - [ ] Retry logic with backoff

### P1 Requirements (Enhanced Functionality)
Important for full functionality:

- [ ] **REQ-P1-001**: Advanced health monitoring
  - Dependency checks
  - Performance metrics
  - Resource utilization
  
- [ ] **REQ-P1-002**: Content management
  - Batch operations
  - Import/export
  - Versioning
  
- [ ] **REQ-P1-003**: Integration APIs
  - Webhook support
  - Event streaming
  - Bulk operations

### P2 Requirements (Future Enhancements)
Nice to have features:

- [ ] **REQ-P2-001**: Auto-scaling capabilities
- [ ] **REQ-P2-002**: Multi-tenancy support
- [ ] **REQ-P2-003**: Advanced analytics

## Technical Specifications

### Architecture
```
{{ASCII_ARCHITECTURE_DIAGRAM}}
```

### Technology Stack
- **Core Technology**: {{CORE_TECH}}
- **Language**: {{LANGUAGE}}
- **Framework**: {{FRAMEWORK}}
- **Dependencies**: {{DEPENDENCIES}}

### Resource Allocation
| Metric | Minimum | Recommended | Maximum |
|--------|---------|-------------|---------|
| CPU | {{MIN_CPU}} | {{REC_CPU}} | {{MAX_CPU}} |
| Memory | {{MIN_MEM}} | {{REC_MEM}} | {{MAX_MEM}} |
| Storage | {{MIN_STORAGE}} | {{REC_STORAGE}} | {{MAX_STORAGE}} |
| Network | {{MIN_NET}} | {{REC_NET}} | {{MAX_NET}} |

### Port Configuration
- **Primary Port**: {{PRIMARY_PORT}}
- **Admin Port**: {{ADMIN_PORT}} (optional)
- **Metrics Port**: {{METRICS_PORT}} (optional)

### API Specifications

#### Health Check
```yaml
GET /health
Response:
  status: healthy|degraded|unhealthy
  version: string
  uptime: number (seconds)
  checks:
    - name: string
      status: pass|fail
      message: string
```

#### Core APIs
```yaml
{{API_ENDPOINT_1}}:
  method: {{METHOD}}
  purpose: {{PURPOSE}}
  request: {{REQUEST_SCHEMA}}
  response: {{RESPONSE_SCHEMA}}
  
{{API_ENDPOINT_2}}:
  method: {{METHOD}}
  purpose: {{PURPOSE}}
  request: {{REQUEST_SCHEMA}}
  response: {{RESPONSE_SCHEMA}}
```

### Configuration Schema
```json
{
  "name": "{{RESOURCE_NAME}}",
  "version": "{{VERSION}}",
  "port": {{PORT}},
  "config": {
    "{{CONFIG_KEY_1}}": "{{CONFIG_VALUE_1}}",
    "{{CONFIG_KEY_2}}": "{{CONFIG_VALUE_2}}"
  },
  "environment": {
    "{{ENV_VAR_1}}": "{{DEFAULT_1}}",
    "{{ENV_VAR_2}}": "{{DEFAULT_2}}"
  }
}
```

## Integration Specifications

### Inbound Integrations
Resources/scenarios that will call this:
- **{{CALLER_1}}**: {{HOW_USED_1}}
- **{{CALLER_2}}**: {{HOW_USED_2}}
- **{{CALLER_3}}**: {{HOW_USED_3}}

### Outbound Dependencies
Resources this depends on:
- **{{DEPENDENCY_1}}**: {{WHY_NEEDED_1}}
- **{{DEPENDENCY_2}}**: {{WHY_NEEDED_2}}

### Event System
```yaml
Published Events:
  - {{EVENT_1}}: {{WHEN_TRIGGERED_1}}
  - {{EVENT_2}}: {{WHEN_TRIGGERED_2}}
  
Subscribed Events:
  - {{EVENT_3}}: {{ACTION_TAKEN_3}}
  - {{EVENT_4}}: {{ACTION_TAKEN_4}}
```

## CLI Commands

### Standard Commands (Required)
```bash
# Status check
vrooli resource {{RESOURCE_NAME}} status

# Health check
vrooli resource {{RESOURCE_NAME}} health

# View logs
vrooli resource {{RESOURCE_NAME}} logs [--tail=100] [--follow]

# Lifecycle management
vrooli resource {{RESOURCE_NAME}} setup
vrooli resource {{RESOURCE_NAME}} develop
vrooli resource {{RESOURCE_NAME}} test
vrooli resource {{RESOURCE_NAME}} stop
```

### Content Management (If Applicable)
```bash
# Add content
vrooli resource {{RESOURCE_NAME}} content add <type> <path>

# Remove content
vrooli resource {{RESOURCE_NAME}} content remove <id>

# List content
vrooli resource {{RESOURCE_NAME}} content list [--type=<type>]
```

### Custom Commands
```bash
{{CUSTOM_COMMAND_1}}
{{CUSTOM_COMMAND_2}}
```

## Security Requirements

### Authentication
- [ ] API key support
- [ ] OAuth2 integration (if needed)
- [ ] Service-to-service auth

### Authorization
- [ ] Role-based access
- [ ] Resource-level permissions
- [ ] Rate limiting per client

### Data Security
- [ ] Encryption at rest
- [ ] Encryption in transit (TLS)
- [ ] Secrets management
- [ ] Audit logging

## Performance Requirements

### Latency Targets
- **P50**: <{{P50_LATENCY}}ms
- **P95**: <{{P95_LATENCY}}ms
- **P99**: <{{P99_LATENCY}}ms

### Throughput Targets
- **Requests/second**: >{{RPS_TARGET}}
- **Concurrent connections**: >{{CONCURRENT_TARGET}}
- **Data throughput**: >{{DATA_THROUGHPUT}}

### Resource Limits
- **CPU limit**: {{CPU_LIMIT}}
- **Memory limit**: {{MEMORY_LIMIT}}
- **Disk I/O**: {{DISK_IO_LIMIT}}

## Testing Requirements

### Unit Tests
```bash
# Test structure
tests/
├── unit/
│   ├── {{COMPONENT_1}}_test.{{EXT}}
│   ├── {{COMPONENT_2}}_test.{{EXT}}
│   └── {{COMPONENT_3}}_test.{{EXT}}
```

### Integration Tests
- [ ] Health check validation
- [ ] API endpoint testing
- [ ] CLI command testing
- [ ] Error handling verification

### Performance Tests
- [ ] Load testing (normal traffic)
- [ ] Stress testing (peak traffic)
- [ ] Endurance testing (24h run)
- [ ] Spike testing

## Monitoring & Observability

### Metrics to Track
- **System Metrics**: CPU, memory, disk, network
- **Application Metrics**: Request rate, error rate, latency
- **Business Metrics**: {{BUSINESS_METRICS}}

### Logging Requirements
```yaml
Log Levels:
  - ERROR: System errors, failures
  - WARN: Degraded performance, retries
  - INFO: State changes, important events
  - DEBUG: Detailed operational data

Log Format:
  timestamp: ISO-8601
  level: string
  component: string
  message: string
  context: object
```

### Alerting Thresholds
| Metric | Warning | Critical | Action |
|--------|---------|----------|--------|
| Error rate | >1% | >5% | Page on-call |
| Latency P95 | >500ms | >1s | Investigate |
| CPU usage | >70% | >90% | Scale/optimize |
| Memory usage | >80% | >95% | Restart/scale |

## Deployment Specifications

### Docker Configuration
```dockerfile
FROM {{BASE_IMAGE}}
# Configuration steps
EXPOSE {{PORT}}
HEALTHCHECK --interval=30s --timeout=3s \
  CMD {{HEALTH_CHECK_CMD}}
```

### Environment Variables
```bash
# Required
{{RESOURCE_NAME}}_PORT={{PORT}}
{{RESOURCE_NAME}}_HOST={{HOST}}

# Optional
{{RESOURCE_NAME}}_LOG_LEVEL={{LOG_LEVEL}}
{{RESOURCE_NAME}}_MAX_CONNECTIONS={{MAX_CONN}}
```

### Initialization Process
1. Validate configuration
2. Check dependencies
3. Initialize connections
4. Load required data
5. Start health check endpoint
6. Begin accepting requests

## Documentation Requirements

### README Structure
- [ ] Overview and purpose
- [ ] Quick start guide
- [ ] API documentation
- [ ] CLI command reference
- [ ] Configuration options
- [ ] Troubleshooting guide
- [ ] Contributing guidelines

### Code Documentation
- [ ] Function/method documentation
- [ ] Complex algorithm explanations
- [ ] Configuration examples
- [ ] Integration examples

## Migration Strategy

### From Existing Resource
If replacing existing resource:
1. Feature parity analysis
2. Data migration plan
3. Dual-run period
4. Gradual cutover
5. Rollback plan

### Version Upgrade Path
- **Breaking changes**: {{BREAKING_CHANGES}}
- **Deprecations**: {{DEPRECATIONS}}
- **Migration tools**: {{MIGRATION_TOOLS}}

## Success Criteria

### Launch Criteria
- [ ] All P0 requirements met
- [ ] v2.0 contract compliance verified
- [ ] Health checks passing
- [ ] CLI commands working
- [ ] Documentation complete
- [ ] Tests passing (>80% coverage)
- [ ] Performance targets met

### 30-Day Success Metrics
- [ ] Zero critical incidents
- [ ] <1% error rate
- [ ] >99.9% uptime
- [ ] >3 scenarios integrated
- [ ] <100ms P50 latency

## Risks & Mitigations

### Technical Risks
| Risk | Impact | Probability | Mitigation |
|------|--------|-------------|------------|
| {{RISK_1}} | {{IMPACT_1}} | {{PROB_1}} | {{MITIGATION_1}} |
| {{RISK_2}} | {{IMPACT_2}} | {{PROB_2}} | {{MITIGATION_2}} |

### Operational Risks
| Risk | Impact | Probability | Mitigation |
|------|--------|-------------|------------|
| {{OP_RISK_1}} | {{IMPACT}} | {{PROB}} | {{MITIGATION}} |

## Appendix

### Glossary
- **{{TERM_1}}**: {{DEFINITION_1}}
- **{{TERM_2}}**: {{DEFINITION_2}}

### References
- [{{REFERENCE_1}}]({{URL_1}})
- [{{REFERENCE_2}}]({{URL_2}})

### Similar Resources Analysis
| Resource | Purpose | Pros | Cons | Why This Instead |
|----------|---------|------|------|------------------|
| {{SIMILAR_1}} | {{PURPOSE_1}} | {{PROS_1}} | {{CONS_1}} | {{REASON_1}} |
| {{SIMILAR_2}} | {{PURPOSE_2}} | {{PROS_2}} | {{CONS_2}} | {{REASON_2}} |

---
*PRD Version: 1.0*
*Last Updated: {{DATE}}*
*Author: {{AUTHOR}}*
*Status: {{STATUS}}*
*v2.0 Compliance: {{COMPLIANCE_STATUS}}*