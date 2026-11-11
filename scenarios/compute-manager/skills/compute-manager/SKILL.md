---
name: "compute-manager"
description: "Rent, track and retire remote compute: request a remote server or GPU instance, see what it costs and how much of the compute ceiling is used, extend or destroy an instance, and reconcile instances against the provider so nothing leaks."
license: "CC-BY-4.0"
metadata:
  kind: "skill"
  schemaVersion: 1
  status: "active"
  modes: ["tools"]
  tags: ["compute-manager", "remote-compute", "gpu", "server", "rent", "cloud-instance", "cost", "metering", "reconcile"]
  requires:
    scenarios: ["compute-manager"]
    commands: ["compute-manager instance request"]
  origin:
    kind: "authored"
---

## compute-manager

Instances are provisioned through intents and metered; the provider's list is reconciled against the ledger.

| You want | Command |
|---|---|
| Request an instance | `compute-manager instance request` |
| What is running | `compute-manager instance list`, `compute-manager instance get` |
| What it costs | `compute-manager meter usage`, `compute-manager meter reservations`, `compute-manager meter ceiling` |
| Keep or retire it | `compute-manager instance extend`, `compute-manager instance destroy` |
| Adopt a host you already own | `compute-manager instance adopt` |
| Find orphaned or leaked instances | `compute-manager reconcile run`, `compute-manager reconcile list` |

Run `compute-manager <group> <command> --help` for flags.

`instance destroy` and `reconcile quarantine` are destructive; confirm the target first.
