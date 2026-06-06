# Bookmark Intelligence Hub Requirements

`index.json` is the canonical requirement registry for this scenario. Requirements are grouped by the PRD operational target they validate, with at least two requirements mapped to each target so high-level business goals are decomposed into implementation and contract checks.

Current modules:

- `BIH-AUTH-*`: profile isolation, authentication, and profile-specific processing.
- `BIH-PLAT-*`: supported social platform connectors and connector status.
- `BIH-CAT-*`: categorization buckets and category summaries.
- `BIH-ACT-*`: action suggestion, approval, and rejection workflows.
- `BIH-API-*`: versioned cross-scenario API contracts.
- `BIH-PROC-*`: processing, polling, and sync response contracts.
- `BIH-INT-*`: data-structurer and scenario-authenticator integration readiness.
- `BIH-LEARN-*`: learning and accuracy measurement readiness.
- `BIH-BULK-*`: bulk action review contracts.
- `BIH-EXP-*`: export-ready bookmark query contracts.
- `BIH-SEARCH-*`: search and filter payload readiness.
- `BIH-EXT-*`, `BIH-MOB-*`, `BIH-ANA-*`, `BIH-CAL-*`, `BIH-TEAM-*`: future extension, mobile, analytics, calendar, and collaboration readiness.

When adding a requirement, preserve the `BIH-<MODULE>-NNN` ID pattern, map it to the exact PRD operational target, and include an automated validation reference whenever practical.
