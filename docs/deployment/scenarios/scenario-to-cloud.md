# Scenario: scenario-to-cloud (Future)

## Mission

Translate deployment-manager bundle manifests into runnable cloud environments (SaaS installs, managed VPS, Kubernetes, etc.).

## Capabilities

1. **Provider Modules** — Plugins for DigitalOcean, AWS, bare metal, etc., each capable of provisioning compute, storage, networking, and managed services.
2. **Artifact Building** — Build/push Docker images (or native binaries) as needed.
3. **Manifest Rendering** — Output Terraform, Helm, Ansible, or plain shell scripts depending on provider.
4. **Secret Wiring** — Create provider-specific secrets (AWS Secrets Manager, DO App Platform vars, Kubernetes secrets) using metadata from secrets-manager.
5. **Health Verification** — Run smoke tests after provisioning and report status back to deployment-manager.

## Inputs

- Bundle manifest from deployment-manager.
- Provider credentials (referenced via secrets-manager; never embedded).
- Optional `values.yaml` style overrides for staging/prod.

## Outputs

- Infrastructure manifests (`dist/<scenario>/cloud/<provider>/`)
- Deploy scripts (`make deploy-cloud`, `vrooli scenario deploy --tier cloud`)
- Status reports for deployment-manager.

## Future Workflow

1. Choose scenario + tier (SaaS) in deployment-manager.
2. Confirm dependency swaps and secret plan.
3. Click "Deploy" → scenario-to-cloud provisions infrastructure via provider module.
4. Monitor logs/health in deployment-manager.

## Interim Status

Until scenario-to-cloud exists, we rely on Tier 1 servers and manual provider docs. The legacy Kubernetes instructions now live under `docs/deployment/history/` for reference.
