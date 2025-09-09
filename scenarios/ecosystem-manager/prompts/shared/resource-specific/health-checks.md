# Health Checks

## Purpose
Health checks are the heartbeat of the system. They ensure services are alive, responsive, and ready to serve. Health checks must be fast and accurate - they determine service availability and system reliability.

## Checking Component Health

### System-Wide Status
```bash
vrooli status                    # Overall system health
vrooli status --verbose          # Detailed system status
vrooli status --json             # JSON output for scripts
```

### Scenario Health
```bash
vrooli scenario status           # All scenarios
vrooli scenario status <name>    # Specific scenario
vrooli scenario status --json    # JSON format
```

### Resource Health  
```bash
vrooli resource status           # All resources
vrooli resource status <name>    # Specific resource
vrooli resource status --json    # JSON format
```

All commands support the `--json` flag for programmatic use.

## Best Practices

### DO's
✅ **Keep health checks fast** (<1 second ideally)  
✅ **Check actual functionality** not just process existence  
✅ **Include dependency checks** for critical services  
✅ **Add appropriate timeouts** to prevent hanging  
✅ **Return meaningful status** in response body  
✅ **Implement graceful degradation** for non-critical checks  

### DON'Ts
❌ **Don't check everything** - Focus on critical paths  
❌ **Don't block on slow checks** - Use async where possible  
❌ **Don't ignore failures** - Log and alert appropriately  
❌ **Don't return OK when degraded** - Be honest about state  
❌ **Don't forget startup time** - Allow services to initialize  

## Remember

**Health checks are critical** - They determine service availability

**Fast and accurate** - Quick response, truthful status

**Fail fast, recover gracefully** - Detect issues quickly, handle smoothly