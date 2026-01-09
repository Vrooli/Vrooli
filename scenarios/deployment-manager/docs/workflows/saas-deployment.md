# SaaS/Cloud Deployment Workflow (Tier 4)

Deploy a Vrooli scenario to cloud infrastructure (DigitalOcean, AWS, bare metal) as a hosted service.

## Implementation Status

| Provider | Status | Packager |
|----------|--------|----------|
| DigitalOcean | Not Started | `scenario-to-saas` (planned) |
| AWS | Not Started | `scenario-to-saas` (planned) |
| Bare Metal | Not Started | `scenario-to-saas` (planned) |

> **Current Reality**: SaaS deployment is in the planning stage. Scenarios can be manually deployed to cloud infrastructure using Docker/Kubernetes, but automated orchestration via deployment-manager is not yet implemented.

---

## Planned Workflow Overview

SaaS deployment follows the same 8-phase structure as [Desktop Deployment](desktop-deployment.md), with cloud-specific considerations.

### Phase 1: Check Compatibility

```bash
# Check SaaS fitness (tier 4)
deployment-manager fitness <scenario-name> --tier 4
# OR
deployment-manager fitness <scenario-name> --tier saas
deployment-manager fitness <scenario-name> --tier cloud
```

### Phase 2: Address Blocking Issues

SaaS has different constraints than desktop/mobile:

| Consideration | Desktop Swap | SaaS Approach |
|---------------|--------------|---------------|
| PostgreSQL | SQLite | Keep PostgreSQL (managed DB) |
| Redis | In-process | Keep Redis (managed cache) |
| Ollama | Bundled models | Keep Ollama or use cloud GPUs |
| Storage | Local files | Object storage (S3, Spaces) |

**SaaS often reverses desktop swaps** - heavy dependencies are fine when you control the infrastructure.

### Phase 3: Create Deployment Profile

```bash
# Create SaaS profile
deployment-manager profile create <name> <scenario> --tier 4

# Specify provider
deployment-manager profile set <profile-id> provider digitalocean
deployment-manager profile set <profile-id> region nyc1
```

### Phase 4: Configure Secrets

SaaS secrets use different strategies:

| Secret Type | Desktop | SaaS |
|-------------|---------|------|
| Database credentials | Generated/swapped | Vault/environment |
| API keys | User prompted | Vault/secrets manager |
| TLS certificates | Not applicable | Let's Encrypt/managed |
| Service tokens | Generated | Vault rotation |

```bash
# Generate Vault-compatible template
deployment-manager secrets template <profile-id> --format vault
```

### Phase 5: Generate Deployment Manifest

SaaS manifests include infrastructure definitions:

```json
{
  "schema_version": "v0.1",
  "target": "saas",
  "provider": "digitalocean",
  "app": {
    "name": "picker-wheel",
    "version": "1.0.0",
    "domain": "picker-wheel.example.com"
  },
  "infrastructure": {
    "compute": {
      "type": "droplet",
      "size": "s-2vcpu-4gb",
      "count": 2
    },
    "database": {
      "type": "managed-postgres",
      "size": "db-s-1vcpu-1gb"
    },
    "cache": {
      "type": "managed-redis",
      "size": "db-s-1vcpu-1gb"
    },
    "storage": {
      "type": "spaces",
      "region": "nyc3"
    }
  },
  "networking": {
    "load_balancer": true,
    "ssl": "letsencrypt",
    "cdn": true
  }
}
```

### Phase 6: Estimate Costs

```bash
deployment-manager estimate-cost <profile-id> --verbose
```

**Expected output:**
```json
{
  "profile_id": "profile-1234567890",
  "provider": "digitalocean",
  "monthly_estimate": {
    "compute": 48.00,
    "database": 15.00,
    "cache": 15.00,
    "storage": 5.00,
    "bandwidth": 10.00,
    "total": 93.00
  },
  "currency": "USD",
  "notes": [
    "Bandwidth estimate based on 500GB/month",
    "Storage estimate based on 100GB"
  ]
}
```

### Phase 7: Deploy

**Planned workflow:**
```bash
# Validate infrastructure manifest
deployment-manager validate <profile-id> --verbose

# Deploy (creates infrastructure + deploys app)
deployment-manager deploy <profile-id>

# Or dry-run first
deployment-manager deploy <profile-id> --dry-run
```

### Phase 8: Monitor

SaaS monitoring includes:
- Infrastructure health (CPU, memory, disk)
- Application metrics (requests, latency, errors)
- Cost tracking and alerts
- Log aggregation
- Uptime monitoring

---

## Provider-Specific Details

### DigitalOcean

