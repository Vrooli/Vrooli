# `experience.index_parity`

> **Severity default:** ERROR · **Capability:** spec_contract · **Fix class:** auto (fixer pending)

## What it means

`experience/index.json` and the on-disk page or journey files disagree.

## How to fix it

Add missing index entries, remove stale entries, or create the referenced page/journey files so both directions match.

## Provenance

Emitted by experience-manager's spec parser checks (dimension `experience`, advisory cap at ERROR). Mapped by the experience phase maturity descriptor once the provider phase lands.
