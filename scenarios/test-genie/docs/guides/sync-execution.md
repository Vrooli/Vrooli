# Synchronous Test Execution Guide

## Overview

The synchronous execution endpoint allows you to run test suites and get complete results in a **single request** - no polling required. This is perfect for coding agents, CI/CD scripts, and automated workflows.

## Endpoint

```
POST /api/v1/test-suite/:suite_id/execute-sync
```

## When to Use

### ✅ Use Sync Endpoint When:
- Running quick test suites (< 5 minutes)
- You need immediate results in one request
- Writing automation scripts or agent workflows
- Running tests in CI/CD pipelines
- You want simpler error handling

### ❌ Use Async Endpoint Instead When:
- Test suites take > 5-10 minutes
- Building interactive UIs that show progress
- You need to check status mid-execution
- Running very long-running performance tests

## Basic Usage

### Step 1: Find Your Test Suite ID

```bash
# List all test suites
curl http://localhost:17480/api/v1/test-suites | jq '.test_suites[] | {id, scenario_name, suite_type}'

# Example output:
# {
#   "id": "41453489-79b8-4876-99bc-2941b0c44294",
#   "scenario_name": "my-scenario",
#   "suite_type": "unit"
# }
```

### Step 2: Execute Synchronously

```bash
# Simple execution (uses defaults)
curl -X POST http://localhost:17480/api/v1/test-suite/{suite_id}/execute-sync \
  -H "Content-Type: application/json" \
  -d '{}'

# Full configuration
curl -X POST http://localhost:17480/api/v1/test-suite/{suite_id}/execute-sync \
  -H "Content-Type: application/json" \
  -d '{
    "execution_type": "full",
    "environment": "local",
    "timeout_seconds": 600
  }'
```

### Step 3: Parse Results

```bash
# Get just the summary
curl -X POST http://localhost:17480/api/v1/test-suite/{suite_id}/execute-sync \
  -H "Content-Type: application/json" \
  -d '{}' | jq '.summary'

# Output:
# {
#   "total_tests": 15,
#   "passed": 13,
#   "failed": 2,
#   "skipped": 0,
#   "duration": 45.3,
#   "coverage": 86.7
# }
```

## Request Parameters

| Parameter | Type | Required | Default | Description |
|-----------|------|----------|---------|-------------|
| `execution_type` | string | No | `"full"` | Type of execution: `"full"`, `"smoke"`, `"regression"`, `"performance"` |
| `environment` | string | No | `"local"` | Target environment: `"local"`, `"staging"`, `"production"` |
| `timeout_seconds` | number | No | `600` | Maximum execution time in seconds (10 minutes default) |
| `parallel_execution` | boolean | No | `false` | Enable parallel test execution |
| `notification_settings` | object | No | `{}` | Notification configuration (for webhooks) |

## Response Format

### Success Response (200 OK)

```json
{
  "execution_id": "75084688-cd77-47e1-90d6-3afaad94abd1",
  "suite_name": "my-scenario",
  "status": "completed",
  "summary": {
    "total_tests": 15,
    "passed": 13,
    "failed": 2,
    "skipped": 0,
    "duration": 45.3,
    "coverage": 86.7
  },
  "failed_tests": [
    {
      "id": "test-uuid",
      "test_case_id": "case-uuid",
      "status": "failed",
      "duration": 2.5,
      "error_message": "Assertion failed: expected 200, got 500",
      "stack_trace": "...",
      "test_case_name": "API endpoint test",
      "test_case_description": "Verify endpoint returns 200"
    }
  ],
  "performance_metrics": {
    "execution_time": 45.3,
    "resource_usage": {
      "tests_per_second": 0.33
    },
    "error_count": 2
  },
  "recommendations": [
    "Review and fix 2 failed test(s)",
    "Consider adding more test cases to improve coverage"
  ]
}
```

### Error Responses

#### 400 Bad Request
```json
{
  "error": "Invalid suite ID"
}
```

#### 404 Not Found
```json
{
  "error": "Test suite not found",
  "details": "test suite not found in repository"
}
```

#### 500 Internal Server Error
```json
{
  "error": "Failed to create test execution",
  "details": "database connection error"
}
```

## Practical Examples

### Example 1: Check if Tests Pass (Exit Code)

