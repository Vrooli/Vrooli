# VirusTotal Threat Intelligence Resource PRD

## Executive Summary
**What**: VirusTotal threat intelligence integration - Multi-engine malware scanning, URL analysis, IP reputation, and threat hunting with 70+ antivirus engines and sandboxes providing comprehensive threat detection
**Why**: Critical for automated security validation, incident response enrichment, and proactive threat detection across all Vrooli scenarios - enabling enterprise-grade security posture
**Who**: All scenarios requiring malware detection, security validation, threat intelligence enrichment, or incident response capabilities - especially those processing external content or requiring compliance
**Value**: Enables $250K+ in scenario value through automated threat detection, incident response automation, and security validation - each scenario gains enterprise threat intelligence worth $5-10K
**Priority**: P0 - Essential security infrastructure for production environments handling external data

## P0 Requirements (Must Have)
- [x] **v2.0 Contract Compliance**: Full implementation of universal.yaml with all lifecycle hooks, health checks, and CLI commands following resource patterns
- [x] **API Integration**: Support both free tier (500 req/day) and premium tier with proper rate limiting and queue management
- [x] **File Scanning**: Submit files for analysis, retrieve reports with detection ratios and detailed engine results
- [x] **URL Analysis**: Analyze URLs for malicious content, phishing detection, and reputation scoring
- [x] **Health Check Endpoint**: Respond to health checks with API connectivity status and remaining quota
- [x] **Report Caching**: Local caching of scan results to minimize API calls and improve response times
- [x] **Secure Configuration**: API keys stored in environment variables with proper secret management

## P1 Requirements (Should Have)
- [x] **Hash Lookups**: Quick file reputation checks using MD5/SHA1/SHA256 without uploading files (GET /api/report/{hash} implemented, retrieves from cache or API)
- [x] **IP/Domain Analysis**: Reputation checks for IPs and domains with associated threat intelligence (GET /api/reputation/ip/{ip} and /api/reputation/domain/{domain} implemented with caching)
- [x] **Batch Processing**: Queue-based processing for multiple files/URLs with rate limit compliance (Full batch scanning implemented with `content execute` command, supports CSV output)
- [x] **Webhook Integration**: Real-time notifications for completed scans via webhook callbacks (Webhook registration and async processing implemented)

## P2 Requirements (Nice to Have)
- [ ] **YARA Rules**: Custom YARA rule creation and deployment for advanced threat hunting (premium)
- [ ] **Historical Analysis**: Track threat evolution and first-seen dates for indicators
- [x] **Export Formats**: Generate reports in multiple formats (JSON, CSV implemented; PDF not needed) for compliance

## Technical Specifications

### Architecture
- **Base Tool**: Official vt-cli wrapped in Docker container for consistency
- **API Version**: v3 (current stable)
- **Service Port**: 8290 (REST API wrapper for local access)
- **Cache Storage**: Redis for result caching (if available) or local SQLite
- **Queue System**: Built-in rate limiter with exponential backoff

### Dependencies
- **Required**: Docker for containerization
- **Optional**: Redis for enhanced caching, PostgreSQL for historical data
- **External**: VirusTotal API (internet connectivity required)

### API Endpoints (Local Wrapper)
```
POST /api/scan/file              # Submit file for scanning
POST /api/scan/url               # Submit URL for analysis
GET  /api/report/{hash}          # Get file report by hash
GET  /api/report/url/{id}        # Get URL analysis report
GET  /api/reputation/ip/{ip}     # IP reputation check
GET  /api/reputation/domain/{d}  # Domain reputation check
GET  /api/health                 # Service health and quota status
GET  /api/stats                  # Usage statistics and cache metrics
```

### CLI Commands
```bash
resource-virustotal manage install       # Install container and dependencies
resource-virustotal manage start         # Start service with API wrapper
resource-virustotal status               # Check health and API quota
resource-virustotal content add          # Submit file/URL for scanning
resource-virustotal content get          # Retrieve scan results
resource-virustotal content list         # List cached results
resource-virustotal test smoke          # Quick connectivity test
```

### Security Considerations

#### API Key Management
- Never hardcode API keys in code or configuration files
- Use environment variable: `VIRUSTOTAL_API_KEY`
- Support key rotation without service restart
- Validate key permissions on startup

#### Rate Limiting Strategy
- Implement token bucket algorithm for rate limiting
- Queue excess requests with priority levels
- Return 429 status with retry-after header
- Monitor quota usage and alert on threshold

#### Data Privacy
- Option to disable sample sharing with community
- Configurable retention period for cached results
- No logging of file contents, only metadata
- Support for air-gapped deployments with offline mode

### Integration Patterns

#### Scenario Security Validation
```bash
# Scan uploaded files in a scenario
resource-virustotal content add \
  --file /path/to/suspicious.exe \
  --wait --threshold 5

# Batch URL scanning for web scraper
resource-virustotal content add \
  --url-list extracted_urls.txt \
  --async --webhook http://localhost:3000/callback
```

