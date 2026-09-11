# Deployment Manager Cheat Sheet

Use this page for the current Go CLI. Run the scenario through the Vrooli
lifecycle manager before using commands that call its API.

## Start and inspect

```bash
vrooli scenario start deployment-manager
deployment-manager status
deployment-manager analyze "my-scenario"
deployment-manager fitness "my-scenario" --tier desktop
```

The numeric deployment tier is technical target metadata. Tier 2 means the
desktop target. It is not a commercial monetization tier.

## Profiles

```bash
deployment-manager profiles list
deployment-manager profiles create "my-profile" "my-scenario" --tier 2
deployment-manager profiles show "<profile-id>"
deployment-manager profiles update "<profile-id>" --tier 2
deployment-manager profiles versions "<profile-id>"
deployment-manager profiles delete "<profile-id>"
```

The `profiles` group is the canonical typed profile surface. The singular
`profile` command is retained only as a compatibility route; use it only when
working with a legacy migration or an explicitly recorded historical receipt.

## Swaps and desktop preparation

The swap route is a compatibility operation. Review the result before applying
it to a profile.

```bash
deployment-manager swaps list "my-scenario"
deployment-manager swaps analyze postgres sqlite
deployment-manager swaps apply "<profile-id>" postgres sqlite
deployment-manager fitness "my-scenario" --tier desktop
```

Prepare an evidence-bound readiness review before a governed release:

```bash
deployment-manager readiness-reviews prepare \
  "my-scenario" "<profile-id>" "<commit>" "<artifact-digest>" stable linux \
  --fact commercial_release=true \
  --fact paid_release=true
```

Build the desktop target through the owning target ramp:

```bash
deployment-manager deploy-desktop \
  --profile "<profile-id>" \
  --platforms linux \
  --timeout 20m \
  --dry-run
```

`--dry-run` is a preview. It is not publication evidence.

## Release observation and recovery

```bash
deployment-manager releases dossier "<release-id>"
deployment-manager releases health "<release-id>"
deployment-manager releases operation "<operation-id>"
deployment-manager releases reconcile "<release-id>"
```

Recovery requires the exact review, candidate, destination revision, action,
and confirmation. Keep the command effect-free until those values are
approved:

```bash
deployment-manager releases recover "<release-id>" \
  --review-key "<review-key>" \
  --candidate-id "<candidate-id>" \
  --destination-revision-id "<destination-revision-id>" \
  --dry-run
```

## Output and configuration

Global flags must appear before the command group:

```bash
deployment-manager --json profiles list
deployment-manager configure api_base "http://localhost:8080"
deployment-manager configure token "<operator-token>"
```

Do not place secret values in retained documentation, shell history, plans, or
diagnostic output.

## Retired compatibility routes

The following routes remain discoverable only to return retirement guidance.
Use readiness reviews, evidence operations, release operations, and the owning
scenario instead:

```text
cli[old]: deployment-manager build
cli[old]: deployment-manager logs
cli[old]: deployment-manager validate
cli[old]: deployment-manager estimate-cost
cli[old]: deployment-manager secrets
cli[old]: deployment-manager signing
cli[old]: deployment-manager validations
```

## Bundle API compatibility

The bundle REST routes are retained for compatibility. Prefer the typed release
and readiness surfaces for new workflows.

```bash
API_PORT=$(vrooli scenario port deployment-manager API_PORT)
curl -X POST "http://localhost:${API_PORT}/api/v1/bundles/assemble" \
  -H "Content-Type: application/json" \
  -d '{"scenario":"my-scenario","tier":"tier-2-desktop"}'
```
