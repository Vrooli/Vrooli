# `intent.ref_missing`

> **Severity default:** ERROR · **Capability:** intent_linkage (L1) · **Fix class:** manual

## What it means

A code-typed `validation[].ref` points at a file that does not exist on disk (after mini-format normalization: `#fragment` and `::TestName` stripped, globs resolved by directory prefix). The claim's proof path is broken.

## How to fix it

Find where the test went: renamed/moved files need the ref updated; deleted tests need a replacement or an honest status downgrade. Manual-typed validations are never path-checked — if the entry describes an attended procedure, set `type: manual`.

## Provenance

Emitted by business-health's intent linkage checks (dimension `business`, advisory cap at ERROR — this finding never blocks a suite on its own). Mapped in `.vrooli/maturity.json`; severity and ladder impact live there, not here.
