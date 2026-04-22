# Initiative Context

## Strategic Rationale
This initiative defines the storage and backup contract that the rest of the work depends on. The platform should default backup behavior from canonical storage conventions rather than hand-authored scenario logic. Research should identify every persisted surface that matters for self-hosted recovery, explicitly exclude browser-only storage, and define the small declarative exception mechanism needed for true holdouts.

## Cross-Item Decisions
- Default backup behavior should come from canonical storage classes and resource dependency types.
- Browser-only persistence like localStorage, sessionStorage, and IndexedDB is intentionally out of scope.
- Hardcoded per-scenario and per-resource branches inside data-backup-manager are not acceptable.
- Scenario-auditor should be the enforcement surface that prevents storage drift from reappearing.

## Sequencing Notes
Run the persistence surface audit first, then define the backup contract, then land enforcement and docs/template alignment. Storage migration and data-backup-manager work should consume this initiative rather than inventing their own rules.
