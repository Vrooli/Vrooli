# Known Problems and Deferred Ideas

## Open Issues

### High Priority
None currently - scenario is in initialization phase.

### Medium Priority
1. **JS to Go Migration Complexity**: The existing `scripts/scenarios/lib/completeness.js` has ~550 lines of logic to port. Need to ensure parity during migration.

2. **Circuit Breaker Threshold Tuning**: Default threshold of 3 failures may need adjustment based on real-world collector behavior. Consider making this configurable per-collector.

### Low Priority
1. **UI Template Code**: The react-vite template includes placeholder UI that should be replaced with actual dashboard components.

## Deferred Ideas

### P2 Features (Future Consideration)
- **OT-P2-005 Webhook Notifications**: Alert on score changes or collector failures
- **OT-P2-007 CLI Enhancement**: Update `vrooli scenario completeness` to call this API
- **OT-P2-008 Score Badges**: Embeddable badges for README files
- **OT-P2-009 Custom Collectors**: Plugin system for new scoring dimensions
- **OT-P2-010 Anomaly Detection**: Alert on unexpected score changes

### Technical Debt to Address Later
1. Consider shared Go library for direct import by ecosystem-manager (avoid HTTP overhead)
2. Evaluate if SQLite is sufficient for high-volume history storage
3. Plan migration path from existing JS scoring to this API

## Risks

| Risk | Impact | Mitigation |
|------|--------|------------|
| Browser-automation-studio dependency | High - e2e metrics unavailable | Circuit breaker auto-disables; weight redistribution |
| Score parity with JS version | Medium - different scores during migration | Run parallel validation before deprecating JS |
| SQLite performance at scale | Low - many scenarios with long history | Consider PostgreSQL option for large deployments |

## Resolved Issues

None yet - scenario is newly created.
