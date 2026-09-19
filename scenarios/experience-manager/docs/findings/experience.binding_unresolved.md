# `experience.binding_unresolved`

> **Severity default:** ERROR · **Capability:** structure_reconciliation · **Fix class:** manual

## What it means

A declared binding could not be matched to a node in the captured accessibility tree.

## How to fix it

Ensure the UI exposes the declared test id, role, or accessible name, or update the binding to match the real built surface.

## Provenance

Emitted by experience-manager's AX-tree reconciliation checks (dimension `experience`, advisory cap at ERROR). Mapped by the experience phase maturity descriptor once the provider phase lands.
