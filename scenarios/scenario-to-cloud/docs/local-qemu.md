# Local QEMU qualification target

Compute Manager owns local-QEMU guest creation, capacity admission, process and disk receipts, restart recovery, and release. Scenario-to-Cloud holds a typed Compute Manager instance reference and verified SSH locator for deployment work; it does not create a second guest or provider record.

The governed `cloud-qemu-qualification` program acquires an isolated guest through Scenario-to-Cloud's generated QEMU Connect service when the acquire branch is selected. Compute Manager records the provider quote and host-capacity receipt before its local-QEMU adapter can create a disk or process. Qualification resolves that same manager-owned reference before reading or changing the deployment.

Snapshot and reset are optional Compute Manager test-provider capabilities. They are not part of Scenario-to-Cloud's provider interface. Production cloud providers do not need to implement them.

`GET /api/v1/instances/readiness` remains a read-only host observation for the lane. It reports declared QEMU tools, KVM availability, recorded image presence and digest state. It never installs tools or starts a guest. Host setup and image readiness must be handled by the declared `vrooli setup` path; clean-host setup closure is still pending.

The 54-cell QEMU lane has not been run through the replacement path. It remains gated on owner review of the real capacity receipts, lease heartbeat and recovery behavior. No QEMU guest, process or capacity lease is started by reading this document or checking readiness.
