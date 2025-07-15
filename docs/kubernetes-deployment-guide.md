# Kubernetes Deployment Guide

This guide covers deploying Vrooli to Kubernetes using Helm charts.

## Prerequisites

1. **Install Kubernetes Tools** (auto-installed by our scripts):
   ```bash
   ./scripts/main/setup.sh --target k8s-cluster
   ```
   This installs kubectl, helm, and minikube automatically.

2. **Start Minikube** (for local development):
   ```bash
   minikube start --force  # Use --force if running as root
   ```

3. **Install Kubernetes Operators** (optional, for full deployment):
   ```bash
   ./scripts/helpers/deploy/k8s-prerequisites.sh --yes
   ```

## Build Process

### Building Kubernetes Artifacts

```bash
# Build k8s artifacts with specific version
./scripts/main/build.sh --environment development --test no --lint no --bundles zip --artifacts k8s --version 2.0.0
```

Note: The UI build takes 5-10 minutes due to processing 4400+ modules.

## Deployment Options

### Option 1: Using Deploy Script (Recommended)

```bash
./scripts/main/deploy.sh --source k8s --environment dev --version 2.0.0
```

### Option 2: Direct Helm Deployment

```bash
# Package the chart
cd k8s/chart && helm package . --version 2.0.0

# Deploy using Helm
helm upgrade --install vrooli-dev vrooli-2.0.0.tgz \
  --namespace dev \
  --create-namespace \
  --wait \
  --timeout 10m
```

## Registry Configuration

### Local Development with Minikube

Minikube provides a registry on port 32770 (not the default 5000):

1. **Push images to Minikube registry**:
   ```bash
   # Tag images for minikube registry
   docker tag ui:dev localhost:32770/ui:dev
   docker tag server:dev localhost:32770/server:dev
   
   # Push to registry
   docker push localhost:32770/ui:dev
   docker push localhost:32770/server:dev
   ```

2. **Configure Helm values**:
   ```yaml
   image:
     registry: "registry.kube-system.svc.cluster.local"
   ```

### Production Deployment

For production, use your actual Docker registry:
```yaml
image:
  registry: "docker.io/yourusername"
dockerhubUsername: "yourusername"
```

## Selective Service Deployment

To deploy only specific services, create a custom values file:

```yaml
# minimal-values.yaml
services:
  ui:
    enabled: true
    repository: ui
    tag: dev
  server:
    enabled: true
    repository: server
    tag: dev
  jobs:
    enabled: false  # Disable jobs service
  nsfwDetector:
    enabled: false  # Disable NSFW detector

# Disable operators for minimal deployment
pgoPostgresql:
  enabled: false
spotahomeRedis:
  enabled: false
vso:
  enabled: false
```

Note: The deployment template was updated to respect the `enabled` flag for services.

## Accessing the Application

### Port Forwarding (Recommended)

```bash
# Forward UI service
kubectl port-forward svc/vrooli-dev-ui 8080:3000 -n dev

# Access at http://localhost:8080
```

### Minikube Service

```bash
# Open service in browser
minikube service vrooli-dev-ui -n dev
```

### Check Deployment Status

```bash
# View all resources
kubectl get all -n dev

# Check pod logs
kubectl logs -n dev deployment/vrooli-dev-ui

# Describe issues
kubectl describe pod -n dev <pod-name>
```

## Common Issues

### ImagePullBackOff Errors

1. **Wrong registry address**: From inside Kubernetes, use `registry.kube-system.svc.cluster.local` not `localhost`
2. **Missing credentials**: Check if `vrooli-dockerhub-pull-secret` exists
3. **Image not in registry**: Ensure images are pushed to the correct registry

### Build Timeouts

- UI builds take 5-10 minutes - increase timeout settings
- Use `--timeout 15m` for build commands
- Consider optimizing Vite configuration for faster builds

### Services Not Respecting enabled Flag

This was fixed in the deployment template. Ensure you're using the latest chart version.

## Performance Optimization Opportunities

1. **UI Build Performance**:
   - 4444 modules being transformed
   - Large chunk warnings (>1000kb)
   - Consider implementing better code splitting
   - Review and remove unused dependencies

2. **Docker Layer Caching**:
   - Optimize Dockerfile order for better caching
   - Use multi-stage builds efficiently

3. **Helm Chart Improvements**:
   - Add more granular service controls
   - Implement resource quotas
   - Add HPA (Horizontal Pod Autoscaler) support