# cliresolve

Resolves installed Vrooli command-line programs without depending on a repository
checkout or on the installer that produced them. The control plane, harness lifecycle and
vrooli-bridge share it, so every caller finds a CLI the same way.

Governed by `.vrooli/package.json`; see [package governance](../../docs/package-governance.md).
