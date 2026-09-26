# Meta-Optimization Adoption Validation

## Validation Commands

Run these from the repository root after changing this plan of record:

```bash
go test ./scenarios/prompt-manager/api/memberflow
prompt-manager graph operating-model validate --team meta-optimization --id meta-optimization-operating-model
prompt-manager graph operating-model coverage --team meta-optimization --id meta-optimization-operating-model
bash scenarios/prompt-manager/test/agent_system_canon_test.sh
```

## Expected Clean State

The structural plan-of-record manifest should report no missing required files, no missing required headings, no missing package files, and no unregistered durable Markdown files under `docs/meta-optimization/`.

The operating model should validate cleanly against the team graph, topic catalog, work catalog, runtime prompt sections, and plan-of-record registration.

The friction-report taxonomy should resolve from `intake[].taxonomy = "friction-report"` and its `porPath` should point to `docs/meta-optimization/taxonomies/friction-report/README.md`.

## Enforcement Scope

Validation treats [`../manifest.json`](../manifest.json) as the only structural contract for this plan of record. New durable canon must be registered in the manifest, placed under the most specific standard module, and edited only through accepted meta-optimization decisions.
