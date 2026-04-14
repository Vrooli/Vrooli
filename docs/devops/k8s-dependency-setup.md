# Kubernetes Dependency Setup

This file is a historical and research reference for Kubernetes-specific dependency preparation. It is not part of the canonical setup or deployment workflow today.

## Status

Current setup should follow:

- `make setup`
- `vrooli --help`
- [../deployment/README.md](../deployment/README.md)

The older Kubernetes dependency scripts and procedures in prior versions of this document described an experimental path that is no longer the active deployment model.

## How To Use This File

Use it only if you are:

- evaluating what future SaaS or cluster automation would require
- auditing old Kubernetes support assumptions
- preparing a design for deployment-manager or scenario-to-cloud successors

## Non-Canonical Historical Items

The following should be treated as historical unless explicitly revived by current deployment docs:

- `scripts/helpers/deploy/k8s-dependencies-check.sh`
- `scripts/helpers/deploy/k8s-prerequisites.sh`
- `vrooli develop --target k8s-cluster`
- `vrooli setup --target k8s-cluster`
- Vault- and operator-installation flows tied to the old cluster experiment

## Durable Concepts

The main concepts that still matter are:

- cluster deployments need explicit secret lineage
- storage classes and ingress choices affect scenario fitness
- dependency installation has to be reproducible and validated
- cluster support should be expressed through current manifests and deployment metadata, not a parallel shell-era control plane

## Source Of Truth

For current deployment truth, use:

- [../deployment/README.md](../deployment/README.md)
- [server-deployment.md](server-deployment.md)
- [environment-management.md](environment-management.md)
