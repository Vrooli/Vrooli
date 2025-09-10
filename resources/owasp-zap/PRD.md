# OWASP ZAP Security Scanner Resource PRD

## Executive Summary
**What**: OWASP ZAP (Zed Attack Proxy) - Industry-standard web application security scanner with automated vulnerability detection, active/passive scanning, authentication handling, and comprehensive API testing capabilities
**Why**: Essential for automated security validation of all Vrooli scenarios, ensuring compliance, detecting vulnerabilities early, and maintaining security posture across the ecosystem
**Who**: All scenarios requiring security testing, compliance validation, API security scanning, or continuous security monitoring - especially those handling sensitive data or requiring certifications
**Value**: Enables $200K+ in scenario value through automated security testing, compliance validation, and vulnerability prevention - each scenario gains enterprise-grade security validation worth $2-5K
**Priority**: P0 - Critical security infrastructure for production-ready scenarios

## P0 Requirements (Must Have)
- [ ] **v2.0 Contract Compliance**: Full implementation of universal.yaml with all lifecycle hooks, health checks, and CLI commands
- [ ] **Daemon Mode Operation**: Run ZAP in daemon mode with REST API exposed on port 8180 for programmatic control
- [ ] **Health Check Endpoint**: Respond to health checks within 5 seconds with scanner status and readiness state
- [ ] **Active Scanning**: Execute active security scans against target URLs with configurable attack strength and policies
- [ ] **Passive Scanning**: Continuous passive analysis of all proxied traffic for security issues without active attacks
- [ ] **API Definition Scanning**: Support OpenAPI, SOAP, and GraphQL specifications for comprehensive API security testing
- [ ] **Authentication Support**: Handle form-based, OAuth, SAML, and script-based authentication for protected applications

## P1 Requirements (Should Have)
- [ ] **Custom Scan Policies**: Create and manage custom scan policies with specific rule configurations
- [ ] **Report Generation**: Generate HTML, JSON, and XML security reports with detailed vulnerability information
- [ ] **Spider/Crawler**: Automated application crawling to discover all endpoints and attack surface
- [ ] **WebSocket Scanning**: Security testing for WebSocket connections and real-time communication channels

## P2 Requirements (Nice to Have)
- [ ] **CI/CD Integration Templates**: Pre-built templates for GitLab CI, GitHub Actions, Jenkins integration
- [ ] **Alert Thresholds**: Configurable risk thresholds for failing builds based on severity levels
- [ ] **Baseline Scanning**: Compare scans against security baselines to track improvements/regressions

## Technical Specifications

### Architecture
- **Base Image**: ghcr.io/zaproxy/zaproxy:stable (official Docker image)
- **API Port**: 8180 (REST API for automation)
- **Proxy Port**: 8181 (HTTP/HTTPS proxy for traffic interception)
- **Mode**: Daemon mode for headless operation
- **API Security**: API key authentication (configurable)

### Dependencies
- **Runtime**: Docker for containerized deployment
- **Storage**: Local filesystem for scan results and configurations
- **Network**: Access to target applications for scanning

### API Endpoints
```
GET  /JSON/core/view/version              # ZAP version information
GET  /JSON/core/view/alerts               # Retrieved security alerts
GET  /JSON/spider/action/scan             # Start spider scan
GET  /JSON/ascan/action/scan              # Start active scan
GET  /JSON/ascan/view/status              # Active scan status
GET  /JSON/pscan/view/recordsToScan       # Passive scan queue
POST /JSON/authentication/action/setAuth  # Configure authentication
GET  /JSON/core/other/htmlreport          # Generate HTML report
```

### CLI Commands
```bash
resource-owasp-zap manage install         # Install ZAP container
resource-owasp-zap manage start           # Start ZAP daemon
resource-owasp-zap status                 # Check health and scan status
resource-owasp-zap content add            # Add scan target/policy
resource-owasp-zap content execute        # Run security scan
resource-owasp-zap content list           # List scan results
resource-owasp-zap test smoke            # Quick health validation
```

### Security Scan Types

#### Passive Scanning
- Analyzes HTTP/HTTPS traffic without modifying requests
- Detects: Information disclosure, missing headers, cookies without flags
- Non-intrusive, safe for production environments
- Continuous background analysis

#### Active Scanning
- Actively attacks the application to find vulnerabilities
- Detects: SQL injection, XSS, path traversal, command injection
- Configurable attack strength (Low/Medium/High/Insane)
- Should only run against test environments

#### API Scanning
- Imports OpenAPI/Swagger specifications
- Tests all defined endpoints systematically
- Validates authentication and authorization
- Checks for API-specific vulnerabilities

### Integration Patterns

