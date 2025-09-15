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
- **Mock Mode**: Test and develop without API key using simulated responses

### New Features (This Update)
- **Batch Processing**: Bulk file/URL submission with rate limiting
- **Export Formats**: CSV and summary report formats for compliance
- **URL Report Retrieval**: Fetch analysis results for previously scanned URLs
- **Mock Mode Support**: Full functionality testing without API key
- **Automatic Cache Rotation**: Prevents unbounded cache growth with configurable limits
- **Enhanced Integration Examples**: Comprehensive examples with other Vrooli resources

### New Features (Latest Update)
- **URL-Based File Submission**: Download and scan files from URLs (S3 pre-signed URLs, etc.)
- **Threat Intelligence Feed Export**: Export aggregated threat data in JSON, CSV, or IOC formats

### Planned
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

- `VIRUSTOTAL_API_KEY`: Your VirusTotal API key (or omit for mock mode)

### Optional Environment Variables

- `VIRUSTOTAL_PORT`: Service port (default: 8290)
- `VIRUSTOTAL_API_TIER`: API tier - "free" or "premium" (default: free)
- `VIRUSTOTAL_CACHE_ENABLED`: Enable result caching (default: true)
- `VIRUSTOTAL_CACHE_TTL`: Cache TTL in seconds (default: 86400)
- `VIRUSTOTAL_WEBHOOK_URL`: Webhook endpoint for notifications
- `VIRUSTOTAL_LOG_LEVEL`: Logging level (default: INFO)

### Mock Mode (Testing Without API Key)

The resource supports full mock mode for testing and development:

```bash
# Start in mock mode (no API key required)
unset VIRUSTOTAL_API_KEY
vrooli resource virustotal manage start

# All endpoints work with simulated responses
curl http://localhost:8290/api/health  # Returns healthy status
curl -X POST http://localhost:8290/api/scan/file -F "file=@test.txt"  # Returns mock scan results
curl -X POST http://localhost:8290/api/scan/url -H "Content-Type: application/json" -d '{"url": "http://example.com"}'
```

Mock mode features:
- All API endpoints return realistic mock data
- EICAR test strings trigger "malicious" detections  
- Rate limiting still enforced for realistic testing
- Perfect for CI/CD pipelines and local development

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

# Get scan results by hash (JSON format)
vrooli resource virustotal content get --hash SHA256_HASH

# Export scan results as CSV
vrooli resource virustotal content get --hash SHA256_HASH --format csv --output report.csv

# Get human-readable summary
vrooli resource virustotal content get --hash SHA256_HASH --format summary

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

# Batch URL scanning
vrooli resource virustotal content execute --batch-file urls.txt --type url

# Batch file scanning with CSV output
vrooli resource virustotal content execute --batch-file files.txt --type file --format csv

# Batch scanning with webhook notifications
vrooli resource virustotal content execute --batch-file urls.txt --type url --webhook http://localhost:3000/callback

# Wait for all batch results
vrooli resource virustotal content execute --batch-file files.txt --type file --wait
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

#### Integration with Other Vrooli Resources

##### With Unstructured-IO (Document Processing)
```bash
# Extract and scan URLs from documents
vrooli resource unstructured-io content add --file document.pdf
vrooli resource unstructured-io content get --id DOC_ID | \
  grep -Eo '(http|https)://[^"]+' | \
  while read url; do
    vrooli resource virustotal content add --url "$url"
  done
```

##### With Web Scrapers (URL Safety)
```bash
# Validate scraped URLs before processing
SCRAPED_URL="http://suspicious-site.com"
RESULT=$(curl -s -X POST http://localhost:8290/api/scan/url \
  -H "Content-Type: application/json" \
  -d "{\"url\": \"$SCRAPED_URL\"}")

# Wait for analysis
sleep 30
REPORT=$(curl -s "http://localhost:8290/api/report/url/$(echo $RESULT | jq -r '.analysis_id')")
MALICIOUS=$(echo $REPORT | jq -r '.data.attributes.stats.malicious')

if [ "$MALICIOUS" -gt 5 ]; then
  echo "URL is malicious, skipping processing"
  exit 1
fi
```

##### With S3/MinIO (File Validation)
```bash
# Scan files before uploading to storage
FILE_PATH="/tmp/upload.zip"
HASH=$(sha256sum "$FILE_PATH" | cut -d' ' -f1)

# Check if already scanned
CACHED=$(curl -s "http://localhost:8290/api/report/$HASH" | jq -r '.from_cache')

if [ "$CACHED" != "true" ]; then
  # New file, scan it
  vrooli resource virustotal content add --file "$FILE_PATH" --wait
fi

# Get results
REPORT=$(vrooli resource virustotal content get --hash "$HASH" --format json)
DETECTIONS=$(echo "$REPORT" | jq -r '.positives')

if [ "$DETECTIONS" -eq 0 ]; then
  # Safe to upload
  vrooli resource minio content add --file "$FILE_PATH"
else
  echo "File detected as malware by $DETECTIONS engines"
fi
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

#### GitHub Actions Integration
```yaml
name: Security Scan
on: [push, pull_request]

