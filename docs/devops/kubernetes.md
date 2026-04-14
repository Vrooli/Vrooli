# Kubernetes Infrastructure

This document is a legacy and forward-looking infrastructure reference. It is not the canonical way to deploy Vrooli today.

## Status

The current production-ready path remains the Tier 1 local stack described in the [Deployment Hub](../deployment/README.md).

Kubernetes matters for future Tier 4 and Tier 5 automation, but the old direct "deploy scenarios straight to Kubernetes" workflow is not current platform truth.

Use this file only when you need to understand:

- which infrastructure primitives future cloud automation may need
- what older Kubernetes experiments assumed
- what deployment-manager or scenario-to-cloud successors may eventually reproduce

## What Still Matters

The durable ideas from the older Kubernetes work are:

- operator-managed shared infrastructure is a plausible future delivery path
- secrets, storage, and dependency fitness must be handled per deployment target
- scenario packaging for cloud targets must be manifest-driven, not shell-script-driven

## What Is Not Canonical

Do not treat the following as current guidance:

- `vrooli develop --target k8s-cluster`
- `vrooli setup --target k8s-cluster`
- direct scenario deployment to Kubernetes from old scripts
- older multi-node cluster topologies as the default operational model

If those flows are revived in the future, they should be documented through the Deployment Hub and deployment-manager architecture, not through ad hoc historical commands.

## If You Are Researching Kubernetes Support

Start with:

- [../deployment/README.md](../deployment/README.md)
- [../operations/production-guide.md](../operations/production-guide.md)
- [server-deployment.md](server-deployment.md)

Then treat older operator, storage, ingress, and packaging notes as research input only.