#### CI/CD Pipeline Integration
```yaml
# GitLab CI example
security_scan:
  script:
    - resource-virustotal content add --file ./build/app.exe
    - resource-virustotal content get --hash $(sha256sum app.exe)
    - '[ $(jq .positives < report.json) -lt 3 ] || exit 1'
```

#### Incident Response Automation
```bash
# Enrich SIEM alert with threat intelligence
alert_hash=$(extract_hash_from_alert)
resource-virustotal content get --hash $alert_hash --format json | \
  jq '.data.attributes' | \
  send_to_siem
```

### Performance Targets
- **Response Time**: <500ms for cached results, <5s for new submissions
- **Throughput**: Handle 100 req/min with proper queueing
- **Cache Hit Rate**: >80% for common files/URLs
- **Availability**: 99.9% uptime with graceful API degradation

### Error Handling
- **API Errors**: Exponential backoff with jitter
- **Rate Limits**: Queue with priority and retry logic
- **Network Issues**: Circuit breaker pattern
- **Invalid Input**: Clear error messages with remediation steps

## Success Metrics

### Completion Criteria
- [x] Health endpoint responds in <1 second
- [x] File scanning returns results within 30 seconds
- [x] URL analysis completes within 20 seconds
- [x] Rate limiting properly enforces API quotas
- [x] All v2.0 contract commands implemented

### Quality Metrics
- Zero false negatives on EICAR test file
- Successful integration with 3+ scenarios
- Documentation covers all use cases
- Test coverage >80%

### Performance Metrics
- API quota efficiency >95% (no wasted calls)
- Cache effectiveness >80% hit rate
- Queue processing <1 minute average wait
- Memory usage <512MB under load

## Revenue Justification

### Direct Value
- **Security Validation**: $5K per scenario requiring compliance
- **Incident Response**: $10K in automation per SOC integration
- **Threat Intelligence**: $25K annual value for premium features

### Multiplier Effect
- 50+ scenarios need security validation = $250K base value
- Premium tier enables advanced scenarios = $100K additional
- Reduces security incidents by 60% = $500K risk mitigation

### Market Validation
- VirusTotal premium licenses: $10-50K/year
- Comparable platforms: $20-100K/year
- SOC automation tools: $50-200K/year

## Implementation Notes

### Phase 1: Core Implementation (This Task)
1. Create v2.0 compliant structure
2. Implement basic file scanning
3. Add health checks and lifecycle
4. Create CLI wrapper

### Phase 2: Enhanced Features (Future)
1. Add URL and IP analysis
2. Implement result caching
3. Add batch processing
4. Create webhook system

### Phase 3: Premium Features (Future)
1. YARA rule support
2. Historical analysis
3. Advanced threat hunting
4. Enterprise integrations

## Research Findings
- **Similar Work**: OWASP ZAP (web scanning), agent-s2 (security monitoring) - VirusTotal provides unique multi-engine file/URL analysis
- **Template Selected**: Following OWASP ZAP structure with v2.0 contract patterns
- **Unique Value**: Only resource providing 70+ antivirus engine consensus, essential for malware detection
- **External References**: 
  - https://docs.virustotal.com/docs/api-overview
  - https://github.com/VirusTotal/vt-cli
  - https://support.virustotal.com/hc/en-us/articles/115002100149-API-v3
  - https://blog.virustotal.com/2020/08/virustotal-apiv3-introduction.html
  - https://github.com/Gawen/virustotal-python
- **Security Notes**: API key protection critical, implement rate limiting, respect ToS for free tier

## Progress History
- 2025-09-12: Initial PRD creation - 0% → 20% (PRD drafted, research completed)
- 2025-09-12: Scaffolding implementation - 20% → 40% (v2.0 structure, health checks, secure config)
- 2025-09-12: Core implementation - 40% → 100% (All P0 requirements completed, API integration, caching, testing)
- 2025-09-12: Validation and improvement - Verified P0 requirements functional, P1 partially implemented (hash lookups, rate limiting)
- 2025-09-12: P1 Enhancement - All P1 requirements completed (IP/Domain analysis, webhook integration added)
- 2025-01-14: Enhancement iteration - Implemented batch scanning, CSV export format, created PROBLEMS.md documentation
- 2025-09-14: Mock mode and reliability improvements:
  - Added comprehensive mock mode support for testing without API key
  - Fixed integration tests to use proper multipart/form-data for file uploads
  - Implemented URL report retrieval endpoint (/api/report/url/{url_id})
  - Fixed Docker health checks to work in mock mode
  - All tests passing (smoke, unit, integration)
- 2025-09-14: Cache management and integration enhancements:
  - Fixed Docker healthcheck by adding requests module to container
  - Implemented automatic cache rotation (hourly, 100MB/30day/10K entry limits)
  - Added /api/cache/info endpoint for monitoring cache status
  - Added comprehensive integration examples with other Vrooli resources
  - Enhanced documentation with CI/CD and security monitoring examples