# Vrooli Deployment Guide

## Deployment Overview

Vrooli scenarios run **directly** without conversion to standalone apps. This guide helps you choose the right deployment approach for your use case.

## Quick Navigation

### 🚀 **Start Here**: [Direct Scenario Deployment](../scenarios/DEPLOYMENT.md)
**Most Common Use Case** - Running scenarios directly from Vrooli
- Local development and testing
- Direct scenario execution
- Immediate deployment for most use cases

### 🏗️ **Infrastructure Setup**
Choose based on your infrastructure needs:

| Deployment Type | Guide | Best For |
|-----------------|-------|----------|
| **Local Development** | [Development Environment](../devops/development-environment.md) | Testing scenarios locally |
| **Kubernetes (Local)** | [Kubernetes Setup](../devops/kubernetes.md) | Local K8s development |
| **Kubernetes (Cloud)** | [Production Deployment](production-deployment-guide.md) | Cloud production deployment |
| **VPS/Server** | [Server Deployment](../devops/server-deployment.md) | Custom server setup |

## Deployment Decision Tree

```
Do you need to run scenarios directly?
├─ YES → Use Direct Scenario Deployment (scenarios/DEPLOYMENT.md)
└─ NO → Do you need cloud production infrastructure?
    ├─ YES → Use Production Deployment (production-deployment-guide.md)
    └─ NO → Use Local Development (devops/development-environment.md)
```

## Migration Notes

**Current Architecture**: Scenarios run directly without conversion, providing simpler, faster, and more reliable deployment.

Direct execution eliminates conversion complexity while maintaining all functionality.

## Quick Commands

```bash
# Most common deployment workflow
vrooli resource start-all       # Start required resources
vrooli scenario run <name>      # Run scenario directly
vrooli scenario test <name>     # Test scenario integration

# For production cloud deployment
vrooli build                    # Build production artifacts
vrooli deploy                   # Deploy to configured target
```

## Getting Help

- **Current Deployment Issues**: Check [scenarios/DEPLOYMENT.md](../scenarios/DEPLOYMENT.md)
- **Infrastructure Problems**: See [devops/troubleshooting.md](../devops/troubleshooting.md)
- **Performance Issues**: Review [server requirements](../devops/server-deployment.md#server-requirements)