#### Scenario Security Validation
```bash
# Scan a scenario's API
resource-owasp-zap content execute \
  --target "http://localhost:3100/api/openapi.json" \
  --type api \
  --format openapi

# Scan a web application
resource-owasp-zap content execute \
  --target "http://localhost:3000" \
  --type spider \
  --depth 10
```

#### CI/CD Pipeline Integration
```yaml
security-scan:
  script:
    - vrooli resource owasp-zap manage start --wait
    - vrooli resource owasp-zap content execute --target $APP_URL
    - vrooli resource owasp-zap content get --format json > report.json
    - if [ $(jq '.alerts | map(select(.risk == "High")) | length' report.json) -gt 0 ]; then exit 1; fi
```

### Performance Characteristics
- **Startup Time**: 10-20 seconds for daemon initialization
- **Passive Scan**: <100ms per request overhead
- **Active Scan**: 5-60 minutes depending on application size
- **Spider Time**: 2-30 minutes based on depth and complexity
- **Memory Usage**: 512MB-2GB depending on scan scope
- **CPU Usage**: Low during passive, high during active scans

## Success Metrics

### Completion Metrics
- [ ] All P0 requirements implemented and tested
- [ ] Health checks respond within 5 seconds
- [ ] Can scan APIs via OpenAPI specifications
- [ ] Authentication workflows functional
- [ ] Reports generated in multiple formats

### Quality Metrics
- [ ] Zero false positives in critical alerts
- [ ] Scan completion rate >95%
- [ ] API response time <500ms
- [ ] Resource cleanup on shutdown

### Performance Targets
- [ ] Start/stop within 30 seconds
- [ ] Passive scan overhead <100ms
- [ ] Active scan rate >100 requests/second
- [ ] Support concurrent scans (5+)

## Business Value

### Direct Value Generation
- **Security Compliance**: $50K+ value for compliance-required scenarios
- **Vulnerability Prevention**: $100K+ in prevented security incidents
- **Automated Testing**: $30K+ in manual penetration testing savings
- **CI/CD Security**: $20K+ in DevSecOps automation value

### Ecosystem Amplification
- Every scenario gains enterprise-grade security validation
- Enables security-sensitive scenarios (finance, healthcare, government)
- Provides compliance evidence for certifications (SOC2, ISO27001, PCI-DSS)
- Multiplies trust and adoption of Vrooli scenarios

### Enabled Scenarios
1. **Compliance Validator**: Automated compliance checking against standards
2. **Security Dashboard**: Real-time security posture monitoring
3. **Vulnerability Manager**: Track and remediate security issues
4. **Penetration Testing Service**: Automated security assessments
5. **API Security Gateway**: Validate API security before deployment

## Implementation Plan

### Phase 1: Core Setup (Week 1)
1. Docker container deployment with daemon mode
2. REST API configuration and authentication
3. Basic health check implementation
4. CLI command structure per v2.0 contract

### Phase 2: Scanning Features (Week 2)
1. Passive scanning configuration
2. Active scanning with policies
3. API specification import
4. Authentication method support

### Phase 3: Integration & Reporting (Week 3)
1. Report generation in multiple formats
2. CI/CD integration examples
3. Scenario scanning templates
4. Performance optimization

## Risk Mitigation

### Technical Risks
- **False Positives**: Implement filtering and validation rules
- **Performance Impact**: Use passive scanning for production
- **Resource Usage**: Configure memory limits and timeouts
- **Network Access**: Ensure proper firewall configurations

### Security Risks
- **API Key Management**: Use Vault for secure key storage
- **Scan Data Sensitivity**: Encrypt scan results at rest
- **Target Validation**: Verify authorization before scanning
- **Rate Limiting**: Implement scan throttling controls

## Research Findings

### Similar Work
- **api-manager scenario**: Has basic vulnerability scanning (30% overlap, different focus on API management)
- **judge0 resource**: Code execution security (10% overlap, different domain)
- **vault resource**: Secret management (15% overlap, complementary security)

### Template Selected
Using standard v2.0 resource template with security-specific enhancements from existing patterns

### Unique Value
OWASP ZAP provides industry-standard security scanning that no existing resource offers - it's the missing piece for production-ready security validation

### External References
- https://www.zaproxy.org/docs/docker/ - Official Docker documentation
- https://www.zaproxy.org/docs/api/ - Complete API reference
- https://github.com/zaproxy/zaproxy - Source code and examples
- https://www.zaproxy.org/docs/docker/api-scan/ - API scanning guide
- https://medium.com/@janloo/deploy-zap-zed-attack-proxy-on-server-rest-api-browser-gui-16f31b9b286b - Deployment patterns

### Security Notes
- Never run active scans against production without authorization
- API keys must be rotated regularly
- Scan results may contain sensitive information
- Consider legal implications of security scanning

## Progress History
- 2025-01-10: Initial PRD creation (0% → 0%)