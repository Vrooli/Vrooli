# envkit-go

The repository-wide process environment boundary (Go package `envkit`). An environment is an
exec boundary, not an ownership boundary, so relative variables are never inherited into a
process that belongs to another scenario or resource. Names are folded for the target platform.

Governed by `.vrooli/package.json`; see [package governance](../../docs/package-governance.md).
