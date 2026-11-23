# binaryfetch

This governed Go package is owned by the Vrooli repository. Its package
manifest is the machine-readable adoption and lifecycle record.

Acquisition manifests declare credential references, never values:
`"auth":{"scheme":"bearer","logical_id":"vrooli/model","field":"huggingface-token"}`.
The resource owner validates the reference against its credential declarations,
resolves it only when download is required, and passes transient authorization.
Bearer requests require HTTPS, attach only to the exact origin, and strip the
header on redirects to other origins (including subdomains/ports). A cloned HTTP
client preserves caller redirect policy; no global transport is replaced.
Authenticated remote error text is withheld because it may contain credentials or
signed URLs; HTTP status and cancellation identity remain diagnostic.

`Fetch` retains its executable size/HTML guard. `FetchData` is the explicit
checksum-required non-executable file path, allowing small JSON metadata while
refusing empty/HTML payloads. `InspectData` is separate discovery: at most1MiB,
expected Git blob SHA-1, new0600 file only, SHA-256/bytes/origin receipt. Discovery
never substitutes Git SHA-1 for mandatory executable/model manifest SHA-256 pins.

The committed acquisition fragment is generated from `schemagen.go`. From this
package directory, use the maintained owner command:

```sh
go run ./cmd/acquisition-schema --root /absolute/path/to/Vrooli --check
go run ./cmd/acquisition-schema --root /absolute/path/to/Vrooli
```

Run the check before consumers. Regenerate only after changing the owning types
and schema source; never hand-edit `.vrooli/schemas/acquisition.schema.json`.

Managed composed URL archives carry exact `download_bytes` and positive
`max_expanded_bytes`/`max_entries` on each compose step. The shared capacity
planner sums those per-step ceilings and reserves
`3 * (sum(download_bytes) + sum(max_expanded_bytes)) + 1 MiB`. `FetchTree`
enforces the same exact download and extraction limits. Composed Python-wheel
and local-copy steps remain unsupported by managed capacity admission until
their owners can enforce an equivalent write ceiling.
