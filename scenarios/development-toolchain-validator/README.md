# Development Toolchain Validator

Validates steer skill interoperability, development tooling correctness, and cross-scenario coherence against known-good reference implementations.

## Why This Exists

The Vrooli ecosystem has 45+ steer skills that guide AI agents during scenario development. Each skill focuses on one architectural dimension. **This scenario detects when skills conflict with each other** and when development tools (scenario-auditor, test-genie, scenario-completeness-scoring) produce incorrect results.

For the full vision, see [docs/concepts/VISION.md](docs/concepts/VISION.md).

## Quick Start

```bash
# Start the scenario
cd scenarios/development-toolchain-validator && make start

# Register a reference
development-toolchain-validator references add reference-react-vite --template react-vite

# Connect a skill
development-toolchain-validator skills connect api-steer --reference reference-react-vite

# Add expectations and validate
development-toolchain-validator validate reference-react-vite
```

See [docs/QUICKSTART.md](docs/QUICKSTART.md) for the full walkthrough.

## Architecture

- **Go API** (`api/`): REST API for managing references, connections, expectations, and validation
- **React UI** (`ui/`): Dashboard for viewing validation reports, overlaps, conflicts, and maturity
- **CLI** (`cli/`): Command-line interface for all operations

See [docs/concepts/ARCHITECTURE.md](docs/concepts/ARCHITECTURE.md) for the technical design.

## Documentation

| Document | Description |
|----------|-------------|
| [QUICKSTART](docs/QUICKSTART.md) | Get running in 5 minutes |
| [Vision & Purpose](docs/concepts/VISION.md) | Why this exists and ecosystem integration |
| [Architecture](docs/concepts/ARCHITECTURE.md) | Technical design and data flow |
| [Glossary](docs/concepts/GLOSSARY.md) | Key terms and definitions |
| [Skill Connections](docs/concepts/SKILL-CONNECTIONS.md) | How skills connect to references |
| [Assertion Engine](docs/concepts/ASSERTION-ENGINE.md) | How validation works |
| [Connecting Skills Guide](docs/guides/connecting-skills.md) | Step-by-step workflow |
| [Writing Assertions Guide](docs/guides/writing-assertions.md) | Assertion syntax and patterns |
| [Interpreting Reports Guide](docs/guides/interpreting-reports.md) | Reading validation results |
| [Tooling Baselines Guide](docs/guides/tooling-baselines.md) | Validating dev tools against references |
| [API Reference](docs/reference/api-endpoints.md) | REST API documentation |
| [CLI Reference](docs/reference/cli-commands.md) | CLI command documentation |
| [Configuration](docs/reference/configuration.md) | Environment variables and settings |
| [Data Model](docs/reference/data-model.md) | PostgreSQL schema |

## Dependencies

- **PostgreSQL**: Primary data store
- **prompt-manager API**: Read skills, versions, metadata (runtime dependency)
- **scenario-auditor** (P1): Tooling baseline validation
- **test-genie** (P1): Tooling baseline validation
- **scenario-completeness-scoring** (P1): Tooling baseline validation

## Testing

```bash
cd scenarios/development-toolchain-validator && make test
```

## Related Scenarios

- [reference-react-vite](../reference-react-vite/) — First reference scenario (golden react-vite implementation)
- [prompt-manager](../prompt-manager/) — Skill source (DTV reads skills from its API)
- [ecosystem-manager](../ecosystem-manager/) — Uses the tools DTV validates
- [scenario-auditor](../scenario-auditor/) — Validated via tooling baselines
- [test-genie](../test-genie/) — Validated via tooling baselines
- [scenario-completeness-scoring](../scenario-completeness-scoring/) — Validated via tooling baselines

## Progress & Issues

- [Progress Log](docs/internal/PROGRESS.md)
- [Known Issues](docs/internal/PROBLEMS.md)
- [Integration Boundaries](docs/internal/SEAMS.md)
