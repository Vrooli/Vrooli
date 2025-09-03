# 🔄 Cross-Scenario Impact Analysis

## Critical Understanding

Every scenario and resource in Vrooli is part of an interconnected ecosystem. Changes to one component can ripple through the entire system. This document ensures you consider and handle these impacts properly.

## Impact Categories

### 1. Direct Dependencies 🎯
Components that directly rely on this one:

```bash
# Find scenarios using this resource
find_dependent_scenarios() {
    local resource="$1"
    grep -r "\"$resource\"" scenarios/*/service.json | \
        cut -d: -f1 | xargs dirname | sort -u
}

# Find resources used by this scenario
find_resource_dependencies() {
    local scenario="$1"
    jq '.resources | keys[]' "scenarios/$scenario/.vrooli/service.json"
}
```

### 2. Shared Resources 🤝
Resources used by multiple scenarios:

```yaml
# High-Impact Resources (used by 10+ scenarios)
postgres: 47 scenarios     # Breaking changes affect almost everything
n8n: 31 scenarios          # Workflow changes have wide impact
ollama: 28 scenarios       # Model changes affect AI capabilities
redis: 24 scenarios        # Cache changes affect performance

# Medium-Impact Resources (5-10 scenarios)
qdrant: 9 scenarios        # Embedding changes affect search
browserless: 7 scenarios   # Browser automation dependencies
vault: 6 scenarios         # Security configuration changes

# Low-Impact Resources (1-5 scenarios)
comfyui: 4 scenarios       # Image generation specific
judge0: 3 scenarios        # Code execution specific
```

### 3. API Contracts 📋
Changes that break existing integrations:

```javascript
// Before: Working API
GET /api/v1/data
Response: { data: [...], count: 10 }

// After: Breaking change!
GET /api/v2/data  // Different version
Response: { items: [...], total: 10 }  // Different structure

// Proper way: Backward compatible
GET /api/v1/data  // Keep old endpoint
GET /api/v2/data  // Add new endpoint
```

### 4. Port Conflicts ⚡
Port allocation collisions:

```bash
# Check for port conflicts
check_port_conflicts() {
    local new_port="$1"
    
    # Search all service.json files
    if grep -r "\"port\": $new_port" scenarios/*/service.json; then
        echo "❌ Port $new_port already in use!"
        return 1
    fi
    
    # Check runtime
    if nc -z localhost "$new_port" 2>/dev/null; then
        echo "❌ Port $new_port is actively in use!"
        return 1
    fi
    
    echo "✅ Port $new_port is available"
}
```

### 5. Workflow Dependencies 🔄
Shared n8n/node-red workflows:

```yaml
# Shared workflows and their dependents
ollama.json:
  used_by: [agent-metareasoning, prompt-manager, app-debugger]
  breaking_changes: Input/output format changes
  
rate-limiter.json:
  used_by: [system-monitor, api-gateway, web-scraper]
  breaking_changes: Rate limit parameter changes

structured-data-extractor.json:
  used_by: [document-manager, invoice-processor, research-assistant]
  breaking_changes: Schema format changes
```

## Impact Assessment Process

### Before Making Changes

1. **Identify Affected Components**
```bash
#!/bin/bash
# impact-analysis.sh

analyze_impact() {
    local component="$1"
    
    echo "🔍 Analyzing impact of changes to: $component"
    
    # Find direct dependents
    echo "📌 Direct dependents:"
    find_dependent_scenarios "$component"
    
    # Check shared resources
    echo "🤝 Shared resources:"
    find_shared_resources "$component"
    
    # Identify API consumers
    echo "📋 API consumers:"
    find_api_consumers "$component"
    
    # Check for workflow usage
    echo "🔄 Workflow dependencies:"
    find_workflow_usage "$component"
}
```

2. **Classify Impact Level**
```yaml
# Impact Classification
CRITICAL:
  criteria:
    - Affects 10+ scenarios
    - Breaks API contracts
    - Changes shared workflows
    - Modifies database schemas
  action: Requires migration plan and phased rollout
  
HIGH:
  criteria:
    - Affects 5-10 scenarios
    - Changes configuration format
    - Modifies resource interfaces
  action: Requires compatibility layer
  
MEDIUM:
  criteria:
    - Affects 2-5 scenarios
    - Changes optional features
    - Updates documentation only
  action: Requires notification to dependents
  
LOW:
  criteria:
    - Affects 1 scenario
    - Internal changes only
    - Performance improvements
  action: Standard validation gates
```

3. **Create Mitigation Plan**
```markdown
## Change Mitigation Plan

### Change Description
[What is being changed and why]

### Impact Assessment
- Affected scenarios: [list]
- Affected resources: [list]
- Breaking changes: [yes/no]
- Risk level: [CRITICAL/HIGH/MEDIUM/LOW]

### Mitigation Strategy
1. Compatibility layer for [duration]
2. Migration documentation for [affected components]
3. Rollback procedure if needed
4. Communication plan to stakeholders

### Testing Plan
- [ ] Test with each affected scenario
- [ ] Verify backward compatibility
- [ ] Load test with typical usage
- [ ] Test rollback procedure
```