```bash
#!/bin/bash
# ci-test.sh - Run tests and exit with proper code

SUITE_ID="41453489-79b8-4876-99bc-2941b0c44294"

response=$(curl -s -X POST "http://localhost:17480/api/v1/test-suite/${SUITE_ID}/execute-sync" \
  -H "Content-Type: application/json" \
  -d '{"execution_type": "full"}')

# Check if any tests failed
failed=$(echo "$response" | jq -r '.summary.failed')

if [ "$failed" -gt 0 ]; then
  echo "❌ Tests failed! $failed test(s) failed"
  echo "$response" | jq '.failed_tests'
  exit 1
else
  echo "✅ All tests passed!"
  exit 0
fi
```

### Example 2: Generate Test Report

```bash
#!/bin/bash
# generate-report.sh

SUITE_ID="$1"

response=$(curl -s -X POST "http://localhost:17480/api/v1/test-suite/${SUITE_ID}/execute-sync" \
  -H "Content-Type: application/json" \
  -d '{"execution_type": "full"}')

echo "# Test Execution Report"
echo ""
echo "**Suite**: $(echo "$response" | jq -r '.suite_name')"
echo "**Status**: $(echo "$response" | jq -r '.status')"
echo "**Duration**: $(echo "$response" | jq -r '.summary.duration')s"
echo ""
echo "## Summary"
echo "- Total: $(echo "$response" | jq -r '.summary.total_tests')"
echo "- Passed: $(echo "$response" | jq -r '.summary.passed')"
echo "- Failed: $(echo "$response" | jq -r '.summary.failed')"
echo "- Coverage: $(echo "$response" | jq -r '.summary.coverage')%"
echo ""
echo "## Recommendations"
echo "$response" | jq -r '.recommendations[]' | sed 's/^/- /'
```

### Example 3: Coding Agent Usage (Python)

```python
import requests
import sys

def run_tests_sync(suite_id: str, api_url: str = "http://localhost:17480") -> dict:
    """Run test suite synchronously and return results."""

    response = requests.post(
        f"{api_url}/api/v1/test-suite/{suite_id}/execute-sync",
        json={
            "execution_type": "full",
            "environment": "local",
            "timeout_seconds": 600
        }
    )

    if response.status_code == 400:
        raise ValueError(f"Invalid suite ID: {response.json()}")
    elif response.status_code == 404:
        raise ValueError(f"Suite not found: {response.json()}")
    elif response.status_code != 200:
        raise RuntimeError(f"Test execution failed: {response.json()}")

    return response.json()

def main():
    suite_id = sys.argv[1] if len(sys.argv) > 1 else "41453489-79b8-4876-99bc-2941b0c44294"

    print(f"🧪 Running tests for suite {suite_id}...")

    results = run_tests_sync(suite_id)

    # Print summary
    summary = results['summary']
    print(f"\n📊 Test Results:")
    print(f"   Total: {summary['total_tests']}")
    print(f"   Passed: {summary['passed']} ✅")
    print(f"   Failed: {summary['failed']} ❌")
    print(f"   Duration: {summary['duration']:.2f}s")
    print(f"   Coverage: {summary['coverage']:.1f}%")

    # Print failed tests if any
    if summary['failed'] > 0:
        print(f"\n❌ Failed Tests:")
        for test in results['failed_tests']:
            print(f"   - {test['test_case_name']}: {test['error_message']}")
        sys.exit(1)
    else:
        print("\n✅ All tests passed!")
        sys.exit(0)

if __name__ == "__main__":
    main()
```

### Example 4: Claude Code Agent Usage

```bash
# As a coding agent, I would use it like this:

# 1. Get the test suite ID
SUITE_ID=$(curl -s http://localhost:17480/api/v1/test-suites | \
  jq -r '.test_suites[] | select(.scenario_name == "my-scenario") | .id' | head -1)

# 2. Run tests synchronously
RESULTS=$(curl -s -X POST "http://localhost:17480/api/v1/test-suite/${SUITE_ID}/execute-sync" \
  -H "Content-Type: application/json" \
  -d '{"execution_type": "full"}')

# 3. Check results and take action
FAILED=$(echo "$RESULTS" | jq -r '.summary.failed')

if [ "$FAILED" -gt 0 ]; then
  echo "Tests failed. Here are the issues:"
  echo "$RESULTS" | jq -r '.failed_tests[] | "- \(.test_case_name): \(.error_message)"'

  # I would then analyze failures and fix the code
  echo "$RESULTS" | jq -r '.recommendations[]'
else
  echo "All tests passed! Ready to proceed."
fi
```

