# scheduler

The shared wait-from-end primitive for periodic work. A ticker is wrong for filesystem walks:
it buffers a tick while a walk is still running and can keep the host under continuous metadata
load, so periodic work waits a full interval after each run ends.

Governed by `.vrooli/package.json`; see [package governance](../../docs/package-governance.md).
