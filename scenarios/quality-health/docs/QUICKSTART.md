# Quickstart

Quality Health is currently in foundation state. The scaffold exists, the product docs and requirements describe the intended static-quality authority, and the implementation domains are planned.

## Prerequisites

- Vrooli repository setup is complete.
- The Vrooli CLI is available on `PATH`.
- Generated scenario dependencies are present in the repository checkout.

## 1 — Setup

```bash
cd scenarios/quality-health
make setup
```

Do not install new dependencies manually. If setup fails, record the failure in `docs/internal/PROBLEMS.md` before changing package files.

## 2 — Start

```bash
cd scenarios/quality-health
make start
```

The scenario must be started through the lifecycle system.

## 3 — Open

```bash
cd scenarios/quality-health
make open
```

The generated UI is a placeholder until the Phase 3 audit workbench is implemented.

## 4 — Inspect Orientation

```bash
cd scenarios/quality-health
make orient
```

Orientation currently tracks scaffold health, charter, requirements, domain map, dependency decisions, sample-domain removal, and progress handoff.

## 5 — Run the tests

```bash
vrooli scenario test quality-health
```

Use the durable-run wait command printed by Test Genie if the run backgrounds or the shell times out. Do not start API/UI binaries directly.
