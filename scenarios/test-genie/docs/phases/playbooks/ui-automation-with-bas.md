# BAS Workflow Authoring Alias

This page is retained for old links. BAS workflow authoring, validation, and
execution are now documented and enforced by `workflow-health`, then surfaced in
Test Genie through the delegated [`workflow` phase](../workflow/README.md).

Use the workflow-health documentation for current guidance:

- `scenarios/workflow-health/docs/concepts/DOMAINS.md`
- `scenarios/workflow-health/docs/concepts/DATA.md`
- `scenarios/workflow-health/docs/concepts/FLOWS.md`

The `playbooks` phase name is accepted only as a deprecated alias. New commands
should use:

```bash
test-genie execute my-scenario --phases workflow
```
