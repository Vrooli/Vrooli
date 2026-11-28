# Bundle Storage

This document is a specialized reference for bundle storage in deployment-oriented scenarios such as `scenario-to-cloud`. It is not a general storage guide for all Vrooli deployments.

## Status

Use this file only when working on:

- deployment bundle generation
- bundle retention and cleanup
- scenario-to-cloud or deployment-manager-adjacent workflows

The current deployment source of truth is still the [Deployment Hub](README.md).

## Scope

This file should cover only:

- where deployment bundles are stored
- how bundle cleanup and retention work
- how bundle buildup affects local or target-host disk usage

It should not be treated as the canonical storage model for:

- scenario runtime state in general
- resource persistence in general
- Tier 1 local stack operations in general

## Practical Guidance

When bundle storage matters:

- document the exact scenario or deployment workflow involved
- distinguish local cache paths from remote target paths
- make retention rules explicit
- avoid implying that bundle-based deployment is the universal default across tiers
- keep Tier 1 operational storage assumptions separate from bundle-export storage assumptions

For broader deployment guidance, start with:

- [README.md](README.md)
- [../operations/production-guide.md](../operations/production-guide.md)
