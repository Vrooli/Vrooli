# scenario-manifest

Exposes the canonical typed scenario service manifest (`.vrooli/service.json`) to
scenario modules (Go package `scenariomanifest`), which cannot import the root
`internal/scenario` package.

Governed by `.vrooli/package.json`; see [package governance](../../docs/package-governance.md).
