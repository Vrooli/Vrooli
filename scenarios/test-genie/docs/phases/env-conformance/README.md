# Environment Conformance Phase

The `env-conformance` phase validates Go environment reads against producers
derived from resource and scenario manifests. It is the satisfiability sibling
of `env_validation`: `env_validation` checks read-site discipline, while this
phase checks whether the lifecycle can supply the value at all.

## Capabilities

- `producer_declared`: resource producers are declared under
  `dependencies.resources`.
- `package_adoption`: reads owned by a shared package seam identify that
  package as the remediation owner.
- `address_resolution`: peer addresses are resolved through discovery rather
  than frozen in environment variables.
- `residual_routing`: values without producers are routed to scenario config,
  credential descriptors, an OS/toolchain allowlist, or deletion.

Each capability has three rungs: `L0` inventory unavailable, `L1` findings
present, and `L2` clean. ERROR findings fail the phase; WARNING findings record
reviewable residual debt.

## Findings

| Code | Severity | Meaning |
|---|---|---|
| `env.undeclared_resource_producer` | ERROR | A declared resource can produce the read variable but is absent from the scenario dependencies. |
| `env.package_bypassed` | ERROR | A package owns the resource environment seam and the consumer reads the raw variables directly. |
| `env.address_in_environment` | ERROR | A foreign scenario address is being carried through the environment instead of resolved by discovery. |
| `env.producer_absent` | WARNING | No manifest-derived producer exists; route the value to config/credentials or delete the read. |

The executable scanner is `packages/envresolve-go/cmd/env-conformance`. It
derives every producer and variable name from repository artifacts; it contains
no consumer-specific environment-variable map.

## Verification

```bash
cd packages/envresolve-go
go run ./cmd/env-conformance --root ../.. > /tmp/env-findings.json
jq 'group_by(.code) | map({code: .[0].code, count: length})' /tmp/env-findings.json
```

The authoritative census and re-scope baseline are stored in
`packages/envresolve-go/census_baseline.json` and must be regenerated after
each migration phase.

## Re-scope record (2026-08-23)

The authoritative worktree scan supersedes the investigation estimate of 548
pairs. It currently reports 718 pairs across 87 scenarios. The committed
census baseline records zero class-A and class-B findings; the remaining 718
occurrences are class C because they are scenario configuration, credentials,
tool settings, or other non-producer environment inputs pending explicit
ownership review. Database, Redis, peer-address, and UI migrations use this
finding list as their input.
