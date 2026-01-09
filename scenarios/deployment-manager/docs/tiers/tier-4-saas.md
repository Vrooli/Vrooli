# Tier 4 · SaaS / Cloud

Tier 4 covers scenarios deployed to remote servers (DigitalOcean, AWS, bare metal) where we still run the full Vrooli stack, but expect customers or team members to access it over the internet.

## Current State

- Legacy docs described a Kubernetes packaging pipeline driven by `./scripts/deployment/package-scenario-deployment.sh`. That workflow is **retired**.
- We typically run Tier 1 servers on beefier hardware and tunnel via Cloudflare. There is no reproducible automation for customer-ready cloud installs.
- Infrastructure scripts (Kubernetes, Vault, operators) exist but do not integrate with the scenario lifecycle.

## What Tier 4 Needs

| Requirement | Description |
|-------------|-------------|
| Dependency manifest | Same DAG as Tier 2/3, but annotated with cloud-friendly defaults (managed DBs, GPU/CPU requirements, scaling hints). |
| Secrets strategy | Infrastructure secrets stay in the mothership; per deployment secrets (DB passwords, OAuth secrets) generated & stored via secrets-manager. |
| Provisioning targets | `deployment-manager` should emit Terraform/Ansible/manifests per provider (DigitalOcean, AWS, bare metal). |
| Monitoring hooks | SaaS deployments need logging/metrics/alerting integration. |

## Provider Notes

- [DigitalOcean](../providers/digitalocean.md) retains manual steps (costing, `doctl`). These become callable recipes once deployment-manager supports provider modules.
- Vault/Kubernetes hardening instructions were moved to [history](../history) until we re-spec them for the new flow.

## Roadmap

1. Capture provider capabilities in `.vrooli/service.json` (e.g., `deployment.cloud.preferred_providers: ["digitalocean", "aws"]`).
2. Extend `scenario-dependency-analyzer` to translate dependency requirements into infrastructure primitives (managed DB vs embedded, GPU vs CPU, storage, etc.).
3. Build `deployment-manager` modules per provider that can:
   - Render Terraform/Helm/Ansible from the dependency manifest.
   - Provision secrets via secrets-manager + provider-specific secret stores.
4. Re-implement CI/CD instructions once deployment-manager can output reproducible artifacts (Docker images, manifests, secret templates).

## Interim Guidance

- Continue using Tier 1 + Cloudflare tunnels for remote demos/customers.
- When a true SaaS deployment is unavoidable, document every manual step under `docs/deployment/providers/` and file app-issue-tracker tasks to automate it in deployment-manager.
