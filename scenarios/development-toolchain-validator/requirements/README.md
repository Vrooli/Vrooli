# Requirements Registry: development-toolchain-validator

## Overview

This directory contains the requirements definitions for this scenario, organized by operational target.

> **Rewritten 2026-05-18**: The earlier requirements set (21 modules built around the declarative-expectations model) has been retired and replaced by this set, which matches the new "execute skill on pristine golden, evaluate sandbox diff against expected-diff manifest" model. See `../PRD.md` Appendix → *Why the vision changed* for the rationale.

## Structure

```
requirements/
├── index.json           # Module registry
├── README.md            # This file
└── XX-target-name/      # Per-target module folders
    └── module.json      # Requirements for that target
```

## Operational Targets

Requirements are linked to PRD operational targets using the `prd_ref` field. All requirements are `status: planned` — the scenario is being regenerated against the new vision and no implementation exists yet.

| Priority | Target ID | Title | Requirements |
|----------|-----------|-------|-------------|
| P0 | OT-P0-001 | Golden Registry & Regeneration | 2 |
| P0 | OT-P0-002 | Skill Catalog Sync | 2 |
| P0 | OT-P0-003 | Expected-Diff Manifest | 2 |
| P0 | OT-P0-004 | Skill Validation Run + Diff Evaluation | 2 |
| P0 | OT-P0-005 | Tooling Baseline Validation | 2 |
| P0 | OT-P0-006 | Validation Record Storage | 2 |
| P0 | OT-P0-007 | Staleness Tracking | 2 |
| P0 | OT-P0-008 | Validation Report API | 2 |
| P0 | OT-P0-009 | CLI Interface | 1 |
| P0 | OT-P0-010 | UI Dashboard | 2 |
| P1 | OT-P1-001 | Template Version Watcher | 1 |
| P1 | OT-P1-002 | Skill Maturity Score | 1 |
| P1 | OT-P1-003 | Manifest Convergence Tracking | 1 |
| P1 | OT-P1-004 | Run History & Trend Detection | 1 |
| P1 | OT-P1-005 | Coverage Map | 1 |
| P1 | OT-P1-006 | Bulk Re-Validation | 1 |
| P2 | OT-P2-001 | Cross-Golden Consistency | 1 |
| P2 | OT-P2-002 | Per-Skill Cost Budget Alerts | 1 |
| P2 | OT-P2-003 | Webhook Notifications | 1 |
| P2 | OT-P2-004 | Manifest Auto-Generation from Observed Diffs | 1 |
| P2 | OT-P2-005 | Input/Output Token Breakdown | 1 |

## Notes

- All `validation` arrays are empty pending implementation. They will be populated during the rebuild as test files land, not before — to avoid fake references that mislead the auditor.
- This requirements set is intentionally thin. The implementation tree will be regenerated from `templates/scenarios/react-vite` and the per-module test references will be added as code lands.
