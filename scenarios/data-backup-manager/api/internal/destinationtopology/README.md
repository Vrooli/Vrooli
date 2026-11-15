# Destination topology evidence

`Compare` is a pure, conservative comparator. It does not inspect the OS, resolve
mount paths, parse URLs, or infer physical placement from backend names.

Adapters populate `Evidence` from authoritative sources:

- `HostID`: a stable, confirmed computer or fleet-node identity. For a local
  filesystem destination, use the current host identity. For a remote-host
  endpoint, use the destination server's confirmed identity. Do not use a
  hostname string unless the owner has established that it uniquely identifies
  the physical host.
- `DeviceID`: a stable physical device identity from the platform device
  resolver. Do not use a drive letter, mount point, repository path, or volume
  label. A logical volume spanning multiple devices needs an explicit owner
  decision about its failure domain; leave it empty until established.
- `Endpoint`: `local` for storage on this host; `remote_host` only with a
  confirmed remote host identity; `provider_hosted` only when evidence confirms
  the endpoint is hosted by the storage provider; otherwise `unknown`.

An S3-compatible backend may be a local MinIO endpoint, a remote server, or a
provider service. Backend kind alone must map to `unknown`; the endpoint owner
must supply facts that distinguish those cases. A provider-hosted endpoint is
independent of a confirmed local host. Two provider endpoints cannot be called
independent from this pairwise evidence alone; keep the relation unknown unless
an owner supplies stronger host identities.

Comparison rules: matching known host and device is `same_device`; matching
host with any other device combination is `same_host`; distinct confirmed host
identities are `independent`; confirmed provider-hosted versus confirmed local
host is `independent`; all other cases are `unknown`. `same_host` is not
independent for whole-computer loss, even when devices differ.
