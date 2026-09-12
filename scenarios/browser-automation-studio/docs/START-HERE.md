# Start Here

Browser Automation Studio owns visual workflow authoring, recorded browser interaction, deterministic replay evidence, and automated end-to-end validation for Vrooli scenarios.

## Initialization Protocol

Start the scenario with `make start` or `vrooli scenario start browser-automation-studio`; never launch API or driver binaries directly. Use `vrooli scenario test browser-automation-studio` for the server-owned suite and wait on its returned run once.

## Architecture Rules

The API owns workflow validation, persistence, and orchestration. The Playwright driver owns browser sessions and typed instruction execution. Proto definitions are the cross-language contract. UI code may not recreate retired legacy instruction shapes.

## Replacing The Example Domain

New browser capabilities begin as typed proto actions, then gain API compilation, driver execution, UI authoring, and deterministic coverage. See [Architecture](concepts/ARCHITECTURE.md) and [Domains](concepts/DOMAINS.md).
