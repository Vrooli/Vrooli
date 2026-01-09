# Providers

Infrastructure-specific documentation for deployment targets.

## Cloud Providers

| Provider | Tier | Status | Documentation |
|----------|------|--------|---------------|
| [DigitalOcean](digitalocean.md) | 4 | Reference | VPS/Kubernetes setup |
| AWS | 4 | Planned | Not yet documented |
| GCP | 4 | Planned | Not yet documented |

## Access Providers

| Provider | Tier | Status | Documentation |
|----------|------|--------|---------------|
| [Cloudflare Tunnel](cloudflare-tunnel.md) | 1 | Production | Secure Tier 1 remote access |

## Hardware

| Provider | Tier | Status | Documentation |
|----------|------|--------|---------------|
| [Hardware Appliance](hardware-appliance.md) | 5 | Vision | Planning notes |

## Provider Selection

### For Tier 1 (Local)

Use **Cloudflare Tunnel** via app-monitor for secure remote access to your local development stack.

### For Tier 4 (SaaS)

Choose based on your requirements:

| Factor | DigitalOcean | AWS | GCP |
|--------|--------------|-----|-----|
| Simplicity | Best | Complex | Moderate |
| Cost (small) | Low | Moderate | Moderate |
| Cost (scale) | Moderate | Best | Best |
| Managed K8s | Yes (DOKS) | Yes (EKS) | Yes (GKE) |

### For Tier 5 (Enterprise)

Hardware appliance documentation covers planning for physical device deployments.

## Using Provider Docs

These docs provide:

1. **Setup instructions** - How to configure the provider
2. **Cost guidance** - Pricing considerations
3. **Integration notes** - How it connects to deployment-manager

## Related

- [Tier 4: SaaS/Cloud](../tiers/tier-4-saas.md) - Cloud deployment reference
- [Tier 5: Enterprise](../tiers/tier-5-enterprise.md) - Enterprise deployment reference
- [SaaS Workflow](../workflows/saas-deployment.md) - Cloud deployment guide
