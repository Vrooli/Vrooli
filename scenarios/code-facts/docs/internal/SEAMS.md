# Seams

| Seam | Production Wiring | Test Fake | Why It Exists |
|---|---|---|---|
| Target filesystem | OS filesystem rooted at repo/target | Fixture FS | Resolve targets without global state. |
| Scenario metadata reader | `.vrooli/*`, endpoint JSON, CLI manifest | Fixture metadata | Inventory surfaces deterministically. |
| Go graph provider | go-code-graph Connect client | Fake provider | Avoid live provider dependency in unit tests. |
| TS graph provider | typescript-code-graph Connect client | Fake provider | Avoid sidecar/provider dependency in unit tests. |
| Cache store | SQLite/filesystem store | In-memory store | Validate key/invalidation behavior. |
| Clock | Real clock | Fixed clock | Deterministic cache timestamps. |

## Architecture Alignment Notes

Code Facts owns brokering and proof synthesis. Provider scenarios own parsing. Consumer health scenarios own policy.