jobs:
  virustotal-scan:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v2
      
      - name: Setup Vrooli
        run: |
          ./scripts/manage.sh setup --yes yes
          vrooli resource virustotal manage install
          vrooli resource virustotal manage start --wait
      
      - name: Scan Artifacts
        run: |
          find . -name "*.exe" -o -name "*.dll" | while read file; do
            vrooli resource virustotal content add --file "$file"
          done
      
      - name: Check Results
        run: |
          for file in $(find . -name "*.exe" -o -name "*.dll"); do
            HASH=$(sha256sum "$file" | cut -d' ' -f1)
            RESULT=$(vrooli resource virustotal content get --hash "$HASH" --format json)
            DETECTIONS=$(echo "$RESULT" | jq -r '.positives')
            if [ "$DETECTIONS" -gt 0 ]; then
              echo "::error::File $file detected by $DETECTIONS engines"
              exit 1
            fi
          done
```

#### Incident Response

```bash
# Enrich security alert with threat intelligence
HASH=$(extract_ioc_from_alert)
vrooli resource virustotal content get --hash $HASH --format json | \
  send_to_siem
```

#### Automated Security Monitoring
```bash
#!/bin/bash
# Monitor directory for new files and scan them

WATCH_DIR="/var/uploads"
inotifywait -m -r -e create "$WATCH_DIR" |
while read path action file; do
  if [[ "$file" =~ \.(exe|dll|zip|rar|7z)$ ]]; then
    echo "New file detected: $path$file"
    
    # Scan with VirusTotal
    RESULT=$(vrooli resource virustotal content add --file "$path$file" --wait)
    DETECTIONS=$(echo "$RESULT" | jq -r '.positives')
    
    if [ "$DETECTIONS" -gt 0 ]; then
      # Quarantine the file
      mkdir -p /var/quarantine
      mv "$path$file" /var/quarantine/
      
      # Send alert
      curl -X POST http://localhost:3000/alerts \
        -H "Content-Type: application/json" \
        -d "{\"type\": \"malware\", \"file\": \"$file\", \"detections\": $DETECTIONS}"
    fi
  fi
done
```

#### Scanning Files from S3/MinIO URLs
```bash
# Scan a file from S3 pre-signed URL
curl -X POST http://localhost:8290/api/scan/file-url \
  -H "Content-Type: application/json" \
  -d '{
    "url": "https://s3.amazonaws.com/bucket/file.exe?presigned-url-params",
    "webhook": "http://localhost:3000/scan-complete",
    "max_size": 33554432
  }'

# The endpoint will:
# 1. Download the file from the URL
# 2. Calculate its hash
# 3. Check cache for existing results
# 4. Submit to VirusTotal if not cached
# 5. Call webhook when analysis completes
```

#### Threat Intelligence Feed Export
```bash
# Export threats from last 7 days as JSON
curl "http://localhost:8290/api/threat-feed/export?format=json&days=7&min_detections=5"

# Export as CSV for SIEM import
curl "http://localhost:8290/api/threat-feed/export?format=csv&days=30" \
  -o threat_feed.csv

# Export as IOC (Indicators of Compromise) format
curl "http://localhost:8290/api/threat-feed/export?format=ioc&days=14" \
  -o indicators.txt

# Filter by threat type
curl "http://localhost:8290/api/threat-feed/export?format=json&types=files&types=urls"

# Integration with SIEM
*/15 * * * * curl -s "http://localhost:8290/api/threat-feed/export?format=json&days=1" | \
  /usr/local/bin/send-to-siem.sh
```

## API Endpoints

The service exposes a REST API on port 8290:

- `GET /api/health` - Service health status
- `POST /api/scan/file` - Submit file for scanning (supports webhook param)
- `POST /api/scan/file-url` - Submit file from URL for scanning (NEW)
- `POST /api/scan/url` - Submit URL for analysis (supports webhook param)
- `GET /api/report/{hash}` - Get file report by hash
- `GET /api/report/url/{id}` - Get URL analysis report
- `GET /api/reputation/ip/{ip}` - IP reputation check
- `GET /api/reputation/domain/{domain}` - Domain reputation check
- `GET /api/stats` - Usage statistics and quota
- `GET /api/webhooks` - List webhook registrations
- `GET /api/cache/list` - List cached scan results
- `DELETE /api/cache/clear` - Clear all cached data
- `GET /api/cache/info` - Cache rotation and size information
- `GET /api/threat-feed/export` - Export threat intelligence feed (NEW)

## Cache Management

The resource includes automatic cache rotation to prevent unbounded growth:

### Automatic Rotation
- **Hourly rotation**: Cache is automatically pruned every hour
- **Size limit**: 100MB maximum database size
- **Age limit**: Entries older than 30 days are removed
- **Entry limit**: Maximum 10,000 entries per table

### Manual Management
```bash
# View cache information
curl http://localhost:8290/api/cache/info

# List cached entries
curl http://localhost:8290/api/cache/list

# Clear all cache
curl -X DELETE http://localhost:8290/api/cache/clear
```

### Cache Benefits
- Reduces API quota usage by 80%+
- Instant response for previously scanned files
- Preserves results during API outages
- Shared across all scenario integrations

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