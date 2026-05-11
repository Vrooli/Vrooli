# Scenario QA Adoption Validation

## Validation Commands

Run these from the repository root after changing this plan of record:

```bash
go test ./scenarios/prompt-manager/api/memberflow
prompt-manager graph operating-model validate --team scenario-qa --id scenario-qa-operating-model
prompt-manager graph operating-model coverage --team scenario-qa --id scenario-qa-operating-model
```

## Expected Clean State

The structural plan-of-record manifest should report no missing required files, no missing required headings, no missing package files, and no unregistered durable Markdown files under `docs/scenario-qa/`.

The operating model should validate cleanly against the team graph, topic catalog, decision catalog, runtime prompt sections, and plan-of-record registration.

## Migration Notes

This folder migrated from top-level taxonomy and registry folders into the shared `team-plan-of-record/v1` shape:

- `OPERATING_MODEL.md` moved to `operating/OPERATING_MODEL.md`.
- `BUG_REPORT_TAXONOMY.md` and `taxonomy.json` moved to `taxonomies/bug-report/`.
- `investigation-techniques/`, `audit-techniques/`, and `readiness-checks/` moved under `methods/`.

