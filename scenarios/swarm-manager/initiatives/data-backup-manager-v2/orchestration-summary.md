# Initiative Context

## Strategic Rationale
This initiative builds the real backup and restore capability that self-hosted Vrooli deployments need for trust and monetization. It should provide a credible answer to full-system backup and recovery without turning into a pile of one-off scenario branches.

## Cross-Item Decisions
- Data Backup Manager v2 must be manifest-driven and storage-contract-driven.
- It should discover scenario storage classes and resource runtime roots generically.
- It should support local archive destinations, attached storage paths, and S3-compatible targets.
- Restore ordering, cataloging, integrity verification, and dry-run safety checks are first-class requirements.
- Browser-only storage remains out of scope.

## Sequencing Notes
Do not implement v2 against the legacy storage shape. First consume the backup contract and the first storage convergence wave. Then build discovery, then adapters and backup targets, then restore catalog and operator-facing surfaces.
