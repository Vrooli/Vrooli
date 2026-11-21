# `business_registry_unparseable`

> **Severity default:** ERROR · **Capability:** requirements_registry (L0) · **Fix class:** manual

## What it means

A registry file (index.json or a module.json) exists but cannot be parsed, or index.json declares an import that does not exist. Until it parses, none of its requirement claims can be trusted or evaluated.

## How to fix it

Fix the JSON syntax (the finding message carries the parser error) or repair the dangling import path in `requirements/index.json`. Repairing by hand is required: a fixer would have to guess what the author meant.

## Provenance

Emitted by business-health's requirements registry checks (dimension `business`, advisory cap at ERROR — this finding never blocks a suite on its own). Mapped in `.vrooli/maturity.json`; severity and ladder impact live there, not here.
