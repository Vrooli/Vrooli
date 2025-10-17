# Network Tools API Documentation

## Base URL
```
http://localhost:15000/api/v1/network
```

## Authentication

The API supports API key authentication. Set the `NETWORK_TOOLS_API_KEY` environment variable and include it in requests:

```bash
# Using X-API-Key header
curl -H "X-API-Key: your-api-key" http://localhost:15000/api/v1/network/health

# Using Bearer token
curl -H "Authorization: Bearer your-api-key" http://localhost:15000/api/v1/network/health
```

In development mode, authentication is not required by default.

## Rate Limiting

- **Default**: 100 requests per minute per IP address
- **Configurable**: Set `RATE_LIMIT` and `RATE_WINDOW` environment variables
- **Response Headers**: `X-RateLimit-Limit`, `X-RateLimit-Remaining`, `X-RateLimit-Reset`

## Endpoints

### Health Check

Check API health status.

**GET** `/health` or `/api/health`

**Response** (200 OK):
```json
{
  "status": "healthy",
  "service": "network-tools",
  "version": "1.0.0",
  "database": "healthy",
  "timestamp": "2025-09-28T20:00:00Z"
}
```

---

### HTTP Client Operations

Perform HTTP requests with full configuration.

**POST** `/api/v1/network/http`

**Request Body**:
```json
{
  "url": "https://api.example.com/resource",
  "method": "GET",
  "headers": {
    "Accept": "application/json",
    "User-Agent": "Network-Tools/1.0"
  },
  "body": {"key": "value"},
  "options": {
    "timeout_ms": 5000,
    "follow_redirects": true,
    "verify_ssl": true,
    "max_retries": 3
  }
}
```

**Response** (200 OK):
```json
{
  "success": true,
  "data": {
    "status_code": 200,
    "headers": {
      "Content-Type": "application/json",
      "Content-Length": "123"
    },
    "body": "{\"result\": \"success\"}",
    "response_time_ms": 234,
    "final_url": "https://api.example.com/resource",
    "ssl_info": {...},
    "redirect_chain": []
  }
}
```

**Error Response** (400 Bad Request):
```json
{
  "success": false,
  "error": "URL must include a scheme (http:// or https://)"
}
```

---

### DNS Operations

Perform DNS queries for various record types.

**POST** `/api/v1/network/dns`

**Request Body**:
```json
{
  "query": "example.com",
  "record_type": "A",
  "dns_server": "8.8.8.8",
  "options": {
    "timeout_ms": 2000,
    "recursive": true,
    "validate_dnssec": false
  }
}
```

**Supported Record Types**: `A`, `AAAA`, `CNAME`, `MX`, `TXT`, `SOA`, `NS`

**Response** (200 OK):
```json
{
  "success": true,
  "data": {
    "query": "example.com",
    "record_type": "A",
    "answers": [
      {
        "name": "example.com",
        "type": "A",
        "ttl": 300,
        "data": "93.184.216.34"
      }
    ],
    "response_time_ms": 45,
    "authoritative": false,
    "dnssec_valid": false
  }
}
```

---

### SSL/TLS Certificate Validation

Validate SSL/TLS certificates with comprehensive analysis.

**POST** `/api/v1/network/ssl/validate`

**Request Body**:
```json
{
  "url": "https://example.com",
  "options": {
    "check_expiry": true,
    "check_chain": true,
    "check_hostname": true,
    "timeout_ms": 10000
  }
}
```

**Response** (200 OK):
```json
{
  "success": true,
  "data": {
    "url": "https://example.com",
    "hostname": "example.com",
    "port": "443",
    "valid": true,
    "issues": [],
    "warnings": [],
    "certificate": {
      "subject": "CN=example.com",
      "issuer": "CN=Let's Encrypt Authority X3",
      "serial": "123456789",
      "not_before": "2025-09-01T00:00:00Z",
      "not_after": "2025-12-01T00:00:00Z",
      "dns_names": ["example.com", "www.example.com"],
      "signature_algorithm": "SHA256-RSA",
      "key_usage": 5
    },
    "chain_length": 3,
    "cipher_suite": "TLS_AES_128_GCM_SHA256",
    "tls_version": "TLS 1.3",
    "days_remaining": 60
  }
}
```

---

### Port Scanning

Scan TCP ports on target hosts.

**POST** `/api/v1/network/scan`

**Request Body**:
```json
{
  "target": "192.168.1.1",
  "ports": [22, 80, 443, 3306],
  "options": {
    "timeout_ms": 1000,
    "aggressive": false,
    "service_detection": true,
    "os_detection": false
  }
}
```

