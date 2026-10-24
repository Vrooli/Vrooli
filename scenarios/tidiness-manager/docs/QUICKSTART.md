# Quickstart

## Prerequisites

- Vrooli development environment is set up.
- PostgreSQL resource is available through lifecycle configuration.
- Optional analyzer tools can improve scan depth, but the light and tidiness scans degrade gracefully when they are absent.

## 1 — Setup

```bash
cd scenarios/tidiness-manager
make setup
```

## 2 — Start

```bash
cd scenarios/tidiness-manager
make start
```

## 3 — Open

Use lifecycle status to find the assigned UI/API ports:

```bash
cd scenarios/tidiness-manager
make status
```

Open the UI URL reported by lifecycle. The dashboard shows scenario summaries, issues, file metrics, and campaign state.

## 4 — Use

Run a maintainability scan:

```bash
tidiness-manager scan quality-health --type tidiness
```

Inspect findings:

```bash
tidiness-manager issues list quality-health --limit 20
tidiness-manager recommend-refactors quality-health --limit 10
tidiness-manager score quality-health
```

## 5 — Run the tests

```bash
test-genie execute tidiness-manager --phases tidiness --json
test-genie execute tidiness-manager --phases quality --json
```

For the full scenario suite, use:

```bash
vrooli scenario test tidiness-manager
```
