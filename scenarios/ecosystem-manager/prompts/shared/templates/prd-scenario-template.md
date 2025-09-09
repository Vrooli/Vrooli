# PRD: {{SCENARIO_NAME}}

## What This Does
{{ONE_SENTENCE_VISION}}

**Problem**: {{CLEAR_PROBLEM_DESCRIPTION}}
**Solution**: {{HIGH_LEVEL_SOLUTION}}
**Revenue**: ${{REVENUE_ESTIMATE}} potential value

## Success Metrics (3-4 key measures)
- [ ] {{PRIMARY_METRIC_1}} reaches {{TARGET_1}}
- [ ] {{PRIMARY_METRIC_2}} reaches {{TARGET_2}}
- [ ] User adoption: >{{ADOPTION_TARGET}} active users
- [ ] Performance: {{PERFORMANCE_TARGET}}

## P0 Requirements (Must Have - 5-7 items max)
**These must work before launch:**

- [ ] **{{CORE_FEATURE_1}}**: {{WHAT_IT_MUST_DO_1}}
- [ ] **{{CORE_FEATURE_2}}**: {{WHAT_IT_MUST_DO_2}}
- [ ] **{{CORE_FEATURE_3}}**: {{WHAT_IT_MUST_DO_3}}
- [ ] **Health Check**: API responds with 200 OK
- [ ] **User Authentication**: Users can login/logout
- [ ] **Error Handling**: Clear error messages for failures
- [ ] **Basic UI**: Core workflow works end-to-end

## P1 Requirements (Should Have - 3-4 items max)
- [ ] **{{ENHANCED_FEATURE_1}}**: {{WHAT_IT_SHOULD_DO_1}}
- [ ] **{{ENHANCED_FEATURE_2}}**: {{WHAT_IT_SHOULD_DO_2}}
- [ ] **Performance**: Meets speed/load targets
- [ ] **Analytics**: Usage tracking and reporting

## P2 Requirements (Nice to Have - 2-3 items max)
- [ ] **{{FUTURE_FEATURE_1}}**: {{WHAT_IT_COULD_DO_1}}
- [ ] **{{FUTURE_FEATURE_2}}**: {{WHAT_IT_COULD_DO_2}}
- [ ] **Advanced Analytics**: ML insights and predictions

## Technical Specs
**Frontend**: {{FRONTEND_TECH}}
**Backend**: {{BACKEND_TECH}}
**Database**: {{DATABASE_TECH}}
**Resources Used**: {{RESOURCE_1}}, {{RESOURCE_2}}, {{RESOURCE_3}}

### Key API Endpoints
```
GET  /api/health         → status check
{{PRIMARY_ENDPOINT_1}}   → {{WHAT_IT_DOES_1}}
{{PRIMARY_ENDPOINT_2}}   → {{WHAT_IT_DOES_2}}
```

### User Workflows
1. **{{PRIMARY_WORKFLOW}}**: {{WORKFLOW_STEPS_1}}
2. **{{SECONDARY_WORKFLOW}}**: {{WORKFLOW_STEPS_2}}

## Integration Points
**Resources Used**: {{RESOURCE_1}}, {{RESOURCE_2}}
**Scenarios Enhanced**: {{RELATED_SCENARIO_1}}, {{RELATED_SCENARIO_2}}

## Validation Tests
```bash
# Test these work before marking P0 complete:
{{STANDARD_HEALTH_CHECK}}/api            # Returns 200 OK  
{{PRIMARY_TEST_COMMAND_1}}             # {{EXPECTED_RESULT_1}}
{{PRIMARY_TEST_COMMAND_2}}             # {{EXPECTED_RESULT_2}}
```

## Completion Status
- **P0**: {{P0_COMPLETED}}/{{P0_TOTAL}} ({{P0_PERCENTAGE}}%)
- **P1**: {{P1_COMPLETED}}/{{P1_TOTAL}} ({{P1_PERCENTAGE}}%)
- **P2**: {{P2_COMPLETED}}/{{P2_TOTAL}} ({{P2_PERCENTAGE}}%)
- **Overall**: {{OVERALL_PERCENTAGE}}%

---
*Created: {{DATE}} | Revenue Potential: ${{REVENUE_ESTIMATE}} | Status: {{STATUS}}*