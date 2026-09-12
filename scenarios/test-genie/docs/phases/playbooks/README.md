# Playbooks Alias

`playbooks`, `playbook`, and `e2e` are deprecated aliases for the canonical
[`workflow` phase](../workflow/README.md).

Test Genie no longer owns a native playbooks phase. Workflow catalog scanning,
safe execution, findings, deterministic fixes, and BAS artifacts are delegated
to the `workflow-health` scenario through `ScenarioValidationService`.

Compatibility remains for existing callers:

```bash
test-genie execute my-scenario --phases playbooks
test-genie execute my-scenario --phases e2e
```

Both requests normalize to:

```bash
test-genie execute my-scenario --phases workflow
```

Legacy playbooks seed endpoints and claim routes remain while seed lifecycle
ownership is migrated. Treat those as compatibility surfaces, not evidence that
`playbooks` is still a catalog phase.

