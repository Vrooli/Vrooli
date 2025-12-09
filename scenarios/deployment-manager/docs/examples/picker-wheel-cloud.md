# Example: Picker Wheel — Cloud / SaaS

## Current Approach

- Deploy Tier 1 stack on a DigitalOcean VPS (same machine we develop on) and expose Picker Wheel through app-monitor + Cloudflare tunnel.
- No automated provisioning; setup is manual using scripts from `docs/deployment/providers/digitalocean.md`.
- Secrets are copied from local dev, which is unacceptable for real customers.

## Pain Points

1. **Manual Infrastructure** — Creating droplets, installing dependencies, and wiring tunnels is a bespoke process.
2. **Secret Leakage Risk** — Without a secrets plan, we either ship dev secrets or hand-edit `.env` files.
3. **Scaling Unknowns** — No fitness data to estimate CPU/RAM needs per customer.
4. **Monitoring** — No standard logging/alerting for remote installs.

## Desired Flow (Tier 4)

1. deployment-manager calculates Picker Wheel's dependency DAG and cloud fitness.
2. We review swap suggestions (e.g., use managed Postgres instead of local container) and secrets plan.
3. scenario-to-cloud renders Terraform/Helm for DigitalOcean, provisioning managed DBs, app pods, ingress, TLS, etc.
4. Secrets-manager generates per-customer credentials and injects them via provider secret stores.
5. Automated smoke tests confirm the deployment before sharing URLs with customers.

## Interim Documentation

- Keep track of manual steps in `docs/deployment/providers/digitalocean.md`.
- Note any customer-specific tweaks so they can become deployment-manager modules later.
