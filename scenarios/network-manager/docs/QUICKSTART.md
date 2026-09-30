# Quickstart — Network Manager

Network Manager is currently a generated scaffold with completed PRD, requirements, and docs. Product implementation has not started.

## Prerequisites

- Vrooli development environment.
- Generated scenario dependencies from the React/Vite template.
- Future P0 implementation will need AdGuard Home or an AdGuard Home resource.

## 1 — Setup

```bash
cd scenarios/network-manager
make setup
```

## 2 — Start

```bash
make start
make status
```

## 3 — Open

```bash
vrooli scenario open network-manager
```

## 4 — Current scope

This session intentionally stops at scaffold, PRD, requirements, and docs. The generated UI/API/CLI still contain template example behavior until implementation work replaces it.

## 5 — Run the tests

```bash
make test
```

For long scenario tests, use the test-genie wait guidance in the repository instructions.

## Next implementation slice

Build the first real vertical slice:

1. Adapter capability registry.
2. Read-only network health snapshot with fake probes.
3. UI/CLI snapshot report.
4. `[REQ:NM-P0-001]` tagged tests.
5. Remove template example domain after a real domain is green.
