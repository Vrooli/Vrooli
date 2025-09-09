# Standard Command Patterns

## Health Check Commands
```bash
# Basic health check
curl localhost:${PORT}/health

# Health check with timeout
timeout 5 curl -sf http://localhost:${PORT}/health

# Health check with output capture
curl localhost:${PORT}/health > /tmp/health-check.txt
```

## Validation Commands
```bash
# Resource validation
vrooli resource {{NAME}} status    # Show current status
vrooli resource {{NAME}} health    # Run health check  
vrooli resource {{NAME}} logs      # View recent logs
vrooli resource {{NAME}} test      # Run tests

# Scenario validation  
vrooli scenario {{NAME}} run       # Start scenario
vrooli scenario {{NAME}} test      # Run tests
curl localhost:{{PORT}}/api/health # API health check
```

## Common Test Patterns
```bash
# Returns 200 OK
curl -sf http://localhost:${PORT}/health

# All tests pass
./test.sh && echo "✅ Tests passed"

# Service responds within timeout
timeout 5 curl localhost:${PORT}/health || echo "❌ Timeout"
```