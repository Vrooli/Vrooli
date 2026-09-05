# Reach and configuration architecture

Vrooli addresses every scenario call with two independent decisions: the
scenario name and its target. `local` addresses this machine; a registered
node id addresses a remote machine through Bridge. Callers use the same typed
client shape after resolving the target. Local resolution remains a cheap
discovery ladder, while remote resolution is a Bridge-mediated transport.

For a registered target, the resolver returns a Bridge proxy base URL and the
typed client appends its normal `/<service>/<method>` procedure path. Bridge
admits that exact method against the scenario CLI catalog, signs a bounded
`ScenarioRequest` frame, and waits for the node's bounded `ScenarioResponse`
over the authenticated Presence RPC. The proxy is therefore not an ungoverned
shell relay.

## One configuration document

A versioned configuration document describes what a machine should be. Bridge
keeps the desired copy for each Machine. Onboarding writes the applied copy on
the target. Drift is the computed difference between them; editing the target
does not silently replace intent. A preset is a named starting point that
expands into a document. `environment` is an expansion choice, not a persisted
wire field.

The crossing document remains capability-shaped: Bridge does not depend on
Onboarding's persistence schema, and Onboarding remains the authority for
operator state and host application.

## Decisions and boundaries

* `Selection` stays capability-shaped so it can cross a deployment boundary
  without coupling Bridge to Onboarding storage. The handoff is the supported
  configuration path after pairing.
* Bridge owns reaching a node, while Onboarding owns operator state. The same
  boundary applies during first touch and after pairing.
* The node axis belongs beside reach, not in a CLI-only transport seam. The old
  `discovery/transport.go` deletion was correct when the reach client lived in a
  higher module that `api-core` could not import without a cycle. The client now
  lives in `packages/api-core/nodereach`, so discovery exposes one target-aware
  resolver without reintroducing the cycle.

## Credential delivery

Credential values are delivered through grants and sealed pushes, never through
CLI arguments or ordinary relay input. Bridge authorizes the grant, seals the
value to the node key with context binding, and records the signed receipt;
the node opens and zeroes plaintext. Operators reach this flow through Bridge
and the Web Console machine surface.

Implementation references:

* [CODE: packages/api-core/discovery/resolve.go]
* [CODE: scenarios/vrooli-bridge/api/internal/onboarding/client.go]
* [CODE: scenarios/vrooli-bridge/api/handlers/credentialgrant/module.go#deliverGrant]

See also [the port-resolution reference](../reference/port-resolution.md) and
[the onboarding boundary](../../scenarios/vrooli-bridge/docs/concepts/ONBOARDING-BOUNDARY.md).
