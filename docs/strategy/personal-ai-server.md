# Personal AI Server

This document is a reference architecture and exploration note, not a canonical Vrooli deployment standard.

## Status

It describes one possible way to host a powerful local AI-heavy environment around Vrooli. It should not be read as:

- the required hardware shape for Vrooli
- the default production deployment model
- an officially standardized operations recipe

The current real deployment baseline is still the Tier 1 local stack in the [Deployment Hub](../deployment/README.md).

## Why Keep This Document

It remains useful for thinking about:

- local-first infrastructure with strong operator control
- GPU-heavy planning or inference hosts
- isolation boundaries between workloads
- remote access patterns for phone, browser, and voice interfaces
- future hardware-appliance and specialized-server directions

## How To Read It

Treat the host OS, hypervisor, VPN, proxy, and monitoring choices in earlier versions of this document as one plausible pattern, not a project-wide requirement.

If future appliance or specialized-server work lands, this document should evolve into one of:

- a product-specific reference architecture
- a provider note under deployment docs
- a hardware-appliance planning artifact

Until then, it is best read as strategic infrastructure exploration.

## Current Canonical Docs

For current operational guidance, use:

- [../deployment/README.md](../deployment/README.md)
- [../operations/production-guide.md](../operations/production-guide.md)
- [server-deployment.md](server-deployment.md)
