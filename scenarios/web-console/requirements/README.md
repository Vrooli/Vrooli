# Web Console Requirements Registry

This directory contains the requirements registry for web-console, tracking operational targets defined in the PRD.

## Structure

```
requirements/
└── index.json                    # All operational targets from PRD
```

## Coverage

Run coverage reports with:

```bash
# JSON output (for CI/automation)
node ../../scripts/requirements/report.js --scenario web-console --format json

# Markdown output (for README badges)
node ../../scripts/requirements/report.js --scenario web-console --format markdown

# Auto-sync from test results
node ../../scripts/requirements/report.js --scenario web-console --mode sync
```

## Operational Targets

### P0 – Must ship for viability (6 targets)
- **WC-P0-001**: Session Lifecycle API
- **WC-P0-002**: Configurable Default Commands
- **WC-P0-003**: Browser UI with Shortcuts
- **WC-P0-004**: Iframe Bridge Protocol
- **WC-P0-005**: Proxy Guard Enforcement
- **WC-P0-006**: Security Documentation

### P1 – Should have post-launch (4 targets)
- **WC-P1-001**: Concurrent Session Handling
- **WC-P1-002**: Prometheus Metrics
- **WC-P1-003**: Panic-Stop Endpoint
- **WC-P1-004**: Mobile-First UI

### P2 – Future / expansion (4 targets)
- **WC-P2-001**: Configurable Shortcut Palette
- **WC-P2-002**: Distributed Transcript Storage
- **WC-P2-003**: Voice Dictation/TTS Integration
- **WC-P2-004**: Parent Callbacks

**Total**: 14 operational targets (6 P0, 4 P1, 4 P2)

## Validation Strategy

All requirements link to one or more validation methods:

- **test**: Go unit tests in `api/*_test.go`
- **manual**: UI verification via manual testing

Test tags like `[REQ:WC-P0-001]` automatically update requirement status when tests run.

## PRD Linkage

Each requirement in `index.json` includes a `prd_ref` field that links back to the operational target in `PRD.md`:

```
"prd_ref": "Operational Targets > P0 > OT-P0-001"
```

This enables PRD Control Tower to:
1. Parse operational targets from the PRD
2. Match them to requirements in this registry
3. Track completion status via checkboxes
4. Surface coverage gaps in the Quality Scanner
