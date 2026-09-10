# scenario-to-cloud

Deploy a single Vrooli scenario and its declaration-derived dependency closure to an enrolled cloud target. The managed path uses shared reach, the target-owned `cloud-target` control plane, and typed durable operations; a bounded SSH adapter remains available when the target is not Bridge-enrolled.

This scenario is designed to be invoked by `deployment-manager` (mirroring the “scenario-to-* packager” pattern used by `scenario-to-desktop`).

## Quickstart (local dev)

```bash
cd scenarios/scenario-to-cloud
make start
```

- UI: `http://localhost:<UI_PORT>/`
- API health: `http://localhost:<API_PORT>/health`

## CLI (via Vrooli lifecycle)

```bash
# Validate a manifest
scenario-to-cloud manifest validate cloud-manifest.json

# Generate starter manifest + inspect schema
scenario-to-cloud manifest init --scenario landing-page-business-suite --host 203.0.113.10 --domain example.com --out cloud-manifest.json
scenario-to-cloud manifest schema
scenario-to-cloud manifest doctor cloud-manifest.json
scenario-to-cloud manifest fix cloud-manifest.json --write

# Resolve the declaration-derived closure and review the executable plan
scenario-to-cloud deployment resolve --scenario landing-page-business-suite --environment production
scenario-to-cloud deployment plan --scenario landing-page-business-suite --environment production

# Apply the reviewed plan and wait on its durable operation
scenario-to-cloud deployment apply --scenario landing-page-business-suite --environment production --plan-digest <digest> --request-key launch-<id>
scenario-to-cloud operation wait <operation-id> --timeout 600

# Inspect target-owned health and operation receipts
scenario-to-cloud deployment health <deployment-id>
scenario-to-cloud operation list <deployment-id>
```

## Docs

- PRD: `scenarios/scenario-to-cloud/PRD.md`
- Requirements: `scenarios/scenario-to-cloud/requirements/`
- Research: `scenarios/scenario-to-cloud/docs/internal/RESEARCH.md`
- Problems/Risks: `scenarios/scenario-to-cloud/docs/internal/PROBLEMS.md`

## Deployment contract

The deployment path:

- Resolves scenarios, resources, host tools, safeguards, credentials and persistent-data bindings from declarations and the analyzer.
- Builds a digest-bound release bundle from that closure without unrelated repository packages.
- Delivers the bundle through the deployment's selected reach adapter and delegates setup, activation, health and host actions to their owning control-plane services.
- Allocates declared listeners through the target-owned deployment contract; it does not assume fixed listener ports.
- Applies edge, credential and recovery actions through typed durable operations. Missing authority or an unsupported target produces an explicit refusal or onboarding handoff.

The UI, CLI and API are equivalent deployment surfaces. The UI is a required professional operator surface, and all three surfaces consume the same closure, plan, operation and evidence identities.

## Identity topology

Cloud deployment must preserve the target scenario's declared identity mode.
For self-hosted LPBS, the dependency snapshot may include
`scenario-authenticator`, producing one installation-scoped identity realm in
the mini-Vrooli bundle. For a public hosted LPBS deployment, use the approved
hosted/shared identity boundary instead of creating one private authenticator
instance per request or customer by accident.

The deployed scenario's authentication metadata remains the source of truth
for `personal_local`, `local_multi_user`, `remote_vrooli`, or
`shared_provider`. The deployment manifest carries dependency and deployment
intent; it must not contain passwords, private keys, refresh tokens, or LPBS
website sessions. A required provider that cannot be resolved is a deployment
failure, not permission to fall back to an unauthenticated shared mode.

See the [manifest identity topology](docs/guides/manifest-reference.md#identity-topology)
and the project [identity contract](../../docs/concepts/IDENTITY-AND-AUTHENTICATION.md).
