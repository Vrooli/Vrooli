# Credential delivery reference

Bridge exposes credential delivery as a consented, node-bound flow. An
operator creates a grant for a logical credential and field. When the node is
online, Bridge sends signed grant metadata, resolves the value through the
credential authority, seals the value to the node's X25519 public key with
AAD binding, and pushes the sealed frame. The node verifies and opens the
frame, writes the credential, and zeroes the plaintext. A signed receipt and
the grant metadata remain available for audit and retry.

The value is never placed in a command argument, ordinary relay output, or
persisted grant metadata. A grant may be created before provisioning; delivery retries
when the node reconnects or the credential becomes available. A locked node
store is a typed operational failure: the operator must unlock the node's
credential store and retry, rather than receiving a misleading success.

Revocation is local and distributed state. Bridge denies future delivery immediately,
marks the grant's remote purge as pending, and records a node-specific purge receipt
when the node reconnects or receives the purge frame. A prior raw read can remain in a
running process, and revoking the grant does not rotate the upstream provider secret;
the operator must rotate the provider credential and restart affected workloads when
that residual exposure matters. `RotateAddress` reports `local_generation_only` for
the same reason: it advances Bridge delivery generations and makes stale frames fail
closed, but it does not claim that an upstream service rotated its credential.

Operator entry points are the Bridge credential CLI and the Web Console
machine detail surface. Both use the same Bridge API and show held-credential
metadata without revealing values.

Secret questions use the authenticated `AnswerSecret` operation. Bridge writes
the answer to its local credential authority and immediately calls the same
sealed `deliverGrant` path; the answer is not returned, logged, put on argv, or
stored in a configuration plan. Non-secret answers continue through the target
onboarding question endpoint.

Implementation references:

* [CODE: scenarios/vrooli-bridge/api/handlers/credentialgrant/module.go#deliverGrant]
* [CODE: scenarios/vrooli-bridge/agent/internal/channel/channel.go]
* [CODE: scenarios/vrooli-bridge/api/internal/onboarding/client.go]
