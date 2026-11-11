# n8n resource CLI ownership

This resource intentionally does not ship an independent command binary. Its
task-scoped runtime is exposed by the existing `vrooli resource n8n runtime ...`
control-plane command tree, which validates the manifest image and owns image
admission, container labels, private networks, loopback binding, and cleanup.
Keeping resource-local CLI disabled prevents a second path around that owner.

The control-plane exposes only typed operations. Workflow import reads one
bounded document from stdin; execution selects the workflow from its Sandbox,
runtime, and operation UUID. Neither operation accepts a caller path, command,
or workflow ID. Import replaces supported credential references with fixed
synthetic stand-ins, and both operations keep n8n's privileged Server CLI
backend-only.

`dispatch-webhook-event` takes the same Sandbox/runtime/operation identity as
the import, plus `--event-stdin`; it has no route, host, port, URL, or workflow
ID flags. It re-exports the owner-derived inactive workflow with fixed
`export:workflow`, validates one static unauthenticated POST Webhook trigger
and the restricted action-node profile, then calls fixed `publish:workflow`,
posts one bounded JSON event to the inspected loopback endpoint, and calls
`unpublish:workflow` on every terminal path. Publication polling is explicitly
enabled for the runtime and the owner waits longer than n8n 2.48.0's pinned
15-second publication outbox poll interval. HTTP Request nodes are accepted
only for `http://*.synthetic.invalid:18080`; redirects are not followed. The
Webhook must use `lastNode` response mode so dispatch does not unpublish before
the synchronous webhook response completes. Unknown node types fail closed.
Receipts expose only hashes, byte counts, HTTP status, and cleanup status. With
n8n 2.48.0's publication service, `unpublish:workflow` commits an outbox
request; it does not prove that route teardown has completed. The receipt
therefore reports `publicationState: "unpublish-requested"` and conservatively
keeps `published: true` until the disposable runtime is stopped. An unpublish
CLI failure reports `publicationState: "unpublish-failed"` and
`cleanupRequired: true`.

Execution receipts expose only output digest/size and aggregate node,
execution, and item counts. They never return n8n output values, node names, or
runtime logs; unsupported output shapes are rejected.

`execute-manual-workflow` exports the operation-derived workflow first and only
executes when its active trigger profile is exactly one built-in Manual
Trigger. Its fixed node profile excludes Schedule/Webhook/other triggers,
custom and Code nodes, and requires HTTP Request nodes to use the synthetic
HTTP mock. It then invokes the pinned Server CLI `execute --id <derived-id>
--rawOutput`; the receipt omits the workflow ID. n8n 2.48.0's CLI start-node
selection includes `n8n-nodes-base.manualTrigger` but not Schedule Trigger.
Schedule execution therefore remains unsupported here; activating a schedule
would require a separate bounded timer lifecycle, which this operation does
not create.

The separate `execute-workflow` command remains for CLI-startable workflows:
in pinned n8n 2.48.0, `execute --id` selects a CLI start node rather than a
Webhook trigger. Webhook imports remain inactive until the typed event
operation validates and publishes them. Test URLs still require editor listener
registration and are not used by the owner.

The same owner exposes only `companion-start` and `companion-stop` for the
synthetic Postgres profile. Start accepts the fixed name/version/database tuple
(`pg`, `16.4`, `workflow`), checks the already-installed digest-pinned
`postgres:16.4-alpine` image, and fails without pulling if it is missing. It
creates the container with `--network none`, verifies the Sandbox-owned labels,
then attaches through the private n8n task network owner. Stop accepts the
Sandbox UUID and owner-issued companion runtime ID; it detaches and removes
only a matching labeled container and its anonymous volumes.

For HTTP fixtures, `companion-start-http-mock --sandbox-id <UUID>
--config-stdin --json` reads at most 2 MiB from stdin. It independently
validates fixtures, starts one fixed Node source server in the already-running
n8n task's private internal network using the pinned n8n image, and creates
only the declared `*.synthetic.invalid` network aliases. There is no upstream
forwarding path, caller-provided script, host port, image, path, or command.
`companion-stop-http-mock` takes the Sandbox UUID and returned owner runtime ID
and removes only the matching labeled container. Both responses are bounded
and omit fixture response bodies.

Each HTTP fixture may include an optional exact `query` string. Resource-side
validation and matching mirror Workspace Sandbox canonical form encoding: no
duplicate keys, one value per key, at most 32 keys, 2 KiB raw/canonical size,
512-byte values, and rejection of malformed encoding, sensitive keys,
credential-shaped values, or non-synthetic emails. Key order does not change
the exact route; wildcard or query-template matching is unsupported.
