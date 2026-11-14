# Start Here — Tech Tree Designer

## Initialization Protocol

This is the entry point for developing the existing scenario, not instructions to regenerate a template. The September 2026 target extends the current scenario/proto planner into repository-wide ecosystem design. Documentation defines intent; it does not certify runtime readiness.

1. Read [PRD](../PRD.md), [architecture](concepts/ARCHITECTURE.md), [requirements](../requirements/README.md), and [experience contract](../experience/README.md).
2. Read the shared [contract-driven development method](../../../docs/agent-system/SCENARIO_DEVELOPMENT.md). Swarm Manager owns the approved mandate; Agent Manager owns execution. One approved development backlog item may authorize iterative improvement within its boundary. Do not create an approval item for every local improvement.
3. Read `prompt-manager skill read tech-tree-designer` for current operations and `prompt-manager skill read tech-tree-designer-improve` for development judgment. Run `program-runtime library run tech-tree-designer.setpoint-read` for the outcome inventory and diagnostics. The [program contract](../.vrooli/program-runtime/setpoint-read.json) owns its output semantics; `ok` is not acceptance. Missing owner sensors remain explicit.
4. Inspect actual implementation and evidence. Locate the highest broken contract, obligation, evidence or implementation layer with scenario-work-ladder. Preserve existing completed evidence while distinguishing the expanded target.
5. Use the [outcome evidence inventory](internal/TESTING.md#outcome-evidence-inventory) to establish missing owner sensors and integrations within scope. Review the [development packet](internal/DECISIONS.md#development-review-proposal--ecosystem-design-v1) and [proposed numeric cohorts](internal/PERFORMANCE.md#proposed-qualification-profile-v1) before launch; proposals are not approved allowances.
6. Implement in the sequence below; use the owner's validators and scoped Test Genie phases. Keep the mandate's authority, budget and stop conditions visible.
7. Record evidence against exact revisions and update requirements through their evidence workflow. Swarm, not TTD, decides mandate acceptance.

### Delivery sequence

1. **Model and authority:** separate observed facts, intended capabilities and proposal-local deltas; define identities, ownership and safe selection.
2. **Durable draft slice:** create a bounded multi-target documentation proposal, retain immutable revisions and render meaningful diffs without canonical writes.
3. **Review and apply:** link exact revisions from plans; qualify grants, base conflicts, owner dispatch, interruption and recovery.
4. **Qualification:** exercise cross-repository targets, source failure, accessible operator journeys and constrained-machine scale cohorts.
5. **Development proof:** qualify this scenario as the second real development target after Audio Tools through the approved Swarm/Agent Manager path.
6. **Expansion:** alternatives, experiments, richer fulfillment and strategic analysis, according to their PRD priority. P2 is not a prerequisite for P0 completion.

### Decisions still requiring qualification

Numeric latency/resource/storage budgets and supported cohorts are pending in [PERFORMANCE](internal/PERFORMANCE.md). Owner adapter capabilities, retention windows, remote/multi-user identity, and experiment allowances must be established before claiming those surfaces supported. These do not reopen the approved three-view product direction.

| Decision | Required output before claiming support |
|---|---|
| Supported repository/artifact owners | Capability inventory, safe roots, canonical/generated distinctions, version compatibility and explicit unsupported cases |
| Scale and response guarantees | Reproducible cohorts, numeric thresholds, bounded query/storage policies and regression evidence |
| Retention and recovery | Review/receipt pin policy, deletion consequences, backup/restore test and recovery objectives |
| Authority and amendment | Qualified grant schema, revision/selection binding, revocation behavior and material-change disposition |
| Experiments or remote users | Explicit runtime/effect allowances, isolation/identity evidence and deployment-specific access model |
| Autonomous-development setup | Discoverable scenario-specific skill/program coverage, sensor matrix and real Swarm/Agent Manager acceptance-path evidence |

The implementing agent should investigate these owners and propose missing material choices under its mandate. It must not choose new spending, deployment or destructive retention authority merely to close a readiness gap.

## Architecture Rules

Domain code owns behavior and storage; API/CLI/UI translate the same typed contract. SDA owns observation, TTD owns authored design, Plan Manager owns plans, Swarm owns authority, and qualified workspace/artifact owners perform application. The control plane owns runtime isolation. Do not build replacement scanners, Git engines, schedulers or sandboxes.

Read [FLOWS](concepts/FLOWS.md), [DATA](concepts/DATA.md), [SECURITY](internal/SECURITY.md) and [TESTING](internal/TESTING.md) before implementing proposal effects. Root DESIGN.md remains the visual/accessibility contract.

## Replacing The Example Domain

Historical template initialization is complete: graph, planning and ontology are real domains. Do not recreate the removed notes example or delete working domains. Evolve the existing planning seam beyond ProtoFile only when owner-directed generic artifact handling is qualified.
