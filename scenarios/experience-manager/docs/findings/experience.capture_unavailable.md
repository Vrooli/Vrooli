# `experience.capture_unavailable`

> **Severity default:** INFO · **Capability:** structure_reconciliation · **Fix class:** external

## What it means

An active page needed a BAS accessibility capture, but the capture engine was unavailable.

## How to fix it

Start or repair browser-automation-studio, then rerun the experience phase. This finding is skipped-never-failed by design.

## Provenance

Emitted by experience-manager's BAS capture runner (dimension `experience`). It reports skipped evidence, never a failed UX claim.
