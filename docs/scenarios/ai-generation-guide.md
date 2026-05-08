# AI Generation Guide

This page captures project-level guidance for generating scenarios from templates or structured requirements.

## Current Truth

Scenario generation is no longer something that should be documented as pure “one-shot magic.”

The practical current path is:

1. choose a real template
2. scaffold a scenario with the CLI
3. complete the template's orientation gates when present
4. refine manifests, requirements, and implementation
5. validate with scenario-aware testing and requirement-aware workflows

## Canonical Commands

```bash
vrooli scenario template list
vrooli scenario template show <template>
vrooli scenario generate <template> --id <slug> --display-name <name> --description <text>
```

For generation planning without writing files:

```bash
vrooli scenario generate <template> --id <slug> --display-name <name> --description <text> --dry-run
```

For first-session orientation after generation:

```bash
vrooli scenario orient <slug>
vrooli scenario orient <slug> --finalize
```

For requirement-aware work:

```bash
vrooli scenario requirements init
vrooli scenario requirements validate
vrooli scenario requirements report
```

## Guidance For Good Generation

- start from a maintained template
- keep the scenario description specific
- complete orientation before expanding product features when the
  template provides an orientation manifest
- declare required resources honestly
- document the scenario's intended behavior through requirements and tests
- avoid claiming production readiness before validation and deployment checks exist
- treat generation as the start of scenario work, not the end of it

## Related

- [getting-started.md](getting-started.md)
- [VALIDATION.md](VALIDATION.md)
- [DEPLOYMENT.md](DEPLOYMENT.md)
