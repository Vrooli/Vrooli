# Cloud deployment quick start

> **Product workflow.** Operators do not prepare a VPS or deploy from a terminal. Compute Manager
> owns provider onboarding and admitted compute; Scenario-to-Cloud connects deployment targets;
> Deployment Manager owns the resumable deployment pipeline.

## Existing VPS

1. Open **Compute Manager → Adopt** and submit the server address, Unix user, and declared monthly
   cost facts.
2. Continue to the secure Onboarding credential step when the separately approved first-touch
   grant is available. The one-time credential belongs only in that write-only field.
3. Wait for Compute Manager to verify Bridge enrollment and mark the instance ready.
4. Select the ready Compute Manager instance in the Deployment Manager operation.

Compute Manager releases only its adoption record and Bridge attachment. It does not destroy an
operator-owned VPS.

## Provider-managed compute

1. Complete provider-owned account and server actions in that provider's console. For the approved
   staging path, that means only the previously approved Hetzner website steps.
2. Complete provider verification, quote and admission in Compute Manager.
3. Start or resume the deployment in Deployment Manager; Scenario-to-Cloud receives its stable
   Compute Manager instance reference.

No provider credential belongs in this guide. Do not paste credentials into chat or documents.
Live provider readiness, VPS adoption and deployment are not claimed here. See the
[superseded manual VPS guide](guides/vps-setup.md) for its replacement notice and the current
ownership boundary.
