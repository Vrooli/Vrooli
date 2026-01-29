# Research Notes

## Uniqueness Analysis

### What Makes Swarm Manager Unique

Swarm Manager fills a specific gap in the Vrooli ecosystem:

1. **Ecosystem Orchestration**: Unlike individual scenarios that solve specific problems, Swarm Manager orchestrates the entire scenario ecosystem - tracking what exists, what's planned, and what should be built next.

2. **Backlog-to-Scenario Pipeline**: Provides a structured path from research/ideas to implemented scenarios, with git-tracked context preservation and integration with ecosystem-manager for automated builds.

3. **Autonomous Recommendation Engine**: Three-mode operation (off/suggestions/yolo) with configurable data sources allows both human-guided and fully autonomous improvement cycles.

### Relationship to Other Scenarios

| Scenario | Relationship | Notes |
|----------|--------------|-------|
| ecosystem-manager | Producer | Consumes ecosystem-manager API to initialize/improve scenarios |
| agent-manager | Producer | All agent work spawned through agent-manager |
| knowledge-observatory | Data Source | Reads PROBLEMS.md for recommendation engine |
| scenario-completeness-scoring | Data Source | Reads completeness scores for priority/recommendations |
| test-genie | Data Source | Reads test results for recommendations |
| visited-tracker | Integration | Context cleanup campaigns |
| app-issue-tracker | Integration | Issue management per scenario |

### Not Duplicating Existing Tools

- **Not a code editor**: Doesn't write code, delegates to ecosystem-manager
- **Not a test runner**: Integrates with test-genie for test execution
- **Not a project manager**: Focuses on scenario orchestration, not task tracking
- **Not a dashboard**: Provides controls, not just monitoring

## Technical Research

### File-Based Backlog Storage Rationale

Chose git-tracked folder-per-item over database-only storage:
- **Version History**: Git provides free version control
- **Collaboration**: Multiple agents/humans can work on backlog items
- **Context Preservation**: Rich media (images, docs) alongside metadata
- **Portability**: Backlog items can be moved, copied, shared easily

### Recommendation Engine Data Sources

Research on useful signals for autonomous recommendations:
1. **PROBLEMS.md**: Direct issue tracking, high signal
2. **Completeness scores**: Quantitative readiness metric
3. **Test phase results**: Quality indicator
4. **Test coverage**: Code health metric
5. **Custom focus**: Human override for priorities

## References

- [ecosystem-manager PRD](../../ecosystem-manager/PRD.md)
- [agent-manager PRD](../../agent-manager/PRD.md)
- [scenario-generator template](../../ecosystem-manager/prompts/templates/scenario-generator.md)