**Response** (200 OK):
```json
{
  "success": true,
  "data": {
    "target": "192.168.1.1",
    "scan_type": "port",
    "results": [
      {
        "port": 22,
        "protocol": "tcp",
        "state": "open",
        "service": "ssh",
        "version": "OpenSSH 8.2",
        "banner": ""
      },
      {
        "port": 80,
        "protocol": "tcp",
        "state": "open",
        "service": "http",
        "version": "nginx 1.18",
        "banner": ""
      }
    ],
    "scan_duration_ms": 1234,
    "total_ports_scanned": 4
  }
}
```

---

### Network Connectivity Testing

Test network connectivity to targets.

**POST** `/api/v1/network/test/connectivity`

**Request Body**:
```json
{
  "target": "8.8.8.8",
  "test_type": "ping",
  "options": {
    "count": 5,
    "timeout_ms": 1000,
    "packet_size": 64,
    "interval_ms": 1000
  }
}
```

**Supported Test Types**: `ping`, `traceroute`

**Response** (200 OK):
```json
{
  "success": true,
  "data": {
    "target": "8.8.8.8",
    "test_type": "ping",
    "statistics": {
      "packets_sent": 5,
      "packets_received": 5,
      "packet_loss_percent": 0.0,
      "min_rtt_ms": 10.5,
      "avg_rtt_ms": 12.3,
      "max_rtt_ms": 15.2,
      "stddev_rtt_ms": 1.8
    },
    "route_hops": []
  }
}
```

---

### API Testing

Test API endpoints with validation.

**POST** `/api/v1/network/api/test`

**Request Body**:
```json
{
  "base_url": "https://api.example.com",
  "test_suite": [
    {
      "endpoint": "/users",
      "method": "GET",
      "test_cases": [
        {
          "name": "List users",
          "input": {},
          "expected_status": 200,
          "expected_schema": {
            "type": "array",
            "items": {
              "type": "object",
              "properties": {
                "id": {"type": "number"},
                "name": {"type": "string"}
              }
            }
          },
          "assertions": [
            {"path": "$.length", "operator": ">", "value": 0}
          ]
        }
      ]
    }
  ]
}
```

**Response** (200 OK):
```json
{
  "success": true,
  "data": {
    "test_results": [
      {
        "endpoint": "/users",
        "total_tests": 1,
        "passed_tests": 1,
        "failed_tests": 0,
        "execution_time_ms": 234,
        "failures": []
      }
    ],
    "overall_success_rate": 100.0
  }
}
```

---

### Network Targets Management

Manage network monitoring targets.

**GET** `/api/v1/network/targets`

Query Parameters:
- `limit`: Maximum number of results (default: 100)
- `offset`: Pagination offset (default: 0)
- `active`: Filter by active status (true/false)

**Response** (200 OK):
```json
{
  "success": true,
  "data": [
    {
      "id": "uuid-123",
      "name": "Production API",
      "target_type": "api",
      "address": "https://api.prod.example.com",
      "port": 443,
      "protocol": "https",
      "is_active": true,
      "created_at": "2025-09-28T10:00:00Z",
      "last_scanned": "2025-09-28T19:00:00Z"
    }
  ]
}
```

**POST** `/api/v1/network/targets`

Create a new network target for monitoring.

**Request Body**:
```json
{
  "name": "New API Endpoint",
  "target_type": "api",
  "address": "https://api.example.com",
  "port": 443,
  "protocol": "https",
  "authentication": {
    "type": "bearer",
    "token": "secret-token"
  },
  "tags": ["production", "critical"],
  "scan_schedule": {
    "interval": "5m",
    "enabled": true
  }
}
```

---

### Monitoring Alerts

View network monitoring alerts.

**GET** `/api/v1/network/alerts`

Query Parameters:
- `severity`: Filter by severity (low, medium, high, critical)
- `status`: Filter by status (new, acknowledged, resolved)
- `limit`: Maximum results (default: 100)

**Response** (200 OK):
```json
{
  "success": true,
  "data": [
    {
      "id": "uuid-456",
      "target_id": "uuid-123",
      "alert_type": "ssl_expiry",
      "severity": "high",
      "message": "SSL certificate expires in 7 days",
      "created_at": "2025-09-28T18:00:00Z",
      "status": "new"
    }
  ]
}
```

## Error Responses

All endpoints use consistent error response format:

```json
{
  "success": false,
  "error": "Detailed error message"
}
```

### Common HTTP Status Codes

- **200 OK**: Request successful
- **400 Bad Request**: Invalid request parameters
- **401 Unauthorized**: Missing or invalid API key
- **403 Forbidden**: Rate limit exceeded
- **404 Not Found**: Endpoint not found
- **500 Internal Server Error**: Server error

