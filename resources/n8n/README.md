# n8n Workflow Rehearsal Runtime

This is a task-scoped n8n resource for F7 client workflow rehearsal. It does not
run as one shared n8n service. `vrooli resource n8n runtime start` creates one
container for a Workspace Sandbox UUID, with an owner-generated runtime ID, a
private Docker `internal` network, loopback-only editor/API access, and bounded
ephemeral `tmpfs` storage. Workflow files, credentials and execution history
disappear when that task container stops.

Workspace Sandbox calls the existing control-plane command tree:

```bash
vrooli resource n8n runtime acquire --json
vrooli resource n8n runtime start --sandbox-id <sandbox-uuid> --json
vrooli resource n8n runtime attach-companion --sandbox-id <sandbox-uuid> --runtime-id <runtime-id> --companion-runtime-id <sandbox-runtime-id> --json
vrooli resource n8n runtime detach-companion --sandbox-id <sandbox-uuid> --runtime-id <runtime-id> --companion-runtime-id <sandbox-runtime-id> --json
vrooli resource n8n runtime import-workflow --sandbox-id <sandbox-uuid> --runtime-id <runtime-id> --operation-id <uuid> [--postgres-companion-runtime-id <sandbox-runtime-id>] --json < sanitized-workflow.json
vrooli resource n8n runtime execute-workflow --sandbox-id <sandbox-uuid> --runtime-id <runtime-id> --operation-id <uuid> --json
vrooli resource n8n runtime stop --sandbox-id <sandbox-uuid> --runtime-id <runtime-id> --json
```

`import-workflow` reads the workflow document from stdin, capped at 10 MiB; it
does not accept a path or arbitrary CLI command. The operation UUID is the
idempotency key used to derive the only workflow ID the owner will import and
later execute. `execute-workflow` accepts that operation UUID, not a workflow
ID, and applies a fixed five-minute execution deadline. Both commands return
bounded owner receipts/output, never raw or synthetic credential values.

`acquire` is the only image-pull path. It uses the shared capacity disk owner on
Docker's reported data-root filesystem, reserves a 3 GiB peak while retaining
at least 120 GiB free, heartbeats the admission lease for the complete pull,
verifies the exact digest, and returns the admission receipt. `start` never
pulls. If the image is absent, it fails with an instruction to run `acquire`.

The runtime accepts only nonzero Sandbox UUIDs and owner-issued runtime IDs.
Companion attachment requires a running container with Workspace Sandbox labels
for the same Sandbox UUID, the exact companion runtime ID, and a declared
companion marker. Before attach, Workspace Sandbox must keep the companion at
`--network none` (Docker may report the non-egress `none` attachment); the n8n
owner refuses `bridge`, `host`, or any other additional network. The n8n owner
generates the Docker network alias. It accepts
no image, command, network, host path, or credential from the caller. A missing
or foreign companion is denied without changing the network. Stop removes only
objects whose resource, Sandbox, runtime and object-kind labels match; it
detaches verified companions but never stops them. Unattested network members
are retained and surfaced as a cleanup error.

The n8n container has no host filesystem mounts, no Docker socket, no Linux
capabilities, a read-only root filesystem, no-new-privileges, a 512-process
limit, CPU and memory limits, and bounded temporary filesystems (1 GiB for
n8n's user data and 256 MiB for `/tmp`). Its only network is the per-Sandbox
internal network. Only declared companion containers join it, which blocks
direct internet and arbitrary LAN access while allowing the synthetic service
APIs used in the rehearsal.

The typed owner import path rewrites workflow credential references to
deterministic, Sandbox-scoped stand-in IDs and imports only fixed synthetic
credential data for its explicit allowlist. It never accepts exported
credential values. Unsupported credential types fail closed. Postgres
stand-ins require the declared companion to be attached to this task network;
the Workspace Sandbox companion must use the synthetic database name and
credentials defined by the owner. OpenAI, Pinecone, GitHub, Notion, Supabase,
WooCommerce and HTTP auth values are inert, and outbound service access is
blocked by the internal network. Only backend code may call the privileged
Server CLI broker because n8n's Server CLI bypasses n8n's access controls.

Do not import client credentials, client records, or live webhook destinations.
Make sure all node parameters, expressions, code and webhook destinations are
synthetic before import. The resource replaces credential references, but it
cannot infer whether arbitrary workflow values came from a client or make every
workflow safe to execute.

## Image and licensing

The image is pinned to n8n 2.48.0 at OCI index digest
`sha256:416c29f75a0f9ac65b4339761a372af48f4a7e948de9df0d837b080d2b9d47a8`.
Registry metadata reported 318,785,581 compressed layer bytes for amd64 and
317,934,581 for arm64. The 3 GiB acquisition peak is an estimate that includes
download staging, unpacking and container-store overhead; actual image size is
returned after acquisition and must be reconciled in the F7 run checkpoint. The
pinned image is retained and reused while this manifest references its digest.
Only the n8n owner may remove it, after all task runtimes are stopped and a newer
digest replaces it or the resource is retired.

n8n describes its community source as fair-code under the Sustainable Use
License, not OSI open source. Its license permits internal business use and
commercial consulting services, while restricting providing the software to
others. F7 uses n8n internally to rehearse client workflow deliverables; it
does not host or resell n8n. Review the exact [n8n community license terms]
(https://github.com/n8n-io/n8n-docs/blob/main/docs/n8n-community-license/README.md)
before changing this use or distributing the runtime.

## Current limits

- Linux amd64 and arm64 are the currently declared targets. macOS and Windows
  remain unsupported until the resource can verify admission against the actual
  Docker Desktop VM disk rather than a host path with the same name.
- No image pull or container start has been performed in this delivery. The
  control-plane owner API, not an independent Docker command, owns runtime
  acquisition and lifecycle.
- Workspace Sandbox must label companion containers with
  `com.vrooli.workspace-sandbox.sandbox-id`,
  `com.vrooli.workspace-sandbox.runtime-id`, and
  `com.vrooli.workspace-sandbox.companion-declared=true`, and call the n8n
  resource commands with the returned runtime IDs. That integration is owned
  separately.
