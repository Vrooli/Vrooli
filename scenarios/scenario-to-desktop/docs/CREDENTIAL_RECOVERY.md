# Desktop credential recovery

Desktop bundles use recovery option A: a customer can export and restore the
credentials declared by that bundle through the authenticated supervisor IPC
channel.

`POST /credentials/recovery/export` accepts a passphrase in the request body
and returns an encrypted bundle as base64. `POST /credentials/recovery/restore`
accepts that bundle and the same passphrase. The server scopes both operations
to the bundle manifest, writes recovery metadata without values, and records
only an entry count in telemetry.

The customer must keep the encrypted bundle and passphrase separately and
outside the application data directory. The runtime never embeds a credential
value in `bundle.json`; generated manifests carry only the declared logical ID,
field, and provisioning metadata. A secret that lacks declaration-backed
identity is assigned a private fallback namespace and emits a warning during
manifest generation so it cannot remain an invisible seam.

This choice is intentionally limited to recovery. A downloadable bundle is
not allowed to carry secret values: every copy would contain the customer's
credential. A future signed per-customer distribution channel would need a
separate threat model before this decision could change.
