# Deployment secret strategy

This guide explains how deployment-manager classifies secrets for a target. It
does not define credential storage or runtime resolution. Those are owned by
the [credential authority](../../../../docs/configuration/secrets.md) and the
target ramp.

## Four deployment classes

| Class | Bundle action | Example |
| --- | --- | --- |
| `infrastructure` | Never ship; remove through a swap or keep the dependency remote | Database master password, Vault root material |
| `per_install_generated` | Generate on first run and store through the target credential authority | JWT signing key, local service token |
| `user_prompt` | Ask the user through a first-run or settings flow | External API key, SMTP credential |
| `remote_fetch` | Store only a reference and fetch at runtime | Organization-managed secret |

The classification belongs in the target plan. A secret that has no safe target
strategy is a deployment blocker, not a reason to omit the declaration.

## Rules

1. Infrastructure secrets must never appear in a desktop installer, bundle
   manifest, source archive, log, evidence artifact, or command argument.
2. Generated secrets must have an explicit generator and stable per-install
   identity.
3. User-prompted secrets must have clear copy, validation, optional/required
   status, and a documented storage route.
4. Remote-fetched secrets must declare the network and authentication
   requirements. They cannot support a full offline claim.
5. A runtime may expose secret presence and source metadata, but never the
   stored value.
6. Credential provider states must remain distinct: unconfigured, unavailable,
   and absent require different operator actions.

## Deployment-time checks

Before a target build, deployment-manager should be able to answer:

- Which declared credentials are required by the dependency graph?
- Which credentials are infrastructure-only and therefore excluded?
- Which values are generated locally?
- Which values require user input?
- Which values require a remote service?
- What does the application do when the value is unavailable?
- Is the resulting artifact eligible for an offline claim?

Use the CLI to inspect the plan:

```bash
deployment-manager secrets identify <profile-id>
deployment-manager secrets template <profile-id> --format json
deployment-manager secrets validate <profile-id>
```

These commands describe a deployment strategy. They do not print stored secret
values.

## Runtime behavior

The target runtime owns the user-facing provisioning flow. For desktop bundles,
see [desktop credential recovery](../../../scenario-to-desktop/docs/CREDENTIAL_RECOVERY.md)
and the [scenario-to-desktop overview](../../../scenario-to-desktop/docs/OVERVIEW.md).

The platform credential authority owns:

- native secure-store selection;
- encrypted headless-host fallback;
- on-demand resolution;
- provider diagnosis and repair;
- encrypted recovery export and restore.

Application credentials should not block the scenario process from starting;
the affected capability should report a named unhealthy state. Resource-native
bootstrap material is stricter: if it cannot be stored or recovered securely,
the resource must not be published as usable.

## Related references

- [Credential configuration](../../../../docs/configuration/secrets.md)
- [Resource deployment contract](../../../../docs/resources/deployment-contract.md)
- [Desktop deployment workflow](../workflows/desktop-deployment.md)
- [Bundle manifest schema](bundle-manifest-schema.md)
