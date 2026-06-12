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

The Phase 6 CLI contract will expose:

```bash
code-facts describe scenario:proto-health --include surfaces,proto_adoption --json
```

Until Phase 6 lands, this scenario remains in documentation-first orientation.

## 5 — Run the tests

```bash
cd scenarios/code-facts
make test
```

Known Phase 5 scaffold caveat: the generated template still contains the `notes` example domain until the implementation phases replace it with Code Facts domains.
