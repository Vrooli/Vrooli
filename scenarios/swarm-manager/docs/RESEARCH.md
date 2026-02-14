# Research Notes

## Uniqueness Analysis

### What Makes Swarm Manager Unique

Swarm Manager fills a specific role in Vrooli:

1. **Execution Governance for Scenario Change**
   - Swarm Manager is the control plane for deciding when backlog work executes.

2. **Backlog Artifact Persistence**
   - Findings from Prompt Manager teams are preserved as git-tracked backlog items with editable supporting files.

3. **Archive-and-Rebuild Workflow**
   - Existing scenarios can be archived into backlog context, iterated on, then reimplemented through controlled execution.

4. **Operator + Agent Team Interop**
   - Human operators and autonomous teams use the same backlog and execution surfaces.

## Relationship to Other Scenarios

| Scenario | Relationship | Notes |
|----------|--------------|-------|
| prompt-manager | Primary producer | Teams produce findings and write backlog items via shared swarm-manager tool skill |
| agent-manager | Execution engine | Runs spawned from backlog research/execution operations |
| ecosystem-manager | Optional integration | Scenario task initialization/coordination where applicable |
| visited-tracker | Integration | Context cleanup campaigns |
| app-issue-tracker | Integration | Issue tracking per scenario |

## Boundaries (Non-Goals)

- **Not a recommendation engine**: Recommendation generation lives in Prompt Manager teams.
- **Not a direct code editor**: Work is governed through backlog + execution artifacts.
- **Not a generic PM tool**: Scope is scenario lifecycle governance.

## Technical Rationale

### File-Based Backlog Storage

Folder-per-item storage (`ideas/`, `research/`, `fix/`, `execute/`) was chosen for:
- Version history through git
- Human readability and manual review
- Rich context attachments next to `spec.json`
- Easy sharing across operators and agents

### Execution Control Modes

Execution modes (`manual`, `scheduled`, `yolo`) balance speed and risk:
- **manual**: explicit human trigger
- **scheduled**: delayed auto-start (for review windows)
- **yolo**: immediate autonomous execution

## References

- [prompt-manager skill: swarm-manager-tools](../../prompt-manager/store/skills/packs/core/swarm-manager-tools/SKILL.md)
- [agent-manager PRD](../../agent-manager/PRD.md)
- [ecosystem-manager PRD](../../ecosystem-manager/PRD.md)
