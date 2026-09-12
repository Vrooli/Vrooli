# Vrooli Authorization

This document defines the project-wide authorization boundary. It complements
[`IDENTITY-AND-AUTHENTICATION.md`](IDENTITY-AND-AUTHENTICATION.md), which
defines identity, providers, desktop modes, account linking, and entitlements.

## Four decisions, four owners

```text
Authentication      Who is this principal?
Capability          May this principal use this scenario/effect?
Domain authorization May this principal act on this exact object?
Entitlement         Has this customer paid for or activated this feature?
```

These decisions may inform one another, but they must not be collapsed into a
single role or token claim.

| Decision | Owner |
|---|---|
| Principal authentication | `scenario-authenticator` or explicitly configured provider |
| Scenario capability grants | Authenticator assignment plus `api-core/scopecatalog` |
| Resource/object authorization | Relying scenario |
| Agent-run attenuation | `agent-manager` |
| Commercial entitlement | LPBS or declared product commerce authority |

## Administrative boundaries

An administrator is an actor with authority in a particular control plane,
not a universal bypass:

| Control plane | Administrative decisions | Where enforcement lives |
|---|---|---|
| Identity | User lifecycle, credentials, MFA, sessions, roles, and coarse capabilities | `scenario-authenticator` |
| Product | Business accounts, billing, downloads, entitlements, and product data | LPBS or the relying scenario |
| Installation | Provider selection, deployment mode, secrets, backups, and service lifecycle | Vrooli control/deployment surfaces |

The same person may be an administrator in more than one control plane. A
relying party must preserve the actor context when it asks the identity
provider to perform an administrative operation. The target flow is:

```text
LPBS/product admin session
        -> verify actor and local product permission
        -> resolve scenario-authenticator by api-core/discovery
        -> authenticated API-to-API management call
        -> authenticator checks identity-management capability
        -> audit actor, target, operation, result
```

The browser does not call the authenticator management API directly. A
product administrator must not gain identity-admin authority merely because
the product has an administrator role. The product-to-authenticator mapping
must be explicit, scoped, revocable, and auditable.

Identity-management operations such as unlock, credential-reset initiation,
MFA reset, session revocation, and capability assignment belong in the
authenticator's management contract. If a management operation is not yet
implemented, the relying party must expose it as unavailable rather than
recreating the operation against authenticator storage.

## Capability vocabulary

The current coarse vocabulary is:

```text
<scenario>:<effect>
```

Supported effects are:

- `read` — observe or retrieve data;
- `write` — create or modify non-destructive state;
- `destructive` — delete, publish, execute, revoke, or otherwise high-impact state.

Grants are additive. There are no deny rules in the coarse scope grammar. An
absent scope never grants authority. An issued token must carry an explicit
scope list, including an empty list when the principal has no capabilities.

The catalog is derived from scenario CLI and service declarations. The
authenticator stores grant values without importing scenario code or interpreting
scenario commands. This keeps the provider independent from relying-party
release cycles.

## Enforcement sequence

Every protected mutation follows this order:

1. Verify the presented provider credential.
2. Verify issuer, algorithm, key id, audience, expiry, and revocation state.
3. Resolve the canonical principal from request context.
4. Check the required coarse capability.
5. Apply relying-party domain authorization.
6. Apply exact approval, human-control, or entitlement rules when required.
7. Audit the decision and the mutation result.

The UI and CLI may explain or hide unavailable actions. They never make the
authorization decision.

## Scope is not object authorization

`git-control-tower:write` may permit a principal to reach a Git mutation
surface. Git Control Tower must still decide whether:

- the repository is in scope;
- the current branch is protected;
- the operation requires a human;
- the actor is an allowed agent or service;
- the requested change has the required evidence or approval.

Similarly, `device-sync-hub:read` does not automatically grant access to every
device in every tenant.

## Agent delegation

An agent run receives the intersection of:

```text
owner grants ∩ agent profile grants ∩ requested grants
```

The child token or provenance record must be narrower than its parent and must
expire no later than its parent. Human-only capabilities are not materialized
into agent grants unless an explicit reviewed policy allows them.

## Commercial entitlements

An LPBS entitlement may determine whether a product feature is available, but
it does not replace technical authorization. A desktop app may need both:

```text
valid local principal
    + required scenario capability
    + valid commercial entitlement
```

The entitlement must be scoped to a product, installation or audience, and
feature set. It must have an explicit expiry and offline behavior.

## Desktop modes

`personal_local` uses the OS user and private application boundary. It does
not require human sign-in by default. `local_multi_user` requires a local
identity provider and explicit account selection. `remote_vrooli` uses the
remote server's provider. `shared_provider` uses a broker-issued expiring
lease. See the canonical identity document for the full deployment contract.

## Relying-party implementation rule

Relying parties must use the shared provider-neutral identity and authentication
packages. They must not:

- parse JWTs independently;
- trust identity fields from request bodies;
- call authenticator validation on every request;
- treat email equality as account linking;
- treat UI visibility as authorization;
- issue broader child credentials than the verified parent;
- use a commercial entitlement as a substitute for domain policy.

The scenario-specific scope and authorization details remain in
[`../../scenarios/scenario-authenticator/docs/concepts/AUTHORIZATION.md`](../../scenarios/scenario-authenticator/docs/concepts/AUTHORIZATION.md).
