# Vrooli Deployment Guide

## 🚀 Choose Your Deployment Path

Vrooli scenarios run **directly** from source without conversion. Choose your deployment approach based on your specific needs:

## Quick Decision Tree

```
What are you trying to do?

🚀 RUN SCENARIOS (95% of use cases)
├─ Local testing/development → vrooli scenario run <name>
├─ Customer delivery → Package multiple scenarios for production
└─ Production deployment → Deploy to Kubernetes cluster

⚙️ DEVELOP VROOLI CORE (Contributors only)
└─ Core platform development → vrooli develop
```

## 🎯 Main Deployment Paths

### 1. Run Scenarios Locally (95% of users)
```bash
# Start required resources
vrooli resource start-all

# Run any scenario directly  
vrooli scenario run research-assistant
vrooli scenario run invoice-generator

# Test scenario integration
vrooli scenario test research-assistant
```
**Perfect for:** Development, testing, local business applications, customer demos  
**Time to deploy:** 30 seconds  
**Documentation:** [Direct Scenario Execution](../scenarios/DEPLOYMENT.md)

### 2. Deploy to Production (Customer delivery)
```bash
# Package scenarios for customer deployment
./scripts/deployment/package-scenario-deployment.sh \
  "customer-suite" ~/deployments/customer \
  research-assistant invoice-generator customer-portal

# Deploy to production cluster
kubectl apply -f ~/deployments/customer/k8s/
```
**Perfect for:** Customer deliveries, production business applications  
**Time to deploy:** 10-15 minutes  
**Documentation:** [Production Deployment](production-deployment-guide.md)

## 📋 Deployment Comparison

| Approach | Complexity | Time | Best For | Resources Needed |
|----------|------------|------|----------|------------------|
| **Local Scenarios** | ⭐ Simple | 30s | Development, testing, demos | Local Docker |
| **Production Suite** | ⭐⭐⭐ Complex | 15min | Customer delivery | Kubernetes cluster |

## 🛠️ Infrastructure Requirements

### Local Scenario Execution
- **Hardware:** 4GB RAM, 2 CPU cores
- **Software:** Docker, Vrooli CLI
- **Network:** Internet for initial setup
- **Time:** 5 minutes setup

### Production Business Deployment  
- **Hardware:** 8GB RAM, 4 CPU cores minimum
- **Software:** Kubernetes cluster, kubectl, helm
- **Network:** External access, SSL certificates
- **Time:** 2-4 hours initial setup

## 🚨 Quick Troubleshooting

| Issue | Solution |
|-------|----------|
| Scenario won't start | Check `vrooli resource status` |
| Port conflicts | Check `~/.vrooli/port-registry.json` |
| Resource unavailable | Run `resource-<name> start` |
| Performance slow | See resource requirements above |

## 📚 Detailed Guides

- **[Direct Scenario Deployment](../scenarios/DEPLOYMENT.md)** - Run scenarios directly (recommended)
- **[Production Deployment](production-deployment-guide.md)** - Cloud Kubernetes deployment
- **[Development Environment](../devops/development-environment.md)** - Core contributor setup
- **[CI/CD Integration](../devops/ci-cd.md)** - Automated testing and deployment
- **[Troubleshooting](../devops/troubleshooting.md)** - Common issues and solutions