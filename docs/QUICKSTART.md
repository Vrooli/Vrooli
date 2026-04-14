# Vrooli Quick Start

This is the canonical first-touch guide for the project-level Vrooli platform.

## What You Are Starting

Vrooli is a local, cross-platform platform for orchestrating:

- resources such as databases, inference services, automation systems, and supporting infrastructure
- scenarios that compose those capabilities into products, internal tools, operator surfaces, and meta-systems

The root control surface is the Go-native `vrooli` CLI.

## Setup

```bash
git clone https://github.com/Vrooli/Vrooli.git
cd Vrooli
make setup
```

## Start The Development Stack

```bash
vrooli develop
```

For project-level command discovery:

```bash
vrooli help
```

## Inspect The Platform

```bash
vrooli status
vrooli scenario list
vrooli resource list
```

## Work With A Scenario

Scenario-local Makefiles are the preferred operational path for individual scenarios:

```bash
cd scenarios/<scenario-name>
make start
make test
make logs
make stop
```

The root CLI remains available when you need cross-scenario operations:

```bash
vrooli scenario list
vrooli scenario info <name>
vrooli scenario start <name>
vrooli scenario test <name>
vrooli scenario logs <name>
```

## Work With Resources

```bash
vrooli resource list
vrooli resource status
vrooli resource start-all
vrooli resource start postgres
vrooli resource logs postgres
```

## Testing

Run project-level validation with:

```bash
make test
```

Run scenario-focused testing with:

```bash
vrooli scenario test <name>
```

For the deeper testing workflow, see [TESTING.md](TESTING.md).

## Where To Go Next

- [README.md](README.md) for the docs hub
- [concepts/ARCHITECTURE.md](concepts/ARCHITECTURE.md) for the platform mental model
- [concepts/GLOSSARY.md](concepts/GLOSSARY.md) for core terminology
- [reference/cli-commands.md](reference/cli-commands.md) for command reference
- [deployment/README.md](deployment/README.md) for deployment tiers and current packaging reality
