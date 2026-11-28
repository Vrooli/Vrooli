# `experience.route_unspecced`

> **Severity default:** WARNING · **Capability:** route_coverage · **Fix class:** manual

## What it means

A UI route exists without a corresponding page entry in `experience/`.

## How to fix it

Add a page spec for the route or remove the route if it is not part of the intended product surface.

## Provenance

Emitted by experience-manager's route coverage checks (dimension `experience`, advisory cap at ERROR). Mapped by the experience phase maturity descriptor once the provider phase lands.
