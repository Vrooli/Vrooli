# desktop-component

Owns the signed, engine-independent optional desktop component contract. Release
production (scenario-to-desktop), delivery (landing-page-business-suite) and runtime
admission all verify components against this one definition.

## Same-machine full app precedence

Packaged desktop apps for the same user share local peer records under the
project config directory (`peers/<scenario>.json`). A standalone app publishes
its live scenario identity, version, ports and declared features there. A
bundling app reads that record before starting an embedded peer or optional
slice: a compatible full app owns the request and the bundled copy stands down.
When the standalone app exits, a later feature lookup may restart the compatible
bundled provider. The record is same-user local discovery; it does not make a
slice a full scenario, route a request to another machine, or let one app replace
another live process's record.

An app that encounters a live record must leave it intact. The standalone app
and the bundling app therefore communicate through the same local peer contract
while preserving each one's own lifecycle and user data.

Governed by `.vrooli/package.json`; see [package governance](../../docs/package-governance.md).
