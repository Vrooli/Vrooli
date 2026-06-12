# Quickstart

## Prerequisites

- Vrooli repository setup is complete.
- `code-facts` is installed through the scenario lifecycle.
- Future implementation phases require `go-code-graph` and `typescript-code-graph` running for provider-backed facts.

## 1 — Setup

```bash
cd scenarios/code-facts
make setup
```

## 2 — Start

```bash
cd scenarios/code-facts
make start
```

## 3 — Open

Use `make status` to find the assigned UI and API ports. The UI is the operator workbench for target inspection, fact-family filtering, evidence, warnings, and cache status.

## 4 — First Describe

The Phase 6 CLI contract exposes:

```bash
code-facts facts describe scenario:code-facts --include surfaces,parse_units,proto_adoption --json
```

Analyzer-backed facts return typed `unsupported` evidence until the target resolver and provider broker land in later phases.

## 5 — Run the tests

```bash
cd scenarios/code-facts
make test
```

The generated `notes` example domain has been removed; active scenario surfaces are health plus the Code Facts `facts` and `cache` contracts.
