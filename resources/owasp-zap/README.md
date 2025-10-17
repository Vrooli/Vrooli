# OWASP ZAP Security Scanner Resource

OWASP ZAP (Zed Attack Proxy) provides automated web application security scanning for Vrooli scenarios, enabling vulnerability detection, compliance validation, and continuous security monitoring.

## Quick Start

```bash
# Install OWASP ZAP
vrooli resource owasp-zap manage install

# Start the scanner
vrooli resource owasp-zap manage start

# Check health
vrooli resource owasp-zap status

# Run a basic scan
vrooli resource owasp-zap content execute --target http://localhost:3000
```

## Features

- **Active Scanning**: Actively test for vulnerabilities like SQL injection, XSS
- **Passive Scanning**: Analyze traffic for security issues without attacks
- **API Security**: Test OpenAPI, SOAP, and GraphQL APIs
- **Authentication**: Support for various authentication methods
- **Reporting**: Generate detailed security reports

## Usage Examples

### Scan a Web Application
```bash
# Spider and scan a web app
vrooli resource owasp-zap content execute \
  --target http://localhost:3000 \
  --type spider \
  --depth 5
```

### Scan an API
```bash
# Import and scan OpenAPI specification
vrooli resource owasp-zap content execute \
  --target http://localhost:3100/api/openapi.json \
  --type api \
  --format openapi
```

### Generate Reports
```bash
# Get scan results as JSON
vrooli resource owasp-zap content list --format json

# Generate HTML report
vrooli resource owasp-zap content get --format html > report.html
```

## Configuration

Default configuration in `config/defaults.sh`:
- API Port: 8180
- Proxy Port: 8181
- Scan Timeout: 3600 seconds
- Memory Limit: 1GB

## Security Notes

- Only run active scans against applications you own
- API key required for remote access (auto-generated)
- Scan results may contain sensitive information
- Use passive scanning for production environments

## Integration

Works with:
- **api-manager**: Validate API security
- **vault**: Secure API key storage
- **scenario-authenticator**: Test authentication flows
- **All scenarios**: Security validation and compliance

## Troubleshooting

If ZAP fails to start:
```bash
# Check logs
vrooli resource owasp-zap logs

# Verify port availability
netstat -tlnp | grep 8180

# Restart the service
vrooli resource owasp-zap manage restart
```

For more information, see the [PRD.md](PRD.md) file.