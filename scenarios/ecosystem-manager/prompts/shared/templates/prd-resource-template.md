# PRD: {{RESOURCE_NAME}}

## What This Does
{{ONE_SENTENCE_PURPOSE}}

**Problem**: {{WHAT_PROBLEM_DOES_THIS_SOLVE}}
**Solution**: {{HIGH_LEVEL_SOLUTION}}
**Value**: Enables {{SCENARIOS_THAT_NEED_THIS}} scenarios

## Success Metrics (3-4 key measures)
- [ ] Health check responds in <1s
- [ ] {{PRIMARY_PERFORMANCE_METRIC}}
- [ ] Used by >{{ADOPTION_TARGET}} scenarios
- [ ] {{BUSINESS_VALUE_METRIC}}

## P0 Requirements (Must Have - 5-7 items max)
**These must work before marking complete:**

- [ ] **Health Check**: `/health` endpoint responds with status/version/uptime
- [ ] **Lifecycle**: `setup`, `develop`, `test`, `stop` commands work
- [ ] **CLI Integration**: `status`, `health`, `logs` commands work  
- [ ] **Service Config**: Valid service.json with correct port allocation
- [ ] **Error Handling**: Graceful failures with clear error messages
- [ ] **{{CORE_FUNCTIONALITY_1}}**: {{WHAT_IT_MUST_DO_1}}
- [ ] **{{CORE_FUNCTIONALITY_2}}**: {{WHAT_IT_MUST_DO_2}}

## P1 Requirements (Should Have - 3-4 items max)
- [ ] **Advanced Health**: Dependency checks and performance metrics
- [ ] **Content Management**: Add/remove/list operations
- [ ] **{{ENHANCED_FEATURE_1}}**: {{WHAT_IT_SHOULD_DO_1}}
- [ ] **{{ENHANCED_FEATURE_2}}**: {{WHAT_IT_SHOULD_DO_2}}

## P2 Requirements (Nice to Have - 2-3 items max)  
- [ ] **Auto-scaling**: Handle load spikes automatically
- [ ] **Advanced Analytics**: Usage metrics and insights
- [ ] **{{FUTURE_FEATURE}}**: {{WHAT_IT_COULD_DO}}

## Technical Specs
**Core Technology**: {{CORE_TECH}}
**Port**: {{PRIMARY_PORT}}
**Dependencies**: {{KEY_DEPENDENCIES}}

### Key API Endpoints
```
GET  /health              → status, version, uptime
{{PRIMARY_ENDPOINT_1}}    → {{WHAT_IT_DOES_1}}
{{PRIMARY_ENDPOINT_2}}    → {{WHAT_IT_DOES_2}}
```

### Required CLI Commands
```bash
vrooli resource {{RESOURCE_NAME}} status    # Show current status
vrooli resource {{RESOURCE_NAME}} health    # Run health check  
vrooli resource {{RESOURCE_NAME}} logs      # View recent logs
```

## Integration Points
**Depends On**: {{DEPENDENCY_1}}, {{DEPENDENCY_2}}
**Used By**: {{CALLER_1}}, {{CALLER_2}}

## Validation Tests
```bash
# Test these work before marking P0 complete:
timeout 5 curl -sf http://localhost:${PORT}/health  # Returns 200 OK
vrooli resource {{RESOURCE_NAME}} status    # Shows running
vrooli resource {{RESOURCE_NAME}} test      # All tests pass
```

## Completion Status
- **P0**: {{P0_COMPLETED}}/{{P0_TOTAL}} ({{P0_PERCENTAGE}}%)
- **P1**: {{P1_COMPLETED}}/{{P1_TOTAL}} ({{P1_PERCENTAGE}}%)  
- **P2**: {{P2_COMPLETED}}/{{P2_TOTAL}} ({{P2_PERCENTAGE}}%)
- **Overall**: {{OVERALL_PERCENTAGE}}%

---
*Created: {{DATE}} | Status: {{STATUS}} | v2.0 Compliant: {{COMPLIANCE_STATUS}}*