## Comparison: Async vs Sync

### Async Pattern (Original)
```bash
# Step 1: Start execution
EXEC_ID=$(curl -s -X POST ".../execute" -d '{}' | jq -r '.execution_id')

# Step 2: Poll for results
while true; do
  STATUS=$(curl -s ".../test-execution/${EXEC_ID}/results" | jq -r '.status')
  if [ "$STATUS" != "running" ]; then
    break
  fi
  sleep 5
done

# Step 3: Get final results
curl -s ".../test-execution/${EXEC_ID}/results"
```

### Sync Pattern (New - Much Simpler!)
```bash
# Single request gets everything
curl -s -X POST ".../execute-sync" -d '{}'
```

## Best Practices

### ✅ DO:
- Use appropriate timeout values for your test suite size
- Check HTTP status codes before parsing JSON
- Handle error responses gracefully
- Store execution_id for audit trails
- Use jq or similar tools to parse JSON responses

### ❌ DON'T:
- Use sync endpoint for very long test suites (>10 min)
- Ignore failed_tests array when summary.failed > 0
- Skip timeout configuration for large suites
- Parse responses without checking status codes

## Troubleshooting

### Issue: Request Times Out

**Problem**: Test suite takes longer than HTTP timeout

**Solution**:
```bash
# Increase timeout in request
curl -X POST ".../execute-sync" \
  -d '{"timeout_seconds": 1800}'  # 30 minutes

# Or use async endpoint for very long suites
curl -X POST ".../execute"
```

### Issue: Getting "Test suite not found"

**Problem**: Suite ID doesn't exist or hasn't been synced to database

**Solution**:
```bash
# List available suites
curl http://localhost:17480/api/v1/test-suites | jq '.test_suites[] | .id'

# Or generate a new suite first
curl -X POST http://localhost:17480/api/v1/test-suite/generate \
  -d '{"scenario_name": "my-scenario", "test_types": ["unit"]}'
```

### Issue: Empty Results (0 tests)

**Problem**: Test suite has no test cases

**Solution**:
```bash
# Check suite details
curl http://localhost:17480/api/v1/test-suite/{suite_id} | jq '.test_cases | length'

# If 0, the suite needs test cases generated
# Test generation is async via app-issue-tracker
```

## API Reference

### Find Test Suite by Scenario Name

```bash
curl "http://localhost:17480/api/v1/test-suites" | \
  jq -r --arg scenario "my-scenario" \
    '.test_suites[] | select(.scenario_name == $scenario) | .id'
```

### Get Suite Details Before Execution

```bash
curl "http://localhost:17480/api/v1/test-suite/{suite_id}" | \
  jq '{id, scenario_name, suite_type, test_count: (.test_cases | length)}'
```

### Execute with Full Configuration

```bash
curl -X POST "http://localhost:17480/api/v1/test-suite/{suite_id}/execute-sync" \
  -H "Content-Type: application/json" \
  -d '{
    "execution_type": "regression",
    "environment": "staging",
    "timeout_seconds": 1800,
    "parallel_execution": true,
    "notification_settings": {
      "on_completion": true,
      "on_failure": true,
      "webhook_url": "https://hooks.example.com/test-results"
    }
  }'
```

## Summary

The synchronous execution endpoint provides a **simple, single-request solution** for running tests:

- 🚀 **Fast**: No polling loop needed
- 🎯 **Simple**: One request, complete results
- 🤖 **Agent-friendly**: Perfect for automation
- ✅ **Reliable**: Proper error handling and status codes

**Quick Start:**
```bash
curl -X POST http://localhost:17480/api/v1/test-suite/{suite_id}/execute-sync \
  -H "Content-Type: application/json" \
  -d '{}' | jq '.summary'
```

## See Also

### Related Guides
- [Phased Testing](phased-testing.md) - 7-phase testing architecture
- [Requirements Sync](requirements-sync.md) - Automatic requirement tracking
- [Test Generation](test-generation.md) - AI-powered test creation

### Reference
- [API Endpoints](../reference/api-endpoints.md) - Complete REST API reference
- [Sync Execution Cheatsheet](../reference/sync-execution-cheatsheet.md) - Quick reference
- [CLI Commands](../reference/cli-commands.md) - CLI reference
- [Presets](../reference/presets.md) - Test preset configurations

### Concepts
- [Architecture](../concepts/architecture.md) - Go orchestrator design
