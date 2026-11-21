# Task-private virtual API gateway contract

`virtualapi.New(Contract)` validates a finite set of exact public API hostnames
and exact method/path/query fixtures, then issues an ephemeral task CA and a
server certificate with only those hostnames in its SANs. `Gateway.ServeHTTP`
returns only fixture data and has no dialer, resolver override, proxy, or
forwarding path. `Gateway.Close` drops references to fixtures, TLS keys, and
route counters. `Receipt.CACertPEM` is public trust material; private key PEM is
available only from `ServerCertificate` for listener startup.

`Gateway.ServerConfigJSON()` produces a deterministic, size-capped config for
the fixed `ContainerServerProgram`. The config includes fixture bytes and the
TLS leaf key, but never the task CA private key. The owner stages both config
and program only in task tmpfs. The Node adapter checks the config bound and
fixture shape again, binds HTTPS on port 443, rejects undeclared SNI/Host and
unmatched routes, bounds request and response bodies, and stores only aggregate
route counters in task tmpfs. It has no client request or forwarding code.

This package is the HTTPS/profile seam, not a container or OS-network owner.
Its receipt deliberately sets `NoEgressVerified=false`. The lifecycle owner may
set that field only after it proves the gateway and workflow runtime share the
same task network, that the network is internal-only, the workflow container
resolves each exact alias to the gateway, and unrelated outbound addresses fail.
`requiredNetworkPolicy` records the required policy; it is not proof.

The task CA has to be trusted before an unmodified Node.js workflow process
starts. Integration must arrange `NODE_EXTRA_CA_CERTS` (or an equivalent
task-local trust bundle) at runtime creation and remove it with the task. An
already-running n8n process cannot be made to trust a newly issued CA by adding
an environment variable to an owner-side Docker exec. Do not fall back to
disabled certificate verification or host-wide trust installation.

Expected provider sequence:

1. Validate and normalize the immutable contract; derive a runtime identity
   from task ID and profile fingerprint.
2. Generate the CA and gateway cert, stage public CA trust in the task-local
   n8n runtime before its Node process starts, and start the pinned gateway on
   the task's internal-only network with each exact alias. HTTPS hostnames
   without explicit ports require port 443; an unprivileged gateway process
   therefore needs a task-network-namespace-only low-port sysctl or an
   equivalent narrowly scoped bind mechanism. Do not publish that port to the
   host or add broad capabilities to solve the bind.
3. Verify gateway aliases, SNI/Host routing, private network membership,
   internal-only network mode, and no outbound path before returning a receipt
   with `NoEgressVerified=true`.
4. Read aggregate route hits without storing request headers/bodies. On stop,
   remove gateway and task-local trust material, then verify both are absent.

No Docker lifecycle was exercised by this package's tests.
