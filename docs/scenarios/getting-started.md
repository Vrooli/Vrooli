# Getting Started With Scenarios

This page is the canonical entrypoint for creating or modifying scenarios at the platform level.

## Inspect Available Templates

```bash
vrooli scenario template list
vrooli scenario template show <template>
```

## Generate A Scenario

```bash
vrooli scenario generate <template> \
  --id my-scenario \
  --display-name "My Scenario" \
  --description "One-sentence summary"
```

The CLI help is the final authority:

```bash
vrooli scenario generate --help
```

## Preferred Workflow After Generation

```bash
cd scenarios/<scenario-name>
make orient
make start
make test
make logs
make stop
```

You can also inspect the scenario via the root CLI:

```bash
vrooli scenario info <name>
vrooli scenario orient <name>
vrooli scenario status <name>
vrooli scenario test <name>
```

Orientation-enabled templates render temporary startup metadata to the
generated scenario. `vrooli scenario orient <name>` reports which
template-owned initialization gates are complete, including early work
such as charter, requirements, domain map, dependency decisions, design
language, and replacement of reference domains. When all required gates
pass, finalize explicitly:

```bash
vrooli scenario orient <name> --finalize
```

Finalization removes only the template-declared temporary orientation
metadata. It does not remove scenario provenance, docs, requirements, or
implementation files.

## Requirements

If the scenario uses requirement tracking:

```bash
vrooli scenario requirements init
vrooli scenario requirements validate
vrooli scenario requirements report
```

## What To Focus On

When shaping a scenario, focus on:

- clear purpose
- honest resource dependencies
- early completion of the template's orientation gates when present
- lifecycle correctness
- requirement coverage where appropriate
- validation that matches intended behavior
- docs that describe the specific scenario locally rather than expanding project-wide canon

## What To Avoid

- inventing undocumented scenario structure rules as if they were universal
- claiming deployment readiness without tier-aware validation
- describing old shell-era patterns as canonical
- turning one scenario's internal conventions into platform-wide law

## Related

- [CONCEPTS.md](CONCEPTS.md)
- [VALIDATION.md](VALIDATION.md)
- [DEPLOYMENT.md](DEPLOYMENT.md)
- [PRODUCTION_BUNDLES.md](PRODUCTION_BUNDLES.md)
- [../deployment/README.md](../deployment/README.md)