## Common Impact Patterns

### Pattern 1: Database Schema Changes
```sql
-- BAD: Breaking change
ALTER TABLE users DROP COLUMN email;

-- GOOD: Backward compatible
ALTER TABLE users ADD COLUMN email_new VARCHAR(255);
-- Migrate data
UPDATE users SET email_new = email;
-- Later, after all code updated
ALTER TABLE users DROP COLUMN email;
```

### Pattern 2: API Version Changes
```javascript
// BAD: Replace endpoint
app.delete('/api/users/:id');
app.post('/api/users/:id/delete'); // New endpoint

// GOOD: Support both during transition
app.delete('/api/users/:id', handleDelete);
app.post('/api/users/:id/delete', handleDelete); // Same handler
// Log deprecation warning for old endpoint
```

### Pattern 3: Configuration Format Changes
```yaml
# BAD: Complete format change
# Old: { database: "postgresql://..." }
# New: { db: { type: "postgres", url: "..." } }

# GOOD: Support both formats
config = loadConfig();
if (config.database) {
  // Handle old format
  config.db = { type: "postgres", url: config.database };
}
// Use new format internally
```

### Pattern 4: Resource Interface Changes
```bash
# BAD: Change command syntax
# Old: resource-ollama add-model llama2
# New: resource-ollama model add llama2

# GOOD: Support both syntaxes
case "$1" in
    add-model|model)
        if [ "$1" = "add-model" ]; then
            echo "Warning: 'add-model' is deprecated, use 'model add'"
            add_model "$2"
        elif [ "$2" = "add" ]; then
            add_model "$3"
        fi
        ;;
esac
```

## Impact Monitoring

### Real-time Monitoring
```bash
# Monitor for impact during changes
monitor_impact() {
    # Check health of dependent services
    for scenario in $(find_dependent_scenarios "$1"); do
        curl -sf "http://localhost:$(get_port "$scenario")/health" || \
            echo "⚠️ $scenario may be affected"
    done
    
    # Check for errors in logs
    grep -i "error.*$1" /var/log/vrooli/*.log
    
    # Monitor resource usage
    check_resource_usage "$1"
}
```

### Post-Change Validation
```bash
# Validate after changes
validate_no_negative_impact() {
    local component="$1"
    
    # Run integration tests for all dependents
    for dependent in $(find_dependents "$component"); do
        echo "Testing $dependent..."
        (cd "scenarios/$dependent" && ./test.sh) || return 1
    done
    
    # Check performance metrics
    check_performance_regression "$component"
    
    # Verify API compatibility
    test_api_compatibility "$component"
}
```

## Communication Protocol

### Before High-Impact Changes
```markdown
## 🚨 Planned Change Notice

**Component**: [name]
**Change Type**: [breaking/non-breaking]
**Impact Level**: [CRITICAL/HIGH/MEDIUM/LOW]
**Scheduled Date**: [date]

### Affected Components
- [List of scenarios/resources]

### Required Actions
- [ ] Update configuration by [date]
- [ ] Test integration by [date]
- [ ] Confirm readiness

### Support
- Documentation: [link]
- Migration guide: [link]
- Contact: [person/channel]
```

### After Changes
```markdown
## ✅ Change Completed

**Component**: [name]
**Completion Time**: [timestamp]
**Status**: SUCCESS/PARTIAL/ROLLED_BACK

### Results
- All dependent services: [status]
- Performance impact: [metrics]
- Issues encountered: [list]

### Follow-up Actions
- [ ] Monitor for 24 hours
- [ ] Remove compatibility layer by [date]
- [ ] Update documentation
```

## Best Practices

### Do's ✅
- Always run impact analysis first
- Maintain backward compatibility when possible
- Provide migration paths for breaking changes
- Test with all dependent components
- Document all changes clearly
- Monitor after deployment

### Don'ts ❌
- Make breaking changes without notice
- Assume "no one uses this"
- Skip impact analysis for "small" changes
- Ignore failing dependent tests
- Remove old interfaces immediately

## Quick Reference

```bash
# Commands for impact analysis
vrooli analyze-impact [component]        # Full impact report
vrooli find-dependents [component]       # List dependents
vrooli test-compatibility [component]    # Test with dependents
vrooli validate-no-regression [component] # Performance check

# Impact levels
CRITICAL: Stop and create migration plan
HIGH: Implement compatibility layer
MEDIUM: Notify affected parties
LOW: Standard validation sufficient
```

Remember: In an interconnected system, there are no truly isolated changes. Every modification has the potential to affect other components. Respect the ecosystem.