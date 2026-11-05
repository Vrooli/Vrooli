# Testing

## Commands

```bash
vrooli scenario test fall-foliage-explorer structure
vrooli scenario test fall-foliage-explorer dependencies
vrooli scenario test fall-foliage-explorer unit
vrooli scenario test fall-foliage-explorer integration
vrooli scenario test fall-foliage-explorer business
vrooli scenario test fall-foliage-explorer performance
vrooli scenario test fall-foliage-explorer all
```

Full suite runs synchronize requirement coverage.

## Requirement Tags

Existing Go tests use `[REQ:...]` comments to make coverage discoverable by scenario tooling. Keep tags on tests that validate an operational target and avoid tagging tests that only exercise helpers without product significance.

## Coverage Risk

The unit phase has historical coverage failures below the configured 50% threshold. Add targeted tests before raising thresholds.
