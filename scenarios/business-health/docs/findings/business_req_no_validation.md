# `business_req_no_validation`

> **Severity default:** WARNING · **Capability:** requirements_registry (L2) · **Fix class:** manual

## What it means

A non-starter requirement has zero validation entries (escalates to ERROR when criticality is P0). A claim without a path to proof cannot ever earn its status.

## How to fix it

Add a real validation: a `test`-typed entry pointing at a test file (tag it `[REQ:ID]` so sync sees it), or a `manual`-typed entry whose evidence is recorded via `business-health manual-log`. Never point a ref at an unrelated file to silence the finding — and never delete a P0 requirement to dodge it.

## Provenance

Emitted by business-health's requirements registry checks (dimension `business`, advisory cap at ERROR — this finding never blocks a suite on its own). Mapped in `.vrooli/maturity.json`; severity and ladder impact live there, not here.
