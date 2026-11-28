# capabilityprobe

Probes host capabilities without importing the control plane. It is deliberately
standalone so the Bridge node agent (`scenarios/vrooli-bridge/agent`) can cross-compile it
for every supported OS and architecture; web-console reads the same probes.

Governed by `.vrooli/package.json`; see [package governance](../../docs/package-governance.md).