**Resources:**
- Droplets (compute)
- Managed Databases (PostgreSQL, MySQL, Redis)
- Spaces (object storage)
- Load Balancers
- App Platform (alternative to Droplets)

**Authentication:**
```bash
# Set DigitalOcean token
deployment-manager configure do_token <your-token>
```

See [DigitalOcean Provider Guide](../providers/digitalocean.md) for details.

### AWS

**Resources:**
- EC2 (compute) or ECS/Fargate (containers)
- RDS (managed databases)
- ElastiCache (Redis)
- S3 (object storage)
- ALB (load balancing)
- CloudFront (CDN)

**Authentication:**
```bash
# Set AWS credentials
deployment-manager configure aws_access_key <key>
deployment-manager configure aws_secret_key <secret>
deployment-manager configure aws_region us-east-1
```

### Bare Metal

For self-hosted infrastructure:

```json
{
  "provider": "bare-metal",
  "infrastructure": {
    "servers": [
      {"host": "server1.example.com", "role": "app"},
      {"host": "server2.example.com", "role": "app"},
      {"host": "db.example.com", "role": "database"}
    ]
  }
}
```

---

## Fitness Scoring for SaaS

SaaS has higher baseline fitness than desktop/mobile because cloud infrastructure is flexible:

| Tier | Baseline | Notes |
|------|----------|-------|
| Desktop (2) | 75/100 | Lightweight deps required |
| Mobile (3) | 40/100 | Heavy swaps often needed |
| SaaS (4) | 85/100 | Most deps work natively |

### Common Blockers

1. **Licensing** - Some dependencies have cloud licensing restrictions
2. **Compliance** - Data residency requirements
3. **Cost** - GPU workloads can be expensive
4. **Secrets** - Infrastructure secrets must be properly managed

---

## Comparison: Desktop vs SaaS

| Aspect | Desktop (Tier 2) | SaaS (Tier 4) |
|--------|------------------|---------------|
| PostgreSQL | Swap to SQLite | Keep (managed DB) |
| Redis | Swap to in-process | Keep (managed cache) |
| Ollama | Swap to bundled models | Keep (cloud GPU) |
| Secrets | User prompted | Vault managed |
| Updates | electron-updater | Rolling deployment |
| Scaling | Single user | Horizontal scaling |
| Cost model | One-time purchase | Recurring infrastructure |

---

## Zero-Downtime Updates (Planned)

SaaS deployments will support zero-downtime updates:

```bash
# Blue-green deployment
deployment-manager deploy <profile-id> --strategy blue-green

# Rolling update
deployment-manager deploy <profile-id> --strategy rolling

# Canary deployment
deployment-manager deploy <profile-id> --strategy canary --canary-percent 10
```

---

## Roadmap

1. **Define SaaS manifest schema** - Extend v0.1 with infrastructure definitions
2. **Implement cost estimation** - Provider-specific pricing calculations
3. **Build scenario-to-saas** - Infrastructure provisioning + app deployment
4. **Add provider integrations** - DigitalOcean, AWS, bare metal
5. **Implement deployment strategies** - Blue-green, rolling, canary
6. **Add monitoring integration** - Prometheus, Grafana, alerting
7. **Build rollback automation** - One-click rollback to previous version

---

## Current Manual Approach

Until automated SaaS deployment is implemented, scenarios can be manually deployed:

### Docker-Based Deployment

```bash
# Build Docker image
cd scenarios/<scenario>
docker build -t <scenario>:latest .

# Push to registry
docker tag <scenario>:latest registry.example.com/<scenario>:latest
docker push registry.example.com/<scenario>:latest

# Deploy to server
ssh server "docker pull registry.example.com/<scenario>:latest && docker-compose up -d"
```

### Kubernetes Deployment

See [Kubernetes Legacy Guide](../history/k8s-legacy.md) for historical K8s deployment patterns.

---

## Related Documentation

- [Tier 4 SaaS Reference](../tiers/tier-4-saas.md)
- [DigitalOcean Provider Guide](../providers/digitalocean.md)
- [Desktop Deployment Workflow](desktop-deployment.md) - Similar workflow structure
- [Secrets Management Guide](../guides/secrets-management.md) - Vault integration
- [Kubernetes Legacy Guide](../history/k8s-legacy.md) - Historical patterns

---

## Contributing

SaaS deployment requires infrastructure automation expertise. If you're interested in contributing:

1. Review the [Tier 4 SaaS Reference](../tiers/tier-4-saas.md) for context
2. Evaluate infrastructure-as-code tools (Terraform, Pulumi, CDK)
3. Prototype with a simple scenario on DigitalOcean
4. Document findings in this workflow guide
5. Create scenario-to-saas packager scenario
