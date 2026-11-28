# `business_req_missing_title`

> **Severity default:** WARNING · **Capability:** requirements_registry (L2) · **Fix class:** manual

## What it means

A requirement has an empty `title`. The title is the one-line falsifiable claim humans scan.

## How to fix it

Write a title that states the behavior someone could test, not the component name.

## Provenance

Emitted by business-health's requirements registry checks (dimension `business`, advisory cap at ERROR — this finding never blocks a suite on its own). Mapped in `.vrooli/maturity.json`; severity and ladder impact live there, not here.
