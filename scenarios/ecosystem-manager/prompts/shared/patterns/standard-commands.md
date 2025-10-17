# Standard Command Patterns

## Health Check Commands
```bash
# Basic health check (may hang if service down)
curl -sf http://localhost:${PORT}/health

# Recommended: Health check with timeout
timeout 5 curl -sf http://localhost:${PORT}/health

# Health check with output capture
timeout 5 curl -sf http://localhost:${PORT}/health > /tmp/health-check.txt

# Scenario API health check
timeout 5 curl -sf http://localhost:${PORT}/api/health
```

## Validation Commands
```bash
# Resource validation
vrooli resource [name] status    # Show current status
vrooli resource [name] health    # Run health check  
vrooli resource [name] logs      # View recent logs
vrooli resource [name] test      # Run tests

# Scenario validation  
vrooli scenario [name] run       # Start scenario
vrooli scenario [name] test      # Run tests
timeout 5 curl -sf http://localhost:${PORT}/api/health  # API health check
```

## Common Test Patterns
```bash
# Service responds with 200
timeout 5 curl -sf http://localhost:${PORT}/health

# All tests pass
./test.sh && echo "✅ Tests passed"

# Service responds within timeout
timeout 5 curl -sf http://localhost:${PORT}/health || echo "❌ Timeout"
```