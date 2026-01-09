# Synchronous Test Execution - Quick Reference

## 🎯 One-Liner

```bash
curl -X POST http://localhost:17480/api/v1/test-suite/{suite_id}/execute-sync \
  -H "Content-Type: application/json" -d '{}' | jq '.summary'
```

## 📋 Common Commands

### Find Suite ID
```bash
# By scenario name
curl -s http://localhost:17480/api/v1/test-suites | \
  jq -r '.test_suites[] | select(.scenario_name == "my-scenario") | .id' | head -1

# List all
curl -s http://localhost:17480/api/v1/test-suites | \
  jq -r '.test_suites[] | "\(.id) - \(.scenario_name)"'
```

### Execute Tests
```bash
# Basic (uses defaults)
curl -X POST http://localhost:17480/api/v1/test-suite/{suite_id}/execute-sync \
  -H "Content-Type: application/json" -d '{}'

# With options
curl -X POST http://localhost:17480/api/v1/test-suite/{suite_id}/execute-sync \
  -H "Content-Type: application/json" \
  -d '{"execution_type": "full", "environment": "local", "timeout_seconds": 600}'
```

### Parse Results
```bash
# Summary only
... | jq '.summary'

# Check if passed
... | jq -r '.summary.failed == 0'

# Get failed tests
... | jq '.failed_tests[]'

# Recommendations
... | jq -r '.recommendations[]'
```

## 🔑 Response Fields

| Field | Type | Description |
|-------|------|-------------|
| `execution_id` | UUID | Unique execution identifier |
| `suite_name` | string | Scenario name |
| `status` | string | "completed", "failed", "running" |
| `summary.total_tests` | number | Total test count |
| `summary.passed` | number | Passed test count |
| `summary.failed` | number | Failed test count |
| `summary.duration` | number | Execution time (seconds) |
| `summary.coverage` | number | Code coverage % |
| `failed_tests` | array | Failed test details |
| `recommendations` | array | Improvement suggestions |

## 🚦 Status Codes

| Code | Meaning |
|------|---------|
| 200 | Success - tests completed |
| 400 | Bad request (invalid UUID) |
| 404 | Suite not found |
| 500 | Server error |

## 💡 Quick Scripts

### CI/CD Integration
```bash
#!/bin/bash
response=$(curl -s -X POST "http://localhost:17480/api/v1/test-suite/$1/execute-sync" \
  -H "Content-Type: application/json" -d '{}')

failed=$(echo "$response" | jq -r '.summary.failed')
[ "$failed" -eq 0 ] && echo "✅ Pass" && exit 0 || echo "❌ Fail" && exit 1
```

### Get Test Results as JSON
```bash
curl -s -X POST "http://localhost:17480/api/v1/test-suite/{suite_id}/execute-sync" \
  -H "Content-Type: application/json" -d '{}' > test-results.json
```

### Human-Readable Report
```bash
response=$(curl -s -X POST ".../execute-sync" -d '{}')
echo "Tests: $(echo "$response" | jq -r '.summary.total_tests')"
echo "Passed: $(echo "$response" | jq -r '.summary.passed')"
echo "Failed: $(echo "$response" | jq -r '.summary.failed')"
echo "Duration: $(echo "$response" | jq -r '.summary.duration')s"
```

## ⚖️ Async vs Sync

| | Async | Sync |
|-|-------|------|
| **Endpoint** | `/execute` | `/execute-sync` |
| **HTTP Status** | 202 Accepted | 200 OK |
| **Response** | execution_id + tracking_url | Complete results |
| **Polling** | Required | Not needed |
| **Use Case** | Long tests, UI | Quick tests, automation |

## 🎓 Examples by Language

### Bash
```bash
curl -X POST ".../execute-sync" -d '{}' | jq '.summary.failed'
```

### Python
```python
import requests
r = requests.post(f"{url}/execute-sync", json={})
print(r.json()['summary'])
```

### Node.js
```javascript
const response = await fetch(`${url}/execute-sync`, {
  method: 'POST',
  headers: {'Content-Type': 'application/json'},
  body: '{}'
});
const results = await response.json();
console.log(results.summary);
```

### Go
```go
resp, _ := http.Post(url+"/execute-sync", "application/json", strings.NewReader("{}"))
var results TestExecutionResultsResponse
json.NewDecoder(resp.Body).Decode(&results)
fmt.Println(results.Summary)
```

## 🔧 Port Discovery

```bash
# Get test-genie API port
API_PORT=$(vrooli status --scenarios --verbose | grep "test-genie" -A 1 | grep -oP 'localhost:\K[0-9]+')
curl -X POST "http://localhost:${API_PORT}/api/v1/test-suite/{suite_id}/execute-sync" -d '{}'
```

## 📚 More Info

- Full Guide: `/scenarios/test-genie/docs/synchronous-execution-guide.md`
- API Reference: `/scenarios/test-genie/PRD.md`
- Implementation: `/scenarios/test-genie/api/main.go:2597`
