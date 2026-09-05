# Standing approvals

Standing approvals are host-local operator decisions for conditional recovery
providers. They permit only the named provider and its bounded subject; they do
not authorize arbitrary commands, paths, Docker volumes, or scenario data.

The broker validates every request again before execution. If an approval is
absent, storage-manager reports the provider as withheld and continues through
the safe, regenerable, and owner-budgeted rungs.

Approvals are recorded through the storage-manager management surface on the
current host. Read them with `GET /api/v1/cleanup/approvals`, create one with
`POST /api/v1/cleanup/approvals/{provider}`, and revoke one with
`DELETE /api/v1/cleanup/approvals/{provider}`. The request must include
`approved_at`, `approved_by`, and the current host's `host_id`. An approval is
never copied to another host.
