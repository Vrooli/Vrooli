# n8n resource CLI ownership

This resource intentionally does not ship an independent command binary. Its
task-scoped runtime is exposed by the existing `vrooli resource n8n runtime ...`
control-plane command tree, which validates the manifest image and owns image
admission, container labels, private networks, loopback binding, and cleanup.
Keeping resource-local CLI disabled prevents a second path around that owner.