## Examples

### cURL Examples

```bash
# Health check
curl http://localhost:15000/api/health

# HTTP request
curl -X POST http://localhost:15000/api/v1/network/http \
  -H "Content-Type: application/json" \
  -d '{"url":"https://httpbin.org/get","method":"GET"}'

# DNS lookup
curl -X POST http://localhost:15000/api/v1/network/dns \
  -H "Content-Type: application/json" \
  -d '{"query":"google.com","record_type":"A"}'

# SSL validation
curl -X POST http://localhost:15000/api/v1/network/ssl/validate \
  -H "Content-Type: application/json" \
  -d '{"url":"https://google.com"}'

# Port scan
curl -X POST http://localhost:15000/api/v1/network/scan \
  -H "Content-Type: application/json" \
  -d '{"target":"localhost","ports":[22,80,443]}'

# Connectivity test
curl -X POST http://localhost:15000/api/v1/network/test/connectivity \
  -H "Content-Type: application/json" \
  -d '{"target":"8.8.8.8","test_type":"ping"}'
```

### JavaScript/Node.js Example

```javascript
const axios = require('axios');

const networkTools = axios.create({
  baseURL: 'http://localhost:15000/api/v1/network',
  headers: {
    'Content-Type': 'application/json',
    'X-API-Key': process.env.NETWORK_TOOLS_API_KEY
  }
});

// Perform DNS lookup
async function dnsLookup(domain) {
  try {
    const response = await networkTools.post('/dns', {
      query: domain,
      record_type: 'A'
    });
    return response.data;
  } catch (error) {
    console.error('DNS lookup failed:', error.response?.data || error.message);
  }
}

// Validate SSL certificate
async function validateSSL(url) {
  try {
    const response = await networkTools.post('/ssl/validate', {
      url: url,
      options: {
        check_expiry: true,
        check_chain: true,
        check_hostname: true
      }
    });
    return response.data;
  } catch (error) {
    console.error('SSL validation failed:', error.response?.data || error.message);
  }
}
```

### Python Example

```python
import requests
import os

class NetworkTools:
    def __init__(self, base_url='http://localhost:15000/api/v1/network'):
        self.base_url = base_url
        self.headers = {
            'Content-Type': 'application/json',
            'X-API-Key': os.environ.get('NETWORK_TOOLS_API_KEY', '')
        }
    
    def http_request(self, url, method='GET', **kwargs):
        """Perform HTTP request"""
        response = requests.post(
            f'{self.base_url}/http',
            json={
                'url': url,
                'method': method,
                **kwargs
            },
            headers=self.headers
        )
        return response.json()
    
    def dns_lookup(self, query, record_type='A'):
        """Perform DNS lookup"""
        response = requests.post(
            f'{self.base_url}/dns',
            json={
                'query': query,
                'record_type': record_type
            },
            headers=self.headers
        )
        return response.json()
    
    def port_scan(self, target, ports):
        """Scan ports on target"""
        response = requests.post(
            f'{self.base_url}/scan',
            json={
                'target': target,
                'ports': ports
            },
            headers=self.headers
        )
        return response.json()

# Usage
tools = NetworkTools()
result = tools.dns_lookup('example.com')
print(result)
```

## Performance Considerations

- **Response Time**: Health checks respond in <100ms
- **Concurrent Requests**: Supports 1000+ concurrent connections
- **Request Timeout**: Default 30 seconds, configurable
- **Rate Limiting**: 100 req/min per IP (configurable)
- **Cache**: DNS results cached for TTL duration
- **Database**: PostgreSQL with connection pooling

## Security Best Practices

1. **Always use HTTPS in production**
2. **Set strong API keys via environment variables**
3. **Implement IP whitelisting for sensitive operations**
4. **Monitor rate limit headers to avoid blocking**
5. **Validate SSL certificates for external APIs**
6. **Use timeout options to prevent hanging requests**

## Troubleshooting

### Common Issues

**Connection Refused**
- Check if the API server is running: `make status`
- Verify the port: Default is 15000, check `API_PORT` env var

**Authentication Failed**
- Ensure API key is set correctly in environment
- Check auth mode: Development vs Production

**Rate Limit Exceeded**
- Monitor `X-RateLimit-Remaining` header
- Implement exponential backoff in clients
- Request rate limit increase for production use

**Timeout Errors**
- Increase timeout_ms in request options
- Check network connectivity to target
- Verify target is responding

## Support

For issues, feature requests, or questions:
- Create an issue in the project repository
- Check logs: `vrooli scenario logs network-tools`
- API health: `curl http://localhost:15000/api/health`