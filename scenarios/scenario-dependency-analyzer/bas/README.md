# BAS Automation

SDA does not currently require browser automation playbooks for its core validation path.

When UI workflows are added, place cases under `bas/cases/**` and regenerate the tracked registry from the scenario root:

```bash
test-genie registry build
```

`bas/registry.json` is generated and should not be edited by hand.
