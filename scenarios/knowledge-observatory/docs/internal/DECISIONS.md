# Decisions

## Decision Log

| Date | Decision | Reason |
|---|---|---|
| 2026-05-10 | Scenario documentation validation is manifest-driven | Templates should own documentation expectations so Knowledge Observatory can validate any template contract |
| 2026-05-10 | Append/reset operations are declared per document | Log behavior should be extensible without hardcoded doc types |
| 2026-05-10 | Knowledge Observatory dogfoods the v2 docs contract | The validator should pass its own health checks |

## Superseded Decisions

The previous hardcoded `docschema` model has been retired in favor of template
and scenario manifests.
