# Initiative Context

## Strategic Rationale
This initiative exists to move the highest-risk scenario and resource persistence onto the canonical storage seams so backup behavior can be inferred from structure instead of special-case code. The purpose is not to normalize every holdout immediately, but to remove the most dangerous ambiguity before data-backup-manager v2 is built.

## Cross-Item Decisions
- Scenarios should converge on api-core/storage for mutable runtime state.
- Resources should converge on RESOURCE_*_DIR and the resource runtime storage layer.
- True exceptions may remain temporarily, but they must be declared explicitly and documented.
- STORAGE_AUDIT docs should be added or updated where persistence layout materially affects recovery and portability.

## Sequencing Notes
Start with the high-risk scenario migration wave and explicit declaration of legacy resource storage exceptions. Use the persistence audit and backup-contract research as the authority for choosing the first migration targets. The resource normalization wave should focus on high-value disaster-recovery surfaces rather than breadth-first cleanup.
