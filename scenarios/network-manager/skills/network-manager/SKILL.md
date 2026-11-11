---
name: "network-manager"
description: "Check and manage my home network: run a network health snapshot, diagnose a slow or broken internet connection, block ads and unwanted sites with DNS filtering, set household filtering profiles and schedules, and see which devices are on the network."
license: "CC-BY-4.0"
metadata:
  kind: "skill"
  schemaVersion: 1
  status: "active"
  modes: ["tools"]
  tags: ["network-manager", "network", "internet", "wifi", "dns", "dns-filtering", "ad-blocking", "adguard", "parental-controls", "devices", "diagnose"]
  requires:
    scenarios: ["network-manager"]
    commands: ["network-manager snapshot run"]
  origin:
    kind: "authored"
---

## network-manager

Local-first network management. Start read-only; filtering changes are previewed before they are applied.

| You want | Command |
|---|---|
| Is my network healthy / why is it slow | `network-manager snapshot run`, `network-manager snapshot get` |
| DNS filtering status and health | `network-manager resolver status`, `network-manager resolver health` |
| Change filtering safely | `network-manager policy preview`, `network-manager policy apply`, `network-manager policy rollback` |
| Pause or resume filtering for a device | `network-manager policy pause`, `network-manager policy resume` |
| Household profiles and schedules | `network-manager policy profiles`, `network-manager policy profile-set`, `network-manager policy schedule` |
| Devices on the network | `network-manager devices list`, `network-manager devices refresh` |

Run `network-manager <group> <command> --help` for flags.

Always `policy preview` before `policy apply`; `policy rollback` undoes a change.
