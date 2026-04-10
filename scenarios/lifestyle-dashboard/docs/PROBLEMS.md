# Known Issues and Limitations

## Current Limitations

### PROB-001: No Wearable Integration Yet
- **Status:** Deferred to P2
- **Impact:** Sleep tracker and activity data require manual entry
- **Workaround:** Design with dependency injection for future device support
- **Resolution:** Implement OT-P2-003 Hardware Integration Framework

### PROB-002: Single-Writer SQLite Constraint
- **Status:** By design
- **Impact:** All writes must go through API; no direct UI→DB writes
- **Workaround:** None needed — this is the intended architecture
- **Notes:** Enables future WAL mode and concurrent readers

### PROB-003: No Offline Support in P0
- **Status:** Deferred
- **Impact:** Dashboard requires network to fetch data
- **Workaround:** Morning/evening briefs designed to be screenshots/cached
- **Resolution:** Add service worker and local storage sync in P1

## Deferred Decisions

### Mobile App Type
- **Options:** PWA, React Native, Capacitor
- **Decision needed by:** P2 (when mobile becomes priority)
- **Current direction:** PWA for P0/P1, evaluate native for P2

### Wearable Device Selection
- **Options:** Oura Ring, Whoop, Apple Watch, Garmin
- **Decision needed by:** Before Sleep Tracker implementation
- **Current direction:** Oura Ring (sleep-focused data quality)

## Open Questions

### Event Schema Evolution
- How to handle breaking changes to event payloads?
- Current approach: JSON payloads are versioned implicitly; migrations handle schema changes

### Correlation Noise
- Risk of surfacing spurious correlations with limited data
- Mitigation: Configurable minimum threshold (default 14 data points), p-value requirements
