# deployability

Exposes the control-plane deployability contracts (`internal/deployability`) to scenario
modules as type aliases, because Go's internal rule forbids scenario modules from importing
them directly. Ownership stays in `internal/deployability`; `TestAliasSurfaceIsComplete`
fails when an exported internal type has no alias here.

Governed by `.vrooli/package.json`; see [package governance](../../docs/package-governance.md).
