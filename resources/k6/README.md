# K6 Load Testing Resource

Modern load testing tool with JavaScript scripting for performance testing and synthetic monitoring at scale.

## Overview

K6 is a developer-centric, open-source load testing tool built for modern DevOps teams. It provides:
- **JavaScript/ES6 scripting** for test scenarios
- **Protocol support** for HTTP/1.1, HTTP/2, WebSocket, and gRPC
- **Performance metrics** with built-in statistics
- **CI/CD integration** for automated performance testing
- **Cloud execution** option via Grafana Cloud

## Quick Start

```bash
# Install K6
vrooli resource k6 install

# Check status
vrooli resource k6 status

# Run a basic test
resource-k6 run-test basic-http.js

# Run with custom settings
resource-k6 run-test api-load.js --vus 100 --duration 5m
```

## Features

### Test Scripting
- Write tests in JavaScript/ES6
- Access to k6 API for checks, thresholds, and custom metrics
- Support for test data and environment variables
- Modular test organization with imports

### Performance Testing Types
- **Load Testing**: Verify system performance under expected load
- **Stress Testing**: Find breaking points and recovery behavior
- **Spike Testing**: Test sudden traffic increases
- **Soak Testing**: Long-duration tests for memory leaks

### Metrics & Analysis
- Response times (min, max, avg, percentiles)
- Request rates and throughput
- Error rates and status codes
- Custom business metrics

## Usage Examples

### Basic HTTP Test
```javascript
import http from 'k6/http';
import { check, sleep } from 'k6';

export const options = {
  vus: 10,        // 10 virtual users
  duration: '30s', // 30 second test
};

export default function () {
  const res = http.get('https://test.k6.io');
  check(res, {
    'status is 200': (r) => r.status === 200,
    'response time < 500ms': (r) => r.timings.duration < 500,
  });
  sleep(1);
}
```

### API Load Test with Stages
```javascript
export const options = {
  stages: [
    { duration: '30s', target: 20 },  // Ramp up
    { duration: '1m', target: 20 },   // Stay at 20 users
    { duration: '30s', target: 0 },   // Ramp down
  ],
  thresholds: {
    http_req_duration: ['p(95)<500'], // 95% of requests under 500ms
    http_req_failed: ['rate<0.1'],    // Error rate under 10%
  },
};
```

### Data-Driven Testing
```javascript
import { SharedArray } from 'k6/data';

const users = new SharedArray('users', function () {
  return JSON.parse(open('./users.json'));
});

export default function () {
  const user = users[Math.floor(Math.random() * users.length)];
  // Use user data in test
}
```

## Integration

### With Vrooli Scenarios
```json
{
  "resources": ["k6"],
  "data": [
    {
      "type": "k6",
      "scripts": [
        {
          "name": "api-performance.js",
          "content": "// K6 test script content"
        }
      ]
    }
  ]
}
```

### CI/CD Pipeline
```yaml
# GitHub Actions example
- name: Run K6 tests
  run: |
    vrooli resource k6 run-test smoke-test.js
    vrooli resource k6 run-test load-test.js --vus 50
```

### Grafana Cloud Integration
```bash
# Set up Grafana Cloud credentials
export K6_GRAFANA_CLOUD_TOKEN="your-token"
export K6_GRAFANA_CLOUD_PROJECT_ID="your-project"

# Run test with cloud output
k6 run --out cloud script.js
```

## Commands

| Command | Description |
|---------|-------------|
| `status` | Show K6 installation and test status |
| `install` | Install K6 load testing tool |
| `inject` | Inject test scripts into K6 |
| `run-test` | Execute a K6 test script |
| `list-tests` | List available test scripts |
| `credentials` | Show K6 configuration |

## Configuration

### Environment Variables
- `K6_PORT`: API port (default: 6565)
- `K6_DATA_DIR`: Data directory (default: ~/.k6)
- `K6_DEFAULT_VUS`: Default virtual users (default: 10)
- `K6_DEFAULT_DURATION`: Default test duration (default: 30s)

### Test Options
```javascript
export const options = {
  vus: 100,                    // Virtual users
  duration: '5m',               // Test duration
  iterations: 1000,             // Total iterations
  rps: 100,                     // Requests per second limit
  
  thresholds: {
    http_req_duration: ['p(95)<500'],
    http_req_failed: ['rate<0.1'],
  },
  
  tags: {
    environment: 'staging',
    team: 'platform',
  },
};
```

## Best Practices

### Test Design
1. **Start small**: Begin with smoke tests before load tests
2. **Use stages**: Gradually ramp up load
3. **Set thresholds**: Define success criteria
4. **Monitor resources**: Watch system metrics during tests

### Performance Targets
- **Response time**: p95 < 500ms for APIs
- **Error rate**: < 1% for production
- **Throughput**: Match expected traffic patterns
- **Availability**: > 99.9% uptime target

### Test Data Management
- Use SharedArray for large datasets
- Parameterize test data
- Avoid hardcoded values
- Clean up test data after runs

## Troubleshooting

### Common Issues

**High memory usage**
```bash
# Limit VUs or use SharedArray for data
k6 run script.js --vus 50
```

**Connection errors**
```javascript
// Add retry logic
import { Retry } from 'k6/http';
const retry = new Retry({ attempts: 3 });
```

**Metric aggregation**
```bash
# Output detailed metrics
k6 run --out json=results.json script.js
```

## Architecture

```
K6 Load Testing
├── Test Scripts (JavaScript)
│   ├── Scenarios
│   ├── Checks
│   └── Thresholds
├── Execution Engine
│   ├── VU Scheduler
│   ├── HTTP Client
│   └── Metrics Collector
└── Output
    ├── Console Summary
    ├── JSON Results
    └── Cloud Dashboard
```

## See Also
- [K6 Documentation](https://k6.io/docs/)
- [JavaScript API](https://k6.io/docs/javascript-api/)
- [Grafana Cloud](https://grafana.com/products/cloud/)