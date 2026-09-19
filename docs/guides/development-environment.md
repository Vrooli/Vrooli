# Development Environment

This page describes the current project-level development environment flow for contributors working on Vrooli itself.

## Canonical Setup Flow

```bash
git clone https://github.com/Vrooli/Vrooli.git
cd Vrooli
make setup
vrooli develop
```

## Current Entry Points

Project-level:

```bash
make setup
vrooli setup
vrooli develop
vrooli status
vrooli doctor
```

Scenario-level:

```bash
cd scenarios/<scenario-name>
make start
make test
make logs
make stop
```

## Setup Profiles

The root setup flow supports environment profiles:

```bash
vrooli setup --environment development
vrooli setup --environment production
vrooli setup --environment minimal
```

It also supports resource and scenario selection:

```bash
vrooli setup --resources enabled
vrooli setup --resources none
vrooli setup --resources postgres,redis
vrooli develop --scenarios none
```

Use `--help` for current details:

```bash
vrooli setup --help
vrooli develop --help
```

## What To Expect

At the project level, current development work assumes:

- the Go-native `vrooli` control plane is authoritative
- resources are managed through `vrooli resource ...`
- scenarios are managed through `vrooli scenario ...` and their local Makefiles
- deeper deployment portability work is separate from day-to-day development setup

## Useful Checks

```bash
vrooli status
vrooli scenario list
vrooli resource list
vrooli doctor
```

## Related Docs

- [../QUICKSTART.md](../QUICKSTART.md)
- [../reference/cli-commands.md](../reference/cli-commands.md)
- [../operations/troubleshooting.md](../operations/troubleshooting.md)
- [../reference/environment-management.md](../reference/environment-management.md)
