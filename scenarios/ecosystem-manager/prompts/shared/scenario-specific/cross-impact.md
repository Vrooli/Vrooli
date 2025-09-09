# Cross-Scenario Impact Analysis

## Core Principle
Every component in Vrooli is interconnected. Changes ripple through the ecosystem. Always analyze impact before making changes.

## Impact Categories

### 1. Direct Dependencies
- Components that directly depend on this one
- Use service.json and grep to find dependents
- Check both directions: what uses this, what this uses

### 2. Shared Resources  
- High-impact resources affect many scenarios (postgres, ollama, redis)
- Medium-impact resources affect several scenarios (qdrant, browserless)
- Low-impact resources affect few scenarios (comfyui, judge0)

### 3. API Contracts
- Breaking API changes affect all consumers
- Always maintain backward compatibility during transitions
- Version APIs properly (v1, v2) rather than replacing
- Ensure that old API version have clear deprecation warnings, so that other agents can clean them up later when they're no longer in use

### 4. Port Conflicts
- Check service.json files for port conflicts
- Verify ports aren't actively in use before allocation
- Use vrooli port ranges to avoid conflicts

## Impact Assessment Process

### Before Changes
1. **Identify Affected Components** - Find all dependents
2. **Classify Impact Level** - CRITICAL/HIGH/MEDIUM/LOW based on:
   - Number of affected scenarios
   - Type of change (breaking vs non-breaking)  
   - Core infrastructure vs peripheral features
3. **Create Mitigation Plan** - Compatibility layers, migration docs, rollback procedures

### Impact Classification
- **CRITICAL**: Affects many scenarios, breaks contracts, requires migration plan
- **HIGH**: Affects several scenarios, needs compatibility layer  
- **MEDIUM**: Affects few scenarios, requires notification
- **LOW**: Affects one scenario, standard validation sufficient

## Change Patterns

### Safe Patterns ✅
- Add new endpoints alongside old ones
- Support multiple configuration formats during transition
- Use feature flags for gradual rollouts
- Maintain backward compatibility with deprecation warnings

### Dangerous Patterns ❌
- Immediate replacement of working interfaces
- Assumption that "no one uses this"
- Skipping impact analysis for "small" changes
- Ignoring failing dependent tests

## Best Practices

### Before Making Changes
- Run impact analysis first
- Test with all dependent components  
- Create migration documentation
- Plan rollback procedures

### During Changes
- Monitor dependent service health
- Check for errors in logs
- Verify API compatibility

### After Changes
- Validate no performance regression
- Monitor for issues
- Update documentation
- Plan removal of compatibility layers

Remember: In interconnected systems, there are no isolated changes. Respect the ecosystem.