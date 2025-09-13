# VirusTotal Resource

Multi-engine threat intelligence and malware scanning service integration for Vrooli.

## Overview

The VirusTotal resource provides comprehensive threat detection capabilities by leveraging 70+ antivirus engines and sandboxes. It enables automated security validation, incident response enrichment, and proactive threat detection across all Vrooli scenarios.

## Features

### Implemented
- **File Scanning**: Submit files for multi-engine malware analysis
- **URL Analysis**: Analyze URLs for malicious content and phishing
- **Hash Lookups**: Quick reputation checks without file upload
- **IP/Domain Intelligence**: Reputation and threat intelligence for network indicators
- **Webhook Support**: Real-time notifications for scan completions
- **Rate Limiting**: Intelligent queue management for API quotas
- **Result Caching**: Local SQLite caching to minimize API calls
- **Health Monitoring**: Service health and quota tracking

### Planned
- **Batch Processing**: Bulk file/URL submission
- **YARA Rules**: Custom threat hunting rules (premium feature)
- **Historical Analysis**: Track threat evolution over time

## Installation

```bash
# Install the resource
vrooli resource virustotal manage install

# Set your API key
export VIRUSTOTAL_API_KEY="your-api-key-here"

# Start the service
vrooli resource virustotal manage start --wait
```

## Configuration

### Required Environment Variables

- `VIRUSTOTAL_API_KEY`: Your VirusTotal API key (required)

### Optional Environment Variables

- `VIRUSTOTAL_PORT`: Service port (default: 8290)
- `VIRUSTOTAL_API_TIER`: API tier - "free" or "premium" (default: free)
- `VIRUSTOTAL_CACHE_ENABLED`: Enable result caching (default: true)
- `VIRUSTOTAL_CACHE_TTL`: Cache TTL in seconds (default: 86400)
- `VIRUSTOTAL_WEBHOOK_URL`: Webhook endpoint for notifications
- `VIRUSTOTAL_LOG_LEVEL`: Logging level (default: INFO)

### API Limits

**Free Tier:**
- 500 requests per day
- 4 requests per minute
- 32MB maximum file size
- No commercial use

**Premium Tier:**
- Unlimited requests
- 650MB maximum file size
- Commercial use allowed
- Advanced features (YARA, hunting, etc.)

## Usage

### Basic Commands

```bash
# Check service status
vrooli resource virustotal status

# View service logs
vrooli resource virustotal logs

# Run health checks
vrooli resource virustotal test smoke

# Display API configuration
vrooli resource virustotal credentials
```

### Scanning Files

```bash
# Submit a file for scanning
vrooli resource virustotal content add --file /path/to/suspicious.exe

# Get scan results by hash
vrooli resource virustotal content get --hash SHA256_HASH

# List cached results
vrooli resource virustotal content list
```

### Scanning URLs

```bash
# Submit URL for analysis
vrooli resource virustotal content add --url http://suspicious-site.com

# Submit with webhook for completion notification
curl -X POST http://localhost:8290/api/scan/url \
  -H "Content-Type: application/json" \
  -d '{"url": "http://suspicious-site.com", "webhook": "http://localhost:3000/webhook"}'

# Batch URL scanning (planned)
vrooli resource virustotal content execute --url-list urls.txt
```

### IP and Domain Reputation

```bash
# Check IP reputation
curl http://localhost:8290/api/reputation/ip/8.8.8.8

# Check domain reputation
curl http://localhost:8290/api/reputation/domain/example.com

# Results are cached automatically to minimize API calls
```

### Integration Examples

#### Scenario Integration

```bash
# In your scenario code, scan uploaded files
curl -X POST http://localhost:8290/api/scan/file \
  -F "file=@uploaded.exe"

# Check scan results
curl http://localhost:8290/api/report/FILE_HASH
```

#### CI/CD Pipeline

```yaml
# GitLab CI example
security_scan:
  script:
    - vrooli resource virustotal content add --file ./build/app
    - vrooli resource virustotal content get --hash $(sha256sum app)
    - test $(jq .engines_detected < report.json) -lt 3
```

#### Incident Response

```bash
# Enrich security alert with threat intelligence
HASH=$(extract_ioc_from_alert)
vrooli resource virustotal content get --hash $HASH --format json | \
  send_to_siem
```

## API Endpoints

The service exposes a REST API on port 8290:

- `GET /api/health` - Service health status
- `POST /api/scan/file` - Submit file for scanning (supports webhook param)
- `POST /api/scan/url` - Submit URL for analysis (supports webhook param)
- `GET /api/report/{hash}` - Get file report by hash
- `GET /api/report/url/{id}` - Get URL analysis report
- `GET /api/reputation/ip/{ip}` - IP reputation check
- `GET /api/reputation/domain/{domain}` - Domain reputation check
- `GET /api/stats` - Usage statistics and quota
- `GET /api/webhooks` - List webhook registrations
- `GET /api/cache/list` - List cached scan results
- `DELETE /api/cache/clear` - Clear all cached data

## Testing

```bash
# Run all tests
vrooli resource virustotal test all

# Run specific test suites
vrooli resource virustotal test smoke       # Quick health check (<30s)
vrooli resource virustotal test integration # Full functionality (<120s)
vrooli resource virustotal test unit        # Library functions (<60s)
```

## Troubleshooting

### Service won't start

1. Check API key is set: `echo $VIRUSTOTAL_API_KEY`
2. Verify port is available: `lsof -i :8290`
3. Check Docker is running: `docker ps`
4. Review logs: `vrooli resource virustotal logs`

### Rate limiting errors

- Free tier is limited to 4 requests/minute
- The service automatically queues excess requests
- Consider upgrading to premium for higher limits

### API key issues

```bash
# Verify API key format (should be 64 characters)
echo $VIRUSTOTAL_API_KEY | wc -c

# Test API key validity
curl -H "x-apikey: $VIRUSTOTAL_API_KEY" \
  https://www.virustotal.com/api/v3/files/test
```

## Security Considerations

- **API Key Protection**: Never commit API keys to version control
- **Sample Sharing**: Disabled by default to protect privacy
- **Rate Limiting**: Enforced to prevent quota exhaustion
- **Caching**: Results cached locally to minimize API exposure
- **HTTPS Only**: All API communications use TLS

## Integration with Other Resources

Works well with:
- `owasp-zap`: Comprehensive web application security testing
- `vault`: Secure API key storage
- `postgres`: Historical threat data storage
- `redis`: Enhanced result caching

## Support

- Get API key: https://www.virustotal.com/gui/join-us
- API Documentation: https://docs.virustotal.com/docs/api-overview
- Premium Features: https://www.virustotal.com/gui/contact-us

## License

This resource integrates with VirusTotal's API. Usage is subject to VirusTotal's Terms of Service. Free tier usage is limited to non-commercial purposes only.